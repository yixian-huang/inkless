package features_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/handler/features"
	"blotting-consultancy/internal/model"
)

// MockSiteConfigRepository is a minimal in-memory mock satisfying repository.SiteConfigRepository.
type MockSiteConfigRepository struct {
	FindByKeyFunc    func(ctx context.Context, key string) (*model.SiteConfig, error)
	UpsertFunc       func(ctx context.Context, sc *model.SiteConfig) error
	UpdateFunc       func(ctx context.Context, sc *model.SiteConfig) error
	UpdateDraftFunc  func(ctx context.Context, key string, expectedVersion int, draftConfig model.JSONMap) (int, error)
	UpdatePublishedFunc func(ctx context.Context, key string, publishedConfig model.JSONMap, publishedVersion int) error
}

func (m *MockSiteConfigRepository) FindByKey(ctx context.Context, key string) (*model.SiteConfig, error) {
	if m.FindByKeyFunc != nil {
		return m.FindByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}
func (m *MockSiteConfigRepository) Upsert(ctx context.Context, sc *model.SiteConfig) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, sc)
	}
	return nil
}
func (m *MockSiteConfigRepository) Update(ctx context.Context, sc *model.SiteConfig) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, sc)
	}
	return nil
}
func (m *MockSiteConfigRepository) UpdateDraft(ctx context.Context, key string, expectedVersion int, draftConfig model.JSONMap) (int, error) {
	if m.UpdateDraftFunc != nil {
		return m.UpdateDraftFunc(ctx, key, expectedVersion, draftConfig)
	}
	return expectedVersion + 1, nil
}
func (m *MockSiteConfigRepository) UpdatePublished(ctx context.Context, key string, publishedConfig model.JSONMap, publishedVersion int) error {
	if m.UpdatePublishedFunc != nil {
		return m.UpdatePublishedFunc(ctx, key, publishedConfig, publishedVersion)
	}
	return nil
}

func newRouter(repo *MockSiteConfigRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/admin")
	features.NewHandler(repo, cache.New(0*time.Second)).RegisterRoutes(admin)
	return r
}

func TestAdminPutDraft_CreatesIfMissing(t *testing.T) {
	upserted := false
	repo := &MockSiteConfigRepository{
		FindByKeyFunc: func(ctx context.Context, key string) (*model.SiteConfig, error) {
			return nil, nil
		},
		UpsertFunc: func(ctx context.Context, sc *model.SiteConfig) error {
			upserted = true
			return nil
		},
	}
	r := newRouter(repo)
	body := `{"draftConfig":{"publicPages":{"home":true,"blog":true,"contact":true,"about":false,"experts":false,"coreServices":false,"advantages":false,"cases":false},"blog":{"comments":true,"rss":true}}}`
	req := httptest.NewRequest(http.MethodPut, "/admin/features/draft", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !upserted {
		t.Fatalf("Upsert was not called")
	}
}

func TestAdminPublish_RequiresDraft(t *testing.T) {
	repo := &MockSiteConfigRepository{
		FindByKeyFunc: func(ctx context.Context, key string) (*model.SiteConfig, error) {
			return nil, nil
		},
	}
	r := newRouter(repo)
	req := httptest.NewRequest(http.MethodPost, "/admin/features/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminGet_ReturnsEmptyDefaultsWhenMissing(t *testing.T) {
	repo := &MockSiteConfigRepository{
		FindByKeyFunc: func(ctx context.Context, key string) (*model.SiteConfig, error) {
			return nil, nil
		},
	}
	r := newRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/admin/features", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAdminPutDraft_VersionConflictReturns409(t *testing.T) {
	repo := &MockSiteConfigRepository{
		FindByKeyFunc: func(ctx context.Context, key string) (*model.SiteConfig, error) {
			return &model.SiteConfig{ID: 1, Key: model.SiteConfigKeyFeatures, DraftVersion: 5}, nil
		},
		UpdateDraftFunc: func(ctx context.Context, key string, expected int, draft model.JSONMap) (int, error) {
			return 0, errors.New("draft version conflict or config not found")
		},
	}
	r := newRouter(repo)
	body := `{"draftConfig":{"publicPages":{"home":true,"blog":true,"contact":true,"about":false,"experts":false,"coreServices":false,"advantages":false,"cases":false},"blog":{"comments":true,"rss":true}},"expectedDraftVersion":1}`
	req := httptest.NewRequest(http.MethodPut, "/admin/features/draft", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
