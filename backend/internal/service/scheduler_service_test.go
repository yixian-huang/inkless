package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/yixian-huang/inkless/backend/internal/eventbus"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

type schedulerHarness struct {
	db        *gorm.DB
	jobs      repository.ScheduledPublishJobRepository
	articles  repository.ArticleRepository
	pages     repository.UnifiedPageRepository
	versions  repository.PageVersionRepository
	scheduler *service.SchedulerService
	published int
}

func newSchedulerHarness(t *testing.T) *schedulerHarness {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Article{},
		&model.UnifiedPage{},
		&model.PageVersion{},
		&model.ScheduledPublishJob{},
	))

	bus := eventbus.New()
	h := &schedulerHarness{
		db:       db,
		jobs:     repository.NewGormScheduledPublishJobRepository(db),
		articles: repository.NewGormArticleRepository(db),
		pages:    repository.NewGormUnifiedPageRepository(db),
		versions: repository.NewGormPageVersionRepository(db),
	}
	bus.Subscribe(eventbus.ContentPublished, eventbus.SyncHandler(func(e eventbus.Event) {
		h.published++
	}))
	articleSvc := service.NewArticlePublicationService(h.articles, nil, bus)
	pageSvc := service.NewUnifiedPageService(h.pages, h.versions, bus)
	h.scheduler = service.NewSchedulerService(h.jobs, articleSvc, pageSvc)
	return h
}

func TestSchedulerRunDuePublishesArticleAndPreventsDuplicateTicks(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	article := &model.Article{Slug: "scheduled-article", ZhTitle: "Scheduled", Status: model.ArticleStatusDraft}
	require.NoError(t, h.articles.Create(ctx, article))
	job, err := h.scheduler.Schedule(ctx, model.ScheduledContentArticle, article.ID, now.Add(-time.Minute), nil, nil, 11)
	require.NoError(t, err)

	count, err := h.scheduler.RunDue(ctx, now)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = h.scheduler.RunDue(ctx, now)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	updated, err := h.articles.FindByID(ctx, article.ID)
	require.NoError(t, err)
	require.Equal(t, model.ArticleStatusPublished, updated.Status)
	require.NotNil(t, updated.PublishedAt)
	require.True(t, updated.PublishedAt.Equal(now))
	require.Nil(t, updated.ScheduledAt)
	require.Equal(t, 1, h.published)

	updatedJob, err := h.jobs.FindByID(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledJobSucceeded, updatedJob.Status)
	require.Equal(t, 1, updatedJob.Attempts)
	require.NotNil(t, updatedJob.SucceededAt)
}

func TestSchedulerCancelPreventsPublication(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	article := &model.Article{Slug: "cancelled-article", ZhTitle: "Cancelled", Status: model.ArticleStatusDraft}
	require.NoError(t, h.articles.Create(ctx, article))
	job, err := h.scheduler.Schedule(ctx, model.ScheduledContentArticle, article.ID, now.Add(time.Hour), nil, nil, 11)
	require.NoError(t, err)

	cancelled, err := h.scheduler.Cancel(ctx, job.ID, 12, now)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledJobCancelled, cancelled.Status)

	count, err := h.scheduler.RunDue(ctx, now.Add(2*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 0, count)

	updated, err := h.articles.FindByID(ctx, article.ID)
	require.NoError(t, err)
	require.Equal(t, model.ArticleStatusDraft, updated.Status)
	require.Nil(t, updated.ScheduledAt)
}

func TestSchedulerFailedAttemptRecordsRetryMetadata(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	article := &model.Article{
		Slug:    "deleted-before-publish",
		ZhTitle: "Deleted Before Publish",
		Status:  model.ArticleStatusDraft,
	}
	require.NoError(t, h.articles.Create(ctx, article))
	job := &model.ScheduledPublishJob{
		ContentType: model.ScheduledContentArticle,
		ContentID:   article.ID,
		Status:      model.ScheduledJobPending,
		ScheduledAt: now.Add(-time.Minute),
		MaxAttempts: 3,
	}
	require.NoError(t, h.jobs.Schedule(ctx, job))
	require.NoError(t, h.db.Delete(&model.Article{}, article.ID).Error)

	count, err := h.scheduler.RunDue(ctx, now)
	require.Error(t, err)
	require.Equal(t, 0, count)

	updatedJob, err := h.jobs.FindByID(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledJobPending, updatedJob.Status)
	require.Equal(t, 1, updatedJob.Attempts)
	require.NotEmpty(t, updatedJob.LastError)
	require.NotNil(t, updatedJob.LastAttemptAt)
	require.NotNil(t, updatedJob.LastErrorAt)
	require.True(t, updatedJob.ScheduledAt.After(now))
}

func TestSchedulerRunDuePublishesUnifiedPageWithVersion(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	page := &model.UnifiedPage{
		Slug:         "scheduled-page",
		ZhTitle:      "Scheduled Page",
		Mode:         model.PageModeComposable,
		DraftConfig:  model.JSONMap{"sections": []interface{}{"hero"}},
		DraftVersion: 1,
		Status:       "draft",
	}
	require.NoError(t, h.pages.Create(ctx, page))
	expectedVersion := page.DraftVersion
	_, err := h.scheduler.Schedule(
		ctx,
		model.ScheduledContentPage,
		page.ID,
		now.Add(-time.Minute),
		&expectedVersion,
		nil,
		22,
	)
	require.NoError(t, err)

	count, err := h.scheduler.RunDue(ctx, now)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	updated, err := h.pages.FindByID(ctx, page.ID)
	require.NoError(t, err)
	require.Equal(t, "published", updated.Status)
	require.Equal(t, 1, updated.PublishedVersion)
	require.Equal(t, model.NullableJSONMap(model.JSONMap{"sections": []interface{}{"hero"}}), updated.PublishedConfig)

	versions, total, err := h.versions.ListByPageID(ctx, page.ID, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, versions, 1)
	require.Equal(t, 1, versions[0].Version)
	require.Equal(t, uint(22), versions[0].CreatedBy)
}

func TestSchedulerPublishesArticleSnapshotWithoutChangingLiveContentEarly(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-24 * time.Hour)

	article := &model.Article{
		Slug:          "old-slug",
		ZhTitle:       "Old title",
		ZhBody:        "Old body",
		Status:        model.ArticleStatusPublished,
		PublishedAt:   &publishedAt,
		AllowComments: true,
	}
	require.NoError(t, h.articles.Create(ctx, article))
	payload := model.JSONMap{
		"slug":          "new-slug",
		"zhTitle":       "New title",
		"zhBody":        "New body",
		"categoryIds":   []interface{}{},
		"tagIds":        []interface{}{},
		"allowComments": true,
	}
	_, err := h.scheduler.Schedule(
		ctx,
		model.ScheduledContentArticle,
		article.ID,
		now.Add(-time.Minute),
		nil,
		payload,
		11,
	)
	require.NoError(t, err)

	beforeDue, err := h.articles.FindByID(ctx, article.ID)
	require.NoError(t, err)
	require.Equal(t, model.ArticleStatusPublished, beforeDue.Status)
	require.Equal(t, "Old title", beforeDue.ZhTitle)
	// Job repo projects scheduled_at onto the content row (without going
	// through ContentPublisher.MarkScheduled, which would Save() and bump
	// UpdatedAt). Live title/body stay on the old snapshot until fire time.
	require.NotNil(t, beforeDue.ScheduledAt)
	require.Equal(t, "Old body", beforeDue.ZhBody)

	count, err := h.scheduler.RunDue(ctx, now)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	afterDue, err := h.articles.FindByID(ctx, article.ID)
	require.NoError(t, err)
	require.Equal(t, "new-slug", afterDue.Slug)
	require.Equal(t, "New title", afterDue.ZhTitle)
	require.Equal(t, "New body", afterDue.ZhBody)
	require.Nil(t, afterDue.ScheduledAt)
	require.True(t, afterDue.PublishedAt.Equal(now))
}

func TestSchedulerPageVersionConflictDoesNotPublishNewerDraft(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	page := &model.UnifiedPage{
		Slug:         "version-locked-page",
		ZhTitle:      "Version Locked",
		Mode:         model.PageModeComposable,
		DraftConfig:  model.JSONMap{"sections": []interface{}{"v1"}},
		DraftVersion: 1,
		Status:       "draft",
	}
	require.NoError(t, h.pages.Create(ctx, page))
	expectedVersion := 1
	job, err := h.scheduler.Schedule(
		ctx,
		model.ScheduledContentPage,
		page.ID,
		now.Add(-time.Minute),
		&expectedVersion,
		nil,
		22,
	)
	require.NoError(t, err)

	_, err = h.pages.UpdateDraft(ctx, page.ID, 1, model.JSONMap{"sections": []interface{}{"v2"}})
	require.NoError(t, err)

	count, err := h.scheduler.RunDue(ctx, now)
	require.ErrorIs(t, err, service.ErrPageVersionConflict)
	require.Equal(t, 0, count)

	updated, err := h.pages.FindByID(ctx, page.ID)
	require.NoError(t, err)
	require.Equal(t, 0, updated.PublishedVersion)
	require.Nil(t, updated.PublishedConfig)

	updatedJob, err := h.jobs.FindByID(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledJobPending, updatedJob.Status)
	require.Contains(t, updatedJob.LastError, service.ErrPageVersionConflict.Error())
}

func TestSchedulerArticleVersionConflictDoesNotOverwriteLaterEdit(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	article := &model.Article{
		Slug:    "version-locked-article",
		ZhTitle: "Original",
		ZhBody:  "Original body",
		Status:  model.ArticleStatusDraft,
	}
	require.NoError(t, h.articles.Create(ctx, article))
	payload := model.JSONMap{
		"slug":    "version-locked-article",
		"zhTitle": "Scheduled snapshot",
		"zhBody":  "Scheduled body",
	}
	job, err := h.scheduler.Schedule(
		ctx,
		model.ScheduledContentArticle,
		article.ID,
		now.Add(-time.Minute),
		nil,
		payload,
		11,
	)
	require.NoError(t, err)
	require.NotNil(t, job.ExpectedUpdatedAt)

	require.NoError(t, h.db.Model(&model.Article{}).
		Where("id = ?", article.ID).
		Updates(map[string]interface{}{
			"zh_title":   "Edited after scheduling",
			"updated_at": now.Add(time.Minute),
		}).Error)

	count, err := h.scheduler.RunDue(ctx, now)
	require.ErrorIs(t, err, service.ErrArticleVersionConflict)
	require.Equal(t, 0, count)

	updated, err := h.articles.FindByID(ctx, article.ID)
	require.NoError(t, err)
	require.Equal(t, "Edited after scheduling", updated.ZhTitle)
	require.NotEqual(t, "Scheduled body", updated.ZhBody)
	require.NotEqual(t, model.ArticleStatusPublished, updated.Status)

	updatedJob, err := h.jobs.FindByID(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, model.ScheduledJobPending, updatedJob.Status)
	require.Contains(t, updatedJob.LastError, service.ErrArticleVersionConflict.Error())
}

func TestSchedulerArticleRescheduleOnlyPreservesOriginalVersionLock(t *testing.T) {
	h := newSchedulerHarness(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	article := &model.Article{
		Slug:    "reschedule-version-lock",
		ZhTitle: "Original",
		Status:  model.ArticleStatusDraft,
	}
	require.NoError(t, h.articles.Create(ctx, article))
	payload := model.JSONMap{
		"slug":    article.Slug,
		"zhTitle": "Scheduled snapshot",
	}
	job, err := h.scheduler.Schedule(
		ctx,
		model.ScheduledContentArticle,
		article.ID,
		now.Add(time.Hour),
		nil,
		payload,
		11,
	)
	require.NoError(t, err)

	require.NoError(t, h.db.Model(&model.Article{}).
		Where("id = ?", article.ID).
		Updates(map[string]interface{}{
			"zh_title":   "Edited after scheduling",
			"updated_at": now.Add(time.Minute),
		}).Error)

	_, err = h.scheduler.Reschedule(ctx, job.ID, now.Add(2*time.Hour), nil, nil, 11)
	require.ErrorIs(t, err, service.ErrArticleVersionConflict)

	replacementPayload := model.JSONMap{
		"slug":    article.Slug,
		"zhTitle": "Replacement snapshot",
	}
	rescheduled, err := h.scheduler.Reschedule(
		ctx,
		job.ID,
		now.Add(2*time.Hour),
		nil,
		replacementPayload,
		11,
	)
	require.NoError(t, err)
	require.NotNil(t, rescheduled.ExpectedUpdatedAt)
	require.Equal(t, "Replacement snapshot", rescheduled.PublishPayload["zhTitle"])
}
