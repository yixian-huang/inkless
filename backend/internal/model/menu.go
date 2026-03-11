package model

import (
	"errors"
	"time"
)

// MenuGroup represents a named collection of menu items
type MenuGroup struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"not null;size:100" json:"name"`
	Slug      string     `gorm:"uniqueIndex;not null;size:100" json:"slug"`
	IsPrimary bool       `gorm:"default:false" json:"isPrimary"`
	Items     []MenuItem `gorm:"foreignKey:GroupID" json:"items,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

// Validate validates the menu group model
func (g *MenuGroup) Validate() error {
	if g.Name == "" {
		return errors.New("name is required")
	}
	if g.Slug == "" {
		return errors.New("slug is required")
	}
	return nil
}

// MenuItem represents a single item within a menu group
type MenuItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"not null;index" json:"groupId"`
	ParentID  *uint     `gorm:"index" json:"parentId"`
	ZhName    string    `gorm:"not null;size:200" json:"zhName"`
	EnName    string    `gorm:"size:200" json:"enName"`
	Type      string    `gorm:"not null;size:30" json:"type"` // custom_link, article, page, category, tag
	Target    string    `gorm:"size:10;default:'_self'" json:"target"` // _self, _parent, _blank, _top
	URL       string    `gorm:"size:500" json:"url"`
	RefID     *uint     `json:"refId"`
	RefSlug   string    `gorm:"size:200" json:"refSlug"`
	Visible   *bool     `gorm:"default:true" json:"visible"`
	Metadata  JSONMap   `gorm:"type:jsonb" json:"metadata"`
	SortOrder int       `gorm:"default:0" json:"sortOrder"`
	Children  []MenuItem `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// Validate validates the menu item model
func (i *MenuItem) Validate() error {
	if i.ZhName == "" {
		return errors.New("zhName is required")
	}
	if i.Type == "" {
		return errors.New("type is required")
	}
	validTypes := map[string]bool{
		"custom_link": true,
		"article":     true,
		"page":        true,
		"category":    true,
		"tag":         true,
	}
	if !validTypes[i.Type] {
		return errors.New("type must be one of: custom_link, article, page, category, tag")
	}
	return nil
}
