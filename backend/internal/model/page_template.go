package model

import (
	"errors"
	"time"
	"gorm.io/gorm"
)

const (
	TemplateCategoryBuiltin = "builtin"
	TemplateCategoryCustom  = "custom"
	TemplateCategoryTheme   = "theme"
)

type PageTemplate struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Key           string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	NameZh        string    `gorm:"size:200;not null" json:"nameZh"`
	NameEn        string    `gorm:"size:200;not null;default:''" json:"nameEn"`
	DescriptionZh string    `gorm:"type:text;not null;default:''" json:"descriptionZh"`
	DescriptionEn string    `gorm:"type:text;not null;default:''" json:"descriptionEn"`
	Category      string    `gorm:"size:50;not null;default:'custom'" json:"category"`
	Config        JSONMap   `gorm:"type:text;not null" json:"config"`
	Thumbnail     string    `gorm:"size:500" json:"thumbnail"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (PageTemplate) TableName() string { return "page_templates" }

func (pt *PageTemplate) Validate() error {
	if pt.Key == "" {
		return errors.New("key is required")
	}
	if pt.NameZh == "" {
		return errors.New("nameZh is required")
	}
	switch pt.Category {
	case TemplateCategoryBuiltin, TemplateCategoryCustom, TemplateCategoryTheme:
	default:
		return errors.New("category must be 'builtin', 'custom', or 'theme'")
	}
	return nil
}

func (pt *PageTemplate) BeforeSave(tx *gorm.DB) error {
	if pt.Category == "" {
		pt.Category = TemplateCategoryCustom
	}
	return pt.Validate()
}
