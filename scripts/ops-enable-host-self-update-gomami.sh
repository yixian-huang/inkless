#!/usr/bin/env bash
# Enable host self-update env + unit write paths + sudoers on a multi-instance host.
# Run as root on the app host (e.g. gomami). Idempotent.
#
# Sites (default):
#   /opt/inkless       unit=inkless       (yx.ink)
#   /opt/inkless-ops   unit=inkless-ops   (inkless.run)
#   /opt/inkless-imgli unit=inkless-imgli (imgli.com)
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "must run as root" >&2
  exit 1
fi

SITES=(
  "/opt/inkless|inkless"
  "/opt/inkless-ops|inkless-ops"
  "/opt/inkless-imgli|inkless-imgli"
)

upsert_env() {
  local file="$1" key="$2" val="$3"
  mkdir -p "$(dirname "$file")"
  touch "$file"
  chmod 600 "$file"
  if grep -qE "^${key}=" "$file" 2>/dev/null; then
    # portable in-place replace
    local tmp
    tmp="$(mktemp)"
    awk -v k="$key" -v v="$val" 'BEGIN{FS=OFS="="} $1==k{$0=k"="v} {print}' "$file" >"$tmp"
    cat "$tmp" >"$file"
    rm -f "$tmp"
  else
    printf '%s=%s\n' "$key" "$val" >>"$file"
  fi
}

ensure_site() {
  local root="$1" unit="$2"
  local envf="${root}/backend/.env"
  local dropin_dir="/etc/systemd/system/${unit}.service.d"
  local dropin="${dropin_dir}/self-update.conf"

  echo "==> site root=${root} unit=${unit}"

  if [[ ! -d "$root" ]]; then
    echo "  skip: root missing"
    return 0
  fi
  if [[ ! -f "$envf" ]]; then
    echo "  skip: env missing $envf"
    return 0
  fi

  # env
  upsert_env "$envf" "INKLESS_SELF_UPDATE_ENABLED" "true"
  upsert_env "$envf" "INKLESS_RELEASE_ROOT" "$root"
  upsert_env "$envf" "INKLESS_SYSTEMD_UNIT" "$unit"
  upsert_env "$envf" "INKLESS_UPDATE_CHANNEL" "stable"
  upsert_env "$envf" "INKLESS_UPDATE_REPO" "yixian-huang/inkless"
  chown inkless:inkless "$envf" 2>/dev/null || true
  chmod 600 "$envf"

  # dirs for staging / versions
  mkdir -p \
    "${root}/backend/versions" \
    "${root}/frontend/versions" \
    "${root}/var/updates/jobs" \
    "${root}/var/updates/incoming"
  # inkless must create version dirs and switch current/previous
  chown -R inkless:inkless \
    "${root}/backend" \
    "${root}/frontend" \
    "${root}/var" 2>/dev/null || true
  # keep .env mode
  chmod 600 "$envf" 2>/dev/null || true

  # systemd drop-in: write paths + allow sudo for restart
  mkdir -p "$dropin_dir"
  cat >"$dropin" <<EOF
[Service]
# Host self-update (H1): stage artifacts + switch current/previous under RELEASE_ROOT
ReadWritePaths=${root}/backend ${root}/frontend ${root}/var ${root}/data ${root}/uploads ${root}/backups ${root}/plugins ${root}/data/plugins
# Required so the service can use passwordless sudo for systemctl restart
NoNewPrivileges=false
EOF
  echo "  wrote $dropin"
}

# Deferred restart helper (process cannot systemctl-restart itself synchronously)
HELPER=/usr/local/sbin/inkless-deferred-restart
cat >"$HELPER" <<'EOF'
#!/bin/bash
set -euo pipefail
unit="${1:-}"
case "$unit" in
  inkless|inkless.service|inkless-ops|inkless-ops.service|inkless-imgli|inkless-imgli.service) ;;
  *) echo "refusing unit: $unit" >&2; exit 2 ;;
esac
unit="${unit%.service}"
nohup /bin/bash -c "sleep 1; /usr/bin/systemctl restart ${unit}.service" \
  >"/tmp/inkless-deferred-restart-${unit}.log" 2>&1 &
echo "scheduled restart ${unit}"
EOF
chmod 755 "$HELPER"
echo "wrote $HELPER"

SUDOERS=/etc/sudoers.d/inkless-self-update
cat >"$SUDOERS" <<'EOF'
# Managed by ops-enable-host-self-update-gomami.sh
inkless ALL=(root) NOPASSWD: /usr/local/sbin/inkless-deferred-restart inkless, /usr/local/sbin/inkless-deferred-restart inkless-ops, /usr/local/sbin/inkless-deferred-restart inkless-imgli
inkless ALL=(root) NOPASSWD: /usr/bin/systemctl restart inkless, /usr/bin/systemctl restart inkless.service, /bin/systemctl restart inkless, /bin/systemctl restart inkless.service
inkless ALL=(root) NOPASSWD: /usr/bin/systemctl restart inkless-ops, /usr/bin/systemctl restart inkless-ops.service, /bin/systemctl restart inkless-ops, /bin/systemctl restart inkless-ops.service
inkless ALL=(root) NOPASSWD: /usr/bin/systemctl restart inkless-imgli, /usr/bin/systemctl restart inkless-imgli.service, /bin/systemctl restart inkless-imgli, /bin/systemctl restart inkless-imgli.service
EOF
chmod 440 "$SUDOERS"
if command -v visudo >/dev/null 2>&1; then
  visudo -cf "$SUDOERS"
fi
echo "wrote $SUDOERS"

for entry in "${SITES[@]}"; do
  IFS='|' read -r root unit <<<"$entry"
  ensure_site "$root" "$unit"
done

systemctl daemon-reload
for entry in "${SITES[@]}"; do
  IFS='|' read -r root unit <<<"$entry"
  if systemctl cat "${unit}.service" >/dev/null 2>&1; then
    systemctl restart "$unit"
    echo "restarted $unit -> $(systemctl is-active "$unit")"
  fi
done

echo
echo "verify env keys:"
for entry in "${SITES[@]}"; do
  IFS='|' read -r root unit <<<"$entry"
  envf="${root}/backend/.env"
  echo "--- $envf ---"
  grep -E '^INKLESS_(SELF_UPDATE|RELEASE_ROOT|SYSTEMD_UNIT|UPDATE_)' "$envf" || true
done

echo
echo "health:"
for p in 8088 8089 8090; do
  code=$(curl -sS -m 3 -o /dev/null -w '%{http_code}' "http://127.0.0.1:${p}/health" || echo fail)
  echo "  :${p} -> ${code}"
done

echo "done"
