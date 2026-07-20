package repository

import (
	"context"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
)

// PageViewStats holds aggregated view counts for a single page key
type PageViewStats struct {
	PageKey        string `json:"pageKey"`
	Today          int64  `json:"today"`
	Last7d         int64  `json:"last7d"`
	Last30d        int64  `json:"last30d"`
	UniqueVisitors int64  `json:"uniqueVisitors"`
}

// PageViewRepository defines the interface for page view data access
type PageViewRepository interface {
	// Create records a new page view
	Create(ctx context.Context, pv *model.PageView) error

	// GetSummary returns aggregated view stats grouped by page key
	GetSummary(ctx context.Context, now time.Time) ([]PageViewStats, error)

	// CountByPageKey returns total views for a page key (all time)
	CountByPageKey(ctx context.Context, pageKey string) (int64, error)
}
