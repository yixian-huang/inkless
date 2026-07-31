package service

import (
	"context"
	"errors"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"gorm.io/gorm"
)

// LoadPublishedGlobalConfig returns published site identity config.
// SSOT is site_configs key "global"; content_documents page_key "global" is legacy fallback only.
func LoadPublishedGlobalConfig(
	ctx context.Context,
	siteCfg repository.SiteConfigRepository,
	legacyDoc repository.ContentDocumentRepository,
) (cfg model.JSONMap, version int, source string) {
	if siteCfg != nil {
		sc, err := siteCfg.FindByKey(ctx, model.SiteConfigKeyGlobal)
		if err == nil && sc != nil && sc.ID != 0 && len(sc.PublishedConfig) > 0 {
			return sc.PublishedConfig, sc.PublishedVersion, "site_config"
		}
	}
	if legacyDoc != nil {
		doc, err := legacyDoc.FindByPageKey(ctx, model.PageKeyGlobal)
		if err == nil && doc != nil && len(doc.PublishedConfig) > 0 {
			return doc.PublishedConfig, doc.PublishedVersion, "content_document"
		}
	}
	return model.JSONMap{}, 0, ""
}

// HydrateSiteGlobalFromLegacy copies content_documents.global into site_configs when the
// SSOT row is missing. Idempotent. Returns the site_config row (hydrated or existing).
func HydrateSiteGlobalFromLegacy(
	ctx context.Context,
	siteCfg repository.SiteConfigRepository,
	legacyDoc repository.ContentDocumentRepository,
) (*model.SiteConfig, error) {
	if siteCfg == nil {
		return nil, errors.New("site config repository required")
	}
	sc, err := siteCfg.FindByKey(ctx, model.SiteConfigKeyGlobal)
	if sc != nil && sc.ID != 0 {
		return sc, nil
	}
	// FindByKey may return zero-value + err on miss; ignore not-found style errors.
	_ = err

	if legacyDoc == nil {
		return nil, gorm.ErrRecordNotFound
	}
	doc, derr := legacyDoc.FindByPageKey(ctx, model.PageKeyGlobal)
	if derr != nil || doc == nil {
		return nil, gorm.ErrRecordNotFound
	}

	draftVer := doc.DraftVersion
	if draftVer < 1 {
		draftVer = 1
	}
	row := &model.SiteConfig{
		Key:              model.SiteConfigKeyGlobal,
		DraftConfig:      doc.DraftConfig,
		DraftVersion:     draftVer,
		PublishedConfig:  doc.PublishedConfig,
		PublishedVersion: doc.PublishedVersion,
	}
	if row.DraftConfig == nil {
		row.DraftConfig = model.JSONMap{}
	}
	if row.PublishedConfig == nil {
		row.PublishedConfig = model.JSONMap{}
	}
	if err := siteCfg.Upsert(ctx, row); err != nil {
		return nil, err
	}
	return siteCfg.FindByKey(ctx, model.SiteConfigKeyGlobal)
}
