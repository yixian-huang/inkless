package model

import (
	"errors"
	"time"
	"gorm.io/gorm"
)

const (
	PageModeTemplate   = "template"
	PageModeComposable = "composable"
)

type UnifiedPage struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Slug string `gorm:"uniqueIndex;size:200;not null" json:"slug"`
	ZhTitle       string `gorm:"size:500;not null;default:''" json:"zhTitle"`
	EnTitle       string `gorm:"size:500;not null;default:''" json:"enTitle"`
	ZhDescription string `gorm:"type:text;not null;default:''" json:"zhDescription"`
	EnDescription string `gorm:"type:text;not null;default:''" json:"enDescription"`
	Mode       string `gorm:"size:20;not null;default:'composable'" json:"mode"`
	TemplateID *uint  `gorm:"index" json:"templateId"`
	DraftConfig      JSONMap         `gorm:"type:text" json:"draftConfig"`
	DraftVersion     int             `gorm:"not null;default:1" json:"draftVersion"`
	PublishedConfig  NullableJSONMap `gorm:"type:text" json:"publishedConfig"`
	PublishedVersion int             `gorm:"not null;default:0" json:"publishedVersion"`
	Status      string     `gorm:"size:20;not null;default:'draft'" json:"status"`
	ScheduledAt *time.Time `json:"scheduledAt"`
	TranslationStatus JSONMap `gorm:"type:text" json:"translationStatus"`
	ZhMetaTitle       string `gorm:"size:200;not null;default:''" json:"zhMetaTitle"`
	EnMetaTitle       string `gorm:"size:200;not null;default:''" json:"enMetaTitle"`
	ZhMetaDescription string `gorm:"size:500;not null;default:''" json:"zhMetaDescription"`
	EnMetaDescription string `gorm:"size:500;not null;default:''" json:"enMetaDescription"`
	ZhMetaKeywords    string `gorm:"size:500;not null;default:''" json:"zhMetaKeywords"`
	EnMetaKeywords    string `gorm:"size:500;not null;default:''" json:"enMetaKeywords"`
	SortOrder int   `gorm:"not null;default:0" json:"sortOrder"`
	ShowInNav bool  `gorm:"not null;default:false" json:"showInNav"`
	ParentID  *uint `gorm:"index" json:"parentId"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	PublishedAt *time.Time     `json:"publishedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UnifiedPage) TableName() string { return "unified_pages" }

func (p *UnifiedPage) Validate() error {
	if p.Slug == "" {
		return errors.New("slug is required")
	}
	if p.Mode != PageModeTemplate && p.Mode != PageModeComposable {
		return errors.New("mode must be 'template' or 'composable'")
	}
	if p.Mode == PageModeTemplate && p.TemplateID == nil {
		return errors.New("templateId is required for template mode")
	}
	if p.Status != "" && p.Status != "draft" && p.Status != "published" && p.Status != "scheduled" {
		return errors.New("invalid status")
	}
	return nil
}

func (p *UnifiedPage) BeforeSave(tx *gorm.DB) error {
	if p.Status == "" {
		p.Status = "draft"
	}
	if p.Mode == "" {
		p.Mode = PageModeComposable
	}
	return p.Validate()
}
