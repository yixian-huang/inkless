package article

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/contentexcerpt"
	"github.com/yixian-huang/inkless/backend/internal/eventbus"
	"github.com/yixian-huang/inkless/backend/internal/handlerutil"
	"github.com/yixian-huang/inkless/backend/internal/middleware"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// pageViewTracker is implemented by service.PageViewRecorder.
type pageViewTracker interface {
	Track(pageKey, locale, visitorID, referer string)
}

// Handler handles article-related HTTP requests
type Handler struct {
	articleRepo   repository.ArticleRepository
	versionRepo   repository.ArticleVersionRepository
	categoryRepo  repository.CategoryRepository
	tagRepo       repository.TagRepository
	pvRepo        repository.PageViewRepository
	viewTracker   pageViewTracker
	searchService *service.SearchService
	articleSvc    *service.ArticlePublicationService
	eventBus      eventbus.EventBus
	cache         *cache.Cache
}

// NewHandler creates a new article handler
func NewHandler(
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	tagRepo repository.TagRepository,
	searchService *service.SearchService,
	eventBus eventbus.EventBus,
	cache *cache.Cache,
) *Handler {
	return &Handler{
		articleRepo:   articleRepo,
		categoryRepo:  categoryRepo,
		tagRepo:       tagRepo,
		searchService: searchService,
		articleSvc:    service.NewArticlePublicationService(articleRepo, searchService, eventBus),
		eventBus:      eventBus,
		cache:         cache,
	}
}

// WithVersionRepo enables article version history + comparison endpoints.
func (h *Handler) WithVersionRepo(repo repository.ArticleVersionRepository) *Handler {
	h.versionRepo = repo
	return h
}

// WithPageViews enables visit tracking and viewCount on public article detail.
func (h *Handler) WithPageViews(pvRepo repository.PageViewRepository) *Handler {
	h.pvRepo = pvRepo
	return h
}

// WithViewTracker sets the async page-view recorder (preferred over sync Create).
func (h *Handler) WithViewTracker(t pageViewTracker) *Handler {
	h.viewTracker = t
	return h
}

// --- Public endpoints ---

// PublicList returns a paginated list of published articles.
// @Summary      List published articles
// @Description  Returns paginated published articles with optional category/tag filtering
// @Tags         Articles
// @Produce      json
// @Param        page      query int    false "Page number"    default(1)
// @Param        pageSize  query int    false "Items per page" default(10)
// @Param        category  query string false "Category slug filter"
// @Param        tag       query string false "Tag slug filter"
// @Success      200 {object} object{items=[]object,total=int,page=int,pageSize=int}
// @Router       /public/articles [get]
func (h *Handler) PublicList(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 10, 100)
	page, pageSize := p.Page, p.PageSize

	categorySlug := c.Query("category")
	tagSlug := c.Query("tag")

	cacheKey := "articles:list:" + strconv.Itoa(page) + ":" + strconv.Itoa(pageSize) + ":" + categorySlug + ":" + tagSlug
	if cached, ok := h.cache.Get(cacheKey); ok {
		c.Header("X-Cache", "HIT")
		c.Header("Cache-Control", "public, max-age=30, stale-while-revalidate=15")
		c.JSON(http.StatusOK, cached)
		return
	}

	offset := (page - 1) * pageSize

	items, total, err := h.articleRepo.ListPublished(c.Request.Context(), offset, pageSize, categorySlug, tagSlug)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询文章失败")
		return
	}

	// Replace full HTML bodies with plain-text excerpts for list cards
	// (home / archive). Detail view still loads full body via PublicGetBySlug.
	contentexcerpt.ApplyListExcerpts(items)

	result := gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}
	h.cache.Set(cacheKey, result)
	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", "public, max-age=30, stale-while-revalidate=15")
	c.JSON(http.StatusOK, result)
}

// publicArticleDTO embeds article fields and adds viewCount for public readers.
type publicArticleDTO struct {
	model.Article
	ViewCount int64 `json:"viewCount"`
}

// articlePageKey builds the page_views key for an article.
func articlePageKey(articleID uint) string {
	return fmt.Sprintf("article:%d", articleID)
}

// PublicGetBySlug returns a single published article by slug.
// @Summary      Get article by slug
// @Description  Returns a single published article with full content
// @Tags         Articles
// @Produce      json
// @Param        slug path string true "Article slug"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /public/articles/{slug} [get]
func (h *Handler) PublicGetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	var article *model.Article
	cacheKey := "article:" + slug
	if cached, ok := h.cache.Get(cacheKey); ok {
		switch v := cached.(type) {
		case *model.Article:
			article = v
			c.Header("X-Cache", "HIT")
		case model.Article:
			a := v
			article = &a
			c.Header("X-Cache", "HIT")
		}
	}

	if article == nil {
		found, err := h.articleRepo.FindBySlug(c.Request.Context(), slug)
		if err != nil {
			apierror.Message(c, http.StatusNotFound, "文章不存在")
			return
		}

		if found.Status != model.ArticleStatusPublished {
			apierror.Message(c, http.StatusNotFound, "文章不存在")
			return
		}

		// Only return publicly visible articles
		if found.Visibility != "" && found.Visibility != "public" {
			apierror.Message(c, http.StatusNotFound, "文章不存在")
			return
		}

		article = found
		h.cache.Set(cacheKey, article)
		c.Header("X-Cache", "MISS")
	}

	pageKey := articlePageKey(article.ID)
	// Async best-effort tracking (never blocks the response).
	h.recordArticleViewAsync(pageKey, c)

	// Approximate viewCount with a short-TTL cache so cache-hit article
	// responses do not issue a COUNT on every request.
	viewCount := h.cachedViewCount(c.Request.Context(), pageKey)

	c.Header("Cache-Control", "public, max-age=60, stale-while-revalidate=30")
	c.JSON(http.StatusOK, publicArticleDTO{
		Article:   *article,
		ViewCount: viewCount,
	})
}

const articleViewCountCacheTTL = 15 * time.Second

func (h *Handler) cachedViewCount(ctx context.Context, pageKey string) int64 {
	if h.pvRepo == nil {
		return 0
	}
	cacheKey := "article-views:" + pageKey
	if h.cache != nil {
		if cached, ok := h.cache.Get(cacheKey); ok {
			switch n := cached.(type) {
			case int64:
				return n
			case int:
				return int64(n)
			}
		}
	}
	n, err := h.pvRepo.CountByPageKey(ctx, pageKey)
	if err != nil {
		return 0
	}
	if h.cache != nil {
		h.cache.SetWithTTL(cacheKey, n, articleViewCountCacheTTL)
	}
	return n
}

func (h *Handler) recordArticleViewAsync(pageKey string, c *gin.Context) {
	clientIP := c.ClientIP()
	referer := c.GetHeader("Referer")
	locale := c.DefaultQuery("locale", "zh")
	hash := sha256.Sum256([]byte(clientIP))
	visitorID := fmt.Sprintf("%x", hash[:])[:16]

	if h.viewTracker != nil {
		h.viewTracker.Track(pageKey, locale, visitorID, referer)
		return
	}
	// Fallback when no recorder is wired (tests): fire-and-forget single insert.
	if h.pvRepo == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := h.pvRepo.Create(ctx, &model.PageView{
			PageKey:   pageKey,
			Locale:    locale,
			VisitorID: visitorID,
			Referer:   referer,
		}); err != nil {
			slog.Error("failed to record article page view", "pageKey", pageKey, "error", err)
		}
	}()
}

// --- Admin endpoints ---

// AdminList returns all articles for admin management.
// @Summary      List all articles (admin)
// @Description  Returns paginated articles including drafts for admin management
// @Tags         Articles (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int    false "Page number"    default(1)
// @Param        pageSize query int    false "Items per page" default(10)
// @Param        status   query string false "Status filter (draft/published/scheduled)"
// @Success      200 {object} object{items=[]object,total=int}
// @Failure      401 {object} object{error=string}
// @Router       /admin/articles [get]
func (h *Handler) AdminList(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 10, 100)
	status := handlerutil.QueryTrim(c, "status")
	q := handlerutil.QueryTrim(c, "q")

	var categoryID *uint
	if c.Query("categoryId") != "" {
		id, ok := handlerutil.ParseUintParamOptional(c, "categoryId")
		if !ok {
			return
		}
		if id > 0 {
			categoryID = &id
		}
	}
	var tagID *uint
	if c.Query("tagId") != "" {
		id, ok := handlerutil.ParseUintParamOptional(c, "tagId")
		if !ok {
			return
		}
		if id > 0 {
			tagID = &id
		}
	}

	sortClause := handlerutil.ParseSort(c, "sort", map[string]string{
		"created_at_desc":   "created_at DESC",
		"created_at_asc":    "created_at ASC",
		"updated_at_desc":   "updated_at DESC",
		"updated_at_asc":    "updated_at ASC",
		"published_at_desc": "published_at DESC",
		"published_at_asc":  "published_at ASC",
		"title_asc":         "zh_title ASC",
		"title_desc":        "zh_title DESC",
	}, "created_at_desc")

	items, total, err := h.articleRepo.ListFilter(c.Request.Context(), repository.ArticleListFilter{
		Offset:     p.Offset,
		Limit:      p.PageSize,
		Status:     status,
		CategoryID: categoryID,
		TagID:      tagID,
		Query:      q,
		Sort:       sortClause,
	})
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询文章失败")
		return
	}

	handlerutil.ListResponse(c, items, total, p)
}

// AdminGetByID returns a single article by ID.
// @Summary      Get article by ID (admin)
// @Description  Returns a single article by its database ID
// @Tags         Articles (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Article ID"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/articles/{id} [get]
func (h *Handler) AdminGetByID(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	article, err := h.articleRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "文章不存在")
		return
	}

	c.JSON(http.StatusOK, article)
}

// createUpdateInput is the JSON body for creating/updating articles
type createUpdateInput struct {
	Slug              string        `json:"slug"`
	Status            string        `json:"status"`
	ZhTitle           string        `json:"zhTitle"`
	EnTitle           string        `json:"enTitle"`
	ZhBody            string        `json:"zhBody"`
	EnBody            string        `json:"enBody"`
	CoverImage        string        `json:"coverImage"`
	ZhSeoTitle        string        `json:"zhSeoTitle"`
	EnSeoTitle        string        `json:"enSeoTitle"`
	ZhMetaDescription string        `json:"zhMetaDescription"`
	EnMetaDescription string        `json:"enMetaDescription"`
	OgImage           string        `json:"ogImage"`
	CategoryID        *uint         `json:"categoryId"`
	CategoryIDs       []uint        `json:"categoryIds"`
	TagIDs            []uint        `json:"tagIds"`
	Author            string        `json:"author"`
	AutoSummary       bool          `json:"autoSummary"`
	AllowComments     *bool         `json:"allowComments"`
	Pinned            bool          `json:"pinned"`
	Visibility        string        `json:"visibility"`
	Metadata          model.JSONMap `json:"metadata"`
	ScheduledAt       *time.Time    `json:"scheduledAt"`
	// BaseUpdatedAt is the client's last-known updatedAt for optimistic concurrency.
	// When set (and Force is false), the update is rejected with 409 on mismatch.
	BaseUpdatedAt *time.Time `json:"baseUpdatedAt"`
	// Force skips optimistic lock and overwrites the latest row.
	Force bool `json:"force"`
}

// AdminCreate creates a new article.
// @Summary      Create article
// @Description  Create a new article (draft by default)
// @Tags         Articles (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createUpdateInput true "Article data"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/articles [post]
func (h *Handler) AdminCreate(c *gin.Context) {
	var input createUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	status := model.ArticleStatus(input.Status)
	if status == "" {
		status = model.ArticleStatusDraft
	}

	allowComments := true
	if input.AllowComments != nil {
		allowComments = *input.AllowComments
	}

	visibility := input.Visibility
	if visibility == "" {
		visibility = "public"
	}

	article := &model.Article{
		Slug:              input.Slug,
		Status:            status,
		ZhTitle:           input.ZhTitle,
		EnTitle:           input.EnTitle,
		ZhBody:            input.ZhBody,
		EnBody:            input.EnBody,
		CoverImage:        input.CoverImage,
		ZhSeoTitle:        input.ZhSeoTitle,
		EnSeoTitle:        input.EnSeoTitle,
		ZhMetaDescription: input.ZhMetaDescription,
		EnMetaDescription: input.EnMetaDescription,
		OgImage:           input.OgImage,
		CategoryID:        input.CategoryID,
		Author:            input.Author,
		AutoSummary:       input.AutoSummary,
		AllowComments:     allowComments,
		Pinned:            input.Pinned,
		Visibility:        visibility,
		Metadata:          input.Metadata,
	}

	if status == model.ArticleStatusPublished {
		now := time.Now()
		article.PublishedAt = &now
		contentexcerpt.FillStoredExcerpts(article)
	}
	if input.ScheduledAt != nil {
		article.ScheduledAt = input.ScheduledAt
	}

	// Resolve categories
	if len(input.CategoryIDs) > 0 {
		categories, err := h.resolveCategoryIDs(c, input.CategoryIDs)
		if err != nil {
			return
		}
		article.Categories = categories
	}

	// Resolve tags
	if len(input.TagIDs) > 0 {
		tags, err := h.resolveTagIDs(c, input.TagIDs)
		if err != nil {
			return // error already written to response
		}
		article.Tags = tags
	}

	if err := h.articleRepo.Create(c.Request.Context(), article); err != nil {
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}

	// Re-fetch with preloads
	created, err := h.articleRepo.FindByID(c.Request.Context(), article.ID)
	if err != nil {
		c.JSON(http.StatusCreated, article)
		return
	}

	// Snapshot initial version for history
	action := "create"
	if created.Status == model.ArticleStatusPublished {
		action = "publish"
	}
	h.recordArticleVersion(c, created, action)

	if article.Status == model.ArticleStatusPublished && h.articleSvc != nil {
		go h.articleSvc.AfterPublish(context.Background(), article, 0)
	}

	// Publish content created event
	if h.eventBus != nil {
		h.eventBus.Publish(eventbus.Event{
			Type: eventbus.ContentCreated,
			Payload: eventbus.ContentEventPayload{
				ContentType: "article",
				ContentID:   article.ID,
				Slug:        article.Slug,
				Action:      eventbus.ContentCreated,
			},
		})
	}

	c.JSON(http.StatusCreated, created)
}

// AdminUpdate updates an existing article.
// @Summary      Update article
// @Description  Update an existing article by ID
// @Tags         Articles (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int    true "Article ID"
// @Param        body body createUpdateInput true "Updated article data"
// @Success      200 {object} object
// @Failure      404 {object} object{error=string}
// @Router       /admin/articles/{id} [put]
func (h *Handler) AdminUpdate(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	existing, err := h.articleRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "文章不存在")
		return
	}
	previousStatus := existing.Status

	var input createUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	// Update fields
	existing.Slug = input.Slug
	existing.ZhTitle = input.ZhTitle
	existing.EnTitle = input.EnTitle
	existing.ZhBody = input.ZhBody
	existing.EnBody = input.EnBody
	existing.CoverImage = input.CoverImage
	existing.ZhSeoTitle = input.ZhSeoTitle
	existing.EnSeoTitle = input.EnSeoTitle
	existing.ZhMetaDescription = input.ZhMetaDescription
	existing.EnMetaDescription = input.EnMetaDescription
	existing.OgImage = input.OgImage
	existing.CategoryID = input.CategoryID
	existing.Author = input.Author
	existing.AutoSummary = input.AutoSummary
	existing.Pinned = input.Pinned
	if input.AllowComments != nil {
		existing.AllowComments = *input.AllowComments
	}
	if input.Visibility != "" {
		existing.Visibility = input.Visibility
	}
	if input.Metadata != nil {
		existing.Metadata = input.Metadata
	}

	if input.ScheduledAt != nil {
		existing.ScheduledAt = input.ScheduledAt
	}

	if input.Status != "" {
		newStatus := model.ArticleStatus(input.Status)
		// Set publishedAt when transitioning to published
		if newStatus == model.ArticleStatusPublished && existing.Status != model.ArticleStatusPublished {
			now := time.Now()
			existing.PublishedAt = &now
		}
		existing.Status = newStatus
	}
	if existing.Status == model.ArticleStatusPublished {
		contentexcerpt.FillStoredExcerpts(existing)
	}

	// Resolve categories: prefer CategoryIDs, fallback to CategoryID
	if input.CategoryIDs != nil {
		categories, err := h.resolveCategoryIDs(c, input.CategoryIDs)
		if err != nil {
			return
		}
		existing.Categories = categories
	}

	// Resolve tags
	if input.TagIDs != nil {
		tags, err := h.resolveTagIDs(c, input.TagIDs)
		if err != nil {
			return
		}
		existing.Tags = tags
	}

	// Optimistic concurrency: when the client supplies baseUpdatedAt, require a match
	// unless Force is set (explicit overwrite after conflict dialog).
	var updateErr error
	if input.BaseUpdatedAt != nil && !input.Force {
		updateErr = h.articleRepo.UpdateIfMatch(c.Request.Context(), existing, *input.BaseUpdatedAt)
	} else {
		updateErr = h.articleRepo.Update(c.Request.Context(), existing)
	}
	if updateErr != nil {
		if errors.Is(updateErr, repository.ErrArticleVersionConflict) {
			// Re-read current row so the client can show server timestamp / reload.
			current, _ := h.articleRepo.FindByID(c.Request.Context(), existing.ID)
			details := map[string]any{"code": "version_conflict"}
			if current != nil {
				details["currentUpdatedAt"] = current.UpdatedAt
			}
			apierror.Write(c, apierror.Conflict("文章已被他人修改，请重新加载或强制覆盖").WithDetails(details))
			return
		}
		apierror.Message(c, http.StatusBadRequest, updateErr.Error())
		return
	}

	// Re-fetch with preloads
	updated, err := h.articleRepo.FindByID(c.Request.Context(), existing.ID)
	if err != nil {
		c.JSON(http.StatusOK, existing)
		return
	}

	// Snapshot after successful save for version history / comparison
	action := "save"
	if updated.Status == model.ArticleStatusPublished && previousStatus != model.ArticleStatusPublished {
		action = "publish"
	} else if updated.Status == model.ArticleStatusPublished {
		action = "update"
	}
	h.recordArticleVersion(c, updated, action)

	// Auto-index on publish, remove from index otherwise
	if h.articleSvc != nil {
		if existing.Status == model.ArticleStatusPublished {
			if previousStatus != model.ArticleStatusPublished {
				go h.articleSvc.AfterPublish(context.Background(), existing, 0)
			} else {
				go h.articleSvc.RefreshPublished(context.Background(), existing)
			}
		} else if previousStatus == model.ArticleStatusPublished {
			go h.articleSvc.AfterUnpublish(context.Background(), existing)
		}
	}

	// Publish content updated event
	if h.eventBus != nil {
		h.eventBus.Publish(eventbus.Event{
			Type: eventbus.ContentUpdated,
			Payload: eventbus.ContentEventPayload{
				ContentType: "article",
				ContentID:   existing.ID,
				Slug:        existing.Slug,
				Action:      eventbus.ContentUpdated,
			},
		})
	}

	c.JSON(http.StatusOK, updated)
}

// AdminDelete deletes an article.
// @Summary      Delete article
// @Description  Permanently delete an article by ID
// @Tags         Articles (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Article ID"
// @Success      200 {object} object{message=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/articles/{id} [delete]
func (h *Handler) AdminDelete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.articleRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusNotFound, "文章不存在")
		return
	}

	// Remove from search index
	if h.searchService != nil {
		go func() {
			h.searchService.RemoveFromIndex(context.Background(), "article", id)
		}()
	}

	// Publish content deleted event
	if h.eventBus != nil {
		h.eventBus.Publish(eventbus.Event{
			Type: eventbus.ContentDeleted,
			Payload: eventbus.ContentEventPayload{
				ContentType: "article",
				ContentID:   id,
				Action:      eventbus.ContentDeleted,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// --- Version history endpoints ---

// AdminListVersions lists version history for an article.
func (h *Handler) AdminListVersions(c *gin.Context) {
	if h.versionRepo == nil {
		apierror.Message(c, http.StatusNotImplemented, "版本历史未启用")
		return
	}
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}
	p := handlerutil.ParsePagination(c, 20, 100)
	page, pageSize := p.Page, p.PageSize
	offset := p.Offset

	versions, total, err := h.versionRepo.ListByArticleID(c.Request.Context(), id, offset, pageSize)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询版本失败")
		return
	}

	// Lightweight list items (omit full body from list payload for speed)
	items := make([]gin.H, 0, len(versions))
	for _, v := range versions {
		items = append(items, gin.H{
			"id":        v.ID,
			"articleId": v.ArticleID,
			"version":   v.Version,
			"action":    v.Action,
			"summary":   v.Summary,
			"createdBy": v.CreatedBy,
			"createdAt": v.CreatedAt,
			// Include titles from snapshot for list display
			"zhTitle": snapshotString(v.Snapshot, "zhTitle"),
			"enTitle": snapshotString(v.Snapshot, "enTitle"),
			"status":  snapshotString(v.Snapshot, "status"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AdminGetVersion returns a specific version snapshot.
func (h *Handler) AdminGetVersion(c *gin.Context) {
	if h.versionRepo == nil {
		apierror.Message(c, http.StatusNotImplemented, "版本历史未启用")
		return
	}
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}
	versionNum, err := strconv.Atoi(c.Param("version"))
	if err != nil || versionNum < 1 {
		apierror.Message(c, http.StatusBadRequest, "无效的版本号")
		return
	}
	v, err := h.versionRepo.FindByArticleIDAndVersion(c.Request.Context(), id, versionNum)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "版本不存在")
		return
	}
	c.JSON(http.StatusOK, v)
}

// AdminCompareVersions returns two version snapshots for client-side comparison.
// Query: left=<version>&right=<version>  (defaults: previous vs latest)
func (h *Handler) AdminCompareVersions(c *gin.Context) {
	if h.versionRepo == nil {
		apierror.Message(c, http.StatusNotImplemented, "版本历史未启用")
		return
	}
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}
	articleID := id

	leftNum, _ := strconv.Atoi(c.Query("left"))
	rightNum, _ := strconv.Atoi(c.Query("right"))

	// Default: latest two versions
	if leftNum < 1 || rightNum < 1 {
		latest, err := h.versionRepo.GetLatestVersion(c.Request.Context(), articleID)
		if err != nil || latest < 1 {
			apierror.Message(c, http.StatusNotFound, "暂无版本记录")
			return
		}
		if rightNum < 1 {
			rightNum = latest
		}
		if leftNum < 1 {
			leftNum = latest - 1
			if leftNum < 1 {
				leftNum = 1
			}
		}
	}

	left, err := h.versionRepo.FindByArticleIDAndVersion(c.Request.Context(), articleID, leftNum)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, fmt.Sprintf("左版本 v%d 不存在", leftNum))
		return
	}
	right, err := h.versionRepo.FindByArticleIDAndVersion(c.Request.Context(), articleID, rightNum)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, fmt.Sprintf("右版本 v%d 不存在", rightNum))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"left":  left,
		"right": right,
	})
}

// recordArticleVersion writes a snapshot of the article after create/update.
// Failures are logged but do not fail the parent request.
func (h *Handler) recordArticleVersion(c *gin.Context, article *model.Article, action string) {
	if h.versionRepo == nil || article == nil {
		return
	}
	ctx := c.Request.Context()
	latest, err := h.versionRepo.GetLatestVersion(ctx, article.ID)
	if err != nil {
		slog.Warn("article version: get latest failed", "articleId", article.ID, "err", err)
		return
	}
	var createdBy uint
	if uc := middleware.GetUserContext(c); uc != nil {
		createdBy = uc.UserID
	}
	summary := article.ZhTitle
	if summary == "" {
		summary = article.EnTitle
	}
	if len(summary) > 200 {
		summary = summary[:200]
	}
	v := &model.ArticleVersion{
		ArticleID: article.ID,
		Version:   latest + 1,
		Snapshot:  articleToSnapshot(article),
		Action:    action,
		Summary:   summary,
		CreatedBy: createdBy,
	}
	if err := h.versionRepo.Create(ctx, v); err != nil {
		slog.Warn("article version: create failed", "articleId", article.ID, "err", err)
	}
}

func articleToSnapshot(a *model.Article) model.JSONMap {
	snap := model.JSONMap{
		"id":                a.ID,
		"slug":              a.Slug,
		"status":            string(a.Status),
		"zhTitle":           a.ZhTitle,
		"enTitle":           a.EnTitle,
		"zhBody":            a.ZhBody,
		"enBody":            a.EnBody,
		"coverImage":        a.CoverImage,
		"zhSeoTitle":        a.ZhSeoTitle,
		"enSeoTitle":        a.EnSeoTitle,
		"zhMetaDescription": a.ZhMetaDescription,
		"enMetaDescription": a.EnMetaDescription,
		"ogImage":           a.OgImage,
		"author":            a.Author,
		"autoSummary":       a.AutoSummary,
		"allowComments":     a.AllowComments,
		"pinned":            a.Pinned,
		"visibility":        a.Visibility,
	}
	if a.Metadata != nil {
		snap["metadata"] = a.Metadata
	}
	if a.CategoryID != nil {
		snap["categoryId"] = *a.CategoryID
	}
	if a.PublishedAt != nil {
		snap["publishedAt"] = a.PublishedAt.Format(time.RFC3339)
	}
	return snap
}

func snapshotString(snap model.JSONMap, key string) string {
	if snap == nil {
		return ""
	}
	if v, ok := snap[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// resolveCategoryIDs looks up categories by their IDs and returns them
func (h *Handler) resolveCategoryIDs(c *gin.Context, categoryIDs []uint) ([]model.Category, error) {
	categories, err := h.categoryRepo.FindByIDs(c.Request.Context(), categoryIDs)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询分类失败")
		return nil, err
	}
	if len(categories) != len(categoryIDs) {
		found := make(map[uint]bool, len(categories))
		for _, cat := range categories {
			found[cat.ID] = true
		}
		for _, id := range categoryIDs {
			if !found[id] {
				apierror.Message(c, http.StatusBadRequest, "分类 ID "+strconv.FormatUint(uint64(id), 10)+" 不存在")
				return nil, fmt.Errorf("category %d not found", id)
			}
		}
	}
	return categories, nil
}

// resolveTagIDs looks up tags by their IDs and returns them
func (h *Handler) resolveTagIDs(c *gin.Context, tagIDs []uint) ([]model.Tag, error) {
	tags, err := h.tagRepo.FindByIDs(c.Request.Context(), tagIDs)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询标签失败")
		return nil, err
	}
	if len(tags) != len(tagIDs) {
		found := make(map[uint]bool, len(tags))
		for _, tag := range tags {
			found[tag.ID] = true
		}
		for _, id := range tagIDs {
			if !found[id] {
				apierror.Message(c, http.StatusBadRequest, "标签 ID "+strconv.FormatUint(uint64(id), 10)+" 不存在")
				return nil, fmt.Errorf("tag %d not found", id)
			}
		}
	}
	return tags, nil
}
