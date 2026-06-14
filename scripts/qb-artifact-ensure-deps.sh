#!/usr/bin/env bash
# Install build toolchain on bare-metal deploy/build hosts when missing.
set -euo pipefail

qb_log_info() { echo "[qb-artifact][INFO] $*"; }

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
  if command -v go >/dev/null 2>&1; then
    qb_log_info "go present: $(go version)"
    return 0
  fi
  local ver="${QB_GO_VERSION:-1.24.2}"
  local arch
  arch="$(uname -m)"
  case "${arch}" in
    x86_64) arch="amd64" ;;
    aarch64) arch="arm64" ;;
    *) qb_log_info "unsupported arch for go bootstrap: ${arch}"; return 1 ;;
  esac
  qb_log_info "installing go ${ver} (${arch})"
  curl -fsSL "https://go.dev/dl/go${ver}.linux-${arch}.tar.gz" | tar -C /usr/local -xzf -
  export PATH="/usr/local/go/bin:${PATH}"
  ln -sf /usr/local/go/bin/go /usr/local/bin/go 2>/dev/null || true
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
