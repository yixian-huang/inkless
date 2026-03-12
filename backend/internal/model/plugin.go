package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Plugin stores installed plugin state in the database.
type Plugin struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PluginID    string         `gorm:"uniqueIndex;size:100;not null" json:"pluginId"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	NameZh      string         `gorm:"size:200" json:"nameZh"`
	Version     string         `gorm:"size:50;not null" json:"version"`
	Description string         `gorm:"size:1000" json:"description"`
	Author      string         `gorm:"size:200" json:"author"`
	License     string         `gorm:"size:100" json:"license"`
	Homepage    string         `gorm:"size:500" json:"homepage"`
	State       string         `gorm:"size:20;not null;default:'installed'" json:"state"`
	Source      string         `gorm:"size:20;not null;default:'local'" json:"source"` // local, marketplace
	BinaryPath  string         `gorm:"size:500" json:"binaryPath"`
	Permissions JSONStringSlice `gorm:"type:text" json:"permissions"`
	Settings    JSONMap        `gorm:"type:text" json:"settings"`
	ErrorMsg    string         `gorm:"size:2000" json:"errorMsg,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the default table name.
func (Plugin) TableName() string { return "plugins" }

// PluginSetting stores per-plugin configuration key-value pairs.
type PluginSetting struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	PluginID string `gorm:"index;size:100;not null" json:"pluginId"`
	Key      string `gorm:"size:200;not null" json:"key"`
	Value    string `gorm:"type:text" json:"value"`
}

// TableName overrides the default table name.
func (PluginSetting) TableName() string { return "plugin_settings" }

// JSONStringSlice is a []string that serializes to/from JSON in the database.
type JSONStringSlice []string

// Value implements the driver.Valuer interface for database serialization.
func (j JSONStringSlice) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database deserialization.
func (j *JSONStringSlice) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONStringSlice.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}
