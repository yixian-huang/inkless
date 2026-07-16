#!/usr/bin/env bash
# Rollback deployment script
# Rolls back to a previous version using symlink management

set -euo pipefail

# Configuration
ENVIRONMENT="${ENVIRONMENT:-production}"
DEPLOY_USER="${DEPLOY_USER:-deploy}"
DEPLOY_HOST="${DEPLOY_HOST:?DEPLOY_HOST environment variable is required}"
DEPLOY_ROOT="${DEPLOY_ROOT:-/opt/blotting}"
COMPONENT="${COMPONENT:?COMPONENT environment variable is required (frontend|backend|all)}"
TARGET_VERSION="${TARGET_VERSION:-previous}"

# Service control
BACKEND_SERVICE="${BACKEND_SERVICE:-blotting-api}"
FRONTEND_PATH="${FRONTEND_PATH:-${DEPLOY_ROOT}/frontend}"
BACKEND_PATH="${BACKEND_PATH:-${DEPLOY_ROOT}/backend}"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
  echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $*"
}

# Rollback frontend
rollback_frontend() {
  log_info "Rolling back frontend to ${TARGET_VERSION}..."

  ssh "${DEPLOY_USER}@${DEPLOY_HOST}" bash <<EOF
set -euo pipefail

run_privileged() {
  if [ "\$(id -u)" -eq 0 ]; then
    "\$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "\$@"
  else
    echo "ERROR: Root privileges are required to run: \$*"
    return 1
  fi
}

if [ "${TARGET_VERSION}" = "previous" ]; then
  # Use previous symlink
  if [ ! -L "${FRONTEND_PATH}/previous" ]; then
    echo "ERROR: No previous frontend version found"
    exit 1
  fi
  TARGET_PATH=\$(readlink "${FRONTEND_PATH}/previous")
else
  # Use explicit version
  TARGET_PATH="${FRONTEND_PATH}/versions/${TARGET_VERSION}"
  if [ ! -d "\${TARGET_PATH}" ]; then
    echo "ERROR: Frontend version not found: ${TARGET_VERSION}"
    exit 1
  fi
fi

# Show current and target versions
echo "Current version: \$(readlink "${FRONTEND_PATH}/current" || echo 'none')"
echo "Target version: \${TARGET_PATH}"

# Backup current symlink
if [ -L "${FRONTEND_PATH}/current" ]; then
  CURRENT_TARGET=\$(readlink "${FRONTEND_PATH}/current")
  ln -snf "\${CURRENT_TARGET}" "${FRONTEND_PATH}/rollback_backup"
fi

# Atomic symlink swap
ln -snf "\${TARGET_PATH}" "${FRONTEND_PATH}/current_tmp"
mv -Tf "${FRONTEND_PATH}/current_tmp" "${FRONTEND_PATH}/current"

# Reload web server (if using nginx)
if command -v nginx &> /dev/null && systemctl is-active --quiet nginx; then
  run_privileged systemctl reload nginx
fi

echo "Frontend rolled back to: \${TARGET_PATH}"
EOF

  log_info "Frontend rollback completed"
}

# Rollback backend
rollback_backend() {
  log_info "Rolling back backend to ${TARGET_VERSION}..."

  ssh "${DEPLOY_USER}@${DEPLOY_HOST}" bash <<EOF
set -euo pipefail

run_privileged() {
  if [ "\$(id -u)" -eq 0 ]; then
    "\$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo "\$@"
  else
    echo "ERROR: Root privileges are required to run: \$*"
    return 1
  fi
}

if [ "${TARGET_VERSION}" = "previous" ]; then
  # Use previous symlink
  if [ ! -L "${BACKEND_PATH}/previous" ]; then
    echo "ERROR: No previous backend version found"
    exit 1
  fi
  TARGET_PATH=\$(readlink "${BACKEND_PATH}/previous")
else
  # Use explicit version
  TARGET_PATH="${BACKEND_PATH}/versions/${TARGET_VERSION}"
  if [ ! -d "\${TARGET_PATH}" ]; then
    echo "ERROR: Backend version not found: ${TARGET_VERSION}"
    exit 1
  fi
fi

# Show current and target versions
echo "Current version: \$(readlink "${BACKEND_PATH}/current" || echo 'none')"
echo "Target version: \${TARGET_PATH}"

# Backup current symlink
if [ -L "${BACKEND_PATH}/current" ]; then
  CURRENT_TARGET=\$(readlink "${BACKEND_PATH}/current")
  ln -snf "\${CURRENT_TARGET}" "${BACKEND_PATH}/rollback_backup"
fi

# Stop service
run_privileged systemctl stop "${BACKEND_SERVICE}" || true

# Atomic symlink swap
ln -snf "\${TARGET_PATH}" "${BACKEND_PATH}/current_tmp"
mv -Tf "${BACKEND_PATH}/current_tmp" "${BACKEND_PATH}/current"

# Start service
run_privileged systemctl start "${BACKEND_SERVICE}"

# Wait for health check
sleep 3
if ! systemctl is-active --quiet "${BACKEND_SERVICE}"; then
  echo "ERROR: Backend service failed to start after rollback"
  exit 1
fi

echo "Backend rolled back to: \${TARGET_PATH}"
EOF

  log_info "Backend rollback completed"
}

# List available versions
list_versions() {
  log_info "Listing available versions on ${DEPLOY_HOST}..."

  ssh "${DEPLOY_USER}@${DEPLOY_HOST}" bash <<EOF
echo "Frontend versions:"
echo "  Current: \$(readlink -f "${FRONTEND_PATH}/current" 2>/dev/null || echo 'none')"
echo "  Previous: \$(readlink -f "${FRONTEND_PATH}/previous" 2>/dev/null || echo 'none')"
echo "  Available:"
if [ -d "${FRONTEND_PATH}/versions" ]; then
  ls -1t "${FRONTEND_PATH}/versions" | sed 's/^/    /'
fi

echo ""
echo "Backend versions:"
echo "  Current: \$(readlink -f "${BACKEND_PATH}/current" 2>/dev/null || echo 'none')"
echo "  Previous: \$(readlink -f "${BACKEND_PATH}/previous" 2>/dev/null || echo 'none')"
echo "  Available:"
if [ -d "${BACKEND_PATH}/versions" ]; then
  ls -1t "${BACKEND_PATH}/versions" | sed 's/^/    /'
fi
EOF
}

# Main execution
main() {
  echo "=========================================="
  echo "Deployment Rollback"
  echo "=========================================="
  echo "Environment: ${ENVIRONMENT}"
  echo "Component: ${COMPONENT}"
  echo "Target Version: ${TARGET_VERSION}"
  echo "Deploy Host: ${DEPLOY_HOST}"
  echo "=========================================="

  # List versions if requested
  if [ "$COMPONENT" = "list" ]; then
    list_versions
    exit 0
  fi

  # Confirm rollback
  log_warn "This will rollback the ${COMPONENT} component(s) to ${TARGET_VERSION}"
  read -p "Proceed with rollback? (yes/no): " -r
  if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    log_warn "Rollback cancelled"
    exit 0
  fi

  case "$COMPONENT" in
    frontend)
      rollback_frontend
      ;;
    backend)
      rollback_backend
      ;;
    all)
      rollback_frontend
      rollback_backend
      ;;
    *)
      log_error "Invalid component: ${COMPONENT}. Must be frontend, backend, or all"
      exit 1
      ;;
  esac

  echo "=========================================="
  log_info "Rollback completed successfully!"
  echo "=========================================="
}

# Script entry point
main "$@"
