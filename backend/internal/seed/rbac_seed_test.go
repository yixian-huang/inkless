package seed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

func TestSeedRBACReframesSiteAdminAndLeavesLegacySitePermissionsUnexposed(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.RBACRole{},
		&model.Permission{},
		&model.UserRole{},
	))

	roleRepo := repository.NewGormRoleRepository(database)
	ctx := context.Background()
	legacyPermission := &model.Permission{Resource: "sites", Action: "read", Description: "legacy"}
	require.NoError(t, roleRepo.CreatePermission(ctx, legacyPermission))
	legacyRole := &model.RBACRole{
		Name:        model.BuiltinRoleSiteAdmin,
		DisplayName: "Site Admin",
		Description: "Full site management access except system-level operations",
		IsSystem:    true,
	}
	require.NoError(t, roleRepo.Create(ctx, legacyRole))
	require.NoError(t, roleRepo.SetPermissions(ctx, legacyRole.ID, []uint{legacyPermission.ID}))

	require.NoError(t, SeedRBAC(ctx, roleRepo))

	updated, err := roleRepo.FindByName(ctx, model.BuiltinRoleSiteAdmin)
	require.NoError(t, err)
	assert.Contains(t, updated.Description, "current instance")
	for _, permission := range updated.Permissions {
		assert.NotEqual(t, "sites", permission.Resource)
	}

	stillStored, err := roleRepo.FindPermissionByResourceAction(ctx, "sites", "read")
	require.NoError(t, err)
	assert.Equal(t, legacyPermission.ID, stillStored.ID)
}
