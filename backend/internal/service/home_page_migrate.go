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

// HomePageMigrator ensures slug=home unified pages exist for theme-as-templates (T1/T2).
type HomePageMigrator struct {
	unifiedRepo repository.UnifiedPageRepository
	contentRepo repository.ContentDocumentRepository
}

// NewHomePageMigrator creates a migrator.
func NewHomePageMigrator(
	unifiedRepo repository.UnifiedPageRepository,
	contentRepo repository.ContentDocumentRepository,
) *HomePageMigrator {
	return &HomePageMigrator{unifiedRepo: unifiedRepo, contentRepo: contentRepo}
}

// TemplateKeyForTheme returns the canonical home template key for a theme id.
func TemplateKeyForTheme(themeID string) string {
	themeID = strings.TrimSpace(themeID)
	if themeID == "" {
		return ""
	}
	switch themeID {
	case builtinthemes.ProductFirst:
		return "product-first/home"
	case builtinthemes.BlogFirst:
		return "blog-first/home"
	default:
		return themeID + "/home"
	}
}

// ShouldEnsureHomePage reports whether this theme participates in home-as-Page.
func ShouldEnsureHomePage(themeID string) bool {
	switch strings.TrimSpace(themeID) {
	case builtinthemes.ProductFirst, builtinthemes.BlogFirst:
		return true
	default:
		return false
	}
}

// EnsureHomePage creates or lightly upgrades slug=home for the active theme.
// Does not overwrite published content when publishedVersion > 0 and config non-empty.
func (m *HomePageMigrator) EnsureHomePage(ctx context.Context, themeID string) error {
	if m == nil || m.unifiedRepo == nil {
		return nil
	}
	if !ShouldEnsureHomePage(themeID) {
		return nil
	}
	templateKey := TemplateKeyForTheme(themeID)
	cfg := m.loadHomeConfig(ctx)

	existing, err := m.unifiedRepo.FindBySlug(ctx, "home")
	if err == nil && existing != nil {
		return m.upgradeExisting(ctx, existing, templateKey, cfg)
	}

	now := time.Now().UTC()
	page := &model.UnifiedPage{
		Slug:             "home",
		ZhTitle:          "首页",
		EnTitle:          "Home",
		Mode:             model.PageModeTemplate,
		TemplateKey:      templateKey,
		DraftConfig:      cloneHomeJSONMap(cfg),
		DraftVersion:     1,
		PublishedConfig:  model.NullableJSONMap(cloneHomeJSONMap(cfg)),
		PublishedVersion: 1,
		Status:           "published",
		SortOrder:        0,
		ShowInNav:        true,
		PublishedAt:      &now,
	}
	// Ensure non-nil published config for list filters that check length on some paths
	if len(page.PublishedConfig) == 0 {
		page.PublishedConfig = model.NullableJSONMap{"_templateKey": templateKey}
		page.DraftConfig = model.JSONMap{"_templateKey": templateKey}
	}
	if err := m.unifiedRepo.Create(ctx, page); err != nil {
		return fmt.Errorf("create home page: %w", err)
	}
	log.Printf("theme-as-templates: created unified page home templateKey=%s", templateKey)
	return nil
}

func (m *HomePageMigrator) upgradeExisting(ctx context.Context, page *model.UnifiedPage, templateKey string, cfg model.JSONMap) error {
	changed := false
	if page.TemplateKey == "" && templateKey != "" {
		page.TemplateKey = templateKey
		changed = true
	}
	if page.Mode == "" || (page.Mode == model.PageModeComposable && templateKey != "") {
		page.Mode = model.PageModeTemplate
		changed = true
	}
	// Fill empty published config from content_documents once
	if len(page.PublishedConfig) == 0 && len(cfg) > 0 {
		page.PublishedConfig = model.NullableJSONMap(cloneHomeJSONMap(cfg))
		if page.PublishedVersion == 0 {
			page.PublishedVersion = 1
		}
		if page.Status == "" || page.Status == "draft" {
			page.Status = "published"
		}
		if len(page.DraftConfig) == 0 {
			page.DraftConfig = cloneHomeJSONMap(cfg)
		}
		changed = true
	}
	if page.Status == "published" && page.PublishedVersion == 0 {
		page.PublishedVersion = 1
		changed = true
	}
	if !changed {
		return nil
	}
	if err := m.unifiedRepo.Update(ctx, page); err != nil {
		return fmt.Errorf("upgrade home page: %w", err)
	}
	log.Printf("theme-as-templates: upgraded unified page home id=%d templateKey=%s", page.ID, page.TemplateKey)
	return nil
}

func (m *HomePageMigrator) loadHomeConfig(ctx context.Context) model.JSONMap {
	if m.contentRepo == nil {
		return model.JSONMap{}
	}
	doc, err := m.contentRepo.FindByPageKey(ctx, model.PageKeyHome)
	if err != nil || doc == nil {
		return model.JSONMap{}
	}
	if len(doc.PublishedConfig) > 0 {
		return cloneHomeJSONMap(model.JSONMap(doc.PublishedConfig))
	}
	if len(doc.DraftConfig) > 0 {
		return cloneHomeJSONMap(model.JSONMap(doc.DraftConfig))
	}
	return model.JSONMap{}
}

func cloneHomeJSONMap(in model.JSONMap) model.JSONMap {
	if in == nil {
		return model.JSONMap{}
	}
	out := make(model.JSONMap, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
