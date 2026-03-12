# Impress CMS Developer Guide

This document provides a comprehensive architecture overview for developers who want to understand, extend, or contribute to Impress CMS.

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Backend Layers](#backend-layers)
3. [Provider Pattern](#provider-pattern)
4. [EventBus](#eventbus)
5. [Provider Registry](#provider-registry)
6. [Database](#database)
7. [Frontend Architecture](#frontend-architecture)
8. [Adding a New Feature](#adding-a-new-feature)

---

## System Architecture

Impress CMS is a bilingual (Chinese/English) content management system with a clear separation between backend and frontend:

```
                    ┌─────────────────────────────────────┐
                    │         Client (Browser)             │
                    └──────────┬───────────────────────────┘
                               │
                    ┌──────────▼───────────────────────────┐
                    │     Nginx / Reverse Proxy             │
                    │  (static assets + API proxy)          │
                    └──────────┬───────────────────────────┘
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
  ┌────────▼─────────┐ ┌──────▼───────┐  ┌───────▼──────┐
  │  Frontend SPA     │ │  /api/*       │  │  /uploads/*  │
  │  React + Vite     │ │  Go/Gin API   │  │  Static files│
  │  Port 3000 (dev)  │ │  Port 8088    │  │              │
  └──────────────────┘ └──────┬───────┘  └──────────────┘
                               │
                    ┌──────────▼───────────────────────────┐
                    │           Database                     │
                    │  SQLite (dev) / PostgreSQL (prod)      │
                    └──────────────────────────────────────┘
```

### Key Design Principles

- **Layered architecture**: Each backend layer has a single responsibility
- **Interface-driven**: Repository and provider layers use Go interfaces for testability
- **Bilingual by default**: All content supports `zh` and `en` locales
- **Config-driven rendering**: Frontend pages are assembled from CMS-managed section configs
- **Single binary**: The backend compiles to a single Go binary that serves both the API and the SPA

---

## Backend Layers

The backend follows a strict layered architecture. Dependencies flow downward only.

```
┌─────────────────────────────────────────────────────┐
│  cmd/server/main.go                                  │
│  Entry point: wires all dependencies, registers      │
│  routes, starts the HTTP server                      │
├─────────────────────────────────────────────────────┤
│  internal/handler/                                   │
│  HTTP handlers: parse requests, call services,       │
│  return JSON responses. One file per domain.         │
├─────────────────────────────────────────────────────┤
│  internal/service/                                   │
│  Business logic: validation, publishing rules,       │
│  version control, translation state                  │
├─────────────────────────────────────────────────────┤
│  internal/repository/                                │
│  Data access: GORM queries behind interfaces.        │
│  Each domain has interface.go + _impl.go             │
├─────────────────────────────────────────────────────┤
│  internal/model/                                     │
│  GORM models: struct definitions with tags           │
├─────────────────────────────────────────────────────┤
│  internal/middleware/                                │
│  Cross-cutting: JWT auth, rate limiting              │
├─────────────────────────────────────────────────────┤
│  pkg/                                                │
│  Shared packages: apierror, audit, auth, config,     │
│  logger, metrics                                     │
└─────────────────────────────────────────────────────┘
```

### Handler Layer (`internal/handler/`)

Handlers are grouped by domain. Each handler struct holds references to the services it needs.

Domains:
- `auth` -- login, register, token refresh
- `article` -- CRUD for articles
- `content` -- content documents and versions (draft/publish workflow)
- `page` -- CMS page configuration
- `theme` -- theme packages and installed themes
- `media` -- file uploads
- `category`, `tag` -- taxonomy
- `public` -- unauthenticated read endpoints
- `analytics` -- page views
- `backup` -- data export/import
- `auditlog` -- audit trail
- `sitemap` -- sitemap generation
- `form_submission` -- contact form submissions

Each handler implements `RegisterRoutes(public, admin *gin.RouterGroup)` to set up its routes. Routes are registered in `cmd/server/main.go`.

### Service Layer (`internal/service/`)

Services contain business logic that is independent of HTTP concerns:

- **content_service.go** -- draft/publish workflow, version management, translation state
- **validation_service.go** -- schema validation for page configs
- **theme_page_service.go** -- theme-page relationship management

### Repository Layer (`internal/repository/`)

Each domain follows the **interface + implementation** pattern:

```
article_repository.go       # Interface definition
article_repository_impl.go  # GORM implementation
```

The interface defines the contract:

```go
type ArticleRepository interface {
    Create(article *model.Article) error
    FindByID(id uint) (*model.Article, error)
    FindAll(page, pageSize int) ([]model.Article, int64, error)
    Update(article *model.Article) error
    Delete(id uint) error
}
```

The `_impl.go` file provides the GORM implementation. This pattern enables unit testing with mock repositories.

### Model Layer (`internal/model/`)

GORM models with struct tags. Key models include:

| Model | Description |
|-------|-------------|
| `User` | Admin users with hashed passwords |
| `Article` | Blog articles with bilingual content |
| `ContentDocument` | CMS page configs (draft + published) |
| `ContentVersion` | Version history for content documents |
| `Page` | Page metadata and routing |
| `Media` | Uploaded files (images, documents) |
| `Category`, `Tag` | Content taxonomy |
| `FormSubmission` | Contact form entries |
| `InstalledTheme` | Active theme packages |
| `PageView` | Analytics data |
| `AuditEvent` | Audit trail entries |

### Middleware (`internal/middleware/`)

- **auth.go** -- JWT authentication middleware. Validates access tokens and attaches user context.
- **ratelimit.go** -- Token-bucket rate limiting per IP.

### Shared Packages (`pkg/`)

| Package | Purpose |
|---------|---------|
| `apierror` | Structured API error responses |
| `audit` | Audit event recording |
| `auth` | JWT token generation and validation |
| `config` | Environment-based configuration |
| `logger` | Structured logging |
| `metrics` | Application metrics |

---

## Provider Pattern

Impress uses a **Provider** pattern to abstract external dependencies behind interfaces. This allows swapping implementations without changing business logic.

### Available Provider Interfaces

| Provider | Interface | Default Implementation | Purpose |
|----------|-----------|----------------------|---------|
| Search | `SearchProvider` | FTS5-based search service | Full-text search across content |
| Notifier | `NotifierProvider` | Log-based notifier | Send notifications (email, webhook, etc.) |
| Storage | `StorageProvider` | Local filesystem | File storage for media uploads |
| CAPTCHA | `CaptchaProvider` | No-op (disabled) | Anti-spam verification |

### Implementing a Provider

To implement a custom provider:

1. Implement the provider interface (e.g., `SearchProvider`)
2. Register it with the Provider Registry at startup
3. The system will use your implementation instead of the default

Example: implementing a custom notifier that sends emails:

```go
// internal/service/email_notifier.go
type EmailNotifier struct {
    smtpHost string
    smtpPort int
}

func (n *EmailNotifier) Notify(ctx context.Context, event string, data map[string]interface{}) error {
    // Send email notification
    return nil
}
```

Register it in `cmd/server/main.go` before the server starts.

---

## EventBus

The EventBus provides an in-process publish/subscribe system for decoupling components. When content is created, updated, or deleted, events are published to the bus, and any number of subscribers can react.

### Event Types

Content lifecycle events are defined in `internal/eventbus/events.go`:

- `content.created` -- a new content document was created
- `content.updated` -- a content document was modified
- `content.published` -- a draft was published
- `content.deleted` -- a content document was deleted
- `article.created`, `article.updated`, `article.deleted`
- `media.uploaded`, `media.deleted`
- `comment.created`, `comment.moderated`

### Subscribing to Events

```go
bus.Subscribe("content.published", func(event eventbus.Event) {
    // Invalidate cache, send notification, update search index, etc.
    log.Printf("Content published: %s", event.Payload["slug"])
})
```

### Design Notes

- The EventBus is synchronous by default (subscribers run in the same goroutine)
- Subscribers should be fast and non-blocking; offload heavy work to goroutines
- The bus is wired in `cmd/server/main.go` and passed to handlers/services that need it

---

## Provider Registry

The Provider Registry is a centralized place to register and resolve provider implementations at startup.

```go
// Register a provider
registry.Register("search", mySearchProvider)

// Resolve a provider
searchProvider := registry.Get("search").(SearchProvider)
```

Providers are registered in `cmd/server/main.go` during initialization. The registry supports:

- Registering providers by name
- Listing all registered providers
- Checking if a provider is registered

---

## Database

### Supported Databases

| Database | Use Case | DSN Example |
|----------|----------|-------------|
| SQLite | Local development | `file:./data/blotting.db?cache=shared&mode=rwc` |
| PostgreSQL | Production / Docker | `postgres://user:pass@localhost:5432/impress?sslmode=disable` |

The `DB_DSN` environment variable determines which database is used. GORM abstracts the differences.

### Migrations

Database migrations use [goose](https://github.com/pressly/goose) and are stored in `backend/migrations/`. Migrations run automatically on server startup.

To create a new migration:

```bash
goose -dir backend/migrations create <name> sql
```

### GORM Conventions

- Models embed `gorm.Model` for `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`
- Use struct tags for column definitions: `gorm:"type:varchar(255);not null"`
- Soft deletes are enabled by default via `gorm.Model`
- JSON fields use `datatypes.JSON` for SQLite/PostgreSQL compatibility

---

## Frontend Architecture

### Technology Stack

- **React 19** with functional components and hooks
- **TypeScript** with strict mode
- **Vite 7** for build tooling
- **Tailwind CSS 3** for styling
- **React Router 7** for client-side routing
- **i18next** for internationalization
- **axios** for API communication

### Provider Hierarchy

The application wraps components in a provider hierarchy defined in `App.tsx`:

```
I18nextProvider          -- translations
  └─ BrowserRouter       -- client-side routing
    └─ ThemeProvider      -- CMS-driven theme
      └─ GlobalConfigProvider  -- site-wide config per locale
        └─ AuthProvider   -- JWT auth state
          └─ AppRoutes    -- route definitions
```

### Routing

Routes are defined in `src/router/config.tsx`. All route components are lazy-loaded:

- `/` -- home page (CMS-driven)
- `/about`, `/services`, `/contact` -- static pages
- `/p/*` -- dynamic CMS pages (resolved by slug)
- `/admin/*` -- admin panel (protected)

### Theme System

The theme system enables CMS-driven page rendering:

```
src/theme/
├── DynamicPage.tsx       # Fetches page config, renders sections
├── SectionRenderer.tsx   # Maps section types to components
├── packages/             # Theme packages (CSS variables, layouts)
│   ├── default/
│   ├── modern-dark/
│   └── warm-earth/
└── sections/             # Section components
    ├── HeroSection.tsx
    ├── CardGridSection.tsx
    ├── ContactFormSection.tsx
    └── ...
```

A page config is a JSON document that defines an array of sections. Each section has a `type` and locale-specific `props`. The `SectionRenderer` maps each section type to a React component.

### API Client

The API client is centralized in `src/api/`:

- `http.ts` -- axios instance with base URL and interceptors
- `pages.ts`, `articles.ts`, etc. -- domain-specific typed fetch functions
- The `VITE_API_BASE_URL` env var configures the backend origin

### Internationalization

- Translation resources in `src/i18n/local/{zh,en}/common.ts`
- `zh` is the fallback locale
- `useTranslation` hook is auto-imported (do not import manually)
- Locale detection via `i18next-browser-languagedetector`

---

## Adding a New Feature

Follow this checklist when adding a new domain feature to the backend:

### 1. Define the Model

Create `backend/internal/model/<domain>.go`:

```go
package model

import "gorm.io/gorm"

type Widget struct {
    gorm.Model
    Name        string `gorm:"type:varchar(255);not null" json:"name"`
    Description string `gorm:"type:text" json:"description"`
    Locale      string `gorm:"type:varchar(10);default:'zh'" json:"locale"`
    Status      string `gorm:"type:varchar(50);default:'draft'" json:"status"`
}
```

### 2. Create a Migration

```bash
goose -dir backend/migrations create add_widgets_table sql
```

Edit the generated file to add the `CREATE TABLE` statement.

### 3. Define the Repository Interface

Create `backend/internal/repository/<domain>_repository.go`:

```go
package repository

import "impress/internal/model"

type WidgetRepository interface {
    Create(widget *model.Widget) error
    FindByID(id uint) (*model.Widget, error)
    FindAll(page, pageSize int) ([]model.Widget, int64, error)
    Update(widget *model.Widget) error
    Delete(id uint) error
}
```

### 4. Implement the Repository

Create `backend/internal/repository/<domain>_repository_impl.go`:

```go
package repository

import (
    "gorm.io/gorm"
    "impress/internal/model"
)

type widgetRepositoryImpl struct {
    db *gorm.DB
}

func NewWidgetRepository(db *gorm.DB) WidgetRepository {
    return &widgetRepositoryImpl{db: db}
}

func (r *widgetRepositoryImpl) Create(widget *model.Widget) error {
    return r.db.Create(widget).Error
}

// ... implement other methods
```

### 5. Add Business Logic (if needed)

Create `backend/internal/service/<domain>_service.go` for any business rules beyond simple CRUD.

### 6. Create the Handler

Create `backend/internal/handler/<domain>/handler.go`:

```go
package widget

import (
    "github.com/gin-gonic/gin"
    "impress/internal/repository"
)

type Handler struct {
    repo repository.WidgetRepository
}

func NewHandler(repo repository.WidgetRepository) *Handler {
    return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(public, admin *gin.RouterGroup) {
    public.GET("/widgets", h.List)
    public.GET("/widgets/:id", h.Get)
    admin.POST("/widgets", h.Create)
    admin.PUT("/widgets/:id", h.Update)
    admin.DELETE("/widgets/:id", h.Delete)
}
```

### 7. Wire It Up

In `cmd/server/main.go`:

```go
widgetRepo := repository.NewWidgetRepository(db)
widgetHandler := widget.NewHandler(widgetRepo)
widgetHandler.RegisterRoutes(publicGroup, adminGroup)
```

### 8. Add Tests

- Model tests in `internal/model/<domain>_test.go`
- Repository tests in `internal/repository/<domain>_test.go`
- Handler tests (if complex logic) in `internal/handler/<domain>/<domain>_test.go`

### 9. Verify

```bash
cd backend && go vet ./... && go test -v -race ./...
```

---

## Environment Variables

### Backend

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8088` |
| `DB_DSN` | Database connection string | (required) |
| `JWT_SECRET` | Secret for access tokens | (required) |
| `JWT_REFRESH_SECRET` | Secret for refresh tokens | (required) |
| `ENV` | Environment (`development`, `production`) | `development` |
| `UPLOAD_DIR` | Directory for uploaded files | `./uploads` |
| `FRONTEND_DIR` | Directory for SPA static files | (optional) |

### Frontend (build-time)

| Variable | Description |
|----------|-------------|
| `VITE_API_BASE_URL` | Backend API origin |
| `BASE_PATH` | Base path for the SPA |
| `IS_PREVIEW` | Preview mode flag |
