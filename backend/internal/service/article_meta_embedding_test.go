package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/internal/provider"
)

// embMockAI returns deterministic topic vectors (no network).
type embMockAI struct {
	failEmbed bool
}

func (m *embMockAI) Name() string { return "emb-mock" }

func (m *embMockAI) Embed(_ context.Context, text string) ([]float64, error) {
	if m.failEmbed {
		return nil, fmt.Errorf("embed not supported")
	}
	return topicVector(text), nil
}

func (m *embMockAI) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	out := make([][]float64, len(texts))
	for i, t := range texts {
		v, err := m.Embed(ctx, t)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// topicVector: axis0=rust/async, axis1=weather, axis2=food — unit vectors.
func topicVector(text string) []float64 {
	t := strings.ToLower(text)
	v := []float64{0.05, 0.05, 0.05}
	if strings.Contains(t, "rust") || strings.Contains(t, "tokio") || strings.Contains(t, "异步") ||
		strings.Contains(t, "async") || strings.Contains(t, "future") || strings.Contains(t, "调度") {
		v[0] = 1
	}
	if strings.Contains(t, "天气") || strings.Contains(t, "weather") || strings.Contains(t, "预报") {
		v[1] = 1
	}
	if strings.Contains(t, "美食") || strings.Contains(t, "recipe") || strings.Contains(t, "food") ||
		strings.Contains(t, "gourmet") || strings.Contains(t, "cookbook") {
		v[2] = 1
	}
	var mag float64
	for _, x := range v {
		mag += x * x
	}
	if mag > 0 {
		inv := 1 / math.Sqrt(mag)
		for i := range v {
			v[i] *= inv
		}
	}
	return v
}

func (m *embMockAI) Chat(context.Context, provider.ChatRequest) (*provider.ChatResponse, error) {
	return nil, ErrAINotConfigured
}
func (m *embMockAI) Complete(context.Context, provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return nil, ErrAINotConfigured
}
func (m *embMockAI) Summarize(context.Context, string, int) (string, error) {
	return "", ErrAINotConfigured
}
func (m *embMockAI) SuggestTitles(context.Context, string, int) ([]string, error) {
	return nil, ErrAINotConfigured
}
func (m *embMockAI) SuggestTags(context.Context, string, []string) ([]string, error) {
	return nil, ErrAINotConfigured
}
func (m *embMockAI) StreamChat(context.Context, provider.ChatRequest) (<-chan provider.ChatChunk, error) {
	return nil, ErrAINotConfigured
}
func (m *embMockAI) ChatComplete(context.Context, string, string) (string, error) {
	return "", ErrAINotConfigured
}

func TestCosineSimilarity_Basic(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{1, 0, 0}
	require.InDelta(t, 1.0, cosineSimilarity(a, b), 1e-9)
	require.InDelta(t, 0.0, cosineSimilarity(a, []float64{0, 1, 0}), 1e-9)
}

func TestEmbeddingRelevance_SynonymPass(t *testing.T) {
	body := strings.Repeat("Rust 异步编程与 Tokio 运行时调度实践。", 5)
	// Keyword-low but same embedding topic (异步/调度)
	sug := ArticleMetaSuggested{
		ZhTitle: "异步任务调度与协作实践",
	}
	base := []ArticleMetaWarning{
		{Code: "low_relevance", Field: "zhTitle", Message: "kw", Severity: "warn"},
	}
	out := EvaluateArticleMetaEmbeddingRelevance(context.Background(), &embMockAI{}, body, sug, base)
	for _, w := range out {
		require.NotEqual(t, "low_relevance", w.Code)
		require.NotEqual(t, "low_relevance_embedding", w.Code, "same-topic synonym should pass")
	}
}

func TestEmbeddingRelevance_OffTopicWarn(t *testing.T) {
	body := strings.Repeat("Rust 异步编程与 Tokio 运行时调度实践。", 5)
	sug := ArticleMetaSuggested{ZhTitle: "今日天气预报"}
	base := []ArticleMetaWarning{
		{Code: "low_relevance", Field: "zhTitle", Message: "kw", Severity: "warn"},
	}
	out := EvaluateArticleMetaEmbeddingRelevance(context.Background(), &embMockAI{}, body, sug, base)
	var found bool
	for _, w := range out {
		if w.Code == "low_relevance_embedding" && w.Field == "zhTitle" {
			found = true
		}
		require.NotEqual(t, "low_relevance", w.Code)
	}
	require.True(t, found, "off-topic should emit embedding warn: %+v", out)
}

func TestEmbeddingRelevance_DegradeOnEmbedFailure(t *testing.T) {
	body := strings.Repeat("Rust 异步编程与 Tokio 运行时调度实践。", 5)
	sug := ArticleMetaSuggested{ZhTitle: "今日天气预报"}
	base := []ArticleMetaWarning{
		{Code: "low_relevance", Field: "zhTitle", Message: "kw", Severity: "warn"},
		{Code: "placeholder", Field: "enTitle", Message: "p", Severity: "warn"},
	}
	out := EvaluateArticleMetaEmbeddingRelevance(context.Background(), &embMockAI{failEmbed: true}, body, sug, base)
	require.Equal(t, base, out)
}

func TestEmbeddingRelevance_KeywordHitButLowEmbed(t *testing.T) {
	body := strings.Repeat("Rust async Tokio runtime scheduler futures. ", 8)
	sug := ArticleMetaSuggested{
		EnTitle: "gourmet food recipe cookbook dinner",
	}
	// No keyword-low in base — collision path when cos is low
	base := []ArticleMetaWarning{}
	out := EvaluateArticleMetaEmbeddingRelevance(context.Background(), &embMockAI{}, body, sug, base)
	var found bool
	for _, w := range out {
		if w.Code == "low_relevance_embedding" && w.Field == "enTitle" {
			found = true
		}
	}
	require.True(t, found, "low embed without kw-low should still warn: %+v", out)
}

func TestGenerateArticleMeta_EmbeddingRefinesRelevance(t *testing.T) {
	// Full generate path: chat returns off-topic title; mock embed flags it.
	payload := articleMetaLLMPayload{
		ZhTitles:          []string{"今日天气预报"},
		EnTitles:          []string{"Weather Forecast Today"},
		Slug:              "weather-forecast",
		ZhSeoTitle:        "今日天气预报",
		EnSeoTitle:        "Weather Forecast",
		ZhMetaDescription: "全国大部地区晴到多云，适宜出行游玩。",
		EnMetaDescription: "Sunny to cloudy across most regions, great for outdoor travel plans.",
	}
	// Need longer meta for length checks
	payload.ZhMetaDescription = strings.Repeat("全国大部地区晴到多云，适宜出行游玩。", 2)
	payload.EnMetaDescription = strings.Repeat("Sunny to cloudy across most regions for travel. ", 2)

	b, err := json.Marshal(payload)
	require.NoError(t, err)
	ai := &generateWithEmbedMock{chatJSON: string(b)}
	body := strings.Repeat("Rust 异步编程与 Tokio 运行时调度、Future 状态机与 pin 取消安全实践。", 4)
	resp, err := GenerateArticleMeta(context.Background(), ai, ArticleMetaRequest{
		SourceLang: "zh",
		ZhBody:     body,
		Mode:       "rewrite",
		Fields:     []string{"titles", "seo", "meta"},
	})
	require.NoError(t, err)
	codes := map[string]bool{}
	for _, w := range resp.Warnings {
		codes[w.Code] = true
	}
	require.True(t, codes["low_relevance_embedding"] || codes["low_relevance"],
		"expected relevance warning, got %+v", resp.Warnings)
}

// generateWithEmbedMock implements Chat + Embed for end-to-end GenerateArticleMeta.
type generateWithEmbedMock struct {
	chatJSON string
}

func (m *generateWithEmbedMock) Name() string { return "gen-emb-mock" }
func (m *generateWithEmbedMock) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{Content: m.chatJSON, Model: "mock"}, nil
}
func (m *generateWithEmbedMock) Embed(_ context.Context, text string) ([]float64, error) {
	return topicVector(text), nil
}
func (m *generateWithEmbedMock) Complete(context.Context, provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return nil, ErrAINotConfigured
}
func (m *generateWithEmbedMock) Summarize(context.Context, string, int) (string, error) {
	return "", ErrAINotConfigured
}
func (m *generateWithEmbedMock) SuggestTitles(context.Context, string, int) ([]string, error) {
	return nil, ErrAINotConfigured
}
func (m *generateWithEmbedMock) SuggestTags(context.Context, string, []string) ([]string, error) {
	return nil, ErrAINotConfigured
}
func (m *generateWithEmbedMock) StreamChat(context.Context, provider.ChatRequest) (<-chan provider.ChatChunk, error) {
	return nil, ErrAINotConfigured
}
func (m *generateWithEmbedMock) ChatComplete(context.Context, string, string) (string, error) {
	return "", ErrAINotConfigured
}
