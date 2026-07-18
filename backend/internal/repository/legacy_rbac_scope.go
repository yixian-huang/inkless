package repository

import "gorm.io/gorm"

type legacyRBACScope struct {
	roles     bool
	userRoles bool
}

func detectLegacyRBACScope(db *gorm.DB) legacyRBACScope {
	return legacyRBACScope{
		roles:     db.Migrator().HasColumn("roles", "site_id"),
		userRoles: db.Migrator().HasColumn("user_roles", "site_id"),
	}
}

func (s legacyRBACScope) currentRoles(query *gorm.DB) *gorm.DB {
	if s.roles {
		return query.Where("roles.site_id IS NULL")
	}
	return query
}

func (s legacyRBACScope) currentUserRoles(query *gorm.DB) *gorm.DB {
	if s.userRoles {
		query = query.Where("user_roles.site_id IS NULL")
	}
	if s.roles {
		query = query.Where("user_roles.role_id IN (SELECT id FROM roles WHERE site_id IS NULL)")
	}
	return query
}
