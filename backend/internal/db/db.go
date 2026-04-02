package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps the GORM database connection
type DB struct {
	*gorm.DB
}

// InitOptions holds configuration for database initialization
type InitOptions struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
	MaxLifetime time.Duration
	LogLevel    logger.LogLevel
}

// IsPostgresDSN reports whether a DSN should use PostgreSQL dialector.
func IsPostgresDSN(dsn string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dsn))
	if strings.HasPrefix(normalized, "postgres://") || strings.HasPrefix(normalized, "postgresql://") {
		return true
	}

	// libpq-style DSN, e.g. "host=... user=... dbname=..."
	signals := 0
	for _, key := range []string{"host=", "dbname=", "user=", "password=", "sslmode="} {
		if strings.Contains(normalized, key) {
			signals++
		}
	}

	return signals >= 2
}

// Init initializes a GORM database connection based on the DSN
// Supports SQLite (file:*.db or :memory:) and PostgreSQL (postgres:// or libpq key=value)
func Init(opts InitOptions) (*DB, error) {
	dsn := strings.TrimSpace(opts.DSN)
	if dsn == "" {
		return nil, fmt.Errorf("database DSN cannot be empty")
	}

	usePostgres := IsPostgresDSN(dsn)
	if !usePostgres {
		if err := ensureSQLiteDirectory(dsn); err != nil {
			return nil, err
		}
	}

	var dialector gorm.Dialector
	if usePostgres {
		dialector = postgres.Open(dsn)
	} else {
		dialector = sqlite.Open(dsn)
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(opts.LogLevel),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if !usePostgres {
		if err := applySQLitePragmas(db); err != nil {
			return nil, err
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if opts.MaxOpenConn > 0 {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConn)
	}
	if opts.MaxIdleConn > 0 {
		sqlDB.SetMaxIdleConns(opts.MaxIdleConn)
	}
	if opts.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(opts.MaxLifetime)
	}

	return &DB{DB: db}, nil
}

func ensureSQLiteDirectory(dsn string) error {
	dbFile, ok := sqliteFilePath(dsn)
	if !ok {
		return nil
	}

	dir := filepath.Dir(dbFile)
	if dir == "." || dir == "" {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create sqlite directory %q: %w", dir, err)
	}

	return nil
}

func sqliteFilePath(dsn string) (string, bool) {
	trimmed := strings.TrimSpace(dsn)
	lower := strings.ToLower(trimmed)
	if trimmed == "" || lower == ":memory:" || strings.HasPrefix(lower, "file::memory:") || strings.Contains(lower, "mode=memory") {
		return "", false
	}

	if strings.HasPrefix(lower, "file:") {
		filePath := trimmed[len("file:"):]
		if idx := strings.Index(filePath, "?"); idx >= 0 {
			filePath = filePath[:idx]
		}
		filePath = strings.TrimPrefix(filePath, "//")
		if filePath == "" || strings.EqualFold(filePath, ":memory:") {
			return "", false
		}
		return filePath, true
	}

	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if strings.Contains(trimmed, "://") {
		return "", false
	}

	return trimmed, true
}

func applySQLitePragmas(database *gorm.DB) error {
	statements := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, stmt := range statements {
		if err := database.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to apply sqlite pragma %q: %w", stmt, err)
		}
	}

	return nil
}

// HealthCheck performs a health check query (SELECT 1) to verify connection
func (db *DB) HealthCheck(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}
