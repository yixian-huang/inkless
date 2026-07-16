# CI/CD Workflows

This directory contains two GitHub Actions workflows:

1. `quality-gate.yml` - CI quality checks
2. `deploy.yml` - CD deployment automation (SSH or HTTP)

## 1) quality-gate.yml

`quality-gate.yml` is the merge gate. It runs:

- Frontend checks: lint, type-check, unit tests, and Playwright admin navigation E2E
- Backend checks: `go mod verify`, `go mod tidy`, `go vet`, `go test -race`
- Integration smoke: frontend build + backend build
- Summary job: fails if any upstream job fails

### Trigger

- Push: `main`, `master`, `develop`
- Pull request target: `main`, `master`, `develop`

### Branch protection

Require these checks before merge:

- `Frontend Quality Checks`
- `Backend Quality Checks`
- `Integration Smoke Checks`
- `Quality Gate Summary`

## 2) deploy.yml

`deploy.yml` provides automated deployment for frontend + backend after CI passes.

### Trigger

- Automatic: `workflow_run` when `Quality Gate` succeeds on `main`/`master` and
  repository variable `AUTO_DEPLOY_ENABLED` is set to `true`
- Manual: `workflow_dispatch` (choose ref/method/environment/version)

### Deployment methods

- `ssh` method:
  - Uses `scripts/deploy.sh`
  - Copies frontend/backend artifacts via SSH/SCP
  - Activates versions with atomic symlink swap
  - Restarts backend service and checks health
- `http` method:
  - Uses `scripts/deploy-http.sh`
  - Uploads both artifacts to a deployment API endpoint

### Notifications

After deploy, workflow supports:

- Webhook notification (`NOTIFY_WEBHOOK_URL`)
- Email notification (SMTP secrets configured)

### Required secrets (SSH mode)

- `DEPLOY_HOST`
- `DEPLOY_SSH_PRIVATE_KEY`

Optional for SSH mode:

- `DEPLOY_USER`
- `DEPLOY_ROOT`
- `DEPLOY_KNOWN_HOSTS`

### Required secrets (HTTP mode)

- `DEPLOY_HTTP_ENDPOINT`

Optional for HTTP mode:

- `DEPLOY_HTTP_TOKEN`

### Optional notification secrets

- `NOTIFY_WEBHOOK_URL`
- `SMTP_SERVER`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `NOTIFY_EMAIL_TO`
- `NOTIFY_EMAIL_FROM`

### Optional repository variables

- `AUTO_DEPLOY_ENABLED` (`true` enables automatic deploys after the main branch
  quality gate; unset/other values keep automatic deploys disabled)
- `DEPLOY_METHOD` (default deploy method for auto run, `ssh`/`http`)
- `DEPLOY_ENVIRONMENT` (default `production`)
- `BACKEND_SERVICE` (for SSH script override)
- `BACKEND_HEALTH_URL` (for SSH post-deploy health check)
