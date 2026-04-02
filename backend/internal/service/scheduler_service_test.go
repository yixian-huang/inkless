package service_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/service"
)

func TestSchedulerPublishesOverdueArticles(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Article{}, &model.Page{})

	past := time.Now().Add(-1 * time.Hour)
	db.Create(&model.Article{
		Slug:        "test",
		ZhTitle:     "Test",
		Status:      "scheduled",
		ScheduledAt: &past,
	})

	sched := service.NewSchedulerService(db)
	count, err := sched.PublishOverdue(context.Background())
	if err != nil {
		t.Fatalf("publish overdue: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 published, got %d", count)
	}

	var article model.Article
	db.First(&article, "slug = ?", "test")
	if article.Status != "published" {
		t.Errorf("expected published, got %s", article.Status)
	}
}
