package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(up00007, down00007)
}

// up00007 is a no-op for both dialects. The RBAC tables (`roles`,
// `permissions`, `role_permissions`, `user_roles`) are created by GORM
// AutoMigrate on the corresponding models (`RBACRole`, `Permission`,
// `UserRole`) in `main.go`. The original SQL used SQLite's `AUTOINCREMENT`
// which Postgres rejects.
//
// Existing SQLite databases keep their applied-version row; new databases
// (either dialect) get the tables from AutoMigrate instead.
func up00007(tx *sql.Tx) error { return nil }

func down00007(tx *sql.Tx) error {
	for _, stmt := range []string{
		"DROP TABLE IF EXISTS user_roles",
		"DROP TABLE IF EXISTS role_permissions",
		"DROP TABLE IF EXISTS permissions",
		"DROP TABLE IF EXISTS roles",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
