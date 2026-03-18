package email_settings

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
)

// Handler handles email settings HTTP requests.
type Handler struct {
	siteConfigRepo repository.SiteConfigRepository
	emailService   *service.EmailService
}

// NewHandler creates a new email settings handler.
func NewHandler(siteConfigRepo repository.SiteConfigRepository, emailService *service.EmailService) *Handler {
	return &Handler{
		siteConfigRepo: siteConfigRepo,
		emailService:   emailService,
	}
}

// HandleGet returns the email configuration (with masked password).
// @Summary      Get email settings
// @Description  Returns the email configuration with SMTP password masked
// @Tags         Email Settings (Admin)
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} object
// @Router       /admin/email-settings [get]
func (h *Handler) HandleGet(c *gin.Context) {
	doc, err := h.siteConfigRepo.FindByKey(c.Request.Context(), model.SiteConfigKeyEmail)
	if err != nil {
		// Return empty config — not an error
		c.JSON(http.StatusOK, model.JSONMap{})
		return
	}

	config := doc.PublishedConfig
	if len(config) == 0 {
		config = model.JSONMap{}
	}

	maskPassword(config)

	c.JSON(http.StatusOK, config)
}

// HandleUpdate updates email settings (immediate publish like theme).
// @Summary      Update email settings
// @Description  Update email SMTP/template configuration (immediate publish)
// @Tags         Email Settings (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Email configuration"
// @Success      200 {object} object
// @Failure      400 {object} object{error=object}
// @Router       /admin/email-settings [put]
func (h *Handler) HandleUpdate(c *gin.Context) {
	var input model.JSONMap
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request body"}})
		return
	}

	ctx := c.Request.Context()

	// Try to find existing config to preserve password if masked
	existing, err := h.siteConfigRepo.FindByKey(ctx, model.SiteConfigKeyEmail)
	if err != nil {
		// Not found — create new
		existing = nil
	}

	if existing != nil {
		// Preserve password if the client sent the masked value
		preservePassword(input, existing.PublishedConfig)
	}

	if existing == nil {
		// First time — create new
		sc := &model.SiteConfig{
			Key:              model.SiteConfigKeyEmail,
			DraftConfig:      input,
			DraftVersion:     1,
			PublishedConfig:  input,
			PublishedVersion: 1,
		}
		if createErr := h.siteConfigRepo.Upsert(ctx, sc); createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to save email settings"}})
			return
		}
	} else {
		// Immediate publish (like theme handler)
		newVersion := existing.DraftVersion + 1
		existing.DraftConfig = input
		existing.DraftVersion = newVersion
		existing.PublishedConfig = input
		existing.PublishedVersion = newVersion
		if err := h.siteConfigRepo.Update(ctx, existing); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to save email settings"}})
			return
		}
	}

	maskPassword(input)
	c.JSON(http.StatusOK, input)
}

// HandleTest sends a test email using current configuration.
// @Summary      Send test email
// @Description  Sends a test email to verify SMTP configuration
// @Tags         Email Settings (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Test email recipient"
// @Success      200 {object} object{success=bool,message=string}
// @Failure      400 {object} object{error=object}
// @Router       /admin/email-settings/test [post]
func (h *Handler) HandleTest(c *gin.Context) {
	var input struct {
		To string `json:"to" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid email address"}})
		return
	}

	cfg := h.emailService.LoadConfig(c.Request.Context())
	if cfg == nil || !cfg.SMTP.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "SMTP is not configured"}})
		return
	}

	if err := h.emailService.SendTest(c.Request.Context(), input.To, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "发送失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试邮件已发送到 " + input.To,
	})
}

// ---------- Helpers ----------

// maskPassword replaces the SMTP password with "****" in the config map.
func maskPassword(config model.JSONMap) {
	smtpRaw, ok := config["smtp"]
	if !ok {
		return
	}
	smtpMap, ok := smtpRaw.(map[string]interface{})
	if !ok {
		return
	}
	if _, exists := smtpMap["password"]; exists {
		smtpMap["password"] = "****"
	}
}

// preservePassword restores the real password from existing config when client sends "****".
func preservePassword(incoming, existing model.JSONMap) {
	smtpRaw, ok := incoming["smtp"]
	if !ok {
		return
	}
	smtpMap, ok := smtpRaw.(map[string]interface{})
	if !ok {
		return
	}
	pwd, _ := smtpMap["password"].(string)
	if pwd != "****" {
		return
	}
	// Restore from existing
	existingSMTP, ok := existing["smtp"]
	if !ok {
		return
	}
	existingMap, ok := existingSMTP.(map[string]interface{})
	if !ok {
		return
	}
	if realPwd, exists := existingMap["password"]; exists {
		smtpMap["password"] = realPwd
	}
}

// cloneJSONMap creates a shallow copy of a JSONMap.
func cloneJSONMap(m model.JSONMap) model.JSONMap {
	out := make(model.JSONMap, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
