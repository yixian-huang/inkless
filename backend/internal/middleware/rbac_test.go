package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"blotting-consultancy/internal/model"

	"github.com/gin-gonic/gin"
)

type rbacTestUserRepository struct {
	user *model.User
}

func (r *rbacTestUserRepository) Create(context.Context, *model.User) error {
	return nil
}

func (r *rbacTestUserRepository) FindByID(context.Context, uint) (*model.User, error) {
	return r.user, nil
}

func (r *rbacTestUserRepository) FindByUsername(context.Context, string) (*model.User, error) {
	return r.user, nil
}

func (r *rbacTestUserRepository) Update(context.Context, *model.User) error {
	return nil
}

func (r *rbacTestUserRepository) Delete(context.Context, uint) error {
	return nil
}

func (r *rbacTestUserRepository) List(context.Context, int, int) ([]*model.User, int64, error) {
	return []*model.User{r.user}, 1, nil
}

func (r *rbacTestUserRepository) CountSuperAdmins(context.Context) (int64, error) {
	return 0, nil
}

func (r *rbacTestUserRepository) FindByIDWithRoles(context.Context, uint) (*model.User, error) {
	return r.user, nil
}

func TestRequirePermission_LegacyEditorBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &rbacTestUserRepository{
		user: &model.User{ID: 7, Role: model.RoleEditor},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(UserContextKey), &UserContext{UserID: 7, Role: model.RoleEditor})
		c.Next()
	})
	router.PUT("/pages/:id", RequirePermission("pages", "update", repo, nil), okHandler)
	router.POST("/pages/:id/publish", RequirePermission("pages", "publish", repo, nil), okHandler)
	router.POST("/migration/import", RequirePermission("system", "manage", repo, nil), okHandler)

	assertStatus(t, router, http.MethodPut, "/pages/1", http.StatusOK)
	assertStatus(t, router, http.MethodPost, "/pages/1/publish", http.StatusForbidden)
	assertStatus(t, router, http.MethodPost, "/migration/import", http.StatusForbidden)
}

func okHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

func assertStatus(t *testing.T, router http.Handler, method, path string, want int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != want {
		t.Fatalf("%s %s returned %d, want %d; body=%s", method, path, recorder.Code, want, recorder.Body.String())
	}
}
