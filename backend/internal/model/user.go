package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Role represents user role enum
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
)

// ValidRoles contains all valid role values
var ValidRoles = []Role{RoleAdmin, RoleEditor}

// IsValid checks if a role value is valid
func (r Role) IsValid() bool {
	for _, valid := range ValidRoles {
		if r == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the role
func (r Role) String() string {
	return string(r)
}

// StringSlice is a custom type for storing []string as JSON in the database
type StringSlice []string

// Value implements driver.Valuer for database storage
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner for database retrieval
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("unsupported type for StringSlice")
	}
	return json.Unmarshal(bytes, s)
}

// ValidPermissions contains all valid permission keys
var ValidPermissions = []string{
	"dashboard",
	"content",
	"pages",
	"articles",
	"media",
	"form-submissions",
	"menus",
	"theme",
	"analytics",
	"audit-logs",
	"backups",
	"users",
}

// User represents a user in the system
type User struct {
	ID           uint        `gorm:"primaryKey"`
	Username     string      `gorm:"uniqueIndex;not null;size:100"`
	PasswordHash string      `gorm:"not null;size:255"`
	Role         Role        `gorm:"not null;size:20"`
	IsSuperAdmin bool        `gorm:"not null;default:false"`
	Permissions  StringSlice `gorm:"type:text"`
	CreatedAt    time.Time   `gorm:"autoCreateTime"`
	UpdatedAt    time.Time   `gorm:"autoUpdateTime"`
}

// TableName overrides the default table name
func (User) TableName() string {
	return "users"
}

// Validate validates the user model
func (u *User) Validate() error {
	if u.Username == "" {
		return errors.New("username is required")
	}
	if u.PasswordHash == "" {
		return errors.New("password hash is required")
	}
	if !u.Role.IsValid() {
		return errors.New("role must be 'admin' or 'editor'")
	}
	return nil
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(perm string) bool {
	if u.IsSuperAdmin {
		return true
	}
	for _, p := range u.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}
