package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/eventbus"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/pkg/audit"
)

var ErrArticleVersionConflict = errors.New("article changed after scheduling")

type ArticlePublicationService struct {
	articleRepo   repository.ArticleRepository
	categoryRepo  repository.CategoryRepository
	tagRepo       repository.TagRepository
	searchService *SearchService
	eventBus      eventbus.EventBus
	auditWriter   audit.Writer
}

func NewArticlePublicationService(
	articleRepo repository.ArticleRepository,
	searchService *SearchService,
	eventBus eventbus.EventBus,
) *ArticlePublicationService {
	return &ArticlePublicationService{
		articleRepo:   articleRepo,
		searchService: searchService,
		eventBus:      eventBus,
	}
}

func (s *ArticlePublicationService) WithTaxonomyRepositories(
	categoryRepo repository.CategoryRepository,
	tagRepo repository.TagRepository,
) *ArticlePublicationService {
	s.categoryRepo = categoryRepo
	s.tagRepo = tagRepo
	return s
}

func (s *ArticlePublicationService) WithAuditWriter(writer audit.Writer) *ArticlePublicationService {
	s.auditWriter = writer
	return s
}

// Ensure ArticlePublicationService satisfies ContentPublisher.
var _ ContentPublisher = (*ArticlePublicationService)(nil)

func (s *ArticlePublicationService) ContentType() model.ScheduledContentType {
	return model.ScheduledContentArticle
}

// PrepareSchedule validates payload and captures UpdatedAt as the concurrency token.
func (s *ArticlePublicationService) PrepareSchedule(
	ctx context.Context,
	contentID uint,
	_ *int,
	expectedUpdatedAt *time.Time,
	payload model.JSONMap,
) (*int, *time.Time, error) {
	if s == nil {
		return nil, nil, errors.New("article publication service is not configured")
	}
	article, err := s.articleRepo.FindByID(ctx, contentID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.ValidatePublishPayload(ctx, payload); err != nil {
		return nil, nil, err
	}
	if expectedUpdatedAt != nil && !article.UpdatedAt.Equal(*expectedUpdatedAt) {
		return nil, nil, ErrArticleVersionConflict
	}
	resolvedUpdatedAt := article.UpdatedAt
	return nil, &resolvedUpdatedAt, nil
}

// ExecuteScheduled publishes an article for a claimed schedule job.
func (s *ArticlePublicationService) ExecuteScheduled(
	ctx context.Context,
	contentID uint,
	publishedAt time.Time,
	actorID uint,
	_ *int,
	expectedUpdatedAt *time.Time,
	payload model.JSONMap,
) error {
	if s == nil {
		return errors.New("article publication service is not configured")
	}
	_, err := s.Publish(ctx, contentID, publishedAt, actorID, expectedUpdatedAt, payload)
	return err
}

// MarkScheduled sets article schedule metadata used by admin UI / status filters.
func (s *ArticlePublicationService) MarkScheduled(ctx context.Context, contentID uint, scheduledAt time.Time) error {
	_, err := s.Schedule(ctx, contentID, scheduledAt)
	return err
}

// ClearSchedule clears article schedule metadata after job cancel.
func (s *ArticlePublicationService) ClearSchedule(ctx context.Context, contentID uint) error {
	_, err := s.CancelSchedule(ctx, contentID)
	return err
}

// Describe prefers the scheduled payload snapshot, then falls back to the live row.
func (s *ArticlePublicationService) Describe(ctx context.Context, contentID uint, payload model.JSONMap) (title, slug string) {
	if payloadDecoded, err := decodeScheduledArticlePayload(payload); err == nil && payloadDecoded != nil {
		if payloadDecoded.ZhTitle != nil {
			title = *payloadDecoded.ZhTitle
		}
		if payloadDecoded.Slug != nil {
			slug = *payloadDecoded.Slug
		}
	}
	if title != "" || slug != "" || s == nil || s.articleRepo == nil {
		return title, slug
	}
	if article, err := s.articleRepo.FindByID(ctx, contentID); err == nil {
		return article.ZhTitle, article.Slug
	}
	return title, slug
}

type scheduledArticlePayload struct {
	Slug              *string        `json:"slug"`
	ZhTitle           *string        `json:"zhTitle"`
	EnTitle           *string        `json:"enTitle"`
	ZhBody            *string        `json:"zhBody"`
	EnBody            *string        `json:"enBody"`
	CoverImage        *string        `json:"coverImage"`
	ZhSeoTitle        *string        `json:"zhSeoTitle"`
	EnSeoTitle        *string        `json:"enSeoTitle"`
	ZhMetaDescription *string        `json:"zhMetaDescription"`
	EnMetaDescription *string        `json:"enMetaDescription"`
	OgImage           *string        `json:"ogImage"`
	CategoryIDs       *[]uint        `json:"categoryIds"`
	TagIDs            *[]uint        `json:"tagIds"`
	Author            *string        `json:"author"`
	AutoSummary       *bool          `json:"autoSummary"`
	AllowComments     *bool          `json:"allowComments"`
	Pinned            *bool          `json:"pinned"`
	Visibility        *string        `json:"visibility"`
	Metadata          *model.JSONMap `json:"metadata"`
}

func decodeScheduledArticlePayload(payload model.JSONMap) (*scheduledArticlePayload, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode article publish payload: %w", err)
	}
	var decoded scheduledArticlePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("decode article publish payload: %w", err)
	}
	return &decoded, nil
}

func (s *ArticlePublicationService) ValidatePublishPayload(ctx context.Context, payload model.JSONMap) error {
	decoded, err := decodeScheduledArticlePayload(payload)
	if err != nil || decoded == nil {
		return err
	}
	if decoded.Slug == nil || *decoded.Slug == "" {
		return errors.New("article publish payload requires slug")
	}
	if decoded.ZhTitle == nil || *decoded.ZhTitle == "" {
		return errors.New("article publish payload requires zhTitle")
	}
	if decoded.CategoryIDs != nil {
		if len(*decoded.CategoryIDs) == 0 {
			// Empty is valid and clears the many-to-many association.
		} else if s.categoryRepo == nil {
			return errors.New("category repository is not configured")
		} else {
			items, err := s.categoryRepo.FindByIDs(ctx, *decoded.CategoryIDs)
			if err != nil {
				return err
			}
			if len(items) != len(*decoded.CategoryIDs) {
				return errors.New("article publish payload contains invalid categories")
			}
		}
	}
	if decoded.TagIDs != nil {
		if len(*decoded.TagIDs) == 0 {
			// Empty is valid and clears the many-to-many association.
		} else if s.tagRepo == nil {
			return errors.New("tag repository is not configured")
		} else {
			items, err := s.tagRepo.FindByIDs(ctx, *decoded.TagIDs)
			if err != nil {
				return err
			}
			if len(items) != len(*decoded.TagIDs) {
				return errors.New("article publish payload contains invalid tags")
			}
		}
	}
	return nil
}

func (s *ArticlePublicationService) Schedule(ctx context.Context, articleID uint, scheduledAt time.Time) (*model.Article, error) {
	article, err := s.articleRepo.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if article.Status != model.ArticleStatusPublished {
		article.Status = model.ArticleStatusScheduled
	}
	article.ScheduledAt = &scheduledAt
	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, err
	}
	return article, nil
}

func (s *ArticlePublicationService) CancelSchedule(ctx context.Context, articleID uint) (*model.Article, error) {
	article, err := s.articleRepo.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if article.Status == model.ArticleStatusScheduled {
		article.Status = model.ArticleStatusDraft
	}
	article.ScheduledAt = nil
	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, err
	}
	return article, nil
}

func (s *ArticlePublicationService) Publish(
	ctx context.Context,
	articleID uint,
	publishedAt time.Time,
	actorID uint,
	expectedUpdatedAt *time.Time,
	payload model.JSONMap,
) (article *model.Article, err error) {
	details := map[string]interface{}{"scheduled": true}
	recordAudit := true
	defer func() {
		if recordAudit {
			s.recordAudit(ctx, articleID, err, details)
		}
	}()

	article, err = s.articleRepo.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if article.Status == model.ArticleStatusPublished && article.PublishedAt != nil && article.ScheduledAt == nil {
		recordAudit = false
		return article, nil
	}
	if expectedUpdatedAt != nil && !article.UpdatedAt.Equal(*expectedUpdatedAt) {
		return nil, ErrArticleVersionConflict
	}
	if err := s.applyPublishPayload(ctx, article, payload); err != nil {
		return nil, err
	}

	article.Status = model.ArticleStatusPublished
	article.PublishedAt = &publishedAt
	article.ScheduledAt = nil
	if expectedUpdatedAt == nil {
		return nil, errors.New("scheduled article job has no expected update time")
	}
	if err := s.articleRepo.UpdateScheduledPublication(ctx, article, *expectedUpdatedAt); err != nil {
		if errors.Is(err, repository.ErrArticleVersionConflict) {
			return nil, ErrArticleVersionConflict
		}
		return nil, err
	}
	s.AfterPublish(ctx, article, actorID)
	return article, nil
}

func (s *ArticlePublicationService) applyPublishPayload(
	ctx context.Context,
	article *model.Article,
	payload model.JSONMap,
) error {
	decoded, err := decodeScheduledArticlePayload(payload)
	if err != nil || decoded == nil {
		return err
	}
	if err := s.ValidatePublishPayload(ctx, payload); err != nil {
		return err
	}
	if decoded.Slug != nil {
		article.Slug = *decoded.Slug
	}
	if decoded.ZhTitle != nil {
		article.ZhTitle = *decoded.ZhTitle
	}
	if decoded.EnTitle != nil {
		article.EnTitle = *decoded.EnTitle
	}
	if decoded.ZhBody != nil {
		article.ZhBody = *decoded.ZhBody
	}
	if decoded.EnBody != nil {
		article.EnBody = *decoded.EnBody
	}
	if decoded.CoverImage != nil {
		article.CoverImage = *decoded.CoverImage
	}
	if decoded.ZhSeoTitle != nil {
		article.ZhSeoTitle = *decoded.ZhSeoTitle
	}
	if decoded.EnSeoTitle != nil {
		article.EnSeoTitle = *decoded.EnSeoTitle
	}
	if decoded.ZhMetaDescription != nil {
		article.ZhMetaDescription = *decoded.ZhMetaDescription
	}
	if decoded.EnMetaDescription != nil {
		article.EnMetaDescription = *decoded.EnMetaDescription
	}
	if decoded.OgImage != nil {
		article.OgImage = *decoded.OgImage
	}
	if decoded.Author != nil {
		article.Author = *decoded.Author
	}
	if decoded.AutoSummary != nil {
		article.AutoSummary = *decoded.AutoSummary
	}
	if decoded.AllowComments != nil {
		article.AllowComments = *decoded.AllowComments
	}
	if decoded.Pinned != nil {
		article.Pinned = *decoded.Pinned
	}
	if decoded.Visibility != nil {
		article.Visibility = *decoded.Visibility
	}
	if decoded.Metadata != nil {
		article.Metadata = *decoded.Metadata
	}
	if decoded.CategoryIDs != nil {
		article.Categories = []model.Category{}
		if len(*decoded.CategoryIDs) > 0 {
			items, err := s.categoryRepo.FindByIDs(ctx, *decoded.CategoryIDs)
			if err != nil {
				return err
			}
			article.Categories = items
		}
	}
	if decoded.TagIDs != nil {
		article.Tags = []model.Tag{}
		if len(*decoded.TagIDs) > 0 {
			items, err := s.tagRepo.FindByIDs(ctx, *decoded.TagIDs)
			if err != nil {
				return err
			}
			article.Tags = items
		}
	}
	return nil
}

func (s *ArticlePublicationService) AfterPublish(ctx context.Context, article *model.Article, actorID uint) {
	if article == nil {
		return
	}
	s.indexPublishedArticle(ctx, article)
	if s.eventBus != nil {
		s.eventBus.Publish(eventbus.Event{
			Type: eventbus.ContentPublished,
			Payload: eventbus.ContentEventPayload{
				ContentType: "article",
				ContentID:   article.ID,
				Slug:        article.Slug,
				Title:       article.ZhTitle,
				ActorID:     actorID,
				Action:      eventbus.ContentPublished,
			},
		})
	}
}

func (s *ArticlePublicationService) AfterUnpublish(ctx context.Context, article *model.Article) {
	if s.searchService == nil || article == nil {
		return
	}
	_ = s.searchService.RemoveFromIndex(ctx, "article", article.ID)
}

func (s *ArticlePublicationService) RefreshPublished(ctx context.Context, article *model.Article) {
	s.indexPublishedArticle(ctx, article)
}

func (s *ArticlePublicationService) indexPublishedArticle(ctx context.Context, article *model.Article) {
	if s.searchService == nil || article == nil {
		return
	}
	if article.ZhTitle != "" {
		_ = s.searchService.IndexArticle(ctx, article.ID, "zh", article.ZhTitle, article.ZhBody, article.Slug)
	}
	if article.EnTitle != "" {
		_ = s.searchService.IndexArticle(ctx, article.ID, "en", article.EnTitle, article.EnBody, article.Slug)
	}
}

func (s *ArticlePublicationService) recordAudit(
	ctx context.Context,
	articleID uint,
	operationErr error,
	details map[string]interface{},
) {
	if s.auditWriter == nil {
		return
	}
	metadata := audit.MetadataFromContext(ctx)
	result := "success"
	if operationErr != nil {
		result = "failure"
		details["reason"] = operationErr.Error()
	}
	_ = s.auditWriter.Write(ctx, audit.Event{
		Action:   "content.publish",
		Actor:    metadata.ActorLabel(),
		Resource: fmt.Sprintf("articles:%d", articleID),
		Result:   result,
		Details:  audit.AddMetadata(details, metadata),
	})
}
