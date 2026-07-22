# Theme demo site: themes.inkless.run

**URL:** https://themes.inkless.run  
**Host:** gomami (`103.73.220.161`)  
**Role:** Public Inkless **theme showcase** (currently **editorial-firm**)  
**Isolation:** Separate from yx.ink (`:8088` / `/opt/inkless`) and inkless.run (`:8089` / `/opt/inkless-ops`)

| Item | Value |
|------|--------|
| Container | `inkless-theme-demo` |
| Image | `inkless-theme-demo:latest` |
| Port | `127.0.0.1:8098` → container `:8088` |
| Data | `/opt/inkless-theme-demo/{data,uploads,backups}` |
| Caddy | `themes.inkless.run { reverse_proxy 127.0.0.1:8098 }` |
| DNS | Cloudflare A `themes.inkless.run` → `103.73.220.161` (proxied) |
| Admin | `/admin` — default seed `admin` / `admin123` (**change immediately**) |
| Active theme | `editorial-firm` |
| NoPanel env | `impress` / `theme-demo` (created; docker method incomplete — live deploy is container above) |

## Ops commands

```bash
# Status
npc server exec command gomami -- 'docker ps --filter name=inkless-theme-demo; curl -sS http://127.0.0.1:8098/health'

# Logs
npc server exec command gomami -- 'docker logs --tail 100 inkless-theme-demo'

# Rebuild from branch (on gomami)
# 1) git pull in /tmp/inkless-theme-src
# 2) docker build -t inkless-theme-demo:latest .
# 3) docker rm -f inkless-theme-demo && docker run ... (same flags as first deploy)

# Switch demo theme (admin UI or API)
# PUT /admin/themes/{id}/activate with admin JWT
```

## Notes

- Blank-site seed does **not** insert `installed_themes` rows; first activate requires themes to exist (seeded once for this demo).
- Editorial seed sections apply only when unified page sections are empty.
- Do not point this data dir at personal or product DBs.

## Branch note (2026-07-22 close-out)

Feature work lives on `feat/editorial-firm-theme` (theme package, extract pin, demo docs, Dockerfile packages/ fix). Merge to `main` when ready via PR (was opened as editorial-firm stack). Demo site does **not** require main merge to stay up — image built from that branch SHA on gomami.

KB snapshot: omni page `ops/inkless/self-hosted-sites-and-themes-2026-07`.

## Credentials (npc vault)

| Item | Value |
|------|--------|
| Vault name | `inkless-themes-demo-admin` |
| Vault id | `fbfb4b27-93cf-486d-ad84-bf329af3f115` |
| Kind | `login` |
| Username (public) | `admin` |
| URL (public) | `https://themes.inkless.run/admin` |

```bash
npc vault get fbfb4b27-93cf-486d-ad84-bf329af3f115 -o json   # metadata only
# Reveal is TTY/JWT only — not for agents:
# npc vault reveal fbfb4b27-93cf-486d-ad84-bf329af3f115
```

Default seed password was rotated; do not use `admin123`.

## NoPanel deploy (package path)

Environment: **`impress` / `theme-demo-compose`** (compose_stack, server **gomami**, workDir `/opt/inkless-theme-demo`).

```bash
# 1) Build short-lived source package from GitHub (control plane has repo access)
npc env package build impress theme-demo-compose --ref main -o json

# 2) On gomami, run returned deployCommand (or upload script and bash it)
# Requires /opt/inkless-theme-demo/.env with JWT_SECRET / JWT_REFRESH_SECRET
# Compose file: current/docker-compose.npc.yml (ports 127.0.0.1:8098:8088 only)

npc env package build impress theme-demo-compose --ref main -o json \
  | jq -r '.deployCommand' > /tmp/theme-demo-deploy.sh
npc server file upload gomami /tmp/theme-demo-deploy.sh /tmp/theme-demo-deploy.sh
npc server exec command gomami --async --wait --timeout 900 -- 'bash /tmp/theme-demo-deploy.sh'

# Fallback if compose build context fails: keep prebuilt image
# docker compose -f current/docker-compose.npc.yml --env-file .env -p inkless-theme-demo up -d --no-build
```

Legacy env name `theme-demo` (docker method) remains but is superseded by `theme-demo-compose`.

`npc deploy impress theme-demo-compose` may fail on this host when the worker cannot complete compose build in one shot; prefer package build + deployCommand above.
