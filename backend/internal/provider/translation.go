package provider

import "context"

// TranslateRequest represents a translation request
type TranslateRequest struct {
	Text       string            `json:"text"`
	SourceLang string            `json:"sourceLang"`
	TargetLang string            `json:"targetLang"`
	Glossary   map[string]string `json:"glossary,omitempty"`
}

// TranslateResponse represents a translation response
type TranslateResponse struct {
	OriginalText   string `json:"originalText"`
	TranslatedText string `json:"translatedText"`
	SourceLang     string `json:"sourceLang"`
	TargetLang     string `json:"targetLang"`
}

// TranslationProvider defines the interface for translation services
type TranslationProvider interface {
	// Translate translates a single text
	Translate(ctx context.Context, req TranslateRequest) (*TranslateResponse, error)

	// BatchTranslate translates multiple texts
	BatchTranslate(ctx context.Context, items []TranslateRequest) ([]TranslateResponse, error)

	// DetectLanguage detects the language of the given text
	DetectLanguage(ctx context.Context, text string) (string, error)
}

// TextGenerator is a minimal AI interface used by translation services.
// The AIProvider from ai.go satisfies this via an adapter.
type TextGenerator interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
}
