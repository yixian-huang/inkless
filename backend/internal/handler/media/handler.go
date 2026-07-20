package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	stdDraw "image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"

	"github.com/yixian-huang/inkless/backend/pkg/apierror"
	xdraw "golang.org/x/image/draw"

	"github.com/yixian-huang/inkless/backend/internal/handlerutil"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/service"
)

// Handler handles media-related HTTP requests
type Handler struct {
	mediaRepo      repository.MediaRepository
	uploadDir      string
	baseURL        string
	storageRuntime *service.StorageRuntimeService
}

// NewHandler creates a new media handler
func NewHandler(mediaRepo repository.MediaRepository, uploadDir string, baseURL string) *Handler {
	return NewHandlerWithStorage(mediaRepo, uploadDir, baseURL, service.ConfigureDefaultStorageRuntime(nil, uploadDir))
}

// NewHandlerWithStorage creates a media handler with an explicit storage runtime.
func NewHandlerWithStorage(mediaRepo repository.MediaRepository, uploadDir string, baseURL string, storageRuntime *service.StorageRuntimeService) *Handler {
	if storageRuntime == nil {
		storageRuntime = service.ConfigureDefaultStorageRuntime(nil, uploadDir)
	}
	return &Handler{
		mediaRepo:      mediaRepo,
		uploadDir:      uploadDir,
		baseURL:        baseURL,
		storageRuntime: storageRuntime,
	}
}

// Upload handles file upload via multipart form.
// @Summary      Upload media file
// @Description  Upload an image file via multipart form data
// @Tags         Media
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Image file to upload"
// @Success      201 {object} object
// @Failure      400 {object} object{error=string}
// @Router       /admin/media/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		apierror.Message(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}
	defer file.Close()

	// Validate MIME type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		// Detect from file content
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		mimeType = http.DetectContentType(buf[:n])
		// Seek back to beginning
		if seeker, ok := file.(io.ReadSeeker); ok {
			seeker.Seek(0, io.SeekStart)
		}
	}

	if !strings.HasPrefix(mimeType, "image/") && !strings.HasPrefix(mimeType, "video/") && !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "font/") {
		apierror.Message(c, http.StatusBadRequest, "仅支持上传图片、视频、音频或字体文件（woff2/woff）")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		switch {
		case strings.HasPrefix(mimeType, "video/"):
			ext = ".mp4"
		case strings.HasPrefix(mimeType, "audio/"):
			ext = ".mp3"
		case strings.HasPrefix(mimeType, "font/"):
			if strings.Contains(mimeType, "woff2") {
				ext = ".woff2"
			} else {
				ext = ".woff"
			}
		default:
			ext = ".jpg"
		}
	}
	uniqueName := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), sanitizeFilename(strings.TrimSuffix(header.Filename, ext)), ext)

	// Reset file reader position
	if seeker, ok := file.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "读取文件失败")
		return
	}
	written := int64(len(data))
	storageKey, storageProvider, url, err := h.storageRuntime.Save(c.Request.Context(), uniqueName, bytes.NewReader(data), written)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "保存文件失败")
		return
	}

	// Try to get image dimensions (only for image files)
	var width, height *int
	if strings.HasPrefix(mimeType, "image/") {
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
			w := cfg.Width
			h := cfg.Height
			width = &w
			height = &h
		}
	}

	// Save to database
	media := &model.Media{
		URL:             url,
		Filename:        header.Filename,
		MimeType:        mimeType,
		Size:            written,
		Width:           width,
		Height:          height,
		StorageKey:      storageKey,
		StorageProvider: storageProvider,
	}

	if err := h.mediaRepo.Create(c.Request.Context(), media); err != nil {
		_ = h.storageRuntime.Delete(c.Request.Context(), storageProvider, storageKey)
		apierror.Message(c, http.StatusInternalServerError, "保存记录失败")
		return
	}

	// Async WebP conversion and thumbnail generation for non-WebP image uploads
	if storageProvider == "local" && strings.HasPrefix(mimeType, "image/") && !strings.EqualFold(ext, ".webp") {
		destPath := filepath.Join(h.uploadDir, storageKey)
		go func() {
			if err := generateWebP(destPath); err != nil {
				log.Printf("[media] generateWebP(%s): %v", destPath, err)
			}
			if err := generateThumbnail(destPath, 300); err != nil {
				log.Printf("[media] generateThumbnail(%s): %v", destPath, err)
			}
		}()
	}

	c.JSON(http.StatusCreated, media)
}

// List returns a paginated list of media items.
// @Summary      List media files
// @Description  Returns paginated list of uploaded media files
// @Tags         Media
// @Produce      json
// @Security     BearerAuth
// @Param        page     query int false "Page number"    default(1)
// @Param        pageSize query int false "Items per page" default(20)
// @Success      200 {object} object{items=[]object,total=int,page=int,pageSize=int}
// @Router       /admin/media [get]
func (h *Handler) List(c *gin.Context) {
	p := handlerutil.ParsePagination(c, 20, 100)

	// Optional MIME type prefix filter (e.g. ?type=image, ?type=video, ?type=audio)
	mimePrefix := ""
	if typeParam := handlerutil.QueryTrim(c, "type"); typeParam != "" {
		mimePrefix = typeParam + "/"
	}

	filter := repository.MediaListFilter{
		Offset:     p.Offset,
		Limit:      p.PageSize,
		MimePrefix: mimePrefix,
		Query:      handlerutil.QueryTrim(c, "q"),
		Sort: handlerutil.ParseSort(c, "sort", map[string]string{
			"created_at_desc": "created_at DESC",
			"created_at_asc":  "created_at ASC",
			"filename_asc":    "filename ASC",
			"filename_desc":   "filename DESC",
			"size_desc":       "size DESC",
			"size_asc":        "size ASC",
		}, "created_at_desc"),
	}

	// folderId empty = all; folderId=root|0 = unfiled; folderId=N = that folder.
	if raw := handlerutil.QueryTrim(c, "folderId"); raw != "" {
		if raw == "root" || raw == "0" {
			filter.FolderRoot = true
		} else {
			id, ok := handlerutil.ParseUintParamOptional(c, "folderId")
			if !ok {
				return
			}
			if id > 0 {
				filter.FolderID = &id
			}
		}
	}

	items, total, err := h.mediaRepo.ListFilter(c.Request.Context(), filter)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询失败")
		return
	}

	handlerutil.ListResponse(c, items, total, p)
}

// Delete removes a media item and its file.
// @Summary      Delete media file
// @Description  Remove a media file and its database record
// @Tags         Media
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Media ID"
// @Success      200 {object} object{message=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/media/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Find the media record
	media, err := h.mediaRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "未找到该媒体文件")
		return
	}

	storageProvider, storageKey := storageLocation(media)
	if err := h.storageRuntime.Delete(c.Request.Context(), storageProvider, storageKey); err != nil {
		if errors.Is(err, service.ErrStorageProviderUnavailable) {
			apierror.Message(c, http.StatusConflict, err.Error())
			return
		}
		apierror.Message(c, http.StatusInternalServerError, "删除文件失败")
		return
	}
	if storageProvider == "local" {
		h.removeLocalDerivatives(storageKey)
	}

	// Delete database record
	if err := h.mediaRepo.Delete(c.Request.Context(), id); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "删除记录失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *Handler) removeLocalDerivatives(storageKey string) {
	filePath := filepath.Join(h.uploadDir, storageKey)
	absUpload, _ := filepath.Abs(h.uploadDir)
	absFile, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFile, absUpload+string(filepath.Separator)) {
		return
	}
	base := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	os.Remove(base + ".webp")
	os.Remove(base + "_thumb.webp")
}

func storageLocation(media *model.Media) (string, string) {
	if media.StorageKey != "" {
		return media.StorageProvider, media.StorageKey
	}
	parts := strings.Split(media.URL, "/uploads/")
	if len(parts) == 2 {
		return "local", parts[1]
	}
	return media.StorageProvider, media.StorageKey
}

// Recrop replaces the physical file for an existing media item with a re-cropped version
// @Summary      Recrop media file
// @Description  Replaces the physical file for an existing media item with a new cropped version
// @Tags         Media (Admin)
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path     int  true "Media ID"
// @Param        file formData file true "Re-cropped image file"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/media/{id}/recrop [post]
func (h *Handler) Recrop(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	media, err := h.mediaRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "未找到该媒体文件")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		apierror.Message(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}
	defer file.Close()

	// Resolve physical file path from URL
	parts := strings.Split(media.URL, "/uploads/")
	if len(parts) != 2 {
		apierror.Message(c, http.StatusInternalServerError, "无法解析文件路径")
		return
	}
	destPath := filepath.Join(h.uploadDir, parts[1])
	// Verify resolved path is within uploadDir to prevent path traversal
	absUpload, _ := filepath.Abs(h.uploadDir)
	absFile, _ := filepath.Abs(destPath)
	if !strings.HasPrefix(absFile, absUpload+string(filepath.Separator)) {
		apierror.Message(c, http.StatusBadRequest, "无效的文件路径")
		return
	}

	// Overwrite the physical file
	out, err := os.Create(destPath)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "保存文件失败")
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "写入文件失败")
		return
	}

	// Re-detect image dimensions
	var width, height *int
	savedFile, err := os.Open(destPath)
	if err == nil {
		defer savedFile.Close()
		if cfg, _, decErr := image.DecodeConfig(savedFile); decErr == nil {
			w := cfg.Width
			h := cfg.Height
			width = &w
			height = &h
		}
	}

	media.Size = written
	media.Width = width
	media.Height = height

	if err := h.mediaRepo.Update(c.Request.Context(), media); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "更新记录失败")
		return
	}

	c.JSON(http.StatusOK, media)
}

// GetUsages returns a list of pages/articles that reference a media item
// @Summary      Get media usages
// @Description  Returns a list of pages and articles that reference the given media item
// @Tags         Media (Admin)
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Media ID"
// @Success      200 {object} object{usages=[]object}
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/media/{id}/usages [get]
func (h *Handler) GetUsages(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	media, err := h.mediaRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "未找到该媒体文件")
		return
	}

	usages, err := h.mediaRepo.FindUsages(c.Request.Context(), media.URL)
	if err != nil {
		apierror.Message(c, http.StatusInternalServerError, "查询引用失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"usages": usages})
}

// Rename updates the display filename of a media item
// @Summary      Rename media file
// @Description  Updates the display filename of a media item
// @Tags         Media (Admin)
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int                        true "Media ID"
// @Param        body body object{filename=string}     true "New filename"
// @Success      200 {object} object
// @Failure      400 {object} object{error=string}
// @Failure      404 {object} object{error=string}
// @Router       /admin/media/{id}/rename [put]
func (h *Handler) Rename(c *gin.Context) {
	id, ok := handlerutil.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var input struct {
		Filename string `json:"filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Message(c, http.StatusBadRequest, "请提供新的文件名")
		return
	}

	input.Filename = strings.TrimSpace(input.Filename)
	if input.Filename == "" {
		apierror.Message(c, http.StatusBadRequest, "文件名不能为空")
		return
	}

	media, err := h.mediaRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "未找到该媒体文件")
		return
	}

	media.Filename = input.Filename

	if err := h.mediaRepo.Update(c.Request.Context(), media); err != nil {
		apierror.Message(c, http.StatusInternalServerError, "更新记录失败")
		return
	}

	c.JSON(http.StatusOK, media)
}

// generateWebP converts an image file to WebP format, saving it alongside the
// original with a ".webp" extension.
func generateWebP(srcPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	destPath := strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ".webp"
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create webp file: %w", err)
	}
	defer out.Close()

	if err := webp.Encode(out, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
		return fmt.Errorf("encode webp: %w", err)
	}
	return nil
}

// generateThumbnail creates a thumbnail of the image at srcPath, resizing it so
// that its width equals maxWidth (preserving aspect ratio). The thumbnail is
// saved alongside the original with a "_thumb.webp" suffix.
func generateThumbnail(srcPath string, maxWidth int) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if origW == 0 {
		return fmt.Errorf("image has zero width")
	}

	// Only downscale; if already smaller than maxWidth keep original dimensions.
	thumbW := origW
	thumbH := origH
	if origW > maxWidth {
		thumbW = maxWidth
		thumbH = origH * maxWidth / origW
	}

	dst := image.NewRGBA(image.Rect(0, 0, thumbW, thumbH))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, stdDraw.Over, nil)

	ext := filepath.Ext(srcPath)
	base := strings.TrimSuffix(srcPath, ext)
	thumbPath := base + "_thumb.webp"

	out, err := os.Create(thumbPath)
	if err != nil {
		return fmt.Errorf("create thumb file: %w", err)
	}
	defer out.Close()

	if err := webp.Encode(out, dst, &webp.Options{Lossless: false, Quality: 75}); err != nil {
		return fmt.Errorf("encode thumb webp: %w", err)
	}
	return nil
}

// sanitizeFilename removes non-alphanumeric characters from filename
func sanitizeFilename(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
		if result.Len() >= 50 {
			break
		}
	}
	if result.Len() == 0 {
		return "file"
	}
	return result.String()
}
