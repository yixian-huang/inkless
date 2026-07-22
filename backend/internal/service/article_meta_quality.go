package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ArticleMetaWarning is a soft quality signal attached to a meta generation response.
// Severity: "warn" (review before apply) | "info".
type ArticleMetaWarning struct {
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // warn | info
}

const (
	// Meta description soft minimum (aligned with frontend SEO_DESC_MIN).
	ArticleMetaDescMin = 50
	// Soft max for display titles (not hard SEO limit).
	ArticleMetaDisplayTitleMax = 80
)

var placeholderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*(untitled|no\s*title|tbd|todo|placeholder|lorem\s+ipsum)\s*$`),
	regexp.MustCompile(`(?i)^\s*(未命名|无标题|请填写|待补充|占位|测试标题)\s*$`),
	regexp.MustCompile(`(?i)lorem\s+ipsum`),
	regexp.MustCompile(`(?i)^(this\s+is\s+a\s+(test|sample|placeholder))`),
	regexp.MustCompile(`^(这是一段?(测试|示例|占位))`),
}

// EvaluateArticleMetaQuality runs Phase 1.5 hard/soft checks on suggested fields vs source body.
// sourcePlain should already be plain text (HTML stripped).
func EvaluateArticleMetaQuality(sourcePlain string, sug ArticleMetaSuggested, candidates ArticleMetaCandidates) []ArticleMetaWarning {
	var out []ArticleMetaWarning
	body := strings.TrimSpace(sourcePlain)
	bodyTokens := extractSignificantTokens(body)

	checkField := func(field, value, expectLang string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		out = append(out, fieldQualityWarnings(field, value, expectLang, bodyTokens)...)
	}

	// Primary suggested fields
	checkField("zhTitle", sug.ZhTitle, "zh")
	checkField("enTitle", sug.EnTitle, "en")
	checkField("zhSeoTitle", sug.ZhSeoTitle, "zh")
	checkField("enSeoTitle", sug.EnSeoTitle, "en")
	checkField("zhMetaDescription", sug.ZhMetaDescription, "zh")
	checkField("enMetaDescription", sug.EnMetaDescription, "en")
	if s := strings.TrimSpace(sug.Slug); s != "" {
		out = append(out, slugQualityWarnings(s)...)
	}

	// Candidates (titles only): language + placeholder + soft length
	for i, t := range candidates.ZhTitles {
		field := fmt.Sprintf("zhTitles[%d]", i)
		out = append(out, fieldQualityWarnings(field, t, "zh", bodyTokens)...)
	}
	for i, t := range candidates.EnTitles {
		field := fmt.Sprintf("enTitles[%d]", i)
		out = append(out, fieldQualityWarnings(field, t, "en", bodyTokens)...)
	}

	return dedupeWarnings(out)
}

func fieldQualityWarnings(field, value, expectLang string, bodyTokens map[string]int) []ArticleMetaWarning {
	var out []ArticleMetaWarning
	value = strings.TrimSpace(value)
	if value == "" {
		return out
	}
	n := utf8.RuneCountInString(value)

	// Placeholders
	for _, re := range placeholderPatterns {
		if re.MatchString(value) {
			out = append(out, ArticleMetaWarning{
				Code:     "placeholder",
				Field:    field,
				Message:  fmt.Sprintf("%s 疑似占位/模板文案", fieldLabel(field)),
				Severity: "warn",
			})
			break
		}
	}

	// Length
	switch {
	case strings.Contains(field, "SeoTitle") || strings.HasSuffix(field, "SeoTitle"):
		if n > ArticleMetaSEOTitleMax {
			out = append(out, ArticleMetaWarning{
				Code:     "length_long",
				Field:    field,
				Message:  fmt.Sprintf("%s 偏长（%d/%d）", fieldLabel(field), n, ArticleMetaSEOTitleMax),
				Severity: "warn",
			})
		}
	case strings.Contains(field, "MetaDescription") || strings.Contains(field, "meta"):
		if n < ArticleMetaDescMin {
			out = append(out, ArticleMetaWarning{
				Code:     "length_short",
				Field:    field,
				Message:  fmt.Sprintf("%s 偏短（%d，建议 ≥%d）", fieldLabel(field), n, ArticleMetaDescMin),
				Severity: "warn",
			})
		}
		if n > ArticleMetaDescMax {
			out = append(out, ArticleMetaWarning{
				Code:     "length_long",
				Field:    field,
				Message:  fmt.Sprintf("%s 偏长（%d/%d）", fieldLabel(field), n, ArticleMetaDescMax),
				Severity: "warn",
			})
		}
	case strings.Contains(field, "Title") || strings.Contains(field, "title"):
		if n > ArticleMetaDisplayTitleMax {
			out = append(out, ArticleMetaWarning{
				Code:     "length_long",
				Field:    field,
				Message:  fmt.Sprintf("%s 偏长（%d，建议 ≤%d）", fieldLabel(field), n, ArticleMetaDisplayTitleMax),
				Severity: "info",
			})
		}
	}

	// Language
	if expectLang != "" {
		got := detectScriptLang(value)
		if got != "mixed" && got != "unknown" && got != expectLang {
			out = append(out, ArticleMetaWarning{
				Code:     "language_mismatch",
				Field:    field,
				Message:  fmt.Sprintf("%s 语言与预期不符（期望 %s，检测为 %s）", fieldLabel(field), expectLang, got),
				Severity: "warn",
			})
		}
	}

	// Keyword overlap with body (skip if body tokens empty)
	if len(bodyTokens) > 0 && !strings.HasPrefix(field, "zhTitles[") && !strings.HasPrefix(field, "enTitles[") {
		// Only primary suggested fields for overlap to reduce noise
		if isPrimaryMetaField(field) {
			overlap := tokenOverlapRatio(value, bodyTokens)
			if overlap == 0 {
				out = append(out, ArticleMetaWarning{
					Code:     "low_relevance",
					Field:    field,
					Message:  fmt.Sprintf("%s 与正文关键词重叠偏低，请确认是否跑题", fieldLabel(field)),
					Severity: "warn",
				})
			}
		}
	}

	return out
}

func slugQualityWarnings(slug string) []ArticleMetaWarning {
	var out []ArticleMetaWarning
	if !regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`).MatchString(slug) {
		out = append(out, ArticleMetaWarning{
			Code:     "slug_format",
			Field:    "slug",
			Message:  "Slug 格式不规范（应为小写英文 kebab-case）",
			Severity: "warn",
		})
	}
	if utf8.RuneCountInString(slug) > ArticleMetaSlugMax {
		out = append(out, ArticleMetaWarning{
			Code:     "length_long",
			Field:    "slug",
			Message:  fmt.Sprintf("Slug 偏长（%d/%d）", utf8.RuneCountInString(slug), ArticleMetaSlugMax),
			Severity: "warn",
		})
	}
	if utf8.RuneCountInString(slug) < 3 {
		out = append(out, ArticleMetaWarning{
			Code:     "length_short",
			Field:    "slug",
			Message:  "Slug 过短",
			Severity: "info",
		})
	}
	return out
}

func isPrimaryMetaField(field string) bool {
	switch field {
	case "zhTitle", "enTitle", "zhSeoTitle", "enSeoTitle", "zhMetaDescription", "enMetaDescription":
		return true
	default:
		return false
	}
}

func fieldLabel(field string) string {
	labels := map[string]string{
		"zhTitle":           "中文标题",
		"enTitle":           "英文标题",
		"slug":              "Slug",
		"zhSeoTitle":        "中文 SEO 标题",
		"enSeoTitle":        "英文 SEO 标题",
		"zhMetaDescription": "中文 Meta",
		"enMetaDescription": "英文 Meta",
	}
	if l, ok := labels[field]; ok {
		return l
	}
	return field
}

// detectScriptLang returns "zh", "en", "mixed", or "unknown".
func detectScriptLang(s string) string {
	var cjk, latin, other int
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Han, r):
			cjk++
		case unicode.IsLetter(r) && r < unicode.MaxASCII:
			latin++
		case unicode.IsLetter(r):
			other++
		}
	}
	letters := cjk + latin + other
	if letters == 0 {
		return "unknown"
	}
	// Ratio thresholds
	if float64(cjk)/float64(letters) >= 0.35 {
		if float64(latin)/float64(letters) >= 0.45 {
			return "mixed"
		}
		return "zh"
	}
	if float64(latin)/float64(letters) >= 0.6 {
		return "en"
	}
	if cjk > 0 && latin > 0 {
		return "mixed"
	}
	if cjk > latin {
		return "zh"
	}
	if latin > 0 {
		return "en"
	}
	return "unknown"
}

// extractSignificantTokens builds a frequency map of tokens from plain text.
// Latin: words length >= 3; CJK: overlapping bigrams from continuous Han runs.
func extractSignificantTokens(text string) map[string]int {
	freq := map[string]int{}
	if strings.TrimSpace(text) == "" {
		return freq
	}
	// Latin words
	reWord := regexp.MustCompile(`[A-Za-z][A-Za-z0-9]{2,}`)
	for _, w := range reWord.FindAllString(strings.ToLower(text), -1) {
		freq[w]++
	}
	// CJK bigrams
	var run []rune
	flush := func() {
		if len(run) == 1 {
			freq[string(run[0])]++
		}
		for i := 0; i+1 < len(run); i++ {
			freq[string(run[i:i+2])]++
		}
		run = run[:0]
	}
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			run = append(run, r)
		} else {
			flush()
		}
	}
	flush()

	// Drop very common function-ish tokens
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "this": true, "that": true,
		"from": true, "are": true, "was": true, "have": true, "has": true,
		"的": true, "了": true, "和": true, "是": true, "在": true, "我": true, "有": true,
		"也": true, "就": true, "不": true, "人": true, "都": true, "一": true, "一个": true,
		"我们": true, "可以": true, "没有": true, "什么": true, "这个": true, "那个": true,
	}
	for k := range stop {
		delete(freq, k)
	}
	return freq
}

// tokenOverlapRatio returns 0..1: share of field tokens present in body token set.
func tokenOverlapRatio(field string, bodyTokens map[string]int) float64 {
	ft := extractSignificantTokens(field)
	if len(ft) == 0 {
		return 0
	}
	hit := 0
	total := 0
	for tok := range ft {
		total++
		if bodyTokens[tok] > 0 {
			hit++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total)
}

func dedupeWarnings(in []ArticleMetaWarning) []ArticleMetaWarning {
	seen := map[string]bool{}
	out := make([]ArticleMetaWarning, 0, len(in))
	for _, w := range in {
		key := w.Code + "|" + w.Field + "|" + w.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, w)
	}
	return out
}
