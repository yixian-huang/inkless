package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/yixian-huang/inkless/backend/internal/provider"
)

// Embedding relevance thresholds (calibrate with golden samples).
const (
	// Body truncate for embedding cost control.
	ArticleMetaEmbedBodyMaxRunes = 2000
	// Below this → semantic off-topic (warn).
	ArticleMetaEmbedWarnThreshold = 0.55
	// At or above this → treat as relevant even if keyword overlap is 0 (synonym rewrite).
	ArticleMetaEmbedPassThreshold = 0.70
	// Keyword hit but embedding very low → still warn (accidental token collision).
	ArticleMetaEmbedCollideThreshold = 0.45
)

// batchEmbedder is optional; OpenAI-compatible providers may implement multi-input embeds.
type batchEmbedder interface {
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
}

// primaryMetaFieldValues returns ordered (field, text) for embedding relevance checks.
func primaryMetaFieldValues(sug ArticleMetaSuggested) [][2]string {
	pairs := [][2]string{
		{"zhTitle", sug.ZhTitle},
		{"enTitle", sug.EnTitle},
		{"zhSeoTitle", sug.ZhSeoTitle},
		{"enSeoTitle", sug.EnSeoTitle},
		{"zhMetaDescription", sug.ZhMetaDescription},
		{"enMetaDescription", sug.EnMetaDescription},
	}
	out := make([][2]string, 0, len(pairs))
	for _, p := range pairs {
		if strings.TrimSpace(p[1]) != "" {
			out = append(out, [2]string{p[0], strings.TrimSpace(p[1])})
		}
	}
	return out
}

// EvaluateArticleMetaEmbeddingRelevance embeds body + primary fields and returns
// refined relevance warnings. On any embed failure, returns nil (caller keeps keyword-only).
//
// Combined policy with keyword low_relevance:
//   - keyword=0 && cos >= pass  → drop keyword warn (synonym OK)
//   - keyword=0 && cos < warn   → low_relevance_embedding warn
//   - keyword=0 && mid band     → info weak relevance
//   - keyword>0 && cos < collide → low_relevance_embedding warn
//   - keyword>0 && cos >= collide → drop keyword warn for that field if present
func EvaluateArticleMetaEmbeddingRelevance(
	ctx context.Context,
	ai provider.AIProvider,
	bodyPlain string,
	sug ArticleMetaSuggested,
	existing []ArticleMetaWarning,
) []ArticleMetaWarning {
	if ai == nil {
		return existing
	}
	body := strings.TrimSpace(bodyPlain)
	if body == "" {
		return existing
	}
	fields := primaryMetaFieldValues(sug)
	if len(fields) == 0 {
		return existing
	}

	// Collect texts: body first, then fields
	texts := make([]string, 0, 1+len(fields))
	texts = append(texts, truncateRunes(body, ArticleMetaEmbedBodyMaxRunes))
	for _, f := range fields {
		texts = append(texts, f[1])
	}

	vectors, err := embedTexts(ctx, ai, texts)
	if err != nil || len(vectors) != len(texts) {
		// Degrade silently — keyword heuristics remain.
		return existing
	}
	bodyVec := vectors[0]
	if len(bodyVec) == 0 {
		return existing
	}

	// Map field → cosine
	scores := make(map[string]float64, len(fields))
	for i, f := range fields {
		scores[f[0]] = cosineSimilarity(bodyVec, vectors[i+1])
	}

	// Strip keyword-only low_relevance for primary fields; we re-emit combined decisions.
	filtered := make([]ArticleMetaWarning, 0, len(existing))
	kwLow := map[string]bool{}
	for _, w := range existing {
		if w.Code == "low_relevance" && isPrimaryMetaField(w.Field) {
			kwLow[w.Field] = true
			continue
		}
		// Drop previous embedding warns if re-running
		if w.Code == "low_relevance_embedding" || w.Code == "low_relevance_weak" {
			continue
		}
		filtered = append(filtered, w)
	}

	var embWarns []ArticleMetaWarning
	for _, f := range fields {
		field, _ := f[0], f[1]
		cos := scores[field]
		hasKWLow := kwLow[field]

		switch {
		case hasKWLow && cos >= ArticleMetaEmbedPassThreshold:
			// Synonym rewrite: no relevance warn
			continue
		case hasKWLow && cos < ArticleMetaEmbedWarnThreshold:
			embWarns = append(embWarns, ArticleMetaWarning{
				Code:     "low_relevance_embedding",
				Field:    field,
				Message:  fmt.Sprintf("%s 与正文语义相似度偏低（%.2f），且关键词无重叠", fieldLabel(field), cos),
				Severity: "warn",
			})
		case hasKWLow && cos >= ArticleMetaEmbedWarnThreshold && cos < ArticleMetaEmbedPassThreshold:
			embWarns = append(embWarns, ArticleMetaWarning{
				Code:     "low_relevance_weak",
				Field:    field,
				Message:  fmt.Sprintf("%s 关键词无重叠，语义相似度中等（%.2f），建议核对", fieldLabel(field), cos),
				Severity: "info",
			})
		case !hasKWLow && cos < ArticleMetaEmbedCollideThreshold:
			// Token overlap by chance but embedding says off-topic
			embWarns = append(embWarns, ArticleMetaWarning{
				Code:     "low_relevance_embedding",
				Field:    field,
				Message:  fmt.Sprintf("%s 与正文语义相似度偏低（%.2f），请确认是否跑题", fieldLabel(field), cos),
				Severity: "warn",
			})
		case !hasKWLow:
			// Keyword OK and embedding OK enough
			continue
		default:
			// hasKWLow handled above
			continue
		}
	}

	return dedupeWarnings(append(filtered, embWarns...))
}

func embedTexts(ctx context.Context, ai provider.AIProvider, texts []string) ([][]float64, error) {
	if be, ok := ai.(batchEmbedder); ok {
		return be.EmbedBatch(ctx, texts)
	}
	out := make([][]float64, len(texts))
	for i, t := range texts {
		v, err := ai.Embed(ctx, t)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	magA := math.Sqrt(normA)
	magB := math.Sqrt(normB)
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (magA * magB)
}
