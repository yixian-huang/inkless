# Phase 1: Basic Enhancements Implementation Plan

> 历史实施计划：本文保留早期“为未来 multi-site 预留 site_id”的注释用于追溯，该方向已由 2026-07-18 的单实例单站点 ADR 否决，不得作为当前实现要求。参见 `docs/adr/0001-single-instance-single-site.md`。

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full-text search, comments, scheduled publishing, Markdown editor, SEO enhancements, and misc improvements to make the product competitive with Halo for self-hosted users.

**Architecture:** Each subsystem follows existing patterns: GORM model → repository interface + impl → handler → frontend API module → React page/component. New tables use goose migrations (from Phase 0). SEO meta injection uses the Phase 0 renderer with route-specific meta resolution.

**Tech Stack:** Go/Gin/GORM (backend), React 19/TypeScript/TipTap (frontend), SQLite FTS5 / PostgreSQL tsvector (search), time.Ticker (scheduling)

**Spec:** `docs/superpowers/specs/2026-03-11-open-source-evolution-design.md` — Phase 1

**Prerequisites:** Phase 0 MUST be complete before starting Phase 1. Phase 0 provides: goose migration tooling, `backend/internal/seo/meta.go` (PageMeta struct + DefaultPageMeta), SPA meta renderer, testing strategy, API versioning.

**Convention note:** New handlers use `RegisterRoutes(public, admin)` method for route setup. This is a deliberate improvement over the existing centralized registration in main.go. Existing handlers will be migrated to this pattern in future refactors.

---

## File Structure Overview

```
backend/
├── internal/db/migrations/
│   ├── 00002_add_seo_fields.sql           (1.1 SEO)
│   ├── 00003_create_search_index.sql      (1.2 Search)
│   ├── 00004_create_comments.sql          (1.3 Comments)
│   ├── 00005_add_scheduled_fields.sql     (1.4 Scheduling)
├── internal/model/
│   ├── comment.go                         (create)
│   ├── search_index.go                    (create)
│   └── article.go                         (modify - add ScheduledAt)
│   └── page.go                            (modify - add ScheduledAt)
├── internal/repository/
│   ├── comment_repository.go              (create)
│   ├── comment_repository_impl.go         (create)
│   (search uses raw SQL for FTS5 — no repository layer, handled in SearchService)
├── internal/handler/
│   ├── comment/handler.go                 (create)
│   ├── search/handler.go                 (create)
│   ├── seo/handler.go                    (create)
├── internal/service/
│   ├── search_service.go                 (create)
│   ├── comment_service.go               (create)
│   ├── scheduler_service.go             (create)
│   ├── antispam_service.go              (create)
├── internal/seo/
│   ├── resolver.go                       (create - route-to-meta resolver)
│   ├── resolver_test.go                  (create)
│   ├── jsonld.go                         (create - Schema.org JSON-LD)
│   ├── jsonld_test.go                    (create)
├── internal/provider/
│   ├── search.go                         (create - SearchProvider interface)
│   ├── notifier.go                       (create - Notifier interface)
│   ├── captcha.go                        (create - CaptchaProvider interface)
frontend/
├── src/api/
│   ├── search.ts                         (create)
│   ├── comments.ts                       (create)
├── src/pages/
│   ├── search/page.tsx                   (create - search results page)
├── src/components/
│   ├── feature/SearchBox.tsx             (create - global search)
│   ├── feature/CommentSection.tsx        (create - article comments)
│   ├── admin/SchedulePublishModal.tsx    (create)
│   ├── admin/ScheduleDashboard.tsx       (create)
│   ├── admin/RobotsTxtEditor.tsx         (create)
│   ├── admin/SeoFieldGroup.tsx           (create - reusable SEO form fields)
│   ├── admin/editor/MarkdownMode.tsx     (create)
│   ├── admin/editor/EditorModeSwitcher.tsx (create)
│   └── admin/comments/                   (create - admin comments management)
├── src/hooks/
│   └── useSearch.ts                      (create)
```

---

## Chunk 1: SEO Enhancements (Tasks 1.1.1 — 1.1.6)

### Task 1: SEO fields in article/page models (1.1.1)

**Files:**
- Create: `backend/internal/db/migrations/00002_add_seo_fields.sql`
- Modify: `backend/internal/model/article.go` — verify existing SEO fields
- Modify: `backend/internal/model/page.go` — verify existing SEO fields

- [ ] **Step 1: Check existing SEO fields**

The Article model already has: `ZhSeoTitle`, `EnSeoTitle`, `ZhMetaDescription`, `EnMetaDescription`, `OgImage`. The Page model already has `SeoTitle` (JSONMap), `SeoDescription` (JSONMap). These are sufficient for basic per-page/per-article meta. No schema migration needed for 1.1.1 — the fields exist.

Verify by reading models:
```bash
cd /home/dev/impress/backend && grep -n "Seo\|MetaDescription\|OgImage\|Keywords" internal/model/article.go internal/model/page.go
```

- [ ] **Step 2: Add Keywords field to Page model if missing**

If Page model lacks a `Keywords` field (JSONMap for zh/en keywords), add it:

In `backend/internal/model/page.go`, add:
```go
Keywords JSONMap `json:"keywords" gorm:"type:text"`
```

Create migration `backend/internal/db/migrations/00002_add_seo_fields.sql`:
```sql
-- +goose Up
-- Add keywords field to pages table if not exists
ALTER TABLE pages ADD COLUMN IF NOT EXISTS keywords TEXT DEFAULT '{}';

-- +goose StatementBegin
-- For SQLite which doesn't support IF NOT EXISTS on ALTER TABLE:
-- This will be a no-op if column already exists (goose tracks by migration ID)
-- +goose StatementEnd

-- +goose Down
-- SQLite doesn't support DROP COLUMN before 3.35.0, so this is best-effort
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/page.go backend/internal/db/migrations/00002_add_seo_fields.sql
git commit -m "feat(1.1.1): add keywords field to page model with migration"
```

### Task 2: SEO meta resolver — route to meta mapping (1.1.1 + 1.1.2 + 1.1.6)

**Files:**
- Create: `backend/internal/seo/resolver.go`
- Create: `backend/internal/seo/resolver_test.go`

- [ ] **Step 1: Write resolver tests**

Create `backend/internal/seo/resolver_test.go`:

```go
package seo_test

import (
	"testing"

	"blotting-consultancy/internal/seo"
)

func TestResolveHomePage(t *testing.T) {
	meta := seo.ResolveFromPath("/", "https://example.com", "zh")
	if meta.CanonicalURL != "https://example.com/" {
		t.Errorf("expected canonical /, got %q", meta.CanonicalURL)
	}
	if meta.OgURL != "https://example.com/" {
		t.Errorf("expected og:url /, got %q", meta.OgURL)
	}
	if meta.Locale != "zh" {
		t.Errorf("expected locale zh, got %q", meta.Locale)
	}
}

func TestResolveAboutPage(t *testing.T) {
	meta := seo.ResolveFromPath("/about", "https://example.com", "en")
	if meta.CanonicalURL != "https://example.com/about" {
		t.Errorf("expected canonical /about, got %q", meta.CanonicalURL)
	}
	if meta.Locale != "en" {
		t.Errorf("expected locale en, got %q", meta.Locale)
	}
}

func TestResolveArticlePath(t *testing.T) {
	meta := seo.ResolveFromPath("/blog/my-article", "https://example.com", "zh")
	if meta.CanonicalURL != "https://example.com/blog/my-article" {
		t.Errorf("expected canonical, got %q", meta.CanonicalURL)
	}
	if meta.OgType != "article" {
		t.Errorf("expected og:type article for blog path, got %q", meta.OgType)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /home/dev/impress/backend && go test -v -run TestResolve ./internal/seo/
```

Expected: FAIL — ResolveFromPath not found.

- [ ] **Step 3: Implement resolver**

Create `backend/internal/seo/resolver.go`:

```go
package seo

import (
	"strings"
)

// ResolveFromPath returns PageMeta with canonical URL, og:url, og:type set based on the request path.
// This provides baseline meta for any page. Route-specific meta (article title, page description)
// should override these fields after fetching from the database.
func ResolveFromPath(path, baseURL, locale string) PageMeta {
	meta := DefaultPageMeta()
	meta.Locale = locale

	canonical := strings.TrimRight(baseURL, "/") + path
	meta.CanonicalURL = canonical
	meta.OgURL = canonical

	// Determine og:type based on path pattern
	if strings.HasPrefix(path, "/blog/") && path != "/blog/" {
		meta.OgType = "article"
	}

	return meta
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /home/dev/impress/backend && go test -v -run TestResolve ./internal/seo/
```

Expected: All PASS

- [ ] **Step 5: Wire resolver into SPA fallback**

Modify `backend/cmd/server/main.go` SPA fallback: replace `seo.DefaultPageMeta()` with `seo.ResolveFromPath(c.Request.URL.Path, cfg.BaseURL, locale)`.

Determine locale from query param or Accept-Language header:
```go
locale := c.DefaultQuery("locale", "zh")
if locale != "zh" && locale != "en" {
    locale = "zh"
}
meta := seo.ResolveFromPath(c.Request.URL.Path, cfg.BaseURL, locale)
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/seo/resolver.go backend/internal/seo/resolver_test.go backend/cmd/server/main.go
git commit -m "feat(1.1.1): SEO meta resolver with canonical URL and og:type by route"
```

### Task 3: Schema.org JSON-LD output (1.1.3)

**Files:**
- Create: `backend/internal/seo/jsonld.go`
- Create: `backend/internal/seo/jsonld_test.go`
- Modify: `backend/internal/seo/meta.go` (add JSONLD field)
- Modify: `frontend/index.html` (add JSON-LD template slot)

- [ ] **Step 1: Write JSON-LD test**

Create `backend/internal/seo/jsonld_test.go`:

```go
package seo_test

import (
	"encoding/json"
	"testing"

	"blotting-consultancy/internal/seo"
)

func TestOrganizationJSONLD(t *testing.T) {
	ld := seo.OrganizationJSONLD("印迹法规咨询", "https://example.com", "https://example.com/logo.png")
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(ld), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["@type"] != "Organization" {
		t.Errorf("expected @type Organization, got %v", m["@type"])
	}
	if m["name"] != "印迹法规咨询" {
		t.Errorf("expected name, got %v", m["name"])
	}
}

func TestArticleJSONLD(t *testing.T) {
	ld := seo.ArticleJSONLD("Title", "Desc", "https://example.com/blog/test", "https://example.com/img.png", "2026-01-01", "Author")
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(ld), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["@type"] != "Article" {
		t.Errorf("expected @type Article, got %v", m["@type"])
	}
}

func TestBreadcrumbJSONLD(t *testing.T) {
	items := []seo.BreadcrumbItem{
		{Name: "Home", URL: "https://example.com/"},
		{Name: "Blog", URL: "https://example.com/blog"},
		{Name: "Article", URL: "https://example.com/blog/test"},
	}
	ld := seo.BreadcrumbJSONLD(items)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(ld), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["@type"] != "BreadcrumbList" {
		t.Errorf("expected @type BreadcrumbList, got %v", m["@type"])
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestOrganizationJSONLD ./internal/seo/
```

Expected: FAIL

- [ ] **Step 3: Implement JSON-LD generators**

Create `backend/internal/seo/jsonld.go`:

```go
package seo

import (
	"encoding/json"
	"fmt"
)

type BreadcrumbItem struct {
	Name string
	URL  string
}

func OrganizationJSONLD(name, url, logoURL string) string {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     name,
		"url":      url,
		"logo":     logoURL,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func ArticleJSONLD(title, description, url, image, datePublished, author string) string {
	data := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "Article",
		"headline":      title,
		"description":   description,
		"url":           url,
		"image":         image,
		"datePublished": datePublished,
		"author": map[string]interface{}{
			"@type": "Person",
			"name":  author,
		},
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func BreadcrumbJSONLD(items []BreadcrumbItem) string {
	listItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		listItems[i] = map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     item.Name,
			"item":     item.URL,
		}
	}
	data := map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": listItems,
	}
	b, _ := json.Marshal(data)
	return string(b)
}
```

- [ ] **Step 4: Add JSONLD field to PageMeta**

In `backend/internal/seo/meta.go`, add to PageMeta struct:

```go
JSONLD string // JSON-LD script content (rendered as <script type="application/ld+json">)
```

- [ ] **Step 5: Add JSON-LD template slot to index.html**

In `frontend/index.html`, add inside `<head>` before closing `</head>`:

```html
{{if .JSONLD}}<script type="application/ld+json">{{.JSONLD}}</script>{{end}}
```

Note: Use `template.HTML` type or mark JSONLD as safe in the template to avoid double-escaping. Update `renderer.go` to use `template.HTML` for JSONLD field, or use a custom template function.

- [ ] **Step 6: Run tests**

```bash
cd /home/dev/impress/backend && go test -v ./internal/seo/
```

Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/seo/jsonld.go backend/internal/seo/jsonld_test.go backend/internal/seo/meta.go frontend/index.html
git commit -m "feat(1.1.3): Schema.org JSON-LD generators for Organization, Article, BreadcrumbList"
```

### Task 4: Enhanced sitemap with hreflang (1.1.4)

**Files:**
- Modify: `backend/internal/handler/sitemap/handler.go`

- [ ] **Step 1: Write test for enhanced sitemap**

Create `backend/internal/handler/sitemap/handler_test.go`:

```go
package sitemap_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	// Import handler and mock dependencies as needed
)

func TestSitemapContainsHreflang(t *testing.T) {
	// Test that sitemap XML output contains xhtml:link hreflang="zh" and hreflang="en"
	// and includes lastmod timestamps
	// This is an integration test outline - implement with actual handler setup
	t.Skip("Implement with mock repositories")
}
```

- [ ] **Step 2: Enhance sitemap handler**

In `backend/internal/handler/sitemap/handler.go`, modify `GetSitemap`:

1. Add `<lastmod>` from `doc.UpdatedAt.Format("2006-01-02")`
2. Add `<changefreq>` and `<priority>` based on page type (home=1.0/daily, articles=0.8/weekly, static pages=0.6/monthly)
3. Add articles from ArticleRepository to sitemap (currently only content docs are included)
4. Ensure xhtml:link alternates already present are correct

- [ ] **Step 3: Verify sitemap output**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/ && echo "Build OK"
# Manual: start server, curl http://localhost:8088/sitemap.xml and verify output
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/sitemap/
git commit -m "feat(1.1.4): enhanced sitemap with lastmod, changefreq, priority, and article URLs"
```

### Task 5: Robots.txt management (1.1.5)

**Files:**
- Create: `backend/internal/handler/seo/handler.go`
- Modify: `backend/cmd/server/main.go` (add route)

- [ ] **Step 1: Create SEO handler with robots.txt endpoint**

Create `backend/internal/handler/seo/handler.go`:

```go
package seo

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	mu       sync.RWMutex
	robotsTxt string
}

func NewHandler() *Handler {
	return &Handler{
		robotsTxt: defaultRobotsTxt(),
	}
}

func defaultRobotsTxt() string {
	return "User-agent: *\nAllow: /\n\nSitemap: /sitemap.xml\n"
}

// GetRobotsTxt serves robots.txt as plain text.
func (h *Handler) GetRobotsTxt(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(h.robotsTxt))
}

// AdminGetRobotsTxt returns current robots.txt content for admin editing.
func (h *Handler) AdminGetRobotsTxt(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"content": h.robotsTxt})
}

// AdminUpdateRobotsTxt updates robots.txt content.
func (h *Handler) AdminUpdateRobotsTxt(c *gin.Context) {
	var input struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}
	h.mu.Lock()
	h.robotsTxt = input.Content
	h.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"content": input.Content})
}

func (h *Handler) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.GET("/robots.txt", h.GetRobotsTxt)
	admin.GET("/seo/robots", h.AdminGetRobotsTxt)
	admin.PUT("/seo/robots", h.AdminUpdateRobotsTxt)
}
```

Note: For persistence across restarts, store in database (content_documents with page_key="robots_txt") or a config file. For now, in-memory with default is sufficient — persistence can be added when the admin UI is built.

- [ ] **Step 2: Register routes in main.go**

```go
seoHandler := seohandler.NewHandler()
seoHandler.RegisterRoutes(publicGroup, adminGroup)
```

Also add `GET /robots.txt` at the root level (before SPA fallback):
```go
r.GET("/robots.txt", seoHandler.GetRobotsTxt)
```

- [ ] **Step 3: Verify build and test manually**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/seo/ backend/cmd/server/main.go
git commit -m "feat(1.1.5): robots.txt management with admin GET/PUT endpoints"
```

### Task 6: Admin SEO form fields component (1.1.1 frontend)

**Files:**
- Create: `frontend/src/components/admin/SeoFieldGroup.tsx`

- [ ] **Step 1: Create reusable SEO field group component**

Create `frontend/src/components/admin/SeoFieldGroup.tsx`:

```tsx
interface SeoFieldGroupProps {
  seoTitle: string;
  onSeoTitleChange: (value: string) => void;
  metaDescription: string;
  onMetaDescriptionChange: (value: string) => void;
  ogImage?: string;
  onOgImageChange?: (value: string) => void;
  keywords?: string;
  onKeywordsChange?: (value: string) => void;
  label?: string;
}

export default function SeoFieldGroup({
  seoTitle,
  onSeoTitleChange,
  metaDescription,
  onMetaDescriptionChange,
  ogImage,
  onOgImageChange,
  keywords,
  onKeywordsChange,
  label = "SEO",
}: SeoFieldGroupProps) {
  return (
    <div className="space-y-3">
      <h4 className="text-sm font-medium text-gray-700">{label}</h4>
      <div>
        <label className="block text-xs text-gray-500 mb-1">SEO Title</label>
        <input
          type="text"
          value={seoTitle}
          onChange={(e) => onSeoTitleChange(e.target.value)}
          className="w-full border rounded px-3 py-1.5 text-sm"
          placeholder="Override page title for search engines"
          maxLength={70}
        />
        <span className="text-xs text-gray-400">{seoTitle.length}/70</span>
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">Meta Description</label>
        <textarea
          value={metaDescription}
          onChange={(e) => onMetaDescriptionChange(e.target.value)}
          className="w-full border rounded px-3 py-1.5 text-sm"
          rows={2}
          placeholder="Description for search engine results"
          maxLength={160}
        />
        <span className="text-xs text-gray-400">{metaDescription.length}/160</span>
      </div>
      {onKeywordsChange && (
        <div>
          <label className="block text-xs text-gray-500 mb-1">Keywords</label>
          <input
            type="text"
            value={keywords ?? ""}
            onChange={(e) => onKeywordsChange(e.target.value)}
            className="w-full border rounded px-3 py-1.5 text-sm"
            placeholder="Comma-separated keywords"
          />
        </div>
      )}
      {onOgImageChange && (
        <div>
          <label className="block text-xs text-gray-500 mb-1">OG Image URL</label>
          <input
            type="text"
            value={ogImage ?? ""}
            onChange={(e) => onOgImageChange(e.target.value)}
            className="w-full border rounded px-3 py-1.5 text-sm"
            placeholder="Image for social sharing preview"
          />
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Write render test**

Create `frontend/src/components/admin/SeoFieldGroup.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import SeoFieldGroup from "./SeoFieldGroup";

describe("SeoFieldGroup", () => {
  it("renders SEO fields", () => {
    render(
      <SeoFieldGroup
        seoTitle="Test Title"
        onSeoTitleChange={vi.fn()}
        metaDescription="Test desc"
        onMetaDescriptionChange={vi.fn()}
      />
    );
    expect(screen.getByDisplayValue("Test Title")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Test desc")).toBeInTheDocument();
  });

  it("shows character count", () => {
    render(
      <SeoFieldGroup
        seoTitle="Hello"
        onSeoTitleChange={vi.fn()}
        metaDescription=""
        onMetaDescriptionChange={vi.fn()}
      />
    );
    expect(screen.getByText("5/70")).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests**

```bash
cd /home/dev/impress/frontend && pnpm test -- src/components/admin/SeoFieldGroup.test.tsx
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/admin/SeoFieldGroup.tsx frontend/src/components/admin/SeoFieldGroup.test.tsx
git commit -m "feat(1.1.1): reusable SeoFieldGroup component for admin editors"
```

---

## Chunk 2: Full-Text Search (Tasks 1.2.1 — 1.2.6)

### Task 7: SearchProvider interface (1.2.6 — define first, implement second)

**Files:**
- Create: `backend/internal/provider/search.go`

- [ ] **Step 1: Define SearchProvider interface**

Create `backend/internal/provider/search.go`:

```go
package provider

import "context"

// SearchResult represents a single search hit.
type SearchResult struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`      // "article", "page"
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`   // highlighted text excerpt
	URL       string `json:"url"`
	Locale    string `json:"locale"`
	Score     float64 `json:"score"`
}

// SearchResponse wraps paginated search results.
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	Query      string         `json:"query"`
}

// SearchProvider defines the contract for full-text search backends.
// Default implementation uses SQLite FTS5 / PostgreSQL tsvector.
// Plugins can replace with Meilisearch, Elasticsearch, etc.
type SearchProvider interface {
	// Search performs a full-text search query.
	Search(ctx context.Context, query string, locale string, contentType string, page int, pageSize int) (*SearchResponse, error)

	// Suggest returns autocomplete suggestions for a partial query.
	Suggest(ctx context.Context, prefix string, locale string, limit int) ([]string, error)

	// IndexArticle adds or updates an article in the search index.
	IndexArticle(ctx context.Context, id uint, locale string, title string, body string, slug string) error

	// IndexPage adds or updates a page in the search index.
	IndexPage(ctx context.Context, id uint, locale string, title string, body string, slug string) error

	// RemoveFromIndex removes a document from the search index.
	RemoveFromIndex(ctx context.Context, contentType string, id uint) error

	// RebuildIndex rebuilds the entire search index from scratch.
	RebuildIndex(ctx context.Context) error
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/provider/search.go
git commit -m "feat(1.2.6): define SearchProvider interface for pluggable search backends"
```

### Task 8: Search model and migration (1.2.2)

**Files:**
- Create: `backend/internal/db/migrations/00003_create_search_index.sql`
- Create: `backend/internal/model/search_index.go`

- [ ] **Step 1: Create search index migration**

Create `backend/internal/db/migrations/00003_create_search_index.sql`:

```sql
-- +goose Up

-- SQLite FTS5 virtual table for full-text search
-- Note: This migration only runs on SQLite. For PostgreSQL, use tsvector columns.
-- The application code detects the DB driver and uses the appropriate search strategy.

-- For SQLite: FTS5 virtual table
-- +goose StatementBegin
CREATE VIRTUAL TABLE IF NOT EXISTS search_index_fts USING fts5(
    content_type,
    content_id UNINDEXED,
    locale,
    title,
    body,
    slug UNINDEXED,
    tokenize='unicode61'
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS search_index_fts;
```

Note: For PostgreSQL, the search_service will use tsvector columns on the articles/pages tables directly. The FTS5 table is SQLite-specific.

- [ ] **Step 2: Create search index model for non-FTS tracking**

Create `backend/internal/model/search_index.go`:

```go
package model

import "time"

// SearchIndexEntry tracks what content is indexed (for rebuild tracking).
type SearchIndexEntry struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ContentType string    `json:"contentType" gorm:"size:20;index:idx_search_content,unique"` // "article" or "page"
	ContentID   uint      `json:"contentId" gorm:"index:idx_search_content,unique"`
	Locale      string    `json:"locale" gorm:"size:5"`
	IndexedAt   time.Time `json:"indexedAt" gorm:"autoCreateTime"`
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/db/migrations/00003_create_search_index.sql backend/internal/model/search_index.go
git commit -m "feat(1.2.2): search index migration (SQLite FTS5) and tracking model"
```

### Task 9: Search service — SQLite FTS5 implementation (1.2.2)

**Files:**
- Create: `backend/internal/service/search_service.go`
- Create: `backend/internal/service/search_service_test.go`

- [ ] **Step 1: Write search service tests**

Create `backend/internal/service/search_service_test.go`:

```go
package service_test

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/service"
)

func setupSearchTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	// Create FTS5 table
	sqlDB.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS search_index_fts USING fts5(
		content_type, content_id UNINDEXED, locale, title, body, slug UNINDEXED, tokenize='unicode61'
	)`)
	db.AutoMigrate(&model.Article{}, &model.Page{})
	return db
}

func TestSearchServiceIndexAndSearch(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false) // false = SQLite mode

	ctx := context.Background()

	// Index an article
	err := svc.IndexArticle(ctx, 1, "zh", "Go语言入门教程", "这是一篇关于Go语言的入门文章", "go-intro")
	if err != nil {
		t.Fatalf("index article: %v", err)
	}

	// Search for it
	resp, err := svc.Search(ctx, "Go语言", "zh", "", 1, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least 1 result")
	}
	if resp.Results[0].Title != "Go语言入门教程" {
		t.Errorf("expected matching title, got %q", resp.Results[0].Title)
	}
}

func TestSearchServiceRemoveFromIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	svc.IndexArticle(ctx, 1, "zh", "Test", "Content", "test")
	svc.RemoveFromIndex(ctx, "article", 1)

	resp, _ := svc.Search(ctx, "Test", "zh", "", 1, 10)
	if resp.Total != 0 {
		t.Errorf("expected 0 results after remove, got %d", resp.Total)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/dev/impress/backend && go test -v -run TestSearchService ./internal/service/
```

Expected: FAIL — NewSearchService not found.

- [ ] **Step 3: Implement SearchService**

Create `backend/internal/service/search_service.go`:

```go
package service

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"blotting-consultancy/internal/provider"
)

// SearchService implements provider.SearchProvider using SQLite FTS5 or PostgreSQL tsvector.
type SearchService struct {
	db         *gorm.DB
	isPostgres bool
}

func NewSearchService(db *gorm.DB, isPostgres bool) *SearchService {
	return &SearchService{db: db, isPostgres: isPostgres}
}

func (s *SearchService) Search(ctx context.Context, query, locale, contentType string, page, pageSize int) (*provider.SearchResponse, error) {
	if s.isPostgres {
		return s.searchPostgres(ctx, query, locale, contentType, page, pageSize)
	}
	return s.searchSQLite(ctx, query, locale, contentType, page, pageSize)
}

func (s *SearchService) searchSQLite(ctx context.Context, query, locale, contentType string, page, pageSize int) (*provider.SearchResponse, error) {
	offset := (page - 1) * pageSize

	// Build WHERE clause
	conditions := []string{"search_index_fts MATCH ?"}
	args := []interface{}{query}

	if locale != "" {
		conditions = append(conditions, "locale = ?")
		args = append(args, locale)
	}
	if contentType != "" {
		conditions = append(conditions, "content_type = ?")
		args = append(args, contentType)
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM search_index_fts WHERE %s", where)
	s.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total)

	// Fetch results
	selectSQL := fmt.Sprintf(
		"SELECT content_type, content_id, locale, title, snippet(search_index_fts, 4, '<mark>', '</mark>', '...', 32) as body, slug, rank FROM search_index_fts WHERE %s ORDER BY rank LIMIT ? OFFSET ?",
		where,
	)
	fetchArgs := append(args, pageSize, offset)

	var rows []struct {
		ContentType string
		ContentID   uint
		Locale      string
		Title       string
		Body        string
		Slug        string
		Rank        float64
	}
	s.db.WithContext(ctx).Raw(selectSQL, fetchArgs...).Scan(&rows)

	results := make([]provider.SearchResult, len(rows))
	for i, row := range rows {
		url := "/" + row.Slug
		if row.ContentType == "article" {
			url = "/blog/" + row.Slug
		}
		results[i] = provider.SearchResult{
			ID:      row.ContentID,
			Type:    row.ContentType,
			Title:   row.Title,
			Snippet: row.Body,
			URL:     url,
			Locale:  row.Locale,
			Score:   -row.Rank, // FTS5 rank is negative (lower = better)
		}
	}

	return &provider.SearchResponse{
		Results:  results,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Query:    query,
	}, nil
}

func (s *SearchService) searchPostgres(ctx context.Context, query, locale, contentType string, page, pageSize int) (*provider.SearchResponse, error) {
	// PostgreSQL implementation using tsvector/tsquery
	// Will search articles and pages tables directly using to_tsvector/to_tsquery
	// TODO: implement when PostgreSQL testing is available
	return s.searchSQLite(ctx, query, locale, contentType, page, pageSize)
}

func (s *SearchService) Suggest(ctx context.Context, prefix, locale string, limit int) ([]string, error) {
	if limit == 0 {
		limit = 5
	}
	var titles []string
	sql := "SELECT DISTINCT title FROM search_index_fts WHERE title MATCH ? AND locale = ? LIMIT ?"
	s.db.WithContext(ctx).Raw(sql, prefix+"*", locale, limit).Scan(&titles)
	return titles, nil
}

func (s *SearchService) IndexArticle(ctx context.Context, id uint, locale, title, body, slug string) error {
	// Remove existing entry first
	s.RemoveFromIndex(ctx, "article", id)
	sql := "INSERT INTO search_index_fts(content_type, content_id, locale, title, body, slug) VALUES(?, ?, ?, ?, ?, ?)"
	return s.db.WithContext(ctx).Exec(sql, "article", id, locale, title, body, slug).Error
}

func (s *SearchService) IndexPage(ctx context.Context, id uint, locale, title, body, slug string) error {
	s.RemoveFromIndex(ctx, "page", id)
	sql := "INSERT INTO search_index_fts(content_type, content_id, locale, title, body, slug) VALUES(?, ?, ?, ?, ?, ?)"
	return s.db.WithContext(ctx).Exec(sql, "page", id, locale, title, body, slug).Error
}

func (s *SearchService) RemoveFromIndex(ctx context.Context, contentType string, id uint) error {
	// FTS5 uses rowid-based delete; find and delete matching rows
	sql := "DELETE FROM search_index_fts WHERE content_type = ? AND content_id = ?"
	return s.db.WithContext(ctx).Exec(sql, contentType, id).Error
}

func (s *SearchService) RebuildIndex(ctx context.Context) error {
	// Clear all entries
	s.db.WithContext(ctx).Exec("DELETE FROM search_index_fts")

	// Re-index all published articles
	var articles []struct {
		ID      uint
		ZhTitle string
		EnTitle string
		ZhBody  string
		EnBody  string
		Slug    string
	}
	s.db.WithContext(ctx).Table("articles").Where("status = ?", "published").Find(&articles)
	for _, a := range articles {
		if a.ZhTitle != "" {
			s.IndexArticle(ctx, a.ID, "zh", a.ZhTitle, a.ZhBody, a.Slug)
		}
		if a.EnTitle != "" {
			s.IndexArticle(ctx, a.ID, "en", a.EnTitle, a.EnBody, a.Slug)
		}
	}

	return nil
}

// Verify interface compliance
var _ provider.SearchProvider = (*SearchService)(nil)
```

- [ ] **Step 4: Run tests**

```bash
cd /home/dev/impress/backend && go test -v -run TestSearchService ./internal/service/
```

Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/search_service.go backend/internal/service/search_service_test.go
git commit -m "feat(1.2.2): SearchService with SQLite FTS5 full-text search implementation"
```

### Task 10: Search API handler (1.2.1)

**Files:**
- Create: `backend/internal/handler/search/handler.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Create search handler**

Create `backend/internal/handler/search/handler.go`:

```go
package search

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"blotting-consultancy/internal/provider"
)

type Handler struct {
	search provider.SearchProvider
}

func NewHandler(search provider.SearchProvider) *Handler {
	return &Handler{search: search}
}

// PublicSearch handles GET /public/search?q=&locale=&type=&page=&pageSize=
func (h *Handler) PublicSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q parameter is required"})
		return
	}

	locale := c.DefaultQuery("locale", "zh")
	contentType := c.Query("type") // optional: "article", "page"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	resp, err := h.search.Search(c.Request.Context(), query, locale, contentType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PublicSuggest handles GET /public/search/suggest?q=&locale=
func (h *Handler) PublicSuggest(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusOK, []string{})
		return
	}

	locale := c.DefaultQuery("locale", "zh")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	suggestions, err := h.search.Suggest(c.Request.Context(), prefix, locale, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "suggest failed"})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

// AdminRebuildIndex handles POST /admin/search/rebuild
func (h *Handler) AdminRebuildIndex(c *gin.Context) {
	if err := h.search.RebuildIndex(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rebuild failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "index rebuilt successfully"})
}

func (h *Handler) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.GET("/search", h.PublicSearch)
	public.GET("/search/suggest", h.PublicSuggest)
	admin.POST("/search/rebuild", h.AdminRebuildIndex)
}
```

- [ ] **Step 2: Wire into main.go**

```go
searchService := service.NewSearchService(database.DB, db.IsPostgresDSN(cfg.DBDSN))
searchHandler := searchhandler.NewHandler(searchService)
searchHandler.RegisterRoutes(publicGroup, adminGroup)
```

- [ ] **Step 3: Verify build**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/search/ backend/cmd/server/main.go
git commit -m "feat(1.2.1): search API handler — GET /public/search, /public/search/suggest, POST /admin/search/rebuild"
```

### Task 11: Search index auto-update on publish (1.2.3)

**Files:**
- Modify: `backend/internal/handler/article/handler.go`
- Modify: `backend/internal/handler/page/handler.go`

- [ ] **Step 1: Add search service to article handler**

In `backend/internal/handler/article/handler.go`, add `searchService *service.SearchService` to the Handler struct and constructor. In `AdminCreate` and `AdminUpdate`, after successful save, index the article:

```go
// After article creation/update with status=published
if article.Status == "published" {
    go func() {
        ctx := context.Background()
        if article.ZhTitle != "" {
            h.searchService.IndexArticle(ctx, article.ID, "zh", article.ZhTitle, article.ZhBody, article.Slug)
        }
        if article.EnTitle != "" {
            h.searchService.IndexArticle(ctx, article.ID, "en", article.EnTitle, article.EnBody, article.Slug)
        }
    }()
}
```

In `AdminDelete`, remove from index:
```go
go func() {
    h.searchService.RemoveFromIndex(context.Background(), "article", id)
}()
```

- [ ] **Step 2: Wire searchService into article handler in main.go**

Update the ArticleHandler constructor call to pass `searchService`.

- [ ] **Step 3: Verify build**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/article/ backend/cmd/server/main.go
git commit -m "feat(1.2.3): auto-index articles on publish, remove from index on delete"
```

### Task 12: Frontend search components (1.2.4 + 1.2.5)

**Files:**
- Create: `frontend/src/api/search.ts`
- Create: `frontend/src/hooks/useSearch.ts`
- Create: `frontend/src/components/feature/SearchBox.tsx`
- Create: `frontend/src/pages/search/page.tsx`
- Modify: `frontend/src/router/config.tsx`

- [ ] **Step 1: Create search API module**

Create `frontend/src/api/search.ts`:

```tsx
import http from "./http";

export interface SearchResult {
  id: number;
  type: string;
  title: string;
  snippet: string;
  url: string;
  locale: string;
  score: number;
}

export interface SearchResponse {
  results: SearchResult[];
  total: number;
  page: number;
  pageSize: number;
  query: string;
}

export async function searchContent(
  q: string,
  locale = "zh",
  contentType = "",
  page = 1,
  pageSize = 10
): Promise<SearchResponse> {
  const params = new URLSearchParams({ q, locale, page: String(page), pageSize: String(pageSize) });
  if (contentType) params.set("type", contentType);
  const { data } = await http.get<SearchResponse>(`/public/search?${params}`);
  return data;
}

export async function searchSuggest(q: string, locale = "zh", limit = 5): Promise<string[]> {
  const { data } = await http.get<string[]>(`/public/search/suggest?q=${encodeURIComponent(q)}&locale=${locale}&limit=${limit}`);
  return data;
}
```

- [ ] **Step 2: Create useSearch hook**

Create `frontend/src/hooks/useSearch.ts`:

```tsx
import { useState, useCallback } from "react";
import { searchContent, searchSuggest, type SearchResponse } from "@/api/search";

export function useSearch() {
  const [results, setResults] = useState<SearchResponse | null>(null);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const { i18n } = useTranslation();

  const search = useCallback(async (query: string, contentType = "", page = 1) => {
    setLoading(true);
    try {
      const resp = await searchContent(query, i18n.language, contentType, page);
      setResults(resp);
    } finally {
      setLoading(false);
    }
  }, [i18n.language]);

  const suggest = useCallback(async (prefix: string) => {
    if (prefix.length < 2) {
      setSuggestions([]);
      return;
    }
    const items = await searchSuggest(prefix, i18n.language);
    setSuggestions(items);
  }, [i18n.language]);

  return { results, suggestions, loading, search, suggest };
}
```

- [ ] **Step 3: Create SearchBox component**

Create `frontend/src/components/feature/SearchBox.tsx`:

```tsx
import { useState, useRef, useEffect } from "react";
import { useSearch } from "@/hooks/useSearch";

interface SearchBoxProps {
  onSelect?: (url: string) => void;
  className?: string;
}

export default function SearchBox({ onSelect, className }: SearchBoxProps) {
  const [query, setQuery] = useState("");
  const [showSuggestions, setShowSuggestions] = useState(false);
  const { suggestions, suggest } = useSearch();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const timer = setTimeout(() => {
      if (query) suggest(query);
    }, 300);
    return () => clearTimeout(timer);
  }, [query, suggest]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      navigate(`/search?q=${encodeURIComponent(query.trim())}`);
      setShowSuggestions(false);
    }
  };

  const handleSuggestionClick = (text: string) => {
    setQuery(text);
    navigate(`/search?q=${encodeURIComponent(text)}`);
    setShowSuggestions(false);
  };

  return (
    <form onSubmit={handleSubmit} className={`relative ${className ?? ""}`}>
      <input
        ref={inputRef}
        type="text"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setShowSuggestions(true);
        }}
        onFocus={() => setShowSuggestions(true)}
        onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
        placeholder={t("search.placeholder", "Search...")}
        className="w-full border rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"
      />
      {showSuggestions && suggestions.length > 0 && (
        <ul className="absolute z-50 w-full mt-1 bg-white border rounded-lg shadow-lg max-h-60 overflow-auto">
          {suggestions.map((s, i) => (
            <li
              key={i}
              className="px-4 py-2 hover:bg-gray-100 cursor-pointer text-sm"
              onMouseDown={() => handleSuggestionClick(s)}
            >
              {s}
            </li>
          ))}
        </ul>
      )}
    </form>
  );
}
```

- [ ] **Step 4: Create search results page**

Create `frontend/src/pages/search/page.tsx`:

```tsx
import { useEffect } from "react";
import { useSearch } from "@/hooks/useSearch";
import SearchBox from "@/components/feature/SearchBox";

export default function SearchPage() {
  const [searchParams] = useSearchParams();
  const query = searchParams.get("q") ?? "";
  const page = Number(searchParams.get("page") ?? "1");
  const { results, loading, search } = useSearch();
  const { t } = useTranslation();

  useEffect(() => {
    if (query) search(query, "", page);
  }, [query, page, search]);

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <SearchBox className="mb-8" />

      {loading && <p className="text-gray-500">{t("search.loading", "Searching...")}</p>}

      {results && (
        <>
          <p className="text-sm text-gray-500 mb-4">
            {t("search.results_count", { count: results.total, defaultValue: "{{count}} results found" })}
          </p>

          <div className="space-y-6">
            {results.results.map((r) => (
              <Link key={`${r.type}-${r.id}`} to={r.url} className="block group">
                <h3 className="text-lg font-medium text-blue-700 group-hover:underline">{r.title}</h3>
                <p
                  className="text-sm text-gray-600 mt-1"
                  dangerouslySetInnerHTML={{ __html: r.snippet }}
                />
                <span className="text-xs text-gray-400">{r.url}</span>
              </a>
            ))}
          </div>

          {results.total > results.pageSize && (
            <div className="flex justify-center gap-2 mt-8">
              {Array.from({ length: Math.ceil(results.total / results.pageSize) }, (_, i) => (
                <Link
                  key={i}
                  to={`/search?q=${encodeURIComponent(query)}&page=${i + 1}`}
                  className={`px-3 py-1 rounded ${i + 1 === results.page ? "bg-blue-600 text-white" : "bg-gray-100 hover:bg-gray-200"}`}
                >
                  {i + 1}
                </a>
              ))}
            </div>
          )}
        </>
      )}

      {results && results.total === 0 && (
        <p className="text-gray-500">{t("search.no_results", "No results found")}</p>
      )}
    </div>
  );
}
```

- [ ] **Step 5: Add search route**

In `frontend/src/router/config.tsx`, add:

```tsx
{
  path: "/search",
  lazy: () => import("@/pages/search/page").then(m => ({ Component: m.default })),
},
```

- [ ] **Step 6: Run lint and type-check**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check
```

- [ ] **Step 7: Commit**

```bash
git add frontend/src/api/search.ts frontend/src/hooks/useSearch.ts frontend/src/components/feature/SearchBox.tsx frontend/src/pages/search/page.tsx frontend/src/router/config.tsx
git commit -m "feat(1.2.4+1.2.5): frontend search — SearchBox, search results page, autocomplete"
```

---

## Chunk 3: Comment System (Tasks 1.3.1 — 1.3.7)

### Task 13: Comment model and migration (1.3.1)

**Files:**
- Create: `backend/internal/model/comment.go`
- Create: `backend/internal/db/migrations/00004_create_comments.sql`

- [ ] **Step 1: Create Comment model**

Create `backend/internal/model/comment.go`:

```go
package model

import (
	"time"

	"gorm.io/gorm"
)

type CommentStatus string

const (
	CommentStatusPending  CommentStatus = "pending"
	CommentStatusApproved CommentStatus = "approved"
	CommentStatusSpam     CommentStatus = "spam"
	CommentStatusTrash    CommentStatus = "trash"
)

// Comment represents a user comment on an article or page.
type Comment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Content
	Content string `json:"content" gorm:"type:text;not null"`

	// Author info (anonymous users, no account required)
	AuthorName  string `json:"authorName" gorm:"size:100;not null"`
	AuthorEmail string `json:"authorEmail" gorm:"size:255"`
	AuthorURL   string `json:"authorUrl" gorm:"size:500"`
	AuthorIP    string `json:"-" gorm:"size:45"` // hidden from public API

	// Relations
	ContentType string `json:"contentType" gorm:"size:20;not null;index:idx_comment_target"` // "article" or "page"
	ContentID   uint   `json:"contentId" gorm:"not null;index:idx_comment_target"`

	// Nesting
	ParentID *uint      `json:"parentId" gorm:"index"`
	Children []*Comment `json:"children,omitempty" gorm:"foreignKey:ParentID"`

	// Moderation
	Status CommentStatus `json:"status" gorm:"size:20;default:pending;index"`
	Pinned bool          `json:"pinned" gorm:"default:false"`

	// Optional: site_id for future multi-site support (Phase 4)
	SiteID *uint `json:"-" gorm:"index"`
}

func (c *Comment) Validate() error {
	if c.Content == "" {
		return errors.New("content is required")
	}
	if c.AuthorName == "" {
		return errors.New("author name is required")
	}
	if c.ContentType != "article" && c.ContentType != "page" {
		return errors.New("content type must be 'article' or 'page'")
	}
	if c.ContentID == 0 {
		return errors.New("content id is required")
	}
	return nil
}
```

Add import `"errors"` to the file. This follows the existing model convention (Article, Page use `errors.New`).

- [ ] **Step 2: Create migration**

Create `backend/internal/db/migrations/00004_create_comments.sql`:

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    content TEXT NOT NULL,
    author_name VARCHAR(100) NOT NULL,
    author_email VARCHAR(255),
    author_url VARCHAR(500),
    author_ip VARCHAR(45),
    content_type VARCHAR(20) NOT NULL,
    content_id INTEGER NOT NULL,
    parent_id INTEGER REFERENCES comments(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    site_id INTEGER
);

CREATE INDEX IF NOT EXISTS idx_comment_target ON comments(content_type, content_id);
CREATE INDEX IF NOT EXISTS idx_comment_status ON comments(status);
CREATE INDEX IF NOT EXISTS idx_comment_parent ON comments(parent_id);
CREATE INDEX IF NOT EXISTS idx_comment_site ON comments(site_id);
CREATE INDEX IF NOT EXISTS idx_comment_deleted ON comments(deleted_at);

-- +goose Down
DROP TABLE IF EXISTS comments;
```

- [ ] **Step 3: Add Comment to AutoMigrate in main.go**

Add `&model.Comment{}` to the AutoMigrate call in `main.go`.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/model/comment.go backend/internal/db/migrations/00004_create_comments.sql backend/cmd/server/main.go
git commit -m "feat(1.3.1): Comment model with nesting, moderation status, and migration"
```

### Task 14: Comment repository (1.3.1)

**Files:**
- Create: `backend/internal/repository/comment_repository.go`
- Create: `backend/internal/repository/comment_repository_impl.go`

- [ ] **Step 1: Define comment repository interface**

Create `backend/internal/repository/comment_repository.go`:

```go
package repository

import (
	"context"

	"blotting-consultancy/internal/model"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	FindByID(ctx context.Context, id uint) (*model.Comment, error)
	Update(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id uint) error

	// ListByContent returns top-level comments (parentID=nil) for a content item, with children preloaded.
	ListByContent(ctx context.Context, contentType string, contentID uint, status model.CommentStatus, page, pageSize int) ([]*model.Comment, int64, error)

	// ListAll returns all comments for admin management.
	ListAll(ctx context.Context, status string, page, pageSize int) ([]*model.Comment, int64, error)

	// CountByContent returns comment count for a content item.
	CountByContent(ctx context.Context, contentType string, contentID uint) (int64, error)

	// UpdateStatus updates the moderation status of a comment.
	UpdateStatus(ctx context.Context, id uint, status model.CommentStatus) error

	// Pin or unpin a comment.
	SetPinned(ctx context.Context, id uint, pinned bool) error
}
```

- [ ] **Step 2: Implement comment repository**

Create `backend/internal/repository/comment_repository_impl.go`:

```go
package repository

import (
	"context"

	"gorm.io/gorm"
	"blotting-consultancy/internal/model"
)

type GormCommentRepositoryImpl struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &GormCommentRepositoryImpl{db: db}
}

func (r *GormCommentRepositoryImpl) Create(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *GormCommentRepositoryImpl) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.WithContext(ctx).Preload("Children").First(&comment, id).Error
	return &comment, err
}

func (r *GormCommentRepositoryImpl) Update(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

func (r *GormCommentRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Comment{}, id).Error
}

func (r *GormCommentRepositoryImpl) ListByContent(ctx context.Context, contentType string, contentID uint, status model.CommentStatus, page, pageSize int) ([]*model.Comment, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("content_type = ? AND content_id = ? AND parent_id IS NULL", contentType, contentID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	var comments []*model.Comment
	err := query.
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", model.CommentStatusApproved).Order("created_at ASC")
		}).
		Order("pinned DESC, created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error

	return comments, total, err
}

func (r *GormCommentRepositoryImpl) ListAll(ctx context.Context, status string, page, pageSize int) ([]*model.Comment, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Comment{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	var comments []*model.Comment
	err := query.
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error

	return comments, total, err
}

func (r *GormCommentRepositoryImpl) CountByContent(ctx context.Context, contentType string, contentID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("content_type = ? AND content_id = ? AND status = ?", contentType, contentID, model.CommentStatusApproved).
		Count(&count).Error
	return count, err
}

func (r *GormCommentRepositoryImpl) UpdateStatus(ctx context.Context, id uint, status model.CommentStatus) error {
	return r.db.WithContext(ctx).Model(&model.Comment{}).Where("id = ?", id).Update("status", status).Error
}

func (r *GormCommentRepositoryImpl) SetPinned(ctx context.Context, id uint, pinned bool) error {
	return r.db.WithContext(ctx).Model(&model.Comment{}).Where("id = ?", id).Update("pinned", pinned).Error
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/comment_repository.go backend/internal/repository/comment_repository_impl.go
git commit -m "feat(1.3.1): comment repository interface and GORM implementation"
```

### Task 15: Anti-spam service with CaptchaProvider interface (1.3.5)

**Files:**
- Create: `backend/internal/provider/captcha.go`
- Create: `backend/internal/service/antispam_service.go`

- [ ] **Step 1: Define CaptchaProvider interface**

Create `backend/internal/provider/captcha.go`:

```go
package provider

import "context"

// CaptchaProvider defines the contract for human verification.
// Default: no-op (always passes). Plugins can implement reCAPTCHA, hCaptcha, etc.
type CaptchaProvider interface {
	// Verify checks the captcha response token. Returns nil if valid.
	Verify(ctx context.Context, token string, remoteIP string) error
}

// NoopCaptchaProvider always passes verification (default when no plugin installed).
type NoopCaptchaProvider struct{}

func (p *NoopCaptchaProvider) Verify(ctx context.Context, token string, remoteIP string) error {
	return nil
}
```

- [ ] **Step 2: Define Notifier interface**

Create `backend/internal/provider/notifier.go`:

```go
package provider

import "context"

// NotifyEvent describes a notification to send.
type NotifyEvent struct {
	Type    string            // "new_comment", "form_submission", etc.
	Subject string
	Body    string
	Meta    map[string]string // additional key-value pairs
}

// NotifierProvider defines the contract for sending notifications.
// Default: log-only. Plugins can implement email, Slack, webhook, etc.
type NotifierProvider interface {
	Notify(ctx context.Context, event NotifyEvent) error
}
```

- [ ] **Step 3: Create anti-spam service**

Create `backend/internal/service/antispam_service.go`:

```go
package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"blotting-consultancy/internal/provider"
)

type AntiSpamService struct {
	captcha     provider.CaptchaProvider
	keywords    []string
	mu          sync.RWMutex
	ipTracker   map[string][]time.Time // IP -> submission timestamps
	rateLimit   int                    // max submissions per window
	rateWindow  time.Duration
	done        chan struct{}
}

func NewAntiSpamService(captcha provider.CaptchaProvider) *AntiSpamService {
	svc := &AntiSpamService{
		captcha:    captcha,
		keywords:   []string{}, // loaded from config/DB in future
		ipTracker:  make(map[string][]time.Time),
		rateLimit:  5,
		rateWindow: 10 * time.Minute,
		done:       make(chan struct{}),
	}
	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				svc.cleanupTracker()
			case <-svc.done:
				return
			}
		}
	}()
	return svc
}

// Stop stops the cleanup goroutine.
func (s *AntiSpamService) Stop() {
	close(s.done)
}

// Check runs all anti-spam checks. Returns nil if OK, error describing the failure otherwise.
func (s *AntiSpamService) Check(ctx context.Context, ip, content, captchaToken string) error {
	// 1. Rate limit by IP
	if !s.checkRateLimit(ip) {
		return &SpamError{Reason: "rate_limit", Message: "Too many submissions, please try again later"}
	}

	// 2. Keyword filter
	if s.containsBannedKeyword(content) {
		return &SpamError{Reason: "keyword", Message: "Content contains blocked keywords"}
	}

	// 3. Captcha verification (no-op by default)
	if captchaToken != "" {
		if err := s.captcha.Verify(ctx, captchaToken, ip); err != nil {
			return &SpamError{Reason: "captcha", Message: "Captcha verification failed"}
		}
	}

	// Record this submission
	s.recordSubmission(ip)
	return nil
}

func (s *AntiSpamService) checkRateLimit(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	timestamps := s.ipTracker[ip]
	cutoff := time.Now().Add(-s.rateWindow)
	count := 0
	for _, t := range timestamps {
		if t.After(cutoff) {
			count++
		}
	}
	return count < s.rateLimit
}

func (s *AntiSpamService) recordSubmission(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ipTracker[ip] = append(s.ipTracker[ip], time.Now())
}

func (s *AntiSpamService) containsBannedKeyword(content string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lower := strings.ToLower(content)
	for _, kw := range s.keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func (s *AntiSpamService) cleanupTracker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-s.rateWindow)
	for ip, timestamps := range s.ipTracker {
		var active []time.Time
		for _, t := range timestamps {
			if t.After(cutoff) {
				active = append(active, t)
			}
		}
		if len(active) == 0 {
			delete(s.ipTracker, ip)
		} else {
			s.ipTracker[ip] = active
		}
	}
}

type SpamError struct {
	Reason  string
	Message string
}

func (e *SpamError) Error() string { return e.Message }
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/provider/captcha.go backend/internal/provider/notifier.go backend/internal/service/antispam_service.go
git commit -m "feat(1.3.5): CaptchaProvider + NotifierProvider interfaces, anti-spam service"
```

### Task 16: Comment handler (1.3.2 + 1.3.3)

**Files:**
- Create: `backend/internal/handler/comment/handler.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Create comment handler**

Create `backend/internal/handler/comment/handler.go`:

```go
package comment

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
)

type Handler struct {
	repo     repository.CommentRepository
	antispam *service.AntiSpamService
}

func NewHandler(repo repository.CommentRepository, antispam *service.AntiSpamService) *Handler {
	return &Handler{repo: repo, antispam: antispam}
}

type createInput struct {
	Content      string `json:"content" binding:"required"`
	AuthorName   string `json:"authorName" binding:"required"`
	AuthorEmail  string `json:"authorEmail"`
	AuthorURL    string `json:"authorUrl"`
	ContentType  string `json:"contentType" binding:"required"`
	ContentID    uint   `json:"contentId" binding:"required"`
	ParentID     *uint  `json:"parentId"`
	CaptchaToken string `json:"captchaToken"`
}

// PublicCreate handles POST /public/comments
func (h *Handler) PublicCreate(c *gin.Context) {
	var input createInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Anti-spam check
	ip := c.ClientIP()
	if err := h.antispam.Check(c.Request.Context(), ip, input.Content, input.CaptchaToken); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	comment := &model.Comment{
		Content:     input.Content,
		AuthorName:  input.AuthorName,
		AuthorEmail: input.AuthorEmail,
		AuthorURL:   input.AuthorURL,
		AuthorIP:    ip,
		ContentType: input.ContentType,
		ContentID:   input.ContentID,
		ParentID:    input.ParentID,
		Status:      model.CommentStatusPending, // Default: requires approval
	}

	if err := comment.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// PublicList handles GET /public/comments?contentType=article&contentId=1&page=1&pageSize=20
func (h *Handler) PublicList(c *gin.Context) {
	contentType := c.Query("contentType")
	contentID, _ := strconv.ParseUint(c.Query("contentId"), 10, 32)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if contentType == "" || contentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentType and contentId required"})
		return
	}

	comments, total, err := h.repo.ListByContent(c.Request.Context(), contentType, uint(contentID), model.CommentStatusApproved, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminList handles GET /admin/comments?status=pending&page=1&pageSize=20
func (h *Handler) AdminList(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	comments, total, err := h.repo.ListAll(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminUpdateStatus handles PATCH /admin/comments/:id/status
func (h *Handler) AdminUpdateStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var input struct {
		Status model.CommentStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), uint(id), input.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// AdminDelete handles DELETE /admin/comments/:id
func (h *Handler) AdminDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// AdminPin handles PUT /admin/comments/:id/pin
func (h *Handler) AdminPin(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var input struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SetPinned(c.Request.Context(), uint(id), input.Pinned); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to pin comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *Handler) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.POST("/comments", h.PublicCreate)
	public.GET("/comments", h.PublicList)

	admin.GET("/comments", h.AdminList)
	admin.PATCH("/comments/:id/status", h.AdminUpdateStatus)
	admin.DELETE("/comments/:id", h.AdminDelete)
	admin.PUT("/comments/:id/pin", h.AdminPin)
}
```

- [ ] **Step 2: Wire in main.go**

```go
commentRepo := repository.NewCommentRepository(database.DB)
captchaProvider := &provider.NoopCaptchaProvider{}
antispamService := service.NewAntiSpamService(captchaProvider)
commentHandler := commenthandler.NewHandler(commentRepo, antispamService)
commentHandler.RegisterRoutes(publicGroup, adminGroup)
```

- [ ] **Step 3: Verify build**

```bash
cd /home/dev/impress/backend && go build -o server ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/comment/ backend/cmd/server/main.go
git commit -m "feat(1.3.2+1.3.3): comment handler with public CRUD, admin moderation, anti-spam"
```

### Task 17: Frontend comment section (1.3.4 + 1.3.6)

**Files:**
- Create: `frontend/src/api/comments.ts`
- Create: `frontend/src/components/feature/CommentSection.tsx`

- [ ] **Step 1: Create comments API module**

Create `frontend/src/api/comments.ts`:

```tsx
import http from "./http";

export interface Comment {
  id: number;
  content: string;
  authorName: string;
  authorEmail?: string;
  authorUrl?: string;
  contentType: string;
  contentId: number;
  parentId: number | null;
  status: string;
  pinned: boolean;
  children?: Comment[];
  createdAt: string;
}

export interface CommentListResponse {
  comments: Comment[];
  total: number;
  page: number;
  pageSize: number;
}

export async function getComments(contentType: string, contentId: number, page = 1): Promise<CommentListResponse> {
  const { data } = await http.get<CommentListResponse>(
    `/public/comments?contentType=${contentType}&contentId=${contentId}&page=${page}`
  );
  return data;
}

export async function postComment(input: {
  content: string;
  authorName: string;
  authorEmail?: string;
  contentType: string;
  contentId: number;
  parentId?: number;
  captchaToken?: string;
}): Promise<Comment> {
  const { data } = await http.post<Comment>("/public/comments", input);
  return data;
}
```

- [ ] **Step 2: Create CommentSection component**

Create `frontend/src/components/feature/CommentSection.tsx`:

```tsx
import { useState, useEffect, useCallback } from "react";
import { getComments, postComment, type Comment } from "@/api/comments";

interface CommentSectionProps {
  contentType: "article" | "page";
  contentId: number;
}

function CommentItem({ comment, onReply }: { comment: Comment; onReply: (id: number) => void }) {
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="border-l-2 border-gray-200 pl-4 py-3">
      <div className="flex items-center gap-2 mb-1">
        <span className="font-medium text-sm">{comment.authorName}</span>
        <span className="text-xs text-gray-400">
          {new Date(comment.createdAt).toLocaleDateString()}
        </span>
        {comment.pinned && <span className="text-xs bg-yellow-100 text-yellow-700 px-1 rounded">Pinned</span>}
      </div>
      <p className="text-sm text-gray-700 mb-2">{comment.content}</p>
      <button
        onClick={() => onReply(comment.id)}
        className="text-xs text-blue-600 hover:underline"
      >
        Reply
      </button>

      {comment.children && comment.children.length > 0 && (
        <div className="mt-2">
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="text-xs text-gray-500 mb-1"
          >
            {collapsed ? `Show ${comment.children.length} replies` : "Hide replies"}
          </button>
          {!collapsed && (
            <div className="space-y-1">
              {comment.children.map((child) => (
                <CommentItem key={child.id} comment={child} onReply={onReply} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function CommentSection({ contentType, contentId }: CommentSectionProps) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [replyTo, setReplyTo] = useState<number | null>(null);
  const [form, setForm] = useState({ content: "", authorName: "", authorEmail: "" });
  const [submitting, setSubmitting] = useState(false);
  const { t } = useTranslation();

  const loadComments = useCallback(async () => {
    const resp = await getComments(contentType, contentId, page);
    setComments(resp.comments ?? []);
    setTotal(resp.total);
  }, [contentType, contentId, page]);

  useEffect(() => {
    loadComments();
  }, [loadComments]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.content.trim() || !form.authorName.trim()) return;
    setSubmitting(true);
    try {
      await postComment({
        content: form.content,
        authorName: form.authorName,
        authorEmail: form.authorEmail,
        contentType,
        contentId,
        parentId: replyTo ?? undefined,
      });
      setForm({ content: "", authorName: "", authorEmail: "" });
      setReplyTo(null);
      loadComments();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mt-12 border-t pt-8">
      <h3 className="text-lg font-semibold mb-4">
        {t("comments.title", "Comments")} ({total})
      </h3>

      {/* Comment form */}
      <form onSubmit={handleSubmit} className="mb-8 space-y-3 bg-gray-50 p-4 rounded-lg">
        {replyTo && (
          <div className="text-sm text-blue-600">
            Replying to comment #{replyTo}{" "}
            <button type="button" onClick={() => setReplyTo(null)} className="text-gray-400 ml-1">
              Cancel
            </button>
          </div>
        )}
        <div className="flex gap-3">
          <input
            type="text"
            placeholder={t("comments.name", "Name *")}
            value={form.authorName}
            onChange={(e) => setForm({ ...form, authorName: e.target.value })}
            className="border rounded px-3 py-1.5 text-sm flex-1"
            required
          />
          <input
            type="email"
            placeholder={t("comments.email", "Email (optional)")}
            value={form.authorEmail}
            onChange={(e) => setForm({ ...form, authorEmail: e.target.value })}
            className="border rounded px-3 py-1.5 text-sm flex-1"
          />
        </div>
        <textarea
          placeholder={t("comments.write", "Write a comment...")}
          value={form.content}
          onChange={(e) => setForm({ ...form, content: e.target.value })}
          className="w-full border rounded px-3 py-2 text-sm"
          rows={3}
          required
        />
        <button
          type="submit"
          disabled={submitting}
          className="bg-blue-600 text-white px-4 py-1.5 rounded text-sm hover:bg-blue-700 disabled:opacity-50"
        >
          {submitting ? t("comments.submitting", "Submitting...") : t("comments.submit", "Submit")}
        </button>
      </form>

      {/* Comment list */}
      <div className="space-y-2">
        {comments.map((c) => (
          <CommentItem key={c.id} comment={c} onReply={setReplyTo} />
        ))}
      </div>

      {/* Pagination */}
      {total > 20 && (
        <div className="flex justify-center gap-2 mt-4">
          {page > 1 && (
            <button onClick={() => setPage(page - 1)} className="px-3 py-1 bg-gray-100 rounded text-sm">
              Prev
            </button>
          )}
          <span className="px-3 py-1 text-sm text-gray-500">Page {page}</span>
          {total > page * 20 && (
            <button onClick={() => setPage(page + 1)} className="px-3 py-1 bg-gray-100 rounded text-sm">
              Next
            </button>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Integrate into article detail page**

In `frontend/src/pages/blog/[slug]/page.tsx`, add CommentSection at the bottom of the article:

```tsx
import CommentSection from "@/components/feature/CommentSection";

// Inside the return, after article body:
{article.allowComments && (
  <CommentSection contentType="article" contentId={article.id} />
)}
```

- [ ] **Step 4: Run lint and type-check**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/comments.ts frontend/src/components/feature/CommentSection.tsx frontend/src/pages/blog/
git commit -m "feat(1.3.4+1.3.6): frontend comment section with nested replies, pagination, and reply form"
```

---

## Chunk 4: Scheduled Publishing (Tasks 1.4.1 — 1.4.4)

### Task 18: Scheduled publishing model changes (1.4.1)

**Files:**
- Create: `backend/internal/db/migrations/00005_add_scheduled_fields.sql`
- Modify: `backend/internal/model/article.go`

- [ ] **Step 1: Create migration**

Create `backend/internal/db/migrations/00005_add_scheduled_fields.sql`:

```sql
-- +goose Up
ALTER TABLE articles ADD COLUMN IF NOT EXISTS scheduled_at DATETIME;
ALTER TABLE pages ADD COLUMN IF NOT EXISTS scheduled_at DATETIME;

-- +goose Down
-- Best-effort: SQLite <3.35 doesn't support DROP COLUMN
```

- [ ] **Step 2: Add ScheduledAt to Article model**

In `backend/internal/model/article.go`, add:
```go
ScheduledAt *time.Time `json:"scheduledAt" gorm:"index"`
```

Add `"scheduled"` to the valid status values alongside `"draft"` and `"published"`.

- [ ] **Step 3: Add ScheduledAt to Page model similarly**

- [ ] **Step 4: Commit**

```bash
git add backend/internal/db/migrations/00005_add_scheduled_fields.sql backend/internal/model/article.go backend/internal/model/page.go
git commit -m "feat(1.4.1): add ScheduledAt field to Article and Page models with migration"
```

### Task 19: Scheduler service (1.4.2)

**Files:**
- Create: `backend/internal/service/scheduler_service.go`
- Create: `backend/internal/service/scheduler_service_test.go`

- [ ] **Step 1: Write scheduler test**

Create `backend/internal/service/scheduler_service_test.go`:

```go
package service_test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/service"
)

func TestSchedulerPublishesOverdueArticles(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Article{})

	past := time.Now().Add(-1 * time.Hour)
	db.Create(&model.Article{
		Slug:        "test",
		ZhTitle:     "Test",
		Status:      "scheduled",
		ScheduledAt: &past,
	})

	sched := service.NewSchedulerService(db)
	count, err := sched.PublishOverdue()
	if err != nil {
		t.Fatalf("publish overdue: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 published, got %d", count)
	}

	var article model.Article
	db.First(&article, "slug = ?", "test")
	if article.Status != "published" {
		t.Errorf("expected published, got %s", article.Status)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd /home/dev/impress/backend && go test -v -run TestScheduler ./internal/service/
```

- [ ] **Step 3: Implement SchedulerService**

Create `backend/internal/service/scheduler_service.go`:

```go
package service

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type SchedulerService struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSchedulerService(db *gorm.DB) *SchedulerService {
	return &SchedulerService{
		db:     db,
		logger: slog.Default(),
	}
}

// PublishOverdue finds all articles/pages with status=scheduled and scheduledAt <= now,
// updates them to published. Returns count of published items.
func (s *SchedulerService) PublishOverdue() (int, error) {
	now := time.Now()
	total := 0

	// Publish articles
	result := s.db.Table("articles").
		Where("status = ? AND scheduled_at <= ?", "scheduled", now).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	total += int(result.RowsAffected)

	// Publish pages
	result = s.db.Table("pages").
		Where("status = ? AND scheduled_at <= ?", "scheduled", now).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": now,
		})
	if result.Error != nil {
		return total, result.Error
	}
	total += int(result.RowsAffected)

	if total > 0 {
		s.logger.Info("Scheduled publishing completed", "count", total)
	}

	return total, nil
}

// Start begins a cron loop that checks for overdue content every minute.
func (s *SchedulerService) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.PublishOverdue(); err != nil {
				s.logger.Error("Scheduler error", "error", err)
			}
		}
	}()
	s.logger.Info("Scheduler started (checking every 1 minute)")
}
```

- [ ] **Step 4: Run test**

```bash
cd /home/dev/impress/backend && go test -v -run TestScheduler ./internal/service/
```

Expected: PASS

- [ ] **Step 5: Wire into main.go**

```go
schedulerService := service.NewSchedulerService(database.DB)
schedulerService.Start()
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/service/scheduler_service.go backend/internal/service/scheduler_service_test.go backend/cmd/server/main.go
git commit -m "feat(1.4.2): scheduler service — auto-publishes overdue scheduled content every minute"
```

### Task 20: Frontend schedule publish UI (1.4.3 + 1.4.4)

**Files:**
- Create: `frontend/src/components/admin/SchedulePublishModal.tsx`

- [ ] **Step 1: Create schedule modal**

Create `frontend/src/components/admin/SchedulePublishModal.tsx`:

```tsx
import { useState } from "react";

interface SchedulePublishModalProps {
  open: boolean;
  onClose: () => void;
  onSchedule: (date: string) => void;
  currentSchedule?: string | null;
}

export default function SchedulePublishModal({ open, onClose, onSchedule, currentSchedule }: SchedulePublishModalProps) {
  const [date, setDate] = useState(currentSchedule ?? "");
  const { t } = useTranslation();

  if (!open) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (date) onSchedule(date);
  };

  // Minimum date: now + 5 minutes
  const minDate = new Date(Date.now() + 5 * 60000).toISOString().slice(0, 16);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-lg p-6 w-96 shadow-xl">
        <h3 className="text-lg font-semibold mb-4">{t("schedule.title", "Schedule Publish")}</h3>
        <form onSubmit={handleSubmit}>
          <label className="block text-sm text-gray-600 mb-2">
            {t("schedule.datetime", "Publish Date & Time")}
          </label>
          <input
            type="datetime-local"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            min={minDate}
            className="w-full border rounded px-3 py-2 mb-4"
            required
          />
          <div className="flex justify-end gap-2">
            <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded">
              {t("common.cancel", "Cancel")}
            </button>
            {currentSchedule && (
              <button
                type="button"
                onClick={() => onSchedule("")}
                className="px-4 py-2 text-sm text-red-600 hover:bg-red-50 rounded"
              >
                {t("schedule.cancel_schedule", "Cancel Schedule")}
              </button>
            )}
            <button type="submit" className="px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700">
              {t("schedule.confirm", "Schedule")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Integrate into article editor**

In the article editor page (`frontend/src/pages/admin/articles/editor/page.tsx`), add a "Schedule" button next to the existing Publish button. When clicked, open `SchedulePublishModal`. On confirm, set `status: "scheduled"` and `scheduledAt` in the article data.

- [ ] **Step 3: Run lint and type-check**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/admin/SchedulePublishModal.tsx frontend/src/pages/admin/articles/
git commit -m "feat(1.4.3): schedule publish modal and article editor integration"
```

---

## Chunk 5: Markdown Editor (Tasks 1.5.1 — 1.5.5)

### Task 21: Editor mode switcher (1.5.1)

**Files:**
- Create: `frontend/src/components/admin/editor/EditorModeSwitcher.tsx`
- Create: `frontend/src/components/admin/editor/MarkdownMode.tsx`

- [ ] **Step 1: Create mode switcher component**

Create `frontend/src/components/admin/editor/EditorModeSwitcher.tsx`:

```tsx
import { useState } from "react";

interface EditorModeSwitcherProps {
  mode: "richtext" | "markdown";
  onModeChange: (mode: "richtext" | "markdown") => void;
}

export default function EditorModeSwitcher({ mode, onModeChange }: EditorModeSwitcherProps) {
  return (
    <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-0.5">
      <button
        onClick={() => onModeChange("richtext")}
        className={`px-3 py-1 text-xs rounded-md transition ${
          mode === "richtext" ? "bg-white shadow text-gray-900" : "text-gray-500 hover:text-gray-700"
        }`}
      >
        Rich Text
      </button>
      <button
        onClick={() => onModeChange("markdown")}
        className={`px-3 py-1 text-xs rounded-md transition ${
          mode === "markdown" ? "bg-white shadow text-gray-900" : "text-gray-500 hover:text-gray-700"
        }`}
      >
        Markdown
      </button>
    </div>
  );
}
```

- [ ] **Step 2: Create Markdown mode component**

Create `frontend/src/components/admin/editor/MarkdownMode.tsx`:

This component provides a raw Markdown textarea on the left and a rendered preview on the right. TipTap has a `generateJSON`/`generateHTML` utility and there are libraries like `turndown` (HTML→MD) and `marked` (MD→HTML) for bidirectional conversion.

```tsx
import { useState, useEffect, useMemo } from "react";

interface MarkdownModeProps {
  value: string;
  onChange: (value: string) => void;
  onImageUpload?: (file: File) => Promise<string>;
}

export default function MarkdownMode({ value, onChange, onImageUpload }: MarkdownModeProps) {
  const [preview, setPreview] = useState("");

  // Lazy-load marked for Markdown → HTML conversion
  useEffect(() => {
    let cancelled = false;
    import("marked").then(({ marked }) => {
      if (!cancelled) {
        const html = marked(value) as string;
        setPreview(html);
      }
    });
    return () => { cancelled = true; };
  }, [value]);

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    if (!onImageUpload) return;
    const files = Array.from(e.dataTransfer.files).filter(f => f.type.startsWith("image/"));
    for (const file of files) {
      const url = await onImageUpload(file);
      const md = `![${file.name}](${url})`;
      onChange(value + "\n" + md);
    }
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    if (!onImageUpload) return;
    const items = Array.from(e.clipboardData.items);
    for (const item of items) {
      if (item.type.startsWith("image/")) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) {
          const url = await onImageUpload(file);
          const md = `![image](${url})`;
          onChange(value + "\n" + md);
        }
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const textarea = e.currentTarget;
    const { selectionStart, selectionEnd } = textarea;
    const selected = value.substring(selectionStart, selectionEnd);

    if (e.ctrlKey || e.metaKey) {
      if (e.key === "b") {
        e.preventDefault();
        const wrapped = `**${selected || "bold"}**`;
        onChange(value.substring(0, selectionStart) + wrapped + value.substring(selectionEnd));
      } else if (e.key === "i") {
        e.preventDefault();
        const wrapped = `*${selected || "italic"}*`;
        onChange(value.substring(0, selectionStart) + wrapped + value.substring(selectionEnd));
      } else if (e.key === "k") {
        e.preventDefault();
        const wrapped = `[${selected || "text"}](url)`;
        onChange(value.substring(0, selectionStart) + wrapped + value.substring(selectionEnd));
      }
    }
  };

  return (
    <div className="flex gap-4 h-full">
      {/* Editor */}
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onDrop={handleDrop}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        className="flex-1 font-mono text-sm p-4 border rounded-lg resize-none focus:ring-2 focus:ring-blue-500 focus:outline-none"
        placeholder="Write Markdown here..."
      />
      {/* Preview */}
      <div
        className="flex-1 p-4 border rounded-lg overflow-auto prose prose-sm max-w-none"
        dangerouslySetInnerHTML={{ __html: preview }}
      />
    </div>
  );
}
```

- [ ] **Step 3: Add `marked` dependency**

```bash
cd /home/dev/impress/frontend && pnpm add marked && pnpm add -D @types/marked
```

- [ ] **Step 4: Add `turndown` for HTML→Markdown conversion**

```bash
cd /home/dev/impress/frontend && pnpm add turndown && pnpm add -D @types/turndown
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/admin/editor/EditorModeSwitcher.tsx frontend/src/components/admin/editor/MarkdownMode.tsx frontend/package.json frontend/pnpm-lock.yaml
git commit -m "feat(1.5.1): Markdown editor mode with live preview, shortcuts, image drag/drop"
```

### Task 22: Integrate mode switcher into article editor (1.5.1)

**Files:**
- Modify: `frontend/src/pages/admin/articles/editor/page.tsx`

- [ ] **Step 1: Add mode state and switcher to article editor**

In the article editor:

1. Add state: `const [editorMode, setEditorMode] = useState<"richtext" | "markdown">("richtext")`
2. Add state for markdown content: `const [markdownContent, setMarkdownContent] = useState({ zh: "", en: "" })`
3. Place `EditorModeSwitcher` in the editor toolbar area
4. When switching from richtext to markdown: use `turndown` to convert TipTap HTML to Markdown
5. When switching from markdown to richtext: use `marked` to convert Markdown to HTML and set it in TipTap editor
6. Show MarkdownMode or TipTap editor based on current mode

Key conversion code:
```tsx
import TurndownService from "turndown";
import { marked } from "marked";

const turndown = new TurndownService();

const handleModeChange = (newMode: "richtext" | "markdown") => {
  if (newMode === "markdown" && editorMode === "richtext") {
    // Convert current TipTap content to Markdown
    const html = editor?.getHTML() ?? "";
    setMarkdownContent(prev => ({ ...prev, [currentLang]: turndown.turndown(html) }));
  } else if (newMode === "richtext" && editorMode === "markdown") {
    // Convert Markdown back to HTML and set in TipTap
    const html = marked(markdownContent[currentLang]) as string;
    editor?.commands.setContent(html);
  }
  setEditorMode(newMode);
};
```

- [ ] **Step 2: Run lint and type-check**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/admin/articles/editor/
git commit -m "feat(1.5.1): integrate richtext/markdown mode switching into article editor"
```

### Task 23: Code block syntax highlighting (1.5.4)

**Files:**
- Modify: `frontend/src/components/admin/editor/extension-groups.ts`

- [ ] **Step 1: Add code highlight extension to TipTap**

TipTap has `@tiptap/extension-code-block-lowlight` for syntax highlighting. Check if already installed:

```bash
cd /home/dev/impress/frontend && grep "lowlight\|highlight.js\|shiki" package.json
```

If not installed:
```bash
pnpm add @tiptap/extension-code-block-lowlight lowlight
```

In `extension-groups.ts`, replace the basic `CodeBlock` extension with `CodeBlockLowlight`:

```tsx
import { lowlight } from "lowlight";
import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";

// Replace CodeBlock with:
CodeBlockLowlight.configure({ lowlight })
```

- [ ] **Step 2: Add highlight.js CSS**

Add to the editor component or global styles:
```css
@import "highlight.js/styles/github.css";
```

- [ ] **Step 3: Run lint and type-check**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/admin/editor/ frontend/package.json frontend/pnpm-lock.yaml
git commit -m "feat(1.5.4): code block syntax highlighting with lowlight in TipTap editor"
```

---

## Chunk 6: Misc Enhancements (Tasks 1.6.1 — 1.6.4)

### Task 24: Image auto-optimization (1.6.1)

**Files:**
- Modify: `backend/internal/handler/media/handler.go`

- [ ] **Step 1: Add WebP conversion on upload**

In the media upload handler, after saving the original file, generate a WebP variant using Go's `image` package + `golang.org/x/image/webp` (or a library like `github.com/chai2010/webp`).

```bash
cd /home/dev/impress/backend && go get github.com/chai2010/webp
```

After the file is saved, add an async goroutine:
```go
go func() {
    if strings.HasPrefix(mimeType, "image/") && mimeType != "image/webp" {
        generateWebP(savedFilePath)
        generateThumbnail(savedFilePath, 300) // 300px width thumbnail
    }
}()
```

Implementation of `generateWebP` and `generateThumbnail` as helper functions in the media handler package.

- [ ] **Step 2: Commit**

```bash
git add backend/internal/handler/media/ backend/go.mod backend/go.sum
git commit -m "feat(1.6.1): auto-generate WebP and thumbnail on image upload"
```

### Task 25: Article import/export (1.6.2)

**Files:**
- Create: `backend/internal/handler/article/import_export.go`

- [ ] **Step 1: Add export endpoint**

In `backend/internal/handler/article/import_export.go`:

```go
// AdminExportMarkdown handles GET /admin/articles/:id/export
// Returns the article as a Markdown file with YAML front matter.
func (h *Handler) AdminExportMarkdown(c *gin.Context) {
    // Fetch article, format as:
    // ---
    // title: zhTitle
    // slug: slug
    // date: publishedAt
    // categories: [...]
    // tags: [...]
    // ---
    // zhBody content
}

// AdminImportMarkdown handles POST /admin/articles/import
// Accepts multipart form with .md files. Parses front-matter metadata.
func (h *Handler) AdminImportMarkdown(c *gin.Context) {
    // Parse uploaded .md files
    // Extract YAML front-matter for metadata
    // Create draft articles
}
```

- [ ] **Step 2: Register routes**

Add to article handler route registration:
```go
admin.GET("/articles/:id/export", h.AdminExportMarkdown)
admin.POST("/articles/import", h.AdminImportMarkdown)
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/article/
git commit -m "feat(1.6.2): article Markdown import/export endpoints"
```

### Task 26: Analytics enhancement (1.6.3)

**Files:**
- Modify: `backend/internal/handler/analytics/handler.go`
- Modify: `backend/internal/model/page_view.go` (if exists)

- [ ] **Step 1: Add UV tracking and referer**

Add fields to PageView model:
```go
VisitorID string `json:"visitorId" gorm:"size:64;index"` // hashed IP or fingerprint
Referer   string `json:"referer" gorm:"size:500"`
```

Migration:
```sql
-- +goose Up
ALTER TABLE page_views ADD COLUMN IF NOT EXISTS visitor_id VARCHAR(64);
ALTER TABLE page_views ADD COLUMN IF NOT EXISTS referer VARCHAR(500);
CREATE INDEX IF NOT EXISTS idx_pageview_visitor ON page_views(visitor_id);
```

- [ ] **Step 2: Update analytics summary endpoint**

Add UV (unique visitors) count to the analytics summary:
```go
// UV: COUNT(DISTINCT visitor_id) for the given time range
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/analytics/ backend/internal/model/ backend/internal/db/migrations/
git commit -m "feat(1.6.3): analytics enhancement — UV tracking, referer recording"
```

### Task 27: Custom 404 page (1.6.4)

**Files:**
- Modify: `backend/cmd/server/main.go` (404 handler)
- Create: `frontend/src/pages/not-found/page.tsx` (if not exists)

- [ ] **Step 1: Verify NotFound page exists**

Check if `frontend/src/pages/not-found/` exists and has a component. If it does, enhance it. If not, create a simple one.

- [ ] **Step 2: Add admin-configurable 404 content**

Store custom 404 content in content_documents (page_key="404"). Serve it via the SPA meta renderer when the path doesn't match any known route.

In `main.go` SPA fallback, if the path is not a known static page or API route, and a content_documents entry with page_key="404" exists with published config, inject that config's title/description into the meta.

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go frontend/src/pages/not-found/
git commit -m "feat(1.6.4): configurable 404 page with CMS-driven content"
```

- [ ] **Step 4: Run full verification**

```bash
cd /home/dev/impress && pnpm lint && pnpm type-check
cd /home/dev/impress/backend && go vet ./... && go build -o server ./cmd/server/
```

Expected: All pass. Phase 1 complete.

---

## Verification Checklist

After all chunks are complete, verify:

- [ ] `cd /home/dev/impress && pnpm lint && pnpm type-check` — PASS
- [ ] `cd /home/dev/impress/backend && go vet ./...` — PASS
- [ ] `cd /home/dev/impress/backend && go test -v -race ./...` — PASS
- [ ] `cd /home/dev/impress/frontend && pnpm test:run` — PASS
- [ ] Backend starts without error: `cd /home/dev/impress/backend && go build -o server ./cmd/server/`
- [ ] `GET /public/search?q=test` returns valid JSON
- [ ] `POST /public/comments` creates a comment
- [ ] `GET /sitemap.xml` includes hreflang and lastmod
- [ ] `GET /robots.txt` returns valid robots.txt
- [ ] Article editor shows richtext/markdown mode switcher
- [ ] Schedule publish modal opens from article editor
