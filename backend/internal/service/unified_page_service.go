package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

var (
	ErrPageVersionConflict    = errors.New("draft version conflict")
	ErrUnifiedPageNotFound    = errors.New("page not found")
	ErrPageVersionRecNotFound = errors.New("version not found")
)

type UnifiedPageService struct {
	pageRepo    repository.UnifiedPageRepository
	versionRepo repository.PageVersionRepository
}

func NewUnifiedPageService(pageRepo repository.UnifiedPageRepository, versionRepo repository.PageVersionRepository) *UnifiedPageService {
	return &UnifiedPageService{pageRepo: pageRepo, versionRepo: versionRepo}
}

// Publish copies DraftConfig → PublishedConfig, creates a version record.
func (s *UnifiedPageService) Publish(ctx context.Context, pageID uint, expectedDraftVersion int, userID uint) error {
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
	return s.pageRepo.UpdatePublished(ctx, page.ID, page.DraftConfig, newVersion, now)
}

// Rollback loads a historical version and publishes it as a new version.
func (s *UnifiedPageService) Rollback(ctx context.Context, pageID uint, targetVersion int, userID uint) error {
	historicalVersion, err := s.versionRepo.FindByPageIDAndVersion(ctx, pageID, targetVersion)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPageVersionRecNotFound, err)
	}

	latestVer, err := s.versionRepo.GetLatestVersion(ctx, pageID)
	if err != nil {
		return fmt.Errorf("get latest version: %w", err)
	}
	newVersion := latestVer + 1

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
	return s.pageRepo.UpdateRollback(
		ctx,
		page.ID,
		historicalVersion.Config,
		page.DraftVersion+1,
		historicalVersion.Config,
		newVersion,
		now,
	)
}

// Unpublish sets page back to draft. Sets PublishedConfig to nil (SQL NULL via NullableJSONMap).
func (s *UnifiedPageService) Unpublish(ctx context.Context, pageID uint) error {
	page, err := s.pageRepo.FindByID(ctx, pageID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnifiedPageNotFound, err)
	}
	return s.pageRepo.ClearPublished(ctx, page.ID)
}
