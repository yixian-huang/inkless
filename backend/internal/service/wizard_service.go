package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/repository"
)

// WizardService implements AI-driven site building: plan generation and scaffolding.
type WizardService struct {
	ai       provider.AIProvider
	pageRepo repository.PageRepository
}

// NewWizardService creates a new WizardService.
func NewWizardService(ai provider.AIProvider, pageRepo repository.PageRepository) *WizardService {
	return &WizardService{
		ai:       ai,
		pageRepo: pageRepo,
	}
}

// GenerateSitePlan uses the AI provider to generate a SitePlan from a Questionnaire.
func (s *WizardService) GenerateSitePlan(ctx context.Context, q model.Questionnaire) (*model.SitePlan, error) {
	if s.ai == nil {
		return nil, ErrAINotConfigured
	}
	if q.Industry == "" {
		return nil, fmt.Errorf("industry is required")
	}

	locale := q.Locale
	if locale == "" {
		locale = "zh"
	}

	langNote := "Respond in Chinese (zh). All title values should be in Chinese."
	if locale == "en" {
		langNote = "Respond in English. All title values should be in English."
	}

	systemPrompt := fmt.Sprintf(`You are a professional web design consultant. You help users design site structures for their businesses. %s

When given a questionnaire, you MUST return a valid JSON object matching this exact schema:
{
  "recommendedTheme": "<one of: default, modern-dark, warm-earth>",
  "rationale": "<short explanation of design choices>",
  "colorScheme": {
    "primary": "<hex color>",
    "secondary": "<hex color>",
    "background": "<hex color>",
    "text": "<hex color>",
    "rationale": "<why these colors suit the brand>"
  },
  "pages": [
    {
      "slug": "<url-slug>",
      "title": {"zh": "<chinese title>", "en": "<english title>"},
      "layout": "<layout type>",
      "sections": ["<section1>", "<section2>"],
      "sortOrder": <integer>
    }
  ],
  "suggestedContent": [
    {
      "pageSlug": "<slug>",
      "heading": "<main headline>",
      "subheading": "<subtitle>",
      "body": "<body paragraph>",
      "ctaText": "<call to action>"
    }
  ]
}

Available themes:
- default: clean, professional, suitable for most businesses
- modern-dark: dark-themed, bold, suited for tech/creative industries
- warm-earth: warm tones, approachable, suited for lifestyle/consulting/wellness

Available section types: hero, features, card-grid, testimonials, contact-form, team, portfolio, pricing, faq, cta, stats, timeline

Always include at minimum: home page and contact page. Add relevant pages based on the business type.
Return ONLY valid JSON — no markdown fences, no extra commentary.`, langNote)

	userMessage := buildQuestionnaireSummary(q)

	raw, err := s.ai.ChatComplete(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("AI plan generation failed: %w", err)
	}

	plan, err := parseSitePlan(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI site plan: %w", err)
	}

	// Ensure locale is propagated
	plan.RecommendedTheme = sanitizeThemeName(plan.RecommendedTheme)

	return plan, nil
}

// ScaffoldSite creates pages in the database based on the provided SitePlan.
// Pages that already exist (by slug) are skipped rather than overwritten.
func (s *WizardService) ScaffoldSite(ctx context.Context, plan model.SitePlan) (*model.ScaffoldResult, error) {
	result := &model.ScaffoldResult{
		AppliedTheme: plan.RecommendedTheme,
		CreatedPages: []string{},
		SkippedPages: []string{},
	}

	// Build content map for fast lookup
	contentMap := make(map[string]model.SuggestedContent)
	for _, c := range plan.SuggestedContent {
		contentMap[c.PageSlug] = c
	}

	for _, pp := range plan.Pages {
		if pp.Slug == "" {
			continue
		}

		// Check if page already exists
		existing, _ := s.pageRepo.FindBySlug(ctx, pp.Slug)
		if existing != nil {
			result.SkippedPages = append(result.SkippedPages, pp.Slug)
			continue
		}

		title := model.JSONMap{}
		for locale, t := range pp.Title {
			title[locale] = t
		}
		if len(title) == 0 {
			title["zh"] = pp.Slug
			title["en"] = pp.Slug
		}

		// Build page config from section list and suggested content
		pageConfig := buildPageConfig(pp, contentMap[pp.Slug], plan.ColorScheme)

		page := &model.Page{
			Slug:        pp.Slug,
			Title:       title,
			Template:    "default",
			Config:      pageConfig,
			Status:      model.PageStatusDraft,
			SortOrder:   pp.SortOrder,
			ThemeID:     plan.RecommendedTheme,
			RenderMode:  "dynamic",
			IsThemePage: false,
			Visibility:  "public",
		}

		if err := s.pageRepo.Create(ctx, page); err != nil {
			return nil, fmt.Errorf("failed to create page %q: %w", pp.Slug, err)
		}

		result.CreatedPages = append(result.CreatedPages, pp.Slug)
	}

	return result, nil
}

// SuggestColors returns a color scheme recommendation for an industry/brand.
func (s *WizardService) SuggestColors(ctx context.Context, req model.ColorSuggestionRequest) (*model.ColorScheme, error) {
	if s.ai == nil {
		return nil, ErrAINotConfigured
	}
	if req.Industry == "" {
		return nil, fmt.Errorf("industry is required")
	}

	locale := req.Locale
	if locale == "" {
		locale = "zh"
	}

	langNote := "Respond in Chinese."
	if locale == "en" {
		langNote = "Respond in English."
	}

	systemPrompt := fmt.Sprintf(`You are a professional brand color consultant. %s
Return ONLY a valid JSON object matching this schema (no markdown, no extra text):
{
  "primary": "<hex>",
  "secondary": "<hex>",
  "background": "<hex>",
  "text": "<hex>",
  "rationale": "<explanation>"
}`, langNote)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Industry: %s\n", req.Industry))
	if req.BrandName != "" {
		sb.WriteString(fmt.Sprintf("Brand name: %s\n", req.BrandName))
	}
	sb.WriteString("Please suggest a professional color palette for this brand.")

	raw, err := s.ai.ChatComplete(ctx, systemPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("AI color suggestion failed: %w", err)
	}

	var scheme model.ColorScheme
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &scheme); err != nil {
		// Fallback: try to extract JSON from the response
		extracted := extractJSON(raw)
		if err2 := json.Unmarshal([]byte(extracted), &scheme); err2 != nil {
			return nil, fmt.Errorf("failed to parse color scheme JSON: %w", err)
		}
	}

	return &scheme, nil
}

// GenerateContent returns sample content for a given page type and industry.
func (s *WizardService) GenerateContent(ctx context.Context, req model.GenerateContentRequest) (*model.SuggestedContent, error) {
	if s.ai == nil {
		return nil, ErrAINotConfigured
	}
	if req.PageType == "" {
		return nil, fmt.Errorf("pageType is required")
	}
	if req.Industry == "" {
		return nil, fmt.Errorf("industry is required")
	}

	locale := req.Locale
	if locale == "" {
		locale = "zh"
	}

	langNote := "Write all content in Chinese."
	if locale == "en" {
		langNote = "Write all content in English."
	}

	systemPrompt := fmt.Sprintf(`You are a professional copywriter for websites. %s
Return ONLY a valid JSON object matching this schema (no markdown, no extra text):
{
  "pageSlug": "<page-type-slug>",
  "heading": "<main headline>",
  "subheading": "<supporting subtitle>",
  "body": "<2-3 sentence body paragraph>",
  "ctaText": "<call to action button text>"
}`, langNote)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Page type: %s\n", req.PageType))
	sb.WriteString(fmt.Sprintf("Industry: %s\n", req.Industry))
	if req.BrandName != "" {
		sb.WriteString(fmt.Sprintf("Brand name: %s\n", req.BrandName))
	}
	if req.Description != "" {
		sb.WriteString(fmt.Sprintf("Business description: %s\n", req.Description))
	}
	sb.WriteString("Please generate professional website copy for this page.")

	raw, err := s.ai.ChatComplete(ctx, systemPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("AI content generation failed: %w", err)
	}

	var content model.SuggestedContent
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &content); err != nil {
		extracted := extractJSON(raw)
		if err2 := json.Unmarshal([]byte(extracted), &content); err2 != nil {
			return nil, fmt.Errorf("failed to parse content JSON: %w", err)
		}
	}

	if content.PageSlug == "" {
		content.PageSlug = req.PageType
	}

	return &content, nil
}

// buildQuestionnaireSummary converts the questionnaire to a human-readable prompt string.
func buildQuestionnaireSummary(q model.Questionnaire) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Industry: %s\n", q.Industry))

	if q.BrandName != "" {
		sb.WriteString(fmt.Sprintf("Brand name: %s\n", q.BrandName))
	}
	if q.StylePreference != "" {
		sb.WriteString(fmt.Sprintf("Style preference: %s\n", q.StylePreference))
	}
	if len(q.Features) > 0 {
		sb.WriteString(fmt.Sprintf("Desired features: %s\n", strings.Join(q.Features, ", ")))
	}
	if len(q.ContentTypes) > 0 {
		sb.WriteString(fmt.Sprintf("Content types: %s\n", strings.Join(q.ContentTypes, ", ")))
	}
	if q.Description != "" {
		sb.WriteString(fmt.Sprintf("Business description: %s\n", q.Description))
	}
	sb.WriteString(fmt.Sprintf("Locale: %s\n", q.Locale))
	sb.WriteString("\nPlease generate a complete site plan for this business.")

	return sb.String()
}

// parseSitePlan attempts to unmarshal the raw AI response into a SitePlan.
// It handles cases where the AI wraps JSON in markdown code fences.
func parseSitePlan(raw string) (*model.SitePlan, error) {
	trimmed := strings.TrimSpace(raw)

	var plan model.SitePlan
	if err := json.Unmarshal([]byte(trimmed), &plan); err == nil {
		return &plan, nil
	}

	// Try extracting JSON from the response
	extracted := extractJSON(trimmed)
	if err := json.Unmarshal([]byte(extracted), &plan); err != nil {
		return nil, fmt.Errorf("could not parse site plan from AI response: %w", err)
	}

	return &plan, nil
}

// extractJSON attempts to pull a JSON object out of a string that may contain
// surrounding text or markdown code fences.
func extractJSON(s string) string {
	// Strip markdown code fences
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	// Find first '{' and last '}'
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}

	return s
}

// sanitizeThemeName ensures the theme name is one of the known valid values.
func sanitizeThemeName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "modern-dark", "modern_dark":
		return "modern-dark"
	case "warm-earth", "warm_earth":
		return "warm-earth"
	default:
		return "default"
	}
}

// buildPageConfig assembles the JSON page config from section list and suggested content.
func buildPageConfig(pp model.PagePlan, content model.SuggestedContent, colors model.ColorScheme) model.JSONMap {
	sections := make([]interface{}, 0, len(pp.Sections))

	for i, sectionType := range pp.Sections {
		sec := map[string]interface{}{
			"id":   fmt.Sprintf("%s-%d", sectionType, i),
			"type": sectionType,
		}

		// Inject suggested content into the first hero/banner section
		if i == 0 && (sectionType == "hero" || sectionType == "banner") && content.Heading != "" {
			sec["heading"] = content.Heading
			sec["subheading"] = content.Subheading
			if content.CTAText != "" {
				sec["ctaText"] = content.CTAText
			}
		}

		sections = append(sections, sec)
	}

	cfg := model.JSONMap{
		"layout":   pp.Layout,
		"sections": sections,
	}

	if colors.Primary != "" {
		cfg["colorScheme"] = map[string]interface{}{
			"primary":    colors.Primary,
			"secondary":  colors.Secondary,
			"background": colors.Background,
			"text":       colors.Text,
		}
	}

	return cfg
}
