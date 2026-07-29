package service

import (
	"context"
	"strings"
	"testing"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAPIKeyDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Unique DSN per test so parallel/sequential cases do not share tables.
	dsn := "file:api_key_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.APIKey{}); err != nil {
		t.Fatal(err)
	}
	u := &model.User{Username: "alice", PasswordHash: "x", Role: model.RoleAdmin, IsSuperAdmin: true}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAPIKeyCreateAuthRevoke(t *testing.T) {
	db := setupAPIKeyDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()

	var u model.User
	if err := db.First(&u, "username = ?", "alice").Error; err != nil {
		t.Fatal(err)
	}

	plain, key, err := svc.Create(ctx, u.ID, "picgo", []string{"media:create"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plain, APIKeyPrefix) || len(plain) < 40 {
		t.Fatalf("plaintext shape: %q", plain)
	}
	if key.TokenHash == plain || key.TokenHash == "" {
		t.Fatal("must store hash not plaintext")
	}
	if key.TokenPrefix == "" || !strings.HasPrefix(key.TokenPrefix, "ink_") {
		t.Fatalf("tokenPrefix=%q", key.TokenPrefix)
	}

	p, err := svc.Authenticate(ctx, plain)
	if err != nil || p == nil || p.UserID != u.ID || p.Username != "alice" {
		t.Fatalf("auth: %+v %v", p, err)
	}
	if len(p.Scopes) != 1 || p.Scopes[0] != "media:create" {
		t.Fatalf("scopes=%v", p.Scopes)
	}

	// wrong secret
	if _, err := svc.Authenticate(ctx, APIKeyPrefix+"00"); err == nil {
		t.Fatal("expected not found")
	}

	// not a key prefix → nil,nil
	got, err := svc.Authenticate(ctx, "jwt-looking-token")
	if err != nil || got != nil {
		t.Fatalf("non-key should be nil nil, got %+v %v", got, err)
	}

	list, err := svc.List(ctx, u.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}

	if err := svc.Revoke(ctx, u.ID, key.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, u.ID, key.ID); err != ErrAPIKeyNotFound {
		t.Fatalf("second revoke: %v", err)
	}
	if p, err := svc.Authenticate(ctx, plain); err == nil || p != nil {
		t.Fatal("revoked key must not authenticate")
	}
}

func TestAPIKeyDefaultScopeAndInvalid(t *testing.T) {
	db := setupAPIKeyDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	var u model.User
	_ = db.First(&u, "username = ?", "alice")

	plain, key, err := svc.Create(ctx, u.ID, "default-scope", nil)
	if err != nil || plain == "" {
		t.Fatal(err)
	}
	if len(key.Scopes) != 1 || key.Scopes[0] != "media:create" {
		t.Fatalf("default scopes=%v", key.Scopes)
	}

	// settings:manage is intentionally not grantable via personal API keys
	if _, _, err := svc.Create(ctx, u.ID, "bad", []string{"settings:manage"}); err == nil {
		t.Fatal("want invalid scope")
	}
	if _, _, err := svc.Create(ctx, u.ID, "bad2", []string{"articles:foo"}); err == nil {
		t.Fatal("want invalid scope")
	}
	if _, _, err := svc.Create(ctx, u.ID, "", []string{"media:create"}); err == nil {
		t.Fatal("want invalid name")
	}
}

func TestAPIKeyContentScopesForAgent(t *testing.T) {
	db := setupAPIKeyDB(t)
	svc := NewAPIKeyService(db)
	ctx := context.Background()
	var u model.User
	_ = db.First(&u, "username = ?", "alice")

	scopes := []string{
		"articles:read",
		"articles:update",
		"pages:read",
		"pages:update",
		"media:create",
		"categories:read",
		"tags:read",
	}
	plain, key, err := svc.Create(ctx, u.ID, "content-agent", scopes)
	if err != nil {
		t.Fatal(err)
	}
	if plain == "" || key == nil {
		t.Fatal("expected token and key")
	}
	if len(key.Scopes) != len(scopes) {
		t.Fatalf("scopes=%v", key.Scopes)
	}
	p, err := svc.Authenticate(ctx, plain)
	if err != nil || p == nil {
		t.Fatalf("auth: %+v %v", p, err)
	}
	if len(p.Scopes) != len(scopes) {
		t.Fatalf("auth scopes=%v", p.Scopes)
	}

	// allow-list helper stays in sync with map
	list := AllowedAPIKeyScopeList()
	if len(list) == 0 {
		t.Fatal("empty allow list")
	}
	for _, s := range list {
		if _, ok := AllowedAPIKeyScopes[s]; !ok {
			t.Fatalf("list has %q not in map", s)
		}
	}
	for s := range AllowedAPIKeyScopes {
		found := false
		for _, x := range list {
			if x == s {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("map has %q not in list", s)
		}
	}
}
