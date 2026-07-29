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

func TestAdminThemeCatalog_Embedded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{
		themes: []*model.InstalledTheme{
			{ThemeID: "corporate-classic", Version: "1.0.0", Source: "built-in", IsActive: true},
			{ThemeID: "blog-first", Version: "1.0.0", Source: "built-in", IsActive: false},
		},
	}
	h := NewHandler(themecatalog.NewLoader(""), repo, "0.1.0-alpha.2", themecatalog.DefaultUMDAllowHosts)

	r := gin.New()
	r.GET("/admin/extensions/themes/catalog", h.AdminThemeCatalog)

	req := httptest.NewRequest(http.MethodGet, "/admin/extensions/themes/catalog", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["schemaVersion"])
	assert.Equal(t, string(themecatalog.SourceEmbedded), body["source"])
	assert.Nil(t, body["warning"])

	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(items), 3)

	bySlug := map[string]map[string]any{}
	for _, raw := range items {
		m := raw.(map[string]any)
		bySlug[m["slug"].(string)] = m
	}

	assert.Equal(t, string(themecatalog.InstallStateActive), bySlug["corporate-classic"]["installState"])
	assert.Equal(t, string(themecatalog.InstallStateBuiltin), bySlug["blog-first"]["installState"])
	// product-first not in stub install list → not_installed (has UMD) or builtin if in BuiltinIDs without row
	// host registers product-first as builtin ID → builtin when no row
	assert.Equal(t, string(themecatalog.InstallStateBuiltin), bySlug["product-first"]["installState"])
}

func TestAdminThemeCatalog_RefreshQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(themecatalog.NewLoader(""), &stubThemeRepo{}, "dev", nil)

	r := gin.New()
	r.GET("/admin/extensions/themes/catalog", h.AdminThemeCatalog)

	req := httptest.NewRequest(http.MethodGet, "/admin/extensions/themes/catalog?refresh=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestParseRefreshQuery(t *testing.T) {
	assert.True(t, parseRefreshQuery("1"))
	assert.True(t, parseRefreshQuery("true"))
	assert.False(t, parseRefreshQuery(""))
	assert.False(t, parseRefreshQuery("0"))
}

func TestAdminThemeInstall_CreateMarketplace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{}
	h := NewHandler(themecatalog.NewLoader(""), repo, "0.1.0-alpha.2", themecatalog.DefaultUMDAllowHosts)

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
	assert.Equal(t, false, resp["activated"])
	theme := resp["theme"].(map[string]any)
	assert.Equal(t, "product-first", theme["themeId"])
	assert.Equal(t, "marketplace", theme["source"])
	assert.NotEmpty(t, theme["externalUrl"])
	assert.Equal(t, "0.1.5", theme["version"])
}

func TestAdminThemeInstall_BuiltinOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubThemeRepo{}
	h := NewHandler(themecatalog.NewLoader(""), repo, "0.1.0-alpha.2", themecatalog.DefaultUMDAllowHosts)

	r := gin.New()
	r.POST("/admin/extensions/themes/install", h.AdminThemeInstall)

	body, _ := json.Marshal(map[string]any{"slug": "minimal-starter"})
	req := httptest.NewRequest(http.MethodPost, "/admin/extensions/themes/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	theme := resp["theme"].(map[string]any)
	assert.Equal(t, "built-in", theme["source"])
	// omitempty may drop empty externalUrl
	assert.True(t, theme["externalUrl"] == nil || theme["externalUrl"] == "")
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
	h := NewHandler(themecatalog.NewLoader(""), repo, "0.1.0-alpha.2", themecatalog.DefaultUMDAllowHosts)

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
	assert.Equal(t, false, resp["created"])
	assert.Equal(t, true, resp["activated"])
	theme := resp["theme"].(map[string]any)
	assert.Equal(t, true, theme["isActive"])
	assert.Equal(t, "marketplace", theme["source"])
	assert.Equal(t, "0.1.5", theme["version"])

	// previous active deactivated
	blog, err := repo.FindByThemeID(context.Background(), "blog-first")
	require.NoError(t, err)
	assert.False(t, blog.IsActive)
}

func TestAdminThemeInstall_UnknownSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(themecatalog.NewLoader(""), &stubThemeRepo{}, "0.1.0-alpha.2", themecatalog.DefaultUMDAllowHosts)
	r := gin.New()
	r.POST("/admin/extensions/themes/install", h.AdminThemeInstall)

	body, _ := json.Marshal(map[string]any{"slug": "does-not-exist"})
	req := httptest.NewRequest(http.MethodPost, "/admin/extensions/themes/install", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
