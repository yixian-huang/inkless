#!/usr/bin/env bash
# Shared helpers for Quick-Box artifact deploy scripts.
set -euo pipefail

qb_log_info() { echo "[qb-artifact][INFO] $*"; }

qb_heartbeat_loop() {
  while sleep 60; do
    echo "[qb-artifact][heartbeat] $(date -u +"%Y-%m-%dT%H:%M:%SZ")" >&2
  done
}

qb_run_with_heartbeat() {
  qb_heartbeat_loop &
  local hb_pid=$!
  trap 'kill "${hb_pid}" 2>/dev/null || true' RETURN
  "$@"
}
qb_log_warn() { echo "[qb-artifact][WARN] $*" >&2; }
qb_log_error() { echo "[qb-artifact][ERROR] $*" >&2; }

qb_repo_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd
}

qb_resolve_version() {
  if [[ -n "${QB_VERSION:-}" ]]; then
    echo "${QB_VERSION}"
    return 0
  fi
  local root="${1:-}"
  if [[ -n "${root}" ]] && [[ -d "${root}/.git" ]]; then
    git -C "${root}" describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-unknown"
    return 0
  fi
  echo "v0.0.0-unknown"
}

qb_sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${file}" | awk '{print $1}'
  else
    qb_log_error "sha256sum or shasum required"
    return 1
  fi
}

qb_verify_checksum_file() {
  local artifact="$1"
  local checksum_file="$2"
  if [[ ! -f "${checksum_file}" ]]; then
    qb_log_warn "checksum file missing: ${checksum_file}"
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "${artifact}")" && sha256sum -c "$(basename "${checksum_file}")")
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$(dirname "${artifact}")" && shasum -a 256 -c "$(basename "${checksum_file}")")
  fi
}

qb_component_enabled() {
  local name="$1"
  local list="${QB_BUILD_COMPONENTS:-backend,frontend}"
  [[ ",${list}," == *",${name},"* ]]
}

qb_artifact_incoming_dir() {
  if [[ -n "${QB_ARTIFACT_INCOMING:-}" ]]; then
    echo "${QB_ARTIFACT_INCOMING}"
  elif [[ -n "${QB_STAGING_PATH:-}" ]]; then
    echo "${QB_STAGING_PATH}"
  else
    qb_log_error "QB_ARTIFACT_INCOMING or QB_STAGING_PATH is required"
    return 1
  fi
}

qb_release_root() {
  echo "${QB_RELEASE_ROOT:-/opt/inkless}"
}

qb_health_url() {
  echo "${QB_HEALTH_CHECK_URL:-${HEALTH_CHECK_URL:-http://127.0.0.1:${PORT:-8088}/health}}"
}

qb_systemd_unit() {
  echo "${QB_SYSTEMD_UNIT:-inkless}"
}

qb_runtime_type() {
  echo "${QB_RUNTIME_TYPE:-systemd}"
}

qb_atomic_symlink() {
  local target="$1"
  local link="$2"
  local tmp="${link}_tmp"
  ln -snf "${target}" "${tmp}"
  if command -v python3 >/dev/null 2>&1; then
    python3 - "${tmp}" "${link}" <<'PY'
import os
import sys

os.replace(sys.argv[1], sys.argv[2])
PY
  else
    mv -Tf "${tmp}" "${link}"
  fi
}

qb_backup_current_symlink() {
  local base="$1"
  local current="${base}/current"
  local previous="${base}/previous"
  if [[ -L "${current}" ]]; then
    local current_target
    current_target="$(readlink "${current}")"
    ln -snf "${current_target}" "${previous}"
  fi
}

qb_load_env_file_defaults() {
  local env_file="$1"
  [[ -f "${env_file}" ]] || return 0

  local line key value
  while IFS= read -r line || [[ -n "${line}" ]]; do
    [[ -z "${line}" || "${line}" == \#* ]] && continue
    [[ "${line}" == *=* ]] || continue
    key="${line%%=*}"
    value="${line#*=}"
    # Export any KEY if not already set in the environment (preserve deploy-time overrides).
    # Includes INKLESS_* self-update / catalog keys so activate merge keeps them.
    if [[ -z "${!key+x}" ]]; then
      printf -v "${key}" '%s' "${value}"
      export "${key?}"
    fi
  done <"${env_file}"
}

# Merge-write instance .env: update managed path/runtime keys, never drop unknown
# keys (JWT, INKLESS_SELF_UPDATE_*, catalog URL, etc.).
qb_write_env_file() {
  local env_file="$1"
  local release_root="$2"
  local port="${PORT:-8088}"
  local base_url="${BASE_URL:-http://127.0.0.1:${port}}"
  mkdir -p "$(dirname "${env_file}")"

  if command -v python3 >/dev/null 2>&1; then
    PORT="${port}" \
    ENV="${ENV:-}" \
    SEED_MODE="${SEED_MODE:-}" \
    SETUP_BOOTSTRAP="${SETUP_BOOTSTRAP:-}" \
    FRONTEND_DIR="${FRONTEND_DIR:-${release_root}/frontend/current}" \
    UPLOAD_DIR="${UPLOAD_DIR:-${release_root}/uploads}" \
    BACKUP_DIR="${BACKUP_DIR:-${release_root}/backups}" \
    BASE_URL="${base_url}" \
    CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-${base_url}}" \
    PLUGIN_DIR="${PLUGIN_DIR:-${release_root}/plugins}" \
    PLUGIN_DATA_DIR="${PLUGIN_DATA_DIR:-${release_root}/data/plugins}" \
    ENABLE_EXTERNAL_PLUGINS="${ENABLE_EXTERNAL_PLUGINS:-false}" \
    DB_DSN="${DB_DSN:-}" \
    JWT_SECRET="${JWT_SECRET:-}" \
    JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET:-}" \
    INKLESS_SECRET_KEY="${INKLESS_SECRET_KEY:-}" \
    python3 - "${env_file}" "${release_root}" <<'PY'
import os, sys
from pathlib import Path

env_file = Path(sys.argv[1])
existing = {}
order = []
if env_file.is_file():
    for line in env_file.read_text().splitlines():
        s = line.strip()
        if not s or s.startswith("#") or "=" not in line:
            order.append(("raw", line))
            continue
        k, v = line.split("=", 1)
        k = k.strip()
        if k not in existing:
            order.append(("key", k))
        existing[k] = v

# Managed keys from activate (overlay when non-empty for secrets)
managed = {
    "PORT": os.environ.get("PORT", "8088"),
    "FRONTEND_DIR": os.environ["FRONTEND_DIR"],
    "UPLOAD_DIR": os.environ["UPLOAD_DIR"],
    "BACKUP_DIR": os.environ["BACKUP_DIR"],
    "BASE_URL": os.environ["BASE_URL"],
    "CORS_ALLOWED_ORIGINS": os.environ["CORS_ALLOWED_ORIGINS"],
    "PLUGIN_DIR": os.environ["PLUGIN_DIR"],
    "PLUGIN_DATA_DIR": os.environ["PLUGIN_DATA_DIR"],
    "ENABLE_EXTERNAL_PLUGINS": os.environ.get("ENABLE_EXTERNAL_PLUGINS", "false"),
}
for k in ("ENV", "SEED_MODE", "SETUP_BOOTSTRAP", "DB_DSN", "JWT_SECRET", "JWT_REFRESH_SECRET", "INKLESS_SECRET_KEY"):
    v = os.environ.get(k, "")
    if v != "":
        managed[k] = v

for k, v in managed.items():
    if k not in existing:
        order.append(("key", k))
    existing[k] = v

lines = []
seen = set()
for kind, item in order:
    if kind == "raw":
        lines.append(item)
        continue
    if item in seen:
        continue
    seen.add(item)
    lines.append(f"{item}={existing[item]}")
for k, v in existing.items():
    if k not in seen:
        lines.append(f"{k}={v}")

from datetime import datetime, timezone
header = f"# Merged by qb-artifact-activate at {datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')}"
out = [header]
# drop old generated headers
for line in lines:
    if line.startswith("# Generated by qb-artifact") or line.startswith("# Merged by qb-artifact"):
        continue
    out.append(line)
env_file.write_text("\n".join(out) + "\n")
os.chmod(env_file, 0o600)
PY
  else
    # Fallback without python: preserve by appending only if file missing.
    if [[ -f "${env_file}" ]]; then
      qb_log_warn "python3 missing; leaving existing env file untouched: ${env_file}"
      return 0
    fi
    {
      echo "# Generated by qb-artifact-activate at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
      echo "PORT=${port}"
      echo "FRONTEND_DIR=${FRONTEND_DIR:-${release_root}/frontend/current}"
      echo "UPLOAD_DIR=${UPLOAD_DIR:-${release_root}/uploads}"
      echo "BACKUP_DIR=${BACKUP_DIR:-${release_root}/backups}"
      echo "BASE_URL=${base_url}"
      echo "CORS_ALLOWED_ORIGINS=${CORS_ALLOWED_ORIGINS:-${base_url}}"
      echo "PLUGIN_DIR=${PLUGIN_DIR:-${release_root}/plugins}"
      echo "PLUGIN_DATA_DIR=${PLUGIN_DATA_DIR:-${release_root}/data/plugins}"
      echo "ENABLE_EXTERNAL_PLUGINS=${ENABLE_EXTERNAL_PLUGINS:-false}"
      [[ -n "${DB_DSN:-}" ]] && echo "DB_DSN=${DB_DSN}"
      [[ -n "${JWT_SECRET:-}" ]] && echo "JWT_SECRET=${JWT_SECRET}"
      [[ -n "${JWT_REFRESH_SECRET:-}" ]] && echo "JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}"
    } >"${env_file}"
    chmod 600 "${env_file}" 2>/dev/null || true
  fi

  if id inkless >/dev/null 2>&1; then
    chown inkless:inkless "${env_file}" 2>/dev/null || true
  fi
}

# Strategy A: after activating /opt/inkless, rsync the same VERSION into peer
# instance trees (ops/imgli) and restart those units. Disable with
# QB_PROPAGATE_PEERS=0. Override map with INKLESS_ARTIFACT_PEERS=
#   "unit:root,unit:root"
qb_default_artifact_peers() {
  echo "inkless-ops:/opt/inkless-ops,inkless-imgli:/opt/inkless-imgli"
}

qb_propagate_artifact_to_peers() {
  local release_root="${1:-}"
  local version="${2:-}"
  if [[ "${QB_PROPAGATE_PEERS:-1}" == "0" || "${QB_PROPAGATE_PEERS:-1}" == "false" ]]; then
    qb_log_info "peer propagate disabled (QB_PROPAGATE_PEERS=0)"
    return 0
  fi
  if [[ "${release_root}" != "/opt/inkless" ]]; then
    return 0
  fi
  if [[ -z "${version}" ]]; then
    qb_log_warn "peer propagate skipped: empty version"
    return 0
  fi
  local b_src="${release_root}/backend/versions/${version}"
  local f_src="${release_root}/frontend/versions/${version}"
  if [[ ! -d "${b_src}" ]]; then
    qb_log_warn "peer propagate skipped: missing ${b_src}"
    return 0
  fi
  if ! command -v rsync >/dev/null 2>&1; then
    qb_log_warn "peer propagate skipped: rsync not found"
    return 0
  fi

  local peers="${INKLESS_ARTIFACT_PEERS:-$(qb_default_artifact_peers)}"
  local IFS=','
  local entry unit root
  for entry in ${peers}; do
    entry="$(echo "${entry}" | tr -d ' ')"
    [[ -n "${entry}" ]] || continue
    unit="${entry%%:*}"
    root="${entry#*:}"
    if [[ -z "${unit}" || -z "${root}" || "${unit}" == "${root}" ]]; then
      qb_log_warn "skip bad peer entry: ${entry}"
      continue
    fi
    if [[ ! -d "${root}" ]]; then
      qb_log_warn "peer root missing, skip: ${root}"
      continue
    fi
    qb_log_info "propagating ${version} -> ${root} (unit=${unit})"
    mkdir -p "${root}/backend/versions" "${root}/frontend/versions" "${root}/var/updates"
    rsync -a --delete "${b_src}/" "${root}/backend/versions/${version}/"
    if [[ -d "${f_src}" ]]; then
      rsync -a --delete "${f_src}/" "${root}/frontend/versions/${version}/"
    fi
    if [[ -L "${root}/backend/current" ]]; then
      local old
      old="$(readlink -f "${root}/backend/current" 2>/dev/null || true)"
      if [[ -n "${old}" && "${old}" != "${root}/backend/versions/${version}" ]]; then
        ln -sfn "${old}" "${root}/backend/previous"
      fi
    fi
    if [[ -L "${root}/frontend/current" ]]; then
      local oldf
      oldf="$(readlink -f "${root}/frontend/current" 2>/dev/null || true)"
      if [[ -n "${oldf}" && "${oldf}" != "${root}/frontend/versions/${version}" ]]; then
        ln -sfn "${oldf}" "${root}/frontend/previous"
      fi
    fi
    ln -sfn "${root}/backend/versions/${version}" "${root}/backend/current"
    if [[ -d "${root}/frontend/versions/${version}" ]]; then
      ln -sfn "${root}/frontend/versions/${version}" "${root}/frontend/current"
    fi
    if id inkless >/dev/null 2>&1; then
      chown -R inkless:inkless \
        "${root}/backend/versions/${version}" \
        "${root}/frontend/versions/${version}" \
        "${root}/backend/current" \
        "${root}/frontend/current" \
        "${root}/var" 2>/dev/null || true
      [[ -f "${root}/backend/.env" ]] && chown inkless:inkless "${root}/backend/.env" && chmod 600 "${root}/backend/.env" || true
    fi
    if command -v systemctl >/dev/null 2>&1; then
      if systemctl cat "${unit}.service" >/dev/null 2>&1 || systemctl cat "${unit}" >/dev/null 2>&1; then
        systemctl restart "${unit}" || qb_log_warn "restart ${unit} failed (non-fatal)"
        sleep 1
        if systemctl is-active --quiet "${unit}"; then
          qb_log_info "peer ${unit} active after propagate"
        else
          qb_log_warn "peer ${unit} not active after restart"
        fi
      else
        qb_log_warn "unit ${unit} not found"
      fi
    fi
  done
}

# Legacy name: after primary restart, propagate code to peer trees (strategy A).
qb_restart_shared_frontend_units() {
  local release_root="${1:-}"
  local version="${2:-${QB_VERSION:-}}"
  if [[ -z "${version}" && -L "${release_root}/backend/current" ]]; then
    version="$(basename "$(readlink -f "${release_root}/backend/current" 2>/dev/null || true)")"
  fi
  qb_propagate_artifact_to_peers "${release_root}" "${version}"
}

qb_restart_runtime() {
  local release_root="$1"
  local unit
  unit="$(qb_systemd_unit)"
  local runtime
  runtime="$(qb_runtime_type)"

  if [[ "${runtime}" == "systemd" ]] && command -v systemctl >/dev/null 2>&1; then
    if ! systemctl is-enabled "${unit}" >/dev/null 2>&1; then
      qb_log_warn "systemd unit ${unit} not enabled; attempting start anyway"
    fi
    systemctl daemon-reload 2>/dev/null || true
    systemctl restart "${unit}"
    sleep 2
    if ! systemctl is-active --quiet "${unit}"; then
      qb_log_error "systemd unit ${unit} failed to start"
      systemctl status "${unit}" --no-pager || true
      return 1
    fi
    # Peer propagate runs after primary health in activate.sh (needs VERSION).
    return 0
  fi

  qb_log_warn "systemd unavailable; using process mode on :${PORT:-8088}"
  local pid=""
  if command -v lsof >/dev/null 2>&1; then
    pid="$(lsof -nP -tiTCP:"${PORT:-8088}" -sTCP:LISTEN 2>/dev/null | head -1 || true)"
  elif command -v ss >/dev/null 2>&1; then
    pid="$(ss -tlnp "sport = :${PORT:-8088}" 2>/dev/null | sed -nE 's/.*pid=([0-9]+).*/\1/p' | head -1 || true)"
  fi
  if [[ -n "${pid}" ]]; then
    kill "${pid}" 2>/dev/null || true
    sleep 1
  fi
  local backend_dir="${release_root}/backend/current"
  local log_dir="${release_root}/logs"
  local log_file="${log_dir}/backend.log"
  mkdir -p "${log_dir}"
  # shellcheck disable=SC1091
  set -a
  source "${release_root}/backend/.env"
  set +a
  nohup "${backend_dir}/inkless-api-latest" >"${log_file}" 2>&1 &
  sleep 3
}

qb_health_check() {
  local url
  url="$(qb_health_url)"
  local retries="${QB_HEALTH_CHECK_RETRIES:-10}"
  local interval="${QB_HEALTH_CHECK_INTERVAL_SEC:-3}"
  local i=1
  while [[ "${i}" -le "${retries}" ]]; do
    if curl -sf --max-time 5 "${url}" >/dev/null; then
      qb_log_info "health check passed: ${url}"
      return 0
    fi
    qb_log_warn "health check attempt ${i}/${retries} failed: ${url}"
    sleep "${interval}"
    i=$((i + 1))
  done
  qb_log_error "health check failed after ${retries} attempts: ${url}"
  return 1
}
