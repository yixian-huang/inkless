package qa

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
)

// Handler handles Knowledge Base Q&A HTTP requests.
type Handler struct {
	qaService        *service.QAService
	embeddingService *service.EmbeddingService
	qaLogRepo        repository.QALogRepository
	contentDocRepo   repository.ContentDocumentRepository
	articleRepo      repository.ArticleRepository
}

// NewHandler creates a new Q&A handler.
func NewHandler(
	qaService *service.QAService,
	embeddingService *service.EmbeddingService,
	qaLogRepo repository.QALogRepository,
	contentDocRepo repository.ContentDocumentRepository,
	articleRepo repository.ArticleRepository,
) *Handler {
	return &Handler{
		qaService:        qaService,
		embeddingService: embeddingService,
		qaLogRepo:        qaLogRepo,
		contentDocRepo:   contentDocRepo,
		articleRepo:      articleRepo,
	}
}

// askInput is the JSON body for a Q&A question.
type askInput struct {
	Question string `json:"question"`
	Locale   string `json:"locale"`
}

// PublicAsk handles a public Q&A question.
// POST /public/qa/ask
func (h *Handler) PublicAsk(c *gin.Context) {
	var input askInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request data"}})
		return
	}

	if input.Question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "question is required"}})
		return
	}

	if input.Locale == "" {
		input.Locale = "zh"
	}

	result, err := h.qaService.Ask(c.Request.Context(), input.Question, input.Locale)
	if err != nil {
		if errors.Is(err, service.ErrAINotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AI_NOT_CONFIGURED", "message": "AI provider is not configured."}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to process question"}})
		return
	}

	// Log the Q&A interaction
	sourcesJSON, _ := json.Marshal(result.Sources)
	var sourcesArray model.JSONArray
	_ = json.Unmarshal(sourcesJSON, &sourcesArray)

	qaLog := &model.QALog{
		Question:  input.Question,
		Answer:    result.Answer,
		Sources:   sourcesArray,
		Locale:    input.Locale,
		IPAddress: c.ClientIP(),
	}
	// Best-effort logging; don't fail the response if logging fails
	_ = h.qaLogRepo.Create(c.Request.Context(), qaLog)

	c.JSON(http.StatusOK, gin.H{
		"answer":  result.Answer,
		"sources": result.Sources,
		"logId":   qaLog.ID,
	})
}

// AdminIndex triggers content indexing for the knowledge base.
// POST /admin/qa/index
func (h *Handler) AdminIndex(c *gin.Context) {
	ctx := c.Request.Context()
	totalIndexed := 0

	// Index published content documents
	docs, err := h.contentDocRepo.List(ctx)
	if err == nil {
		for _, doc := range docs {
			if doc.PublishedConfig == nil {
				continue
			}
			// Extract text content from the published config
			text := extractTextFromConfig(doc.PublishedConfig)
			if text == "" {
				continue
			}
			sourceID := "content:" + string(doc.PageKey)
			metadata := map[string]string{
				"type":     "content",
				"page_key": string(doc.PageKey),
			}
			count, err := h.embeddingService.IndexContent(ctx, sourceID, text, metadata)
			if err != nil {
				if errors.Is(err, service.ErrAINotConfigured) {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AI_NOT_CONFIGURED", "message": "AI provider is not configured."}})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "indexing failed: " + err.Error()}})
				return
			}
			totalIndexed += count
		}
	}

	// Index published articles
	articles, _, err := h.articleRepo.List(ctx, 0, 1000, "published", nil, nil)
	if err == nil {
		for _, article := range articles {
			text := article.ZhTitle + "\n" + article.ZhBody
			if article.EnTitle != "" {
				text += "\n" + article.EnTitle + "\n" + article.EnBody
			}
			sourceID := "article:" + strconv.FormatUint(uint64(article.ID), 10)
			metadata := map[string]string{
				"type":       "article",
				"article_id": strconv.FormatUint(uint64(article.ID), 10),
				"slug":       article.Slug,
			}
			count, err := h.embeddingService.IndexContent(ctx, sourceID, text, metadata)
			if err != nil {
				if errors.Is(err, service.ErrAINotConfigured) {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AI_NOT_CONFIGURED", "message": "AI provider is not configured."}})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "indexing failed: " + err.Error()}})
				return
			}
			totalIndexed += count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "indexing complete",
		"chunksStored": totalIndexed,
	})
}

// AdminListLogs returns paginated Q&A logs.
// GET /admin/qa/logs?page=1&pageSize=20
func (h *Handler) AdminListLogs(c *gin.Context) {
	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	items, total, err := h.qaLogRepo.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "query failed"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// feedbackInput is the JSON body for rating a Q&A answer.
type feedbackInput struct {
	Rating string `json:"rating"`
}

// AdminFeedback records feedback for a Q&A log entry.
// POST /admin/qa/logs/:id/feedback
func (h *Handler) AdminFeedback(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid ID"}})
		return
	}

	var input feedbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request data"}})
		return
	}

	rating := model.QAFeedback(input.Rating)
	if rating != model.QAFeedbackPositive && rating != model.QAFeedbackNegative {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "rating must be 'positive' or 'negative'"}})
		return
	}

	if err := h.qaLogRepo.UpdateRating(c.Request.Context(), uint(id), rating); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "qa log not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feedback recorded"})
}

// extractTextFromConfig extracts readable text from a content document's config.
// It recursively walks the JSON structure and collects string values.
func extractTextFromConfig(config model.JSONMap) string {
	var parts []string
	// Convert model.JSONMap to map[string]interface{} for the recursive function
	extractStrings(map[string]interface{}(config), &parts)
	return joinNonEmpty(parts, "\n")
}

// extractStrings recursively collects string values from a nested structure.
func extractStrings(data interface{}, parts *[]string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for _, val := range v {
			extractStrings(val, parts)
		}
	case []interface{}:
		for _, item := range v {
			extractStrings(item, parts)
		}
	case string:
		if v != "" {
			*parts = append(*parts, v)
		}
	}
}

// joinNonEmpty joins non-empty strings with the given separator.
func joinNonEmpty(parts []string, sep string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	result := nonEmpty[0]
	for i := 1; i < len(nonEmpty); i++ {
		result += sep + nonEmpty[i]
	}
	return result
}
