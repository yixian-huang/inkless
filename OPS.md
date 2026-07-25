# Ops — public summary

This file is the **public** ops entrypoint. Host inventory (IPs, project IDs,
deploy hooks) is **not** committed: resolve it from your control plane
(NoPanel / Quick-Box) or private operator notes.

## For self-hosters

| Need | Doc |
|------|-----|
| Docker Compose (Postgres / SQLite) | [`docs/docker-setup.md`](docs/docker-setup.md) |
| Versioned build + deploy scripts | [`docs/deployment.md`](docs/deployment.md) |
| Artifact activate layout | [`docs/quick-box-artifact-deploy-method.md`](docs/quick-box-artifact-deploy-method.md) |
| Getting started | [`docs-site/guide/getting-started.md`](docs-site/guide/getting-started.md) |

### Single-site instance boundary

One Inkless process serves **one** logical site (`BASE_URL` is canonical).  
Two public brands or two customers on one machine need **two processes** (separate
port, data dir, uploads, JWT secrets, systemd unit). Reverse-proxy hostnames alone
do not isolate data.

See the operator lesson: [`docs/internal/ops-lessons-site-isolation.md`](docs/internal/ops-lessons-site-isolation.md).

### Recommended production shape

1. **Build** on a dedicated builder: `scripts/qb-artifact-build.sh` (or CI Release).
2. **Activate** on the app host: `scripts/qb-artifact-activate.sh` under e.g. `/opt/inkless`.
3. **Do not** compile large Docker builds on small production VPS hosts.

```text
/opt/inkless/
├── backend/versions/{version}/, current/, previous/
├── frontend/versions/{version}/, current/
├── data/          # SQLite (or point DB_DSN at Postgres)
└── uploads/
```

### Environment variables (examples only)

| Key | Example |
|-----|---------|
| `PORT` | `8088` |
| `ENV` | `production` |
| `SEED_MODE` | `blank` |
| `SETUP_BOOTSTRAP` | `true` (first boot only) |
| `FRONTEND_DIR` | `/opt/inkless/frontend/current` |
| `UPLOAD_DIR` | `/opt/inkless/uploads` |
| `DB_DSN` | `file:/opt/inkless/data/inkless.db?cache=shared&mode=rwc` |
| `JWT_SECRET` / `JWT_REFRESH_SECRET` | long random secrets — never commit |
| `BASE_URL` | `https://your-domain.example` |
| `CORS_ALLOWED_ORIGINS` | same origin(s) as the admin SPA |

## For repository maintainers

- Operator-specific topology and historical runbooks: [`docs/internal/`](docs/internal/).
- CI quality gate: [`.github/workflows/quality-gate.yml`](.github/workflows/quality-gate.yml).
- Optional self-hosted CD (requires secrets): [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml).
- Public release artifacts (tags): [`.github/workflows/release.yml`](.github/workflows/release.yml).
- GitHub repo listing (description, topics): [`.github/REPO_SETUP.md`](.github/REPO_SETUP.md).

When using NoPanel from this monorepo’s agent workflow:

```bash
# Example — project/env names are yours, not committed inventory
npc deploy <project> <env> --ref main --wait
```
