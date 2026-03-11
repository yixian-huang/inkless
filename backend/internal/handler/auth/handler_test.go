package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"context"
	"testing"
	"time"

	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/pkg/auth"
	"blotting-consultancy/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*model.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) CountSuperAdmins(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) FindByToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.RefreshToken, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*model.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func setupTestHandler() (*Handler, *MockUserRepository, *MockRefreshTokenRepository) {
	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	cfg := &config.Config{
		JWTSecret:        "test-secret",
		JWTRefreshSecret: "test-refresh-secret",
	}

	handler := NewHandler(mockUserRepo, mockRefreshTokenRepo, cfg)
	return handler, mockUserRepo, mockRefreshTokenRepo
}

func TestLogin_Success(t *testing.T) {
	handler, mockUserRepo, mockRefreshTokenRepo := setupTestHandler()
	gin.SetMode(gin.TestMode)

	// Create test user with hashed password
	hashedPassword, _ := auth.HashPassword("password123")
	user := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hashedPassword,
		Role:         model.RoleAdmin,
	}

	mockUserRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	mockRefreshTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.RefreshToken")).Return(nil)

	// Create request
	reqBody := LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "admin", resp.Role)
	assert.Equal(t, 900, resp.ExpiresIn) // 15 minutes in seconds

	mockUserRepo.AssertExpectations(t)
	mockRefreshTokenRepo.AssertExpectations(t)
}

func TestLogin_InvalidUsername(t *testing.T) {
	handler, mockUserRepo, _ := setupTestHandler()
	gin.SetMode(gin.TestMode)

	mockUserRepo.On("FindByUsername", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))

	reqBody := LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockUserRepo.AssertExpectations(t)
}

func TestLogin_InvalidPassword(t *testing.T) {
	handler, mockUserRepo, _ := setupTestHandler()
	gin.SetMode(gin.TestMode)

	hashedPassword, _ := auth.HashPassword("correctpassword")
	user := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hashedPassword,
		Role:         model.RoleAdmin,
	}

	mockUserRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)

	reqBody := LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockUserRepo.AssertExpectations(t)
}

func TestRefresh_Success(t *testing.T) {
	handler, _, mockRefreshTokenRepo := setupTestHandler()
	gin.SetMode(gin.TestMode)

	// Generate valid refresh token
	refreshToken, _ := auth.GenerateRefreshToken(1, "testuser", "admin", "test-refresh-secret")

	storedToken := &model.RefreshToken{
		ID:        1,
		UserID:    1,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	mockRefreshTokenRepo.On("FindByToken", mock.Anything, refreshToken).Return(storedToken, nil)

	reqBody := RefreshRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RefreshResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, 900, resp.ExpiresIn)

	mockRefreshTokenRepo.AssertExpectations(t)
}

func TestRefresh_InvalidToken(t *testing.T) {
	handler, _, _ := setupTestHandler()
	gin.SetMode(gin.TestMode)

	reqBody := RefreshRequest{
		RefreshToken: "invalid-token",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_RevokedToken(t *testing.T) {
	handler, _, mockRefreshTokenRepo := setupTestHandler()
	gin.SetMode(gin.TestMode)

	refreshToken, _ := auth.GenerateRefreshToken(1, "testuser", "admin", "test-refresh-secret")

	mockRefreshTokenRepo.On("FindByToken", mock.Anything, refreshToken).Return(nil, errors.New("not found"))

	reqBody := RefreshRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRefreshTokenRepo.AssertExpectations(t)
}

func TestRefresh_ExpiredToken(t *testing.T) {
	handler, _, mockRefreshTokenRepo := setupTestHandler()
	gin.SetMode(gin.TestMode)

	refreshToken, _ := auth.GenerateRefreshToken(1, "testuser", "admin", "test-refresh-secret")

	// Token expired in the past
	storedToken := &model.RefreshToken{
		ID:        1,
		UserID:    1,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	mockRefreshTokenRepo.On("FindByToken", mock.Anything, refreshToken).Return(storedToken, nil)

	reqBody := RefreshRequest{
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRefreshTokenRepo.AssertExpectations(t)
}

func TestMe_Success(t *testing.T) {
	handler, mockUserRepo, _ := setupTestHandler()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/auth/me", nil)

	// Set user context (normally set by middleware)
	userCtx := &middleware.UserContext{
		UserID:   1,
		Username: "testuser",
		Role:     "admin",
	}
	c.Set(string(middleware.UserContextKey), userCtx)

	// Me now queries the database for full user info
	mockUserRepo.On("FindByID", mock.Anything, uint(1)).Return(&model.User{
		ID:           1,
		Username:     "testuser",
		Role:         model.RoleAdmin,
		IsSuperAdmin: true,
	}, nil)

	handler.Me(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), resp.ID)
	assert.Equal(t, "testuser", resp.Username)
	assert.Equal(t, "admin", resp.Role)
	assert.True(t, resp.IsSuperAdmin)
	mockUserRepo.AssertExpectations(t)
}

func TestMe_MissingContext(t *testing.T) {
	handler, _, _ := setupTestHandler()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/auth/me", nil)

	// No user context set
	handler.Me(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogout_Success(t *testing.T) {
	handler, _, mockRefreshTokenRepo := setupTestHandler()
	gin.SetMode(gin.TestMode)

	mockRefreshTokenRepo.On("DeleteByToken", mock.Anything, "valid-refresh-token").Return(nil)

	reqBody := LogoutRequest{
		RefreshToken: "valid-refresh-token",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/logout", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LogoutResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.OK)

	mockRefreshTokenRepo.AssertExpectations(t)
}

func TestLogout_IdempotentBehavior(t *testing.T) {
	handler, _, mockRefreshTokenRepo := setupTestHandler()
	gin.SetMode(gin.TestMode)

	// Token doesn't exist, but logout should still succeed
	mockRefreshTokenRepo.On("DeleteByToken", mock.Anything, "nonexistent-token").Return(errors.New("not found"))

	reqBody := LogoutRequest{
		RefreshToken: "nonexistent-token",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/logout", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LogoutResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.OK)

	mockRefreshTokenRepo.AssertExpectations(t)
}
