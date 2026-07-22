package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yixian-huang/inkless/backend/internal/provider"
)

// Article meta generation constraints (aligned with frontend publishChecklist).
const (
	ArticleMetaMinBodyRunes   = 80
	ArticleMetaMaxBodyRunes   = 8000
	ArticleMetaSEOTitleMax    = 60
	ArticleMetaDescMax        = 160
	ArticleMetaSlugMax        = 80
	ArticleMetaDefaultTitles  = 3
	ArticleMetaMaxTitleCount  = 5
	ArticleMetaExcerptMax     = 200
)

// ErrArticleMetaContentTooShort is returned when the source body is too short.
var ErrArticleMetaContentTooShort = errors.New("article content is too short for AI meta generation")

// ArticleMetaExisting holds current form values (for fill_empty skip logic).
type ArticleMetaExisting struct {
	Slug              string `json:"slug"`
	ZhSeoTitle        string `json:"zhSeoTitle"`
	EnSeoTitle        string `json:"enSeoTitle"`
	ZhMetaDescription string `json:"zhMetaDescription"`
	EnMetaDescription string `json:"enMetaDescription"`
	ZhTitle           string `json:"zhTitle"`
	EnTitle           string `json:"enTitle"`
}

// ArticleMetaRequest is the input for AI article metadata generation.
type ArticleMetaRequest struct {
	SourceLang   string              `json:"sourceLang"`
	ZhTitle      string              `json:"zhTitle"`
	EnTitle      string              `json:"enTitle"`
	ZhBody       string              `json:"zhBody"`
	EnBody       string              `json:"enBody"`
	Existing     ArticleMetaExisting `json:"existing"`
	Fields       []string            `json:"fields"`
	Mode         string              `json:"mode"` // fill_empty | rewrite
	TitleCount   int                 `json:"titleCount"`
	ExistingTags []string            `json:"existingTags"`
	SlugLocked   bool                `json:"slugLocked"` // published articles: never suggest slug
}

// ArticleMetaSuggested is the primary suggestion package.
type ArticleMetaSuggested struct {
	ZhTitle           string   `json:"zhTitle,omitempty"`
	EnTitle           string   `json:"enTitle,omitempty"`
	Slug              string   `json:"slug,omitempty"`
	ZhSeoTitle        string   `json:"zhSeoTitle,omitempty"`
	EnSeoTitle        string   `json:"enSeoTitle,omitempty"`
	ZhMetaDescription string   `json:"zhMetaDescription,omitempty"`
	EnMetaDescription string   `json:"enMetaDescription,omitempty"`
	ZhExcerpt         string   `json:"zhExcerpt,omitempty"`
	EnExcerpt         string   `json:"enExcerpt,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

// ArticleMetaCandidates holds alternate titles.
type ArticleMetaCandidates struct {
	ZhTitles []string `json:"zhTitles,omitempty"`
	EnTitles []string `json:"enTitles,omitempty"`
}

// ArticleMetaResponse is returned to the admin UI for preview/apply.
type ArticleMetaResponse struct {
	Candidates ArticleMetaCandidates `json:"candidates"`
	Suggested  ArticleMetaSuggested  `json:"suggested"`
	Skipped    []string              `json:"skipped"`
	// Warnings are Phase 1.5 soft quality signals (length/language/placeholder/relevance).
	Warnings []ArticleMetaWarning `json:"warnings"`
	Model    string               `json:"model"`
	Usage    struct {
		PromptTokens int `json:"prompt_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type articleMetaLLMPayload struct {
	ZhTitles          []string `json:"zhTitles"`
	EnTitles          []string `json:"enTitles"`
	Slug              string   `json:"slug"`
	ZhSeoTitle        string   `json:"zhSeoTitle"`
	EnSeoTitle        string   `json:"enSeoTitle"`
	ZhMetaDescription string   `json:"zhMetaDescription"`
	EnMetaDescription string   `json:"enMetaDescription"`
	ZhExcerpt         string   `json:"zhExcerpt"`
	EnExcerpt         string   `json:"enExcerpt"`
	Tags              []string `json:"tags"`
}

// GenerateArticleMeta uses the configured AI provider to propose article shell fields.
func GenerateArticleMeta(ctx context.Context, ai provider.AIProvider, req ArticleMetaRequest) (*ArticleMetaResponse, error) {
	if ai == nil {
		return nil, ErrAINotConfigured
	}
	if ai.Name() == "noop" {
		return nil, ErrAINotConfigured
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "fill_empty"
	}
	if mode != "fill_empty" && mode != "rewrite" {
		return nil, fmt.Errorf("unsupported mode: %s", req.Mode)
	}

	sourceLang := strings.ToLower(strings.TrimSpace(req.SourceLang))
	if sourceLang == "" {
		sourceLang = "zh"
	}
	if sourceLang != "zh" && sourceLang != "en" {
		return nil, fmt.Errorf("unsupported sourceLang: %s", req.SourceLang)
	}

	titleCount := req.TitleCount
	if titleCount <= 0 {
		titleCount = ArticleMetaDefaultTitles
	}
	if titleCount > ArticleMetaMaxTitleCount {
		titleCount = ArticleMetaMaxTitleCount
	}

	fields := normalizeMetaFields(req.Fields)
	zhPlain := plainTextFromHTML(req.ZhBody)
	enPlain := plainTextFromHTML(req.EnBody)
	sourcePlain := zhPlain
	if sourceLang == "en" {
		sourcePlain = enPlain
	}
	if utf8.RuneCountInString(strings.TrimSpace(sourcePlain)) < ArticleMetaMinBodyRunes {
		return nil, ErrArticleMetaContentTooShort
	}
	sourcePlain = truncateRunes(sourcePlain, ArticleMetaMaxBodyRunes)

	system := `You are a bilingual CMS metadata assistant for a Chinese/English blog.
Return ONLY a single JSON object (no markdown fences, no commentary).
Base every claim only on the provided article body; do not invent facts, metrics, or product names absent from the body.
Constraints:
- zhTitles: array of Chinese display titles (concise, natural)
- enTitles: array of English display titles
- slug: English kebab-case URL slug [a-z0-9-], max 80 chars, no leading/trailing hyphen
- zhSeoTitle / enSeoTitle: SEO titles, ideally under 60 characters
- zhMetaDescription / enMetaDescription: search snippets, ideally 50-160 characters
- zhExcerpt / enExcerpt: short list previews under 200 characters
- tags: short tag labels (prefer existing tags when relevant)
All string values must be plain text without surrounding quotes.`

	user := buildArticleMetaUserPrompt(req, sourceLang, sourcePlain, titleCount, fields)

	chatResp, err := ai.Chat(ctx, provider.ChatRequest{
		Messages: []provider.ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens:   1200,
		Temperature: 0.4,
	})
	if err != nil {
		return nil, err
	}

	payload, err := parseArticleMetaLLMJSON(chatResp.Content)
	if err != nil {
		// One retry with a stricter reminder via ChatComplete-style follow-up is expensive;
		// try re-parse after fence strip only (already in parse). Surface error.
		return nil, fmt.Errorf("failed to parse AI article meta: %w", err)
	}

	out := &ArticleMetaResponse{
		Model: chatResp.Model,
	}
	out.Usage.PromptTokens = chatResp.PromptTokens
	out.Usage.OutputTokens = chatResp.OutputTokens

	// Normalize candidates
	zhTitles := cleanStringList(payload.ZhTitles, titleCount)
	enTitles := cleanStringList(payload.EnTitles, titleCount)
	if len(zhTitles) == 0 && strings.TrimSpace(req.ZhTitle) != "" {
		zhTitles = []string{strings.TrimSpace(req.ZhTitle)}
	}
	if len(enTitles) == 0 && strings.TrimSpace(req.EnTitle) != "" {
		enTitles = []string{strings.TrimSpace(req.EnTitle)}
	}
	out.Candidates.ZhTitles = zhTitles
	out.Candidates.EnTitles = enTitles

	primaryZh := firstOrEmpty(zhTitles)
	primaryEn := firstOrEmpty(enTitles)

	want := fieldSet(fields)
	skip := make([]string, 0)
	fillEmpty := mode == "fill_empty"
	ex := req.Existing

	// fill_empty: skip when existing form value is non-empty
	maybeSkip := func(field, existing, primary string) string {
		if fillEmpty && strings.TrimSpace(existing) != "" {
			skip = append(skip, field)
			return ""
		}
		return strings.TrimSpace(primary)
	}

	sug := ArticleMetaSuggested{}
	if want["titles"] {
		if v := maybeSkip("zhTitle", firstNonEmpty(ex.ZhTitle, req.ZhTitle), primaryZh); v != "" {
			sug.ZhTitle = truncateRunes(v, 200)
		}
		if v := maybeSkip("enTitle", firstNonEmpty(ex.EnTitle, req.EnTitle), primaryEn); v != "" {
			sug.EnTitle = truncateRunes(v, 200)
		}
	}

	if want["slug"] {
		if req.SlugLocked {
			skip = append(skip, "slug")
		} else if fillEmpty && strings.TrimSpace(ex.Slug) != "" {
			skip = append(skip, "slug")
		} else {
			slug := normalizeSlug(payload.Slug)
			if slug == "" {
				slug = normalizeSlug(primaryEn)
			}
			if slug == "" {
				slug = normalizeSlug(primaryZh)
			}
			if slug != "" {
				sug.Slug = slug
			}
		}
	}

	// Pre-truncate length signals (Phase 1.5) — capture LLM overshoot before we clip.
	var preLenWarns []ArticleMetaWarning
	noteLong := func(field, raw string, max int) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		n := utf8.RuneCountInString(raw)
		if n > max {
			preLenWarns = append(preLenWarns, ArticleMetaWarning{
				Code:     "length_long",
				Field:    field,
				Message:  fmt.Sprintf("%s 原始结果偏长（%d/%d，已截断）", fieldLabel(field), n, max),
				Severity: "info",
			})
		}
	}

	if want["seo"] {
		if fillEmpty && strings.TrimSpace(ex.ZhSeoTitle) != "" {
			skip = append(skip, "zhSeoTitle")
		} else {
			v := strings.TrimSpace(payload.ZhSeoTitle)
			if v == "" {
				v = primaryZh
			}
			if v != "" {
				noteLong("zhSeoTitle", v, ArticleMetaSEOTitleMax)
				sug.ZhSeoTitle = truncateRunes(v, ArticleMetaSEOTitleMax)
			}
		}
		if fillEmpty && strings.TrimSpace(ex.EnSeoTitle) != "" {
			skip = append(skip, "enSeoTitle")
		} else {
			v := strings.TrimSpace(payload.EnSeoTitle)
			if v == "" {
				v = primaryEn
			}
			if v != "" {
				noteLong("enSeoTitle", v, ArticleMetaSEOTitleMax)
				sug.EnSeoTitle = truncateRunes(v, ArticleMetaSEOTitleMax)
			}
		}
	}

	if want["meta"] {
		if fillEmpty && strings.TrimSpace(ex.ZhMetaDescription) != "" {
			skip = append(skip, "zhMetaDescription")
		} else if v := strings.TrimSpace(payload.ZhMetaDescription); v != "" {
			noteLong("zhMetaDescription", v, ArticleMetaDescMax)
			sug.ZhMetaDescription = truncateRunes(v, ArticleMetaDescMax)
		}
		if fillEmpty && strings.TrimSpace(ex.EnMetaDescription) != "" {
			skip = append(skip, "enMetaDescription")
		} else if v := strings.TrimSpace(payload.EnMetaDescription); v != "" {
			noteLong("enMetaDescription", v, ArticleMetaDescMax)
			sug.EnMetaDescription = truncateRunes(v, ArticleMetaDescMax)
		}
	}

	if want["excerpts"] {
		if v := strings.TrimSpace(payload.ZhExcerpt); v != "" {
			sug.ZhExcerpt = truncateRunes(v, ArticleMetaExcerptMax)
		}
		if v := strings.TrimSpace(payload.EnExcerpt); v != "" {
			sug.EnExcerpt = truncateRunes(v, ArticleMetaExcerptMax)
		}
	}

	if want["tags"] {
		sug.Tags = cleanStringList(payload.Tags, 12)
	}

	out.Suggested = sug
	out.Skipped = uniqueStrings(skip)

	// Phase 1.5 quality gate (soft warnings only — never block response).
	qualityBody := sourcePlain
	// Prefer matching language body for keyword overlap when available.
	if sourceLang == "zh" && zhPlain != "" {
		qualityBody = zhPlain
	} else if sourceLang == "en" && enPlain != "" {
		qualityBody = enPlain
	}
	out.Warnings = append(preLenWarns, EvaluateArticleMetaQuality(qualityBody, sug, out.Candidates)...)
	out.Warnings = dedupeWarnings(out.Warnings)
	return out, nil
}

func buildArticleMetaUserPrompt(req ArticleMetaRequest, sourceLang, sourcePlain string, titleCount int, fields []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "sourceLang: %s\n", sourceLang)
	fmt.Fprintf(&b, "titleCount: %d\n", titleCount)
	fmt.Fprintf(&b, "fields: %s\n", strings.Join(fields, ", "))
	if t := strings.TrimSpace(req.ZhTitle); t != "" {
		fmt.Fprintf(&b, "existingZhTitle: %s\n", t)
	}
	if t := strings.TrimSpace(req.EnTitle); t != "" {
		fmt.Fprintf(&b, "existingEnTitle: %s\n", t)
	}
	if len(req.ExistingTags) > 0 {
		fmt.Fprintf(&b, "existingTags: %s\n", strings.Join(req.ExistingTags, ", "))
	}
	b.WriteString("\nProduce JSON with keys: zhTitles, enTitles, slug, zhSeoTitle, enSeoTitle, zhMetaDescription, enMetaDescription, zhExcerpt, enExcerpt, tags.\n")
	fmt.Fprintf(&b, "zhTitles and enTitles must each have up to %d items.\n\n", titleCount)
	b.WriteString("Article body:\n")
	b.WriteString(sourcePlain)
	return b.String()
}

func normalizeMetaFields(fields []string) []string {
	if len(fields) == 0 {
		return []string{"titles", "slug", "seo", "meta"}
	}
	allowed := map[string]bool{
		"titles": true, "slug": true, "seo": true, "meta": true, "tags": true, "excerpts": true,
	}
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.ToLower(strings.TrimSpace(f))
		if !allowed[f] || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	if len(out) == 0 {
		return []string{"titles", "slug", "seo", "meta"}
	}
	return out
}

func fieldSet(fields []string) map[string]bool {
	m := make(map[string]bool, len(fields))
	for _, f := range fields {
		m[f] = true
	}
	return m
}

func parseArticleMetaLLMJSON(text string) (*articleMetaLLMPayload, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("empty AI response")
	}
	// Strip markdown fences
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			// drop optional language tag line already handled by removing first
			text = strings.Join(lines, "\n")
		}
	}
	// Extract outermost JSON object if model added prose
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			text = text[i : j+1]
		}
	}

	var payload articleMetaLLMPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func plainTextFromHTML(s string) string {
	out := s
	// Remove script/style blocks roughly
	reBlock := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	out = reBlock.ReplaceAllString(out, " ")
	for {
		i := strings.Index(out, "<")
		if i < 0 {
			break
		}
		j := strings.Index(out[i:], ">")
		if j < 0 {
			break
		}
		out = out[:i] + " " + out[i+j+1:]
	}
	fields := strings.Fields(html.UnescapeString(out))
	return strings.Join(fields, " ")
}

func truncateRunes(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxRunes]))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeSlug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	// Transliteration of CJK is left to the model / client; drop non-latin.
	var b strings.Builder
	for _, r := range s {
		if r <= unicode.MaxASCII {
			b.WriteRune(r)
		} else if unicode.Is(unicode.Han, r) {
			// skip CJK; model should provide English slug
			continue
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteByte('-')
		}
	}
	s = nonSlug.ReplaceAllString(b.String(), "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if utf8.RuneCountInString(s) > ArticleMetaSlugMax {
		s = string([]rune(s)[:ArticleMetaSlugMax])
		s = strings.Trim(s, "-")
	}
	return s
}

func cleanStringList(in []string, max int) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

func firstOrEmpty(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return ss[0]
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
