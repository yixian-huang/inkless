package content

import (
	"context"
	"strings"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/audit"
	"gorm.io/gorm"
)

// ValidationService validates page configs.
type ValidationService interface {
	ValidateConfig(pageKey model.PageKey, config model.JSONMap) *service.ValidationResult
	ValidateConfigWithSlot(pageKey model.PageKey, config model.JSONMap, slot *contentslots.Slot, schemaSource string) *service.ValidationResult
	CanPublish(result *service.ValidationResult) bool
}

// ContentService publishes and rolls back content documents.
type ContentService interface {
	Publish(ctx context.Context, pageKey model.PageKey, expectedDraftVersion int, createdBy uint) (*service.PublishResult, error)
	Rollback(ctx context.Context, pageKey model.PageKey, sourceVersion int, createdBy uint) (*service.RollbackResult, error)
}

// Auditor records content publish/validate events (optional).
type Auditor interface {
	LogPublishSuccess(pageKey string, publishedVersion int, actor string, draftVersion int)
	LogPublishFailure(pageKey string, actor string, reason string, details map[string]interface{})
	LogRollbackSuccess(pageKey string, publishedVersion int, sourceVersion int, actor string)
	LogRollbackFailure(pageKey string, actor string, sourceVersion int, reason string)
	LogValidation(pageKey string, actor string, valid bool, errorCount int, translationIssueCount int)
}

// Handler serves Admin theme-content (content_documents) APIs.
type Handler struct {
	db            *gorm.DB
	docRepo       repository.ContentDocumentRepository
	versionRepo   repository.ContentVersionRepository
	validationSvc ValidationService
	contentSvc    ContentService
	auditLog      Auditor
	publicCache   *cache.Cache
	slots         *contentslots.Resolver
}

// NewHandler constructs a content admin handler.
func NewHandler(
	db *gorm.DB,
	docRepo repository.ContentDocumentRepository,
	versionRepo repository.ContentVersionRepository,
	validationSvc ValidationService,
	contentSvc ContentService,
	auditLog Auditor,
	publicCache *cache.Cache,
) *Handler {
	return &Handler{
		db:            db,
		docRepo:       docRepo,
		versionRepo:   versionRepo,
		validationSvc: validationSvc,
		contentSvc:    contentSvc,
		auditLog:      auditLog,
		publicCache:   publicCache,
	}
}

// WithSlots attaches theme contentSlots resolver (discovery + validate).
func (h *Handler) WithSlots(r *contentslots.Resolver) *Handler {
	if h != nil {
		h.slots = r
	}
	return h
}

// NewHandlerWithLogger is a convenience when using pkg/audit.Logger.
func NewHandlerWithLogger(
	db *gorm.DB,
	docRepo repository.ContentDocumentRepository,
	versionRepo repository.ContentVersionRepository,
	validationSvc ValidationService,
	contentSvc ContentService,
	auditLog *audit.Logger,
	publicCache *cache.Cache,
) *Handler {
	return NewHandler(db, docRepo, versionRepo, validationSvc, contentSvc, auditLog, publicCache)
}

func isValidPageKey(pageKey model.PageKey) bool {
	return pageKey.IsValid() && pageKey != model.PageKeyTheme
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	if err.Error() == "content document not found" || err.Error() == "content version not found" {
		return true
	}
	return strings.Contains(err.Error(), "not found")
}

func invalidateContentCache(c *cache.Cache, pageKey string) {
	if c == nil {
		return
	}
	cache.InvalidatePagePublic(c, pageKey)
}

// emptyDraftResponse for missing documents (agent-friendly first write).
func emptyDraftResponse(pageKey model.PageKey) GetDraftResponse {
	return GetDraftResponse{
		PageKey:          string(pageKey),
		Version:          0,
		Config:           model.JSONMap{},
		PublishedVersion: 0,
		UpdatedAt:        time.Time{},
	}
}
