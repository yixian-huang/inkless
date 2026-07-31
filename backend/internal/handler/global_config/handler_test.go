package global_config

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

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

// MockSiteConfigRepository — minimal in-memory mock.
type MockSiteConfigRepository struct {
	FindByKeyFunc       func(ctx context.Context, key string) (*model.SiteConfig, error)
	UpsertFunc          func(ctx context.Context, config *model.SiteConfig) error
	UpdateDraftFunc     func(ctx context.Context, key string, expected int, draft model.JSONMap) (int, error)
	UpdatePublishedFunc func(ctx context.Context, key string, published model.JSONMap, version int) error
	store               map[string]*model.SiteConfig
}

func (m *MockSiteConfigRepository) ensure() {
	if m.store == nil {
		m.store = map[string]*model.SiteConfig{}
	}
}

func (m *MockSiteConfigRepository) FindByKey(ctx context.Context, key string) (*model.SiteConfig, error) {
	if m.FindByKeyFunc != nil {
		return m.FindByKeyFunc(ctx, key)
	}
	m.ensure()
	if sc, ok := m.store[key]; ok {
		return sc, nil
	}
	return &model.SiteConfig{}, errors.New("record not found")
}

func (m *MockSiteConfigRepository) Upsert(ctx context.Context, config *model.SiteConfig) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, config)
	}
	m.ensure()
	cp := *config
	if cp.ID == 0 {
		cp.ID = uint(len(m.store) + 1)
	}
	m.store[config.Key] = &cp
	return nil
}

func (m *MockSiteConfigRepository) Update(ctx context.Context, config *model.SiteConfig) error {
	return m.Upsert(ctx, config)
}

func (m *MockSiteConfigRepository) UpdateDraft(ctx context.Context, key string, expected int, draft model.JSONMap) (int, error) {
	if m.UpdateDraftFunc != nil {
		return m.UpdateDraftFunc(ctx, key, expected, draft)
	}
	m.ensure()
	sc, ok := m.store[key]
	if !ok || sc.DraftVersion != expected {
		return 0, errors.New("draft version conflict or config not found")
	}
	sc.DraftConfig = draft
	sc.DraftVersion = expected + 1
	return sc.DraftVersion, nil
}

func (m *MockSiteConfigRepository) UpdatePublished(ctx context.Context, key string, published model.JSONMap, version int) error {
	if m.UpdatePublishedFunc != nil {
		return m.UpdatePublishedFunc(ctx, key, published, version)
	}
	m.ensure()
	sc, ok := m.store[key]
	if !ok {
		return errors.New("not found")
	}
	sc.PublishedConfig = published
	sc.PublishedVersion = version
	return nil
}

// MockContentDocumentRepository — legacy hydrate only.
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
	return nil, errors.New("not found")
}
func (m *MockContentDocumentRepository) Update(ctx context.Context, doc *model.ContentDocument) error {
	return nil
}
func (m *MockContentDocumentRepository) UpdateDraft(ctx context.Context, pageKey model.PageKey, expected int, draft model.JSONMap) (int, error) {
	return 0, errors.New("legacy write disabled")
}
func (m *MockContentDocumentRepository) UpdatePublished(ctx context.Context, pageKey model.PageKey, published model.JSONMap, version int) error {
	return errors.New("legacy write disabled")
}
func (m *MockContentDocumentRepository) List(ctx context.Context) ([]*model.ContentDocument, error) {
	return nil, nil
}
func (m *MockContentDocumentRepository) Delete(ctx context.Context, pageKey model.PageKey) error {
	return nil
}

func newRouter(site *MockSiteConfigRepository, legacy *MockContentDocumentRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/admin")
	h := NewHandler(site, cache.New(0*time.Second))
	if legacy != nil {
		h = h.WithLegacyContentDoc(legacy)
	}
	h.RegisterRoutes(admin)
	return r
}

func validGlobalConfig() map[string]any {
	return map[string]any{
		"identity": map[string]any{
			"name":          map[string]any{"zh": "My Site"},
			"localeMode":    "mono-zh",
			"defaultLocale": "zh",
		},
		"brand":  map[string]any{},
		"author": map[string]any{"socials": []any{}},
		"footer": map[string]any{},
		"seo":    map[string]any{},
	}
}

func TestAdminPutDraft_RejectsInvalidSchema(t *testing.T) {
	site := &MockSiteConfigRepository{}
	r := newRouter(site, nil)
	body := `{"draftConfig":{"identity":{"name":{},"localeMode":"mono-zh","defaultLocale":"zh"},"brand":{},"author":{"socials":[]},"footer":{},"seo":{}},"expectedDraftVersion":1}`
	req := httptest.NewRequest(http.MethodPut, "/admin/site-config/draft", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminPutDraft_CreatesWhenMissing(t *testing.T) {
	site := &MockSiteConfigRepository{}
	r := newRouter(site, nil)
	body, _ := json.Marshal(map[string]any{"draftConfig": validGlobalConfig(), "expectedDraftVersion": 0})
	req := httptest.NewRequest(http.MethodPut, "/admin/site-config/draft", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if site.store["global"] == nil || site.store["global"].DraftVersion != 1 {
		t.Fatalf("expected site_configs global draftVersion 1, got %+v", site.store["global"])
	}
}

func TestAdminPutDraft_AcceptsValidOnExisting(t *testing.T) {
	site := &MockSiteConfigRepository{
		store: map[string]*model.SiteConfig{
			"global": {
				ID:           1,
				Key:          model.SiteConfigKeyGlobal,
				DraftConfig:  model.JSONMap{},
				DraftVersion: 1,
			},
		},
	}
	called := false
	site.UpdateDraftFunc = func(ctx context.Context, key string, expected int, draft model.JSONMap) (int, error) {
		called = true
		if key != model.SiteConfigKeyGlobal {
			t.Errorf("expected SiteConfigKeyGlobal, got %q", key)
		}
		return expected + 1, nil
	}
	r := newRouter(site, nil)
	body, _ := json.Marshal(map[string]any{"draftConfig": validGlobalConfig(), "expectedDraftVersion": 1})
	req := httptest.NewRequest(http.MethodPut, "/admin/global-config/draft", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("UpdateDraft was not invoked")
	}
}

func TestAdminPutDraft_VersionConflictReturns409(t *testing.T) {
	site := &MockSiteConfigRepository{
		store: map[string]*model.SiteConfig{
			"global": {ID: 1, Key: model.SiteConfigKeyGlobal, DraftVersion: 1},
		},
		UpdateDraftFunc: func(ctx context.Context, key string, expected int, draft model.JSONMap) (int, error) {
			return 0, errors.New("draft version conflict or config not found")
		},
	}
	r := newRouter(site, nil)
	body, _ := json.Marshal(map[string]any{"draftConfig": validGlobalConfig(), "expectedDraftVersion": 1})
	req := httptest.NewRequest(http.MethodPut, "/admin/site-config/draft", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminPublish_BumpsVersion(t *testing.T) {
	cfgMap := model.JSONMap(validGlobalConfig())
	publishedCalled := false
	site := &MockSiteConfigRepository{
		store: map[string]*model.SiteConfig{
			"global": {
				ID:               1,
				Key:              model.SiteConfigKeyGlobal,
				DraftConfig:      cfgMap,
				DraftVersion:     2,
				PublishedConfig:  model.JSONMap{},
				PublishedVersion: 1,
			},
		},
		UpdatePublishedFunc: func(ctx context.Context, key string, published model.JSONMap, version int) error {
			publishedCalled = true
			if key != model.SiteConfigKeyGlobal {
				t.Errorf("expected SiteConfigKeyGlobal, got %q", key)
			}
			if version != 2 {
				t.Errorf("expected published version 2, got %d", version)
			}
			return nil
		},
	}
	r := newRouter(site, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/site-config/publish", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !publishedCalled {
		t.Fatalf("UpdatePublished was not invoked")
	}
}

func TestAdminGet_ReturnsBothDraftAndPublished(t *testing.T) {
	site := &MockSiteConfigRepository{
		store: map[string]*model.SiteConfig{
			"global": {
				ID:               1,
				Key:              model.SiteConfigKeyGlobal,
				DraftConfig:      model.JSONMap{"identity": map[string]any{"name": map[string]any{"zh": "draft"}}},
				DraftVersion:     3,
				PublishedConfig:  model.JSONMap{"identity": map[string]any{"name": map[string]any{"zh": "published"}}},
				PublishedVersion: 1,
			},
		},
	}
	r := newRouter(site, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/site-config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if int(resp["draftVersion"].(float64)) != 3 {
		t.Errorf("draftVersion not 3: %v", resp["draftVersion"])
	}
	if int(resp["publishedVersion"].(float64)) != 1 {
		t.Errorf("publishedVersion not 1: %v", resp["publishedVersion"])
	}
	if resp["storageSource"] != "site_config" {
		t.Errorf("storageSource: %v", resp["storageSource"])
	}
}

func TestAdminGet_HydratesFromLegacyContentDocument(t *testing.T) {
	cfg := model.JSONMap(validGlobalConfig())
	site := &MockSiteConfigRepository{}
	legacy := &MockContentDocumentRepository{
		FindByPageKeyFunc: func(ctx context.Context, pageKey model.PageKey) (*model.ContentDocument, error) {
			return &model.ContentDocument{
				PageKey:          model.PageKeyGlobal,
				DraftConfig:      cfg,
				DraftVersion:     4,
				PublishedConfig:  cfg,
				PublishedVersion: 2,
			}, nil
		},
	}
	r := newRouter(site, legacy)
	req := httptest.NewRequest(http.MethodGet, "/admin/global-config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["storageSource"] != "hydrated_from_content_document" {
		t.Fatalf("expected hydrate source, got %v body=%s", resp["storageSource"], w.Body.String())
	}
	if site.store["global"] == nil {
		t.Fatal("expected hydrate to write site_configs")
	}
	// Second get should be pure site_config.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/admin/site-config", nil))
	var resp2 map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2["storageSource"] != "site_config" {
		t.Fatalf("second get expected site_config, got %v", resp2["storageSource"])
	}
}
