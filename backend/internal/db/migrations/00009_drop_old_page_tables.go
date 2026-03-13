package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDropOldPageTables, downDropOldPageTables)
}

// upDropOldPageTables drops the old content_versions table which is no longer
// written to.  The content_documents and pages tables are kept for now because
// the bootstrap, public, sitemap, QA, and seed systems still read from them.
// A future migration should drop those once those subsystems are ported to the
// unified page model.
func upDropOldPageTables(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS content_versions")
	return err
}

func downDropOldPageTables(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS content_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			page_key TEXT NOT NULL,
			version INTEGER NOT NULL,
			config TEXT NOT NULL,
			created_by INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME
		)
	`)
	return err
}
