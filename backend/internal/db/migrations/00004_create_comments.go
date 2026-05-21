package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(up00004, down00004)
}

// up00004 is a no-op for both SQLite and Postgres. The `comments` table is
// created by the GORM AutoMigrate on `commentMod.Comment` in `main.go` (added
// to the central AutoMigrate list to make table creation cross-dialect; the
// original 00004_create_comments.sql used SQLite's `AUTOINCREMENT` which
// Postgres rejects).
//
// Why keep the version row instead of deleting the migration file:
// existing SQLite production databases already have version 00004 recorded
// in goose_db_version. Removing the file would orphan that row.
func up00004(tx *sql.Tx) error { return nil }

func down00004(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS comments")
	return err
}
