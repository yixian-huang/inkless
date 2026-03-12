package ai

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/service"
)

// Handler handles AI-related HTTP requests.
type Handler struct {
	registry *provider.Registry
}

// NewHandler creates a new AI handler.
func NewHandler(registry *provider.Registry) *Handler {
	return &Handler{registry: registry}
}

// --- Request/Response types ---

type chatInput struct {
	Messages    []provider.ChatMessage `json:"messages" binding:"required"`
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
}

type summarizeInput struct {
	Text      string `json:"text" binding:"required"`
	MaxLength int    `json:"max_length"`
}

type suggestTitlesInput struct {
	Content string `json:"content" binding:"required"`
	Count   int    `json:"count"`
}

type suggestTagsInput struct {
	Content      string   `json:"content" binding:"required"`
	ExistingTags []string `json:"existing_tags"`
}

type completeInput struct {
	Prompt      string  `json:"prompt" binding:"required"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type configResponse struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// getAI returns the AI provider or sends a 503 error if not configured.
func (h *Handler) getAI(c *gin.Context) provider.AIProvider {
	ai := h.registry.AI()
	if ai == nil {
		h.handleAIError(c, service.ErrAINotConfigured)
		return nil
	}
	return ai
}

// Chat handles POST /admin/ai/chat
func (h *Handler) Chat(c *gin.Context) {
	var input chatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request: " + err.Error()}})
		return
	}

	ai := h.getAI(c)
	if ai == nil {
		return
	}
	resp, err := ai.Chat(c.Request.Context(), provider.ChatRequest{
		Messages:    input.Messages,
		Model:       input.Model,
		MaxTokens:   input.MaxTokens,
		Temperature: input.Temperature,
	})
	if err != nil {
		h.handleAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Summarize handles POST /admin/ai/summarize
func (h *Handler) Summarize(c *gin.Context) {
	var input summarizeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request: " + err.Error()}})
		return
	}

	maxLength := input.MaxLength
	if maxLength <= 0 {
		maxLength = 200
	}

	ai := h.getAI(c)
	if ai == nil {
		return
	}
	summary, err := ai.Summarize(c.Request.Context(), input.Text, maxLength)
	if err != nil {
		h.handleAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// SuggestTitles handles POST /admin/ai/suggest-titles
func (h *Handler) SuggestTitles(c *gin.Context) {
	var input suggestTitlesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request: " + err.Error()}})
		return
	}

	count := input.Count
	if count <= 0 {
		count = 5
	}

	ai := h.getAI(c)
	if ai == nil {
		return
	}
	titles, err := ai.SuggestTitles(c.Request.Context(), input.Content, count)
	if err != nil {
		h.handleAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"titles": titles})
}

// SuggestTags handles POST /admin/ai/suggest-tags
func (h *Handler) SuggestTags(c *gin.Context) {
	var input suggestTagsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request: " + err.Error()}})
		return
	}

	ai := h.getAI(c)
	if ai == nil {
		return
	}
	tags, err := ai.SuggestTags(c.Request.Context(), input.Content, input.ExistingTags)
	if err != nil {
		h.handleAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// Complete handles POST /admin/ai/complete
func (h *Handler) Complete(c *gin.Context) {
	var input completeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request: " + err.Error()}})
		return
	}

	ai := h.getAI(c)
	if ai == nil {
		return
	}
	resp, err := ai.Complete(c.Request.Context(), provider.CompletionRequest{
		Prompt:      input.Prompt,
		Model:       input.Model,
		MaxTokens:   input.MaxTokens,
		Temperature: input.Temperature,
	})
	if err != nil {
		h.handleAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetConfig handles GET /admin/ai/config
func (h *Handler) GetConfig(c *gin.Context) {
	ai := h.registry.AI()
	if ai == nil {
		c.JSON(http.StatusOK, configResponse{
			Provider: "noop",
			Enabled:  false,
		})
		return
	}
	name := ai.Name()
	c.JSON(http.StatusOK, configResponse{
		Provider: name,
		Enabled:  name != "noop",
	})
}

// UpdateConfig handles PUT /admin/ai/config
// This is a placeholder; in production, you would persist provider settings.
func (h *Handler) UpdateConfig(c *gin.Context) {
	var input struct {
		Provider string `json:"provider" binding:"required"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request: " + err.Error()}})
		return
	}

	var newProvider provider.AIProvider
	switch input.Provider {
	case "openai":
		newProvider = service.NewOpenAIProvider(service.OpenAIConfig{
			APIKey:  input.APIKey,
			BaseURL: input.BaseURL,
			Model:   input.Model,
		})
	case "anthropic":
		newProvider = service.NewAnthropicProvider(service.AnthropicConfig{
			APIKey:  input.APIKey,
			BaseURL: input.BaseURL,
			Model:   input.Model,
		})
	case "noop", "":
		newProvider = service.NewNoopAIProvider()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "unsupported provider: " + input.Provider}})
		return
	}

	h.registry.SetAI(newProvider)

	c.JSON(http.StatusOK, configResponse{
		Provider: newProvider.Name(),
		Enabled:  newProvider.Name() != "noop",
	})
}

// handleAIError returns an appropriate HTTP error for AI provider errors.
func (h *Handler) handleAIError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrAINotConfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "AI_NOT_CONFIGURED",
				"message": "AI provider is not configured. Please configure an AI provider in settings.",
			},
		})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{
		"error": gin.H{
			"code":    "AI_ERROR",
			"message": err.Error(),
		},
	})
}
