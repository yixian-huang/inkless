package unified_page

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/eventbus"
	auditlogHandler "blotting-consultancy/internal/handler/auditlog"
	bootstrapHandler "blotting-consultancy/internal/handler/bootstrap"
	"blotting-consultancy/internal/middleware"
	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
	"blotting-consultancy/internal/service"
	"blotting-consultancy/pkg/audit"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUnifiedPagePublishWorkflowUpdatesBootstrapAndPublicRoute(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:unified-page-workflow?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.ContentDocument{},
		&model.InstalledTheme{},
		&model.Page{},
		&model.UnifiedPage{},
		&model.PageVersion{},
		&model.SiteConfig{},
		&model.AuditEvent{},
	))

	pageRepo := repository.NewGormUnifiedPageRepository(db)
	versionRepo := repository.NewGormPageVersionRepository(db)
	auditRepo := repository.NewGormAuditEventRepository(db)
	auditWriter := audit.NewDbWriter(auditRepo)
	publicCache := cache.New(time.Minute)
	defer publicCache.Stop()
	bus := eventbus.New()
	var lifecycleEvents []string
	for _, eventType := range []string{
		eventbus.ContentCreated,
		eventbus.ContentUpdated,
		eventbus.ContentDraftUpdated,
		eventbus.ContentPublished,
		eventbus.ContentUnpublished,
		eventbus.ContentDeleted,
	} {
		bus.Subscribe(eventType, eventbus.SyncHandler(func(event eventbus.Event) {
			lifecycleEvents = append(lifecycleEvents, event.Type)
		}))
	}

	pageHandler := NewHandler(
		pageRepo,
		versionRepo,
		service.NewUnifiedPageService(pageRepo, versionRepo, bus).WithAuditWriter(auditWriter),
		publicCache,
		bus,
	)
	auditHandler := auditlogHandler.NewHandler(auditRepo)
	bootstrap := bootstrapHandler.NewHandler(
		repository.NewGormContentDocumentRepository(db),
		repository.NewGormInstalledThemeRepository(db),
		repository.NewGormPageRepository(db),
		pageRepo,
		repository.NewGormSiteConfigRepository(db),
		publicCache,
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuditContext())
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.UserContextKey), &middleware.UserContext{
			UserID:   9,
			Username: "publisher",
			Role:     model.RoleAdmin,
		})
		c.Next()
	})
	router.Use(middleware.AuditMutations(auditWriter))
	router.POST("/admin/pages", pageHandler.AdminCreate)
	router.PUT("/admin/pages/:id", pageHandler.AdminUpdate)
	router.PUT("/admin/pages/:id/draft", pageHandler.AdminUpdateDraft)
	router.POST("/admin/pages/:id/publish", pageHandler.AdminPublish)
	router.POST("/admin/pages/:id/unpublish", pageHandler.AdminUnpublish)
	router.DELETE("/admin/pages/:id", pageHandler.AdminDelete)
	router.GET("/admin/audit-logs", auditHandler.List)
	router.GET("/public/bootstrap", bootstrap.PublicBootstrap)
	router.GET("/public/pages/:slug", pageHandler.PublicGetBySlug)

	initialBootstrap := performJSONRequest(t, router, http.MethodGet, "/public/bootstrap?locale=zh", nil, nil)
	require.Equal(t, http.StatusOK, initialBootstrap.Code)
	require.Empty(t, decodeUnifiedPageFacts(t, initialBootstrap.Body.Bytes()))

	create := performJSONRequest(t, router, http.MethodPost, "/admin/pages", map[string]any{
		"slug":      "launch-page",
		"zhTitle":   "发布页",
		"enTitle":   "Launch Page",
		"mode":      model.PageModeComposable,
		"showInNav": true,
		"sortOrder": 4,
		"draftConfig": map[string]any{
			"sections": []any{},
		},
	}, nil)
	require.Equal(t, http.StatusCreated, create.Code, create.Body.String())

	var created model.UnifiedPage
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	require.NotZero(t, created.ID)
	require.Equal(t, 1, created.DraftVersion)

	updateMetadata := performJSONRequest(
		t,
		router,
		http.MethodPut,
		fmt.Sprintf("/admin/pages/%d", created.ID),
		map[string]any{
			"slug":      "launch-page",
			"zhTitle":   "发布页",
			"enTitle":   "Launch Page",
			"showInNav": true,
			"sortOrder": 4,
		},
		nil,
	)
	require.Equal(t, http.StatusOK, updateMetadata.Code, updateMetadata.Body.String())

	updateDraft := performJSONRequest(
		t,
		router,
		http.MethodPut,
		fmt.Sprintf("/admin/pages/%d/draft", created.ID),
		map[string]any{
			"draftConfig": map[string]any{
				"sections": []any{
					map[string]any{
						"id":       "launch-copy",
						"type":     "rich-text",
						"variant":  "default",
						"data":     map[string]any{"content": "Wave 1 published page"},
						"settings": map[string]any{},
					},
				},
			},
		},
		map[string]string{"If-Match": "1"},
	)
	require.Equal(t, http.StatusOK, updateDraft.Code, updateDraft.Body.String())

	publish := performJSONRequest(
		t,
		router,
		http.MethodPost,
		fmt.Sprintf("/admin/pages/%d/publish", created.ID),
		map[string]any{"expectedDraftVersion": 2},
		nil,
	)
	require.Equal(t, http.StatusOK, publish.Code, publish.Body.String())

	publishAudit := performJSONRequest(
		t,
		router,
		http.MethodGet,
		"/admin/audit-logs?action=content.publish&actor=publisher",
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, publishAudit.Code, publishAudit.Body.String())
	var auditPayload struct {
		Items []model.AuditEvent `json:"items"`
		Total int64              `json:"total"`
	}
	require.NoError(t, json.Unmarshal(publishAudit.Body.Bytes(), &auditPayload))
	require.Equal(t, int64(1), auditPayload.Total)
	require.Len(t, auditPayload.Items, 1)
	require.Equal(t, "success", auditPayload.Items[0].Result)
	require.Equal(t, fmt.Sprintf("pages:%d", created.ID), auditPayload.Items[0].Resource)

	publishedBootstrap := performJSONRequest(t, router, http.MethodGet, "/public/bootstrap?locale=zh", nil, nil)
	require.Equal(t, http.StatusOK, publishedBootstrap.Code)
	publishedFacts := decodeUnifiedPageFacts(t, publishedBootstrap.Body.Bytes())
	require.Len(t, publishedFacts, 1)
	require.Equal(t, "launch-page", publishedFacts[0]["slug"])
	require.Equal(t, true, publishedFacts[0]["showInNav"])

	publicPage := performJSONRequest(t, router, http.MethodGet, "/public/pages/launch-page?locale=zh", nil, nil)
	require.Equal(t, http.StatusOK, publicPage.Code, publicPage.Body.String())
	var publicPayload map[string]any
	require.NoError(t, json.Unmarshal(publicPage.Body.Bytes(), &publicPayload))
	require.Equal(t, "发布页", publicPayload["title"])
	require.NotNil(t, publicPayload["publishedConfig"])

	unpublish := performJSONRequest(
		t,
		router,
		http.MethodPost,
		fmt.Sprintf("/admin/pages/%d/unpublish", created.ID),
		map[string]any{},
		nil,
	)
	require.Equal(t, http.StatusOK, unpublish.Code, unpublish.Body.String())

	unpublishedBootstrap := performJSONRequest(t, router, http.MethodGet, "/public/bootstrap?locale=zh", nil, nil)
	require.Equal(t, http.StatusOK, unpublishedBootstrap.Code)
	require.Empty(t, decodeUnifiedPageFacts(t, unpublishedBootstrap.Body.Bytes()))

	unpublishedPublicPage := performJSONRequest(t, router, http.MethodGet, "/public/pages/launch-page?locale=zh", nil, nil)
	require.Equal(t, http.StatusNotFound, unpublishedPublicPage.Code, unpublishedPublicPage.Body.String())

	deleted := performJSONRequest(
		t,
		router,
		http.MethodDelete,
		fmt.Sprintf("/admin/pages/%d", created.ID),
		nil,
		nil,
	)
	require.Equal(t, http.StatusOK, deleted.Code, deleted.Body.String())
	require.Equal(t, []string{
		eventbus.ContentCreated,
		eventbus.ContentUpdated,
		eventbus.ContentDraftUpdated,
		eventbus.ContentPublished,
		eventbus.ContentUnpublished,
		eventbus.ContentDeleted,
	}, lifecycleEvents)
}

func performJSONRequest(
	t *testing.T,
	router http.Handler,
	method string,
	target string,
	body any,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		requestBody = bytes.NewReader(encoded)
	}

	request := httptest.NewRequest(method, target, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeUnifiedPageFacts(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	var payload struct {
		UnifiedPages []map[string]any `json:"unifiedPages"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.UnifiedPages
}
