package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectScriptLang(t *testing.T) {
	require.Equal(t, "zh", detectScriptLang("这是中文标题"))
	require.Equal(t, "en", detectScriptLang("This is an English title"))
	require.Equal(t, "unknown", detectScriptLang("12345 ---"))
}

func TestPlaceholderDetection(t *testing.T) {
	body := strings.Repeat("深度学习与推荐系统实践分享内容。", 5)
	sug := ArticleMetaSuggested{
		ZhTitle: "未命名",
		EnTitle: "Untitled",
	}
	warns := EvaluateArticleMetaQuality(body, sug, ArticleMetaCandidates{})
	codes := warningCodes(warns)
	require.Contains(t, codes, "placeholder")
}

func TestLanguageMismatch(t *testing.T) {
	body := strings.Repeat("Kubernetes cluster operations guide for production. ", 5)
	sug := ArticleMetaSuggested{
		ZhTitle: "Production Kubernetes Operations", // expected zh but Latin
		EnTitle: "生产环境运维手册",                       // expected en but CJK
	}
	warns := EvaluateArticleMetaQuality(body, sug, ArticleMetaCandidates{})
	var mismatch int
	for _, w := range warns {
		if w.Code == "language_mismatch" {
			mismatch++
		}
	}
	require.GreaterOrEqual(t, mismatch, 2)
}

func TestLowRelevance(t *testing.T) {
	body := strings.Repeat("Rust 异步运行时与 Tokio 调度模型深入解析。", 6)
	sug := ArticleMetaSuggested{
		ZhTitle:           "今日天气预报",
		ZhMetaDescription: strings.Repeat("完全无关的内容描述", 5),
	}
	warns := EvaluateArticleMetaQuality(body, sug, ArticleMetaCandidates{})
	codes := warningCodes(warns)
	require.Contains(t, codes, "low_relevance")
}

func TestMetaLengthShort(t *testing.T) {
	body := strings.Repeat("content about graphql apis and schema design. ", 6)
	sug := ArticleMetaSuggested{
		EnMetaDescription: "too short",
	}
	warns := EvaluateArticleMetaQuality(body, sug, ArticleMetaCandidates{})
	codes := warningCodes(warns)
	require.Contains(t, codes, "length_short")
}

func TestHighRelevanceNoLowWarn(t *testing.T) {
	body := "GraphQL schema design patterns for large APIs. " +
		strings.Repeat("GraphQL resolvers and type systems matter. ", 8)
	sug := ArticleMetaSuggested{
		EnTitle:           "GraphQL Schema Design Patterns",
		EnMetaDescription: "Learn GraphQL schema design patterns for large APIs and type systems in production resolvers.",
	}
	warns := EvaluateArticleMetaQuality(body, sug, ArticleMetaCandidates{})
	for _, w := range warns {
		require.NotEqual(t, "low_relevance", w.Code, "unexpected low_relevance on %s", w.Field)
	}
}

func TestSlugFormat(t *testing.T) {
	warns := slugQualityWarnings("Bad_Slug!")
	require.True(t, len(warns) > 0)
	require.Equal(t, "slug_format", warns[0].Code)
}

func TestTokenOverlapRatio(t *testing.T) {
	body := extractSignificantTokens("GraphQL schema design patterns for APIs")
	r := tokenOverlapRatio("GraphQL schema design", body)
	require.Greater(t, r, 0.0)
}

func warningCodes(ws []ArticleMetaWarning) []string {
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Code)
	}
	return out
}
