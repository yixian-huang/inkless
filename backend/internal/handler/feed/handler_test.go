package feed_test

import (
	"context"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handler/feed"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

type mockArticleRepo struct {
	items []*model.Article
}

func (m *mockArticleRepo) Create(context.Context, *model.Article) error { return nil }
func (m *mockArticleRepo) FindByID(context.Context, uint) (*model.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) FindBySlug(context.Context, string) (*model.Article, error) {
	return nil, nil
}
func (m *mockArticleRepo) Update(context.Context, *model.Article) error { return nil }
func (m *mockArticleRepo) UpdateScheduledPublication(context.Context, *model.Article, time.Time) error {
	return nil
}
func (m *mockArticleRepo) Delete(context.Context, uint) error { return nil }
func (m *mockArticleRepo) List(context.Context, int, int, string, *uint, *uint) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (m *mockArticleRepo) ListPublished(context.Context, int, int, string, string) ([]*model.Article, int64, error) {
	return m.items, int64(len(m.items)), nil
}

func (m *mockArticleRepo) Count(context.Context, string) (int64, error) {
	return 0, nil
}

type mockSiteCfgRepo struct {
	published model.JSONMap
}

func (m *mockSiteCfgRepo) FindByKey(ctx context.Context, key string) (*model.SiteConfig, error) {
	if key != model.SiteConfigKeyFeatures {
		return nil, nil
	}
	return &model.SiteConfig{
		ID:              1,
		PublishedConfig: m.published,
	}, nil
}
func (m *mockSiteCfgRepo) Upsert(context.Context, *model.SiteConfig) error { return nil }
func (m *mockSiteCfgRepo) Update(context.Context, *model.SiteConfig) error { return nil }
func (m *mockSiteCfgRepo) UpdateDraft(context.Context, string, int, model.JSONMap) (int, error) {
	return 0, nil
}
func (m *mockSiteCfgRepo) UpdatePublished(context.Context, string, model.JSONMap, int) error {
	return nil
}

func TestGetFeed_DisabledReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pub := time.Now()
	h := feed.NewHandler(
		&mockArticleRepo{items: []*model.Article{{Slug: "hi", ZhTitle: "Hi", Status: model.ArticleStatusPublished, PublishedAt: &pub}}},
		&mockSiteCfgRepo{published: model.JSONMap{"blog": map[string]interface{}{"rss": false}}},
		"https://example.com",
		"Blog",
		"Desc",
	)
	r := gin.New()
	r.GET("/feed.xml", h.GetFeed)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetFeed_ReturnsXML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pub := time.Now()
	h := feed.NewHandler(
		&mockArticleRepo{items: []*model.Article{{
			Slug: "hello", ZhTitle: "Hello", ZhBody: "<p>World</p>",
			Status: model.ArticleStatusPublished, PublishedAt: &pub, Visibility: "public",
		}}},
		&mockSiteCfgRepo{published: model.JSONMap{"blog": map[string]interface{}{"rss": true}}},
		"https://example.com",
		"My Blog",
		"Posts",
	)
	r := gin.New()
	r.GET("/feed.xml", h.GetFeed)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Hello") || !strings.Contains(body, "/blog/hello") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func (m *mockArticleRepo) UpdateIfMatch(context.Context, *model.Article, time.Time) error { return nil }

func (m *mockArticleRepo) ListFilter(context.Context, repository.ArticleListFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
