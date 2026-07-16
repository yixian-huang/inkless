package repository

import (
	"context"
	"testing"
	"time"

	"blotting-consultancy/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGormAuditEventRepository_ListFiltersAndPaginates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditEvent{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	events := []*model.AuditEvent{
		{Action: "content.publish", Actor: "alice", Resource: "pages:1", Result: "success", CreatedAt: base},
		{Action: "content.publish", Actor: "alice", Resource: "pages:2", Result: "failure", CreatedAt: base.Add(time.Hour)},
		{Action: "content.rollback", Actor: "bob", Resource: "pages:1", Result: "success", CreatedAt: base.Add(2 * time.Hour)},
	}
	for _, event := range events {
		if err := db.Create(event).Error; err != nil {
			t.Fatalf("create event: %v", err)
		}
	}

	repo := NewGormAuditEventRepository(db)
	from := base.Add(-time.Minute)
	to := base.Add(90 * time.Minute)
	items, total, err := repo.List(
		context.Background(),
		1,
		1,
		"content.publish",
		"alice",
		&from,
		&to,
	)
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 1 || items[0].Resource != "pages:1" {
		t.Fatalf("unexpected page of items: %+v", items)
	}
}
