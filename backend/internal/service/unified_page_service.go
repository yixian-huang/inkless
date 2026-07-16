package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"blotting-consultancy/internal/eventbus"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/pkg/audit"
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

// Publish copies DraftConfig → PublishedConfig, creates a version record.
func (s *UnifiedPageService) Publish(ctx context.Context, pageID uint, expectedDraftVersion int, userID uint) (err error) {
	details := map[string]interface{}{"expected_draft_version": expectedDraftVersion}
	defer func() {
		s.recordAudit(ctx, "content.publish", pageID, err, details)
	}()

	page, err := s.pageRepo.FindByID(ctx, pageID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnifiedPageNotFound, err)
	}
	if page.DraftVersion != expectedDraftVersion {
		return ErrPageVersionConflict
	}

	// Determine next published version
	latestVer, err := s.versionRepo.GetLatestVersion(ctx, pageID)
	if err != nil {
		return fmt.Errorf("get latest version: %w", err)
	}
	newVersion := latestVer + 1
	details["published_version"] = newVersion
	details["draft_version"] = page.DraftVersion

	// Create version record
	version := &model.PageVersion{
		PageID:    pageID,
		Version:   newVersion,
		Config:    page.DraftConfig,
		CreatedBy: userID,
	}
	if err := s.versionRepo.Create(ctx, version); err != nil {
		return fmt.Errorf("create version: %w", err)
	}

	// Update publication fields only so concurrent live route/navigation edits
	// cannot be overwritten by this publish operation.
	now := time.Now()
	if err := s.pageRepo.UpdatePublished(ctx, page.ID, page.DraftConfig, newVersion, now); err != nil {
		return err
	}
	s.publishEvent(eventbus.ContentPublished, page, userID, newVersion, 0)
	return nil
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
