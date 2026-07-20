#!/usr/bin/env bash
# Bootstrap a second Inkless process for inkless.run product ops site.
# Does NOT touch /opt/inkless (yx.ink personal site) data.
set -euo pipefail

PERSONAL_ROOT="${PERSONAL_ROOT:-/opt/inkless}"
PRODUCT_ROOT="${PRODUCT_ROOT:-/opt/inkless-ops}"
PRODUCT_PORT="${PRODUCT_PORT:-8089}"
PRODUCT_BASE_URL="${PRODUCT_BASE_URL:-https://inkless.run}"
UNIT_NAME="${UNIT_NAME:-inkless-ops}"

if [[ ! -x "${PERSONAL_ROOT}/backend/current/inkless-api-latest" ]]; then
  echo "missing personal binary at ${PERSONAL_ROOT}/backend/current/inkless-api-latest" >&2
  exit 1
fi

id inkless >/dev/null 2>&1 || useradd --system --home "${PERSONAL_ROOT}" --shell /usr/sbin/nologin inkless

mkdir -p \
  "${PRODUCT_ROOT}/backend" \
  "${PRODUCT_ROOT}/frontend" \
  "${PRODUCT_ROOT}/data" \
  "${PRODUCT_ROOT}/uploads" \
  "${PRODUCT_ROOT}/backups" \
  "${PRODUCT_ROOT}/plugins" \
  "${PRODUCT_ROOT}/data/plugins"

# Share release binaries/assets via symlink (same code version; separate runtime state).
ln -sfn "${PERSONAL_ROOT}/backend/current" "${PRODUCT_ROOT}/backend/current"
ln -sfn "${PERSONAL_ROOT}/frontend/current" "${PRODUCT_ROOT}/frontend/current"

# Independent JWT secrets for the product process.
JWT_SECRET="$(openssl rand -hex 32)"
JWT_REFRESH_SECRET="$(openssl rand -hex 32)"

cat >"${PRODUCT_ROOT}/backend/.env" <<EOF
PORT=${PRODUCT_PORT}
ENV=production
SEED_MODE=blank
SETUP_BOOTSTRAP=true
FRONTEND_DIR=${PRODUCT_ROOT}/frontend/current
UPLOAD_DIR=${PRODUCT_ROOT}/uploads
BACKUP_DIR=${PRODUCT_ROOT}/backups
PLUGIN_DIR=${PRODUCT_ROOT}/plugins
PLUGIN_DATA_DIR=${PRODUCT_ROOT}/data/plugins
ENABLE_EXTERNAL_PLUGINS=false
BASE_URL=${PRODUCT_BASE_URL}
CORS_ALLOWED_ORIGINS=${PRODUCT_BASE_URL}
DB_DSN=file:${PRODUCT_ROOT}/data/inkless.db?cache=shared&mode=rwc
JWT_SECRET=${JWT_SECRET}
JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
EOF
chmod 600 "${PRODUCT_ROOT}/backend/.env"

chown -R inkless:inkless \
  "${PRODUCT_ROOT}/data" \
  "${PRODUCT_ROOT}/uploads" \
  "${PRODUCT_ROOT}/backups" \
  "${PRODUCT_ROOT}/plugins" \
  "${PRODUCT_ROOT}/backend/.env"

# systemd unit for product process
cat >"/etc/systemd/system/${UNIT_NAME}.service" <<EOF
[Unit]
Description=Inkless CMS product site (inkless.run)
After=network.target
Wants=network.target

[Service]
Type=simple
User=inkless
Group=inkless
WorkingDirectory=${PRODUCT_ROOT}/backend/current
EnvironmentFile=-${PRODUCT_ROOT}/backend/.env
ExecStart=${PRODUCT_ROOT}/backend/current/inkless-api-latest
Restart=on-failure
RestartSec=5s

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${PRODUCT_ROOT}/data ${PRODUCT_ROOT}/uploads ${PRODUCT_ROOT}/backups ${PRODUCT_ROOT}/plugins ${PRODUCT_ROOT}/data/plugins

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${UNIT_NAME}.service"
systemctl restart "${UNIT_NAME}.service"
sleep 2
systemctl is-active "${UNIT_NAME}.service"
curl -fsS "http://127.0.0.1:${PRODUCT_PORT}/health"
echo
echo "bootstrap ok: ${PRODUCT_ROOT} :${PRODUCT_PORT} unit=${UNIT_NAME}"
