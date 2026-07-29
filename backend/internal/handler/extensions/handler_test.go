package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
)

type stubThemeRepo struct {
	themes []*model.InstalledTheme
	err    error
	nextID uint
}

func (s *stubThemeRepo) List(ctx context.Context) ([]*model.InstalledTheme, error) {
	return s.themes, s.err
}
func (s *stubThemeRepo) FindByThemeID(ctx context.Context, themeID string) (*model.InstalledTheme, error) {
	for _, t := range s.themes {
		if t != nil && t.ThemeID == themeID {
			return t, nil
		}
	}
	return nil, errors.New("theme not found")
}
func (s *stubThemeRepo) FindActive(ctx context.Context) (*model.InstalledTheme, error) {
	for _, t := range s.themes {
		if t != nil && t.IsActive {
			return t, nil
		}
	}
	return nil, errors.New("no active theme found")
}
func (s *stubThemeRepo) SetActive(ctx context.Context, themeID string) error {
	found := false
	for _, t := range s.themes {
		if t == nil {
			continue
		}
		if t.ThemeID == themeID {
			t.IsActive = true
			found = true
		} else {
			t.IsActive = false
		}
	}
	if !found {
		return errors.New("theme not found")
	}
	return nil
}
func (s *stubThemeRepo) Create(ctx context.Context, theme *model.InstalledTheme) error {
	s.nextID++
	theme.ID = s.nextID
	cp := *theme
	s.themes = append(s.themes, &cp)
	return nil
}
func (s *stubThemeRepo) Update(ctx context.Context, theme *model.InstalledTheme) error {
	for i, t := range s.themes {
		if t != nil && t.ID == theme.ID {
			cp := *theme
			s.themes[i] = &cp
			return nil
		}
	}
	return errors.New("theme not found")
}
func (s *stubThemeRepo) Delete(ctx context.Context, id uint) error { return nil }

type stubSiteCfg struct {
	cfg *model.SiteConfig
}

func (s *stubSiteCfg) FindByKey(ctx context.Context, key string) (*model.SiteConfig, error) {
	if s.cfg == nil || s.cfg.Key != key {
		return &model.SiteConfig{}, errors.New("record not found")
	}
	return s.cfg, nil
}
func (s *stubSiteCfg) Upsert(ctx context.Context, config *model.SiteConfig) error {
	cp := *config
	s.cfg = &cp
	return nil
}
func (s *stubSiteCfg) Update(ctx context.Context, config *model.SiteConfig) error {
	cp := *config
	s.cfg = &cp
	return nil
}
func (s *stubSiteCfg) UpdateDraft(ctx context.Context, key string, expectedVersion int, draftConfig model.JSONMap) (int, error) {
	return 0, nil
}
func (s *stubSiteCfg) UpdatePublished(ctx context.Context, key string, publishedConfig model.JSONMap, publishedVersion int) error {
	return nil
}

func testHandler(repo *stubThemeRepo) *Handler {
	inst := service.NewOfficialThemeInstaller(
		themecatalog.NewLoader(""),
		repo,
		"0.1.0-alpha.2",
		themecatalog.DefaultUMDAllowHosts,
	)
	auto := service.NewThemeAutoUpdateService(inst, &stubSiteCfg{}, repo)
	return NewHandler(inst, auto)
}

func TestAdminThemeCatalog_Embedded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{
		themes: []*model.InstalledTheme{
			{ThemeID: "corporate-classic", Version: "1.0.0", Source: "built-in", IsActive: true},
			{ThemeID: "blog-first", Version: "1.0.0", Source: "built-in", IsActive: false},
		},
	}
	h := testHandler(repo)

	r := gin.New()
	r.GET("/admin/extensions/themes/catalog", h.AdminThemeCatalog)
	req := httptest.NewRequest(http.MethodGet, "/admin/extensions/themes/catalog", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, string(themecatalog.SourceEmbedded), body["source"])
	items := body["items"].([]any)
	require.GreaterOrEqual(t, len(items), 3)
}

func TestAdminThemeInstall_CreateMarketplace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{}
	h := testHandler(repo)
	r := gin.New()
	r.POST("/admin/extensions/themes/install", h.AdminThemeInstall)

	body, _ := json.Marshal(map[string]any{"slug": "product-first"})
	req := httptest.NewRequest(http.MethodPost, "/admin/extensions/themes/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["created"])
	theme := resp["theme"].(map[string]any)
	assert.Equal(t, "product-first", theme["themeId"])
	assert.Equal(t, "marketplace", theme["source"])
}

func TestAdminThemeInstall_UpsertAndActivate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{
		themes: []*model.InstalledTheme{
			{ID: 1, ThemeID: "product-first", Version: "0.1.0", Source: "built-in", IsActive: false},
			{ID: 2, ThemeID: "blog-first", Version: "1.0.0", Source: "built-in", IsActive: true},
		},
		nextID: 2,
	}
	h := testHandler(repo)
	r := gin.New()
	r.POST("/admin/extensions/themes/install", h.AdminThemeInstall)

	body, _ := json.Marshal(map[string]any{"slug": "product-first", "activate": true})
	req := httptest.NewRequest(http.MethodPost, "/admin/extensions/themes/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["activated"])
	theme := resp["theme"].(map[string]any)
	assert.Equal(t, true, theme["isActive"])
	assert.Equal(t, "marketplace", theme["source"])
}

func TestAdminThemeAutoUpdate_GetPutRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{
		themes: []*model.InstalledTheme{
			{
				ID: 1, ThemeID: "product-first", Version: "0.1.0", Source: "marketplace",
				ExternalURL: "https://github.com/yixian-huang/inkless-theme-product-first/releases/download/v0.1.0/theme.umd.js",
				IsActive:    true,
			},
		},
		nextID: 1,
	}
	h := testHandler(repo)
	r := gin.New()
	r.GET("/admin/extensions/themes/auto-update", h.AdminThemeAutoUpdateGet)
	r.PUT("/admin/extensions/themes/auto-update", h.AdminThemeAutoUpdatePut)
	r.POST("/admin/extensions/themes/auto-update/run", h.AdminThemeAutoUpdateRun)

	// GET defaults
	req := httptest.NewRequest(http.MethodGet, "/admin/extensions/themes/auto-update", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var settings map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &settings))
	assert.Equal(t, false, settings["enabled"])

	// Enable
	body, _ := json.Marshal(map[string]any{"enabled": true, "intervalMinutes": 30, "onlyMarketplace": true})
	req = httptest.NewRequest(http.MethodPut, "/admin/extensions/themes/auto-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Run dry
	body, _ = json.Marshal(map[string]any{"dryRun": true})
	req = httptest.NewRequest(http.MethodPost, "/admin/extensions/themes/auto-update/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var report map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &report))
	assert.GreaterOrEqual(t, int(report["checked"].(float64)), 1)

	// Apply
	body, _ = json.Marshal(map[string]any{"dryRun": false})
	req = httptest.NewRequest(http.MethodPost, "/admin/extensions/themes/auto-update/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// product-first should be upgraded to catalog latest 0.1.5
	got, err := repo.FindByThemeID(context.Background(), "product-first")
	require.NoError(t, err)
	assert.Equal(t, "0.1.5", got.Version)
	assert.Contains(t, got.ExternalURL, "v0.1.5")
}
