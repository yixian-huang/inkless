# Phase 0: Prerequisites Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish database migration tooling, SPA meta injection, testing strategy, and API versioning before Phase 1 begins.

**Architecture:** Introduce goose for schema migrations alongside existing GORM AutoMigrate. Add Go template rendering of index.html for dynamic meta tags. Define testing conventions and API version prefix strategy.

**Tech Stack:** Go (goose migrations), Go html/template (meta injection), Vitest + go test (testing)

**Spec:** `docs/superpowers/specs/2026-03-11-open-source-evolution-design.md`

---

## File Structure

```
backend/
├── internal/db/
│   ├── db.go                    (modify - add goose integration)
│   └── migrations/              (create - migration SQL files)
│       ├── 00001_baseline.sql
│       └── embed.go
├── internal/seo/
│   ├── meta.go                  (create - meta tag data structures)
│   ├── meta_test.go             (create - unit tests)
│   ├── renderer.go              (create - index.html template renderer)
│   └── renderer_test.go         (create - renderer tests)
├── cmd/server/main.go           (modify - wire SEO renderer + migration)
docs/
├── testing-strategy.md          (create - testing conventions)
├── api-versioning.md            (create - API version strategy)
frontend/
├── index.html                   (modify - add template placeholders)
```

---

## Chunk 1: Database Migration Tooling (Task 0.1)

### Task 1: Introduce goose migration framework

**Files:**
- Create: `backend/internal/db/migrations/embed.go`
- Create: `backend/internal/db/migrations/00001_baseline.sql`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/go.mod` (via go get)

- [ ] **Step 1: Add goose dependency**

```bash
cd /home/dev/impress/backend && go get github.com/pressly/goose/v3
```

- [ ] **Step 2: Create migrations embed file**

Create `backend/internal/db/migrations/embed.go`:

```go
package migrations

import "embed"

//go:embed *.sql
var EmbedMigrations embed.FS
```

- [ ] **Step 3: Create baseline migration**

Create `backend/internal/db/migrations/00001_baseline.sql`:

```sql
-- +goose Up
-- Baseline migration: documents existing schema managed by GORM AutoMigrate.
-- This is a no-op migration that marks the starting point for goose-managed migrations.
-- All tables up to this point are created by GORM AutoMigrate in main.go.

-- +goose Down
-- Baseline cannot be rolled back.
```

- [ ] **Step 4: Write integration test for goose setup**

Create `backend/internal/db/migrations_test.go`:

```go
package db_test

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	"blotting-consultancy/internal/db/migrations"
	_ "github.com/mattn/go-sqlite3"
)

func TestGooseMigrationsEmbed(t *testing.T) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.EmbedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}

	// Should run baseline without error
	if err := goose.Up(sqlDB, "."); err != nil {
		t.Fatalf("goose up: %v", err)
	}

	// Verify version is 1
	ver, err := goose.GetDBVersion(sqlDB)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if ver != 1 {
		t.Errorf("expected version 1, got %d", ver)
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd /home/dev/impress/backend && go test -v -run TestGooseMigrationsEmbed ./internal/db/
```

Expected: PASS

- [ ] **Step 6: Wire goose into main.go startup**

In `backend/cmd/server/main.go`, after the existing `migrator.AutoMigrate(...)` call and `migrator.RunMigrations(...)` call, add goose migration execution:

```go
// Run goose migrations (for schema changes beyond GORM AutoMigrate)
{
	sqlDB, err := database.DB.DB()
	if err != nil {
		logger.Fatal("Failed to get sql.DB for migrations", "error", err)
	}
	goose.SetBaseFS(migrations.EmbedMigrations)
	dialect := "sqlite3"
	if db.IsPostgresDSN(cfg.DBDSN) {
		dialect = "postgres"
	}
	if err := goose.SetDialect(dialect); err != nil {
		logger.Fatal("Failed to set goose dialect", "error", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		logger.Fatal("Failed to run goose migrations", "error", err)
	}
	logger.Info("Goose migrations applied successfully")
}
```

Add imports: `"github.com/pressly/goose/v3"`, `"blotting-consultancy/internal/db/migrations"`, `"blotting-consultancy/internal/db"` (for `db.IsPostgresDSN`).

- [ ] **Step 7: Verify backend compiles and starts**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

Expected: Build succeeds with no errors.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/db/migrations/ backend/internal/db/migrations_test.go backend/cmd/server/main.go backend/go.mod backend/go.sum
git commit -m "feat(0.1): introduce goose migration framework alongside GORM AutoMigrate"
```

---

## Chunk 2: SPA Meta Injection (Task 0.2)

### Task 2: Create SEO meta data structures

**Files:**
- Create: `backend/internal/seo/meta.go`
- Create: `backend/internal/seo/meta_test.go`

- [ ] **Step 1: Write test for PageMeta struct and defaults**

Create `backend/internal/seo/meta_test.go`:

```go
package seo_test

import (
	"testing"

	"blotting-consultancy/internal/seo"
)

func TestDefaultPageMeta(t *testing.T) {
	meta := seo.DefaultPageMeta()
	if meta.Title == "" {
		t.Error("default title should not be empty")
	}
	if meta.OgType != "website" {
		t.Errorf("expected og:type 'website', got %q", meta.OgType)
	}
	if meta.Locale != "zh" {
		t.Errorf("expected default locale 'zh', got %q", meta.Locale)
	}
}

func TestPageMetaWithOverrides(t *testing.T) {
	meta := seo.DefaultPageMeta()
	meta.Title = "Custom Title"
	meta.Description = "Custom Desc"
	meta.OgImage = "https://example.com/img.png"
	meta.CanonicalURL = "https://example.com/about"

	if meta.Title != "Custom Title" {
		t.Errorf("expected custom title, got %q", meta.Title)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/dev/impress/backend && go test -v -run TestDefaultPageMeta ./internal/seo/
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement PageMeta**

Create `backend/internal/seo/meta.go`:

```go
package seo

// PageMeta holds all meta tag values for server-side injection into index.html.
type PageMeta struct {
	Title        string
	Description  string
	Keywords     string
	CanonicalURL string
	Locale       string

	// Open Graph
	OgTitle       string
	OgDescription string
	OgImage       string
	OgURL         string
	OgType        string

	// Twitter Card
	TwitterCard string
}

const (
	defaultTitle       = "印迹法规咨询 - 企业内设型法规团队 | 专业法规咨询服务"
	defaultDescription = "印迹法规咨询（Blotting Consultancy）- 为企业提供专业的内设型法规团队服务"
)

// DefaultPageMeta returns meta with sensible defaults matching current index.html.
func DefaultPageMeta() PageMeta {
	return PageMeta{
		Title:        defaultTitle,
		Description:  defaultDescription,
		Locale:       "zh",
		OgType:       "website",
		OgTitle:      defaultTitle,
		OgDescription: defaultDescription,
		TwitterCard:  "summary_large_image",
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /home/dev/impress/backend && go test -v -run TestDefaultPageMeta ./internal/seo/
cd /home/dev/impress/backend && go test -v -run TestPageMetaWithOverrides ./internal/seo/
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/seo/
git commit -m "feat(0.2): add SEO PageMeta data structures"
```

### Task 3: Create index.html template renderer

**Files:**
- Modify: `frontend/index.html` (add template placeholders)
- Create: `backend/internal/seo/renderer.go`
- Create: `backend/internal/seo/renderer_test.go`

- [ ] **Step 1: Write renderer test**

Create `backend/internal/seo/renderer_test.go`:

```go
package seo_test

import (
	"strings"
	"testing"

	"blotting-consultancy/internal/seo"
)

const testTemplate = `<!DOCTYPE html>
<html lang="{{.Locale}}">
<head>
  <title>{{.Title}}</title>
  <meta name="description" content="{{.Description}}">
  <meta property="og:title" content="{{.OgTitle}}">
  <meta property="og:description" content="{{.OgDescription}}">
  <meta property="og:image" content="{{.OgImage}}">
  <meta property="og:url" content="{{.OgURL}}">
  <meta property="og:type" content="{{.OgType}}">
  <link rel="canonical" href="{{.CanonicalURL}}">
</head>
<body><div id="root"></div></body>
</html>`

func TestRendererRender(t *testing.T) {
	r, err := seo.NewRendererFromString(testTemplate)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}

	meta := seo.DefaultPageMeta()
	meta.Title = "About Us"
	meta.CanonicalURL = "https://example.com/about"

	result, err := r.Render(meta)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "<title>About Us</title>") {
		t.Error("expected custom title in output")
	}
	if !strings.Contains(result, `href="https://example.com/about"`) {
		t.Error("expected canonical URL in output")
	}
	if !strings.Contains(result, `lang="zh"`) {
		t.Error("expected locale in html lang attribute")
	}
}

func TestRendererDefaultMeta(t *testing.T) {
	r, err := seo.NewRendererFromString(testTemplate)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}

	meta := seo.DefaultPageMeta()
	result, err := r.Render(meta)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "印迹法规咨询") {
		t.Error("expected default title in output")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/dev/impress/backend && go test -v -run TestRendererRender ./internal/seo/
```

Expected: FAIL — NewRendererFromString not found.

- [ ] **Step 3: Implement Renderer**

Create `backend/internal/seo/renderer.go`:

```go
package seo

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
)

// Renderer renders index.html with dynamic meta tags.
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer creates a renderer from an index.html file path.
func NewRenderer(filePath string) (*Renderer, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	return NewRendererFromString(string(content))
}

// NewRendererFromString creates a renderer from a template string.
func NewRendererFromString(tmplStr string) (*Renderer, error) {
	t, err := template.New("index").Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	return &Renderer{tmpl: t}, nil
}

// Render executes the template with the given PageMeta.
func (r *Renderer) Render(meta PageMeta) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, meta); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v ./internal/seo/
```

Expected: All PASS

- [ ] **Step 5: Convert frontend index.html to Go template**

Modify `frontend/index.html`: Replace hard-coded meta tags with Go template placeholders. The key change is replacing static values with `{{.Field}}` syntax.

Replace the `<html>` opening tag:
```html
<html lang="{{.Locale}}">
```

Replace the `<head>` content (title, meta description, keywords, OG tags, canonical, twitter) with template variables. Keep all `<script>`, `<link>` (CSS), and Vite entries unchanged.

**Important:** Vite's dev server serves index.html directly (no Go template), so template placeholders must be valid HTML when rendered literally (they will show as `{{.Title}}` in dev mode). This is acceptable — dev mode uses the Vite dev server, not the Go backend.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/seo/ frontend/index.html
git commit -m "feat(0.2): SPA meta injection renderer with Go html/template"
```

### Task 4: Wire SEO renderer into SPA fallback

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Modify SPA fallback handler to use Renderer**

In `main.go`, the current SPA fallback serves `index.html` as a static file. Change it to:

1. Create a `seo.Renderer` at startup (if `FRONTEND_DIR` is set):
```go
var seoRenderer *seo.Renderer
if cfg.FrontendDir != "" {
    indexPath := filepath.Join(cfg.FrontendDir, "index.html")
    var err error
    seoRenderer, err = seo.NewRenderer(indexPath)
    if err != nil {
        logger.Fatal("Failed to create SEO renderer", "error", err)
    }
}
```

2. In the SPA fallback handler, instead of `c.File(indexPath)`, use:
```go
meta := seo.DefaultPageMeta()
// Future: resolve meta from request path via content/article/page lookups
html, err := seoRenderer.Render(meta)
if err != nil {
    c.File(filepath.Join(cfg.FrontendDir, "index.html"))
    return
}
c.Data(200, "text/html; charset=utf-8", []byte(html))
```

- [ ] **Step 2: Verify backend compiles**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

Expected: Build succeeds.

- [ ] **Step 3: Verify the rendered page loads correctly**

Start the backend and verify that navigating to the root URL returns HTML with the default meta tags rendered (not raw template syntax).

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(0.2): wire SEO renderer into SPA fallback handler"
```

---

## Chunk 3: Testing Strategy & API Versioning (Tasks 0.3, 0.4)

### Task 5: Define testing strategy

**Files:**
- Create: `docs/testing-strategy.md`

- [ ] **Step 1: Write testing strategy document**

Create `docs/testing-strategy.md`:

```markdown
# Testing Strategy

## Requirements by Phase

### All Phases
- **Backend API endpoints**: Integration test per endpoint (happy path + key error cases)
- **Backend services/repositories**: Unit tests for business logic
- **Frontend components**: Basic render test for new components
- **Frontend pages**: Smoke test ensuring page renders without crash

### Test Commands
- Backend: `cd backend && go test -v -race ./...`
- Frontend: `cd frontend && pnpm test:run`
- Full check: `pnpm lint && pnpm type-check && pnpm test`

### Coverage Expectations
- New backend packages: ≥ 70% line coverage
- New frontend components: At least 1 render test per component
- API endpoints: At least happy-path test per public endpoint

### Test File Conventions
- Backend: `*_test.go` in same package, or `package_test` for integration tests
- Frontend: `*.test.tsx` co-located with component file

### Integration Test Pattern (Backend)
Use in-memory SQLite for handler/service tests:
```go
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    db.AutoMigrate(&model.Article{}, ...)
    return db
}
```

### Frontend Test Pattern
```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

describe("ComponentName", () => {
  it("renders without crash", () => {
    render(<ComponentName />);
    expect(screen.getByText("expected text")).toBeInTheDocument();
  });
});
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/testing-strategy.md
git commit -m "docs(0.3): define testing strategy for all phases"
```

### Task 6: Define API versioning strategy

**Files:**
- Create: `docs/api-versioning.md`

- [ ] **Step 1: Write API versioning document**

Create `docs/api-versioning.md`:

```markdown
# API Versioning Strategy

## Current State
All routes use implicit v1 via path prefixes: `/public/*`, `/admin/*`, `/auth/*`.

## Strategy
- **No prefix change now**: Existing routes remain as-is (no `/api/v1/` prefix retrofit).
- **Plugin routes**: `GET/POST /api/v1/plugins/{pluginId}/*` (introduced in Phase 3).
- **Breaking changes**: If a v2 is ever needed, new routes at `/api/v2/*` while v1 remains supported for 2 major releases.
- **Additive changes**: New fields in responses are NOT breaking. Clients should ignore unknown fields.
- **Deprecation**: Deprecated fields get a `deprecated` note in Swagger docs for 1 release before removal.

## Response Envelope
Existing pattern (direct JSON) remains. No wrapper envelope.

## Versioning Headers
- `X-API-Version: 1` response header added to all API responses (optional, for client debugging).
```

- [ ] **Step 2: Commit**

```bash
git add docs/api-versioning.md
git commit -m "docs(0.4): define API versioning strategy"
```

- [ ] **Step 3: Run full verification**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check && pnpm test
cd /home/dev/impress/backend && go vet ./... && go test -v -race ./...
```

Expected: All pass. Phase 0 complete.
