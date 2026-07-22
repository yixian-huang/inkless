package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type goldenSample struct {
	ID           string `json:"id"`
	SourceLang   string `json:"sourceLang"`
	Body         string `json:"body"`
	Notes        string `json:"notes"`
	AcceptHints  struct {
		MustContainAny []string `json:"mustContainAny"`
		Forbid         []string `json:"forbid"`
	} `json:"acceptHints"`
}

// TestGoldenSampleFixturesLoad ensures fixtures stay valid JSON and non-empty.
// Full LLM scoring is manual (see docs/article-ai-meta-golden-samples.md).
func TestGoldenSampleFixturesLoad(t *testing.T) {
	dir := filepath.Join("testdata", "article_meta_golden")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err, e.Name())
		var s goldenSample
		require.NoError(t, json.Unmarshal(raw, &s), e.Name())
		require.NotEmpty(t, s.ID)
		require.NotEmpty(t, s.Body)
		require.GreaterOrEqual(t, len([]rune(s.Body)), ArticleMetaMinBodyRunes, s.ID)
	}
}

// TestGoldenSampleQualityOnSyntheticMeta validates quality heuristics against
// deliberately good / bad suggestions for fixture bodies (no LLM call).
func TestGoldenSampleQualityOnSyntheticMeta(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "article_meta_golden", "sample-rust-async.json"))
	require.NoError(t, err)
	var s goldenSample
	require.NoError(t, json.Unmarshal(raw, &s))

	good := ArticleMetaSuggested{
		ZhTitle:           "Rust 异步与 Tokio 调度实践",
		ZhMetaDescription: "从 Future、Tokio 运行时到 pin 与取消安全，梳理 Rust 异步高并发服务的常见坑与生产实践。",
		Slug:              "rust-async-tokio-practices",
	}
	goodWarns := EvaluateArticleMetaQuality(s.Body, good, ArticleMetaCandidates{})
	for _, w := range goodWarns {
		require.NotEqual(t, "low_relevance", w.Code, "good suggestion should overlap body")
		require.NotEqual(t, "placeholder", w.Code)
	}

	bad := ArticleMetaSuggested{
		ZhTitle:           "今日天气预报",
		ZhMetaDescription: "今天全国大部地区晴到多云，气温适宜出行。",
	}
	badWarns := EvaluateArticleMetaQuality(s.Body, bad, ArticleMetaCandidates{})
	codes := map[string]bool{}
	for _, w := range badWarns {
		codes[w.Code] = true
	}
	require.True(t, codes["low_relevance"], "off-topic suggestion should warn")
}
