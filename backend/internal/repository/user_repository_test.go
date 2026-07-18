package repository

import (
	"context"
	"testing"

	"blotting-consultancy/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_CreateAndFind(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewGormUserRepository(database.DB)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Role:         model.RoleAdmin,
	}

	// Test Create
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}

	// Test FindByID
	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to find user by ID: %v", err)
	}

	if found.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, found.Username)
	}

	// Test FindByUsername
	found2, err := repo.FindByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to find user by username: %v", err)
	}

	if found2.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, found2.ID)
	}
}

func TestUserRepository_UpdateAndDelete(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewGormUserRepository(database.DB)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Role:         model.RoleAdmin,
	}

	_ = repo.Create(ctx, user)

	// Test Update
	user.Role = model.RoleEditor
	err := repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	found, _ := repo.FindByID(ctx, user.ID)
	if found.Role != model.RoleEditor {
		t.Errorf("Expected role %s, got %s", model.RoleEditor, found.Role)
	}

	// Test Delete
	err = repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	_, err = repo.FindByID(ctx, user.ID)
	if err == nil {
		t.Error("Expected error when finding deleted user")
	}
}

func TestUserRepository_List(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewGormUserRepository(database.DB)
	ctx := context.Background()

	// Create 5 users
	for i := 0; i < 5; i++ {
		user := &model.User{
			Username:     "testuser" + string(rune('a'+i)),
			PasswordHash: "hashedpassword",
			Role:         model.RoleAdmin,
		}
		_ = repo.Create(ctx, user)
	}

	users, total, err := repo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected 5 users, got %d", total)
	}

	if len(users) != 5 {
		t.Errorf("Expected 5 users in list, got %d", len(users))
	}

	// Test pagination
	users, _, _ = repo.List(ctx, 0, 3)
	if len(users) != 3 {
		t.Errorf("Expected 3 users in first page, got %d", len(users))
	}
}

func TestFindByIDWithRolesHidesLegacySitePermissions(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()
	require.NoError(t, database.AutoMigrate(
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
	))

	ctx := context.Background()
	repo := NewGormUserRepository(database.DB)
	user := &model.User{Username: "permission-user", PasswordHash: "hash", Role: model.RoleAdmin}
	require.NoError(t, repo.Create(ctx, user))
	role := &model.RBACRole{Name: "permission-role", DisplayName: "Permission Role"}
	require.NoError(t, database.Create(role).Error)
	legacy := &model.Permission{Resource: model.LegacyResourceSites, Action: "read"}
	current := &model.Permission{Resource: "settings", Action: "read"}
	require.NoError(t, database.Create(legacy).Error)
	require.NoError(t, database.Create(current).Error)
	require.NoError(t, database.Model(role).Association("Permissions").Append(legacy, current))
	require.NoError(t, database.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error)

	found, err := repo.FindByIDWithRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"settings:read"}, found.EffectivePermissionKeys())

	var stored int64
	require.NoError(t, database.Model(&model.Permission{}).
		Where("resource = ?", model.LegacyResourceSites).
		Count(&stored).Error)
	assert.EqualValues(t, 1, stored)
}

func TestFindByIDWithRolesIgnoresLegacyScopedAssignments(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()
	require.NoError(t, database.AutoMigrate(
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
	))
	require.NoError(t, database.Exec("ALTER TABLE roles ADD COLUMN site_id INTEGER").Error)
	require.NoError(t, database.Exec("ALTER TABLE user_roles ADD COLUMN site_id INTEGER").Error)

	ctx := context.Background()
	repo := NewGormUserRepository(database.DB)
	user := &model.User{Username: "legacy-scope-user", PasswordHash: "hash", Role: model.RoleAdmin}
	require.NoError(t, repo.Create(ctx, user))

	globalRole := &model.RBACRole{Name: "global_role", DisplayName: "Global Role"}
	legacyRole := &model.RBACRole{Name: "legacy_role", DisplayName: "Legacy Role"}
	require.NoError(t, database.Create(globalRole).Error)
	require.NoError(t, database.Create(legacyRole).Error)
	require.NoError(t, database.Model(&model.RBACRole{}).
		Where("id = ?", legacyRole.ID).
		Update("site_id", 42).Error)

	globalPermission := &model.Permission{Resource: "settings", Action: "read"}
	legacyPermission := &model.Permission{Resource: "users", Action: "delete"}
	require.NoError(t, database.Create(globalPermission).Error)
	require.NoError(t, database.Create(legacyPermission).Error)
	require.NoError(t, database.Model(globalRole).Association("Permissions").Append(globalPermission))
	require.NoError(t, database.Model(legacyRole).Association("Permissions").Append(legacyPermission))

	require.NoError(t, database.Create(&model.UserRole{UserID: user.ID, RoleID: globalRole.ID}).Error)
	require.NoError(t, database.Create(&model.UserRole{UserID: user.ID, RoleID: legacyRole.ID}).Error)
	require.NoError(t, database.Model(&model.UserRole{}).
		Where("user_id = ? AND role_id = ?", user.ID, legacyRole.ID).
		Update("site_id", 42).Error)

	found, err := repo.FindByIDWithRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"settings:read"}, found.EffectivePermissionKeys())
	require.Len(t, found.UserRoles, 1)
	assert.Equal(t, globalRole.ID, found.UserRoles[0].RoleID)
}
