package agent

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// Handler serves agent-oriented discovery endpoints (multi-site fleet helpers).
type Handler struct {
	baseURL string
	version string
	slots   *contentslots.Resolver
}

// NewHandler creates an agent handler. baseURL should be the instance BASE_URL (canonical).
func NewHandler(baseURL, version string) *Handler {
	return &Handler{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		version: version,
	}
}

// WithSlots attaches theme contentSlots resolver for whoami discovery fields.
func (h *Handler) WithSlots(r *contentslots.Resolver) *Handler {
	if h != nil {
		h.slots = r
	}
	return h
}

// WhoamiResponse is the payload for GET /admin/agent/whoami.
type WhoamiResponse struct {
	BaseURL     string              `json:"baseUrl"`
	Version     string              `json:"version,omitempty"`
	AuthMethod  string              `json:"authMethod"` // "api_key" | "session"
	APIKeyID    *uint               `json:"apiKeyId,omitempty"`
	Scopes      []string            `json:"scopes"` // empty for session JWT
	User        WhoamiUser          `json:"user"`
	Permissions []string            `json:"permissions"`
	Capabilities WhoamiCapabilities `json:"capabilities"`
}

// WhoamiUser is the authenticated principal (key owner or session user).
type WhoamiUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// WhoamiCapabilities summarizes effective content powers (RBAC ∩ key scopes).
type WhoamiCapabilities struct {
	Articles bool `json:"articles"` // articles:read
	Pages    bool `json:"pages"`    // pages:read (unified /admin/pages)
	// ThemeContent is true when the agent can use theme-bound content_documents
	// Admin API (/admin/content/:pageKey/*). Gated by pages:read (same as content GET draft).
	ThemeContent bool `json:"themeContent"`
	// ThemeContentKeys lists content pageKeys: theme slots when declared, else host whitelist.
	ThemeContentKeys []string `json:"themeContentKeys,omitempty"`
	// Active theme discovery (for multi-theme agents).
	ActiveThemeID      string   `json:"activeThemeId,omitempty"`
	ActiveThemeVersion string   `json:"activeThemeVersion,omitempty"`
	ContentSlots       []string `json:"contentSlots,omitempty"` // pageKeys from theme contentSlots
	MediaUpload        bool     `json:"mediaUpload"`             // media:create
	AIArticleMeta      bool     `json:"aiArticleMeta"`           // articles:update
	Publish            bool     `json:"publish"`                 // articles:publish or pages:publish
}

// Whoami GET /admin/agent/whoami
// Authenticated only (JWT or ink_… key). No extra permission required so any
// scoped key can verify it is talking to the intended instance before writes.
//
// @Summary      Agent whoami / instance probe
// @Description  Returns baseUrl, auth method, scopes, and effective capabilities for multi-site agents
// @Tags         Agent
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} WhoamiResponse
// @Failure      401 {object} object{error=string}
// @Router       /admin/agent/whoami [get]
func (h *Handler) Whoami(c *gin.Context) {
	uc := middleware.GetUserContext(c)
	if uc == nil {
		apierror.Message(c, http.StatusUnauthorized, "需要登录")
		return
	}

	user := middleware.GetRBACUser(c)
	permissions := []string{}
	if user != nil {
		permissions = user.EffectivePermissionKeys()
	}

	scopes := middleware.GetAPIKeyScopes(c)
	if scopes == nil {
		scopes = []string{}
	}
	keyID := middleware.GetAPIKeyID(c)
	authMethod := "session"
	var keyIDPtr *uint
	if keyID != 0 {
		authMethod = "api_key"
		id := keyID
		keyIDPtr = &id
	}

	role := string(uc.Role)
	if user != nil {
		role = string(user.Role)
	}

	canTheme := h.can(c, user, "pages", "read")
	caps := WhoamiCapabilities{
		Articles:      h.can(c, user, "articles", "read"),
		Pages:         h.can(c, user, "pages", "read"),
		ThemeContent:  canTheme,
		MediaUpload:   h.can(c, user, "media", "create"),
		AIArticleMeta: h.can(c, user, "articles", "update"),
		Publish: h.can(c, user, "articles", "publish") ||
			h.can(c, user, "pages", "publish"),
	}

	// Theme contentSlots discovery
	if h.slots != nil {
		res := h.slots.ResolveActive(c.Request.Context())
		caps.ActiveThemeID = res.ActiveThemeID
		caps.ActiveThemeVersion = res.ActiveThemeVersion
		if len(res.Slots) > 0 {
			keys := make([]string, 0, len(res.Slots))
			for _, s := range res.Slots {
				keys = append(keys, s.PageKey)
			}
			caps.ContentSlots = keys
			if canTheme {
				caps.ThemeContentKeys = keys
			}
		}
	}
	if canTheme && len(caps.ThemeContentKeys) == 0 {
		caps.ThemeContentKeys = hostContentPageKeys()
	}

	resp := WhoamiResponse{
		BaseURL:     h.baseURL,
		Version:     h.version,
		AuthMethod:  authMethod,
		APIKeyID:    keyIDPtr,
		Scopes:      scopes,
		User: WhoamiUser{
			ID:       uc.UserID,
			Username: uc.Username,
			Role:     role,
		},
		Permissions:  permissions,
		Capabilities: caps,
	}
	c.JSON(http.StatusOK, resp)
}

// hostContentPageKeys returns full host content_documents whitelist (legacy discovery).
func hostContentPageKeys() []string {
	return contentslots.HostPageKeys()
}

func (h *Handler) can(c *gin.Context, user *model.User, resource, action string) bool {
	if user == nil || !user.HasRBACPermission(resource, action) {
		return false
	}
	scopes := middleware.GetAPIKeyScopes(c)
	if scopes == nil {
		// Session JWT: RBAC only.
		return true
	}
	need := resource + ":" + action
	for _, s := range scopes {
		if s == need {
			return true
		}
	}
	return false
}
