package service_test

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/service"
)

func setupSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	_, err = sqlDB.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS search_index_fts USING fts5(
		content_type, content_id UNINDEXED, locale, title, body, slug UNINDEXED, tokenize='unicode61'
	)`)
	if err != nil {
		t.Skipf("FTS5 not available in this SQLite build, skipping search tests: %v", err)
	}
	if err := db.AutoMigrate(&model.Article{}, &model.Page{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestSearchServiceIndexAndSearchCJK(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false) // false = SQLite mode

	ctx := context.Background()

	err := svc.IndexArticle(ctx, 1, "zh", "Go语言入门教程", "这是一篇关于Go语言的入门文章", "go-intro")
	if err != nil {
		t.Fatalf("index article: %v", err)
	}

	// CJK queries use LIKE-based fallback
	resp, err := svc.Search(ctx, "Go语言", "zh", "", 1, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least 1 result")
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected at least 1 result in Results slice")
	}
	if resp.Results[0].Title != "Go语言入门教程" {
		t.Errorf("expected matching title, got %q", resp.Results[0].Title)
	}
	if resp.Results[0].Type != "article" {
		t.Errorf("expected type 'article', got %q", resp.Results[0].Type)
	}
	if resp.Results[0].URL != "/blog/go-intro" {
		t.Errorf("expected URL '/blog/go-intro', got %q", resp.Results[0].URL)
	}
}

func TestSearchServiceIndexAndSearchEnglish(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	err := svc.IndexArticle(ctx, 1, "en", "Getting Started with Go", "A beginner guide to Go programming", "go-intro")
	if err != nil {
		t.Fatalf("index article: %v", err)
	}

	// English queries use FTS5 MATCH
	resp, err := svc.Search(ctx, "Go programming", "en", "", 1, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least 1 result")
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected results")
	}
	if resp.Results[0].Title != "Getting Started with Go" {
		t.Errorf("unexpected title: %q", resp.Results[0].Title)
	}
}

func TestSearchServiceRemoveFromIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	if err := svc.IndexArticle(ctx, 1, "en", "Test Article", "Some content here", "test"); err != nil {
		t.Fatalf("index: %v", err)
	}
	if err := svc.RemoveFromIndex(ctx, "article", 1); err != nil {
		t.Fatalf("remove: %v", err)
	}

	resp, err := svc.Search(ctx, "content", "en", "", 1, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 results after remove, got %d", resp.Total)
	}
}

func TestSearchServiceIndexPage(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	err := svc.IndexPage(ctx, 1, "en", "About Us", "We are a consulting firm", "about")
	if err != nil {
		t.Fatalf("index page: %v", err)
	}

	resp, err := svc.Search(ctx, "consulting", "en", "page", 1, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least 1 result for page search")
	}
	if len(resp.Results) > 0 && resp.Results[0].URL != "/about" {
		t.Errorf("expected URL '/about', got %q", resp.Results[0].URL)
	}
}

func TestSearchServiceSuggest(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	svc.IndexArticle(ctx, 1, "en", "Getting Started with Go", "body", "go-start")
	svc.IndexArticle(ctx, 2, "en", "Getting Better at Testing", "body", "go-test")

	titles, err := svc.Suggest(ctx, "Getting", "en", 5)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if len(titles) == 0 {
		t.Error("expected suggestions")
	}
}

func TestSearchServiceSuggestCJK(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	svc.IndexArticle(ctx, 1, "zh", "Go语言入门", "内容", "go-intro")
	svc.IndexArticle(ctx, 2, "zh", "Go语言进阶", "内容", "go-adv")

	titles, err := svc.Suggest(ctx, "Go语言", "zh", 5)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if len(titles) == 0 {
		t.Error("expected CJK suggestions")
	}
}

func TestSearchServiceRebuildIndex(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	// Insert a published article directly into the articles table
	article := model.Article{
		Slug:    "rebuild-test",
		Status:  model.ArticleStatusPublished,
		ZhTitle: "重建测试文章",
		ZhBody:  "测试内容",
		EnTitle: "Rebuild Test Article",
		EnBody:  "Test content for rebuild",
	}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	if err := svc.RebuildIndex(ctx); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	// CJK search (uses LIKE fallback)
	zhResp, err := svc.Search(ctx, "重建测试", "zh", "", 1, 10)
	if err != nil {
		t.Fatalf("zh search: %v", err)
	}
	if zhResp.Total == 0 {
		t.Error("expected zh result after rebuild")
	}

	// English search (uses FTS5)
	enResp, err := svc.Search(ctx, "Rebuild", "en", "", 1, 10)
	if err != nil {
		t.Fatalf("en search: %v", err)
	}
	if enResp.Total == 0 {
		t.Error("expected en result after rebuild")
	}
}

func TestSearchServiceContentTypeFilter(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	svc.IndexArticle(ctx, 1, "en", "Shared Term Document", "body content", "article-slug")
	svc.IndexPage(ctx, 1, "en", "Shared Term Page", "body content", "page-slug")

	// Search for articles only
	resp, err := svc.Search(ctx, "Shared", "en", "article", 1, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 article result, got %d", resp.Total)
	}
	if len(resp.Results) > 0 && resp.Results[0].Type != "article" {
		t.Errorf("expected type article, got %s", resp.Results[0].Type)
	}
}

func TestSearchServicePagination(t *testing.T) {
	db := setupSearchTestDB(t)
	svc := service.NewSearchService(db, false)
	ctx := context.Background()

	// Index several articles
	for i := uint(1); i <= 5; i++ {
		svc.IndexArticle(ctx, i, "en", fmt.Sprintf("Testing Article %d", i), "body about testing", fmt.Sprintf("test-%d", i))
	}

	// Page 1, size 2
	resp, err := svc.Search(ctx, "testing", "en", "", 1, 2)
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results on page 1, got %d", len(resp.Results))
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}

	// Page 3, size 2 — should get 1 result
	resp, err = svc.Search(ctx, "testing", "en", "", 3, 2)
	if err != nil {
		t.Fatalf("search page 3: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result on page 3, got %d", len(resp.Results))
	}
}
