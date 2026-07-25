# Operator runbook (maintainers)

> **Inventory is external.** Do not re-introduce production IPs, project UUIDs,
> or deploy-hook URLs into this file. Resolve them with:
>
> ```bash
> npc server list
> npc server handoff brief <server-ref> --section agentSummary -o json
> npc project list
> npc env list <project>
> ```

## Site isolation (read first)

**Personal site ≠ product site.** Domain names are not process boundaries.

Typical dual-process layout on a shared host:

| Public site role | systemd (example) | Port (example) | Tree / DB (example) |
|------------------|-------------------|----------------|---------------------|
| Personal blog | `inkless` | `8088` | `/opt/inkless` + `data/inkless.db` |
| Product marketing | `inkless-ops` | `8089` | `/opt/inkless-ops` + `data/inkless.db` |

Reverse proxy each hostname to its own port. Sharing **code** symlinks is OK;
sharing **data / env / JWT** is not.

Helpers in this repo:

- `scripts/ops-bootstrap-inkless-run.sh` — second process for a product site
- `scripts/ops-product-site-cutover.py` — product DB only (`INKLESS_DB=…`)

Lesson detail: [`ops-lessons-site-isolation.md`](ops-lessons-site-isolation.md).

## Artifact deploy (build server → app host)

Do **not** build large images on the small production host. Prefer:

| Phase | Where | What |
|-------|--------|------|
| build | Builder | `scripts/qb-artifact-build.sh` → tarballs + manifest |
| transfer | Control plane / scp | Bundle to app host |
| activate | App host | `scripts/qb-artifact-activate.sh` → versioned tree + restart |

Template init JSON (edit locally, never commit secrets):
[`ops/qb-init-hk-artifact.json`](../../ops/qb-init-hk-artifact.json)

Host bootstrap:

```bash
bash ./ops/qb-host-bootstrap.sh
```

Trigger deploy via your control plane’s deploy hook or:

```bash
npc deploy <project> <env> --ref main --wait
```

Health checks must assert **application** readiness (e.g. `healthCheckPassed`),
not only job status.

First boot: open `https://<your-host>/setup` with `SEED_MODE=blank` and
`SETUP_BOOTSTRAP=true` as appropriate.

### Repository scripts (artifact contract)

| Script | Role |
|--------|------|
| `scripts/qb-artifact-build.sh` | Builder: `build-*.sh` → staging bundle |
| `scripts/qb-artifact-activate.sh` | Host: verify manifest, extract, systemd, health |
| `scripts/qb-artifact-rollback.sh` | Host: symlink rollback |
| `scripts/qb-artifact-manifest.sh` | Emit `artifact-manifest.json` |
| `ops/artifact-manifest.json` | Static schema reference |
| `ops/systemd/inkless.service` | systemd unit template |

Spec: [`docs/quick-box-artifact-deploy-method.md`](../quick-box-artifact-deploy-method.md)

### Env vars (environment store — not in git)

| Key | Notes |
|-----|--------|
| `PORT` | e.g. `8088` |
| `ENV` | `production` |
| `SEED_MODE` | `blank` for production |
| `SETUP_BOOTSTRAP` | `true` only for first setup |
| `FRONTEND_DIR` / `UPLOAD_DIR` / `DB_DSN` | Per-instance paths |
| `JWT_SECRET` / `JWT_REFRESH_SECRET` | Rotate via vault / env store |

Optional build env: `QB_SKIP_BACKEND_TESTS=true` (faster builder; skips test gate in
`build-backend.sh`).

## Legacy: Docker on target host

Prefer artifact deploy. Same-machine `docker build` on small VPS hosts is slow
and OOM-prone. Docker path remains: root `Dockerfile`, `scripts/qb-docker-deploy.sh`.
