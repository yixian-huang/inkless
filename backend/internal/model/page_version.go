package model

import (
	"errors"
	"time"
	"gorm.io/gorm"
)

type PageVersion struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PageID    uint      `gorm:"not null;index;uniqueIndex:idx_pv_page_version" json:"pageId"`
	Version   int       `gorm:"not null;uniqueIndex:idx_pv_page_version" json:"version"`
	Config    JSONMap   `gorm:"type:text;not null" json:"config"`
	CreatedBy uint      `gorm:"not null" json:"createdBy"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (PageVersion) TableName() string { return "page_versions" }

func (v *PageVersion) Validate() error {
	if v.PageID == 0 {
		return errors.New("pageId is required")
	}
	if v.Version < 1 {
		return errors.New("version must be >= 1")
	}
	return nil
}

func (v *PageVersion) BeforeSave(tx *gorm.DB) error {
	return v.Validate()
}
