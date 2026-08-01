package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

type fakeActiveTheme struct {
	theme *model.InstalledTheme
}

func (f *fakeActiveTheme) FindActive(ctx context.Context) (*model.InstalledTheme, error) {
	return f.theme, nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestWhoami_APIKey(t *testing.T) {
	h := NewHandler("https://ops.example.com/", "1.2.3")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/agent/whoami", nil)

	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID: 7, Username: "alice", Role: model.RoleEditor,
	})
	// scopes + key id (as Auth middleware would set)
	c.Set("api_key_scopes", []string{"articles:read", "articles:update", "media:create"})
	c.Set("api_key_id", uint(42))

	user := &model.User{
		ID: 7, Username: "alice", Role: model.RoleEditor, IsSuperAdmin: false,
	}
	// Editor legacy role has content permissions
	c.Set("rbac_user", user)

	h.Whoami(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp WhoamiResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "https://ops.example.com", resp.BaseURL)
	assert.Equal(t, "1.2.3", resp.Version)
	assert.Equal(t, "api_key", resp.AuthMethod)
	require.NotNil(t, resp.APIKeyID)
	assert.Equal(t, uint(42), *resp.APIKeyID)
	assert.Equal(t, []string{"articles:read", "articles:update", "media:create"}, resp.Scopes)
	assert.Equal(t, uint(7), resp.User.ID)
	assert.Equal(t, "alice", resp.User.Username)
	// Editor can read/update articles in legacy RBAC; scopes allow articles:read/update + media
	assert.True(t, resp.Capabilities.Articles)
	assert.True(t, resp.Capabilities.AIArticleMeta)
	assert.True(t, resp.Capabilities.MediaUpload)
	// pages:read not in key scopes → false even if RBAC allows
	assert.False(t, resp.Capabilities.Pages)
	assert.False(t, resp.Capabilities.ThemeContent)
	assert.Empty(t, resp.Capabilities.ThemeContentKeys)
	assert.False(t, resp.Capabilities.Publish)
}

func TestWhoami_Session(t *testing.T) {
	h := NewHandler("https://blog.example.com", "dev")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/agent/whoami", nil)

	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID: 1, Username: "admin", Role: model.RoleAdmin,
	})
	c.Set("rbac_user", &model.User{
		ID: 1, Username: "admin", Role: model.RoleAdmin, IsSuperAdmin: true,
	})

	h.Whoami(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp WhoamiResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "session", resp.AuthMethod)
	assert.Nil(t, resp.APIKeyID)
	assert.Empty(t, resp.Scopes)
	assert.Equal(t, []string{"*:*"}, resp.Permissions)
	assert.True(t, resp.Capabilities.Articles)
	assert.True(t, resp.Capabilities.Pages)
	assert.True(t, resp.Capabilities.ThemeContent)
	assert.Contains(t, resp.Capabilities.ThemeContentKeys, "home")
	assert.NotContains(t, resp.Capabilities.ThemeContentKeys, "theme")
	assert.True(t, resp.Capabilities.Publish)
}

func TestWhoami_WithContentSlots(t *testing.T) {
	h := NewHandler("https://ops.example.com", "dev")
	h.WithSlots(contentslots.NewResolver(&fakeActiveTheme{
		theme: &model.InstalledTheme{ThemeID: "product-first", Version: "0.1.9", IsActive: true},
	}, contentslots.DefaultRegistry()))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/agent/whoami", nil)
	c.Set(string(middleware.UserContextKey), &middleware.UserContext{
		UserID: 1, Username: "admin", Role: model.RoleAdmin,
	})
	c.Set("rbac_user", &model.User{
		ID: 1, Username: "admin", Role: model.RoleAdmin, IsSuperAdmin: true,
	})

	h.Whoami(c)
	require.Equal(t, http.StatusOK, w.Code)
	var resp WhoamiResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "product-first", resp.Capabilities.ActiveThemeID)
	assert.Equal(t, "0.1.9", resp.Capabilities.ActiveThemeVersion)
	assert.Equal(t, []string{"home"}, resp.Capabilities.ContentSlots)
	assert.Equal(t, []string{"home"}, resp.Capabilities.ThemeContentKeys)
}

func TestWhoami_Unauthorized(t *testing.T) {
	h := NewHandler("https://x.example.com", "")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/agent/whoami", nil)
	h.Whoami(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
