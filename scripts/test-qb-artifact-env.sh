#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=qb-artifact-common.sh
source "${SCRIPT_DIR}/qb-artifact-common.sh"

TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/impress-qb-env.XXXXXX")"
trap 'rm -rf "${TEST_ROOT}"' EXIT

fail() {
  echo "[qb-env-test][FAIL] $*" >&2
  exit 1
}

assert_line() {
  local expected="$1"
  local file="$2"
  grep -Fqx "${expected}" "${file}" || fail "missing '${expected}' in ${file}"
}

default_root="${TEST_ROOT}/default"
default_env="${default_root}/backend/.env"
qb_write_env_file "${default_env}" "${default_root}"
assert_line "PORT=8088" "${default_env}"
assert_line "BASE_URL=http://127.0.0.1:8088" "${default_env}"
assert_line "CORS_ALLOWED_ORIGINS=http://127.0.0.1:8088" "${default_env}"
assert_line "PLUGIN_DIR=${default_root}/plugins" "${default_env}"
assert_line "BACKUP_DIR=${default_root}/backups" "${default_env}"
assert_line "PLUGIN_DATA_DIR=${default_root}/data/plugins" "${default_env}"
assert_line "ENABLE_EXTERNAL_PLUGINS=false" "${default_env}"

custom_root="${TEST_ROOT}/custom"
custom_env="${custom_root}/backend/.env"
PORT=19088 \
BASE_URL=https://primary.example.test \
CORS_ALLOWED_ORIGINS=https://primary.example.test,https://admin.example.test \
PLUGIN_DIR="${custom_root}/plugin-packages" \
BACKUP_DIR="${custom_root}/backup-archives" \
PLUGIN_DATA_DIR="${custom_root}/plugin-state" \
ENABLE_EXTERNAL_PLUGINS=true \
qb_write_env_file "${custom_env}" "${custom_root}"
assert_line "PORT=19088" "${custom_env}"
assert_line "BASE_URL=https://primary.example.test" "${custom_env}"
assert_line "CORS_ALLOWED_ORIGINS=https://primary.example.test,https://admin.example.test" "${custom_env}"
assert_line "PLUGIN_DIR=${custom_root}/plugin-packages" "${custom_env}"
assert_line "BACKUP_DIR=${custom_root}/backup-archives" "${custom_env}"
assert_line "PLUGIN_DATA_DIR=${custom_root}/plugin-state" "${custom_env}"
assert_line "ENABLE_EXTERNAL_PLUGINS=true" "${custom_env}"

unset PORT BASE_URL CORS_ALLOWED_ORIGINS PLUGIN_DIR PLUGIN_DATA_DIR BACKUP_DIR ENABLE_EXTERNAL_PLUGINS
qb_load_env_file_defaults "${custom_env}"
[[ "${PORT}" == "19088" ]] || fail "persisted PORT was not loaded"
[[ "${BASE_URL}" == "https://primary.example.test" ]] || fail "persisted BASE_URL was not loaded"
[[ "${BACKUP_DIR}" == "${custom_root}/backup-archives" ]] || fail "persisted BACKUP_DIR was not loaded"
[[ "$(qb_health_url)" == "http://127.0.0.1:19088/health" ]] || fail "health URL must follow the instance port"

PORT=29088
BASE_URL=https://override.example.test
qb_load_env_file_defaults "${custom_env}"
[[ "${PORT}" == "29088" ]] || fail "explicit PORT must override persisted value"
[[ "${BASE_URL}" == "https://override.example.test" ]] || fail "explicit BASE_URL must override persisted value"
[[ "$(qb_health_url)" == "http://127.0.0.1:29088/health" ]] || fail "health URL must follow the explicit instance port"

mode="$(stat -c '%a' "${custom_env}" 2>/dev/null || stat -f '%Lp' "${custom_env}")"
[[ "${mode}" == "600" ]] || fail "env file mode must be 600, got ${mode}"

rollback_root="${TEST_ROOT}/rollback"
mkdir -p "${rollback_root}/frontend/versions/current-version" \
  "${rollback_root}/frontend/versions/previous-version" \
  "${rollback_root}/backend/versions/current-version"
ln -s "${rollback_root}/frontend/versions/current-version" "${rollback_root}/frontend/current"
ln -s "${rollback_root}/frontend/versions/previous-version" "${rollback_root}/frontend/previous"
ln -s "${rollback_root}/backend/versions/current-version" "${rollback_root}/backend/current"
if QB_RELEASE_ROOT="${rollback_root}" COMPONENT=all TARGET_VERSION=previous \
  "${SCRIPT_DIR}/qb-artifact-rollback.sh" >/dev/null 2>&1; then
  fail "all rollback should fail preflight when one component has no previous release"
fi
[[ "$(readlink "${rollback_root}/frontend/current")" == "${rollback_root}/frontend/versions/current-version" ]] || \
  fail "failed all rollback partially switched the frontend"

echo "[qb-env-test][PASS] env precedence, instance paths, file mode, and atomic rollback preflight"
