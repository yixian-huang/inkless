#!/usr/bin/env bash
# Split a site tree that currently symlinks backend/frontend current to /opt/inkless
# into a private versions/ copy under SITE_ROOT.
#
# DEFAULT: DRY_RUN=1 (print only). Set DRY_RUN=0 to apply.
#
# Usage (on gomami as root):
#   SITE_ROOT=/opt/inkless-ops UNIT=inkless-ops PORT=8089 bash scripts/ops-split-shared-current.sh
#   DRY_RUN=0 SITE_ROOT=/opt/inkless-ops UNIT=inkless-ops PORT=8089 bash scripts/ops-split-shared-current.sh
#
# See: docs/internal/runbook-split-shared-release-roots-gomami.md
set -euo pipefail

DRY_RUN="${DRY_RUN:-1}"
SRC_ROOT="${SRC_ROOT:-/opt/inkless}"
SITE_ROOT="${SITE_ROOT:-}"
UNIT="${UNIT:-}"
PORT="${PORT:-}"
SKIP_RESTART="${SKIP_RESTART:-0}"

log() { printf '[split-current] %s\n' "$*"; }
die() { printf '[split-current] ERROR: %s\n' "$*" >&2; exit 1; }
run() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "DRY_RUN: $*"
  else
    log "RUN: $*"
    eval "$@"
  fi
}

[[ "$(id -u)" -eq 0 ]] || die "must run as root"
[[ -n "${SITE_ROOT}" ]] || die "SITE_ROOT required (e.g. /opt/inkless-ops)"
[[ -n "${UNIT}" ]] || die "UNIT required (e.g. inkless-ops)"
[[ -n "${PORT}" ]] || die "PORT required (e.g. 8089)"
[[ "${SITE_ROOT}" != "/opt/inkless" ]] || die "refusing to split personal authority tree /opt/inkless"
[[ "${SITE_ROOT}" != "${SRC_ROOT}" ]] || die "SITE_ROOT must differ from SRC_ROOT"
[[ -d "${SITE_ROOT}" ]] || die "SITE_ROOT missing: ${SITE_ROOT}"
[[ -d "${SRC_ROOT}/backend/current" || -L "${SRC_ROOT}/backend/current" ]] || die "source current missing"

command -v rsync >/dev/null 2>&1 || die "rsync required"
command -v systemctl >/dev/null 2>&1 || die "systemctl required"

VER="$(basename "$(readlink -f "${SRC_ROOT}/backend/current")")"
FVER="$(basename "$(readlink -f "${SRC_ROOT}/frontend/current")")"
[[ "${VER}" == "${FVER}" ]] || die "backend VER=${VER} != frontend VER=${FVER}"

B_SRC="${SRC_ROOT}/backend/versions/${VER}"
F_SRC="${SRC_ROOT}/frontend/versions/${VER}"
[[ -d "${B_SRC}" ]] || die "missing ${B_SRC}"
[[ -d "${F_SRC}" ]] || die "missing ${F_SRC}"

B_DST="${SITE_ROOT}/backend/versions/${VER}"
F_DST="${SITE_ROOT}/frontend/versions/${VER}"

log "DRY_RUN=${DRY_RUN} SRC_ROOT=${SRC_ROOT} SITE_ROOT=${SITE_ROOT} UNIT=${UNIT} PORT=${PORT} VER=${VER}"
log "src backend realpath: $(readlink -f "${SRC_ROOT}/backend/current")"
log "site backend realpath now: $(readlink -f "${SITE_ROOT}/backend/current" 2>/dev/null || echo MISSING)"

# Already split?
SITE_REAL="$(readlink -f "${SITE_ROOT}/backend/current" 2>/dev/null || true)"
if [[ -n "${SITE_REAL}" && "${SITE_REAL}" == "${SITE_ROOT}/backend/versions/"* ]]; then
  log "already split (backend current under SITE_ROOT). Nothing to do."
  exit 0
fi

if [[ "${SITE_REAL}" != "${SRC_ROOT}/backend/versions/"* && -n "${SITE_REAL}" ]]; then
  die "unexpected site backend realpath (not shared with SRC and not local versions): ${SITE_REAL}"
fi

# Disk rough check: need ~2x version size free on /opt
NEED_KB="$(du -sk "${B_SRC}" "${F_SRC}" | awk '{s+=$1} END{print s+102400}')"
AVAIL_KB="$(df -Pk /opt | awk 'NR==2{print $4}')"
if [[ "${AVAIL_KB}" -lt "${NEED_KB}" ]]; then
  die "low disk on /opt: avail_kb=${AVAIL_KB} need_kb~=${NEED_KB}"
fi

TS="$(date -u +%Y%m%dT%H%M%SZ)"
BACKUP_DIR="${SITE_ROOT}/.split-backup-${TS}"

run "mkdir -p '${BACKUP_DIR}'"
if [[ "${DRY_RUN}" == "1" ]]; then
  log "WOULD backup symlinks to ${BACKUP_DIR}"
  log "WOULD rsync ${B_SRC}/ -> ${B_DST}/"
  log "WOULD rsync ${F_SRC}/ -> ${F_DST}/"
  log "WOULD ln -sfn ${B_DST} ${SITE_ROOT}/backend/current"
  log "WOULD ln -sfn ${F_DST} ${SITE_ROOT}/frontend/current"
  if [[ "${SKIP_RESTART}" != "1" ]]; then
    log "WOULD systemctl restart ${UNIT}"
  fi
  log "dry-run complete; re-run with DRY_RUN=0 to apply"
  exit 0
fi

mkdir -p "${BACKUP_DIR}"
if [[ -L "${SITE_ROOT}/backend/current" || -e "${SITE_ROOT}/backend/current" ]]; then
  cp -a "${SITE_ROOT}/backend/current" "${BACKUP_DIR}/backend.current.link"
  readlink "${SITE_ROOT}/backend/current" >"${BACKUP_DIR}/backend.current.readlink.txt" || true
fi
if [[ -L "${SITE_ROOT}/frontend/current" || -e "${SITE_ROOT}/frontend/current" ]]; then
  cp -a "${SITE_ROOT}/frontend/current" "${BACKUP_DIR}/frontend.current.link"
  readlink "${SITE_ROOT}/frontend/current" >"${BACKUP_DIR}/frontend.current.readlink.txt" || true
fi
echo "${VER}" >"${BACKUP_DIR}/version.txt"
log "backup=${BACKUP_DIR}"

mkdir -p "${SITE_ROOT}/backend/versions" "${SITE_ROOT}/frontend/versions"
if [[ -d "${B_DST}" ]]; then
  mv "${B_DST}" "${B_DST}.pre-split-${TS}"
fi
if [[ -d "${F_DST}" ]]; then
  mv "${F_DST}" "${F_DST}.pre-split-${TS}"
fi

rsync -a --delete "${B_SRC}/" "${B_DST}/"
rsync -a --delete "${F_SRC}/" "${F_DST}/"

chmod a+x "${B_DST}/inkless-api-"* 2>/dev/null || true
if [[ ! -e "${B_DST}/inkless-api-latest" ]]; then
  BIN="$(find "${B_DST}" -maxdepth 1 -type f -name 'inkless-api-*' ! -name 'inkless-api-latest' | head -1)"
  [[ -n "${BIN}" ]] || die "no inkless-api binary in ${B_DST}"
  ln -sfn "$(basename "${BIN}")" "${B_DST}/inkless-api-latest"
fi
[[ -f "${F_DST}/index.html" ]] || die "frontend index.html missing after rsync"

ln -sfn "${B_DST}" "${SITE_ROOT}/backend/current"
ln -sfn "${F_DST}" "${SITE_ROOT}/frontend/current"

NEW_B="$(readlink -f "${SITE_ROOT}/backend/current")"
NEW_F="$(readlink -f "${SITE_ROOT}/frontend/current")"
[[ "${NEW_B}" == "${SITE_ROOT}/backend/versions/"* ]] || die "backend current not under site versions: ${NEW_B}"
[[ "${NEW_F}" == "${SITE_ROOT}/frontend/versions/"* ]] || die "frontend current not under site versions: ${NEW_F}"
[[ "${NEW_B}" != /opt/inkless/* ]] || die "backend still under /opt/inkless"

if [[ "${SKIP_RESTART}" != "1" ]]; then
  systemctl restart "${UNIT}"
  sleep 2
  systemctl is-active --quiet "${UNIT}" || die "unit ${UNIT} not active after restart"
fi

CODE="$(curl -sS -m 5 -o /dev/null -w '%{http_code}' "http://127.0.0.1:${PORT}/health" || true)"
[[ "${CODE}" == "200" ]] || die "health check failed on :${PORT} code=${CODE} — restore from ${BACKUP_DIR}"

log "SUCCESS site=${SITE_ROOT} ver=${VER} backend=${NEW_B}"
log "rollback: ln -sfn \$(cat ${BACKUP_DIR}/backend.current.readlink.txt) ${SITE_ROOT}/backend/current && same for frontend; systemctl restart ${UNIT}"
