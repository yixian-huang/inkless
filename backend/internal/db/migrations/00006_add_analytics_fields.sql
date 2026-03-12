-- +goose Up
-- +goose StatementBegin
-- Columns may already exist from GORM AutoMigrate; use Go-based migration
-- to check before adding. For now, just create the index.
CREATE INDEX IF NOT EXISTS idx_pageview_visitor ON page_views(visitor_id);
-- +goose StatementEnd

-- +goose Down
-- Best-effort: SQLite <3.35 doesn't support DROP COLUMN
DROP INDEX IF EXISTS idx_pageview_visitor;
