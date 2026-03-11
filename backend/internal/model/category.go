package model

import "errors"

// Category represents an article category with hierarchical support
type Category struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Slug            string     `gorm:"uniqueIndex;not null;size:100" json:"slug"`
	ZhName          string     `gorm:"not null;size:100" json:"zhName"`
	EnName          string     `gorm:"size:100" json:"enName"`
	ParentID        *uint      `gorm:"index" json:"parentId"`
	Parent          *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children        []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	CoverImage      string     `gorm:"size:500" json:"coverImage"`
	ZhDescription   string     `gorm:"type:text" json:"zhDescription"`
	EnDescription   string     `gorm:"type:text" json:"enDescription"`
	HideFromList    bool       `gorm:"default:false" json:"hideFromList"`
	PreventCascade  bool       `gorm:"default:false" json:"preventCascade"`
	Metadata        JSONMap    `gorm:"type:jsonb" json:"metadata"`
	SortOrder       int        `gorm:"default:0" json:"sortOrder"`
}

// Validate validates the category model
func (c *Category) Validate() error {
	if c.Slug == "" {
		return errors.New("slug is required")
	}
	if c.ZhName == "" {
		return errors.New("zhName is required")
	}
	return nil
}
