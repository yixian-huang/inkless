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

// updateInput is the JSON body for updating email settings.
type updateInput struct {
	Config model.JSONMap `json:"config"`
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
	var input updateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid request data"}})
		return
	}

	if input.Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "config is required"}})
		return
	}

	// Try to find existing config to preserve password if masked
	existing, err := h.siteConfigRepo.FindByKey(c.Request.Context(), model.SiteConfigKeyEmail)
	if err != nil {
		// Create new config
		sc := &model.SiteConfig{
			Key:              model.SiteConfigKeyEmail,
			DraftConfig:      input.Config,
			DraftVersion:     1,
			PublishedConfig:  input.Config,
			PublishedVersion: 1,
		}
		if createErr := h.siteConfigRepo.Upsert(c.Request.Context(), sc); createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to save email settings"}})
			return
		}

		resp := cloneJSONMap(input.Config)
		maskPassword(resp)
		c.JSON(http.StatusOK, gin.H{
			"config":  resp,
			"message": "Email settings created",
		})
		return
	}

	// Preserve password if the client sent the masked value
	preservePassword(input.Config, existing.PublishedConfig)

	// Immediate publish (like theme handler)
	existing.DraftConfig = input.Config
	existing.DraftVersion++
	existing.PublishedConfig = input.Config
	existing.PublishedVersion = existing.DraftVersion

	if err := h.siteConfigRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to save email settings"}})
		return
	}

	resp := cloneJSONMap(input.Config)
	maskPassword(resp)
	c.JSON(http.StatusOK, gin.H{
		"config":  resp,
		"message": "Email settings updated",
	})
}

// testInput is the JSON body for sending a test email.
type testInput struct {
	Email string `json:"email"`
}

// HandleTest sends a test email using current configuration.
// @Summary      Send test email
// @Description  Sends a test email to verify SMTP configuration
// @Tags         Email Settings (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Test email recipient"
// @Success      200 {object} object{message=string}
// @Failure      400 {object} object{error=object}
// @Router       /admin/email-settings/test [post]
func (h *Handler) HandleTest(c *gin.Context) {
	var input testInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "email is required"}})
		return
	}

	cfg := h.emailService.LoadConfig(c.Request.Context())
	if cfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "email not configured"}})
		return
	}

	if err := h.emailService.SendTest(c.Request.Context(), input.Email, cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test email sent successfully"})
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
