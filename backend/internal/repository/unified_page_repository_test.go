package repository_test

import (
	"context"
	"testing"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.UnifiedPage{}, &model.PageVersion{}, &model.PageTemplate{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUnifiedPageRepo_CreateAndFindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGormUnifiedPageRepository(db)
	ctx := context.Background()
	page := &model.UnifiedPage{Slug: "about", ZhTitle: "关于", Mode: "composable"}
	if err := repo.Create(ctx, page); err != nil {
		t.Fatalf("create: %v", err)
	}
	if page.ID == 0 {
		t.Error("expected ID to be set after create")
	}
	found, err := repo.FindBySlug(ctx, "about")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.ZhTitle != "关于" {
		t.Errorf("expected zhTitle '关于', got %q", found.ZhTitle)
	}
}

func TestUnifiedPageRepo_UpdateDraft_OptimisticLock(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGormUnifiedPageRepository(db)
	ctx := context.Background()
	page := &model.UnifiedPage{Slug: "test", Mode: "composable", DraftVersion: 1}
	repo.Create(ctx, page)
	newVer, err := repo.UpdateDraft(ctx, page.ID, 1, model.JSONMap{"sections": []any{}})
	if err != nil {
		t.Fatalf("updateDraft: %v", err)
	}
	if newVer != 2 {
		t.Errorf("expected version 2, got %d", newVer)
	}
	_, err = repo.UpdateDraft(ctx, page.ID, 1, model.JSONMap{"sections": []any{}})
	if err == nil {
		t.Error("expected conflict error for stale version")
	}
}

func TestUnifiedPageRepo_ListPublished(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGormUnifiedPageRepository(db)
	ctx := context.Background()
	repo.Create(ctx, &model.UnifiedPage{Slug: "pub", Mode: "composable", Status: "published"})
	repo.Create(ctx, &model.UnifiedPage{Slug: "draft", Mode: "composable", Status: "draft"})
	pages, err := repo.ListPublished(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("expected 1 published, got %d", len(pages))
	}
}
