package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gorm.io/gorm"
)

// Status represents the health status of a component.
const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
)

// CheckResult holds the result of a single health check.
type CheckResult struct {
	Status  string                 `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NamedCheck pairs a check function with its name.
type NamedCheck struct {
	Name  string
	Check func(ctx context.Context) CheckResult
}

// HealthResponse is the full health check response.
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
}

// Checker runs a set of named health checks.
type Checker struct {
	checks  []NamedCheck
	version string
}

// NewChecker creates a new health checker with the given version string.
func NewChecker(version string) *Checker {
	return &Checker{version: version}
}

// AddCheck registers a named health check.
func (h *Checker) AddCheck(name string, check func(ctx context.Context) CheckResult) {
	h.checks = append(h.checks, NamedCheck{Name: name, Check: check})
}

// RunAll executes all registered checks and returns the aggregated response.
func (h *Checker) RunAll(ctx context.Context) HealthResponse {
	results := make(map[string]CheckResult)
	overallStatus := StatusHealthy

	for _, nc := range h.checks {
		result := nc.Check(ctx)
		results[nc.Name] = result

		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.Status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
	}

	return HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
		Checks:    results,
	}
}

// DatabaseCheck returns a health check function for database connectivity.
func DatabaseCheck(db *gorm.DB) func(ctx context.Context) CheckResult {
	return func(ctx context.Context) CheckResult {
		start := time.Now()
		sqlDB, err := db.DB()
		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Details: map[string]interface{}{"error": err.Error()},
			}
		}

		if err := sqlDB.PingContext(ctx); err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Details: map[string]interface{}{"error": "ping failed: " + err.Error()},
			}
		}

		latency := time.Since(start)
		details := map[string]interface{}{
			"type":      db.Dialector.Name(),
			"latencyMs": latency.Milliseconds(),
		}

		// Query DB version
		var version string
		switch db.Dialector.Name() {
		case "sqlite":
			db.Raw("SELECT sqlite_version()").Scan(&version)
		case "postgres":
			db.Raw("SHOW server_version").Scan(&version)
		}
		if version != "" {
			details["version"] = version
		}

		return CheckResult{
			Status:  StatusHealthy,
			Details: details,
		}
	}
}

// StorageCheck returns a health check function for upload directory writability.
func StorageCheck(uploadDir string) func(ctx context.Context) CheckResult {
	return func(_ context.Context) CheckResult {
		details := map[string]interface{}{
			"provider": "local",
		}

		// Check if directory exists
		info, err := os.Stat(uploadDir)
		if err != nil {
			if os.IsNotExist(err) {
				// Try to create it
				if mkErr := os.MkdirAll(uploadDir, 0755); mkErr != nil {
					details["error"] = "directory does not exist and cannot be created"
					return CheckResult{Status: StatusUnhealthy, Details: details}
				}
				details["writable"] = true
				return CheckResult{Status: StatusHealthy, Details: details}
			}
			details["error"] = err.Error()
			return CheckResult{Status: StatusUnhealthy, Details: details}
		}

		if !info.IsDir() {
			details["error"] = "upload path is not a directory"
			return CheckResult{Status: StatusUnhealthy, Details: details}
		}

		// Try writing a temp file
		tmpFile := filepath.Join(uploadDir, ".health-check-tmp")
		if err := os.WriteFile(tmpFile, []byte("ok"), 0644); err != nil {
			details["writable"] = false
			return CheckResult{Status: StatusDegraded, Details: details}
		}
		os.Remove(tmpFile)

		details["writable"] = true
		return CheckResult{Status: StatusHealthy, Details: details}
	}
}

// MemoryCheck returns a health check function for Go heap memory usage.
func MemoryCheck(thresholdMB float64) func(ctx context.Context) CheckResult {
	return func(_ context.Context) CheckResult {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		usedMB := float64(m.Alloc) / 1024 / 1024

		status := StatusHealthy
		if usedMB > thresholdMB {
			status = StatusDegraded
		}

		return CheckResult{
			Status: status,
			Details: map[string]interface{}{
				"usedMB":      fmt.Sprintf("%.1f", usedMB),
				"thresholdMB": thresholdMB,
			},
		}
	}
}

// SchedulerCheck returns a health check function for the backup scheduler.
type SchedulerInfo interface {
	IsRunning() bool
	NextRun() time.Time
	Enabled() bool
}

func SchedulerCheckFunc(s SchedulerInfo) func(ctx context.Context) CheckResult {
	return func(_ context.Context) CheckResult {
		details := map[string]interface{}{
			"schedulerEnabled": s.Enabled(),
			"schedulerRunning": s.IsRunning(),
		}

		if s.Enabled() && !s.IsRunning() {
			return CheckResult{Status: StatusDegraded, Details: details}
		}

		next := s.NextRun()
		if !next.IsZero() {
			details["nextBackupAt"] = next.Format(time.RFC3339)
		}

		return CheckResult{Status: StatusHealthy, Details: details}
	}
}
