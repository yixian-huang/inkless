package content

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/audit"
	appLogger "blotting-consultancy/pkg/logger"
)

// Mock repositories and services
type MockContentDocumentRepository struct {
	mock.Mock
}

func (m *MockContentDocumentRepository) Create(ctx context.Context, doc *model.ContentDocument) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockContentDocumentRepository) FindByPageKey(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
	args := m.Called(ctx, pageKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ContentDocument), args.Error(1)
}

func (m *MockContentDocumentRepository) Update(ctx context.Context, doc *model.ContentDocument) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockContentDocumentRepository) UpdateDraft(ctx context.Context, pageKey model.PageKey, expectedDraftVersion int, draftConfig model.JSONMap) (int, error) {
	args := m.Called(ctx, pageKey, expectedDraftVersion, draftConfig)
	return args.Int(0), args.Error(1)
}

func (m *MockContentDocumentRepository) UpdatePublished(ctx context.Context, pageKey model.PageKey, publishedConfig model.JSONMap, publishedVersion int) error {
	args := m.Called(ctx, pageKey, publishedConfig, publishedVersion)
	return args.Error(0)
}

func (m *MockContentDocumentRepository) List(ctx context.Context) ([]*model.ContentDocument, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ContentDocument), args.Error(1)
}

func (m *MockContentDocumentRepository) Delete(ctx context.Context, pageKey model.PageKey) error {
	args := m.Called(ctx, pageKey)
	return args.Error(0)
}

type MockContentVersionRepository struct {
	mock.Mock
}

func (m *MockContentVersionRepository) Create(ctx context.Context, version *model.ContentVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockContentVersionRepository) FindByID(ctx context.Context, id uint) (*model.ContentVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ContentVersion), args.Error(1)
}

func (m *MockContentVersionRepository) FindByPageKeyAndVersion(ctx context.Context, pageKey model.PageKey, version int) (*model.ContentVersion, error) {
	args := m.Called(ctx, pageKey, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ContentVersion), args.Error(1)
}

func (m *MockContentVersionRepository) ListByPageKey(ctx context.Context, pageKey model.PageKey, offset, limit int) ([]*model.ContentVersion, int64, error) {
	args := m.Called(ctx, pageKey, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*model.ContentVersion), args.Get(1).(int64), args.Error(2)
}

func (m *MockContentVersionRepository) GetLatestVersion(ctx context.Context, pageKey model.PageKey) (int, error) {
	args := m.Called(ctx, pageKey)
	return args.Int(0), args.Error(1)
}

func (m *MockContentVersionRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockValidationService struct {
	mock.Mock
}

func (m *MockValidationService) ValidateConfig(pageKey model.PageKey, config model.JSONMap) *service.ValidationResult {
	args := m.Called(pageKey, config)
	return args.Get(0).(*service.ValidationResult)
}

func (m *MockValidationService) CanPublish(result *service.ValidationResult) bool {
	args := m.Called(result)
	return args.Bool(0)
}

type MockContentService struct {
	mock.Mock
}

func (m *MockContentService) Publish(ctx context.Context, pageKey model.PageKey, expectedDraftVersion int, createdBy uint) (*service.PublishResult, error) {
	args := m.Called(ctx, pageKey, expectedDraftVersion, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PublishResult), args.Error(1)
}

func (m *MockContentService) Rollback(ctx context.Context, pageKey model.PageKey, sourceVersion int, createdBy uint) (*service.RollbackResult, error) {
	args := m.Called(ctx, pageKey, sourceVersion, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.RollbackResult), args.Error(1)
}

func setupTestHandler() (*Handler, *MockContentDocumentRepository, *MockContentVersionRepository, *MockValidationService, *MockContentService) {
	docRepo := new(MockContentDocumentRepository)
	versionRepo := new(MockContentVersionRepository)
	validationSvc := new(MockValidationService)
	contentSvc := new(MockContentService)

	log := appLogger.New("test", nil)
	auditLog := audit.NewLogger(log)

	handler := &Handler{
		docRepo:       docRepo,
		versionRepo:   versionRepo,
		validationSvc: validationSvc,
		contentSvc:    contentSvc,
		auditLog:      auditLog,
	}

	return handler, docRepo, versionRepo, validationSvc, contentSvc
}

func TestGetDraft_Success(t *testing.T) {
	handler, docRepo, _, _, _ := setupTestHandler()

	// Setup mock
	expectedDoc := &model.ContentDocument{
		PageKey:      model.PageKeyHome,
		DraftConfig:  model.JSONMap{"title": "Home"},
		DraftVersion: 5,
	}
	docRepo.On("FindByPageKey", mock.Anything, model.PageKeyHome).Return(expectedDoc, nil)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}
	c.Request = httptest.NewRequest("GET", "/admin/content/home/draft", nil)

	// Execute
	handler.GetDraft(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response GetDraftResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "home", response.PageKey)
	assert.Equal(t, 5, response.Version)
	assert.Equal(t, "Home", response.Config["title"])

	docRepo.AssertExpectations(t)
}

func TestGetDraft_NotFound(t *testing.T) {
	handler, docRepo, _, _, _ := setupTestHandler()

	// Setup mock
	docRepo.On("FindByPageKey", mock.Anything, model.PageKeyHome).Return(nil, gorm.ErrRecordNotFound)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}
	c.Request = httptest.NewRequest("GET", "/admin/content/home/draft", nil)

	// Execute
	handler.GetDraft(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	docRepo.AssertExpectations(t)
}

func TestUpdateDraft_Success(t *testing.T) {
	handler, docRepo, _, _, _ := setupTestHandler()

	// Setup mock
	newConfig := model.JSONMap{"title": "Updated Home"}
	docRepo.On("UpdateDraft", mock.Anything, model.PageKeyHome, 5, newConfig).Return(6, nil)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}

	reqBody := UpdateDraftRequest{
		Config:     newConfig,
		ChangeNote: "Update title",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("PUT", "/admin/content/home/draft", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("If-Match", "5")

	// Execute
	handler.UpdateDraft(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response UpdateDraftResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "home", response.PageKey)
	assert.Equal(t, 6, response.Version)

	docRepo.AssertExpectations(t)
}

func TestUpdateDraft_VersionConflict(t *testing.T) {
	handler, docRepo, _, _, _ := setupTestHandler()

	// Setup mock - return the conflict error string that the repository returns
	newConfig := model.JSONMap{"title": "Updated Home"}
	docRepo.On("UpdateDraft", mock.Anything, model.PageKeyHome, 5, newConfig).Return(0, errors.New("draft version conflict or document not found"))

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}

	reqBody := UpdateDraftRequest{
		Config:     newConfig,
		ChangeNote: "Update title",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("PUT", "/admin/content/home/draft", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("If-Match", "5")

	// Execute
	handler.UpdateDraft(c)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)
	docRepo.AssertExpectations(t)
}

func TestValidate_Success(t *testing.T) {
	handler, _, _, validationSvc, _ := setupTestHandler()

	// Setup mock
	config := model.JSONMap{"hero": map[string]interface{}{"title": map[string]interface{}{"zh": "首页", "en": "Home"}}}
	validationResult := &service.ValidationResult{
		Valid:             true,
		Errors:            []service.ValidationError{},
		TranslationStatus: map[string]service.TranslationState{"hero.title": service.TranslationStateDone},
	}
	validationSvc.On("ValidateConfig", model.PageKeyHome, config).Return(validationResult)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}

	reqBody := ValidateRequest{Config: config}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/admin/content/home/validate", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.Validate(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response ValidateResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Valid)

	validationSvc.AssertExpectations(t)
}

func TestPublish_Success(t *testing.T) {
	handler, _, _, _, contentSvc := setupTestHandler()

	// Setup user context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}
	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID:   1,
		Username: "admin",
		Role:     model.RoleAdmin,
	})

	// Setup mock
	publishTime := time.Now().UTC()
	publishResult := &service.PublishResult{
		PageKey:          model.PageKeyHome,
		PublishedVersion: 11,
		PublishedAt:      publishTime,
	}
	contentSvc.On("Publish", mock.Anything, model.PageKeyHome, 10, uint(1)).Return(publishResult, nil)

	// Create test request
	reqBody := PublishRequest{
		ExpectedDraftVersion: 10,
		ChangeNote:           "首页改版发布",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/admin/content/home/publish", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.Publish(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response PublishResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "home", response.PageKey)
	assert.Equal(t, 11, response.PublishedVersion)

	contentSvc.AssertExpectations(t)
}

func TestPublish_VersionMismatch(t *testing.T) {
	handler, _, _, _, contentSvc := setupTestHandler()

	// Setup user context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}
	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID:   1,
		Username: "admin",
		Role:     model.RoleAdmin,
	})

	// Setup mock
	contentSvc.On("Publish", mock.Anything, model.PageKeyHome, 10, uint(1)).Return(nil, service.ErrVersionMismatch)

	// Create test request
	reqBody := PublishRequest{
		ExpectedDraftVersion: 10,
		ChangeNote:           "首页改版发布",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/admin/content/home/publish", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.Publish(c)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)
	contentSvc.AssertExpectations(t)
}

func TestGetVersions_Success(t *testing.T) {
	handler, _, versionRepo, _, _ := setupTestHandler()

	// Setup mock
	publishedAt := time.Now().UTC()
	versions := []*model.ContentVersion{
		{
			ID:          1,
			PageKey:     model.PageKeyHome,
			Version:     11,
			PublishedAt: publishedAt,
			CreatedBy:   1,
		},
	}
	versionRepo.On("ListByPageKey", mock.Anything, model.PageKeyHome, 0, 20).Return(versions, int64(1), nil)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}
	c.Request = httptest.NewRequest("GET", "/admin/content/home/versions", nil)

	// Execute
	handler.GetVersions(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response GetVersionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), response.Total)
	assert.Len(t, response.Items, 1)
	assert.Equal(t, 11, response.Items[0].Version)

	versionRepo.AssertExpectations(t)
}

func TestGetVersionDetail_Success(t *testing.T) {
	handler, _, versionRepo, _, _ := setupTestHandler()

	// Setup mock
	publishedAt := time.Now().UTC()
	version := &model.ContentVersion{
		ID:          1,
		PageKey:     model.PageKeyHome,
		Version:     11,
		Config:      model.JSONMap{"title": "Home"},
		PublishedAt: publishedAt,
		CreatedBy:   1,
	}
	versionRepo.On("FindByPageKeyAndVersion", mock.Anything, model.PageKeyHome, 11).Return(version, nil)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}, {Key: "version", Value: "11"}}
	c.Request = httptest.NewRequest("GET", "/admin/content/home/versions/11", nil)

	// Execute
	handler.GetVersionDetail(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response GetVersionDetailResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "home", response.PageKey)
	assert.Equal(t, 11, response.Version)
	assert.Equal(t, uint(1), response.ID)

	versionRepo.AssertExpectations(t)
}

func TestRollback_Success(t *testing.T) {
	handler, _, _, _, contentSvc := setupTestHandler()

	// Setup user context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}, {Key: "version", Value: "9"}}
	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID:   1,
		Username: "admin",
		Role:     model.RoleAdmin,
	})

	// Setup mock
	rollbackTime := time.Now().UTC()
	rollbackResult := &service.RollbackResult{
		PageKey:          model.PageKeyHome,
		PublishedVersion: 12,
		SourceVersion:    9,
		PublishedAt:      rollbackTime,
	}
	contentSvc.On("Rollback", mock.Anything, model.PageKeyHome, 9, uint(1)).Return(rollbackResult, nil)

	// Create test request
	reqBody := RollbackRequest{
		ChangeNote: "回滚到 v9",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/admin/content/home/rollback/9", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.Rollback(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response RollbackResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "home", response.PageKey)
	assert.Equal(t, 12, response.PublishedVersion)
	assert.Equal(t, 9, response.SourceVersion)

	contentSvc.AssertExpectations(t)
}

func TestRollback_VersionNotFound(t *testing.T) {
	handler, _, _, _, contentSvc := setupTestHandler()

	// Setup user context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}, {Key: "version", Value: "9"}}
	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID:   1,
		Username: "admin",
		Role:     model.RoleAdmin,
	})

	// Setup mock
	contentSvc.On("Rollback", mock.Anything, model.PageKeyHome, 9, uint(1)).Return(nil, service.ErrVersionNotFound)

	// Create test request
	reqBody := RollbackRequest{
		ChangeNote: "回滚到 v9",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/admin/content/home/rollback/9", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.Rollback(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	contentSvc.AssertExpectations(t)
}

func TestUpdateDraft_MissingIfMatch(t *testing.T) {
	handler, _, _, _, _ := setupTestHandler()

	// Create test request without If-Match header
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}

	reqBody := UpdateDraftRequest{
		Config:     model.JSONMap{"title": "Updated Home"},
		ChangeNote: "Update title",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("PUT", "/admin/content/home/draft", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	// No If-Match header set

	// Execute
	handler.UpdateDraft(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPublish_CannotPublish(t *testing.T) {
	handler, _, _, _, contentSvc := setupTestHandler()

	// Setup user context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}
	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID:   1,
		Username: "admin",
		Role:     model.RoleAdmin,
	})

	// Setup mock - service returns ErrCannotPublish
	contentSvc.On("Publish", mock.Anything, model.PageKeyHome, 10, uint(1)).Return(nil, service.ErrCannotPublish)

	// Create test request
	reqBody := PublishRequest{
		ExpectedDraftVersion: 10,
		ChangeNote:           "首页改版发布",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/admin/content/home/publish", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.Publish(c)

	// Assert
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	contentSvc.AssertExpectations(t)
}

func TestGetDraft_InvalidPageKey(t *testing.T) {
	handler, _, _, _, _ := setupTestHandler()

	// Create test request with invalid page key
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "invalid-page"}}
	c.Request = httptest.NewRequest("GET", "/admin/content/invalid-page/draft", nil)

	// Execute
	handler.GetDraft(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetVersionDetail_NotFound(t *testing.T) {
	handler, _, versionRepo, _, _ := setupTestHandler()

	// Setup mock - version not found
	versionRepo.On("FindByPageKeyAndVersion", mock.Anything, model.PageKeyHome, 99).Return(nil, gorm.ErrRecordNotFound)

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}, {Key: "version", Value: "99"}}
	c.Request = httptest.NewRequest("GET", "/admin/content/home/versions/99", nil)

	// Execute
	handler.GetVersionDetail(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	versionRepo.AssertExpectations(t)
}

func TestUpdateDraft_InternalError(t *testing.T) {
	handler, docRepo, _, _, _ := setupTestHandler()

	// Setup mock - database error
	newConfig := model.JSONMap{"title": "Updated Home"}
	docRepo.On("UpdateDraft", mock.Anything, model.PageKeyHome, 5, newConfig).Return(0, errors.New("database error"))

	// Create test request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "pageKey", Value: "home"}}

	reqBody := UpdateDraftRequest{
		Config:     newConfig,
		ChangeNote: "Update title",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("PUT", "/admin/content/home/draft", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("If-Match", "5")

	// Execute
	handler.UpdateDraft(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	docRepo.AssertExpectations(t)
}
