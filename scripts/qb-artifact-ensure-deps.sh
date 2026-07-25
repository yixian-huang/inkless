#!/usr/bin/env bash
# Install build toolchain on bare-metal deploy/build hosts when missing.
set -euo pipefail

qb_log_info() { echo "[qb-artifact][INFO] $*"; }
qb_log_warn() { echo "[qb-artifact][WARN] $*" >&2; }

# True if $1 >= $2 (semver-ish, uses sort -V).
version_ge() {
  local a="$1" b="$2"
  [[ -n "${a}" && -n "${b}" ]] || return 1
  [[ "$(printf '%s\n%s\n' "${b}" "${a}" | sort -V | head -1)" == "${b}" ]]
}

resolve_go_version() {
  # 1) explicit env  2) backend/go.mod in cwd or WORKDIR  3) fallback pin
  if [[ -n "${QB_GO_VERSION:-}" ]]; then
    echo "${QB_GO_VERSION}"
    return 0
  fi
  local mod=""
  if [[ -f backend/go.mod ]]; then
    mod="backend/go.mod"
  elif [[ -n "${WORKDIR:-}" && -f "${WORKDIR}/backend/go.mod" ]]; then
    mod="${WORKDIR}/backend/go.mod"
  elif [[ -f go.mod ]]; then
    mod="go.mod"
  fi
  if [[ -n "${mod}" ]]; then
    local from_mod
    from_mod="$(awk '/^go[[:space:]]+/ { print $2; exit }' "${mod}" 2>/dev/null || true)"
    if [[ -n "${from_mod}" ]]; then
      echo "${from_mod}"
      return 0
    fi
  fi
  echo "1.25.7"
}

ensure_build_essential() {
  if command -v gcc >/dev/null 2>&1; then
    qb_log_info "gcc present: $(gcc --version | head -1)"
    return 0
  fi
  qb_log_info "installing build-essential (gcc for CGO/SQLite)"
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq
    apt-get install -y build-essential
  elif command -v yum >/dev/null 2>&1; then
    yum groupinstall -y "Development Tools"
  else
    qb_log_info "no supported package manager for build-essential"
    return 1
  fi
}

ensure_go() {
  local ver
  ver="$(resolve_go_version)"
  export PATH="/usr/local/go/bin:${PATH}"

  if command -v go >/dev/null 2>&1; then
    local cur
    cur="$(go version 2>/dev/null | awk '{print $3}' | sed 's/^go//')"
    if version_ge "${cur}" "${ver}"; then
      qb_log_info "go present and sufficient: $(go version) (need >= ${ver})"
      return 0
    fi
    qb_log_warn "upgrading go to ${ver} (was: $(go version 2>/dev/null || echo missing))"
  fi

  local arch
  arch="$(uname -m)"
  case "${arch}" in
    x86_64) arch="amd64" ;;
    aarch64) arch="arm64" ;;
    *) qb_log_info "unsupported arch for go bootstrap: ${arch}"; return 1 ;;
  esac
  qb_log_info "installing go ${ver} (${arch})"
  rm -rf /usr/local/go
  curl -fsSL "https://go.dev/dl/go${ver}.linux-${arch}.tar.gz" | tar -C /usr/local -xzf -
  export PATH="/usr/local/go/bin:${PATH}"
  ln -sf /usr/local/go/bin/go /usr/local/bin/go 2>/dev/null || true
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt 2>/dev/null || true
  go version
}

ensure_node_pnpm() {
  if command -v pnpm >/dev/null 2>&1 && command -v node >/dev/null 2>&1; then
    qb_log_info "node=$(node --version) pnpm=$(pnpm --version)"
    return 0
  fi
  if ! command -v node >/dev/null 2>&1; then
    qb_log_info "installing nodejs 20.x"
    if command -v apt-get >/dev/null 2>&1; then
      curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
      apt-get install -y nodejs
    elif command -v yum >/dev/null 2>&1; then
      curl -fsSL https://rpm.nodesource.com/setup_20.x | bash -
      yum install -y nodejs
    else
      qb_log_info "no supported package manager for node install"
      return 1
    fi
  fi
  if ! command -v pnpm >/dev/null 2>&1; then
    qb_log_info "installing pnpm"
    npm install -g pnpm
  fi
  node --version
  pnpm --version
}

ensure_build_essential
ensure_go
ensure_node_pnpm
