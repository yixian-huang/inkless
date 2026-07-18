#!/usr/bin/env bash
# Quick-Box artifact activate (runs on deploy server / hk VPS).
# Env: QB_ARTIFACT_INCOMING, QB_VERSION, QB_RELEASE_ROOT, QB_SYSTEMD_UNIT, app secrets
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=qb-artifact-common.sh
source "${SCRIPT_DIR}/qb-artifact-common.sh"

INCOMING="$(qb_artifact_incoming_dir)"
RELEASE_ROOT="$(qb_release_root)"
VERSION="${QB_VERSION:-}"
MANIFEST="${INCOMING}/artifact-manifest.json"

if [[ ! -f "${MANIFEST}" ]]; then
  qb_log_error "artifact-manifest.json not found in ${INCOMING}"
  exit 1
fi

if [[ -z "${VERSION}" ]]; then
  if command -v python3 >/dev/null 2>&1; then
    VERSION="$(python3 -c "import json; print(json.load(open('${MANIFEST}'))['version'])")"
  else
    qb_log_error "QB_VERSION required when python3 is unavailable"
    exit 1
  fi
fi

qb_log_info "activate version=${VERSION} incoming=${INCOMING} release=${RELEASE_ROOT}"

verify_manifest() {
  if ! command -v python3 >/dev/null 2>&1; then
    qb_log_warn "python3 missing; skipping manifest component verification"
    return 0
  fi
  python3 - "${INCOMING}" "${MANIFEST}" <<'PY'
import json, hashlib, os, sys

incoming, manifest_path = sys.argv[1], sys.argv[2]
manifest = json.load(open(manifest_path))
for comp in manifest.get("components", []):
    path = os.path.join(incoming, os.path.basename(comp["path"]))
    if not os.path.isfile(path):
        raise SystemExit(f"missing artifact: {path}")
    with open(path, "rb") as f:
        digest = hashlib.sha256(f.read()).hexdigest()
    expected = comp.get("sha256", "")
    if expected and digest != expected:
        raise SystemExit(f"checksum mismatch for {path}: expected {expected}, got {digest}")
print("manifest verification ok")
PY
}

deploy_backend() {
  local tar_path="${INCOMING}/backend-${VERSION}.tar.gz"
  [[ -f "${tar_path}" ]] || return 0

  qb_log_info "activating backend ${VERSION}"
  if [[ -f "${tar_path}.sha256" ]]; then
    qb_verify_checksum_file "${tar_path}" "${tar_path}.sha256"
  fi

  local backend_base="${RELEASE_ROOT}/backend"
  local version_dir="${backend_base}/versions/${VERSION}"
  mkdir -p "${version_dir}"

  tar -xzf "${tar_path}" -C "${version_dir}"

  local backend_bin
  backend_bin="$(find "${version_dir}" -maxdepth 1 -type f -name 'blotting-api-*' ! -name 'blotting-api-latest' | head -1)"
  if [[ -z "${backend_bin}" ]]; then
    qb_log_error "backend binary not found in ${version_dir}"
    return 1
  fi
  chmod +x "${backend_bin}"
  ln -snf "$(basename "${backend_bin}")" "${version_dir}/blotting-api-latest"

  qb_backup_current_symlink "${backend_base}"
  qb_atomic_symlink "${backend_base}/versions/${VERSION}" "${backend_base}/current"
}

deploy_frontend() {
  local tar_path="${INCOMING}/frontend-${VERSION}.tar.gz"
  [[ -f "${tar_path}" ]] || return 0

  qb_log_info "activating frontend ${VERSION}"
  if [[ -f "${tar_path}.sha256" ]]; then
    qb_verify_checksum_file "${tar_path}" "${tar_path}.sha256"
  fi

  local frontend_base="${RELEASE_ROOT}/frontend"
  local version_dir="${frontend_base}/versions/${VERSION}"
  mkdir -p "${version_dir}"

  tar -xzf "${tar_path}" -C "${version_dir}"

  qb_backup_current_symlink "${frontend_base}"
  qb_atomic_symlink "${frontend_base}/versions/${VERSION}" "${frontend_base}/current"
}

ensure_layout() {
  mkdir -p "${RELEASE_ROOT}/data" "${RELEASE_ROOT}/uploads" "${BACKUP_DIR:-${RELEASE_ROOT}/backups}"
  mkdir -p "${PLUGIN_DIR:-${RELEASE_ROOT}/plugins}" "${PLUGIN_DATA_DIR:-${RELEASE_ROOT}/data/plugins}"
  mkdir -p "${RELEASE_ROOT}/backend/versions" "${RELEASE_ROOT}/frontend/versions"

  if ! id impress >/dev/null 2>&1; then
    qb_log_warn "user 'impress' not found; systemd may need User= adjustment"
  else
    chown -R impress:impress \
      "${RELEASE_ROOT}/data" \
      "${RELEASE_ROOT}/uploads" \
      "${BACKUP_DIR:-${RELEASE_ROOT}/backups}" \
      "${PLUGIN_DIR:-${RELEASE_ROOT}/plugins}" \
      "${PLUGIN_DATA_DIR:-${RELEASE_ROOT}/data/plugins}" \
      2>/dev/null || true
  fi
}

install_systemd_unit() {
  local unit
  unit="$(qb_systemd_unit)"
  local unit_path="/etc/systemd/system/${unit}.service"
  local template="${QB_SYSTEMD_UNIT_FILE:-}"
  if [[ -z "${template}" ]]; then
    if [[ -f "${INCOMING}/ops/systemd/impress.service" ]]; then
      template="${INCOMING}/ops/systemd/impress.service"
    else
      template="${SCRIPT_DIR}/../ops/systemd/impress.service"
    fi
  fi

  if [[ ! -f "${unit_path}" ]]; then
    if [[ ! -f "${template}" ]]; then
      qb_log_warn "systemd template missing at ${template}; skip unit install"
      return 0
    fi
    qb_log_info "installing systemd unit ${unit} from ${template}"
    cp "${template}" "${unit_path}"
    sed -i "s|/opt/impress|${RELEASE_ROOT}|g" "${unit_path}" 2>/dev/null || \
      sed -i '' "s|/opt/impress|${RELEASE_ROOT}|g" "${unit_path}"
    systemctl daemon-reload
    systemctl enable "${unit}"
  fi

  local dropin_dir="/etc/systemd/system/${unit}.service.d"
  mkdir -p "${dropin_dir}"
  {
    echo "[Service]"
    echo "ReadWritePaths=${BACKUP_DIR:-${RELEASE_ROOT}/backups} ${PLUGIN_DIR:-${RELEASE_ROOT}/plugins} ${PLUGIN_DATA_DIR:-${RELEASE_ROOT}/data/plugins}"
  } >"${dropin_dir}/impress-instance-paths.conf"
  systemctl daemon-reload
}

rollback_on_failure() {
  local component="${1:-all}"
  qb_log_error "activate failed; attempting rollback to previous"
  QB_RELEASE_ROOT="${RELEASE_ROOT}" COMPONENT="${component}" TARGET_VERSION=previous \
    "${SCRIPT_DIR}/qb-artifact-rollback.sh" || true
}

main() {
  verify_manifest

  local env_file="${RELEASE_ROOT}/backend/.env"
  qb_load_env_file_defaults "${env_file}"

  export FRONTEND_DIR="${FRONTEND_DIR:-${RELEASE_ROOT}/frontend/current}"
  export UPLOAD_DIR="${UPLOAD_DIR:-${RELEASE_ROOT}/uploads}"
  export BACKUP_DIR="${BACKUP_DIR:-${RELEASE_ROOT}/backups}"
  export PORT="${PORT:-8088}"
  export BASE_URL="${BASE_URL:-http://127.0.0.1:${PORT}}"
  export CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-${BASE_URL}}"
  export PLUGIN_DIR="${PLUGIN_DIR:-${RELEASE_ROOT}/plugins}"
  export PLUGIN_DATA_DIR="${PLUGIN_DATA_DIR:-${RELEASE_ROOT}/data/plugins}"
  export ENABLE_EXTERNAL_PLUGINS="${ENABLE_EXTERNAL_PLUGINS:-false}"

  if [[ -z "${DB_DSN:-}" ]]; then
    export DB_DSN="file:${RELEASE_ROOT}/data/impress.db?cache=shared&mode=rwc"
  fi

  ensure_layout
  qb_write_env_file "${env_file}" "${RELEASE_ROOT}"

  local deployed_frontend=false
  local deployed_backend=false
  if ! deploy_frontend; then
    rollback_on_failure frontend
    exit 1
  fi
  [[ -f "${INCOMING}/frontend-${VERSION}.tar.gz" ]] && deployed_frontend=true
  if ! deploy_backend; then
    [[ "${deployed_frontend}" == "true" ]] && rollback_on_failure frontend
    exit 1
  fi
  [[ -f "${INCOMING}/backend-${VERSION}.tar.gz" ]] && deployed_backend=true

  install_systemd_unit

  if ! qb_restart_runtime "${RELEASE_ROOT}"; then
    if [[ "${deployed_frontend}" == "true" && "${deployed_backend}" == "true" ]]; then
      rollback_on_failure all
    elif [[ "${deployed_backend}" == "true" ]]; then
      rollback_on_failure backend
    elif [[ "${deployed_frontend}" == "true" ]]; then
      rollback_on_failure frontend
    fi
    exit 1
  fi

  sleep "${QB_HEALTH_CHECK_GRACE_SEC:-3}"
  if ! qb_health_check; then
    if [[ "${deployed_frontend}" == "true" && "${deployed_backend}" == "true" ]]; then
      rollback_on_failure all
    elif [[ "${deployed_backend}" == "true" ]]; then
      rollback_on_failure backend
    elif [[ "${deployed_frontend}" == "true" ]]; then
      rollback_on_failure frontend
    fi
    exit 1
  fi

  qb_log_info "activate complete version=${VERSION}"
}

main "$@"
