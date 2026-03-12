package service

import (
	"context"
	"fmt"
	"strings"

	"blotting-consultancy/internal/provider"
)

// AITranslationProvider implements TranslationProvider using an AIProvider.
// It constructs translation prompts with glossary terms and delegates to the AI backend.
type AITranslationProvider struct {
	ai provider.TextGenerator
}

// NewAITranslationProvider creates a new AITranslationProvider wrapping the given AIProvider
func NewAITranslationProvider(ai provider.TextGenerator) provider.TranslationProvider {
	return &AITranslationProvider{ai: ai}
}

// Translate translates a single text using the AI provider
func (a *AITranslationProvider) Translate(ctx context.Context, req provider.TranslateRequest) (*provider.TranslateResponse, error) {
	prompt := buildTranslationPrompt(req)

	result, err := a.ai.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI translation failed: %w", err)
	}

	return &provider.TranslateResponse{
		OriginalText:   req.Text,
		TranslatedText: strings.TrimSpace(result),
		SourceLang:     req.SourceLang,
		TargetLang:     req.TargetLang,
	}, nil
}

// BatchTranslate translates multiple texts sequentially using the AI provider
func (a *AITranslationProvider) BatchTranslate(ctx context.Context, items []provider.TranslateRequest) ([]provider.TranslateResponse, error) {
	responses := make([]provider.TranslateResponse, 0, len(items))
	for _, req := range items {
		resp, err := a.Translate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("batch translation failed on item: %w", err)
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

// DetectLanguage detects the language of the given text using the AI provider
func (a *AITranslationProvider) DetectLanguage(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(
		"Detect the language of the following text. "+
			"Reply with ONLY the ISO 639-1 language code (e.g., \"zh\", \"en\", \"ja\"). "+
			"Do not include any other text.\n\nText: %s",
		text,
	)

	result, err := a.ai.GenerateText(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("language detection failed: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// buildTranslationPrompt constructs a translation prompt with optional glossary terms
func buildTranslationPrompt(req provider.TranslateRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"Translate the following text from %s to %s.\n",
		langName(req.SourceLang), langName(req.TargetLang),
	))

	if len(req.Glossary) > 0 {
		sb.WriteString("\nUse the following glossary for consistent terminology:\n")
		for source, target := range req.Glossary {
			sb.WriteString(fmt.Sprintf("- \"%s\" → \"%s\"\n", source, target))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(
		"Output ONLY the translated text, without any explanations, " +
			"quotes, or additional formatting.\n\n",
	)
	sb.WriteString(req.Text)

	return sb.String()
}

// langName returns a human-readable language name for common codes
func langName(code string) string {
	switch strings.ToLower(code) {
	case "zh":
		return "Chinese"
	case "en":
		return "English"
	case "ja":
		return "Japanese"
	case "ko":
		return "Korean"
	case "fr":
		return "French"
	case "de":
		return "German"
	case "es":
		return "Spanish"
	default:
		return code
	}
}
