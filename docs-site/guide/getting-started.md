# Getting Started

## Prerequisites

- **Go** 1.24+
- **Node.js** 20+ with **pnpm** 9+
- **Git**

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/your-org/impress.git
cd impress
```

### 2. Install dependencies

```bash
pnpm install
```

### 3. Start the development server

```bash
make dev
```

This starts:
- Backend API at `http://localhost:8088`
- Frontend dev server at `http://localhost:3000`

### 4. Access the admin panel

Open `http://localhost:3000/admin` and log in with the default credentials:
- Username: `admin`
- Password: `admin123`

## Alternative: Docker

```bash
# PostgreSQL variant
docker compose up

# SQLite variant
docker compose -f docker-compose.sqlite.yml up
```

## Project Structure

```
impress/
├── backend/           # Go/Gin/GORM backend
│   ├── cmd/server/    # Main server binary
│   ├── internal/      # Application packages
│   │   ├── handler/   # HTTP handlers
│   │   ├── service/   # Business logic
│   │   ├── repository/# Data access (GORM)
│   │   ├── model/     # Database models
│   │   └── middleware/ # Auth, rate limiting
│   └── pkg/           # Shared packages
├── frontend/          # React/Vite/Tailwind frontend
│   └── src/
│       ├── api/       # Typed API client
│       ├── pages/     # Route pages
│       ├── theme/     # CMS-driven theming
│       └── i18n/      # Translations (zh/en)
├── docs/              # Architecture documentation
├── docs-site/         # This documentation site (VitePress)
└── Makefile           # Build automation
```

## Core Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start backend + frontend dev servers |
| `make stop` | Stop dev servers |
| `make build-backend` | Compile Go binary |
| `make check` | Run lint + type-check |
| `pnpm dev` | Frontend dev server only |
| `pnpm build` | Production frontend build |
| `pnpm lint` | ESLint |
| `pnpm type-check` | TypeScript type check |
| `pnpm test` | Run frontend tests |

## Next Steps

- [Architecture Overview](/guide/architecture) -- understand the system design
- [Extension Points](/guide/extension-points) -- learn about Provider interfaces
- [Your First Plugin](/guide/first-plugin) -- build a simple plugin
