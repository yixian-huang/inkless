# Production Build and Deployment Guide

This document provides instructions for building versioned artifacts and deploying the 印迹官网 (Blotting Consultancy) application to production environments.

## Overview

The production deployment workflow consists of:

1. **Build Phase**: Create versioned artifacts for frontend and backend
2. **Deploy Phase**: Upload and activate artifacts on target servers
3. **Rollback Phase**: Revert to previous versions if issues arise

All scripts support environment-based configuration and maintain version history for safe rollback operations.

### Quick-Box artifact deploy (`hk`)

For production on Quick-Box environment **`hk`** (`82.158.226.66`), prefer **`deployMethod: artifact`**: a dedicated **build server** runs `scripts/qb-artifact-build.sh`; the VPS only runs `scripts/qb-artifact-activate.sh`. See [`OPS.md`](../OPS.md) and [`docs/quick-box-artifact-deploy-method.md`](quick-box-artifact-deploy-method.md).

### Single-site instance boundary

One Impress instance serves one logical site. `BASE_URL` is its sole canonical origin;
additional domains are aliases and should normally issue a 301 redirect to that origin at
Nginx, Caddy, or the load balancer. `CORS_ALLOWED_ORIGINS` contains the complete origins
allowed to call the admin API.

Independent sites require independent instances. On a shared host, each instance must have
its own release root, systemd unit, port, database, upload directory, plugin package/data
directories, and JWT secrets. For example:

```bash
# Site A
QB_RELEASE_ROOT=/opt/impress-a QB_SYSTEMD_UNIT=impress-a PORT=8088 \
DB_DSN='file:/opt/impress-a/data/impress.db?cache=shared&mode=rwc' \
UPLOAD_DIR=/opt/impress-a/uploads PLUGIN_DIR=/opt/impress-a/plugins \
BACKUP_DIR=/opt/impress-a/backups PLUGIN_DATA_DIR=/opt/impress-a/data/plugins \
BASE_URL=https://a.example.com \
CORS_ALLOWED_ORIGINS=https://a.example.com ./scripts/qb-artifact-activate.sh

# Site B uses /opt/impress-b, impress-b, port 8089, its own directories and secrets.
```

Do not share a business database or local backup/media/plugin directories between independent
sites. See [Quick-Box artifact deployment](quick-box-artifact-deploy-method.md#76-同一主机运行两个独立实例)
for complete A/B values, systemd guidance, Nginx/Caddy alias examples, and the repeatable
two-instance smoke test.

## Prerequisites

### Local Build Environment

- Node.js 20+ with pnpm 8+
- Go 1.21+
- Git (for version tagging)
- SSH access to deployment servers

### Target Server Environment

- SSH daemon with key-based authentication
- Systemd for backend service management
- Nginx or similar web server for frontend (optional)
- User with sudo privileges for service control

## Directory Structure

On the deployment server, the following structure is maintained:

```
/opt/blotting/
├── frontend/
│   ├── versions/
│   │   ├── v1.0.0/          # Extracted frontend builds
│   │   ├── v1.0.1/
│   │   └── v1.1.0/
│   ├── current -> versions/v1.1.0    # Active version symlink
│   └── previous -> versions/v1.0.1   # Backup symlink
└── backend/
    ├── versions/
    │   ├── v1.0.0/          # Extracted backend binaries
    │   ├── v1.0.1/
    │   └── v1.1.0/
    ├── current -> versions/v1.1.0    # Active version symlink
    └── previous -> versions/v1.0.1   # Backup symlink
```

## Building Artifacts

### Frontend Build

Build the React SPA and create a versioned tarball artifact:

```bash
# Build with auto-detected version (from git tags)
./scripts/build-frontend.sh

# Build with explicit version
VERSION=v1.2.0 ./scripts/build-frontend.sh
```

**Build Process:**

1. Cleans previous build output
2. Installs dependencies with `pnpm install --frozen-lockfile`
3. Runs `pnpm type-check` for TypeScript validation
4. Runs `pnpm lint` for code quality checks
5. Builds production bundle with `pnpm build`
6. Creates `build-info.json` with version metadata
7. Packages `frontend/out/` directory into `artifacts/frontend-{version}.tar.gz`
8. Generates SHA256 checksum
9. Creates `frontend-latest.tar.gz` symlink

**Output:**

- `artifacts/frontend-v1.2.0.tar.gz` (versioned artifact)
- `artifacts/frontend-v1.2.0.tar.gz.sha256` (checksum)
- `artifacts/frontend-latest.tar.gz` (symlink to latest)

### Backend Build

Build the Go API server binary and create a versioned tarball artifact:

```bash
# Build with auto-detected version (from git tags)
./scripts/build-backend.sh

# Build with explicit version
VERSION=v1.2.0 ./scripts/build-backend.sh
```

**Build Process:**

1. Verifies Go module dependencies with `go mod verify` and `go mod tidy`
2. Runs full test suite with race detection: `go test -v -race -coverprofile=coverage.out ./...`
3. Runs static analysis with `go vet ./...`
4. Builds optimized binary with ldflags injecting version metadata
5. Creates `build-info.json` with version metadata
6. Packages binary and metadata into `artifacts/backend-{version}.tar.gz`
7. Generates SHA256 checksum
8. Creates `backend-latest.tar.gz` symlink

**Output:**

- `artifacts/backend-v1.2.0.tar.gz` (versioned artifact)
- `artifacts/backend-v1.2.0.tar.gz.sha256` (checksum)
- `artifacts/blotting-api-v1.2.0` (standalone binary)
- `artifacts/backend-latest.tar.gz` (symlink to latest)

### Build Artifacts

Both build scripts generate `build-info.json` files containing:

```json
{
  "version": "v1.2.0",
  "buildTime": "2026-02-13T12:34:56Z",
  "gitCommit": "abc123...",
  "gitBranch": "main",
  "nodeVersion": "v20.11.0",     // frontend only
  "pnpmVersion": "8.15.0",       // frontend only
  "goVersion": "go1.21.6"        // backend only
}
```

## Deploying to Production

### Environment Configuration

Set required environment variables before deployment:

```bash
export DEPLOY_HOST="production.example.com"
export DEPLOY_USER="deploy"
export VERSION="v1.2.0"
export ENVIRONMENT="production"

# Optional overrides
export DEPLOY_ROOT="/opt/blotting"
export BACKEND_SERVICE="blotting-api"
```

### Deployment Script

Deploy both frontend and backend components:

```bash
DEPLOY_HOST=prod.example.com VERSION=v1.2.0 ./scripts/deploy.sh
```

**Deployment Process:**

1. **Preflight Checks:**
   - Verifies required commands (`ssh`, `scp`, `tar`)
   - Confirms artifact files exist locally
   - Validates artifact checksums

2. **Frontend Deployment:**
   - Uploads `frontend-{version}.tar.gz` to server
   - Extracts to `${FRONTEND_PATH}/versions/{version}/`
   - Backs up current version to `previous` symlink
   - Atomically swaps `current` symlink to new version
   - Reloads nginx if active

3. **Backend Deployment:**
   - Uploads `backend-{version}.tar.gz` to server
   - Extracts to `${BACKEND_PATH}/versions/{version}/`
   - Backs up current version to `previous` symlink
   - Stops backend systemd service
   - Atomically swaps `current` symlink to new version
   - Starts backend systemd service
   - Verifies service health

4. **Verification:**
   - Displays `build-info.json` for both components
   - Shows backend systemd service status

### Deployment Example

```bash
# Build artifacts locally
./scripts/build-frontend.sh
./scripts/build-backend.sh

# Deploy to staging environment
ENVIRONMENT=staging \
DEPLOY_HOST=staging.example.com \
VERSION=$(git describe --tags --always) \
./scripts/deploy.sh

# Deploy to production environment
ENVIRONMENT=production \
DEPLOY_HOST=prod.example.com \
VERSION=$(git describe --tags --always) \
./scripts/deploy.sh
```

## Rolling Back Deployments

### Rollback to Previous Version

Rollback uses the `previous` symlink created during deployment:

```bash
# Rollback frontend only
DEPLOY_HOST=prod.example.com COMPONENT=frontend ./scripts/rollback.sh

# Rollback backend only
DEPLOY_HOST=prod.example.com COMPONENT=backend ./scripts/rollback.sh

# Rollback both components
DEPLOY_HOST=prod.example.com COMPONENT=all ./scripts/rollback.sh
```

### Rollback to Specific Version

Rollback to any available version in the versions directory:

```bash
# Rollback frontend to v1.1.0
DEPLOY_HOST=prod.example.com \
COMPONENT=frontend \
TARGET_VERSION=v1.1.0 \
./scripts/rollback.sh

# Rollback backend to v1.1.0
DEPLOY_HOST=prod.example.com \
COMPONENT=backend \
TARGET_VERSION=v1.1.0 \
./scripts/rollback.sh
```

### List Available Versions

View all deployed versions on the server:

```bash
DEPLOY_HOST=prod.example.com COMPONENT=list ./scripts/rollback.sh
```

**Output Example:**

```
Frontend versions:
  Current: /opt/blotting/frontend/versions/v1.2.0
  Previous: /opt/blotting/frontend/versions/v1.1.0
  Available:
    v1.2.0
    v1.1.0
    v1.0.1
    v1.0.0

Backend versions:
  Current: /opt/blotting/backend/versions/v1.2.0
  Previous: /opt/blotting/backend/versions/v1.1.0
  Available:
    v1.2.0
    v1.1.0
    v1.0.1
    v1.0.0
```

### Rollback Safety Features

- **Atomic Symlink Swap**: Uses `ln -snf` + `mv -Tf` to ensure atomic updates
- **Backup Tracking**: `previous` symlink always points to last known good version
- **Service Health Checks**: Backend rollback verifies service starts successfully
- **Rollback Backup**: Creates `rollback_backup` symlink before rollback for recovery
- **Version Validation**: Ensures target version directory exists before rollback

### Emergency Rollback Procedure

If automated rollback fails:

1. SSH into the server:
   ```bash
   ssh deploy@prod.example.com
   ```

2. Manually restore symlinks:
   ```bash
   # Frontend
   cd /opt/blotting/frontend
   ln -snf $(readlink previous) current_tmp
   mv -Tf current_tmp current
   sudo systemctl reload nginx

   # Backend
   cd /opt/blotting/backend
   sudo systemctl stop blotting-api
   ln -snf $(readlink previous) current_tmp
   mv -Tf current_tmp current
   sudo systemctl start blotting-api
   ```

3. Verify services:
   ```bash
   systemctl status blotting-api
   curl http://localhost:8088/health
   ```

## Server Setup Requirements

### Backend Systemd Service

Create `/etc/systemd/system/blotting-api.service`:

```ini
[Unit]
Description=Blotting Consultancy API Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/blotting/backend/current
ExecStart=/opt/blotting/backend/current/blotting-api-latest
EnvironmentFile=/opt/blotting/backend/.env
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/blotting/backend/data

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable blotting-api
sudo systemctl start blotting-api
```

### Frontend Nginx Configuration

Create `/etc/nginx/sites-available/blotting-frontend`:

```nginx
server {
    listen 80;
    server_name blotting.example.com;

    root /opt/blotting/frontend/current;
    index index.html;

    # SPA routing - serve index.html for all routes
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
}
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/blotting-frontend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Environment File

Create `/opt/blotting/backend/.env` with production configuration:

```env
ENV=production
PORT=8088
# Option A: SQLite (lightweight/single-host)
DB_DSN=file:/opt/blotting/backend/data/blotting.db?cache=shared&mode=rwc

# Option B: PostgreSQL
# DB_DSN=postgresql://blotting:password@localhost:5432/blotting?sslmode=require
JWT_SECRET=<production-secret>
JWT_REFRESH_SECRET=<production-refresh-secret>
```

**Security Note:** Never commit `.env` files to version control. Use secure secret management systems in production.

## CI/CD Automation

The repository includes two workflows:

- `quality-gate.yml`: lint/type-check/test/build checks
- `deploy.yml`: deploy after quality gate passes on `main`/`master` (or manual dispatch)

Deployment supports:

- SSH transport (`scripts/deploy.sh`)
- HTTP API transport (`scripts/deploy-http.sh`)
- Notification via webhook and optional SMTP email

### Required GitHub Secrets

- SSH mode: `DEPLOY_HOST`, `DEPLOY_SSH_PRIVATE_KEY`
- HTTP mode: `DEPLOY_HTTP_ENDPOINT`

Optional:

- `DEPLOY_USER`, `DEPLOY_ROOT`, `DEPLOY_HTTP_TOKEN`, `DEPLOY_KNOWN_HOSTS`
- Notification: `NOTIFY_WEBHOOK_URL`, `SMTP_*`, `NOTIFY_EMAIL_TO`, `NOTIFY_EMAIL_FROM`

## Troubleshooting

### Build Failures

**Frontend type-check fails:**
- Run `pnpm type-check` locally to see TypeScript errors
- Fix type errors before creating production builds

**Backend tests fail:**
- Run `go test -v ./...` locally to identify failures
- Ensure all tests pass before deploying

**Artifact checksum mismatch:**
- Re-run build scripts to regenerate artifacts
- Verify artifact files are not corrupted during transfer

### Deployment Failures

**SSH connection refused:**
- Verify `DEPLOY_HOST` is correct and reachable
- Check SSH key authentication is configured
- Test manual SSH: `ssh ${DEPLOY_USER}@${DEPLOY_HOST}`

**Backend service fails to start:**
- Check systemd logs: `journalctl -u blotting-api -n 50`
- Verify `.env` file exists and contains correct configuration
- Check database connectivity from server
- Verify binary is executable and compatible with server architecture

**Frontend not serving:**
- Check nginx configuration: `sudo nginx -t`
- Verify symlink: `ls -la /opt/blotting/frontend/current`
- Check nginx logs: `tail -f /var/log/nginx/error.log`

### Rollback Issues

**"No previous version found":**
- Use `COMPONENT=list` to see available versions
- Specify explicit version with `TARGET_VERSION=v1.x.x`

**Backend service won't start after rollback:**
- Check if rolled-back version is compatible with current database schema
- May need to rollback database migrations separately
- Review systemd logs for startup errors

## Version Management Best Practices

### Version Numbering

Use semantic versioning (SemVer) for releases:

- `v1.0.0` - Major release (breaking changes)
- `v1.1.0` - Minor release (new features, backward compatible)
- `v1.1.1` - Patch release (bug fixes)

Create git tags for releases:

```bash
git tag -a v1.2.0 -m "Release version 1.2.0"
git push origin v1.2.0
```

### Artifact Retention

Maintain version history on deployment servers:

```bash
# Keep last 5 versions, remove older ones
ssh deploy@prod.example.com '
  cd /opt/blotting/frontend/versions
  ls -t | tail -n +6 | xargs rm -rf

  cd /opt/blotting/backend/versions
  ls -t | tail -n +6 | xargs rm -rf
'
```

### Deployment Log

Maintain a deployment log for audit trail:

```bash
echo "$(date -u +"%Y-%m-%d %H:%M:%S UTC") - Deployed ${VERSION} to ${ENVIRONMENT}" \
  >> deployments.log
```

## Security Considerations

1. **Artifact Integrity**: Always verify SHA256 checksums before deployment
2. **SSH Keys**: Use dedicated deploy keys with restricted permissions
3. **Environment Secrets**: Never commit `.env` files or secrets to git
4. **Service Isolation**: Run backend service as non-root user with restricted file system access
5. **Network Security**: Use HTTPS/TLS for frontend and enforce secure database connections
6. **Access Control**: Limit sudo permissions for deploy user to only required commands

## Summary

This build and deployment system provides:

- ✅ **Versioned Artifacts**: All builds include version metadata and checksums
- ✅ **Environment Configuration**: Support for multiple deployment targets (staging, production)
- ✅ **Atomic Deployments**: Symlink-based activation ensures zero-downtime updates
- ✅ **Safe Rollback**: Instant rollback to previous versions with service health checks
- ✅ **Audit Trail**: Build metadata and deployment logs for compliance
- ✅ **Automation Ready**: CI/CD integration support with exit codes and logging

For additional help or questions, refer to `docs/docker-setup.md` for local development environment setup.
