package translation

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/provider"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// Handler handles translation and glossary HTTP requests
type Handler struct {
	translator   provider.TranslationProvider
	registry     *provider.Registry
	glossaryRepo repository.GlossaryRepository
	articleRepo  repository.ArticleRepository
}

// NewHandler creates a new translation handler
func NewHandler(
	translator provider.TranslationProvider,
	glossaryRepo repository.GlossaryRepository,
	articleRepo repository.ArticleRepository,
) *Handler {
	return &Handler{
		translator:   translator,
		glossaryRepo: glossaryRepo,
		articleRepo:  articleRepo,
	}
}

// NewHandlerWithRegistry creates a translation handler that resolves the active
// AI provider from the registry for every translation request.
func NewHandlerWithRegistry(
	registry *provider.Registry,
	glossaryRepo repository.GlossaryRepository,
	articleRepo repository.ArticleRepository,
) *Handler {
	return &Handler{
		registry:     registry,
		glossaryRepo: glossaryRepo,
		articleRepo:  articleRepo,
	}
}

func (h *Handler) translationProvider() provider.TranslationProvider {
	if h.registry != nil {
		return service.NewAITranslationProviderWithRegistry(h.registry)
	}
	if h.translator != nil {
		return h.translator
	}
	return service.NewNoopTranslationProvider()
}

func (h *Handler) handleTranslationError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrAINotConfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "AI_NOT_CONFIGURED",
				"message": "AI provider is not configured. Please configure an AI provider in settings.",
			},
		})
		return
	}
	apierror.Message(c, http.StatusInternalServerError, err.Error())
}

// --- Translation endpoints ---

// translateInput is the JSON body for a translation request
type translateInput struct {
	Text       string `json:"text" binding:"required"`
	SourceLang string `json:"sourceLang" binding:"required"`
	TargetLang string `json:"targetLang" binding:"required"`
}

// Translate handles a single text translation
// POST /admin/translate
func (h *Handler) Translate(c *gin.Context) {
	var input translateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: text, sourceLang, and targetLang are required")
		return
	}

	// Load glossary for the language pair
	glossary, err := h.buildGlossaryMap(c, input.SourceLang, input.TargetLang)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "failed to load glossary")
		return
	}

	req := provider.TranslateRequest{
		Text:       input.Text,
		SourceLang: input.SourceLang,
		TargetLang: input.TargetLang,
		Glossary:   glossary,
	}

	resp, err := h.translationProvider().Translate(c.Request.Context(), req)
	if err != nil {
		h.handleTranslationError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// batchTranslateInput is the JSON body for a batch translation request
type batchTranslateInput struct {
	Items []translateInput `json:"items" binding:"required"`
}

// BatchTranslate handles batch text translations
// POST /admin/translate/batch
func (h *Handler) BatchTranslate(c *gin.Context) {
	var input batchTranslateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: items array is required")
		return
	}

	if len(input.Items) == 0 {
		apierror.Message(c, http.StatusBadRequest, "items array must not be empty")
		return
	}

	reqs := make([]provider.TranslateRequest, 0, len(input.Items))
	for _, item := range input.Items {
		glossary, err := h.buildGlossaryMap(c, item.SourceLang, item.TargetLang)
		if err != nil {
			apierror.Message(c, http.StatusInternalServerError, "failed to load glossary")
			return
		}
		reqs = append(reqs, provider.TranslateRequest{
			Text:       item.Text,
			SourceLang: item.SourceLang,
			TargetLang: item.TargetLang,
			Glossary:   glossary,
		})
	}

	responses, err := h.translationProvider().BatchTranslate(c.Request.Context(), reqs)
	if err != nil {
		h.handleTranslationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"translations": responses})
}

// TranslateArticle translates an article's content fields
// POST /admin/translate/article/:id
func (h *Handler) TranslateArticle(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	article, err := h.articleRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "article not found")
		return
	}

	var input translateArticleInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			apierror.Message(c, http.StatusBadRequest, "invalid request data")
			return
		}
	}

	// Determine source and target: if Chinese content exists, translate zh->en; otherwise en->zh
	sourceLang := "zh"
	targetLang := "en"
	if article.ZhTitle == "" && article.EnTitle != "" {
		sourceLang = "en"
		targetLang = "zh"
	}
	if input.SourceLang != "" {
		sourceLang = input.SourceLang
	}
	if input.TargetLang != "" {
		targetLang = input.TargetLang
	}

	apply := input.Mode == "apply"
	if input.Apply != nil {
		apply = *input.Apply
	}
	if input.Preview {
		apply = false
	}
	if sourceLang == targetLang {
		apierror.Message(c, http.StatusBadRequest, "sourceLang and targetLang must differ")
		return
	}

	// Load glossary
	glossary, err := h.buildGlossaryMap(c, sourceLang, targetLang)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "failed to load glossary")
		return
	}

	// Build translation requests for non-empty source fields
	type fieldMapping struct {
		sourceText string
		fieldName  string
		targetText string
	}

	var mappings []fieldMapping
	if sourceLang == "zh" {
		if article.ZhTitle != "" {
			mappings = append(mappings, fieldMapping{sourceText: article.ZhTitle, fieldName: "title", targetText: article.EnTitle})
		}
		if article.ZhBody != "" {
			mappings = append(mappings, fieldMapping{sourceText: article.ZhBody, fieldName: "body", targetText: article.EnBody})
		}
	} else {
		if article.EnTitle != "" {
			mappings = append(mappings, fieldMapping{sourceText: article.EnTitle, fieldName: "title", targetText: article.ZhTitle})
		}
		if article.EnBody != "" {
			mappings = append(mappings, fieldMapping{sourceText: article.EnBody, fieldName: "body", targetText: article.ZhBody})
		}
	}

	if len(mappings) == 0 {
		apierror.Message(c, http.StatusBadRequest, "no source content to translate")
		return
	}

	if apply && !input.Overwrite {
		var protectedFields []string
		for _, m := range mappings {
			if strings.TrimSpace(m.targetText) != "" {
				protectedFields = append(protectedFields, m.fieldName)
			}
		}
		if len(protectedFields) > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "TRANSLATION_TARGET_NOT_EMPTY",
					"message": "target fields are not empty; set overwrite=true to replace them",
					"fields":  protectedFields,
				},
			})
			return
		}
	}

	reqs := make([]provider.TranslateRequest, len(mappings))
	for i, m := range mappings {
		reqs[i] = provider.TranslateRequest{
			Text:       m.sourceText,
			SourceLang: sourceLang,
			TargetLang: targetLang,
			Glossary:   glossary,
		}
	}

	responses, err := h.translationProvider().BatchTranslate(c.Request.Context(), reqs)
	if err != nil {
		h.handleTranslationError(c, err)
		return
	}

	// Apply translations to the article
	translations := make(map[string]string)
	for i, m := range mappings {
		translations[m.fieldName] = responses[i].TranslatedText
	}

	if apply {
		if targetLang == "en" {
			if v, ok := translations["title"]; ok {
				article.EnTitle = v
			}
			if v, ok := translations["body"]; ok {
				article.EnBody = v
			}
		} else {
			if v, ok := translations["title"]; ok {
				article.ZhTitle = v
			}
			if v, ok := translations["body"]; ok {
				article.ZhBody = v
			}
		}

		if err := h.articleRepo.Update(c.Request.Context(), article); err != nil {
			apierror.Message(c, http.StatusInternalServerError, "failed to save translated article")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"article":      article,
		"applied":      apply,
		"mode":         map[bool]string{true: "apply", false: "preview"}[apply],
		"sourceLang":   sourceLang,
		"targetLang":   targetLang,
		"translations": translations,
	})
}

type translateArticleInput struct {
	SourceLang string `json:"sourceLang"`
	TargetLang string `json:"targetLang"`
	Preview    bool   `json:"preview"`
	Apply      *bool  `json:"apply"`
	Overwrite  bool   `json:"overwrite"`
	Mode       string `json:"mode"`
}

// --- Glossary endpoints ---

// GlossaryList returns a paginated list of glossary terms
// GET /admin/glossary?page=1&pageSize=20&sourceLang=zh&targetLang=en
func (h *Handler) GlossaryList(c *gin.Context) {
	page := 1
	pageSize := 50

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
	if pageSize > 200 {
		pageSize = 200
	}

	offset := (page - 1) * pageSize
	sourceLang := c.Query("sourceLang")
	targetLang := c.Query("targetLang")

	items, total, err := h.glossaryRepo.List(c.Request.Context(), offset, pageSize, sourceLang, targetLang)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "query failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// glossaryInput is the JSON body for creating/updating a glossary term
type glossaryInput struct {
	SourceLang string `json:"sourceLang" binding:"required"`
	TargetLang string `json:"targetLang" binding:"required"`
	SourceTerm string `json:"sourceTerm" binding:"required"`
	TargetTerm string `json:"targetTerm" binding:"required"`
	Context    string `json:"context"`
}

// GlossaryCreate creates a new glossary term
// POST /admin/glossary
func (h *Handler) GlossaryCreate(c *gin.Context) {
	var input glossaryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request: sourceLang, targetLang, sourceTerm, and targetTerm are required")
		return
	}

	glossary := &model.Glossary{
		SourceLang: input.SourceLang,
		TargetLang: input.TargetLang,
		SourceTerm: input.SourceTerm,
		TargetTerm: input.TargetTerm,
		Context:    input.Context,
	}

	if err := h.glossaryRepo.Create(c.Request.Context(), glossary); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, glossary)
}

// GlossaryUpdate updates a glossary term
// PUT /admin/glossary/:id
func (h *Handler) GlossaryUpdate(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	existing, err := h.glossaryRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "glossary term not found")
		return
	}

	var input glossaryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "invalid request data")
		return
	}

	existing.SourceLang = input.SourceLang
	existing.TargetLang = input.TargetLang
	existing.SourceTerm = input.SourceTerm
	existing.TargetTerm = input.TargetTerm
	existing.Context = input.Context

	if err := h.glossaryRepo.Update(c.Request.Context(), existing); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, existing)
}

// GlossaryDelete deletes a glossary term
// DELETE /admin/glossary/:id
func (h *Handler) GlossaryDelete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.glossaryRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "glossary term not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// buildGlossaryMap loads glossary terms for a language pair and returns them as a map
func (h *Handler) buildGlossaryMap(c *gin.Context, sourceLang, targetLang string) (map[string]string, error) {
	terms, err := h.glossaryRepo.FindByLangs(c.Request.Context(), sourceLang, targetLang)
	if err != nil {
		return nil, err
	}

	glossary := make(map[string]string, len(terms))
	for _, term := range terms {
		glossary[term.SourceTerm] = term.TargetTerm
	}
	return glossary, nil
}
