package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/yixian-huang/inkless/backend/internal/model"
)

// API key plaintext format: ink_<32 hex bytes> (64 hex chars after prefix).
const APIKeyPrefix = "ink_"

const (
	maxAPIKeysPerUser = 20
	apiKeySecretBytes = 32
)

// AllowedAPIKeyScopes is the allow-list for key scopes at creation time.
// Scopes mirror RBAC resource:action strings; runtime enforcement is RBAC ∩ key scope.
// Keep destructive / settings:manage out of defaults in the UI; they remain opt-in here.
var AllowedAPIKeyScopes = map[string]struct{}{
	// Media (PicGo / uploads)
	"media:create": {},
	"media:read":   {},
	// Articles (local agents: SEO, body edit, publish)
	"articles:read":    {},
	"articles:create":  {},
	"articles:update":  {},
	"articles:publish": {},
	"articles:delete":  {},
	// Unified pages (draft / SEO / publish)
	"pages:read":    {},
	"pages:create":  {},
	"pages:update":  {},
	"pages:publish": {},
	"pages:delete":  {},
	// Taxonomy (optional for agent tag/category alignment)
	"categories:read":   {},
	"categories:create": {},
	"categories:update": {},
	"tags:read":         {},
	"tags:create":       {},
	"tags:update":       {},
}

// AllowedAPIKeyScopeList returns scopes in stable order for docs/UI/tests.
func AllowedAPIKeyScopeList() []string {
	return []string{
		"media:create",
		"media:read",
		"articles:read",
		"articles:create",
		"articles:update",
		"articles:publish",
		"articles:delete",
		"pages:read",
		"pages:create",
		"pages:update",
		"pages:publish",
		"pages:delete",
		"categories:read",
		"categories:create",
		"categories:update",
		"tags:read",
		"tags:create",
		"tags:update",
	}
}

var (
	ErrAPIKeyNotFound     = errors.New("api key not found")
	ErrAPIKeyInvalidName  = errors.New("name must be 1-64 characters")
	ErrAPIKeyInvalidScope = errors.New("invalid or empty scopes")
	ErrAPIKeyLimit        = errors.New("too many api keys for this user")
)

// APIKeyService manages personal access tokens.
type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// Create issues a new key. Plaintext is returned only once.
func (s *APIKeyService) Create(ctx context.Context, userID uint, name string, scopes []string) (plaintext string, key *model.APIKey, err error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 {
		return "", nil, ErrAPIKeyInvalidName
	}
	scopes, err = normalizeScopes(scopes)
	if err != nil {
		return "", nil, err
	}

	var n int64
	if err := s.db.WithContext(ctx).Model(&model.APIKey{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return "", nil, err
	}
	if n >= maxAPIKeysPerUser {
		return "", nil, ErrAPIKeyLimit
	}

	plain, err := generateAPIKeyPlaintext()
	if err != nil {
		return "", nil, err
	}
	rec := &model.APIKey{
		UserID:      userID,
		Name:        name,
		TokenPrefix: plain[:min(12, len(plain))], // ink_ + 4 hex
		TokenHash:   hashAPIKey(plain),
		Scopes:      model.StringSlice(scopes),
	}
	if err := rec.Validate(); err != nil {
		return "", nil, err
	}
	if err := s.db.WithContext(ctx).Create(rec).Error; err != nil {
		return "", nil, err
	}
	return plain, rec, nil
}

// List returns keys for a user (no secrets).
func (s *APIKeyService) List(ctx context.Context, userID uint) ([]model.APIKey, error) {
	var list []model.APIKey
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("id desc").Find(&list).Error
	return list, err
}

// Revoke deletes a key owned by userID.
func (s *APIKeyService) Revoke(ctx context.Context, userID, id uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.APIKey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// APIKeyPrincipal is the result of authenticating a plaintext API key.
type APIKeyPrincipal struct {
	UserID   uint
	Username string
	Role     model.Role
	Scopes   []string
	KeyID    uint
}

// Authenticate validates plaintext and returns the owning user principal.
// Returns (nil, nil) when the token is not an API key (wrong prefix).
func (s *APIKeyService) Authenticate(ctx context.Context, plaintext string) (*APIKeyPrincipal, error) {
	if !strings.HasPrefix(plaintext, APIKeyPrefix) {
		return nil, nil
	}
	var key model.APIKey
	err := s.db.WithContext(ctx).Where("token_hash = ?", hashAPIKey(plaintext)).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w", ErrAPIKeyNotFound)
	}
	if err != nil {
		return nil, err
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, key.UserID).Error; err != nil {
		return nil, err
	}
	// throttle last_used_at writes
	if key.LastUsedAt == nil || time.Since(*key.LastUsedAt) > time.Minute {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&key).Update("last_used_at", &now).Error
	}
	return &APIKeyPrincipal{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Scopes:   append([]string(nil), key.Scopes...),
		KeyID:    key.ID,
	}, nil
}

func normalizeScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		// default for PicGo / media clients
		return []string{"media:create"}, nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := AllowedAPIKeyScopes[s]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrAPIKeyInvalidScope, s)
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, ErrAPIKeyInvalidScope
	}
	return out, nil
}

func generateAPIKeyPlaintext() (string, error) {
	buf := make([]byte, apiKeySecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return APIKeyPrefix + hex.EncodeToString(buf), nil
}

func hashAPIKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
