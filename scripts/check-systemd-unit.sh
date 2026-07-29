#!/usr/bin/env bash
# Validate the production Inkless systemd unit on macOS and Linux CI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
UNIT="${ROOT}/ops/systemd/inkless.service"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

[[ -f "${UNIT}" ]] || fail "missing systemd unit: ${UNIT}"

required_lines=(
  "Description=Inkless CMS (API + SPA)"
  "User=inkless"
  "Group=inkless"
  "WorkingDirectory=/opt/inkless/backend/current"
  "EnvironmentFile=-/opt/inkless/backend/.env"
  "ExecStart=/opt/inkless/backend/current/inkless-api-latest"
  "NoNewPrivileges=true"
  "PrivateTmp=true"
  "ProtectSystem=strict"
  "ProtectHome=true"
  # Includes backend/frontend/var for optional host self-update staging.
  "ReadWritePaths=/opt/inkless/data /opt/inkless/uploads /opt/inkless/backups /opt/inkless/plugins /opt/inkless/data/plugins /opt/inkless/backend /opt/inkless/frontend /opt/inkless/var"
  "WantedBy=multi-user.target"
)

for line in "${required_lines[@]}"; do
  grep -Fqx "${line}" "${UNIT}" || fail "required directive missing: ${line}"
done

if command -v systemd-analyze >/dev/null 2>&1; then
  verify_dir="$(mktemp -d)"
  trap 'rm -rf "${verify_dir}"' EXIT

  # Host-specific paths and accounts are checked exactly above. Normalize only
  # those directives so systemd-analyze can parse the unit on a generic CI host.
  sed \
    -e '/^User=/d' \
    -e '/^Group=/d' \
    -e '/^WorkingDirectory=/d' \
    -e '/^EnvironmentFile=/d' \
    -e '/^ReadWritePaths=/d' \
    -e 's|^ExecStart=.*$|ExecStart=/bin/true|' \
    "${UNIT}" >"${verify_dir}/inkless.service"

  systemd-analyze verify "${verify_dir}/inkless.service"
  echo "OK: systemd-analyze verified the normalized Inkless unit."
else
  echo "SKIP: systemd-analyze unavailable; exact directive validation passed."
fi
