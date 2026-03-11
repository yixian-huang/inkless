package media

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"blotting-consultancy/internal/model"
	"blotting-consultancy/internal/repository"
)

// Handler handles media-related HTTP requests
type Handler struct {
	mediaRepo repository.MediaRepository
	uploadDir string
	baseURL   string
}

// NewHandler creates a new media handler
func NewHandler(mediaRepo repository.MediaRepository, uploadDir string, baseURL string) *Handler {
	return &Handler{
		mediaRepo: mediaRepo,
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

// Upload handles file upload via multipart form
func (h *Handler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "请选择要上传的文件"}})
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

	if !strings.HasPrefix(mimeType, "image/") && !strings.HasPrefix(mimeType, "video/") && !strings.HasPrefix(mimeType, "audio/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "仅支持上传图片、视频或音频文件"}})
		return
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "创建上传目录失败"}})
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
		default:
			ext = ".jpg"
		}
	}
	uniqueName := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), sanitizeFilename(strings.TrimSuffix(header.Filename, ext)), ext)
	destPath := filepath.Join(h.uploadDir, uniqueName)

	// Save file
	out, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "保存文件失败"}})
		return
	}
	defer out.Close()

	// Reset file reader position
	if seeker, ok := file.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	written, err := io.Copy(out, file)
	if err != nil {
		os.Remove(destPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "写入文件失败"}})
		return
	}

	// Try to get image dimensions (only for image files)
	var width, height *int
	if strings.HasPrefix(mimeType, "image/") {
		savedFile, err := os.Open(destPath)
		if err == nil {
			defer savedFile.Close()
			if cfg, _, err := image.DecodeConfig(savedFile); err == nil {
				w := cfg.Width
				h := cfg.Height
				width = &w
				height = &h
			}
		}
	}

	// Build URL
	url := h.baseURL + "/uploads/" + uniqueName

	// Save to database
	media := &model.Media{
		URL:      url,
		Filename: header.Filename,
		MimeType: mimeType,
		Size:     written,
		Width:    width,
		Height:   height,
	}

	if err := h.mediaRepo.Create(c.Request.Context(), media); err != nil {
		os.Remove(destPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "保存记录失败"}})
		return
	}

	c.JSON(http.StatusCreated, media)
}

// List returns a paginated list of media items
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Optional MIME type prefix filter (e.g. ?type=image, ?type=video, ?type=audio)
	mimePrefix := ""
	if typeParam := c.Query("type"); typeParam != "" {
		mimePrefix = typeParam + "/"
	}

	items, total, err := h.mediaRepo.List(c.Request.Context(), offset, pageSize, mimePrefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Delete removes a media item and its file
func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	// Find the media record
	media, err := h.mediaRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "未找到该媒体文件"}})
		return
	}

	// Extract filename from URL to delete the physical file
	parts := strings.Split(media.URL, "/uploads/")
	if len(parts) == 2 {
		filePath := filepath.Join(h.uploadDir, parts[1])
		os.Remove(filePath) // Best effort; ignore error
	}

	// Delete database record
	if err := h.mediaRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "删除记录失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// Recrop replaces the physical file for an existing media item with a re-cropped version
func (h *Handler) Recrop(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	media, err := h.mediaRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "未找到该媒体文件"}})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "请选择要上传的文件"}})
		return
	}
	defer file.Close()

	// Resolve physical file path from URL
	parts := strings.Split(media.URL, "/uploads/")
	if len(parts) != 2 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "无法解析文件路径"}})
		return
	}
	destPath := filepath.Join(h.uploadDir, parts[1])

	// Overwrite the physical file
	out, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "保存文件失败"}})
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "写入文件失败"}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "更新记录失败"}})
		return
	}

	c.JSON(http.StatusOK, media)
}

// GetUsages returns a list of pages/articles that reference a media item
func (h *Handler) GetUsages(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	media, err := h.mediaRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "未找到该媒体文件"}})
		return
	}

	usages, err := h.mediaRepo.FindUsages(c.Request.Context(), media.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "查询引用失败"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"usages": usages})
}

// Rename updates the display filename of a media item
func (h *Handler) Rename(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "无效的 ID"}})
		return
	}

	var input struct {
		Filename string `json:"filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "请提供新的文件名"}})
		return
	}

	input.Filename = strings.TrimSpace(input.Filename)
	if input.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "文件名不能为空"}})
		return
	}

	media, err := h.mediaRepo.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "未找到该媒体文件"}})
		return
	}

	media.Filename = input.Filename

	if err := h.mediaRepo.Update(c.Request.Context(), media); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "更新记录失败"}})
		return
	}

	c.JSON(http.StatusOK, media)
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
