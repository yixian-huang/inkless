package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/auth"
)

const testJWTSecret = "test-secret-key-for-middleware-tests"

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestAuth_ValidToken(t *testing.T) {
	router := setupTestRouter()

	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		userCtx := GetUserContext(c)
		require.NotNil(t, userCtx)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userCtx.UserID,
			"username": userCtx.Username,
			"role":     userCtx.Role,
		})
	})

	// Generate valid access token
	token, err := auth.GenerateAccessToken(1, "testuser", "admin", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing authorization header")
	assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
}

func TestAuth_InvalidHeaderFormat(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "sometoken"},
		{"Wrong prefix", "Basic sometoken"},
		{"Empty token", "Bearer "},
		{"Missing token", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestAuth_WrongSecret(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Generate token with different secret
	token, err := auth.GenerateAccessToken(1, "testuser", "admin", "wrong-secret")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestAuth_InvalidRole(t *testing.T) {
	router := setupTestRouter()
	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Generate token with invalid role (manually craft claims)
	token, err := auth.GenerateAccessToken(1, "testuser", "superadmin", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid role in token")
}

func TestAuth_UserContextInjection(t *testing.T) {
	router := setupTestRouter()

	router.GET("/protected", Auth(testJWTSecret), func(c *gin.Context) {
		userCtx := GetUserContext(c)
		require.NotNil(t, userCtx)
		assert.Equal(t, uint(42), userCtx.UserID)
		assert.Equal(t, "john", userCtx.Username)
		assert.Equal(t, model.RoleEditor, userCtx.Role)
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateAccessToken(42, "john", "editor", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_AdminAccess(t *testing.T) {
	router := setupTestRouter()

	router.GET("/admin", Auth(testJWTSecret), RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	token, err := auth.GenerateAccessToken(1, "admin", "admin", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_EditorDenied(t *testing.T) {
	router := setupTestRouter()

	router.GET("/admin", Auth(testJWTSecret), RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateAccessToken(1, "editor", "editor", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Admin access required")
	assert.Contains(t, w.Body.String(), "FORBIDDEN")
}

func TestRequireAdmin_NoAuthContext(t *testing.T) {
	router := setupTestRouter()

	// RequireAdmin without Auth middleware (should fail)
	router.GET("/admin", RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestRequireAdminOrEditor_AdminAccess(t *testing.T) {
	router := setupTestRouter()

	router.GET("/content", Auth(testJWTSecret), RequireAdminOrEditor(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	token, err := auth.GenerateAccessToken(1, "admin", "admin", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/content", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdminOrEditor_EditorAccess(t *testing.T) {
	router := setupTestRouter()

	router.GET("/content", Auth(testJWTSecret), RequireAdminOrEditor(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	token, err := auth.GenerateAccessToken(2, "editor", "editor", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/content", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdminOrEditor_NoAuthContext(t *testing.T) {
	router := setupTestRouter()

	// RequireAdminOrEditor without Auth middleware
	router.GET("/content", RequireAdminOrEditor(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/content", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestGetUserContext_Exists(t *testing.T) {
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		// Manually set user context
		userCtx := &UserContext{
			UserID:   99,
			Username: "testuser",
			Role:     model.RoleAdmin,
		}
		c.Set(string(UserContextKey), userCtx)

		// Retrieve it
		retrieved := GetUserContext(c)
		require.NotNil(t, retrieved)
		assert.Equal(t, uint(99), retrieved.UserID)
		assert.Equal(t, "testuser", retrieved.Username)
		assert.Equal(t, model.RoleAdmin, retrieved.Role)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserContext_NotExists(t *testing.T) {
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		userCtx := GetUserContext(c)
		assert.Nil(t, userCtx)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserContext_WrongType(t *testing.T) {
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		// Set wrong type
		c.Set(string(UserContextKey), "wrong-type")

		userCtx := GetUserContext(c)
		assert.Nil(t, userCtx)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddlewareChain_AdminOnly(t *testing.T) {
	router := setupTestRouter()

	router.POST("/admin/action", Auth(testJWTSecret), RequireAdmin(), func(c *gin.Context) {
		userCtx := GetUserContext(c)
		c.JSON(http.StatusOK, gin.H{
			"message":  "action performed",
			"actor":    userCtx.Username,
			"actor_id": userCtx.UserID,
		})
	})

	// Test admin access
	adminToken, err := auth.GenerateAccessToken(1, "admin", "admin", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/action", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test editor denied
	editorToken, err := auth.GenerateAccessToken(2, "editor", "editor", testJWTSecret)
	require.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/admin/action", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMiddlewareChain_EditorOrAdmin(t *testing.T) {
	router := setupTestRouter()

	router.PUT("/content/draft", Auth(testJWTSecret), RequireAdminOrEditor(), func(c *gin.Context) {
		userCtx := GetUserContext(c)
		c.JSON(http.StatusOK, gin.H{
			"message": "draft updated",
			"by":      userCtx.Username,
		})
	})

	// Test admin access
	adminToken, err := auth.GenerateAccessToken(1, "admin", "admin", testJWTSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/content/draft", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test editor access
	editorToken, err := auth.GenerateAccessToken(2, "editor", "editor", testJWTSecret)
	require.NoError(t, err)

	req = httptest.NewRequest(http.MethodPut, "/content/draft", nil)
	req.Header.Set("Authorization", "Bearer "+editorToken)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// fakeAPIKeyAuth implements APIKeyAuthenticator for tests.
type fakeAPIKeyAuth struct {
	principal *service.APIKeyPrincipal
	err       error
}

func (f *fakeAPIKeyAuth) Authenticate(ctx context.Context, plaintext string) (*service.APIKeyPrincipal, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.principal, nil
}

func TestAuth_APIKeyBearer(t *testing.T) {
	router := setupTestRouter()
	keyAuth := &fakeAPIKeyAuth{principal: &service.APIKeyPrincipal{
		UserID: 9, Username: "alice", Role: model.RoleAdmin,
		Scopes: []string{"media:create", "articles:update"}, KeyID: 3,
	}}

	router.GET("/protected", Auth(testJWTSecret, keyAuth), func(c *gin.Context) {
		uc := GetUserContext(c)
		require.NotNil(t, uc)
		assert.Equal(t, uint(9), uc.UserID)
		assert.Equal(t, []string{"media:create", "articles:update"}, GetAPIKeyScopes(c))
		assert.Equal(t, uint(3), GetAPIKeyID(c))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ink_deadbeef")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_APIKeyInvalid(t *testing.T) {
	router := setupTestRouter()
	keyAuth := &fakeAPIKeyAuth{err: errors.New("not found")}

	router.GET("/protected", Auth(testJWTSecret, keyAuth), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ink_bad")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireSessionJWT_BlocksAPIKey(t *testing.T) {
	router := setupTestRouter()
	keyAuth := &fakeAPIKeyAuth{principal: &service.APIKeyPrincipal{
		UserID: 1, Username: "a", Role: model.RoleAdmin,
		Scopes: []string{"media:create"}, KeyID: 1,
	}}

	router.GET("/keys", Auth(testJWTSecret, keyAuth), RequireSessionJWT(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// API key blocked
	req := httptest.NewRequest(http.MethodGet, "/keys", nil)
	req.Header.Set("Authorization", "Bearer ink_abc")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// JWT allowed
	token, err := auth.GenerateAccessToken(1, "a", "admin", testJWTSecret)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodGet, "/keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
