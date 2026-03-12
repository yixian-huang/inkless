package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// Service handles database backup operations
type Service struct {
	db         *gorm.DB
	backupDir  string
	maxKeep    int
	uploadDir  string
	appVersion string
	importMu   sync.Mutex
}

// NewService creates a new backup service
func NewService(db *gorm.DB, backupDir string, maxKeep int, uploadDir, appVersion string) *Service {
	if maxKeep <= 0 {
		maxKeep = 10
	}
	return &Service{
		db:         db,
		backupDir:  backupDir,
		maxKeep:    maxKeep,
		uploadDir:  uploadDir,
		appVersion: appVersion,
	}
}

// BackupDir returns the backup directory path.
func (s *Service) BackupDir() string {
	return s.backupDir
}

// RunBackup creates a database backup, compresses it, and records the result
func (s *Service) RunBackup(ctx context.Context) (*model.BackupRecord, error) {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")

	if s.isPostgres() {
		return s.backupPostgres(ctx, timestamp)
	}
	return s.backupSQLite(ctx, timestamp)
}

// ListBackups returns all backup records ordered by creation time (newest first)
func (s *Service) ListBackups(ctx context.Context) ([]model.BackupRecord, error) {
	var records []model.BackupRecord
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Service) isPostgres() bool {
	dialectorName := s.db.Dialector.Name()
	return dialectorName == "postgres"
}

func (s *Service) backupSQLite(ctx context.Context, timestamp string) (*model.BackupRecord, error) {
	// Get the SQLite database file path
	var dbPath string
	row := s.db.WithContext(ctx).Raw("PRAGMA database_list").Row()
	var seq int
	var name, file string
	if err := row.Scan(&seq, &name, &file); err != nil {
		return nil, fmt.Errorf("get sqlite db path: %w", err)
	}
	dbPath = file

	if dbPath == "" {
		return nil, fmt.Errorf("cannot determine SQLite database file path (in-memory?)")
	}

	// Create a temporary copy using VACUUM INTO if available, else file copy
	tmpPath := filepath.Join(s.backupDir, fmt.Sprintf("backup-%s.db", timestamp))
	err := s.db.WithContext(ctx).Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpPath)).Error
	if err != nil {
		// Fallback: file copy
		if cpErr := copyFile(dbPath, tmpPath); cpErr != nil {
			return nil, fmt.Errorf("sqlite backup failed: vacuum=%w, copy=%v", err, cpErr)
		}
	}

	// Compress
	gzPath := tmpPath + ".gz"
	if err := compressFile(tmpPath, gzPath); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("compress backup: %w", err)
	}
	os.Remove(tmpPath)

	// Get file size
	info, err := os.Stat(gzPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	record := &model.BackupRecord{
		Filename: filepath.Base(gzPath),
		Size:     info.Size(),
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		return nil, fmt.Errorf("save backup record: %w", err)
	}

	s.cleanOldBackups(ctx)
	return record, nil
}

func (s *Service) backupPostgres(ctx context.Context, timestamp string) (*model.BackupRecord, error) {
	// Get DSN from the database connection
	sqlDB, err := s.db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	// Use pg_dump via the connection's DSN
	dumpPath := filepath.Join(s.backupDir, fmt.Sprintf("backup-%s.sql", timestamp))
	gzPath := dumpPath + ".gz"

	// Build pg_dump command. We rely on DATABASE_URL or PG* env vars being set.
	cmd := exec.CommandContext(ctx, "pg_dump", "--no-owner", "--no-acl", "-f", dumpPath)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr

	// Verify the sql.DB is alive before running pg_dump
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database not reachable: %w", err)
	}

	if err := cmd.Run(); err != nil {
		os.Remove(dumpPath)
		return nil, fmt.Errorf("pg_dump: %w", err)
	}

	if err := compressFile(dumpPath, gzPath); err != nil {
		os.Remove(dumpPath)
		return nil, fmt.Errorf("compress backup: %w", err)
	}
	os.Remove(dumpPath)

	info, err := os.Stat(gzPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	record := &model.BackupRecord{
		Filename: filepath.Base(gzPath),
		Size:     info.Size(),
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		return nil, fmt.Errorf("save backup record: %w", err)
	}

	s.cleanOldBackups(ctx)
	return record, nil
}

func (s *Service) cleanOldBackups(ctx context.Context) {
	var records []model.BackupRecord
	if err := s.db.WithContext(ctx).Order("created_at DESC").Find(&records).Error; err != nil {
		return
	}

	if len(records) <= s.maxKeep {
		return
	}

	// Remove excess records and their files
	toDelete := records[s.maxKeep:]
	for _, rec := range toDelete {
		filePath := filepath.Join(s.backupDir, rec.Filename)
		os.Remove(filePath)
		s.db.WithContext(ctx).Delete(&rec)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func compressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()

	_, err = io.Copy(gz, in)
	return err
}

// BackupFiles returns the list of backup files in the backup directory sorted by modification time (newest first)
func (s *Service) BackupFiles() ([]string, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gz") {
			files = append(files, entry.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files, nil
}
