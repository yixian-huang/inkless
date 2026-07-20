package translation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/provider"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

type mockGlossaryRepo struct{}

func (m *mockGlossaryRepo) Create(context.Context, *model.Glossary) error { return nil }
func (m *mockGlossaryRepo) FindByID(context.Context, uint) (*model.Glossary, error) {
	return nil, errors.New("not found")
}
func (m *mockGlossaryRepo) Update(context.Context, *model.Glossary) error { return nil }
func (m *mockGlossaryRepo) Delete(context.Context, uint) error            { return nil }
func (m *mockGlossaryRepo) List(context.Context, int, int, string, string) ([]*model.Glossary, int64, error) {
	return nil, 0, nil
}
func (m *mockGlossaryRepo) FindByLangs(context.Context, string, string) ([]*model.Glossary, error) {
	return nil, nil
}

type mockArticleRepo struct {
	article     *model.Article
	updateCount int
}

func (m *mockArticleRepo) Create(context.Context, *model.Article) error { return nil }
func (m *mockArticleRepo) FindByID(context.Context, uint) (*model.Article, error) {
	if m.article == nil {
		return nil, errors.New("not found")
	}
	copy := *m.article
	return &copy, nil
}
func (m *mockArticleRepo) FindBySlug(context.Context, string) (*model.Article, error) {
	return nil, errors.New("not found")
}
func (m *mockArticleRepo) Update(_ context.Context, article *model.Article) error {
	m.updateCount++
	copy := *article
	m.article = &copy
	return nil
}
func (m *mockArticleRepo) UpdateScheduledPublication(context.Context, *model.Article, time.Time) error {
	return nil
}
func (m *mockArticleRepo) Delete(context.Context, uint) error { return nil }
func (m *mockArticleRepo) List(context.Context, int, int, string, *uint, *uint) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (m *mockArticleRepo) ListPublished(context.Context, int, int, string, string) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

func (m *mockArticleRepo) Count(context.Context, string) (int64, error) {
	return 0, nil
}

type mockTranslator struct{}

func (m mockTranslator) Translate(_ context.Context, req provider.TranslateRequest) (*provider.TranslateResponse, error) {
	return &provider.TranslateResponse{
		OriginalText:   req.Text,
		TranslatedText: "translated " + req.Text,
		SourceLang:     req.SourceLang,
		TargetLang:     req.TargetLang,
	}, nil
}

func (m mockTranslator) BatchTranslate(ctx context.Context, items []provider.TranslateRequest) ([]provider.TranslateResponse, error) {
	responses := make([]provider.TranslateResponse, 0, len(items))
	for _, item := range items {
		resp, err := m.Translate(ctx, item)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

func (m mockTranslator) DetectLanguage(context.Context, string) (string, error) { return "zh", nil }

type translationMockAI struct {
	response string
}

func (m *translationMockAI) ChatComplete(context.Context, string, string) (string, error) {
	return m.response, nil
}
func (m *translationMockAI) Chat(context.Context, provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{Content: m.response}, nil
}
func (m *translationMockAI) Complete(context.Context, provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{Text: m.response}, nil
}
func (m *translationMockAI) Summarize(_ context.Context, text string, _ int) (string, error) {
	return text, nil
}
func (m *translationMockAI) SuggestTitles(context.Context, string, int) ([]string, error) {
	return nil, nil
}
func (m *translationMockAI) SuggestTags(context.Context, string, []string) ([]string, error) {
	return nil, nil
}
func (m *translationMockAI) StreamChat(context.Context, provider.ChatRequest) (<-chan provider.ChatChunk, error) {
	ch := make(chan provider.ChatChunk)
	close(ch)
	return ch, nil
}
func (m *translationMockAI) Embed(context.Context, string) ([]float64, error) { return nil, nil }
func (m *translationMockAI) Name() string                                     { return "mock" }

func setupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/translate", handler.Translate)
	router.POST("/translate/article/:id", handler.TranslateArticle)
	return router
}

func postJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestTranslateResolvesAIProviderFromRegistryAtCallTime(t *testing.T) {
	registry := provider.NewRegistry()
	registry.SetAI(&translationMockAI{response: "first"})
	handler := NewHandlerWithRegistry(registry, &mockGlossaryRepo{}, &mockArticleRepo{})
	router := setupRouter(handler)

	registry.SetAI(&translationMockAI{response: "second"})

	w := postJSON(t, router, "/translate", gin.H{
		"text":       "hello",
		"sourceLang": "en",
		"targetLang": "zh",
	})

	require.Equal(t, http.StatusOK, w.Code)
	var resp provider.TranslateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "second", resp.TranslatedText)
}

func TestTranslateReturnsServiceUnavailableWhenAINotConfigured(t *testing.T) {
	handler := NewHandler(service.NewNoopTranslationProvider(), &mockGlossaryRepo{}, &mockArticleRepo{})
	router := setupRouter(handler)

	w := postJSON(t, router, "/translate", gin.H{
		"text":       "hello",
		"sourceLang": "en",
		"targetLang": "zh",
	})

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "AI_NOT_CONFIGURED")
}

func TestTranslateArticlePreviewDoesNotOverwriteTargetFields(t *testing.T) {
	articleRepo := &mockArticleRepo{article: &model.Article{
		ID:      1,
		Slug:    "demo",
		ZhTitle: "中文标题",
		EnTitle: "Existing English",
		ZhBody:  "中文正文",
		EnBody:  "Existing body",
	}}
	handler := NewHandler(mockTranslator{}, &mockGlossaryRepo{}, articleRepo)
	router := setupRouter(handler)

	w := postJSON(t, router, "/translate/article/1", gin.H{"mode": "preview"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, articleRepo.updateCount)
	assert.Equal(t, "Existing English", articleRepo.article.EnTitle)
	assert.Contains(t, w.Body.String(), `"applied":false`)
}

func TestTranslateArticleDefaultsToPreview(t *testing.T) {
	articleRepo := &mockArticleRepo{article: &model.Article{
		ID:      1,
		Slug:    "demo",
		ZhTitle: "中文标题",
		ZhBody:  "中文正文",
	}}
	handler := NewHandler(mockTranslator{}, &mockGlossaryRepo{}, articleRepo)
	router := setupRouter(handler)

	w := postJSON(t, router, "/translate/article/1", gin.H{})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, articleRepo.updateCount)
	assert.Contains(t, w.Body.String(), `"applied":false`)
}

func TestTranslateArticleRefusesApplyOverNonEmptyTargetFieldsWithoutOverwrite(t *testing.T) {
	articleRepo := &mockArticleRepo{article: &model.Article{
		ID:      1,
		Slug:    "demo",
		ZhTitle: "中文标题",
		EnTitle: "Existing English",
		ZhBody:  "中文正文",
	}}
	handler := NewHandler(mockTranslator{}, &mockGlossaryRepo{}, articleRepo)
	router := setupRouter(handler)

	w := postJSON(t, router, "/translate/article/1", gin.H{"mode": "apply"})

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, 0, articleRepo.updateCount)
	assert.Contains(t, w.Body.String(), "TRANSLATION_TARGET_NOT_EMPTY")
}

func TestTranslateArticleAppliesWhenOverwriteTrue(t *testing.T) {
	articleRepo := &mockArticleRepo{article: &model.Article{
		ID:      1,
		Slug:    "demo",
		ZhTitle: "中文标题",
		EnTitle: "Existing English",
		ZhBody:  "中文正文",
	}}
	handler := NewHandler(mockTranslator{}, &mockGlossaryRepo{}, articleRepo)
	router := setupRouter(handler)

	w := postJSON(t, router, "/translate/article/1", gin.H{
		"mode":      "apply",
		"overwrite": true,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, articleRepo.updateCount)
	assert.Equal(t, "translated 中文标题", articleRepo.article.EnTitle)
	assert.Contains(t, w.Body.String(), `"applied":true`)
}

func (m *mockArticleRepo) UpdateIfMatch(context.Context, *model.Article, time.Time) error { return nil }

func (m *mockArticleRepo) ListFilter(context.Context, repository.ArticleListFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
