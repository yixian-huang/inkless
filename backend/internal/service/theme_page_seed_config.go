package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/builtinthemes"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"gorm.io/gorm"
)

// editorialFirmSeedSlugs is the ordered set of unified pages seeded for editorial-firm.
var editorialFirmSeedSlugs = []string{"home", "about", "services", "contact"}

// editorialFirmPageSeeds maps slug → { "sections": [...] } (parsed once).
var editorialFirmPageSeeds map[string]model.JSONMap

func init() {
	editorialFirmPageSeeds = loadEditorialFirmPageSeeds()
}

func loadEditorialFirmPageSeeds() map[string]model.JSONMap {
	out := make(map[string]model.JSONMap)
	if len(builtinthemes.EditorialFirmSeedsJSON) == 0 {
		log.Printf("Warning: editorial-firm seeds JSON is empty")
		return out
	}
	var raw map[string]model.JSONMap
	if err := json.Unmarshal(builtinthemes.EditorialFirmSeedsJSON, &raw); err != nil {
		log.Printf("Warning: failed to parse editorial-firm seeds JSON: %v", err)
		return out
	}
	for slug, cfg := range raw {
		out[slug] = cfg
	}
	return out
}

// publishedConfigHasSections reports whether PublishedConfig contains a non-empty sections array.
// Missing, null, or empty sections → false (seed may apply).
func publishedConfigHasSections(cfg model.NullableJSONMap) bool {
	if cfg == nil {
		return false
	}
	raw, ok := cfg["sections"]
	if !ok || raw == nil {
		return false
	}
	switch s := raw.(type) {
	case []any:
		return len(s) > 0
	default:
		// Non-array present value: treat as having content to avoid overwrite.
		return true
	}
}

func cloneJSONMap(m model.JSONMap) model.JSONMap {
	if m == nil {
		return model.JSONMap{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		// Fallback shallow copy of top-level keys only.
		out := make(model.JSONMap, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	var out model.JSONMap
	if err := json.Unmarshal(b, &out); err != nil {
		return model.JSONMap{}
	}
	if out == nil {
		return model.JSONMap{}
	}
	return out
}

func titleFromThemeDef(slug string) (zh, en string) {
	defs := BuiltInThemePages[builtinthemes.EditorialFirm]
	for _, d := range defs {
		if d.Slug != slug {
			continue
		}
		if d.Title != nil {
			if v, ok := d.Title["zh"].(string); ok {
				zh = v
			}
			if v, ok := d.Title["en"].(string); ok {
				en = v
			}
		}
		return zh, en
	}
	// Fallback to slug if pages.json missing entry.
	return slug, slug
}

func sortOrderFromThemeDef(slug string) int {
	defs := BuiltInThemePages[builtinthemes.EditorialFirm]
	for _, d := range defs {
		if d.Slug == slug {
			return d.SortOrder
		}
	}
	return 0
}

// ApplyEditorialFirmPageSeeds applies embedded section configs to unified pages
// home|about|services|contact when the page is missing or PublishedConfig has no sections.
// Existing non-empty sections are never overwritten.
func ApplyEditorialFirmPageSeeds(ctx context.Context, repo repository.UnifiedPageRepository) error {
	if repo == nil {
		return errors.New("unified page repository is nil")
	}
	if len(editorialFirmPageSeeds) == 0 {
		return errors.New("editorial-firm page seeds not loaded")
	}

	var errs []error
	for _, slug := range editorialFirmSeedSlugs {
		seedCfg, ok := editorialFirmPageSeeds[slug]
		if !ok || seedCfg == nil {
			errs = append(errs, fmt.Errorf("missing seed config for slug %q", slug))
			continue
		}
		if err := applyEditorialFirmSeedForSlug(ctx, repo, slug, seedCfg); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", slug, err))
		}
	}
	return errors.Join(errs...)
}

func applyEditorialFirmSeedForSlug(
	ctx context.Context,
	repo repository.UnifiedPageRepository,
	slug string,
	seedCfg model.JSONMap,
) error {
	cfg := cloneJSONMap(seedCfg)
	page, err := repo.FindBySlug(ctx, slug)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return createEditorialFirmSeedPage(ctx, repo, slug, cfg)
	}
	if publishedConfigHasSections(page.PublishedConfig) {
		log.Printf("Unified page %s already has published sections, skipping editorial-firm seed", slug)
		return nil
	}
	return updateEditorialFirmSeedPage(ctx, repo, page, cfg)
}

func createEditorialFirmSeedPage(
	ctx context.Context,
	repo repository.UnifiedPageRepository,
	slug string,
	cfg model.JSONMap,
) error {
	zh, en := titleFromThemeDef(slug)
	now := time.Now()
	page := &model.UnifiedPage{
		Slug:             slug,
		ZhTitle:          zh,
		EnTitle:          en,
		Mode:             model.PageModeComposable,
		DraftConfig:      cfg,
		DraftVersion:     1,
		PublishedConfig:  model.NullableJSONMap(cloneJSONMap(cfg)),
		PublishedVersion: 1,
		Status:           "published",
		SortOrder:        sortOrderFromThemeDef(slug),
		ShowInNav:        true,
		PublishedAt:      &now,
	}
	if err := repo.Create(ctx, page); err != nil {
		return err
	}
	log.Printf("Created unified page from editorial-firm seed: %s", slug)
	return nil
}

func updateEditorialFirmSeedPage(
	ctx context.Context,
	repo repository.UnifiedPageRepository,
	page *model.UnifiedPage,
	cfg model.JSONMap,
) error {
	page.DraftConfig = cfg
	page.PublishedConfig = model.NullableJSONMap(cloneJSONMap(cfg))
	if page.DraftVersion < 1 {
		page.DraftVersion = 1
	}
	if page.PublishedVersion < 1 {
		page.PublishedVersion = 1
	}
	page.Status = "published"
	if page.Mode == "" {
		page.Mode = model.PageModeComposable
	}
	if page.PublishedAt == nil {
		now := time.Now()
		page.PublishedAt = &now
	}
	if err := repo.Update(ctx, page); err != nil {
		return err
	}
	log.Printf("Applied editorial-firm seed to empty unified page: %s", page.Slug)
	return nil
}
