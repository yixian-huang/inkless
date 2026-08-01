package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"github.com/yixian-huang/inkless/backend/internal/db"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
)

func setupContentServiceTest(t *testing.T) (*ContentService, func()) {
	t.Helper()
	database, err := db.Init(db.InitOptions{DSN: ":memory:", LogLevel: logger.Silent})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.ContentDocument{}, &model.ContentVersion{}))

	docRepo := repository.NewGormContentDocumentRepository(database.DB)
	verRepo := repository.NewGormContentVersionRepository(database.DB)
	svc := NewContentService(database.DB, docRepo, verRepo, NewValidationService())
	return svc, func() { database.Close() }
}

func TestContentService_Publish_ProductFirst(t *testing.T) {
	svc, cleanup := setupContentServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	docRepo := repository.NewGormContentDocumentRepository(svc.db)

	require.NoError(t, docRepo.Create(ctx, &model.ContentDocument{
		PageKey:      model.PageKeyHome,
		DraftVersion: 1,
		DraftConfig: model.JSONMap{
			"hero": map[string]interface{}{
				"title": map[string]interface{}{"zh": "产品", "en": "Product"},
				"media": map[string]interface{}{"url": "/h.png", "alt": "h"},
			},
			"features": map[string]interface{}{
				"items": []interface{}{},
			},
		},
	}))

	result, err := svc.Publish(ctx, model.PageKeyHome, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, result.PublishedVersion)

	doc, err := docRepo.FindByPageKey(ctx, model.PageKeyHome)
	require.NoError(t, err)
	assert.Equal(t, 1, doc.PublishedVersion)
	assert.NotEmpty(t, doc.PublishedConfig)
}

func TestContentService_Publish_VersionMismatch(t *testing.T) {
	svc, cleanup := setupContentServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	docRepo := repository.NewGormContentDocumentRepository(svc.db)

	require.NoError(t, docRepo.Create(ctx, &model.ContentDocument{
		PageKey:      model.PageKeyHome,
		DraftVersion: 2,
		DraftConfig: model.JSONMap{
			"hero": map[string]interface{}{
				"title": map[string]interface{}{"zh": "x", "en": "x"},
			},
			"install": map[string]interface{}{"code": "curl | sh"},
		},
	}))

	_, err := svc.Publish(ctx, model.PageKeyHome, 1, 1)
	assert.ErrorIs(t, err, ErrVersionMismatch)
}

func TestContentService_Publish_MediaRefBlocks(t *testing.T) {
	svc, cleanup := setupContentServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	docRepo := repository.NewGormContentDocumentRepository(svc.db)

	require.NoError(t, docRepo.Create(ctx, &model.ContentDocument{
		PageKey:      model.PageKeyHome,
		DraftVersion: 1,
		DraftConfig: model.JSONMap{
			"hero": map[string]interface{}{
				"media": map[string]interface{}{
					"url": "/x.png",
					"alt": map[string]interface{}{"zh": "中", "en": "En"},
				},
			},
			"showcase": map[string]interface{}{},
		},
	}))

	_, err := svc.Publish(ctx, model.PageKeyHome, 1, 1)
	assert.ErrorIs(t, err, ErrCannotPublish)
}
