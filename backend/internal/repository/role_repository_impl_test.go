package repository

import (
	"context"
	"testing"

	"blotting-consultancy/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleRepositoryExcludesLegacyScopedRows(t *testing.T) {
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
	userRepo := NewGormUserRepository(database.DB)
	roleRepo := NewGormRoleRepository(database.DB)
	user := &model.User{Username: "role-scope-user", PasswordHash: "hash", Role: model.RoleAdmin}
	require.NoError(t, userRepo.Create(ctx, user))

	globalRole := &model.RBACRole{Name: "global_role", DisplayName: "Global Role"}
	legacyRole := &model.RBACRole{Name: "legacy_role", DisplayName: "Legacy Role"}
	require.NoError(t, database.Create(globalRole).Error)
	require.NoError(t, database.Create(legacyRole).Error)
	require.NoError(t, database.Model(&model.RBACRole{}).
		Where("id = ?", legacyRole.ID).
		Update("site_id", 7).Error)
	legacyPermission := &model.Permission{Resource: "users", Action: "delete"}
	require.NoError(t, database.Create(legacyPermission).Error)
	require.NoError(t, database.Model(legacyRole).Association("Permissions").Append(legacyPermission))

	require.NoError(t, roleRepo.AssignRoleToUser(ctx, user.ID, globalRole.ID))
	require.NoError(t, database.Create(&model.UserRole{UserID: user.ID, RoleID: legacyRole.ID}).Error)
	require.NoError(t, database.Model(&model.UserRole{}).
		Where("user_id = ? AND role_id = ?", user.ID, legacyRole.ID).
		Update("site_id", 7).Error)

	roles, err := roleRepo.List(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, globalRole.ID, roles[0].ID)

	_, err = roleRepo.FindByID(ctx, legacyRole.ID)
	require.EqualError(t, err, "role not found")
	_, err = roleRepo.FindByName(ctx, legacyRole.Name)
	require.EqualError(t, err, "role not found")
	require.EqualError(t, roleRepo.AssignRoleToUser(ctx, user.ID, legacyRole.ID), "role not found")

	userRoles, err := roleRepo.GetUserRoles(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, userRoles, 1)
	assert.Equal(t, globalRole.ID, userRoles[0].RoleID)

	count, err := roleRepo.CountUsersWithRole(ctx, legacyRole.ID)
	require.NoError(t, err)
	assert.Zero(t, count)
	require.EqualError(t, roleRepo.RemoveRoleFromUser(ctx, user.ID, legacyRole.ID), "user role assignment not found")
	require.EqualError(t, roleRepo.Delete(ctx, legacyRole.ID), "role not found")

	var stored int64
	require.NoError(t, database.Table("user_roles").
		Where("user_id = ? AND role_id = ? AND site_id = ?", user.ID, legacyRole.ID, 7).
		Count(&stored).Error)
	assert.EqualValues(t, 1, stored)
	stored = 0
	require.NoError(t, database.Table("roles").
		Where("id = ? AND site_id = ?", legacyRole.ID, 7).
		Count(&stored).Error)
	assert.EqualValues(t, 1, stored)
	stored = 0
	require.NoError(t, database.Table("role_permissions").
		Where("rbac_role_id = ? AND permission_id = ?", legacyRole.ID, legacyPermission.ID).
		Count(&stored).Error)
	assert.EqualValues(t, 1, stored)
}
