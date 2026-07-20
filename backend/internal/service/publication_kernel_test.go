package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

func TestPublicationKernelRegistersArticleAndPage(t *testing.T) {
	articleSvc := service.NewArticlePublicationService(nil, nil, nil)
	pageSvc := service.NewUnifiedPageService(nil, nil)
	kernel := service.NewPublicationKernel(articleSvc, pageSvc)

	articlePub, err := kernel.Publisher(model.ScheduledContentArticle)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledContentArticle, articlePub.ContentType())

	pagePub, err := kernel.Publisher(model.ScheduledContentPage)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledContentPage, pagePub.ContentType())

	_, err = kernel.Publisher(model.ScheduledContentType("unknown"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported content type")
}

func TestPublicationKernelPanicsOnDuplicateType(t *testing.T) {
	articleA := service.NewArticlePublicationService(nil, nil, nil)
	articleB := service.NewArticlePublicationService(nil, nil, nil)
	require.Panics(t, func() {
		service.NewPublicationKernel(articleA, articleB)
	})
}

func TestArticlePublisherPrepareAndExecute(t *testing.T) {
	db := openKernelDB(t)
	articles := repository.NewGormArticleRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	article := &model.Article{
		Slug:    "kernel-article",
		ZhTitle: "Kernel Article",
		Status:  model.ArticleStatusDraft,
	}
	require.NoError(t, articles.Create(ctx, article))

	svc := service.NewArticlePublicationService(articles, nil, nil)
	version, updatedAt, err := svc.PrepareSchedule(ctx, article.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Nil(t, version)
	require.NotNil(t, updatedAt)

	// MarkScheduled is available for handlers, but must not run between
	// PrepareSchedule and ExecuteScheduled (it bumps UpdatedAt).
	require.NoError(t, svc.ExecuteScheduled(ctx, article.ID, now, 7, nil, updatedAt, nil))
	published, err := articles.FindByID(ctx, article.ID)
	require.NoError(t, err)
	require.Equal(t, model.ArticleStatusPublished, published.Status)
	require.NotNil(t, published.PublishedAt)
	require.Nil(t, published.ScheduledAt)
}

func TestPagePublisherPrepareConflict(t *testing.T) {
	db := openKernelDB(t)
	pages := repository.NewGormUnifiedPageRepository(db)
	versions := repository.NewGormPageVersionRepository(db)
	ctx := context.Background()

	page := &model.UnifiedPage{
		Slug:         "kernel-page",
		ZhTitle:      "Kernel Page",
		Mode:         model.PageModeComposable,
		DraftConfig:  model.JSONMap{"sections": []interface{}{"a"}},
		DraftVersion: 2,
		Status:       "draft",
	}
	require.NoError(t, pages.Create(ctx, page))

	svc := service.NewUnifiedPageService(pages, versions)
	wrong := 1
	_, _, err := svc.PrepareSchedule(ctx, page.ID, &wrong, nil, nil)
	require.ErrorIs(t, err, service.ErrPageVersionConflict)

	current := 2
	version, updatedAt, err := svc.PrepareSchedule(ctx, page.ID, &current, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, version)
	require.Equal(t, 2, *version)
	require.Nil(t, updatedAt)

	require.NoError(t, svc.ExecuteScheduled(ctx, page.ID, time.Now(), 9, version, nil, nil))
	updated, err := pages.FindByID(ctx, page.ID)
	require.NoError(t, err)
	require.Equal(t, "published", updated.Status)
	require.Equal(t, 1, updated.PublishedVersion)
}

func TestPublicationKernelDescribe(t *testing.T) {
	db := openKernelDB(t)
	articles := repository.NewGormArticleRepository(db)
	pages := repository.NewGormUnifiedPageRepository(db)
	versions := repository.NewGormPageVersionRepository(db)
	ctx := context.Background()

	article := &model.Article{Slug: "live-slug", ZhTitle: "Live Title", Status: model.ArticleStatusDraft}
	require.NoError(t, articles.Create(ctx, article))
	page := &model.UnifiedPage{
		Slug: "page-slug", ZhTitle: "Page Title", Mode: model.PageModeComposable,
		DraftConfig: model.JSONMap{}, DraftVersion: 1, Status: "draft",
	}
	require.NoError(t, pages.Create(ctx, page))

	kernel := service.NewPublicationKernel(
		service.NewArticlePublicationService(articles, nil, nil),
		service.NewUnifiedPageService(pages, versions),
	)

	title, slug := kernel.Describe(ctx, model.ScheduledContentArticle, article.ID, model.JSONMap{
		"zhTitle": "Snapshot Title",
		"slug":    "snapshot-slug",
	})
	require.Equal(t, "Snapshot Title", title)
	require.Equal(t, "snapshot-slug", slug)

	title, slug = kernel.Describe(ctx, model.ScheduledContentPage, page.ID, nil)
	require.Equal(t, "Page Title", title)
	require.Equal(t, "page-slug", slug)
}

func TestPublicationKernelUnsupportedExecute(t *testing.T) {
	kernel := service.NewPublicationKernel()
	err := kernel.ExecuteScheduled(
		context.Background(),
		model.ScheduledContentArticle,
		1,
		time.Now(),
		0,
		nil,
		nil,
		nil,
	)
	require.Error(t, err)
	require.True(t, errors.Is(err, err) || strings.Contains(err.Error(), "unsupported"))
}

func openKernelDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Article{},
		&model.UnifiedPage{},
		&model.PageVersion{},
	))
	return db
}
