package role

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

func newRoleHandlerTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *model.User, *model.RBACRole) {
	t.Helper()

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.User{},
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
	))

	user := &model.User{
		Username:     "role-contract-user",
		PasswordHash: "test-hash",
		Role:         model.RoleAdmin,
	}
	require.NoError(t, database.Create(user).Error)

	role := &model.RBACRole{
		Name:        "contract_role",
		DisplayName: "Contract Role",
	}
	require.NoError(t, database.Create(role).Error)

	handler := NewHandler(
		repository.NewGormRoleRepository(database),
		repository.NewGormUserRepository(database),
	)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/roles", handler.List)
	router.POST("/admin/roles", handler.Create)
	router.PUT("/admin/roles/:id", handler.Update)
	router.GET("/admin/permissions", handler.ListPermissions)
	router.POST("/admin/roles/assign", handler.AssignRole)
	router.POST("/admin/roles/unassign", handler.UnassignRole)

	return router, database, user, role
}

func TestRoleMutationsRejectLegacySiteID(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   func(role *model.RBACRole) string
		body   func(user *model.User, role *model.RBACRole) map[string]any
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   func(_ *model.RBACRole) string { return "/admin/roles" },
			body: func(_ *model.User, _ *model.RBACRole) map[string]any {
				return map[string]any{"name": "legacy_scoped", "displayName": "Legacy Scoped", "siteId": 42}
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   func(role *model.RBACRole) string { return "/admin/roles/" + strconv.FormatUint(uint64(role.ID), 10) },
			body: func(_ *model.User, _ *model.RBACRole) map[string]any {
				return map[string]any{"displayName": "Changed", "siteId": 42}
			},
		},
		{
			name:   "assign",
			method: http.MethodPost,
			path:   func(_ *model.RBACRole) string { return "/admin/roles/assign" },
			body: func(user *model.User, role *model.RBACRole) map[string]any {
				return map[string]any{"userId": user.ID, "roleId": role.ID, "siteId": 42}
			},
		},
		{
			name:   "unassign",
			method: http.MethodPost,
			path:   func(_ *model.RBACRole) string { return "/admin/roles/unassign" },
			body: func(user *model.User, role *model.RBACRole) map[string]any {
				return map[string]any{"userId": user.ID, "roleId": role.ID, "siteId": 42}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, database, user, role := newRoleHandlerTestRouter(t)
			body, err := json.Marshal(test.body(user, role))
			require.NoError(t, err)

			request := httptest.NewRequest(test.method, test.path(role), bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Contains(t, response.Body.String(), "unknown field")
			var assignments int64
			require.NoError(t, database.Model(&model.UserRole{}).Count(&assignments).Error)
			assert.Zero(t, assignments)
			var created int64
			require.NoError(t, database.Model(&model.RBACRole{}).Where("name = ?", "legacy_scoped").Count(&created).Error)
			assert.Zero(t, created)
			var reloaded model.RBACRole
			require.NoError(t, database.First(&reloaded, role.ID).Error)
			assert.Equal(t, "Contract Role", reloaded.DisplayName)
		})
	}
}

func TestLegacySitePermissionsRemainStoredButAreNotExposed(t *testing.T) {
	router, database, _, role := newRoleHandlerTestRouter(t)
	legacy := &model.Permission{Resource: model.LegacyResourceSites, Action: "read", Description: "legacy"}
	current := &model.Permission{Resource: "settings", Action: "read", Description: "current"}
	require.NoError(t, database.Create(legacy).Error)
	require.NoError(t, database.Create(current).Error)
	require.NoError(t, database.Model(role).Association("Permissions").Append(legacy, current))

	for _, path := range []string{"/admin/permissions", "/admin/roles"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
		assert.NotContains(t, response.Body.String(), `"resource":"sites"`)
		assert.Contains(t, response.Body.String(), `"resource":"settings"`)
	}

	var stored int64
	require.NoError(t, database.Model(&model.Permission{}).
		Where("resource = ?", model.LegacyResourceSites).
		Count(&stored).Error)
	assert.EqualValues(t, 1, stored)
}

func TestAssignRoleRejectsLegacySiteID(t *testing.T) {
	router, database, user, role := newRoleHandlerTestRouter(t)

	body, err := json.Marshal(map[string]any{
		"userId": user.ID,
		"roleId": role.ID,
		"siteId": 42,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/admin/roles/assign", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "unknown field")

	var count int64
	require.NoError(t, database.Model(&model.UserRole{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestAssignRoleContractHasNoSiteID(t *testing.T) {
	router, database, user, role := newRoleHandlerTestRouter(t)

	body, err := json.Marshal(AssignRoleRequest{UserID: user.ID, RoleID: role.ID})
	require.NoError(t, err)
	assert.NotContains(t, string(body), "siteId")

	request := httptest.NewRequest(http.MethodPost, "/admin/roles/assign", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)

	var assignment model.UserRole
	require.NoError(t, database.First(&assignment).Error)
	assert.Equal(t, user.ID, assignment.UserID)
	assert.Equal(t, role.ID, assignment.RoleID)
	encodedAssignment, err := json.Marshal(assignment)
	require.NoError(t, err)
	assert.NotContains(t, string(encodedAssignment), "siteId")
}
