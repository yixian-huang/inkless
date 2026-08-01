package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"gorm.io/gorm"
)

var (
	// ErrVersionMismatch is returned when expected draft version does not match.
	ErrVersionMismatch = errors.New("draft version mismatch")
	// ErrCannotPublish is returned when validation fails or translation state blocks publish.
	ErrCannotPublish = errors.New("cannot publish: validation failed or required translations missing/stale")
	// ErrVersionNotFound is returned when requested version does not exist.
	ErrVersionNotFound = errors.New("version not found")
	// ErrDocumentNotFound is returned when content document does not exist.
	ErrDocumentNotFound = errors.New("content document not found")
	// ErrMediaRefInvalid is returned when MediaRef leaves are not plain strings.
	ErrMediaRefInvalid = errors.New("media ref fields must be strings")
)

// ContentService provides transactional publish/rollback for content_documents.
type ContentService struct {
	db            *gorm.DB
	docRepo       repository.ContentDocumentRepository
	versionRepo   repository.ContentVersionRepository
	validationSvc *ValidationService
	// optional: resolve theme contentSlots for publish gate
	slotValidate func(ctx context.Context, pageKey model.PageKey, config model.JSONMap) *ValidationResult
}

// NewContentService creates a ContentService.
func NewContentService(
	db *gorm.DB,
	docRepo repository.ContentDocumentRepository,
	versionRepo repository.ContentVersionRepository,
	validationSvc *ValidationService,
) *ContentService {
	return &ContentService{
		db:            db,
		docRepo:       docRepo,
		versionRepo:   versionRepo,
		validationSvc: validationSvc,
	}
}

// WithSlotValidator injects theme-aware validation used on publish.
func (cs *ContentService) WithSlotValidator(fn func(ctx context.Context, pageKey model.PageKey, config model.JSONMap) *ValidationResult) *ContentService {
	if cs != nil {
		cs.slotValidate = fn
	}
	return cs
}

// PublishResult is the outcome of a successful publish.
type PublishResult struct {
	PageKey          model.PageKey
	PublishedVersion int
	PublishedAt      time.Time
}

// RollbackResult is the outcome of a successful rollback.
type RollbackResult struct {
	PageKey          model.PageKey
	PublishedVersion int
	SourceVersion    int
	PublishedAt      time.Time
}

// Publish promotes draft → published with optimistic draft version check.
func (cs *ContentService) Publish(
	ctx context.Context,
	pageKey model.PageKey,
	expectedDraftVersion int,
	createdBy uint,
) (*PublishResult, error) {
	var result *PublishResult

	err := cs.db.Transaction(func(tx *gorm.DB) error {
		txDocRepo := repository.NewGormContentDocumentRepository(tx)
		txVersionRepo := repository.NewGormContentVersionRepository(tx)

		doc, err := txDocRepo.FindByPageKey(ctx, pageKey)
		if err != nil {
			if isNotFound(err) {
				return ErrDocumentNotFound
			}
			return fmt.Errorf("failed to fetch document: %w", err)
		}

		if doc.DraftVersion != expectedDraftVersion {
			return ErrVersionMismatch
		}

		var validationResult *ValidationResult
		if cs.slotValidate != nil {
			validationResult = cs.slotValidate(ctx, pageKey, doc.DraftConfig)
		} else {
			validationResult = cs.validationSvc.ValidateConfig(pageKey, doc.DraftConfig)
		}
		if !cs.validationSvc.CanPublish(validationResult) {
			return ErrCannotPublish
		}

		newPublishedVersion := doc.PublishedVersion + 1
		publishedAt := time.Now().UTC()

		version := &model.ContentVersion{
			PageKey:     pageKey,
			Version:     newPublishedVersion,
			Config:      doc.DraftConfig,
			PublishedAt: publishedAt,
			CreatedBy:   createdBy,
		}
		if err := txVersionRepo.Create(ctx, version); err != nil {
			return fmt.Errorf("failed to create version snapshot: %w", err)
		}

		if err := txDocRepo.UpdatePublished(ctx, pageKey, doc.DraftConfig, newPublishedVersion); err != nil {
			return fmt.Errorf("failed to update published config: %w", err)
		}

		result = &PublishResult{
			PageKey:          pageKey,
			PublishedVersion: newPublishedVersion,
			PublishedAt:      publishedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Rollback creates a new published version from a historical snapshot.
func (cs *ContentService) Rollback(
	ctx context.Context,
	pageKey model.PageKey,
	sourceVersion int,
	createdBy uint,
) (*RollbackResult, error) {
	var result *RollbackResult

	err := cs.db.Transaction(func(tx *gorm.DB) error {
		txDocRepo := repository.NewGormContentDocumentRepository(tx)
		txVersionRepo := repository.NewGormContentVersionRepository(tx)

		sourceVersionRecord, err := txVersionRepo.FindByPageKeyAndVersion(ctx, pageKey, sourceVersion)
		if err != nil {
			if isNotFound(err) {
				return ErrVersionNotFound
			}
			return fmt.Errorf("failed to fetch source version: %w", err)
		}

		doc, err := txDocRepo.FindByPageKey(ctx, pageKey)
		if err != nil {
			if isNotFound(err) {
				return ErrDocumentNotFound
			}
			return fmt.Errorf("failed to fetch document: %w", err)
		}

		newPublishedVersion := doc.PublishedVersion + 1
		publishedAt := time.Now().UTC()

		version := &model.ContentVersion{
			PageKey:     pageKey,
			Version:     newPublishedVersion,
			Config:      sourceVersionRecord.Config,
			PublishedAt: publishedAt,
			CreatedBy:   createdBy,
		}
		if err := txVersionRepo.Create(ctx, version); err != nil {
			return fmt.Errorf("failed to create rollback version: %w", err)
		}

		if err := txDocRepo.UpdatePublished(ctx, pageKey, sourceVersionRecord.Config, newPublishedVersion); err != nil {
			return fmt.Errorf("failed to update published config: %w", err)
		}

		result = &RollbackResult{
			PageKey:          pageKey,
			PublishedVersion: newPublishedVersion,
			SourceVersion:    sourceVersion,
			PublishedAt:      publishedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	msg := err.Error()
	return msg == "content document not found" || msg == "content version not found"
}
