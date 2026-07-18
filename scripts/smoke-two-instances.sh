#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SMOKE_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/impress-two-instances.XXXXXX")"
PORT_A="${IMPRESS_SMOKE_PORT_A:-18088}"
PORT_B="${IMPRESS_SMOKE_PORT_B:-18089}"
BINARY="${IMPRESS_SMOKE_BINARY:-${SMOKE_ROOT}/impress-server}"
PID_A=""
PID_B=""

log() { echo "[two-instance-smoke] $*"; }
fail() { echo "[two-instance-smoke][FAIL] $*" >&2; exit 1; }

stop_pid() {
  local pid="${1:-}"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}"
    local i
    for ((i = 1; i <= 40; i++)); do
      if ! kill -0 "${pid}" 2>/dev/null; then
        return 0
      fi
      sleep 0.1
    done
    kill -9 "${pid}" 2>/dev/null || true
    for ((i = 1; i <= 20; i++)); do
      kill -0 "${pid}" 2>/dev/null || return 0
      sleep 0.1
    done
  fi
}

cleanup() {
  stop_pid "${PID_A}"
  stop_pid "${PID_B}"
  if [[ "${IMPRESS_SMOKE_KEEP:-false}" == "true" ]]; then
    log "kept workspace: ${SMOKE_ROOT}"
  else
    rm -rf "${SMOKE_ROOT}"
  fi
}
trap cleanup EXIT

for port in "${PORT_A}" "${PORT_B}"; do
  if curl -fsS --max-time 1 "http://127.0.0.1:${port}/health" >/dev/null 2>&1; then
    fail "port ${port} is already serving HTTP; set IMPRESS_SMOKE_PORT_A/B"
  fi
done
[[ "${PORT_A}" != "${PORT_B}" ]] || fail "instance ports must differ"

if [[ -z "${IMPRESS_SMOKE_BINARY:-}" ]]; then
  log "building backend smoke binary"
  (cd "${REPO_ROOT}/backend" && go build -o "${BINARY}" ./cmd/server)
fi
[[ -x "${BINARY}" ]] || fail "backend binary is not executable: ${BINARY}"

for instance in a b; do
  root="${SMOKE_ROOT}/${instance}"
  mkdir -p "${root}/data" "${root}/uploads" "${root}/plugins" "${root}/plugin-data" "${root}/backups"
  mkdir -p "${root}/backend/versions/v1" "${root}/backend/versions/v2"
  cp "${BINARY}" "${root}/backend/versions/v1/impress-server"
  cp "${BINARY}" "${root}/backend/versions/v2/impress-server"
  ln -s "${root}/backend/versions/v1" "${root}/backend/current"
done

start_instance() {
  local instance="$1" port="$2" title="$3"
  local root="${SMOKE_ROOT}/${instance}"
  local instance_binary="${root}/backend/current/impress-server"
  PORT="${port}" \
  ENV=production \
  SEED_MODE=demo \
  DB_DSN="file:${root}/data/impress.db?cache=shared&mode=rwc" \
  UPLOAD_DIR="${root}/uploads" \
  BACKUP_DIR="${root}/backups" \
  BASE_URL="https://${instance}.example.test" \
  CORS_ALLOWED_ORIGINS="https://${instance}.example.test" \
  PLUGIN_DIR="${root}/plugins" \
  PLUGIN_DATA_DIR="${root}/plugin-data" \
  ENABLE_EXTERNAL_PLUGINS=false \
  JWT_SECRET="smoke-${instance}-jwt-secret-0123456789" \
  JWT_REFRESH_SECRET="smoke-${instance}-refresh-secret-0123456789" \
  "${instance_binary}" >"${root}/server.log" 2>&1 &
  local pid=$!
  local i
  for ((i = 1; i <= 60; i++)); do
    if curl -fsS "http://127.0.0.1:${port}/health" >/dev/null 2>&1; then
      printf '%s' "${pid}"
      return 0
    fi
    if ! kill -0 "${pid}" 2>/dev/null; then
      tail -80 "${root}/server.log" >&2 || true
      fail "instance ${title} exited during startup"
    fi
    sleep 0.25
  done
  tail -80 "${root}/server.log" >&2 || true
  fail "instance ${title} did not become healthy"
}

json_value() {
  local key="$1"
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "${key}"
}

login() {
  local port="$1"
  curl -fsS -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"admin123"}' \
    "http://127.0.0.1:${port}/auth/login" | json_value accessToken
}

create_article() {
  local port="$1" token="$2" slug="$3" title="$4"
  curl -fsS -H 'Content-Type: application/json' -H "Authorization: Bearer ${token}" \
    -d "{\"slug\":\"${slug}\",\"status\":\"published\",\"zhTitle\":\"${title}\",\"zhBody\":\"${title} body\"}" \
    "http://127.0.0.1:${port}/admin/articles" >/dev/null
}

assert_article() {
  local port="$1" slug="$2" expected="$3"
  local body
  body="$(curl -fsS "http://127.0.0.1:${port}/public/articles/${slug}")"
  python3 -c 'import json,sys; data=json.loads(sys.argv[1]); expected=sys.argv[2]; assert data["zhTitle"] == expected, data' "${body}" "${expected}" || \
    fail "unexpected article content on port ${port}"
}

assert_article_missing() {
  local port="$1" slug="$2"
  local status
  status="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:${port}/public/articles/${slug}")"
  [[ "${status}" == "404" ]] || fail "article ${slug} unexpectedly exists on port ${port} (HTTP ${status})"
}

trigger_backup() {
  local instance="$1" port="$2" token="$3"
  local root="${SMOKE_ROOT}/${instance}"
  curl -fsS -X POST -H "Authorization: Bearer ${token}" \
    "http://127.0.0.1:${port}/admin/backups/trigger" >/dev/null
  find "${root}/backups" -maxdepth 1 -type f -name 'backup-*.db.gz' -print -quit
}

switch_version() {
  local instance="$1" version="$2"
  local root="${SMOKE_ROOT}/${instance}"
  local target="${root}/backend/versions/${version}"
  ln -sfn "${target}" "${root}/backend/current"
  [[ "$(readlink "${root}/backend/current")" == "${target}" ]] || \
    fail "instance ${instance} release did not switch to ${version}"
}

upload_marker() {
  local instance="$1" port="$2" token="$3"
  local root="${SMOKE_ROOT}/${instance}"
  python3 - "${root}/marker.png" <<'PY'
import base64, pathlib, sys
pathlib.Path(sys.argv[1]).write_bytes(base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
))
PY
  curl -fsS -H "Authorization: Bearer ${token}" -F "file=@${root}/marker.png;type=image/png" \
    "http://127.0.0.1:${port}/admin/media/upload" >/dev/null
  find "${root}/uploads" -type f -not -name '.gitkeep' -print -quit | grep -q . || fail "instance ${instance} upload missing"
}

log "starting isolated instances on ${PORT_A} and ${PORT_B}"
PID_A="$(start_instance a "${PORT_A}" A)"
PID_B="$(start_instance b "${PORT_B}" B)"

TOKEN_A="$(login "${PORT_A}")"
TOKEN_B="$(login "${PORT_B}")"
create_article "${PORT_A}" "${TOKEN_A}" shared-slug "Instance A"
create_article "${PORT_B}" "${TOKEN_B}" shared-slug "Instance B"
assert_article "${PORT_A}" shared-slug "Instance A"
assert_article "${PORT_B}" shared-slug "Instance B"

upload_marker a "${PORT_A}" "${TOKEN_A}"
upload_marker b "${PORT_B}" "${TOKEN_B}"
printf 'plugin state A\n' >"${SMOKE_ROOT}/a/plugin-data/shared-plugin.state"
printf 'plugin state B\n' >"${SMOKE_ROOT}/b/plugin-data/shared-plugin.state"
printf 'plugin package A\n' >"${SMOKE_ROOT}/a/plugins/shared-plugin.package"
printf 'plugin package B\n' >"${SMOKE_ROOT}/b/plugins/shared-plugin.package"
cmp -s "${SMOKE_ROOT}/a/plugin-data/shared-plugin.state" "${SMOKE_ROOT}/b/plugin-data/shared-plugin.state" && \
  fail "plugin data markers unexpectedly match"

log "triggering a real A backup and proving B remains available"
BACKUP_A="$(trigger_backup a "${PORT_A}" "${TOKEN_A}")"
[[ -s "${BACKUP_A}" ]] || fail "instance A backup archive missing"
[[ -z "$(find "${SMOKE_ROOT}/b/backups" -maxdepth 1 -type f -print -quit)" ]] || fail "A backup leaked into B"
curl -fsS "http://127.0.0.1:${PORT_B}/health" >/dev/null
assert_article "${PORT_B}" shared-slug "Instance B"

create_article "${PORT_A}" "${TOKEN_A}" transient-after-backup "Transient A"
assert_article "${PORT_A}" transient-after-backup "Transient A"

log "upgrading and rolling back A's release symlink while B stays online"
stop_pid "${PID_A}"
PID_A=""
switch_version a v2
PID_A="$(start_instance a "${PORT_A}" A-upgrade)"
curl -fsS "http://127.0.0.1:${PORT_B}/health" >/dev/null
assert_article "${PORT_B}" shared-slug "Instance B"
assert_article "${PORT_A}" transient-after-backup "Transient A"

stop_pid "${PID_A}"
PID_A=""
switch_version a v1
PID_A="$(start_instance a "${PORT_A}" A-rollback)"
curl -fsS "http://127.0.0.1:${PORT_B}/health" >/dev/null
assert_article "${PORT_B}" shared-slug "Instance B"

log "restoring A from its real backup while B stays online"
stop_pid "${PID_A}"
PID_A=""
rm -f "${SMOKE_ROOT}/a/data/impress.db-wal" "${SMOKE_ROOT}/a/data/impress.db-shm"
gzip -dc "${BACKUP_A}" >"${SMOKE_ROOT}/a/data/impress.db"
PID_A="$(start_instance a "${PORT_A}" A-restore)"
assert_article "${PORT_A}" shared-slug "Instance A"
assert_article_missing "${PORT_A}" transient-after-backup
curl -fsS "http://127.0.0.1:${PORT_B}/health" >/dev/null
assert_article "${PORT_B}" shared-slug "Instance B"

log "stopping B and proving restarted A remains available"
stop_pid "${PID_B}"
PID_B=""
curl -fsS "http://127.0.0.1:${PORT_A}/health" >/dev/null
assert_article "${PORT_A}" shared-slug "Instance A"

[[ -s "${SMOKE_ROOT}/a/data/impress.db" && -s "${SMOKE_ROOT}/b/data/impress.db" ]] || fail "isolated databases missing"
[[ -s "${BACKUP_A}" ]] || fail "instance A backup missing"
[[ "$(cat "${SMOKE_ROOT}/a/plugin-data/shared-plugin.state")" == "plugin state A" ]] || fail "instance A plugin state changed"
[[ "$(cat "${SMOKE_ROOT}/b/plugin-data/shared-plugin.state")" == "plugin state B" ]] || fail "instance B plugin state changed"
[[ "$(cat "${SMOKE_ROOT}/a/plugins/shared-plugin.package")" == "plugin package A" ]] || fail "instance A plugin package changed"
[[ "$(cat "${SMOKE_ROOT}/b/plugins/shared-plugin.package")" == "plugin package B" ]] || fail "instance B plugin package changed"

log "PASS: ports, databases, uploads, plugin paths, real backup/restore, release upgrade/rollback, stop, and restart are isolated"
