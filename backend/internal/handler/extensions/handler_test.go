package extensions

import (
	"context"
	"encoding/json"
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
}

func (s *stubThemeRepo) List(ctx context.Context) ([]*model.InstalledTheme, error) {
	return s.themes, s.err
}
func (s *stubThemeRepo) FindByThemeID(ctx context.Context, themeID string) (*model.InstalledTheme, error) {
	return nil, nil
}
func (s *stubThemeRepo) FindActive(ctx context.Context) (*model.InstalledTheme, error) {
	return nil, nil
}
func (s *stubThemeRepo) SetActive(ctx context.Context, themeID string) error { return nil }
func (s *stubThemeRepo) Create(ctx context.Context, theme *model.InstalledTheme) error {
	return nil
}
func (s *stubThemeRepo) Update(ctx context.Context, theme *model.InstalledTheme) error {
	return nil
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
