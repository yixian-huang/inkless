package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/eventbus"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/pkg/audit"
)

var (
	ErrPageVersionConflict    = errors.New("draft version conflict")
	ErrUnifiedPageNotFound    = errors.New("page not found")
	ErrPageVersionRecNotFound = errors.New("version not found")
)

type UnifiedPageService struct {
	pageRepo    repository.UnifiedPageRepository
	versionRepo repository.PageVersionRepository
	eventBus    eventbus.EventBus
	auditWriter audit.Writer
}

func NewUnifiedPageService(
	pageRepo repository.UnifiedPageRepository,
	versionRepo repository.PageVersionRepository,
	eventBuses ...eventbus.EventBus,
) *UnifiedPageService {
	var bus eventbus.EventBus
	if len(eventBuses) > 0 {
		bus = eventBuses[0]
	}
	return &UnifiedPageService{pageRepo: pageRepo, versionRepo: versionRepo, eventBus: bus}
}

// WithAuditWriter enables best-effort audit persistence for service-owned
// publish lifecycle operations.
func (s *UnifiedPageService) WithAuditWriter(writer audit.Writer) *UnifiedPageService {
	s.auditWriter = writer
	return s
}

// HandlesAudit reports whether lifecycle audit is owned by this service.
func (s *UnifiedPageService) HandlesAudit() bool {
	return s != nil && s.auditWriter != nil
}

// Ensure UnifiedPageService satisfies ContentPublisher.
var _ ContentPublisher = (*UnifiedPageService)(nil)

func (s *UnifiedPageService) ContentType() model.ScheduledContentType {
	return model.ScheduledContentPage
}

// PrepareSchedule locks the current draftVersion as the job concurrency token.
func (s *UnifiedPageService) PrepareSchedule(
	ctx context.Context,
	contentID uint,
	expectedVersion *int,
	_ *time.Time,
	_ model.JSONMap,
) (*int, *time.Time, error) {
	if s == nil {
		return nil, nil, errors.New("page publication service is not configured")
	}
	page, err := s.pageRepo.FindByID(ctx, contentID)
	if err != nil {
		return nil, nil, err
	}
	resolved := page.DraftVersion
	if expectedVersion != nil {
		resolved = *expectedVersion
	}
	if resolved != page.DraftVersion {
		return nil, nil, ErrPageVersionConflict
	}
	return &resolved, nil, nil
}

// ExecuteScheduled publishes a page for a claimed schedule job.
func (s *UnifiedPageService) ExecuteScheduled(
	ctx context.Context,
	contentID uint,
	_ time.Time,
	actorID uint,
	expectedVersion *int,
	_ *time.Time,
	_ model.JSONMap,
) error {
	if s == nil {
		return errors.New("page publication service is not configured")
	}
	page, err := s.pageRepo.FindByID(ctx, contentID)
	if err != nil {
		return err
	}
	// Idempotent: already live with no pending schedule marker.
	if page.Status == "published" && page.PublishedAt != nil && page.ScheduledAt == nil {
		return nil
	}
	if expectedVersion == nil {
		return errors.New("scheduled page job has no expected version")
	}
	return s.PublishScheduled(ctx, contentID, *expectedVersion, actorID)
}

// MarkScheduled sets page schedule metadata used by admin UI / status filters.
func (s *UnifiedPageService) MarkScheduled(ctx context.Context, contentID uint, scheduledAt time.Time) error {
	_, err := s.Schedule(ctx, contentID, scheduledAt)
	return err
}

// ClearSchedule clears page schedule metadata after job cancel.
func (s *UnifiedPageService) ClearSchedule(ctx context.Context, contentID uint) error {
	_, err := s.CancelSchedule(ctx, contentID)
	return err
}

// Describe returns the live page title/slug (pages do not snapshot title in payload).
func (s *UnifiedPageService) Describe(ctx context.Context, contentID uint, _ model.JSONMap) (title, slug string) {
	if s == nil || s.pageRepo == nil {
		return "", ""
	}
	page, err := s.pageRepo.FindByID(ctx, contentID)
	if err != nil {
		return "", ""
	}
	return page.ZhTitle, page.Slug
}

// Publish copies DraftConfig → PublishedConfig, creates a version record.
func (s *UnifiedPageService) Publish(ctx context.Context, pageID uint, expectedDraftVersion int, userID uint) (err error) {
	return s.publish(ctx, pageID, expectedDraftVersion, userID, false)
}

func (s *UnifiedPageService) PublishScheduled(
	ctx context.Context,
	pageID uint,
	expectedDraftVersion int,
	userID uint,
) error {
	return s.publish(ctx, pageID, expectedDraftVersion, userID, true)
}

func (s *UnifiedPageService) publish(
	ctx context.Context,
	pageID uint,
	expectedDraftVersion int,
	userID uint,
	requireSchedule bool,
) (err error) {
	details := map[string]interface{}{"expected_draft_version": expectedDraftVersion}
	recordAudit := true
	if requireSchedule {
		details["scheduled"] = true
	}
	defer func() {
		if recordAudit {
			s.recordAudit(ctx, "content.publish", pageID, err, details)
		}
	}()

	now := time.Now()
	page, newVersion, didPublish, err := s.pageRepo.PublishDraft(
		ctx,
		pageID,
		expectedDraftVersion,
		userID,
		now,
		requireSchedule,
	)
	if errors.Is(err, repository.ErrUnifiedPageDraftVersionConflict) {
		return ErrPageVersionConflict
	}
	if err != nil {
		return err
	}
	details["published_version"] = newVersion
	details["draft_version"] = page.DraftVersion
	if didPublish {
		s.publishEvent(eventbus.ContentPublished, page, userID, newVersion, 0)
	} else {
		recordAudit = false
	}
	return nil
}

func (s *UnifiedPageService) Schedule(ctx context.Context, pageID uint, scheduledAt time.Time) (*model.UnifiedPage, error) {
	page, err := s.pageRepo.FindByID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnifiedPageNotFound, err)
	}
	if page.Status != "published" {
		page.Status = "scheduled"
	}
	page.ScheduledAt = &scheduledAt
	if err := s.pageRepo.Update(ctx, page); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *UnifiedPageService) CancelSchedule(ctx context.Context, pageID uint) (*model.UnifiedPage, error) {
	page, err := s.pageRepo.FindByID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnifiedPageNotFound, err)
	}
	if page.Status == "scheduled" {
		page.Status = "draft"
	}
	page.ScheduledAt = nil
	if err := s.pageRepo.Update(ctx, page); err != nil {
		return nil, err
	}
	return page, nil
}

// Rollback loads a historical version and publishes it as a new version.
func (s *UnifiedPageService) Rollback(ctx context.Context, pageID uint, targetVersion int, userID uint) (err error) {
	details := map[string]interface{}{"source_version": targetVersion}
	defer func() {
		s.recordAudit(ctx, "content.rollback", pageID, err, details)
	}()

	historicalVersion, err := s.versionRepo.FindByPageIDAndVersion(ctx, pageID, targetVersion)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPageVersionRecNotFound, err)
	}

	latestVer, err := s.versionRepo.GetLatestVersion(ctx, pageID)
	if err != nil {
		return fmt.Errorf("get latest version: %w", err)
	}
	newVersion := latestVer + 1
	details["published_version"] = newVersion

	// Create new version record from historical config
	version := &model.PageVersion{
		PageID:    pageID,
		Version:   newVersion,
		Config:    historicalVersion.Config,
		CreatedBy: userID,
	}
	if err := s.versionRepo.Create(ctx, version); err != nil {
		return fmt.Errorf("create rollback version: %w", err)
	}

	// Update both draft and published
	page, err := s.pageRepo.FindByID(ctx, pageID)
	if err != nil {
		return fmt.Errorf("find page: %w", err)
	}

	now := time.Now()
	if err := s.pageRepo.UpdateRollback(
		ctx,
		page.ID,
		historicalVersion.Config,
		page.DraftVersion+1,
		historicalVersion.Config,
		newVersion,
		now,
	); err != nil {
		return err
	}
	s.publishEvent(eventbus.ContentRolledBack, page, userID, newVersion, targetVersion)
	return nil
}

// Unpublish sets page back to draft and returns the affected page so callers can
// invalidate slug-based caches without owning a separate pre-flight lookup.
func (s *UnifiedPageService) Unpublish(
	ctx context.Context,
	pageID uint,
	userIDs ...uint,
) (page *model.UnifiedPage, err error) {
	details := make(map[string]interface{})
	defer func() {
		s.recordAudit(ctx, "content.unpublish", pageID, err, details)
	}()

	page, err = s.pageRepo.FindByID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnifiedPageNotFound, err)
	}
	if err := s.pageRepo.ClearPublished(ctx, page.ID); err != nil {
		return nil, err
	}
	var actorID uint
	if len(userIDs) > 0 {
		actorID = userIDs[0]
	}
	details["published_version"] = page.PublishedVersion
	s.publishEvent(eventbus.ContentUnpublished, page, actorID, page.PublishedVersion, 0)
	return page, nil
}

func (s *UnifiedPageService) recordAudit(
	ctx context.Context,
	action string,
	pageID uint,
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
		Action:   action,
		Actor:    metadata.ActorLabel(),
		Resource: fmt.Sprintf("pages:%d", pageID),
		Result:   result,
		Details:  audit.AddMetadata(details, metadata),
	})
}

func (s *UnifiedPageService) publishEvent(eventType string, page *model.UnifiedPage, actorID uint, version, sourceVersion int) {
	if s.eventBus == nil || page == nil {
		return
	}
	s.eventBus.Publish(eventbus.Event{
		Type: eventType,
		Payload: eventbus.ContentEventPayload{
			ContentType:   "page",
			ContentID:     page.ID,
			Slug:          page.Slug,
			Title:         page.ZhTitle,
			ActorID:       actorID,
			Version:       version,
			SourceVersion: sourceVersion,
			Action:        eventType,
		},
	})
}
