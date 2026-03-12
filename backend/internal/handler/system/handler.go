package system

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

// Handler handles system status HTTP requests
type Handler struct {
	db        *gorm.DB
	uploadDir string
	startTime time.Time
}

// NewHandler creates a new system status handler
func NewHandler(db *gorm.DB, uploadDir string) *Handler {
	return &Handler{
		db:        db,
		uploadDir: uploadDir,
		startTime: time.Now(),
	}
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
	OpenConnections    int    `json:"openConnections"`
	MaxOpenConnections int    `json:"maxOpenConnections"`
	InUse              int    `json:"inUse"`
	Idle               int    `json:"idle"`
}

// StorageInfo contains storage usage information
type StorageInfo struct {
	UploadDirSizeMB float64 `json:"uploadDirSizeMB"`
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
	Runtime  RuntimeInfo   `json:"runtime"`
	Memory   MemoryInfo    `json:"memory"`
	Database DatabaseInfo  `json:"database"`
	Storage  StorageInfo   `json:"storage"`
	Content  ContentCounts `json:"content"`
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
		Type: h.db.Dialector.Name(),
	}
	if sqlDB, err := h.db.DB(); err == nil {
		stats := sqlDB.Stats()
		di.OpenConnections = stats.OpenConnections
		di.MaxOpenConnections = stats.MaxOpenConnections
		di.InUse = stats.InUse
		di.Idle = stats.Idle
	}

	// Storage info
	si := StorageInfo{}
	si.UploadDirSizeMB = dirSizeMB(h.uploadDir)
	h.db.Table("media").Count(&si.MediaCount)

	// Content counts
	cc := ContentCounts{}
	h.db.Table("articles").Count(&cc.Articles)
	h.db.Table("pages").Count(&cc.Pages)
	cc.Media = si.MediaCount
	h.db.Table("users").Count(&cc.Users)

	c.JSON(http.StatusOK, StatusResponse{
		Runtime:  ri,
		Memory:   mi,
		Database: di,
		Storage:  si,
		Content:  cc,
	})
}

// dirSizeMB calculates the total size of a directory in MB.
func dirSizeMB(dir string) float64 {
	var totalSize int64
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return float64(totalSize) / 1024 / 1024
}
