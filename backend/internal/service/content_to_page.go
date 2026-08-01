package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// ContentToPageResult is one pageKey migration outcome.
type ContentToPageResult struct {
	PageKey     string `json:"pageKey"`
	Slug        string `json:"slug"`
	Action      string `json:"action"` // created | updated | skipped | error
	PageID      uint   `json:"pageId,omitempty"`
	TemplateKey string `json:"templateKey,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ContentToPageMigrator copies content_documents → unified_pages (theme-as-templates T3).
type ContentToPageMigrator struct {
	unifiedRepo repository.UnifiedPageRepository
	contentRepo repository.ContentDocumentRepository
	themeRepo   repository.InstalledThemeRepository
}

// NewContentToPageMigrator builds a migrator.
func NewContentToPageMigrator(
	unifiedRepo repository.UnifiedPageRepository,
	contentRepo repository.ContentDocumentRepository,
	themeRepo repository.InstalledThemeRepository,
) *ContentToPageMigrator {
	return &ContentToPageMigrator{
		unifiedRepo: unifiedRepo,
		contentRepo: contentRepo,
		themeRepo:   themeRepo,
	}
}

// ActiveThemeID returns the active theme id or empty.
func (m *ContentToPageMigrator) ActiveThemeID(ctx context.Context) string {
	if m == nil || m.themeRepo == nil {
		return ""
	}
	t, err := m.themeRepo.FindActive(ctx)
	if err != nil || t == nil {
		return ""
	}
	return t.ThemeID
}

// MigrateHome ensures home Page exists (wraps EnsureHomePage) and optionally force-syncs config.
func (m *ContentToPageMigrator) MigrateHome(ctx context.Context, themeID string, force bool) ContentToPageResult {
	if themeID == "" {
		themeID = m.ActiveThemeID(ctx)
	}
	// Allow ensure when theme known via ShouldEnsureHomePage; if empty, still try product-first key.
	if themeID == "" {
		themeID = builtinthemes.ProductFirst
	}
	existedBefore := false
	if m.unifiedRepo != nil {
		if p, err := m.unifiedRepo.FindBySlug(ctx, "home"); err == nil && p != nil {
			existedBefore = true
		}
	}
	if !ShouldEnsureHomePage(themeID) {
		// Still create a composable home if content exists
		return m.migrateKeyAsComposable(ctx, "home", force)
	}
	hm := NewHomePageMigrator(m.unifiedRepo, m.contentRepo)
	if err := hm.EnsureHomePage(ctx, themeID); err != nil {
		return ContentToPageResult{PageKey: "home", Slug: "home", Action: "error", Message: err.Error()}
	}
	page, err := m.unifiedRepo.FindBySlug(ctx, "home")
	if err != nil || page == nil {
		return ContentToPageResult{PageKey: "home", Action: "error", Message: "home page missing after ensure"}
	}
	res := ContentToPageResult{
		PageKey:     "home",
		Slug:        "home",
		PageID:      page.ID,
		TemplateKey: page.TemplateKey,
		Action:      "created",
	}
	if existedBefore {
		res.Action = "skipped"
		res.Message = "home page already present (use force to overwrite from content_documents)"
	}
	if force {
		if err := m.forceSyncFromContent(ctx, page, model.PageKeyHome); err != nil {
			res.Action = "error"
			res.Message = err.Error()
			return res
		}
		res.Action = "updated"
		res.Message = "force-synced draft+published from content_documents"
		return res
	}
	return res
}

func (m *ContentToPageMigrator) migrateKeyAsComposable(ctx context.Context, pageKey string, force bool) ContentToPageResult {
	// Minimal path when theme does not use template home
	return ContentToPageResult{
		PageKey: pageKey,
		Slug:    pageKey,
		Action:  "skipped",
		Message: "theme does not use template home (product-first/blog-first)",
	}
}

// MigrateAll migrates home (and future keys) for the active or given theme.
func (m *ContentToPageMigrator) MigrateAll(ctx context.Context, themeID string, force bool) []ContentToPageResult {
	if themeID == "" {
		themeID = m.ActiveThemeID(ctx)
	}
	var out []ContentToPageResult
	out = append(out, m.MigrateHome(ctx, themeID, force))

	// Optional: other content_documents with valid page keys → composable pages (no template)
	if m.contentRepo != nil {
		// only home for now; expand later for about/contact when themes declare templates
		_ = builtinthemes.CorporateClassic
	}
	return out
}

func (m *ContentToPageMigrator) forceSyncFromContent(ctx context.Context, page *model.UnifiedPage, key model.PageKey) error {
	cfg := model.JSONMap{}
	if m.contentRepo != nil {
		doc, err := m.contentRepo.FindByPageKey(ctx, key)
		if err == nil && doc != nil {
			if len(doc.DraftConfig) > 0 {
				cfg = cloneHomeJSONMap(model.JSONMap(doc.DraftConfig))
			} else if len(doc.PublishedConfig) > 0 {
				cfg = cloneHomeJSONMap(model.JSONMap(doc.PublishedConfig))
			}
		}
	}
	if len(cfg) == 0 {
		return fmt.Errorf("no content_documents config for %s", key)
	}
	page.DraftConfig = cfg
	page.PublishedConfig = model.NullableJSONMap(cloneHomeJSONMap(cfg))
	if page.DraftVersion < 1 {
		page.DraftVersion = 1
	} else {
		page.DraftVersion++
	}
	if page.PublishedVersion < 1 {
		page.PublishedVersion = 1
	} else {
		page.PublishedVersion++
	}
	page.Status = "published"
	now := time.Now().UTC()
	page.PublishedAt = &now
	if page.Mode == "" {
		page.Mode = model.PageModeTemplate
	}
	return m.unifiedRepo.Update(ctx, page)
}

// SyncContentDocumentFromPage copies page draft/published into content_documents for dual-read fallback.
func SyncContentDocumentFromPage(
	ctx context.Context,
	contentRepo repository.ContentDocumentRepository,
	page *model.UnifiedPage,
	pageKey model.PageKey,
) error {
	if contentRepo == nil || page == nil {
		return nil
	}
	draft := model.JSONMap(page.DraftConfig)
	if draft == nil {
		draft = model.JSONMap{}
	}
	published := model.JSONMap(page.PublishedConfig)
	if published == nil {
		published = model.JSONMap{}
	}

	doc, err := contentRepo.FindByPageKey(ctx, pageKey)
	if err != nil || doc == nil {
		doc = &model.ContentDocument{
			PageKey:          pageKey,
			DraftConfig:      cloneHomeJSONMap(draft),
			DraftVersion:     maxInt(page.DraftVersion, 1),
			PublishedConfig:  cloneHomeJSONMap(published),
			PublishedVersion: page.PublishedVersion,
		}
		return contentRepo.Create(ctx, doc)
	}
	doc.DraftConfig = cloneHomeJSONMap(draft)
	doc.DraftVersion = maxInt(page.DraftVersion, doc.DraftVersion)
	doc.PublishedConfig = cloneHomeJSONMap(published)
	doc.PublishedVersion = maxInt(page.PublishedVersion, doc.PublishedVersion)
	return contentRepo.Update(ctx, doc)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// PageKeyToSlug maps content pageKey to unified page slug (1:1 for MVP).
func PageKeyToSlug(pageKey string) string {
	return strings.TrimSpace(pageKey)
}

// LogMigration prints a one-line summary.
func LogMigration(results []ContentToPageResult) {
	for _, r := range results {
		log.Printf("content→page: %s action=%s pageId=%d %s", r.PageKey, r.Action, r.PageID, r.Message)
	}
}
