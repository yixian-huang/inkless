#!/usr/bin/env bash
# Production deployment script template
# Supports environment-based configuration for frontend and backend deployment

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
ENVIRONMENT="${ENVIRONMENT:-production}"
DEPLOY_USER="${DEPLOY_USER:-deploy}"
DEPLOY_HOST="${DEPLOY_HOST:?DEPLOY_HOST environment variable is required}"
DEPLOY_ROOT="${DEPLOY_ROOT:-/opt/blotting}"
VERSION="${VERSION:?VERSION environment variable is required}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-./artifacts}"
DEPLOY_AUTO_APPROVE="${DEPLOY_AUTO_APPROVE:-false}"

# Service control
BACKEND_SERVICE="${BACKEND_SERVICE:-blotting-api}"
FRONTEND_PATH="${FRONTEND_PATH:-${DEPLOY_ROOT}/frontend}"
BACKEND_PATH="${BACKEND_PATH:-${DEPLOY_ROOT}/backend}"
BACKEND_HEALTH_URL="${BACKEND_HEALTH_URL:-http://127.0.0.1:8088/health}"

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

is_true() {
  case "${1,,}" in
    true|1|yes|y) return 0 ;;
    *) return 1 ;;
  esac
}

# Preflight checks
preflight_checks() {
  log_info "Running preflight checks..."

  # Check required commands
  for cmd in ssh scp tar; do
    if ! command -v "$cmd" &> /dev/null; then
      log_error "Required command not found: $cmd"
      exit 1
    fi
  done

  # Check artifact files
  if [ ! -f "${ARTIFACTS_DIR}/frontend-${VERSION}.tar.gz" ]; then
    log_error "Frontend artifact not found: ${ARTIFACTS_DIR}/frontend-${VERSION}.tar.gz"
    exit 1
  fi

  if [ ! -f "${ARTIFACTS_DIR}/backend-${VERSION}.tar.gz" ]; then
    log_error "Backend artifact not found: ${ARTIFACTS_DIR}/backend-${VERSION}.tar.gz"
    exit 1
  fi

  # Verify checksums if available
  if [ -f "${ARTIFACTS_DIR}/frontend-${VERSION}.tar.gz.sha256" ]; then
    log_info "Verifying frontend artifact checksum..."
    if command -v sha256sum &> /dev/null; then
      (cd "${ARTIFACTS_DIR}" && sha256sum -c "frontend-${VERSION}.tar.gz.sha256")
    elif command -v shasum &> /dev/null; then
      (cd "${ARTIFACTS_DIR}" && shasum -a 256 -c "frontend-${VERSION}.tar.gz.sha256")
    fi
  fi

  if [ -f "${ARTIFACTS_DIR}/backend-${VERSION}.tar.gz.sha256" ]; then
    log_info "Verifying backend artifact checksum..."
    if command -v sha256sum &> /dev/null; then
      (cd "${ARTIFACTS_DIR}" && sha256sum -c "backend-${VERSION}.tar.gz.sha256")
    elif command -v shasum &> /dev/null; then
      (cd "${ARTIFACTS_DIR}" && shasum -a 256 -c "backend-${VERSION}.tar.gz.sha256")
    fi
  fi

  log_info "Preflight checks passed"
}

# Deploy frontend
deploy_frontend() {
  log_info "Deploying frontend version ${VERSION}..."

  # Upload artifact
  log_info "Uploading frontend artifact..."
  scp "${ARTIFACTS_DIR}/frontend-${VERSION}.tar.gz" \
    "${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_ROOT}/frontend-${VERSION}.tar.gz"

  # Extract and activate on remote
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

# Create version directory
mkdir -p "${FRONTEND_PATH}/versions/${VERSION}"

# Extract artifact
tar -xzf "${DEPLOY_ROOT}/frontend-${VERSION}.tar.gz" \
  -C "${FRONTEND_PATH}/versions/${VERSION}"

# Create backup symlink of current version
if [ -L "${FRONTEND_PATH}/current" ]; then
  CURRENT_TARGET=\$(readlink "${FRONTEND_PATH}/current")
  ln -snf "\${CURRENT_TARGET}" "${FRONTEND_PATH}/previous"
fi

# Atomic symlink swap
ln -snf "${FRONTEND_PATH}/versions/${VERSION}" "${FRONTEND_PATH}/current_tmp"
mv -Tf "${FRONTEND_PATH}/current_tmp" "${FRONTEND_PATH}/current"

# Reload web server (if using nginx)
if command -v nginx &> /dev/null && systemctl is-active --quiet nginx; then
  run_privileged systemctl reload nginx
fi

# Cleanup old artifact
rm -f "${DEPLOY_ROOT}/frontend-${VERSION}.tar.gz"

echo "Frontend deployed: ${VERSION}"
EOF

  log_info "Frontend deployment completed"
}

# Deploy backend
deploy_backend() {
  log_info "Deploying backend version ${VERSION}..."

  # Upload artifact
  log_info "Uploading backend artifact..."
  scp "${ARTIFACTS_DIR}/backend-${VERSION}.tar.gz" \
    "${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_ROOT}/backend-${VERSION}.tar.gz"

  # Extract and activate on remote
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

# Create version directory
mkdir -p "${BACKEND_PATH}/versions/${VERSION}"

# Extract artifact
tar -xzf "${DEPLOY_ROOT}/backend-${VERSION}.tar.gz" \
  -C "${BACKEND_PATH}/versions/${VERSION}"

# Ensure stable backend executable symlink exists for systemd
BACKEND_BIN=\$(find "${BACKEND_PATH}/versions/${VERSION}" -maxdepth 1 -type f -name "blotting-api-*" | head -n 1)
if [ -z "\${BACKEND_BIN}" ]; then
  echo "ERROR: Backend binary not found in extracted artifact"
  exit 1
fi
chmod +x "\${BACKEND_BIN}"
ln -snf "\$(basename "\${BACKEND_BIN}")" "${BACKEND_PATH}/versions/${VERSION}/blotting-api-latest"

# Create backup symlink of current version
if [ -L "${BACKEND_PATH}/current" ]; then
  CURRENT_TARGET=\$(readlink "${BACKEND_PATH}/current")
  ln -snf "\${CURRENT_TARGET}" "${BACKEND_PATH}/previous"
fi

# Stop service
run_privileged systemctl stop "${BACKEND_SERVICE}" || true

# Atomic symlink swap
ln -snf "${BACKEND_PATH}/versions/${VERSION}" "${BACKEND_PATH}/current_tmp"
mv -Tf "${BACKEND_PATH}/current_tmp" "${BACKEND_PATH}/current"

# Start service
run_privileged systemctl start "${BACKEND_SERVICE}"

# Wait for health check
sleep 3
if ! systemctl is-active --quiet "${BACKEND_SERVICE}"; then
  echo "ERROR: Backend service failed to start"
  exit 1
fi
if command -v curl &> /dev/null; then
  if ! curl --silent --show-error --fail --max-time 5 "${BACKEND_HEALTH_URL}" > /dev/null; then
    echo "ERROR: Backend health check failed at ${BACKEND_HEALTH_URL}"
    exit 1
  fi
else
  echo "WARNING: curl not found on remote host, skipping HTTP health check"
fi

# Cleanup old artifact
rm -f "${DEPLOY_ROOT}/backend-${VERSION}.tar.gz"

echo "Backend deployed: ${VERSION}"
EOF

  log_info "Backend deployment completed"
}

# Verify deployment
verify_deployment() {
  log_info "Verifying deployment..."

  # Check frontend
  ssh "${DEPLOY_USER}@${DEPLOY_HOST}" bash <<EOF
if [ -f "${FRONTEND_PATH}/current/build-info.json" ]; then
  echo "Frontend build info:"
  cat "${FRONTEND_PATH}/current/build-info.json"
else
  echo "Warning: Frontend build-info.json not found"
fi
EOF

  # Check backend service
  ssh "${DEPLOY_USER}@${DEPLOY_HOST}" bash <<EOF
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

echo "Backend service status:"
run_privileged systemctl status "${BACKEND_SERVICE}" --no-pager || true

if [ -f "${BACKEND_PATH}/current/build-info.json" ]; then
  echo "Backend build info:"
  cat "${BACKEND_PATH}/current/build-info.json"
else
  echo "Warning: Backend build-info.json not found"
fi
EOF

  log_info "Deployment verification completed"
}

# Main execution
main() {
  echo "=========================================="
  echo "Production Deployment"
  echo "=========================================="
  echo "Environment: ${ENVIRONMENT}"
  echo "Version: ${VERSION}"
  echo "Deploy Host: ${DEPLOY_HOST}"
  echo "Deploy User: ${DEPLOY_USER}"
  echo "Deploy Root: ${DEPLOY_ROOT}"
  echo "=========================================="

  # Confirm deployment unless auto-approved (CI mode)
  if ! is_true "${DEPLOY_AUTO_APPROVE}"; then
    read -p "Proceed with deployment? (yes/no): " -r
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
      log_warn "Deployment cancelled"
      exit 0
    fi
  else
    log_info "Auto-approve enabled, skipping interactive confirmation"
  fi

  preflight_checks
  deploy_frontend
  deploy_backend
  verify_deployment

  echo "=========================================="
  log_info "Deployment completed successfully!"
  echo "Version ${VERSION} is now live on ${ENVIRONMENT}"
  echo "=========================================="
}

# Script entry point
main "$@"
