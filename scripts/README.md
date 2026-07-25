# Build and Deployment Scripts

This directory contains production build and deployment automation scripts for the Inkless CMS (Inkless CMS) application.

## Scripts Overview

| Script | Purpose | Usage |
|--------|---------|-------|
| `build-frontend.sh` | Build versioned frontend artifact | `./scripts/build-frontend.sh` |
| `build-backend.sh` | Build versioned backend binary | `./scripts/build-backend.sh` |
| `deploy.sh` | Deploy artifacts to production | `DEPLOY_HOST=prod.example.com VERSION=v1.0.0 ./scripts/deploy.sh` |
| `deploy-http.sh` | Deploy artifacts via HTTP API | `DEPLOY_HTTP_ENDPOINT=https://deploy.example.com/api/releases VERSION=v1.0.0 ./scripts/deploy-http.sh` |
| `rollback.sh` | Rollback to previous version | `DEPLOY_HOST=prod.example.com COMPONENT=all ./scripts/rollback.sh` |
| `qb-docker-deploy.sh` | Quick-Box docker script deploy (legacy) | QB `deployMethod=script` on target host |
| `qb-artifact-build.sh` | Quick-Box artifact build (build server) | `artifactDeployConfig.buildCommand` |
| `qb-artifact-activate.sh` | Quick-Box artifact activate (deploy server) | `artifactDeployConfig.activateCommand` |
| `qb-artifact-rollback.sh` | Quick-Box artifact rollback | `artifactDeployConfig.rollbackCommand` |
| `check-external-identity.sh` | Read-only GitHub, DNS, TLS, npm, and Go identity check | `./scripts/check-external-identity.sh --expect-cutover` |
| `check-systemd-unit.sh` | Validate the production systemd unit and canonical Inkless paths | `bash scripts/check-systemd-unit.sh` |
| `migrate-db.sh` | Migrate SQLite to PostgreSQL | `TARGET_DB_DSN="..." ./scripts/migrate-db.sh` |
| `long-agent.mjs` | Long-running autonomous agent | `pnpm agent:run` |
| `agent-usage.mjs` | Agent cost and usage tracking | `pnpm agent:usage` |

`deploy-run.sh` is a legacy pCloud transport and requires `PCLOUD_USER` and
`PCLOUD_PASS` at runtime. Never commit those credentials; the Quick-Box artifact
path remains the recommended production deployment flow.

## Quick Start

### Building Artifacts

```bash
# Build frontend (uses git tag for version)
pnpm build:frontend

# Build backend (uses git tag for version)
pnpm build:backend

# Build with explicit version
VERSION=v1.2.3 pnpm build:frontend
VERSION=v1.2.3 pnpm build:backend
```

### Deploying to Production

```bash
# Set required environment variables
export DEPLOY_HOST="production.example.com"
export DEPLOY_USER="deploy"
export VERSION="v1.2.3"

# Deploy both components
pnpm deploy

# Or use the script directly
./scripts/deploy.sh
```

### Deploying via HTTP API

```bash
export DEPLOY_HTTP_ENDPOINT="https://deploy.example.com/api/releases"
export DEPLOY_HTTP_TOKEN="replace-with-api-token" # optional if endpoint requires auth
export VERSION="v1.2.3"

./scripts/deploy-http.sh
```

### Quick-Box artifact deploy (recommended for `hk`)

Build server compiles; deploy server (`<APP_HOST>`) only activates. See `OPS.md` and `docs/quick-box-artifact-deploy-method.md`.

```bash
# Local dry-run (build only)
export VERSION="$(git describe --tags --always --dirty)"
export QB_ARTIFACT_STAGING="/tmp/inkless-artifact-${VERSION}"
./scripts/qb-artifact-build.sh

# Activate dry-run (as root on deploy host)
export QB_ARTIFACT_INCOMING="/tmp/inkless-artifact-${VERSION}"
export QB_VERSION="${VERSION}"
export QB_RELEASE_ROOT="/opt/inkless"
export PORT=8088
sudo -E ./scripts/qb-artifact-activate.sh
```

QB environment template: `ops/qb-init-hk-artifact.json`

### Rolling Back

```bash
# Rollback to previous version
DEPLOY_HOST=prod.example.com COMPONENT=all pnpm rollback

# Rollback to specific version
DEPLOY_HOST=prod.example.com COMPONENT=backend TARGET_VERSION=v1.2.0 pnpm rollback

# List available versions
DEPLOY_HOST=prod.example.com COMPONENT=list pnpm rollback
```

### Database Migration

```bash
# Migrate from SQLite to PostgreSQL
export TARGET_DB_DSN="host=localhost user=inkless password=inkless_dev_password dbname=inkless port=5432 sslmode=disable"
pnpm migrate:db

# Custom source database
export SOURCE_DB_DSN="backups/20260212_120000/inkless.db.backup"
export TARGET_DB_DSN="postgres://user:pass@host:5432/dbname"
pnpm migrate:db
```

## Build Scripts

### build-frontend.sh

Builds the React frontend with production optimizations.

**Process:**
1. Cleans previous build (`out/`)
2. Installs dependencies with `pnpm install --frozen-lockfile`
3. Runs `pnpm type-check` for TypeScript validation
4. Runs `pnpm lint` for code quality
5. Builds production bundle with `pnpm build`
6. Creates `build-info.json` with metadata
7. Packages into `artifacts/frontend-{version}.tar.gz`
8. Generates SHA256 checksum

**Environment Variables:**
- `VERSION` (optional): Override version (default: git describe)

**Output:**
- `artifacts/frontend-{version}.tar.gz`
- `artifacts/frontend-{version}.tar.gz.sha256`
- `artifacts/frontend-latest.tar.gz` (symlink)

### build-backend.sh

Builds the Go backend binary with embedded version metadata.

**Process:**
1. Verifies dependencies (`go mod verify`, `go mod tidy`)
2. Runs tests with race detection (`go test -v -race -coverprofile=coverage.out ./...`)
3. Runs static analysis (`go vet ./...`)
4. Builds binary with ldflags injecting version information
5. Creates `build-info.json` with metadata
6. Packages into `artifacts/backend-{version}.tar.gz`
7. Generates SHA256 checksum

**Environment Variables:**
- `VERSION` (optional): Override version (default: git describe)

**Output:**
- `artifacts/backend-{version}.tar.gz`
- `artifacts/backend-{version}.tar.gz.sha256`
- `artifacts/inkless-api-{version}` (standalone binary)
- `artifacts/backend-latest.tar.gz` (symlink)

**Version Metadata:**
The backend binary includes embedded version information accessible at runtime:
- `/health` endpoint returns version, buildTime, and gitCommit
- Server logs include version information on startup

## Deployment Scripts

### deploy.sh

Deploys frontend and backend artifacts to a remote server using SSH.

**Required Environment Variables:**
- `DEPLOY_HOST`: Target server hostname or IP
- `VERSION`: Version to deploy (must match built artifacts)

**Optional Environment Variables:**
- `DEPLOY_USER`: SSH user (default: `deploy`)
- `DEPLOY_ROOT`: Base deployment directory (default: `/opt/inkless`)
- `ENVIRONMENT`: Environment name (default: `production`)
- `BACKEND_SERVICE`: Systemd service name (default: `inkless`)
- `FRONTEND_PATH`: Frontend deployment path (default: `${DEPLOY_ROOT}/frontend`)
- `BACKEND_PATH`: Backend deployment path (default: `${DEPLOY_ROOT}/backend`)
- `BACKEND_HEALTH_URL`: HTTP health check URL on remote host (default: `http://127.0.0.1:8088/health`)
- `DEPLOY_AUTO_APPROVE`: Skip interactive confirmation (`true`/`1`/`yes`) for CI/CD

**Deployment Process:**
1. **Preflight Checks**: Validates artifacts exist and checksums match
2. **Frontend Deployment**: Uploads, extracts, and atomically swaps symlink
3. **Backend Deployment**: Uploads, extracts, stops service, swaps symlink, starts service
4. **Verification**: Displays build info and service status

**Safety Features:**
- Atomic symlink swap ensures zero-downtime updates
- Previous version backed up to `previous` symlink for rollback
- Backend service health check after restart
- Automatic `inkless-api-latest` symlink creation in each backend release directory
- Interactive confirmation prompt (can be disabled with `DEPLOY_AUTO_APPROVE=true`)

### deploy-http.sh

Deploys frontend and backend artifacts through an HTTP endpoint.

**Required Environment Variables:**
- `DEPLOY_HTTP_ENDPOINT`: Deployment API endpoint URL
- `VERSION`: Version to deploy (must match built artifacts)

**Optional Environment Variables:**
- `DEPLOY_HTTP_TOKEN`: Bearer token for endpoint authentication
- `ENVIRONMENT`: Environment name (default: `production`)
- `ARTIFACTS_DIR`: Artifact directory (default: `./artifacts`)

**Behavior:**
1. Validates artifact files exist
2. Builds a multipart HTTP request with frontend/backend artifacts
3. Includes checksum files when present
4. Fails if endpoint returns non-2xx HTTP status

### rollback.sh

Rolls back frontend and/or backend to a previous version.

**Required Environment Variables:**
- `DEPLOY_HOST`: Target server hostname or IP
- `COMPONENT`: Component to rollback (`frontend`, `backend`, or `all`)

**Optional Environment Variables:**
- `TARGET_VERSION`: Specific version to rollback to (default: `previous`)
- `DEPLOY_USER`: SSH user (default: `deploy`)
- `DEPLOY_ROOT`: Base deployment directory (default: `/opt/inkless`)
- `BACKEND_SERVICE`: Systemd service name (default: `inkless`)

**Rollback Process:**
1. Validates target version exists on server
2. Backs up current version to `rollback_backup` symlink
3. Atomically swaps symlink to target version
4. For backend: restarts service and verifies health
5. For frontend: reloads nginx if active

**Special Commands:**
```bash
# List all available versions on server
DEPLOY_HOST=prod.example.com COMPONENT=list ./scripts/rollback.sh
```

## Version Management

### Version Detection

By default, both build scripts use `git describe --tags --always --dirty` to determine the version:

```bash
# With git tags
$ git tag -a v1.2.3 -m "Release 1.2.3"
$ ./scripts/build-frontend.sh
# Produces: artifacts/frontend-v1.2.3.tar.gz

# Without git tags (uses commit hash)
$ ./scripts/build-frontend.sh
# Produces: artifacts/frontend-v0.0.0-abc1234.tar.gz

# Dirty working directory (uncommitted changes)
$ ./scripts/build-frontend.sh
# Produces: artifacts/frontend-v1.2.3-dirty.tar.gz
```

### Manual Version Override

```bash
# Build specific version
VERSION=v1.2.3 ./scripts/build-frontend.sh
VERSION=v1.2.3 ./scripts/build-backend.sh

# Deploy specific version
VERSION=v1.2.3 DEPLOY_HOST=prod.example.com ./scripts/deploy.sh
```

### Semantic Versioning

Follow semantic versioning (SemVer) for releases:
- `vMAJOR.MINOR.PATCH` (e.g., `v1.2.3`)
- `MAJOR`: Breaking changes
- `MINOR`: New features, backward compatible
- `PATCH`: Bug fixes

## Directory Structure

After running build scripts, the following structure is created:

```
/Users/yixian.huang/code/Inkless CMS/
├── artifacts/                           # Build artifacts directory
│   ├── frontend-v1.2.3.tar.gz           # Versioned frontend artifact
│   ├── frontend-v1.2.3.tar.gz.sha256    # Frontend checksum
│   ├── frontend-latest.tar.gz → ...     # Symlink to latest frontend
│   ├── backend-v1.2.3.tar.gz            # Versioned backend artifact
│   ├── backend-v1.2.3.tar.gz.sha256     # Backend checksum
│   ├── backend-latest.tar.gz → ...      # Symlink to latest backend
│   ├── inkless-api-v1.2.3              # Standalone backend binary
│   ├── inkless-api-latest → ...        # Symlink to latest binary
│   └── build-info.json                  # Build metadata
├── frontend/out/                        # Frontend build output (temporary)
│   ├── index.html
│   ├── assets/
│   └── build-info.json
└── coverage.out                         # Backend test coverage (temporary)
```

The `artifacts/` directory is excluded from git via `.gitignore`.

## Integration with CI/CD

### GitHub Actions Integration

This repository includes:

- `.github/workflows/quality-gate.yml`: CI quality gate
- `.github/workflows/deploy.yml`: CD deployment workflow

`deploy.yml` supports:

- auto-deploy when quality gate succeeds on `main`/`master`
- manual deploy via `workflow_dispatch`
- `ssh` and `http` deployment transports
- webhook/email notifications after deployment

## Troubleshooting

### Build Failures

**Frontend build fails with type errors:**
```bash
# Check TypeScript errors locally
pnpm type-check
```

**Backend build fails with test errors:**
```bash
# Run tests locally to see failures
go test -v ./...
```

**Checksum generation fails:**
- Ensure `sha256sum` (Linux) or `shasum` (macOS) is installed
- Script will warn but continue if checksum tools are missing

### Deployment Failures

**SSH connection refused:**
```bash
# Test SSH connection manually
ssh ${DEPLOY_USER}@${DEPLOY_HOST}
```

**Backend service fails to start:**
```bash
# Check systemd logs on server
ssh deploy@prod.example.com 'journalctl -u inkless-api -n 50'
```

**Artifact not found:**
- Ensure you ran build scripts before deployment
- Check `artifacts/` directory contains the specified version

### Rollback Issues

**"No previous version found":**
```bash
# List available versions on server
DEPLOY_HOST=prod.example.com COMPONENT=list ./scripts/rollback.sh

# Rollback to specific version
DEPLOY_HOST=prod.example.com COMPONENT=backend TARGET_VERSION=v1.1.0 ./scripts/rollback.sh
```

## Best Practices

1. **Always tag releases in git:**
   ```bash
   git tag -a v1.2.3 -m "Release 1.2.3"
   git push origin v1.2.3
   ```

2. **Test builds locally before deploying:**
   ```bash
   ./scripts/build-frontend.sh
   ./scripts/build-backend.sh
   ```

3. **Verify artifacts before deployment:**
   ```bash
   # Check checksums
   cd artifacts
   sha256sum -c frontend-v1.2.3.tar.gz.sha256
   sha256sum -c backend-v1.2.3.tar.gz.sha256
   ```

4. **Keep version history on servers:**
   - Maintain at least 3-5 previous versions for rollback
   - Periodically clean up very old versions to save disk space

5. **Monitor deployments:**
   - Check application logs after deployment
   - Verify health endpoints return correct version
   - Monitor metrics for anomalies

6. **Document deployments:**
   - Maintain a deployment log
   - Record which versions are deployed to which environments
   - Note any manual interventions or issues

## Additional Documentation

- **Full Deployment Guide**: See `docs/deployment.md` for comprehensive deployment documentation
- **Database Migration Guide**: See `docs/sqlite-to-postgres-migration.md` for SQLite to PostgreSQL migration playbook
- **Docker Setup**: See `docs/docker-setup.md` for local development with Docker Compose
- **API Specification**: See `docs/api-spec.md` for backend API contracts
- **Development Plan**: See `docs/development-plan.md` for project roadmap

## Database Migration Tools

### migrate-db.sh

Migrates data from SQLite (development) to PostgreSQL (production).

**Process:**
1. Creates timestamped backup of source SQLite database
2. Connects to source and target databases
3. Migrates users, content documents, and content versions
4. Validates data integrity and counts
5. Provides detailed migration summary

**Environment Variables:**
- `SOURCE_DB_DSN` (optional): Path to SQLite database (default: `data/inkless.db`)
- `TARGET_DB_DSN` (required): PostgreSQL connection string

**Usage Examples:**

```bash
# Migrate to local PostgreSQL (Docker Compose)
export TARGET_DB_DSN="host=localhost user=inkless password=inkless_dev_password dbname=inkless port=5432 sslmode=disable"
pnpm migrate:db

# Migrate from backup to production PostgreSQL
export SOURCE_DB_DSN="backups/20260212_120000/inkless.db.backup"
export TARGET_DB_DSN="host=prod-db.example.com user=inkless password=SECURE_PASSWORD dbname=inkless port=5432 sslmode=require"
./scripts/migrate-db.sh

# Migration with custom timezone
export TARGET_DB_DSN="host=localhost user=inkless password=pass dbname=inkless port=5432 sslmode=disable TimeZone=Asia/Shanghai"
pnpm migrate:db
```

**Output:**
- Backup directory: `backups/YYYYMMDD_HHMMSS/`
  - `inkless.db.backup` (SQLite binary copy)
  - `inkless_dump.sql` (SQL text dump)
- Migration summary with success/failure counts per table
- Exit code 0 on success, 1 on errors

**Pre-Migration Checklist:**
- [ ] Backup source SQLite database
- [ ] Start PostgreSQL instance (e.g., `docker compose up -d db`)
- [ ] Verify PostgreSQL is accessible (`pg_isready -h localhost -U inkless`)
- [ ] Set `TARGET_DB_DSN` environment variable
- [ ] Review `docs/sqlite-to-postgres-migration.md` for full migration guide

**Post-Migration Validation:**
1. Compare record counts in source and target
2. Verify JSON/JSONB data integrity
3. Test API behavior with PostgreSQL backend
4. Check application health endpoint

**Notes:**
- Refresh tokens are NOT migrated (users must re-login)
- Migration is idempotent for content documents (uses upsert)
- Duplicate user/version errors are logged but don't stop migration
- Full documentation: `docs/sqlite-to-postgres-migration.md`

## Support

For issues or questions:
1. Check `docs/deployment.md` troubleshooting section
2. Review server logs (`journalctl -u inkless-api`)
3. Verify environment configuration (`.env` files)
4. Check GitHub Actions workflow runs for CI/CD issues
