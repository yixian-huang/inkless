# Phase 2: Developer Experience Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Lower the barrier to entry for contributors and prepare abstraction points that bridge Phase 1 features to the Phase 3 plugin architecture. Deliverables: auto-generated API docs (Swagger UI), a CLI tool for common operations, an EventBus + Provider Registry system, and community infrastructure (CONTRIBUTING.md, templates, docs site).

**Architecture:** Phase 2 builds on existing patterns. New backend packages follow the handler/service/repository layering. The CLI is a separate `cmd/impress/` binary using cobra. EventBus and Provider Registry are new `internal/` packages wired into `cmd/server/main.go`. Community files are root-level markdown and `.github/` templates.

**Tech Stack:** Go/Gin/GORM (backend), swaggo/swag + gin-swagger (OpenAPI), cobra (CLI), sync-based EventBus (in-process), VitePress (docs site)

**Spec:** `docs/superpowers/specs/2026-03-11-open-source-evolution-design.md` — Phase 2

**Prerequisites:** Phase 0 and Phase 1 MUST be complete. Phase 1 provides: SearchProvider interface (`internal/provider/search.go`), NotifierProvider interface (`internal/provider/notifier.go`), CaptchaProvider interface (`internal/provider/captcha.go`), SearchService with FTS5 (`internal/service/search_service.go`), comment system with anti-spam, goose migrations, SPA meta injection.

**Convention note:** New handlers use `RegisterRoutes(public, admin)` method for route setup (same as Phase 1 comment/search/seo handlers). The CLI binary lives in `cmd/impress/` alongside the existing `cmd/server/`.

---

## File Structure Overview

```
backend/
├── cmd/impress/                          (2.2 CLI)
│   ├── main.go                           (create - cobra root command)
│   ├── cmd_init.go                       (create - impress init)
│   ├── cmd_serve.go                      (create - impress serve)
│   ├── cmd_migrate.go                    (create - impress migrate)
│   ├── cmd_seed.go                       (create - impress seed)
│   ├── cmd_export.go                     (create - impress export)
│   ├── cmd_import.go                     (create - impress import)
│   └── cmd_plugin.go                     (create - impress plugin create)
├── cmd/server/main.go                    (modify - wire EventBus, Registry, Swagger)
├── internal/eventbus/
│   ├── eventbus.go                       (create - EventBus interface + in-process impl)
│   ├── eventbus_test.go                  (create)
│   ├── events.go                         (create - content lifecycle event types)
│   └── events_test.go                    (create)
├── internal/provider/
│   ├── search.go                         (exists)
│   ├── notifier.go                       (exists)
│   ├── captcha.go                        (exists)
│   ├── storage.go                        (create - StorageProvider interface)
│   └── registry.go                       (create - Provider Registry)
│   └── registry_test.go                  (create)
├── internal/service/
│   ├── search_service.go                 (exists - already implements SearchProvider)
│   ├── log_notifier.go                   (create - default NotifierProvider impl)
│   └── local_storage.go                  (create - default StorageProvider impl)
├── docs/swagger/                         (generated - swag output)
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── go.mod                                (modify - add swag, cobra, gin-swagger deps)
frontend/                                 (no changes in Phase 2)
.github/
├── ISSUE_TEMPLATE/
│   ├── bug_report.md                     (create)
│   ├── feature_request.md                (create)
│   └── plugin_request.md                 (create)
├── pull_request_template.md              (create)
CONTRIBUTING.md                           (create)
docs/
├── site/                                 (create - VitePress docs site)
│   ├── package.json
│   ├── .vitepress/config.ts
│   ├── index.md
│   ├── guide/getting-started.md
│   ├── guide/architecture.md
│   ├── guide/extension-points.md
│   └── guide/first-plugin.md
```

---

## Chunk 1: OpenAPI Documentation (Tasks 2.1.1 — 2.1.4)

### Task 1: Add swaggo/swag and gin-swagger dependencies (2.1.1)

**Files:**
- Modify: `backend/go.mod`
- Create: `backend/docs/swagger/` (generated)

- [ ] **Step 1: Install swag CLI and add Go dependencies**

```bash
cd /home/dev/impress/backend && go install github.com/swaggo/swag/cmd/swag@latest
cd /home/dev/impress/backend && go get -u github.com/swaggo/swag
cd /home/dev/impress/backend && go get -u github.com/swaggo/gin-swagger
cd /home/dev/impress/backend && go get -u github.com/swaggo/files
```

- [ ] **Step 2: Add top-level Swagger annotations to main.go**

In `backend/cmd/server/main.go`, add these comments above the `main()` function:

```go
// @title           Impress CMS API
// @version         1.0
// @description     Bilingual CMS backend API for Impress (印迹). Supports content management, articles, pages, themes, media, comments, search, and SEO.
// @host            localhost:8088
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}" for JWT authentication
```

- [ ] **Step 3: Generate initial swagger docs**

```bash
cd /home/dev/impress/backend && swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
```

Verify output files exist:
```bash
ls -la /home/dev/impress/backend/docs/swagger/
```

Expected: `docs.go`, `swagger.json`, `swagger.yaml`

- [ ] **Step 4: Verify build**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

- [ ] **Step 5: Commit**

```bash
git add backend/go.mod backend/go.sum backend/cmd/server/main.go backend/docs/swagger/
git commit -m "feat(2.1.1): add swaggo/swag annotations and generate initial Swagger docs"
```

### Task 2: Swagger UI route (2.1.3)

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add Swagger UI route to main.go**

Add import:
```go
_ "blotting-consultancy/docs/swagger" // swagger docs
swaggerFiles "github.com/swaggo/files"
ginSwagger "github.com/swaggo/gin-swagger"
```

Add route after the `/metrics` endpoint (before public group):
```go
// Swagger API documentation (no auth required)
router.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

- [ ] **Step 2: Verify build and route**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(2.1.3): add /api-docs Swagger UI route with gin-swagger"
```

### Task 3: Annotate auth handler (2.1.2 — batch 1)

**Files:**
- Modify: `backend/internal/handler/auth/handler.go`

- [ ] **Step 1: Add Swagger annotations to auth handler methods**

Add above `Login`:
```go
// Login authenticates a user and returns JWT tokens.
// @Summary      User login
// @Description  Authenticate with username and password, receive JWT access + refresh tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body object{username=string,password=string} true "Login credentials"
// @Success      200 {object} object{token=string,refreshToken=string,user=object}
// @Failure      400 {object} object{error=string}
// @Failure      401 {object} object{error=string}
// @Router       /auth/login [post]
```

Add above `Refresh`:
```go
// Refresh issues a new access token using a refresh token.
// @Summary      Refresh token
// @Description  Exchange a valid refresh token for a new access token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body object{refreshToken=string} true "Refresh token"
// @Success      200 {object} object{token=string}
// @Failure      401 {object} object{error=string}
// @Router       /auth/refresh [post]
```

Add above `Me`:
```go
// Me returns the current authenticated user's profile.
// @Summary      Get current user
// @Description  Returns profile of the currently authenticated user
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object{user=object}
// @Failure      401 {object} object{error=string}
// @Router       /auth/me [get]
```

Add above `Logout`:
```go
// Logout invalidates the user's refresh token.
// @Summary      User logout
// @Description  Invalidate the refresh token to log out
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body object{refreshToken=string} true "Refresh token to invalidate"
// @Success      200 {object} object{message=string}
// @Router       /auth/logout [post]
```

- [ ] **Step 2: Regenerate swagger docs and verify**

```bash
cd /home/dev/impress/backend && swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/auth/handler.go backend/docs/swagger/
git commit -m "feat(2.1.2): add Swagger annotations to auth handler"
```

### Task 4: Annotate article handler (2.1.2 — batch 2)

**Files:**
- Modify: `backend/internal/handler/article/handler.go`

- [ ] **Step 1: Add Swagger annotations to article handler methods**

Add above `PublicList`:
```go
// PublicList returns a paginated list of published articles.
// @Summary      List published articles
// @Description  Returns paginated published articles with optional category/tag filtering
// @Tags         Articles
// @Produce      json
// @Param        page      query int    false "Page number"    default(1)
// @Param        pageSize  query int    false "Items per page" default(10)
// @Param        category  query string false "Category slug filter"
// @Param        tag       query string false "Tag slug filter"
// @Success      200 {object} object{articles=[]object,total=int,page=int,pageSize=int}
// @Router       /public/articles [get]
```

Add above `PublicGetBySlug`:
```go
// PublicGetBySlug returns a single published article by slug.
// @Summary      Get article by slug
// @Description  Returns a single published article with full content
// @Tags         Articles
// @Produce      json
// @Param        slug path string true "Article slug"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /public/articles/{slug} [get]
```

Add above `AdminList`:
```go
// AdminList returns all articles for admin management.
// @Summary      List all articles (admin)
// @Description  Returns paginated articles including drafts for admin management
// @Tags         Articles (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int    false "Page number"    default(1)
// @Param        pageSize query int    false "Items per page" default(10)
// @Param        status   query string false "Status filter (draft/published/scheduled)"
// @Success      200 {object} object{articles=[]object,total=int}
// @Failure      401 {object} object{error=string}
// @Router       /admin/articles [get]
```

Add above `AdminCreate`:
```go
// AdminCreate creates a new article.
// @Summary      Create article
// @Description  Create a new article (draft by default)
// @Tags         Articles (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Article data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/articles [post]
```

Add above `AdminUpdate`:
```go
// AdminUpdate updates an existing article.
// @Summary      Update article
// @Description  Update an existing article by ID
// @Tags         Articles (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Article ID"
// @Param        body body object true "Updated article data"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/articles/{id} [put]
```

Add above `AdminDelete`:
```go
// AdminDelete deletes an article.
// @Summary      Delete article
// @Description  Permanently delete an article by ID
// @Tags         Articles (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Article ID"
// @Success      200 {object} object{message=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/articles/{id} [delete]
```

- [ ] **Step 2: Regenerate and verify**

```bash
cd /home/dev/impress/backend && swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/article/handler.go backend/docs/swagger/
git commit -m "feat(2.1.2): add Swagger annotations to article handler"
```

### Task 5: Annotate remaining handlers (2.1.2 — batch 3)

**Files:**
- Modify: `backend/internal/handler/comment/handler.go`
- Modify: `backend/internal/handler/search/handler.go`
- Modify: `backend/internal/handler/page/handler.go`
- Modify: `backend/internal/handler/media/handler.go`
- Modify: `backend/internal/handler/content/handler.go`
- Modify: `backend/internal/handler/category/handler.go`
- Modify: `backend/internal/handler/tag/handler.go`
- Modify: `backend/internal/handler/menu/handler.go`
- Modify: `backend/internal/handler/public/handler.go`
- Modify: `backend/internal/handler/seo/handler.go`
- Modify: `backend/internal/handler/backup/handler.go`
- Modify: `backend/internal/handler/analytics/handler.go`
- Modify: `backend/internal/handler/auditlog/handler.go`
- Modify: `backend/internal/handler/theme/handler.go`
- Modify: `backend/internal/handler/installed_theme/handler.go`
- Modify: `backend/internal/handler/form_submission/handler.go`
- Modify: `backend/internal/handler/user/handler.go`
- Modify: `backend/internal/handler/bootstrap/handler.go`
- Modify: `backend/internal/handler/sitemap/handler.go`

- [ ] **Step 1: Annotate each handler**

For each handler, add `@Summary`, `@Description`, `@Tags`, `@Accept`/`@Produce`, `@Param`, `@Success`/`@Failure`, `@Security` (where auth required), and `@Router` annotations. Group by Swagger tags:

| Handler | Tag | Key Routes |
|---------|-----|------------|
| comment | Comments | POST /public/comments, GET /public/comments, GET/PATCH/DELETE /admin/comments/* |
| search | Search | GET /public/search, GET /public/search/suggest |
| page | Pages | GET /public/pages, GET /public/pages/:slug, CRUD /admin/pages/* |
| media | Media | POST /admin/media/upload, GET/DELETE /admin/media/* |
| content | Content | GET/PUT /admin/content/:pageKey/draft, POST publish/rollback |
| category | Categories | GET /public/categories, CRUD /admin/categories/* |
| tag | Tags | GET /public/tags, CRUD /admin/tags/* |
| menu | Menus | GET /public/menu, CRUD /admin/menus/* |
| public | Public Content | GET /public/content/:pageKey |
| seo | SEO | GET /robots.txt, GET/PUT /admin/seo/robots |
| backup | Backup | GET/POST /admin/backups/*, POST export/import |
| analytics | Analytics | GET /admin/analytics/summary |
| auditlog | Audit Logs | GET /admin/audit-logs |
| theme | Theme Tokens | GET/PUT /admin/theme |
| installed_theme | Themes | GET/POST/PUT/DELETE /admin/themes/* |
| form_submission | Form Submissions | POST /public/form-submissions, GET/PATCH/DELETE /admin/form-submissions/* |
| user | Users | CRUD /admin/users/* |
| bootstrap | Bootstrap | GET /public/bootstrap |
| sitemap | Sitemap | GET /sitemap.xml |

Follow the same annotation pattern as Tasks 3 and 4. Each method gets `@Summary`, `@Description`, `@Tags`, `@Router`, and appropriate `@Param`/`@Success`/`@Failure`/`@Security`.

- [ ] **Step 2: Regenerate and verify**

```bash
cd /home/dev/impress/backend && swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/ backend/docs/swagger/
git commit -m "feat(2.1.2): add Swagger annotations to all remaining handlers"
```

### Task 6: CI swagger generation step (2.1.4)

**Files:**
- Modify: `.github/workflows/quality-gate.yml`
- Create: `backend/scripts/check-swagger.sh`

- [ ] **Step 1: Create swagger check script**

Create `backend/scripts/check-swagger.sh`:
```bash
#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# Regenerate swagger docs
swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal

# Check if generated files differ from committed versions
if ! git diff --quiet docs/swagger/; then
  echo "ERROR: Swagger docs are out of date. Run 'swag init' and commit the changes."
  git diff --stat docs/swagger/
  exit 1
fi

echo "Swagger docs are up to date."
```

- [ ] **Step 2: Add swagger check to quality-gate workflow**

In `.github/workflows/quality-gate.yml`, add a step to the `backend-checks` job after "Run Go vet":

```yaml
      - name: Install swag CLI
        run: go install github.com/swaggo/swag/cmd/swag@latest

      - name: Verify Swagger docs are up to date
        run: |
          cd backend
          swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
          git diff --exit-code docs/swagger/
```

- [ ] **Step 3: Verify script runs locally**

```bash
chmod +x /home/dev/impress/backend/scripts/check-swagger.sh
cd /home/dev/impress/backend && bash scripts/check-swagger.sh
```

- [ ] **Step 4: Commit**

```bash
git add backend/scripts/check-swagger.sh .github/workflows/quality-gate.yml
git commit -m "feat(2.1.4): add CI step to verify Swagger docs stay in sync with code"
```

---

## Chunk 2: CLI Tooling (Tasks 2.2.1 — 2.2.7)

### Task 7: CLI scaffold with cobra (2.2.1)

**Files:**
- Create: `backend/cmd/impress/main.go`
- Modify: `backend/go.mod` (add cobra dependency)

- [ ] **Step 1: Add cobra dependency**

```bash
cd /home/dev/impress/backend && go get -u github.com/spf13/cobra
```

- [ ] **Step 2: Write test for root command**

Create `backend/cmd/impress/main_test.go`:

```go
package main

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	cmd := rootCmd()
	if cmd.Use != "impress" {
		t.Errorf("expected root command name 'impress', got %q", cmd.Use)
	}
	// Verify subcommands are registered
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	expected := []string{"init", "serve", "migrate", "seed"}
	for _, name := range expected {
		if !subNames[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestRootCmd ./cmd/impress/
```

Expected: FAIL — `rootCmd` not defined.

- [ ] **Step 4: Implement root command**

Create `backend/cmd/impress/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "impress",
		Short:   "Impress CMS CLI - manage your bilingual CMS instance",
		Long:    "impress is a command-line tool for managing Impress CMS.\nIt provides commands for initialization, migration, seeding, and data management.",
		Version: Version,
	}

	// Register subcommands
	cmd.AddCommand(initCmd())
	cmd.AddCommand(serveCmd())
	cmd.AddCommand(migrateCmd())
	cmd.AddCommand(seedCmd())
	cmd.AddCommand(exportCmd())
	cmd.AddCommand(importCmd())
	cmd.AddCommand(pluginCmd())

	return cmd
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Create stub subcommands**

Create `backend/cmd/impress/cmd_init.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Impress CMS project",
		Long:  "Interactive project initialization: choose database, port, and generate configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress init: not yet implemented")
			return nil
		},
	}
}
```

Create `backend/cmd/impress/cmd_serve.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the Impress CMS development server",
		Long:  "Start the backend server with sensible defaults. Reads .env or environment variables.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress serve: not yet implemented")
			return nil
		},
	}
}
```

Create `backend/cmd/impress/cmd_migrate.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration management",
		Long:  "Run, rollback, or check status of database migrations.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress migrate up: not yet implemented")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress migrate down: not yet implemented")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress migrate status: not yet implemented")
			return nil
		},
	})

	return cmd
}
```

Create `backend/cmd/impress/cmd_seed.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func seedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long:  "Populate the database with example content for development and demos.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress seed: not yet implemented")
			return nil
		},
	}
}
```

Create `backend/cmd/impress/cmd_export.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func exportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export site data to JSON",
		Long:  "Export all site content (articles, pages, settings) to a JSON file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress export: not yet implemented")
			return nil
		},
	}
}
```

Create `backend/cmd/impress/cmd_import.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import site data from JSON",
		Long:  "Import site content from a previously exported JSON file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("impress import: not yet implemented")
			return nil
		},
	}
}
```

Create `backend/cmd/impress/cmd_plugin.go`:
```go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin management commands",
		Long:  "Create and manage Impress CMS plugins.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create [name]",
		Short: "Generate a new plugin project from template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("impress plugin create %s: not yet implemented\n", args[0])
			return nil
		},
	})

	return cmd
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -run TestRootCmd ./cmd/impress/
```

Expected: PASS

- [ ] **Step 7: Build CLI binary**

```bash
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/ && ./impress --help
```

- [ ] **Step 8: Commit**

```bash
git add backend/cmd/impress/ backend/go.mod backend/go.sum
git commit -m "feat(2.2.1): CLI scaffold with cobra - root command + stub subcommands"
```

### Task 8: `impress init` — interactive project setup (2.2.2)

**Files:**
- Modify: `backend/cmd/impress/cmd_init.go`

- [ ] **Step 1: Implement init command**

Replace `backend/cmd/impress/cmd_init.go` with:

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

const envTemplate = `# Impress CMS Configuration
# Generated by: impress init

PORT={{.Port}}
DB_DSN={{.DBDSN}}
JWT_SECRET={{.JWTSecret}}
JWT_REFRESH_SECRET={{.JWTRefreshSecret}}
ENV=development
UPLOAD_DIR=./uploads
# FRONTEND_DIR=./frontend/out
# BASE_URL=https://your-domain.com
# CORS_ALLOWED_ORIGINS=http://localhost:3000
`

type initConfig struct {
	Port             string
	DBDSN            string
	JWTSecret        string
	JWTRefreshSecret string
}

func initCmd() *cobra.Command {
	var nonInteractive bool
	var port, dbType, outputDir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Impress CMS project",
		Long:  "Interactive project initialization: choose database, port, and generate .env configuration file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := &initConfig{}
			reader := bufio.NewReader(os.Stdin)

			if nonInteractive {
				cfg.Port = port
				if dbType == "postgres" {
					cfg.DBDSN = "postgres://impress:impress@localhost:5432/impress?sslmode=disable"
				} else {
					cfg.DBDSN = "file:./data/blotting.db?cache=shared&mode=rwc"
				}
			} else {
				fmt.Print("Port [8088]: ")
				input, _ := reader.ReadString('\n')
				cfg.Port = strings.TrimSpace(input)
				if cfg.Port == "" {
					cfg.Port = "8088"
				}

				fmt.Print("Database (sqlite/postgres) [sqlite]: ")
				input, _ = reader.ReadString('\n')
				dbChoice := strings.TrimSpace(input)
				if dbChoice == "" || dbChoice == "sqlite" {
					cfg.DBDSN = "file:./data/blotting.db?cache=shared&mode=rwc"
				} else {
					fmt.Print("PostgreSQL DSN [postgres://impress:impress@localhost:5432/impress?sslmode=disable]: ")
					input, _ = reader.ReadString('\n')
					cfg.DBDSN = strings.TrimSpace(input)
					if cfg.DBDSN == "" {
						cfg.DBDSN = "postgres://impress:impress@localhost:5432/impress?sslmode=disable"
					}
				}
			}

			// Generate random-ish secrets (in production users should replace these)
			cfg.JWTSecret = "change-me-jwt-secret-" + fmt.Sprintf("%d", os.Getpid())
			cfg.JWTRefreshSecret = "change-me-jwt-refresh-" + fmt.Sprintf("%d", os.Getpid())

			// Write .env file
			outDir := outputDir
			if outDir == "" {
				outDir = "."
			}
			envPath := filepath.Join(outDir, ".env")

			tmpl, err := template.New("env").Parse(envTemplate)
			if err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
			}

			f, err := os.Create(envPath)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", envPath, err)
			}
			defer f.Close()

			if err := tmpl.Execute(f, cfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			// Create data and uploads directories
			os.MkdirAll(filepath.Join(outDir, "data"), 0755)
			os.MkdirAll(filepath.Join(outDir, "uploads"), 0755)

			fmt.Printf("Configuration written to %s\n", envPath)
			fmt.Println("Directories created: data/, uploads/")
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Review and update .env (especially JWT secrets)")
			fmt.Println("  2. Run: impress migrate up")
			fmt.Println("  3. Run: impress seed")
			fmt.Println("  4. Run: impress serve")
			return nil
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Skip interactive prompts")
	cmd.Flags().StringVar(&port, "port", "8088", "Server port (non-interactive mode)")
	cmd.Flags().StringVar(&dbType, "db", "sqlite", "Database type: sqlite or postgres (non-interactive mode)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for config files")

	return cmd
}
```

- [ ] **Step 2: Test non-interactive mode**

```bash
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/
./impress init --non-interactive --port 9090 --db sqlite -o /tmp/impress-test
cat /tmp/impress-test/.env
ls -la /tmp/impress-test/data/ /tmp/impress-test/uploads/
rm -rf /tmp/impress-test
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/impress/cmd_init.go
git commit -m "feat(2.2.2): impress init - interactive project setup with .env generation"
```

### Task 9: `impress serve` — one-line dev server (2.2.3)

**Files:**
- Modify: `backend/cmd/impress/cmd_serve.go`

- [ ] **Step 1: Implement serve command**

Replace `backend/cmd/impress/cmd_serve.go`. The serve command loads `.env` if present, sets defaults, then calls the same startup logic as `cmd/server/main.go`. For Phase 2, it delegates to `os/exec` to run the server binary:

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var envFile string
	var port string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Impress CMS development server",
		Long:  "Start the backend server. Loads .env file if present. Override with flags or env vars.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load .env file if it exists
			if envFile != "" {
				if err := loadEnvFile(envFile); err != nil {
					return fmt.Errorf("failed to load env file %s: %w", envFile, err)
				}
			} else if _, err := os.Stat(".env"); err == nil {
				loadEnvFile(".env")
			}

			// Apply flag overrides
			if port != "" {
				os.Setenv("PORT", port)
			}

			// Set sensible defaults if not configured
			setDefault("PORT", "8088")
			setDefault("DB_DSN", "file:./data/blotting.db?cache=shared&mode=rwc")
			setDefault("JWT_SECRET", "dev-jwt-secret-change-in-production")
			setDefault("JWT_REFRESH_SECRET", "dev-jwt-refresh-secret-change-in-production")
			setDefault("ENV", "development")
			setDefault("UPLOAD_DIR", "./uploads")

			// Find the server binary
			serverBin := findServerBinary()
			if serverBin == "" {
				return fmt.Errorf("server binary not found. Build it first:\n  cd backend && go build -o server ./cmd/server/")
			}

			fmt.Printf("Starting Impress CMS on port %s...\n", os.Getenv("PORT"))

			// Run the server binary with the current environment
			proc := exec.Command(serverBin)
			proc.Env = os.Environ()
			proc.Stdout = os.Stdout
			proc.Stderr = os.Stderr

			if err := proc.Start(); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			// Forward signals for graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigCh
				proc.Process.Signal(sig)
			}()

			return proc.Wait()
		},
	}

	cmd.Flags().StringVar(&envFile, "env-file", "", "Path to .env file (default: .env in current directory)")
	cmd.Flags().StringVarP(&port, "port", "p", "", "Override server port")

	return cmd
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Don't override existing env vars
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func setDefault(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

func findServerBinary() string {
	// Check common locations
	candidates := []string{
		"./server",
		"../server",
		filepath.Join(".", "cmd", "server", "server"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/ && echo "Build OK"
./impress serve --help
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/impress/cmd_serve.go
git commit -m "feat(2.2.3): impress serve - one-line dev server with .env loading"
```

### Task 10: `impress migrate` — database migration management (2.2.4)

**Files:**
- Modify: `backend/cmd/impress/cmd_migrate.go`

- [ ] **Step 1: Implement migrate command with goose integration**

Replace `backend/cmd/impress/cmd_migrate.go`:

```go
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"blotting-consultancy/internal/db/migrations"
)

func migrateCmd() *cobra.Command {
	var dsn string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration management",
		Long:  "Run, rollback, or check status of goose database migrations.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if dsn == "" {
				dsn = os.Getenv("DB_DSN")
			}
			if dsn == "" {
				dsn = "file:./data/blotting.db?cache=shared&mode=rwc"
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Database DSN (default: DB_DSN env var or SQLite)")

	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dialect, err := openDB(dsn)
			if err != nil {
				return err
			}
			defer db.Close()
			goose.SetBaseFS(migrations.EmbedMigrations)
			if err := goose.SetDialect(dialect); err != nil {
				return err
			}
			return goose.Up(db, ".")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dialect, err := openDB(dsn)
			if err != nil {
				return err
			}
			defer db.Close()
			goose.SetBaseFS(migrations.EmbedMigrations)
			if err := goose.SetDialect(dialect); err != nil {
				return err
			}
			return goose.Down(db, ".")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dialect, err := openDB(dsn)
			if err != nil {
				return err
			}
			defer db.Close()
			goose.SetBaseFS(migrations.EmbedMigrations)
			if err := goose.SetDialect(dialect); err != nil {
				return err
			}
			return goose.Status(db, ".")
		},
	})

	return cmd
}

func openDB(dsn string) (*sql.DB, string, error) {
	dialect := "sqlite3"
	driver := "sqlite3"
	if isPostgresDSN(dsn) {
		dialect = "postgres"
		driver = "postgres"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open database: %w", err)
	}
	return db, dialect, nil
}

func isPostgresDSN(dsn string) bool {
	return len(dsn) > 8 && (dsn[:8] == "postgres" || dsn[:8] == "host=")
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/ && echo "Build OK"
./impress migrate --help
./impress migrate status --dsn "file:./data/blotting.db?cache=shared&mode=rwc"
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/impress/cmd_migrate.go
git commit -m "feat(2.2.4): impress migrate - goose-based migration management (up/down/status)"
```

### Task 11: `impress seed` — sample data population (2.2.5)

**Files:**
- Modify: `backend/cmd/impress/cmd_seed.go`

- [ ] **Step 1: Implement seed command**

Replace `backend/cmd/impress/cmd_seed.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/seed"
	"blotting-consultancy/internal/service"
)

func seedCmd() *cobra.Command {
	var dsn string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long:  "Populate the database with default admin user, example content, and themes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsn == "" {
				dsn = os.Getenv("DB_DSN")
			}
			if dsn == "" {
				dsn = "file:./data/blotting.db?cache=shared&mode=rwc"
			}

			maxOpen := 1
			maxIdle := 1
			var maxLife time.Duration
			if db.IsPostgresDSN(dsn) {
				maxOpen = 25
				maxIdle = 5
				maxLife = 5 * time.Minute
			}

			database, err := db.Init(db.InitOptions{
				DSN:         dsn,
				MaxOpenConn: maxOpen,
				MaxIdleConn: maxIdle,
				MaxLifetime: maxLife,
				LogLevel:    logger.Warn,
			})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			userRepo := repository.NewGormUserRepository(database.DB)
			contentDocRepo := repository.NewGormContentDocumentRepository(database.DB)
			installedThemeRepo := repository.NewGormInstalledThemeRepository(database.DB)
			pageRepo := repository.NewGormPageRepository(database.DB)
			themePageService := service.NewThemePageService(pageRepo)

			seeder := seed.NewSeeder(userRepo, contentDocRepo, installedThemeRepo, themePageService)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := seeder.SeedAll(ctx); err != nil {
				return fmt.Errorf("seeding failed: %w", err)
			}

			fmt.Println("Database seeded successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN (default: DB_DSN env var or SQLite)")
	return cmd
}
```

- [ ] **Step 2: Build and verify**

```bash
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/ && echo "Build OK"
./impress seed --help
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/impress/cmd_seed.go
git commit -m "feat(2.2.5): impress seed - populate database with default data"
```

### Task 12: `impress export/import` — site data portability (2.2.7)

**Files:**
- Modify: `backend/cmd/impress/cmd_export.go`
- Modify: `backend/cmd/impress/cmd_import.go`

- [ ] **Step 1: Implement export command**

Replace `backend/cmd/impress/cmd_export.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/db"
	"blotting-consultancy/internal/model"
)

type siteExport struct {
	ExportedAt string              `json:"exportedAt"`
	Version    string              `json:"version"`
	Articles   []model.Article     `json:"articles"`
	Pages      []model.Page        `json:"pages"`
	Categories []model.Category    `json:"categories"`
	Tags       []model.Tag         `json:"tags"`
	MenuGroups []model.MenuGroup   `json:"menuGroups"`
	MenuItems  []model.MenuItem    `json:"menuItems"`
}

func exportCmd() *cobra.Command {
	var dsn, outputFile string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export site data to JSON",
		Long:  "Export articles, pages, categories, tags, and menus to a JSON file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsn == "" {
				dsn = os.Getenv("DB_DSN")
			}
			if dsn == "" {
				dsn = "file:./data/blotting.db?cache=shared&mode=rwc"
			}

			maxOpen := 1
			maxIdle := 1
			var maxLife time.Duration
			if db.IsPostgresDSN(dsn) {
				maxOpen = 25
				maxIdle = 5
				maxLife = 5 * time.Minute
			}

			database, err := db.Init(db.InitOptions{
				DSN:         dsn,
				MaxOpenConn: maxOpen,
				MaxIdleConn: maxIdle,
				MaxLifetime: maxLife,
				LogLevel:    logger.Warn,
			})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			export := siteExport{
				ExportedAt: time.Now().UTC().Format(time.RFC3339),
				Version:    Version,
			}

			database.DB.Find(&export.Articles)
			database.DB.Find(&export.Pages)
			database.DB.Find(&export.Categories)
			database.DB.Find(&export.Tags)
			database.DB.Find(&export.MenuGroups)
			database.DB.Find(&export.MenuItems)

			data, err := json.MarshalIndent(export, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal export: %w", err)
			}

			if outputFile == "" {
				outputFile = fmt.Sprintf("impress-export-%s.json", time.Now().Format("20060102-150405"))
			}

			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			fmt.Printf("Exported to %s (%d articles, %d pages, %d categories, %d tags)\n",
				outputFile,
				len(export.Articles), len(export.Pages),
				len(export.Categories), len(export.Tags))
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: impress-export-TIMESTAMP.json)")
	return cmd
}
```

- [ ] **Step 2: Implement import command**

Replace `backend/cmd/impress/cmd_import.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"blotting-consultancy/internal/db"
)

func importCmd() *cobra.Command {
	var dsn, inputFile string

	cmd := &cobra.Command{
		Use:   "import [file]",
		Short: "Import site data from JSON",
		Long:  "Import site content from a previously exported JSON file. Creates new records without overwriting existing ones.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				inputFile = args[0]
			}
			if inputFile == "" {
				return fmt.Errorf("input file is required: impress import <file.json>")
			}

			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			var export siteExport
			if err := json.Unmarshal(data, &export); err != nil {
				return fmt.Errorf("failed to parse import file: %w", err)
			}

			if dsn == "" {
				dsn = os.Getenv("DB_DSN")
			}
			if dsn == "" {
				dsn = "file:./data/blotting.db?cache=shared&mode=rwc"
			}

			maxOpen := 1
			maxIdle := 1
			var maxLife time.Duration
			if db.IsPostgresDSN(dsn) {
				maxOpen = 25
				maxIdle = 5
				maxLife = 5 * time.Minute
			}

			database, err := db.Init(db.InitOptions{
				DSN:         dsn,
				MaxOpenConn: maxOpen,
				MaxIdleConn: maxIdle,
				MaxLifetime: maxLife,
				LogLevel:    logger.Warn,
			})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}

			var counts [4]int
			for i := range export.Articles {
				export.Articles[i].ID = 0 // reset ID for new insert
				if err := database.DB.Create(&export.Articles[i]).Error; err == nil {
					counts[0]++
				}
			}
			for i := range export.Pages {
				export.Pages[i].ID = 0
				if err := database.DB.Create(&export.Pages[i]).Error; err == nil {
					counts[1]++
				}
			}
			for i := range export.Categories {
				export.Categories[i].ID = 0
				if err := database.DB.Create(&export.Categories[i]).Error; err == nil {
					counts[2]++
				}
			}
			for i := range export.Tags {
				export.Tags[i].ID = 0
				if err := database.DB.Create(&export.Tags[i]).Error; err == nil {
					counts[3]++
				}
			}

			fmt.Printf("Imported: %d articles, %d pages, %d categories, %d tags\n",
				counts[0], counts[1], counts[2], counts[3])
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN")
	cmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input JSON file path")
	return cmd
}
```

- [ ] **Step 3: Build and verify**

```bash
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/ && echo "Build OK"
./impress export --help
./impress import --help
```

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/impress/cmd_export.go backend/cmd/impress/cmd_import.go
git commit -m "feat(2.2.7): impress export/import - site data portability in JSON format"
```

### Task 13: Add CLI build target to Makefile (2.2.1 follow-up)

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add build-cli target**

Add to `Makefile` after the `build-backend` target:

```makefile
build-cli: ## 编译 CLI 工具
	@cd backend && go build -ldflags '-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)' -o impress ./cmd/impress/
	@echo "Built CLI $(VERSION)"
```

And update the `build` target:
```makefile
build: build-backend build-cli ## 编译前后端 + CLI
	@cd frontend && pnpm build
```

- [ ] **Step 2: Verify**

```bash
make build-cli
./backend/impress --version
```

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(2.2.1): add build-cli Makefile target for impress CLI"
```

---

## Chunk 3: Abstraction Points (Tasks 2.3.1 — 2.3.8)

### Task 14: EventBus interface and in-process implementation (2.3.1)

**Files:**
- Create: `backend/internal/eventbus/eventbus.go`
- Create: `backend/internal/eventbus/eventbus_test.go`

- [ ] **Step 1: Write EventBus tests**

Create `backend/internal/eventbus/eventbus_test.go`:

```go
package eventbus_test

import (
	"sync"
	"testing"
	"time"

	"blotting-consultancy/internal/eventbus"
)

func TestPublishSync(t *testing.T) {
	bus := eventbus.New()
	received := false
	bus.Subscribe("test.event", eventbus.SyncHandler(func(e eventbus.Event) {
		received = true
		if e.Type != "test.event" {
			t.Errorf("expected event type test.event, got %s", e.Type)
		}
	}))

	bus.Publish(eventbus.Event{Type: "test.event", Payload: "hello"})

	if !received {
		t.Error("sync subscriber should have been called")
	}
}

func TestPublishAsync(t *testing.T) {
	bus := eventbus.New()
	var wg sync.WaitGroup
	wg.Add(1)
	bus.Subscribe("test.async", eventbus.AsyncHandler(func(e eventbus.Event) {
		defer wg.Done()
	}))

	bus.Publish(eventbus.Event{Type: "test.async"})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("async subscriber timed out")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := eventbus.New()
	count := 0
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		bus.Subscribe("multi", eventbus.SyncHandler(func(e eventbus.Event) {
			mu.Lock()
			count++
			mu.Unlock()
		}))
	}

	bus.Publish(eventbus.Event{Type: "multi"})

	mu.Lock()
	if count != 3 {
		t.Errorf("expected 3 calls, got %d", count)
	}
	mu.Unlock()
}

func TestUnsubscribe(t *testing.T) {
	bus := eventbus.New()
	called := false
	id := bus.Subscribe("unsub", eventbus.SyncHandler(func(e eventbus.Event) {
		called = true
	}))

	bus.Unsubscribe("unsub", id)
	bus.Publish(eventbus.Event{Type: "unsub"})

	if called {
		t.Error("unsubscribed handler should not be called")
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /home/dev/impress/backend && go test -v ./internal/eventbus/
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement EventBus**

Create `backend/internal/eventbus/eventbus.go`:

```go
package eventbus

import (
	"sync"
	"sync/atomic"
)

// Event represents a domain event published through the bus.
type Event struct {
	Type    string      // e.g. "content.published", "comment.created"
	Payload interface{} // event-specific data
}

// Handler is the function signature for event handlers.
type Handler func(Event)

// Subscriber wraps a handler with sync/async semantics.
type Subscriber struct {
	ID      uint64
	Handler Handler
	Async   bool
}

// SyncHandler creates a synchronous subscriber handler.
func SyncHandler(fn Handler) Subscriber {
	return Subscriber{Handler: fn, Async: false}
}

// AsyncHandler creates an asynchronous subscriber handler.
func AsyncHandler(fn Handler) Subscriber {
	return Subscriber{Handler: fn, Async: true}
}

// EventBus defines the publish/subscribe contract.
type EventBus interface {
	Publish(event Event)
	Subscribe(eventType string, sub Subscriber) uint64
	Unsubscribe(eventType string, id uint64)
}

// Bus is an in-process EventBus implementation.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]Subscriber
	nextID      atomic.Uint64
}

// New creates a new in-process EventBus.
func New() *Bus {
	return &Bus{
		subscribers: make(map[string][]Subscriber),
	}
}

// Publish dispatches an event to all subscribers of that type.
// Sync subscribers are called in order. Async subscribers run in goroutines.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	subs := make([]Subscriber, len(b.subscribers[event.Type]))
	copy(subs, b.subscribers[event.Type])
	b.mu.RUnlock()

	for _, sub := range subs {
		if sub.Async {
			go sub.Handler(event)
		} else {
			sub.Handler(event)
		}
	}
}

// Subscribe registers a handler for an event type. Returns a subscription ID.
func (b *Bus) Subscribe(eventType string, sub Subscriber) uint64 {
	id := b.nextID.Add(1)
	sub.ID = id
	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], sub)
	b.mu.Unlock()
	return id
}

// Unsubscribe removes a handler by subscription ID.
func (b *Bus) Unsubscribe(eventType string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[eventType]
	for i, s := range subs {
		if s.ID == id {
			b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -race ./internal/eventbus/
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/eventbus/
git commit -m "feat(2.3.1): EventBus interface with in-process sync/async implementation"
```

### Task 15: Content lifecycle events (2.3.2)

**Files:**
- Create: `backend/internal/eventbus/events.go`
- Create: `backend/internal/eventbus/events_test.go`

- [ ] **Step 1: Write event type tests**

Create `backend/internal/eventbus/events_test.go`:

```go
package eventbus_test

import (
	"testing"

	"blotting-consultancy/internal/eventbus"
)

func TestContentEventTypes(t *testing.T) {
	events := []string{
		eventbus.ContentCreated,
		eventbus.ContentUpdated,
		eventbus.ContentPublished,
		eventbus.ContentDeleted,
		eventbus.CommentCreated,
		eventbus.CommentApproved,
		eventbus.CommentDeleted,
	}
	for _, e := range events {
		if e == "" {
			t.Error("event type constant should not be empty")
		}
	}
}

func TestContentEventPayload(t *testing.T) {
	payload := eventbus.ContentEventPayload{
		ContentType: "article",
		ContentID:   1,
		Slug:        "test-article",
		Locale:      "zh",
		Title:       "Test",
		Action:      eventbus.ContentPublished,
	}
	if payload.ContentType != "article" {
		t.Errorf("unexpected content type: %s", payload.ContentType)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestContentEvent ./internal/eventbus/
```

- [ ] **Step 3: Implement event types**

Create `backend/internal/eventbus/events.go`:

```go
package eventbus

// Content lifecycle event types.
const (
	ContentCreated   = "content.created"
	ContentUpdated   = "content.updated"
	ContentPublished = "content.published"
	ContentDeleted   = "content.deleted"
)

// Comment lifecycle event types.
const (
	CommentCreated  = "comment.created"
	CommentApproved = "comment.approved"
	CommentDeleted  = "comment.deleted"
)

// ContentEventPayload carries data for content lifecycle events.
type ContentEventPayload struct {
	ContentType string // "article" or "page"
	ContentID   uint
	Slug        string
	Locale      string
	Title       string
	Action      string // the event type constant, for convenience
}

// CommentEventPayload carries data for comment lifecycle events.
type CommentEventPayload struct {
	CommentID   uint
	ContentType string // "article" or "page"
	ContentID   uint
	AuthorName  string
	Action      string
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -race ./internal/eventbus/
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/eventbus/events.go backend/internal/eventbus/events_test.go
git commit -m "feat(2.3.2): content and comment lifecycle event types"
```

### Task 16: Wire EventBus into existing features (2.3.3)

**Files:**
- Modify: `backend/internal/handler/article/handler.go` — publish events on create/update/delete
- Modify: `backend/internal/handler/comment/handler.go` — publish events on create/approve/delete
- Modify: `backend/cmd/server/main.go` — instantiate EventBus and pass to handlers

- [ ] **Step 1: Add EventBus to article handler**

In `backend/internal/handler/article/handler.go`, modify the `Handler` struct:

```go
type Handler struct {
	articleRepo   repository.ArticleRepository
	categoryRepo  repository.CategoryRepository
	tagRepo       repository.TagRepository
	searchService *service.SearchService
	eventBus      eventbus.EventBus
}
```

Update `NewHandler` to accept `eventBus eventbus.EventBus` parameter.

In `AdminCreate`, after successful creation, add:
```go
if h.eventBus != nil {
	h.eventBus.Publish(eventbus.Event{
		Type: eventbus.ContentCreated,
		Payload: eventbus.ContentEventPayload{
			ContentType: "article",
			ContentID:   article.ID,
			Slug:        article.Slug,
			Action:      eventbus.ContentCreated,
		},
	})
}
```

Add similar publish calls in `AdminUpdate` (ContentUpdated) and `AdminDelete` (ContentDeleted).

- [ ] **Step 2: Add EventBus to comment handler**

In `backend/internal/handler/comment/handler.go`, modify the `Handler` struct:

```go
type Handler struct {
	repo     repository.CommentRepository
	antispam *service.AntiSpamService
	eventBus eventbus.EventBus
}
```

Update `NewHandler` to accept `eventBus eventbus.EventBus` parameter.

In `PublicCreate`, after successful creation, publish `CommentCreated` event.

- [ ] **Step 3: Wire in main.go**

In `backend/cmd/server/main.go`:

Add import:
```go
"blotting-consultancy/internal/eventbus"
```

After service initialization, create the bus:
```go
// Initialize event bus
bus := eventbus.New()
log.Info("Event bus initialized")
```

Update handler instantiation to pass `bus`:
```go
articleHandlerInst := articleHandler.NewHandler(articleRepo, categoryRepo, tagRepo, searchService, bus)
commentHandlerInst := commentHandler.NewHandler(commentRepo, antispamService, bus)
```

Subscribe search index updates (moved from direct calls to event-driven):
```go
bus.Subscribe(eventbus.ContentCreated, eventbus.AsyncHandler(func(e eventbus.Event) {
	// Search index and audit log can react to content events
	log.Info("Content event", "type", e.Type)
}))
```

- [ ] **Step 4: Verify build**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
```

- [ ] **Step 5: Run existing tests**

```bash
cd /home/dev/impress/backend && go test -v -race ./...
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/article/handler.go backend/internal/handler/comment/handler.go backend/cmd/server/main.go
git commit -m "feat(2.3.3): wire EventBus into article and comment handlers for lifecycle events"
```

### Task 17: StorageProvider interface (2.3.4)

**Files:**
- Create: `backend/internal/provider/storage.go`
- Create: `backend/internal/service/local_storage.go`
- Create: `backend/internal/service/local_storage_test.go`

- [ ] **Step 1: Write tests**

Create `backend/internal/service/local_storage_test.go`:

```go
package service_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"blotting-consultancy/internal/service"
)

func TestLocalStorageSaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	storage := service.NewLocalStorage(tmpDir)

	ctx := context.Background()
	content := []byte("hello world")
	path, err := storage.Save(ctx, "test.txt", bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists on disk
	if _, err := os.Stat(filepath.Join(tmpDir, path)); err != nil {
		t.Fatalf("file not found on disk: %v", err)
	}

	// Get the file back
	reader, err := storage.Get(ctx, path)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(got))
	}
}

func TestLocalStorageDelete(t *testing.T) {
	tmpDir := t.TempDir()
	storage := service.NewLocalStorage(tmpDir)

	ctx := context.Background()
	content := []byte("delete me")
	path, _ := storage.Save(ctx, "del.txt", bytes.NewReader(content), int64(len(content)))

	if err := storage.Delete(ctx, path); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, path)); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestLocalStorageURL(t *testing.T) {
	storage := service.NewLocalStorage("/uploads")
	url := storage.URL("images/photo.jpg")
	if url != "/uploads/images/photo.jpg" {
		t.Errorf("unexpected URL: %s", url)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestLocalStorage ./internal/service/
```

- [ ] **Step 3: Create StorageProvider interface**

Create `backend/internal/provider/storage.go`:

```go
package provider

import (
	"context"
	"io"
)

// StorageProvider defines the contract for file storage backends.
// Default implementation uses local filesystem.
// Plugins can replace with S3, MinIO, Alibaba OSS, etc.
type StorageProvider interface {
	// Save stores a file and returns its relative path.
	Save(ctx context.Context, filename string, reader io.Reader, size int64) (string, error)

	// Get retrieves a file by its relative path.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file by its relative path.
	Delete(ctx context.Context, path string) error

	// URL returns the public URL for a stored file.
	URL(path string) string

	// Exists checks whether a file exists.
	Exists(ctx context.Context, path string) (bool, error)
}
```

- [ ] **Step 4: Implement LocalStorage**

Create `backend/internal/service/local_storage.go`:

```go
package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements provider.StorageProvider using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local filesystem storage provider.
func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

// Save stores a file to the local filesystem.
func (s *LocalStorage) Save(ctx context.Context, filename string, reader io.Reader, size int64) (string, error) {
	fullPath := filepath.Join(s.baseDir, filename)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filename, nil
}

// Get retrieves a file from the local filesystem.
func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.baseDir, path)
	return os.Open(fullPath)
}

// Delete removes a file from the local filesystem.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.baseDir, path)
	return os.Remove(fullPath)
}

// URL returns the public URL prefix for a stored file.
func (s *LocalStorage) URL(path string) string {
	return filepath.Join(s.baseDir, path)
}

// Exists checks whether a file exists on the local filesystem.
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.baseDir, path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -race -run TestLocalStorage ./internal/service/
```

Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/provider/storage.go backend/internal/service/local_storage.go backend/internal/service/local_storage_test.go
git commit -m "feat(2.3.4): StorageProvider interface + LocalStorage implementation"
```

### Task 18: LogNotifier — default NotifierProvider implementation (2.3.6)

**Files:**
- Create: `backend/internal/service/log_notifier.go`
- Create: `backend/internal/service/log_notifier_test.go`

- [ ] **Step 1: Write test**

Create `backend/internal/service/log_notifier_test.go`:

```go
package service_test

import (
	"context"
	"testing"

	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/service"
)

func TestLogNotifierImplementsInterface(t *testing.T) {
	var n provider.NotifierProvider = service.NewLogNotifier()
	err := n.Notify(context.Background(), provider.NotifyEvent{
		Type:    "test",
		Subject: "Test Notification",
		Body:    "This is a test",
	})
	if err != nil {
		t.Errorf("Notify should not fail: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestLogNotifier ./internal/service/
```

- [ ] **Step 3: Implement LogNotifier**

Create `backend/internal/service/log_notifier.go`:

```go
package service

import (
	"context"
	"log"

	"blotting-consultancy/internal/provider"
)

// LogNotifier implements provider.NotifierProvider by logging notifications.
// This is the default implementation; plugins can replace with email, Webhook, etc.
type LogNotifier struct{}

// NewLogNotifier creates a new log-based notifier.
func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// Notify logs the notification event.
func (n *LogNotifier) Notify(ctx context.Context, event provider.NotifyEvent) error {
	log.Printf("[NOTIFY] type=%s subject=%q body_len=%d meta=%v",
		event.Type, event.Subject, len(event.Body), event.Meta)
	return nil
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
cd /home/dev/impress/backend && go test -v -race -run TestLogNotifier ./internal/service/
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/log_notifier.go backend/internal/service/log_notifier_test.go
git commit -m "feat(2.3.6): LogNotifier - default NotifierProvider implementation"
```

### Task 19: Provider Registry (2.3.8)

**Files:**
- Create: `backend/internal/provider/registry.go`
- Create: `backend/internal/provider/registry_test.go`

- [ ] **Step 1: Write Registry tests**

Create `backend/internal/provider/registry_test.go`:

```go
package provider_test

import (
	"context"
	"testing"

	"blotting-consultancy/internal/provider"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := provider.NewRegistry()

	captcha := &provider.NoopCaptchaProvider{}
	reg.Register("captcha", captcha)

	got := reg.Get("captcha")
	if got == nil {
		t.Fatal("expected to get captcha provider")
	}
	if _, ok := got.(*provider.NoopCaptchaProvider); !ok {
		t.Error("expected NoopCaptchaProvider type")
	}
}

func TestRegistryReplaceOverwrites(t *testing.T) {
	reg := provider.NewRegistry()

	first := &mockNotifier{name: "first"}
	second := &mockNotifier{name: "second"}

	reg.Register("notifier", first)
	reg.Register("notifier", second) // should overwrite

	got := reg.Get("notifier")
	if got == nil {
		t.Fatal("expected to get notifier provider")
	}
	mn, ok := got.(*mockNotifier)
	if !ok {
		t.Fatal("expected mockNotifier type")
	}
	if mn.name != "second" {
		t.Errorf("expected second provider, got %q", mn.name)
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := provider.NewRegistry()
	if got := reg.Get("nonexistent"); got != nil {
		t.Errorf("expected nil for missing provider, got %v", got)
	}
}

func TestRegistryList(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register("a", &provider.NoopCaptchaProvider{})
	reg.Register("b", &provider.NoopCaptchaProvider{})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}
}

type mockNotifier struct {
	name string
}

func (m *mockNotifier) Notify(ctx context.Context, event provider.NotifyEvent) error {
	return nil
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestRegistry ./internal/provider/
```

- [ ] **Step 3: Implement Registry**

Create `backend/internal/provider/registry.go`:

```go
package provider

import (
	"log"
	"sync"
)

// Registry is a centralized store for Provider instances.
// Same-type providers follow last-registration-wins semantics.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]interface{}
}

// NewRegistry creates a new Provider Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]interface{}),
	}
}

// Register adds or replaces a provider by name.
// If a provider with the same name already exists, it is replaced and a log entry is emitted.
func (r *Registry) Register(name string, provider interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.providers[name]; ok {
		log.Printf("[Registry] replacing provider %q: %T -> %T", name, existing, provider)
	}
	r.providers[name] = provider
	log.Printf("[Registry] registered provider %q (%T)", name, provider)
}

// Get retrieves a provider by name. Returns nil if not found.
func (r *Registry) Get(name string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[name]
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// MustGet retrieves a provider by name and panics if not found.
// Use in startup code where missing providers are fatal.
func (r *Registry) MustGet(name string) interface{} {
	p := r.Get(name)
	if p == nil {
		panic("required provider not registered: " + name)
	}
	return p
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -race -run TestRegistry ./internal/provider/
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/provider/registry.go backend/internal/provider/registry_test.go
git commit -m "feat(2.3.8): Provider Registry with register/get/replace/list and last-wins semantics"
```

### Task 20: Wire Registry and default providers into main.go (2.3.5 + 2.3.6 + 2.3.8)

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Register default providers on startup**

In `backend/cmd/server/main.go`, after initializing the event bus:

```go
// Initialize provider registry with defaults
registry := provider.NewRegistry()
registry.Register("search", searchService)              // SearchProvider (2.3.5)
registry.Register("notifier", service.NewLogNotifier())  // NotifierProvider (2.3.6)
registry.Register("captcha", captchaProvider)            // CaptchaProvider
registry.Register("storage", service.NewLocalStorage(cfg.UploadDir)) // StorageProvider (2.3.4)
log.Info("Provider registry initialized",
	"providers", registry.List(),
)
```

Add necessary import for `service` package if not already present.

- [ ] **Step 2: Verify build and all tests**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
cd /home/dev/impress/backend && go test -v -race ./...
```

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(2.3.5+2.3.6+2.3.8): wire Provider Registry with default search, notifier, captcha, storage providers"
```

### Task 21: Hook point definitions (2.3.7)

**Files:**
- Create: `backend/internal/eventbus/hooks.go`
- Create: `backend/internal/eventbus/hooks_test.go`

- [ ] **Step 1: Write hook tests**

Create `backend/internal/eventbus/hooks_test.go`:

```go
package eventbus_test

import (
	"context"
	"testing"

	"blotting-consultancy/internal/eventbus"
)

func TestHookChainExecutesInOrder(t *testing.T) {
	chain := eventbus.NewHookChain()
	order := []int{}

	chain.Add("first", func(ctx context.Context, data interface{}) (interface{}, error) {
		order = append(order, 1)
		return data, nil
	})
	chain.Add("second", func(ctx context.Context, data interface{}) (interface{}, error) {
		order = append(order, 2)
		return data, nil
	})

	_, err := chain.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected [1,2], got %v", order)
	}
}

func TestHookChainAbortOnError(t *testing.T) {
	chain := eventbus.NewHookChain()

	chain.Add("fail", func(ctx context.Context, data interface{}) (interface{}, error) {
		return nil, context.Canceled
	})
	chain.Add("never", func(ctx context.Context, data interface{}) (interface{}, error) {
		t.Error("this hook should not be called")
		return data, nil
	})

	_, err := chain.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error from failed hook")
	}
}

func TestHookChainDataTransform(t *testing.T) {
	chain := eventbus.NewHookChain()

	chain.Add("transform", func(ctx context.Context, data interface{}) (interface{}, error) {
		s := data.(string)
		return s + " modified", nil
	})

	result, err := chain.Execute(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "input modified" {
		t.Errorf("expected 'input modified', got %v", result)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestHookChain ./internal/eventbus/
```

- [ ] **Step 3: Implement Hook system**

Create `backend/internal/eventbus/hooks.go`:

```go
package eventbus

import (
	"context"
)

// Hook point names for request-level interception.
const (
	HookBeforePublish = "hook.before_publish"
	HookAfterPublish  = "hook.after_publish"
	HookBeforeRender  = "hook.before_render"
	HookBeforeCreate  = "hook.before_create"
	HookAfterCreate   = "hook.after_create"
	HookBeforeDelete  = "hook.before_delete"
	HookAfterDelete   = "hook.after_delete"
)

// HookFunc is a function that can inspect and transform data at a hook point.
// It receives context and the current data, and returns potentially modified data.
// Returning an error aborts the chain.
type HookFunc func(ctx context.Context, data interface{}) (interface{}, error)

// HookEntry pairs a name with a hook function for debugging/logging.
type HookEntry struct {
	Name string
	Fn   HookFunc
}

// HookChain manages an ordered list of hooks for a specific hook point.
type HookChain struct {
	hooks []HookEntry
}

// NewHookChain creates an empty hook chain.
func NewHookChain() *HookChain {
	return &HookChain{}
}

// Add appends a named hook to the chain.
func (c *HookChain) Add(name string, fn HookFunc) {
	c.hooks = append(c.hooks, HookEntry{Name: name, Fn: fn})
}

// Execute runs all hooks in order, threading data through each.
// Stops and returns on the first error.
func (c *HookChain) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	var err error
	for _, h := range c.hooks {
		data, err = h.Fn(ctx, data)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

// Len returns the number of hooks in the chain.
func (c *HookChain) Len() int {
	return len(c.hooks)
}

// HookRegistry manages hook chains for multiple hook points.
type HookRegistry struct {
	chains map[string]*HookChain
}

// NewHookRegistry creates a new HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{chains: make(map[string]*HookChain)}
}

// Register adds a hook function to a named hook point.
func (r *HookRegistry) Register(hookPoint string, name string, fn HookFunc) {
	if _, ok := r.chains[hookPoint]; !ok {
		r.chains[hookPoint] = NewHookChain()
	}
	r.chains[hookPoint].Add(name, fn)
}

// Execute runs the hook chain for a given hook point.
// Returns the original data unchanged if no hooks are registered.
func (r *HookRegistry) Execute(ctx context.Context, hookPoint string, data interface{}) (interface{}, error) {
	chain, ok := r.chains[hookPoint]
	if !ok {
		return data, nil
	}
	return chain.Execute(ctx, data)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -race ./internal/eventbus/
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/eventbus/hooks.go backend/internal/eventbus/hooks_test.go
git commit -m "feat(2.3.7): Hook point definitions with HookChain and HookRegistry"
```

---

## Chunk 4: Community Infrastructure (Tasks 2.4.1 — 2.4.6)

### Task 22: CONTRIBUTING.md (2.4.1)

**Files:**
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: Create CONTRIBUTING.md**

Create `CONTRIBUTING.md` at the repo root with:
- Development environment setup (prerequisites: Go 1.24+, Node 20+, pnpm 9+)
- How to run the project locally (`make dev`)
- Backend development workflow (Go test, vet, build)
- Frontend development workflow (pnpm dev, lint, type-check, test)
- Code style guidelines (Go: `go fmt`; TypeScript: ESLint + Prettier; Tailwind utility-first)
- Database migrations (goose, how to create new migrations)
- PR submission guidelines (branch naming, commit format, required checks)
- Architecture overview pointer (link to docs/architecture.md)
- Extension point development (link to docs/guide/extension-points.md)

Keep it concise and actionable. Target audience: first-time contributor who has never seen the codebase.

- [ ] **Step 2: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs(2.4.1): add CONTRIBUTING.md with dev setup, code style, and PR guidelines"
```

### Task 23: GitHub issue and PR templates (2.4.2)

**Files:**
- Create: `.github/ISSUE_TEMPLATE/bug_report.md`
- Create: `.github/ISSUE_TEMPLATE/feature_request.md`
- Create: `.github/ISSUE_TEMPLATE/plugin_request.md`
- Create: `.github/pull_request_template.md`

- [ ] **Step 1: Create bug report template**

Create `.github/ISSUE_TEMPLATE/bug_report.md`:

```markdown
---
name: Bug Report
about: Report a bug in Impress CMS
title: "[Bug] "
labels: bug
assignees: ''
---

## Description
A clear description of the bug.

## Steps to Reproduce
1. Go to '...'
2. Click on '...'
3. See error

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- **OS:** [e.g., Ubuntu 22.04, macOS 14]
- **Go version:** [e.g., 1.24]
- **Node version:** [e.g., 20]
- **Database:** [SQLite / PostgreSQL]
- **Browser:** [e.g., Chrome 120]

## Screenshots / Logs
If applicable, add screenshots or relevant log output.
```

- [ ] **Step 2: Create feature request template**

Create `.github/ISSUE_TEMPLATE/feature_request.md`:

```markdown
---
name: Feature Request
about: Suggest a new feature for Impress CMS
title: "[Feature] "
labels: enhancement
assignees: ''
---

## Problem
What problem does this feature solve?

## Proposed Solution
Describe your ideal solution.

## Alternatives Considered
Any alternative approaches you've thought about.

## Additional Context
Any mockups, references, or related issues.
```

- [ ] **Step 3: Create plugin request template**

Create `.github/ISSUE_TEMPLATE/plugin_request.md`:

```markdown
---
name: Plugin Request
about: Request a new plugin or extension point
title: "[Plugin] "
labels: plugin
assignees: ''
---

## Plugin Description
What should this plugin do?

## Use Case
Who would use this plugin and why?

## Provider Interface
Which provider interface would this implement? (SearchProvider, NotifierProvider, StorageProvider, CaptchaProvider, or new)

## Configuration
What settings would the plugin need?
```

- [ ] **Step 4: Create PR template**

Create `.github/pull_request_template.md`:

```markdown
## Summary
Brief description of changes.

## Changes
- [ ] Change 1
- [ ] Change 2

## Type
- [ ] Bug fix
- [ ] New feature
- [ ] Enhancement
- [ ] Refactoring
- [ ] Documentation
- [ ] Plugin / Extension

## Testing
- [ ] Added/updated unit tests
- [ ] `go test -race ./...` passes
- [ ] `pnpm lint && pnpm type-check` passes
- [ ] Manual testing done

## Related Issues
Closes #

## Screenshots (if UI changes)
```

- [ ] **Step 5: Commit**

```bash
git add .github/ISSUE_TEMPLATE/ .github/pull_request_template.md
git commit -m "docs(2.4.2): add GitHub issue templates (bug/feature/plugin) and PR template"
```

### Task 24: VitePress documentation site scaffold (2.4.3)

**Files:**
- Create: `docs/site/package.json`
- Create: `docs/site/.vitepress/config.ts`
- Create: `docs/site/index.md`
- Create: `docs/site/guide/getting-started.md`

- [ ] **Step 1: Create docs site package.json**

Create `docs/site/package.json`:

```json
{
  "name": "impress-docs",
  "private": true,
  "scripts": {
    "dev": "vitepress dev",
    "build": "vitepress build",
    "preview": "vitepress preview"
  },
  "devDependencies": {
    "vitepress": "^1.5.0"
  }
}
```

- [ ] **Step 2: Create VitePress config**

Create `docs/site/.vitepress/config.ts`:

```typescript
import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Impress CMS",
  description: "A bilingual CMS built with Go and React",
  lang: "en-US",

  themeConfig: {
    nav: [
      { text: "Guide", link: "/guide/getting-started" },
      { text: "API", link: "/api/" },
      { text: "GitHub", link: "https://github.com/your-org/impress" },
    ],

    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Getting Started", link: "/guide/getting-started" },
          { text: "Architecture", link: "/guide/architecture" },
          { text: "Extension Points", link: "/guide/extension-points" },
          { text: "Your First Plugin", link: "/guide/first-plugin" },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/your-org/impress" },
    ],
  },
});
```

- [ ] **Step 3: Create landing page**

Create `docs/site/index.md`:

```markdown
---
layout: home

hero:
  name: Impress CMS
  text: Bilingual CMS for the modern web
  tagline: A self-hosted, extensible content management system built with Go and React
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/your-org/impress

features:
  - title: Bilingual by Default
    details: Built-in Chinese/English content management with locale-aware rendering
  - title: Extensible Architecture
    details: Provider interfaces for search, storage, notifications, and CAPTCHA
  - title: Developer Friendly
    details: CLI tooling, Swagger API docs, EventBus, and clear extension points
  - title: Self-Hosted
    details: Single binary deployment with SQLite or PostgreSQL
---
```

- [ ] **Step 4: Create getting started guide**

Create `docs/site/guide/getting-started.md`:

```markdown
# Getting Started

## Prerequisites

- **Go** 1.24+
- **Node.js** 20+ with **pnpm** 9+
- **Git**

## Quick Start

### 1. Clone the repository

\`\`\`bash
git clone https://github.com/your-org/impress.git
cd impress
\`\`\`

### 2. Install dependencies

\`\`\`bash
pnpm install
\`\`\`

### 3. Initialize the project

\`\`\`bash
cd backend
go build -o impress ./cmd/impress/
./impress init
\`\`\`

### 4. Start the development server

\`\`\`bash
make dev
\`\`\`

This starts:
- Backend API at `http://localhost:8088`
- Frontend dev server at `http://localhost:3000`

### 5. Access the admin panel

Open `http://localhost:3000/admin` and log in with the default credentials:
- Username: `admin`
- Password: `admin123`

## CLI Commands

| Command | Description |
|---------|-------------|
| `impress init` | Initialize project configuration |
| `impress serve` | Start the server |
| `impress migrate up` | Run pending database migrations |
| `impress migrate down` | Rollback the last migration |
| `impress migrate status` | Show migration status |
| `impress seed` | Populate with sample data |
| `impress export` | Export site data to JSON |
| `impress import <file>` | Import site data from JSON |

## Project Structure

\`\`\`
impress/
├── backend/           # Go/Gin/GORM backend
│   ├── cmd/server/    # Main server binary
│   ├── cmd/impress/   # CLI tool
│   └── internal/      # Application packages
├── frontend/          # React/Vite/Tailwind frontend
├── docs/              # Documentation
└── Makefile           # Build automation
\`\`\`

## Next Steps

- [Architecture Overview](/guide/architecture) — understand the system design
- [Extension Points](/guide/extension-points) — learn about Provider interfaces
- [Your First Plugin](/guide/first-plugin) — build a simple plugin
```

- [ ] **Step 5: Create placeholder pages**

Create `docs/site/guide/architecture.md`:
```markdown
# Architecture Overview

> This page is under construction. See `docs/architecture.md` in the repository for the current architecture documentation.
```

Create `docs/site/guide/extension-points.md`:
```markdown
# Extension Points

> This page is under construction. It will document Provider interfaces, EventBus, and Hook points.
```

Create `docs/site/guide/first-plugin.md`:
```markdown
# Your First Plugin

> This tutorial will be available after Phase 3 (Plugin Architecture) is complete.
```

- [ ] **Step 6: Verify VitePress builds**

```bash
cd /home/dev/impress/docs/site && pnpm install && pnpm build
```

Note: This step may fail if pnpm is not available in the docs/site context. The scaffold is sufficient — actual build verification can be done when the docs site is deployed.

- [ ] **Step 7: Commit**

```bash
git add docs/site/
git commit -m "docs(2.4.3): VitePress documentation site scaffold with getting-started guide"
```

### Task 25: Enhanced CI/CD — coverage report + swagger check (2.4.6)

**Files:**
- Modify: `.github/workflows/quality-gate.yml`

- [ ] **Step 1: Add coverage reporting and swagger check**

In `.github/workflows/quality-gate.yml`, enhance the `backend-checks` job:

After the "Run Go tests" step, add:
```yaml
      - name: Display test coverage summary
        if: always()
        run: |
          cd backend
          go tool cover -func=coverage.out | tail -1
```

In the `frontend-checks` job, after "Run frontend tests", add:
```yaml
      - name: Display test coverage summary
        if: always()
        run: |
          echo "Frontend test coverage report:"
          if [ -f frontend/coverage/coverage-summary.json ]; then
            cat frontend/coverage/coverage-summary.json
          else
            echo "No coverage report generated"
          fi
```

Add CLI build verification to `integration-smoke` job:
```yaml
      - name: Build CLI
        run: cd backend && go build -v -o ../impress ./cmd/impress

      - name: Verify CLI binary
        run: |
          ./impress --version
          ./impress --help
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/quality-gate.yml
git commit -m "feat(2.4.6): enhance CI with coverage reporting, swagger check, and CLI build verification"
```

### Task 26: Architecture documentation for developers (2.4.4)

**Files:**
- Create: `docs/developer-guide.md`

- [ ] **Step 1: Create developer-facing architecture document**

Create `docs/developer-guide.md` covering:

1. **System Architecture** — High-level diagram (backend layers, frontend layers, data flow)
2. **Backend Layers** — cmd/ -> handler/ -> service/ -> repository/ -> model/ with examples
3. **Provider Pattern** — How SearchProvider, NotifierProvider, StorageProvider, CaptchaProvider work; how to implement a new one
4. **EventBus** — Event types, sync vs async subscribers, how to subscribe to events
5. **Hook Points** — Available hooks, HookChain execution model, how to register hooks
6. **Provider Registry** — How providers are registered and resolved at startup
7. **Database** — GORM model conventions, goose migrations, SQLite vs PostgreSQL differences
8. **Frontend Architecture** — Provider hierarchy, routing, theme system, i18n
9. **Adding a New Feature** — Step-by-step checklist (model -> migration -> repo -> service -> handler -> route -> test)

- [ ] **Step 2: Commit**

```bash
git add docs/developer-guide.md
git commit -m "docs(2.4.4): developer-facing architecture guide with provider pattern and extension points"
```

---

## Verification

After completing all tasks, run the full verification suite:

```bash
# Backend
cd /home/dev/impress/backend && go vet ./...
cd /home/dev/impress/backend && go test -v -race ./...
cd /home/dev/impress/backend && go build -o server ./cmd/server/
cd /home/dev/impress/backend && go build -o impress ./cmd/impress/

# Frontend (no changes in Phase 2, but verify nothing broke)
cd /home/dev/impress && pnpm lint && pnpm type-check

# CLI smoke test
cd /home/dev/impress/backend && ./impress --help
cd /home/dev/impress/backend && ./impress migrate --help
cd /home/dev/impress/backend && ./impress init --non-interactive -o /tmp/impress-verify && rm -rf /tmp/impress-verify

# Swagger docs
cd /home/dev/impress/backend && swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
```

## Task Summary

| # | Task | Spec ID | Chunk |
|---|------|---------|-------|
| 1 | Swagger deps + annotations | 2.1.1 | 1 |
| 2 | Swagger UI route | 2.1.3 | 1 |
| 3 | Annotate auth handler | 2.1.2 | 1 |
| 4 | Annotate article handler | 2.1.2 | 1 |
| 5 | Annotate remaining handlers | 2.1.2 | 1 |
| 6 | CI swagger generation | 2.1.4 | 1 |
| 7 | CLI scaffold (cobra) | 2.2.1 | 2 |
| 8 | `impress init` | 2.2.2 | 2 |
| 9 | `impress serve` | 2.2.3 | 2 |
| 10 | `impress migrate` | 2.2.4 | 2 |
| 11 | `impress seed` | 2.2.5 | 2 |
| 12 | `impress export/import` | 2.2.7 | 2 |
| 13 | CLI Makefile target | 2.2.1 | 2 |
| 14 | EventBus interface + impl | 2.3.1 | 3 |
| 15 | Content lifecycle events | 2.3.2 | 3 |
| 16 | Wire EventBus into handlers | 2.3.3 | 3 |
| 17 | StorageProvider + LocalStorage | 2.3.4 | 3 |
| 18 | LogNotifier (default Notifier) | 2.3.6 | 3 |
| 19 | Provider Registry | 2.3.8 | 3 |
| 20 | Wire Registry into main.go | 2.3.5+2.3.6+2.3.8 | 3 |
| 21 | Hook point definitions | 2.3.7 | 3 |
| 22 | CONTRIBUTING.md | 2.4.1 | 4 |
| 23 | Issue + PR templates | 2.4.2 | 4 |
| 24 | VitePress docs site | 2.4.3 | 4 |
| 25 | Enhanced CI/CD | 2.4.6 | 4 |
| 26 | Developer architecture docs | 2.4.4 | 4 |
