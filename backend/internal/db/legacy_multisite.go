package db

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"
)

// LegacyTableStatus reports whether a retired multi-site table still exists
// and, when it does, how many rows it contains.
type LegacyTableStatus struct {
	Exists bool  `json:"exists"`
	Rows   int64 `json:"rows"`
}

// LegacyColumnStatus reports whether a retired site scope column still exists
// and, when it does, how many rows contain a non-NULL value.
type LegacyColumnStatus struct {
	Exists      bool  `json:"exists"`
	NonNullRows int64 `json:"nonNullRows"`
}

// LegacyMultiSiteStatus is the read-only preflight report for schema artifacts
// from the experimental shared-database multi-site implementation.
type LegacyMultiSiteStatus struct {
	Sites          LegacyTableStatus  `json:"sites"`
	SiteUsers      LegacyTableStatus  `json:"siteUsers"`
	RoleSiteID     LegacyColumnStatus `json:"roleSiteId"`
	UserRoleSiteID LegacyColumnStatus `json:"userRoleSiteId"`
	HasLegacyData  bool               `json:"hasLegacyData"`
	Recommendation string             `json:"recommendation"`
}

const (
	legacyDataRecommendation   = "legacy multi-site data found; back up the database and export the legacy data before upgrading; this command does not modify or drop schema"
	noLegacyDataRecommendation = "no legacy multi-site rows or scoped role assignments found; this command does not modify or drop schema"
)

// InspectLegacyMultiSiteSchema counts retired multi-site tables and scoped RBAC
// values. It performs no schema or data mutations.
func InspectLegacyMultiSiteSchema(ctx context.Context, database *DB) (LegacyMultiSiteStatus, error) {
	status := LegacyMultiSiteStatus{}
	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		status.Sites, err = inspectLegacyTable(tx, "sites")
		if err != nil {
			return err
		}
		status.SiteUsers, err = inspectLegacyTable(tx, "site_users")
		if err != nil {
			return err
		}
		status.RoleSiteID, err = inspectLegacyColumn(tx, "roles", "site_id")
		if err != nil {
			return err
		}
		status.UserRoleSiteID, err = inspectLegacyColumn(tx, "user_roles", "site_id")
		return err
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return status, fmt.Errorf("inspect legacy multi-site schema in read-only transaction: %w", err)
	}

	status.HasLegacyData = status.Sites.Rows > 0 ||
		status.SiteUsers.Rows > 0 ||
		status.RoleSiteID.NonNullRows > 0 ||
		status.UserRoleSiteID.NonNullRows > 0
	status.Recommendation = noLegacyDataRecommendation
	if status.HasLegacyData {
		status.Recommendation = legacyDataRecommendation
	}

	return status, nil
}

func inspectLegacyTable(database *gorm.DB, table string) (LegacyTableStatus, error) {
	status := LegacyTableStatus{Exists: database.Migrator().HasTable(table)}
	if !status.Exists {
		return status, nil
	}
	if err := database.Table(table).Count(&status.Rows).Error; err != nil {
		return status, fmt.Errorf("count legacy table %s: %w", table, err)
	}
	return status, nil
}

func inspectLegacyColumn(database *gorm.DB, table, column string) (LegacyColumnStatus, error) {
	status := LegacyColumnStatus{Exists: database.Migrator().HasTable(table) && database.Migrator().HasColumn(table, column)}
	if !status.Exists {
		return status, nil
	}
	if err := database.Table(table).Where(column + " IS NOT NULL").Count(&status.NonNullRows).Error; err != nil {
		return status, fmt.Errorf("count legacy column %s.%s: %w", table, column, err)
	}
	return status, nil
}
