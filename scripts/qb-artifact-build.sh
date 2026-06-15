#!/usr/bin/env bash
# Quick-Box artifact build (runs on build server).
# Env: QB_WORKDIR, QB_VERSION, QB_ARTIFACT_STAGING, QB_BUILD_COMPONENTS
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=qb-artifact-common.sh
source "${SCRIPT_DIR}/qb-artifact-common.sh"

WORKDIR="${QB_WORKDIR:-$(qb_repo_root)}"
STAGING="${QB_ARTIFACT_STAGING:?QB_ARTIFACT_STAGING is required}"
VERSION="$(qb_resolve_version "${WORKDIR}")"
export VERSION

qb_log_info "build start version=${VERSION} workdir=${WORKDIR} staging=${STAGING}"

cd "${WORKDIR}"
mkdir -p "${STAGING}"

if [[ "${QB_ENSURE_HOST_DEPS:-true}" == "true" ]]; then
  qb_log_info "ensuring host build dependencies"
  bash "${WORKDIR}/scripts/qb-artifact-ensure-deps.sh"
  export PATH="/usr/local/go/bin:${PATH}"
  export GOTOOLCHAIN=local
fi

if qb_component_enabled "backend"; then
  qb_log_info "building backend component"
  if [[ "${QB_SKIP_BACKEND_TESTS:-false}" == "true" ]]; then
  qb_log_warn "QB_SKIP_BACKEND_TESTS=true — using fast go build (no test/vet gate)"
    mkdir -p "${WORKDIR}/artifacts"
    qb_run_with_heartbeat bash -c "cd '${WORKDIR}/backend' && CGO_ENABLED=1 go build -v -ldflags='-s -w' -o '${WORKDIR}/artifacts/blotting-api-${VERSION}' ./cmd/server"
    chmod +x "${WORKDIR}/artifacts/blotting-api-${VERSION}"
    cat >"${WORKDIR}/artifacts/build-info.json" <<EOF
{"version":"${VERSION}","buildTime":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","gitCommit":"$(git -C "${WORKDIR}" rev-parse HEAD 2>/dev/null || echo unknown)"}
EOF
    (cd "${WORKDIR}/artifacts" && tar -czf "backend-${VERSION}.tar.gz" "blotting-api-${VERSION}" build-info.json)
    (cd "${WORKDIR}/artifacts" && sha256sum "backend-${VERSION}.tar.gz" > "backend-${VERSION}.tar.gz.sha256")
  else
    "${WORKDIR}/scripts/build-backend.sh"
  fi
  cp "${WORKDIR}/artifacts/backend-${VERSION}.tar.gz" "${STAGING}/"
  if [[ -f "${WORKDIR}/artifacts/backend-${VERSION}.tar.gz.sha256" ]]; then
    cp "${WORKDIR}/artifacts/backend-${VERSION}.tar.gz.sha256" "${STAGING}/"
  fi
fi

if qb_component_enabled "frontend"; then
  qb_log_info "building frontend component"
  if [[ "${QB_SKIP_FRONTEND_CHECKS:-true}" == "true" ]]; then
    qb_log_warn "QB_SKIP_FRONTEND_CHECKS=true — vite build only"
    qb_run_with_heartbeat bash -c "cd '${WORKDIR}/frontend' && pnpm install --frozen-lockfile && pnpm build"
    mkdir -p "${WORKDIR}/artifacts"
    (cd "${WORKDIR}/frontend/out" && tar -czf "${WORKDIR}/artifacts/frontend-${VERSION}.tar.gz" .)
    (cd "${WORKDIR}/artifacts" && sha256sum "frontend-${VERSION}.tar.gz" > "frontend-${VERSION}.tar.gz.sha256" 2>/dev/null || shasum -a 256 "frontend-${VERSION}.tar.gz" > "frontend-${VERSION}.tar.gz.sha256")
  else
    "${WORKDIR}/scripts/build-frontend.sh"
  fi
  cp "${WORKDIR}/artifacts/frontend-${VERSION}.tar.gz" "${STAGING}/"
  if [[ -f "${WORKDIR}/artifacts/frontend-${VERSION}.tar.gz.sha256" ]]; then
    cp "${WORKDIR}/artifacts/frontend-${VERSION}.tar.gz.sha256" "${STAGING}/"
  fi
fi

if [[ -f "${WORKDIR}/artifacts/build-info.json" ]]; then
  cp "${WORKDIR}/artifacts/build-info.json" "${STAGING}/"
fi

# Ship activate helpers with the bundle (deploy server may not have a git checkout).
mkdir -p "${STAGING}/scripts" "${STAGING}/ops/systemd"
cp "${WORKDIR}/scripts/qb-artifact-common.sh" \
   "${WORKDIR}/scripts/qb-artifact-activate.sh" \
   "${WORKDIR}/scripts/qb-artifact-rollback.sh" \
   "${STAGING}/scripts/"
cp "${WORKDIR}/ops/systemd/impress.service" "${STAGING}/ops/systemd/"

QB_VERSION="${VERSION}" QB_ARTIFACT_STAGING="${STAGING}" QB_WORKDIR="${WORKDIR}" \
  "${SCRIPT_DIR}/qb-artifact-manifest.sh" >"${STAGING}/artifact-manifest.json"

qb_log_info "artifact bundle ready:"
ls -la "${STAGING}"
qb_log_info "manifest:"
cat "${STAGING}/artifact-manifest.json"
