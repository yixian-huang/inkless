package agent

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

// Handler serves agent-oriented discovery endpoints (multi-site fleet helpers).
type Handler struct {
	baseURL string
	version string
}

// NewHandler creates an agent handler. baseURL should be the instance BASE_URL (canonical).
func NewHandler(baseURL, version string) *Handler {
	return &Handler{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		version: version,
	}
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
	Articles      bool `json:"articles"`      // articles:read
	Pages         bool `json:"pages"`         // pages:read (unified /admin/pages)
	// ThemeContent is true when the agent can use theme-bound content_documents
	// Admin API (/admin/content/:pageKey/*). Gated by pages:read (same as content GET draft).
	ThemeContent bool `json:"themeContent"`
	// ThemeContentKeys lists writable/readable content pageKeys (agent discovery).
	ThemeContentKeys []string `json:"themeContentKeys,omitempty"`
	MediaUpload      bool     `json:"mediaUpload"`   // media:create
	AIArticleMeta    bool     `json:"aiArticleMeta"` // articles:update (article-meta endpoint)
	Publish          bool     `json:"publish"`       // articles:publish or pages:publish
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
		Permissions: permissions,
		Capabilities: WhoamiCapabilities{
			Articles:     h.can(c, user, "articles", "read"),
			Pages:        h.can(c, user, "pages", "read"),
			ThemeContent: h.can(c, user, "pages", "read"),
			ThemeContentKeys: themeContentPageKeys(
				h.can(c, user, "pages", "read"),
			),
			MediaUpload:   h.can(c, user, "media", "create"),
			AIArticleMeta: h.can(c, user, "articles", "update"),
			Publish: h.can(c, user, "articles", "publish") ||
				h.can(c, user, "pages", "publish"),
		},
	}
	c.JSON(http.StatusOK, resp)
}

// themeContentPageKeys returns content_documents page keys for agent discovery.
// Excludes internal-only keys (e.g. theme package blob).
func themeContentPageKeys(allowed bool) []string {
	if !allowed {
		return nil
	}
	out := make([]string, 0, len(model.ValidPageKeys))
	for _, k := range model.ValidPageKeys {
		if k == model.PageKeyTheme {
			continue
		}
		out = append(out, string(k))
	}
	return out
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
