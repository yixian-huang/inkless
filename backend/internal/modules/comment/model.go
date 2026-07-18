package comment

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// CommentStatus represents the moderation status of a comment
type CommentStatus string

const (
	CommentStatusPending  CommentStatus = "pending"
	CommentStatusApproved CommentStatus = "approved"
	CommentStatusSpam     CommentStatus = "spam"
	CommentStatusTrash    CommentStatus = "trash"
)

// Comment represents a user comment on an article or page
type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Content string `gorm:"type:text;not null" json:"content"`

	AuthorName  string `gorm:"size:100;not null" json:"authorName"`
	AuthorEmail string `gorm:"size:255" json:"authorEmail"`
	AuthorURL   string `gorm:"size:500" json:"authorUrl"`
	AuthorIP    string `gorm:"size:45" json:"-"`

	ContentType string `gorm:"size:20;not null;index:idx_comment_target" json:"contentType"`
	ContentID   uint   `gorm:"not null;index:idx_comment_target" json:"contentId"`

	ParentID *uint      `gorm:"index" json:"parentId"`
	Children []*Comment `gorm:"foreignKey:ParentID" json:"children,omitempty"`

	Status     CommentStatus `gorm:"size:20;default:pending;index" json:"status"`
	AuthorRole string        `gorm:"size:20;default:guest" json:"authorRole"`
	Pinned     bool          `gorm:"default:false" json:"pinned"`
}

// Validate validates the comment model
func (c *Comment) Validate() error {
	if c.Content == "" {
		return errors.New("content is required")
	}
	if len(c.Content) > 10000 {
		return errors.New("content must be 10000 characters or fewer")
	}
	if c.AuthorName == "" {
		return errors.New("author name is required")
	}
	if c.ContentType != "article" && c.ContentType != "page" {
		return errors.New("content type must be 'article' or 'page'")
	}
	if c.ContentID == 0 {
		return errors.New("content id is required")
	}
	return nil
}
