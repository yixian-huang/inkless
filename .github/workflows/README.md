# CI/CD Workflows

| Workflow | Who | Purpose |
|----------|-----|---------|
| `quality-gate.yml` | Everyone | Merge gate (lint, tests, smoke build) |
| `release.yml` | Community | Tag `v*` → GitHub Release artifacts |
| `deploy.yml` | Maintainers | Optional CD to **your** hosts (needs secrets) |

## 1) quality-gate.yml

Merge gate. Typically runs:

- Frontend: lint, type-check, unit tests, theme smoke
- Backend: `go mod verify` / `tidy`, `go vet`, `go test -race`
- Integration smoke: frontend + backend build
- Summary job fails if any upstream job fails

### Trigger

- Push: `main`, `master`, `develop`
- Pull request targeting those branches

### Branch protection

Require Quality Gate checks before merge (exact job names as shown in Actions).

## 2) release.yml

Builds versioned frontend/backend tarballs and publishes a **GitHub Release** when
you push a tag matching `v*`.

```bash
git tag -a v0.1.0-alpha.2 -m "Inkless v0.1.0-alpha.2"
git push origin v0.1.0-alpha.2
```

Manual dry-run: **Actions → Release → Run workflow** (uploads CI artifacts only;
does not create a tag).

Does **not** SSH into production hosts.

## 3) deploy.yml

Optional maintainer deployment after CI (SSH or HTTP). **Not required for
contributors.** Leave repository variable `AUTO_DEPLOY_ENABLED` unset/false on
a pure upstream fork.

### Trigger

- Automatic: `workflow_run` when Quality Gate succeeds on `main`/`master` and
  `AUTO_DEPLOY_ENABLED=true`
- Manual: `workflow_dispatch`

### Required secrets (SSH mode)

- `DEPLOY_HOST`
- `DEPLOY_SSH_PRIVATE_KEY`

Optional: `DEPLOY_USER`, `DEPLOY_ROOT`, `DEPLOY_KNOWN_HOSTS`, notification secrets.

### Required secrets (HTTP mode)

- `DEPLOY_HTTP_ENDPOINT`
- optional `DEPLOY_HTTP_TOKEN`

See comments in `deploy.yml` for notification and variable knobs.

## Related

- [REPO_SETUP.md](../REPO_SETUP.md) — description, topics, Pages
- [SECURITY.md](../../SECURITY.md)
- [docs/deployment.md](../../docs/deployment.md)
