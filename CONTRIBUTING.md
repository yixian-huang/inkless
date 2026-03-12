# Contributing to Impress CMS

Thank you for your interest in contributing to Impress CMS! This guide will help you get started.

## Development Environment Setup

### Prerequisites

- **Go** 1.24+
- **Node.js** 20+
- **pnpm** 9+
- **Git**
- **Make** (optional but recommended)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/your-org/impress.git
cd impress

# Install frontend dependencies
pnpm install

# Start both backend and frontend in development mode
make dev
```

This starts:
- Backend API at `http://localhost:8088`
- Frontend dev server at `http://localhost:3000`

To stop both servers:

```bash
make stop
```

### Database

By default, the backend uses **SQLite** for local development. No database setup is required. The database file is created automatically at `backend/data/blotting.db`.

For PostgreSQL, use Docker Compose:

```bash
docker compose up
```

## Development Workflow

### Backend (Go)

```bash
cd backend

# Build the server
go build -o server ./cmd/server/

# Run all tests with race detection
go test -v -race ./...

# Static analysis
go vet ./...
```

The backend follows a **layered architecture**:

```
cmd/server/main.go     -> Entry point, dependency wiring
internal/handler/      -> HTTP handlers (request/response)
internal/service/      -> Business logic
internal/repository/   -> Data access (GORM)
internal/model/        -> Database models
internal/middleware/    -> Auth, rate limiting
pkg/                   -> Shared packages
```

Each repository uses the **interface + `_impl.go`** pattern. When adding a new domain:

1. Define the model in `internal/model/`
2. Create a migration in `backend/migrations/`
3. Define the repository interface in `internal/repository/`
4. Implement it in a `_impl.go` file
5. Add business logic in `internal/service/`
6. Add HTTP handlers in `internal/handler/`
7. Register routes using the `RegisterRoutes(public, admin)` method

### Frontend (React/TypeScript)

```bash
# Development server
pnpm dev

# Linting
pnpm lint

# Type checking
pnpm type-check

# Run tests
pnpm test

# Production build
pnpm build
```

Key conventions:
- **Path alias:** `@` maps to `src/`
- **Auto-imports:** React hooks, router helpers, and `useTranslation`/`Trans` are auto-imported -- do not add redundant imports
- **Styling:** Tailwind utility-first classes
- **Components:** Functional components with TypeScript (`.tsx`)

### Database Migrations

Migrations use [goose](https://github.com/pressly/goose) and are stored in `backend/migrations/`.

```bash
# Create a new migration
goose -dir backend/migrations create <migration_name> sql

# Run pending migrations (handled automatically on server start)
cd backend && go run ./cmd/server/ migrate up

# Check migration status
cd backend && go run ./cmd/server/ migrate status
```

## Code Style

### Go

- Format with `go fmt`
- Follow standard Go conventions
- Use the repository interface + `_impl.go` pattern for data access
- New handlers should implement `RegisterRoutes(public, admin *gin.RouterGroup)`

### TypeScript / React

- 2-space indentation, double quotes, semicolons
- Functional components only
- Tailwind utility-first styles (avoid custom CSS)
- `@typescript-eslint/no-explicit-any` is disabled
- All routes are lazy-loaded

### Commit Messages

Use conventional commit format:

```
type(scope): short description

Examples:
feat(backend): add category filtering to article list
fix(frontend): resolve theme switching flicker
docs: update API documentation
refactor(handler): extract validation logic to service layer
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`

## Pull Request Process

### Branch Naming

```
feat/<short-description>
fix/<short-description>
docs/<short-description>
refactor/<short-description>
```

### Before Submitting

1. Run the quality checks:

```bash
# Frontend
pnpm lint && pnpm type-check && pnpm test

# Backend
cd backend && go vet ./... && go test -v -race ./...
```

2. Ensure your branch is up to date with `main`
3. Write a clear PR description using the PR template

### Required Checks

All PRs must pass the CI quality gate:
- ESLint (no errors)
- TypeScript type check (no errors)
- Frontend tests (vitest)
- Go vet (no issues)
- Go tests with race detection
- Integration smoke build

### Review Process

1. Open a PR against `main`
2. Fill out the PR template
3. Wait for CI to pass
4. Request review from a maintainer
5. Address review feedback
6. Squash and merge when approved

## Architecture Documentation

- [Architecture Overview](docs/architecture.md) -- system design and layering
- [Developer Guide](docs/developer-guide.md) -- detailed architecture for developers
- [API Specification](docs/api-spec.md) -- REST API contract
- [Data Model](docs/data-model.md) -- page config and translation rules

## Getting Help

- Open a [GitHub Issue](https://github.com/your-org/impress/issues) for bugs or feature requests
- Check existing issues and discussions before opening a new one
