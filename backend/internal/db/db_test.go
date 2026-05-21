package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestIsPostgresDSN(t *testing.T) {
	assert.True(t, IsPostgresDSN("postgres://user:pass@localhost:5432/dbname?sslmode=disable"))
	assert.True(t, IsPostgresDSN("host=localhost user=app password=secret dbname=appdb port=5432 sslmode=disable"))
	assert.False(t, IsPostgresDSN(":memory:"))
	assert.False(t, IsPostgresDSN("file:./data/impress.db?cache=shared&mode=rwc"))
}

func TestInit_SQLite(t *testing.T) {
	opts := InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	}

	db, err := Init(opts)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify database is initialized
	sqlDB, err := db.DB.DB()
	require.NoError(t, err)
	require.NotNil(t, sqlDB)
}

func TestInit_SQLiteCreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	dbPath := filepath.Join(baseDir, "nested", "impress.db")

	db, err := Init(InitOptions{
		DSN:      dbPath,
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	_, statErr := os.Stat(filepath.Dir(dbPath))
	require.NoError(t, statErr)
}

func TestInit_EmptyDSN(t *testing.T) {
	_, err := Init(InitOptions{
		DSN:      "",
		LogLevel: logger.Silent,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DSN cannot be empty")
}

func TestInit_WithConnectionPoolSettings(t *testing.T) {
	opts := InitOptions{
		DSN:         ":memory:",
		MaxOpenConn: 10,
		MaxIdleConn: 5,
		MaxLifetime: 30 * time.Minute,
		LogLevel:    logger.Silent,
	}

	db, err := Init(opts)
	require.NoError(t, err)
	defer db.Close()

	sqlDB, err := db.DB.DB()
	require.NoError(t, err)

	stats := sqlDB.Stats()
	assert.Equal(t, 10, stats.MaxOpenConnections)
}

func TestHealthCheck_Success(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestHealthCheck_AfterClose(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Close connection
	err = db.Close()
	require.NoError(t, err)

	// Health check should fail
	ctx := context.Background()
	err = db.HealthCheck(ctx)
	assert.Error(t, err)
}

func TestHealthCheck_WithContext(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.HealthCheck(ctx)
	assert.NoError(t, err)
}

// Test model for migration testing
type TestModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

func (TestModel) TableName() string {
	return "test_models"
}

func TestAutoMigrate(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	// Run auto migration
	err = migrator.AutoMigrate(&TestModel{})
	require.NoError(t, err)

	// Verify table exists
	hasTable := migrator.HasTable(&TestModel{})
	assert.True(t, hasTable)
}

func TestDropTable(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	// Create table first
	err = migrator.AutoMigrate(&TestModel{})
	require.NoError(t, err)
	assert.True(t, migrator.HasTable(&TestModel{}))

	// Drop table
	err = migrator.DropTable(&TestModel{})
	require.NoError(t, err)
	assert.False(t, migrator.HasTable(&TestModel{}))
}

func TestRunMigrations(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	migrations := []Migration{
		{
			ID: "001_create_test_table",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&TestModel{})
			},
			Down: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&TestModel{})
			},
		},
	}

	// Run migrations
	err = migrator.RunMigrations(migrations)
	require.NoError(t, err)

	// Verify migration was recorded
	var history MigrationHistory
	err = db.Where("migration = ?", "001_create_test_table").First(&history).Error
	require.NoError(t, err)
	assert.Equal(t, "001_create_test_table", history.Migration)

	// Verify table was created
	assert.True(t, migrator.HasTable(&TestModel{}))

	// Running migrations again should be idempotent
	err = migrator.RunMigrations(migrations)
	require.NoError(t, err)
}

func TestRunMigrations_MultipleInOrder(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	type SecondModel struct {
		ID   uint   `gorm:"primaryKey"`
		Code string `gorm:"not null"`
	}

	migrations := []Migration{
		{
			ID: "001_create_test_table",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&TestModel{})
			},
			Down: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&TestModel{})
			},
		},
		{
			ID: "002_create_second_table",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&SecondModel{})
			},
			Down: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&SecondModel{})
			},
		},
	}

	err = migrator.RunMigrations(migrations)
	require.NoError(t, err)

	// Verify both migrations were recorded
	var count int64
	err = db.Model(&MigrationHistory{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestRollbackMigration(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	migrations := []Migration{
		{
			ID: "001_create_test_table",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&TestModel{})
			},
			Down: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&TestModel{})
			},
		},
	}

	// Run migration
	err = migrator.RunMigrations(migrations)
	require.NoError(t, err)
	assert.True(t, migrator.HasTable(&TestModel{}))

	// Rollback migration
	err = migrator.RollbackMigration(migrations, "001_create_test_table")
	require.NoError(t, err)

	// Verify table was dropped
	assert.False(t, migrator.HasTable(&TestModel{}))

	// Verify migration record was removed
	var history MigrationHistory
	err = db.Where("migration = ?", "001_create_test_table").First(&history).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestRollbackMigration_NotApplied(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	// Ensure migration_history table exists
	err = migrator.AutoMigrate(&MigrationHistory{})
	require.NoError(t, err)

	migrations := []Migration{
		{
			ID: "001_create_test_table",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&TestModel{})
			},
			Down: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&TestModel{})
			},
		},
	}

	// Try to rollback a migration that was never applied
	err = migrator.RollbackMigration(migrations, "001_create_test_table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "was not applied")
}

func TestRollbackMigration_NotFound(t *testing.T) {
	db, err := Init(InitOptions{
		DSN:      ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)
	defer db.Close()

	migrator := NewMigrator(db)

	migrations := []Migration{}

	// Try to rollback a migration that doesn't exist
	err = migrator.RollbackMigration(migrations, "nonexistent_migration")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
