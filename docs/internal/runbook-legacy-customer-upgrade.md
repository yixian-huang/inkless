# Runbook: upgrade a legacy single-host install

**Audience:** maintainers upgrading an early Impress/Inkless hand-deployed host  
**Transport:** prefer `npc` (or your control plane); do not embed SSH keys in chat  

This is a **generic** checklist. Fill placeholders from handoff / inventory —
never commit real customer hostnames, IPs, or DB names.

| Placeholder | Meaning |
|-------------|---------|
| `<server-ref>` | NoPanel alias or inventory id |
| `<APP_ROOT>` | Legacy tree (e.g. `/home/app/impress`) |
| `<UNIT>` | systemd unit for the API |
| `<PROD_PORT>` | Backend listen port |
| `<CANARY_PORT>` | Parallel canary port (must not clash) |
| `<DB_PATH>` | SQLite path (or set Postgres DSN) |
| `<PUBLIC_URL>` | Customer public origin |

**Default:** do **not** cut over traffic until preflight + canary pass and a human approves.

---

## 0. Risks and success criteria

| Risk | Notes |
|------|--------|
| Schema forward-only | New binary runs migrations; hard to roll schema back |
| Legacy layout | May lack `versions/current/previous` — build rollback layout first |
| Static assets | New frontend may use `out/`; nginx may still expect old paths |
| Branding | Admin UI may show Inkless product brand; content stays in DB |

**Success (within ~15 minutes of cutover):**

1. `<PUBLIC_URL>/` returns 200; expected theme still renders  
2. `/public/bootstrap` OK (no 5xx)  
3. Admin login works for existing roles  
4. Analytics/admin APIs that previously worked still authorize correctly  
5. Historical `/uploads/...` still reachable  
6. Rollback to previous binary + DB backup possible within the window  

---

## 1. Constants (operator machine)

```bash
export SERVER_REF="<server-ref>"
export APP_ROOT="<APP_ROOT>"
export PROD_PORT="<PROD_PORT>"
export CANARY_PORT="<CANARY_PORT>"
export UNIT="<UNIT>"
export DB_PATH="${APP_ROOT}/data/<site>.db"
export UPLOAD_DIR="${APP_ROOT}/uploads"
export FRONTEND_DIR="${APP_ROOT}/frontend"
export BACKUP_ROOT="${APP_ROOT}/backups"
```

---

## 2. Preflight

1. `npc server handoff brief "$SERVER_REF" --section agentSummary -o json`  
2. Confirm disk space, unit active, port, and proxy upstream  
3. Cold backup DB + uploads + current binary  
4. Record current version / git SHA if any  

```bash
npc server exec command "$SERVER_REF" -- \
  "systemctl is-active ${UNIT}; ss -lntp | grep ${PROD_PORT} || true; df -h ${APP_ROOT}"
```

---

## 3. Build artifacts (builder, not prod if possible)

```bash
# on a build machine / CI
VERSION=vX.Y.Z ./scripts/build-frontend.sh
VERSION=vX.Y.Z ./scripts/build-backend.sh
# transfer artifacts/frontend-*.tar.gz and artifacts/backend-*.tar.gz to host
```

Or use your control plane package/deploy flow.

---

## 4. Canary (parallel process, no traffic cut)

1. Extract new backend/frontend under a canary tree  
2. Copy DB to a canary file (do not point canary at live DB for schema experiments)  
3. Start canary on `<CANARY_PORT>` with its own `DB_DSN` / `UPLOAD_DIR` as needed  
4. Smoke: health, bootstrap, admin login, key content pages  
5. Stop canary; keep artifacts for cutover  

---

## 5. Cutover (maintenance window)

1. Enable maintenance / freeze writes if needed  
2. Final DB backup  
3. Stop `<UNIT>`  
4. Install new binary + frontend; keep previous paths for rollback  
5. Start unit; watch logs and migrations  
6. Point proxy at new static/API layout if paths changed  
7. Verify success criteria  
8. Only then clear maintenance  

---

## 6. Rollback

1. Stop unit  
2. Restore previous binary + frontend symlink/tree  
3. Restore DB backup if schema already advanced and site is broken  
4. Start unit; verify public URL  

---

## 7. Aftercare

- Rotate any credentials that were shared during migration  
- Document the new tree (`/opt/inkless` layout preferred going forward)  
- Schedule removal of legacy paths after backup retention expires  
