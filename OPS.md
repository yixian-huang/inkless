# Ops — Quick-Box deployment

Quick-Box project: `bb47ab5c-1e79-4c96-8a9e-2c719d2698e7`  
Primary deploy target: **`hk`** → `82.158.226.66`  
Environment id (`hk`): `53e1049c-72fd-4c2a-9d18-251f70e46415`  
Server id: `4eaa0086-d435-4249-a76f-356fddde5261`

## Site isolation (read first)

**yx.ink ≠ inkless.run.** Incident write-up: [`docs/ops-lessons-yx-ink-vs-inkless-run.md`](docs/ops-lessons-yx-ink-vs-inkless-run.md).

On **gomami** (current dual-process layout):

| Public site | systemd | Port | Tree / DB |
|-------------|---------|------|-----------|
| yx.ink (personal) | `inkless` | 8088 | `/opt/inkless` + `data/inkless.db` |
| inkless.run (product) | `inkless-ops` | 8089 | `/opt/inkless-ops` + `data/inkless.db` |

Caddy must reverse-proxy each hostname to its own port. Sharing code symlinks is OK; sharing data/env is not.

Helpers:

- `scripts/ops-bootstrap-inkless-run.sh` — second process for product site  
- `scripts/ops-product-site-cutover.py` — product DB only (`INKLESS_DB=...`)

## Recommended: `artifact` deploy (build server → hk)

**Do not** build Docker images on hk. Use a separate **build server** to compile artifacts; hk only **activates** versioned tarballs.

| Phase | Where | What |
|-------|--------|------|
| build | QB build server | `scripts/qb-artifact-build.sh` → tarballs + `artifact-manifest.json` |
| transfer | QB | scp/rsync bundle to hk |
| activate | hk VPS | `scripts/qb-artifact-activate.sh` → `/opt/inkless` + `systemctl restart inkless` |

Layout on hk:

```text
/opt/inkless/
├── backend/versions/{version}/, current/, previous/
├── frontend/versions/{version}/, current/
├── data/          # SQLite (default)
└── uploads/
```

### Create / migrate hk environment (QB API)

Template: [`ops/qb-init-hk-artifact.json`](qb-init-hk-artifact.json) — set `buildServerName`, JWT secrets, then:

```bash
curl -X POST -H "X-API-Key: $QB_API_KEY" -H "Content-Type: application/json" \
  -d @ops/qb-init-hk-artifact.json \
  https://ops.zoom.ci/api/v1/onboarding/init-project
```

One-time on hk (also in `preDeployScript`):

```bash
bash ./ops/qb-host-bootstrap.sh
```

### Trigger deploy

```bash
curl -X POST -H "X-API-Key: $QB_API_KEY" -H "Content-Type: application/json" \
  -d '{"gitRef":"main"}' \
  https://ops.zoom.ci/api/v1/deploy-hooks/bb47ab5c-1e79-4c96-8a9e-2c719d2698e7/hk
```

Poll until **`healthCheckPassed: true`** (not `status: success` alone):

```bash
curl -H "X-API-Key: $QB_API_KEY" https://ops.zoom.ci/api/v1/deployments/<deploymentId>
curl -H "X-API-Key: $QB_API_KEY" https://ops.zoom.ci/api/v1/deployments/<deploymentId>/logs
curl -sf http://82.158.226.66:8088/health
```

First boot: open `http://82.158.226.66:8088/setup` (`SEED_MODE=blank`, `SETUP_BOOTSTRAP=true`).

### Repository scripts (Quick-Box contract)

| Script | Role |
|--------|------|
| `scripts/qb-artifact-build.sh` | Build server: `build-*.sh` → staging bundle |
| `scripts/qb-artifact-activate.sh` | hk: verify manifest, extract, systemd, health |
| `scripts/qb-artifact-rollback.sh` | hk: symlink rollback |
| `scripts/qb-artifact-manifest.sh` | Emit `artifact-manifest.json` |
| `ops/artifact-manifest.json` | Static schema reference |
| `ops/systemd/inkless.service` | systemd unit template |

Spec: [`docs/quick-box-artifact-deploy-method.md`](../docs/quick-box-artifact-deploy-method.md)

### Env vars (Quick-Box environment, not in repo)

| Key | Example |
|-----|---------|
| `PORT` | `8088` |
| `ENV` | `production` |
| `SEED_MODE` | `blank` |
| `SETUP_BOOTSTRAP` | `true` |
| `FRONTEND_DIR` | `/opt/inkless/frontend/current` |
| `UPLOAD_DIR` | `/opt/inkless/uploads` |
| `DB_DSN` | `file:/opt/inkless/data/inkless.db?...` |
| `JWT_SECRET` | secret |
| `JWT_REFRESH_SECRET` | secret |

Optional build env: `QB_SKIP_BACKEND_TESTS=true` (faster CI build, skips `build-backend.sh` test gate).

---

## Legacy: Docker on target host

| Environment | Server | Notes |
|-------------|--------|-------|
| `production` | `172.81.57.29` | Legacy |
| `hk` (old) | `82.158.226.66` | Same-machine `docker build` — slow / OOM prone |

Docker path: root `Dockerfile`, `scripts/qb-docker-deploy.sh`.  
Redundant trial envs (`vip`, `vip2`, `vip3`) can be deleted in Quick-Box UI.

---

## AI handoff

```bash
curl -H "X-API-Key: $QB_API_KEY" \
  "https://ops.zoom.ci/api/v1/projects/bb47ab5c-1e79-4c96-8a9e-2c719d2698e7/ai-handoff?format=json"
```
