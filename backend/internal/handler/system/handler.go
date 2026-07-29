package system

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"

	"github.com/yixian-huang/inkless/backend/internal/service"
	"github.com/yixian-huang/inkless/backend/pkg/apierror"
)

const statusProbeTimeout = 2 * time.Second

// Handler handles system status HTTP requests
type Handler struct {
	db         *gorm.DB
	uploadDir  string
	appVersion string
	startTime  time.Time
	selfUpdate *service.HostSelfUpdateService
}

// NewHandler creates a new system status handler
func NewHandler(db *gorm.DB, uploadDir string, appVersion string) *Handler {
	return &Handler{
		db:         db,
		uploadDir:  uploadDir,
		appVersion: appVersion,
		startTime:  time.Now(),
	}
}

// WithSelfUpdate wires optional host self-update (H0/H1).
func (h *Handler) WithSelfUpdate(svc *service.HostSelfUpdateService) *Handler {
	h.selfUpdate = svc
	return h
}

// ApplicationInfo contains build metadata safe to expose in admin status.
type ApplicationInfo struct {
	Version             string `json:"version"`
	UpdateCapable       bool   `json:"updateCapable"`
	UpdateBlockedReason string `json:"updateBlockedReason,omitempty"`
	SelfUpdateEnabled   bool   `json:"selfUpdateEnabled"`
}

// RuntimeInfo contains Go runtime information
type RuntimeInfo struct {
	GoVersion  string `json:"goVersion"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	CPUCount   int    `json:"cpuCount"`
	Goroutines int    `json:"goroutines"`
	UptimeSec  int64  `json:"uptime"`
}

// MemoryInfo contains memory usage information
type MemoryInfo struct {
	AllocMB      float64 `json:"allocMB"`
	TotalAllocMB float64 `json:"totalAllocMB"`
	SysMB        float64 `json:"sysMB"`
	GCPauseMs    float64 `json:"gcPauseMs"`
}

// DatabaseInfo contains database status information
type DatabaseInfo struct {
	Type               string `json:"type"`
	Healthy            bool   `json:"healthy"`
	Status             string `json:"status"`
	Error              string `json:"error,omitempty"`
	OpenConnections    int    `json:"openConnections"`
	MaxOpenConnections int    `json:"maxOpenConnections"`
	InUse              int    `json:"inUse"`
	Idle               int    `json:"idle"`
}

// StorageInfo contains storage usage information
type StorageInfo struct {
	Type            string  `json:"type"`
	Healthy         bool    `json:"healthy"`
	Status          string  `json:"status"`
	Error           string  `json:"error,omitempty"`
	UploadDirSizeMB float64 `json:"uploadDirSizeMB"`
	UploadDirBytes  int64   `json:"uploadDirBytes"`
	MediaCount      int64   `json:"mediaCount"`
}

// ContentCounts contains content statistics
type ContentCounts struct {
	Articles int64 `json:"articles"`
	Pages    int64 `json:"pages"`
	Media    int64 `json:"media"`
	Users    int64 `json:"users"`
}

// StatusResponse is the full system status response
type StatusResponse struct {
	Application ApplicationInfo `json:"application"`
	Runtime     RuntimeInfo     `json:"runtime"`
	Memory      MemoryInfo      `json:"memory"`
	Database    DatabaseInfo    `json:"database"`
	Storage     StorageInfo     `json:"storage"`
	Content     ContentCounts   `json:"content"`
}

// GetStatus returns comprehensive system status.
func (h *Handler) GetStatus(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Runtime info
	ri := RuntimeInfo{
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPUCount:   runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
		UptimeSec:  int64(time.Since(h.startTime).Seconds()),
	}

	// Memory info
	mi := MemoryInfo{
		AllocMB:      float64(m.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(m.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(m.Sys) / 1024 / 1024,
	}
	if m.NumGC > 0 {
		mi.GCPauseMs = float64(m.PauseNs[(m.NumGC+255)%256]) / 1e6
	}

	// Database info
	di := DatabaseInfo{
		Type:    h.db.Dialector.Name(),
		Healthy: true,
		Status:  "healthy",
	}
	databaseCtx, cancelDatabase := context.WithTimeout(c.Request.Context(), statusProbeTimeout)
	defer cancelDatabase()
	storageCtx, cancelStorage := context.WithTimeout(c.Request.Context(), statusProbeTimeout)
	defer cancelStorage()
	storageResult := make(chan StorageInfo, 1)
	go func() {
		storageResult <- localStorageInfo(storageCtx, h.uploadDir)
	}()

	if sqlDB, err := h.db.DB(); err == nil {
		stats := sqlDB.Stats()
		di.OpenConnections = stats.OpenConnections
		di.MaxOpenConnections = stats.MaxOpenConnections
		di.InUse = stats.InUse
		di.Idle = stats.Idle
		if err := sqlDB.PingContext(databaseCtx); err != nil {
			di.Healthy = false
			di.Status = "unhealthy"
			di.Error = "database ping failed"
		}
	} else {
		di.Healthy = false
		di.Status = "unhealthy"
		di.Error = "database connection unavailable"
	}

	// Content counts
	cc := ContentCounts{}
	countQueries := []struct {
		table string
		value *int64
	}{
		{table: "articles", value: &cc.Articles},
		{table: "unified_pages", value: &cc.Pages},
		{table: "media", value: &cc.Media},
		{table: "users", value: &cc.Users},
	}
	for _, query := range countQueries {
		if err := h.db.WithContext(databaseCtx).Table(query.table).Count(query.value).Error; err != nil {
			di.Healthy = false
			di.Status = "unhealthy"
			if di.Error == "" {
				di.Error = "database status query failed"
			}
		}
	}
	si := <-storageResult
	si.MediaCount = cc.Media

	appInfo := ApplicationInfo{Version: h.appVersion}
	if h.selfUpdate != nil {
		st := h.selfUpdate.Status(c.Request.Context())
		appInfo.UpdateCapable = st.Capable
		appInfo.UpdateBlockedReason = st.BlockedReason
		appInfo.SelfUpdateEnabled = st.Enabled
	}

	c.JSON(http.StatusOK, StatusResponse{
		Application: appInfo,
		Runtime:     ri,
		Memory:      mi,
		Database:    di,
		Storage:     si,
		Content:     cc,
	})
}

// GetUpdate returns host self-update status (H0).
// @Router       /admin/system/update [get]
func (h *Handler) GetUpdate(c *gin.Context) {
	if h.selfUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自更新服务未配置")
		return
	}
	c.JSON(http.StatusOK, h.selfUpdate.Status(c.Request.Context()))
}

// CheckUpdate forces a remote release probe (H0).
// @Router       /admin/system/update/check [post]
func (h *Handler) CheckUpdate(c *gin.Context) {
	if h.selfUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自更新服务未配置")
		return
	}
	res, err := h.selfUpdate.Check(c.Request.Context(), true)
	if err != nil {
		// still return partial probe when possible
		if res != nil {
			c.JSON(http.StatusOK, res)
			return
		}
		apierror.Message(c, http.StatusBadGateway, "检查更新失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, res)
}

type applyUpdateInput struct {
	Version string `json:"version"`
}

// ApplyUpdate applies a host release to this instance (H1).
// @Router       /admin/system/update/apply [post]
func (h *Handler) ApplyUpdate(c *gin.Context) {
	if h.selfUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自更新服务未配置")
		return
	}
	var input applyUpdateInput
	_ = c.ShouldBindJSON(&input)
	job, err := h.selfUpdate.Apply(c.Request.Context(), strings.TrimSpace(input.Version))
	if err != nil {
		msg := err.Error()
		if job != nil {
			c.JSON(http.StatusBadRequest, job)
			return
		}
		apierror.Message(c, http.StatusBadRequest, msg)
		return
	}
	c.JSON(http.StatusOK, job)
}

// GetUpdateJob returns a persisted job.
// @Router       /admin/system/update/jobs/{id} [get]
func (h *Handler) GetUpdateJob(c *gin.Context) {
	if h.selfUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自更新服务未配置")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		apierror.Message(c, http.StatusBadRequest, "缺少 job id")
		return
	}
	job, err := h.selfUpdate.GetJob(id)
	if err != nil {
		apierror.Message(c, http.StatusNotFound, "任务不存在")
		return
	}
	c.JSON(http.StatusOK, job)
}

type rollbackInput struct {
	To string `json:"to"` // previous | local version id
}

// RollbackUpdate rolls back to previous or a local versions/* tree.
// @Router       /admin/system/update/rollback [post]
func (h *Handler) RollbackUpdate(c *gin.Context) {
	if h.selfUpdate == nil {
		apierror.Message(c, http.StatusInternalServerError, "自更新服务未配置")
		return
	}
	var input rollbackInput
	_ = c.ShouldBindJSON(&input)
	job, err := h.selfUpdate.Rollback(c.Request.Context(), strings.TrimSpace(input.To))
	if err != nil {
		if job != nil {
			c.JSON(http.StatusBadRequest, job)
			return
		}
		apierror.Message(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, job)
}

func localStorageInfo(ctx context.Context, dir string) StorageInfo {
	bytes, err := dirSizeBytes(ctx, dir)
	if err != nil {
		return StorageInfo{
			Type:    "local",
			Healthy: false,
			Status:  "unhealthy",
			Error:   storageErrorMessage(err),
		}
	}
	return StorageInfo{
		Type:            "local",
		Healthy:         true,
		Status:          "healthy",
		UploadDirBytes:  bytes,
		UploadDirSizeMB: float64(bytes) / 1024 / 1024,
	}
}

// dirSizeMB calculates the total size of a directory in MB.
func dirSizeMB(dir string) float64 {
	bytes, err := dirSizeBytes(context.Background(), dir)
	if err != nil {
		return 0
	}
	return float64(bytes) / 1024 / 1024
}

func dirSizeBytes(ctx context.Context, dir string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalSize, nil
}

func storageErrorMessage(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "upload directory scan timed out"
	}
	if os.IsNotExist(err) {
		return "upload directory not found"
	}
	if os.IsPermission(err) {
		return "upload directory is not accessible"
	}
	return "upload directory scan failed"
}
