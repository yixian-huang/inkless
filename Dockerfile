# Multi-stage build for inkless: frontend SPA + Go backend served from one binary.
# Build context: repo root.

# ────────────────────────────────────────────────────────────────────────────
# Stage 1: build frontend (Vite → frontend/out)
# ────────────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS frontend-builder

# pnpm 9 matches lockfileVersion 9.0 in pnpm-lock.yaml; avoids surprise
# upgrades (pnpm@latest is currently 11 and needs Node 22+ + can rewrite the lock).
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate

WORKDIR /src

# Copy workspace manifests first for better Docker layer caching.
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY frontend/package.json ./frontend/package.json
# Workspace theme packages (e.g. corporate-classic) must exist for pnpm install.
COPY packages ./packages

RUN pnpm install --frozen-lockfile

# Copy the rest of the frontend sources and build.
COPY frontend ./frontend
RUN pnpm -C frontend build

# ────────────────────────────────────────────────────────────────────────────
# Stage 2: build Go backend (CGO enabled for SQLite driver compatibility)
# ────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/cmd ./cmd
COPY backend/internal ./internal
COPY backend/pkg ./pkg
COPY backend/docs ./docs

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /out/inkless-api ./cmd/server

# ────────────────────────────────────────────────────────────────────────────
# Stage 3: minimal runtime
# ────────────────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk --no-cache add ca-certificates wget tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=backend-builder /out/inkless-api /app/inkless-api
COPY --from=frontend-builder /src/frontend/out /app/frontend/out

# Persistent data dir for uploads (and SQLite if DB_DSN ever points to local file).
RUN mkdir -p /app/data /app/uploads

# Defaults; can be overridden by Quick-Box env injection.
ENV FRONTEND_DIR=/app/frontend/out \
    UPLOAD_DIR=/app/uploads \
    PORT=8088 \
    ENV=production

EXPOSE 8088

CMD ["/app/inkless-api"]
