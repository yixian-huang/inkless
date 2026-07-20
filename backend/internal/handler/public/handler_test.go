package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	. "github.com/yixian-huang/inkless/backend/internal/repository"
)

// MockContentDocumentRepository is a mock implementation for testing
type MockContentDocumentRepository struct {
	FindByPageKeyFunc func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error)
}

func (m *MockContentDocumentRepository) Create(ctx context.Context, doc *model.ContentDocument) error {
	return nil
}

func (m *MockContentDocumentRepository) FindByPageKey(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
	if m.FindByPageKeyFunc != nil {
		return m.FindByPageKeyFunc(ctx, pageKey)
	}
	return nil, errors.New("not implemented")
}

func (m *MockContentDocumentRepository) Update(ctx context.Context, doc *model.ContentDocument) error {
	return nil
}

func (m *MockContentDocumentRepository) UpdateDraft(ctx context.Context, pageKey model.PageKey, expectedDraftVersion int, draftConfig model.JSONMap) (int, error) {
	return 0, nil
}

func (m *MockContentDocumentRepository) UpdatePublished(ctx context.Context, pageKey model.PageKey, publishedConfig model.JSONMap, publishedVersion int) error {
	return nil
}

func (m *MockContentDocumentRepository) List(ctx context.Context) ([]*model.ContentDocument, error) {
	return nil, nil
}

func (m *MockContentDocumentRepository) Delete(ctx context.Context, pageKey model.PageKey) error {
	return nil
}

// MockPageViewRepository is a mock implementation for testing
type MockPageViewRepository struct{}

func (m *MockPageViewRepository) Create(ctx context.Context, pv *model.PageView) error {
	return nil
}

func (m *MockPageViewRepository) CountByPageKey(ctx context.Context, pageKey string) (int64, error) {
	return 0, nil
}

func (m *MockPageViewRepository) GetSummary(ctx context.Context, now time.Time) ([]PageViewStats, error) {
	return nil, nil
}

func setupTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public/content/:pageKey", handler.GetPublicContent)
	return router
}

func TestGetPublicContent_Success(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{
		FindByPageKeyFunc: func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
			return &model.ContentDocument{
				PageKey:          model.PageKeyHome,
				PublishedConfig:  model.JSONMap{"title": map[string]interface{}{"zh": "首页", "en": "Home"}},
				PublishedVersion: 10,
				DraftConfig:      model.JSONMap{"secretData": "should not be exposed"},
				DraftVersion:     12,
			}, nil
		},
	}

	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/public/content/home?locale=zh", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Assert published data is returned
	assert.Equal(t, "home", response["pageKey"])
	assert.Equal(t, float64(10), response["version"])
	assert.Equal(t, "zh", response["locale"])
	assert.NotNil(t, response["config"])

	// Assert draft data is NOT included
	config := response["config"].(map[string]interface{})
	_, hasDraftData := config["secretData"]
	assert.False(t, hasDraftData, "Draft data should not be exposed in public endpoint")
}

func TestGetPublicContent_EnglishLocale(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{
		FindByPageKeyFunc: func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
			return &model.ContentDocument{
				PageKey:          model.PageKeyAbout,
				PublishedConfig:  model.JSONMap{"title": map[string]interface{}{"zh": "关于", "en": "About"}},
				PublishedVersion: 5,
			}, nil
		},
	}

	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/public/content/about?locale=en", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "about", response["pageKey"])
	assert.Equal(t, "en", response["locale"])
}

func TestGetPublicContent_DefaultLocale(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{
		FindByPageKeyFunc: func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
			return &model.ContentDocument{
				PageKey:          model.PageKeyHome,
				PublishedConfig:  model.JSONMap{"hero": map[string]interface{}{"title": "Home"}},
				PublishedVersion: 1,
			}, nil
		},
	}

	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	// No locale parameter provided - should default to zh
	req := httptest.NewRequest("GET", "/public/content/home", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "zh", response["locale"])
}

func TestGetPublicContent_InvalidPageKey(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{}
	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/public/content/invalid-page", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "BAD_REQUEST", errorObj["code"])
}

func TestGetPublicContent_InvalidLocale(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{}
	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/public/content/home?locale=fr", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "BAD_REQUEST", errorObj["code"])
	assert.Contains(t, errorObj["message"], "locale must be zh or en")
}

func TestGetPublicContent_PageNotFound(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{
		FindByPageKeyFunc: func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
			return nil, errors.New("record not found")
		},
	}

	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/public/content/home?locale=zh", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errorObj["code"])
}

func TestGetPublicContent_NeverExposesDraftFields(t *testing.T) {
	mockRepo := &MockContentDocumentRepository{
		FindByPageKeyFunc: func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
			return &model.ContentDocument{
				PageKey:          model.PageKeyHome,
				PublishedConfig:  model.JSONMap{"published": "data"},
				PublishedVersion: 10,
				DraftConfig:      model.JSONMap{"draft": "secret"},
				DraftVersion:     15,
			}, nil
		},
	}

	handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
	router := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/public/content/home?locale=zh", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Assert response only contains published data
	_, hasDraftConfig := response["draftConfig"]
	assert.False(t, hasDraftConfig, "draftConfig field should not be exposed")

	_, hasDraftVersion := response["draftVersion"]
	assert.False(t, hasDraftVersion, "draftVersion field should not be exposed")

	// Assert published version is the one returned
	assert.Equal(t, float64(10), response["version"])
}

func TestGetPublicContent_AllValidPageKeys(t *testing.T) {
	validPageKeys := []string{"home", "about", "advantages", "core-services", "cases", "experts", "contact", "global"}

	for _, pageKey := range validPageKeys {
		t.Run("PageKey_"+pageKey, func(t *testing.T) {
			mockRepo := &MockContentDocumentRepository{
				FindByPageKeyFunc: func(ctx context.Context, pk model.PageKey) (*model.ContentDocument, error) {
					return &model.ContentDocument{
						PageKey:          pk,
						PublishedConfig:  model.JSONMap{"test": "data"},
						PublishedVersion: 1,
					}, nil
				},
			}

			handler := NewHandler(mockRepo, &MockPageViewRepository{}, nil, cache.New(60*time.Second))
			router := setupTestRouter(handler)

			req := httptest.NewRequest("GET", "/public/content/"+pageKey+"?locale=zh", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, pageKey, response["pageKey"])
		})
	}
}
