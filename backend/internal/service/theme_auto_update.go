package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
)

const (
	themeAutoUpdateConfigKey = "themeAutoUpdate"
	defaultAutoUpdateInterval = 60 // minutes
	minAutoUpdateInterval     = 15
	maxAutoUpdateInterval     = 24 * 60
)

// ThemeAutoUpdateSettings is stored under site_configs.system.publishedConfig.themeAutoUpdate.
type ThemeAutoUpdateSettings struct {
	Enabled          bool   `json:"enabled"`
	IntervalMinutes  int    `json:"intervalMinutes"`
	OnlyMarketplace  bool   `json:"onlyMarketplace"`  // default true: only source=marketplace|external with URL
	IncludeActive    bool   `json:"includeActive"`    // update active theme package pointer
	OnlyActive       bool   `json:"onlyActive"`       // if true, only check the active theme
	LastCheckAt      string `json:"lastCheckAt,omitempty"`
	LastApplyAt      string `json:"lastApplyAt,omitempty"`
	LastError        string `json:"lastError,omitempty"`
	LastReport       *ThemeAutoUpdateReport `json:"lastReport,omitempty"`
}

// ThemeAutoUpdateReport is one run summary.
type ThemeAutoUpdateReport struct {
	CheckedAt string   `json:"checkedAt"`
	CatalogSource string `json:"catalogSource,omitempty"`
	Checked   int      `json:"checked"`
	Updated   []ThemeAutoUpdateItem `json:"updated"`
	Skipped   []ThemeAutoUpdateItem `json:"skipped"`
	Errors    []ThemeAutoUpdateItem `json:"errors"`
}

// ThemeAutoUpdateItem is one theme outcome.
type ThemeAutoUpdateItem struct {
	ThemeID string `json:"themeId"`
	Slug    string `json:"slug,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ThemeAutoUpdateService polls the official catalog and optionally upgrades installed themes.
type ThemeAutoUpdateService struct {
	installer *OfficialThemeInstaller
	siteCfg   repository.SiteConfigRepository
	themeRepo repository.InstalledThemeRepository
	logger    *slog.Logger

	mu        sync.Mutex
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

func NewThemeAutoUpdateService(
	installer *OfficialThemeInstaller,
	siteCfg repository.SiteConfigRepository,
	themeRepo repository.InstalledThemeRepository,
) *ThemeAutoUpdateService {
	return &ThemeAutoUpdateService{
		installer: installer,
		siteCfg:   siteCfg,
		themeRepo: themeRepo,
		logger:    slog.Default(),
		done:      make(chan struct{}),
	}
}

// Start begins the background loop (safe to call once).
func (s *ThemeAutoUpdateService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.loop()
	})
}

// Stop stops the background loop.
func (s *ThemeAutoUpdateService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()
}

func (s *ThemeAutoUpdateService) loop() {
	defer s.wg.Done()
	// Stagger first check slightly after boot.
	timer := time.NewTimer(45 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-timer.C:
			s.tick()
			settings, _ := s.GetSettings(context.Background())
			mins := settings.IntervalMinutes
			if mins < minAutoUpdateInterval {
				mins = defaultAutoUpdateInterval
			}
			timer.Reset(time.Duration(mins) * time.Minute)
		}
	}
}

func (s *ThemeAutoUpdateService) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	settings, err := s.GetSettings(ctx)
	if err != nil || !settings.Enabled {
		return
	}
	if _, err := s.Run(ctx, false); err != nil {
		s.logger.Warn("theme auto-update run failed", "err", err)
	}
}

// GetSettings loads auto-update settings (defaults when missing).
func (s *ThemeAutoUpdateService) GetSettings(ctx context.Context) (ThemeAutoUpdateSettings, error) {
	defaults := ThemeAutoUpdateSettings{
		Enabled:         false,
		IntervalMinutes: defaultAutoUpdateInterval,
		OnlyMarketplace: true,
		IncludeActive:   true,
		OnlyActive:      false,
	}
	if s == nil || s.siteCfg == nil {
		return defaults, nil
	}
	sc, err := s.siteCfg.FindByKey(ctx, model.SiteConfigKeySystem)
	if err != nil || sc == nil || sc.PublishedConfig == nil {
		return defaults, nil
	}
	raw, ok := sc.PublishedConfig[themeAutoUpdateConfigKey]
	if !ok || raw == nil {
		return defaults, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return defaults, nil
	}
	var cfg ThemeAutoUpdateSettings
	if err := json.Unmarshal(b, &cfg); err != nil {
		return defaults, nil
	}
	if cfg.IntervalMinutes < minAutoUpdateInterval {
		cfg.IntervalMinutes = defaultAutoUpdateInterval
	}
	if cfg.IntervalMinutes > maxAutoUpdateInterval {
		cfg.IntervalMinutes = maxAutoUpdateInterval
	}
	// Defaults for bools when key absent in stored JSON.
	if !jsonHasKey(b, "onlyMarketplace") {
		cfg.OnlyMarketplace = true
	}
	if !jsonHasKey(b, "includeActive") {
		cfg.IncludeActive = true
	}
	return cfg, nil
}

// Defaults returns factory defaults (for API docs / UI).
func DefaultThemeAutoUpdateSettings() ThemeAutoUpdateSettings {
	return ThemeAutoUpdateSettings{
		Enabled:         false,
		IntervalMinutes: defaultAutoUpdateInterval,
		OnlyMarketplace: true,
		IncludeActive:   true,
		OnlyActive:      false,
	}
}

func jsonHasKey(b []byte, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// SaveSettings persists settings into site_configs.system published+draft.
func (s *ThemeAutoUpdateService) SaveSettings(ctx context.Context, in ThemeAutoUpdateSettings) (ThemeAutoUpdateSettings, error) {
	if s == nil || s.siteCfg == nil {
		return in, fmt.Errorf("site config repository not configured")
	}
	if in.IntervalMinutes < minAutoUpdateInterval {
		in.IntervalMinutes = minAutoUpdateInterval
	}
	if in.IntervalMinutes > maxAutoUpdateInterval {
		in.IntervalMinutes = maxAutoUpdateInterval
	}

	// Preserve last run metadata if caller didn't send it.
	prev, _ := s.GetSettings(ctx)
	if in.LastCheckAt == "" {
		in.LastCheckAt = prev.LastCheckAt
	}
	if in.LastApplyAt == "" {
		in.LastApplyAt = prev.LastApplyAt
	}
	if in.LastReport == nil {
		in.LastReport = prev.LastReport
	}
	if in.LastError == "" && !in.Enabled {
		// keep last error when disabling
		in.LastError = prev.LastError
	}

	sc, err := s.siteCfg.FindByKey(ctx, model.SiteConfigKeySystem)
	if err != nil || sc == nil || sc.ID == 0 {
		// Create system row
		pub := model.JSONMap{themeAutoUpdateConfigKey: settingsToMap(in)}
		sc = &model.SiteConfig{
			Key:              model.SiteConfigKeySystem,
			DraftConfig:      pub,
			DraftVersion:     1,
			PublishedConfig:  pub,
			PublishedVersion: 1,
		}
		if err := s.siteCfg.Upsert(ctx, sc); err != nil {
			return in, err
		}
		return in, nil
	}

	if sc.PublishedConfig == nil {
		sc.PublishedConfig = model.JSONMap{}
	}
	if sc.DraftConfig == nil {
		sc.DraftConfig = model.JSONMap{}
	}
	sc.PublishedConfig[themeAutoUpdateConfigKey] = settingsToMap(in)
	sc.DraftConfig[themeAutoUpdateConfigKey] = settingsToMap(in)
	if err := s.siteCfg.Update(ctx, sc); err != nil {
		return in, err
	}
	return in, nil
}

func settingsToMap(in ThemeAutoUpdateSettings) map[string]any {
	b, _ := json.Marshal(in)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// Run checks catalog and applies updates when enabled or force=true.
// force=true runs even if settings.Enabled is false (manual "check now" / "apply now").
// apply=false would be check-only — for simplicity Run always applies when updates found;
// use dryRun for check-only.
func (s *ThemeAutoUpdateService) Run(ctx context.Context, force bool) (*ThemeAutoUpdateReport, error) {
	return s.run(ctx, force, true)
}

// Check only reports available updates without writing.
func (s *ThemeAutoUpdateService) Check(ctx context.Context) (*ThemeAutoUpdateReport, error) {
	return s.run(ctx, true, false)
}

func (s *ThemeAutoUpdateService) run(ctx context.Context, force, apply bool) (*ThemeAutoUpdateReport, error) {
	if s == nil || s.installer == nil || s.themeRepo == nil {
		return nil, fmt.Errorf("auto-update service not configured")
	}

	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !force && !settings.Enabled {
		return nil, fmt.Errorf("theme auto-update is disabled")
	}

	now := time.Now().UTC()
	report := &ThemeAutoUpdateReport{
		CheckedAt: now.Format(time.RFC3339),
		Updated:   []ThemeAutoUpdateItem{},
		Skipped:   []ThemeAutoUpdateItem{},
		Errors:    []ThemeAutoUpdateItem{},
	}

	// Always refresh catalog on auto-update so we see newly published themes without host redeploy.
	loadRes, err := s.installer.Loader().Load(ctx, true)
	if err != nil || loadRes == nil || loadRes.Catalog == nil {
		msg := "catalog load failed"
		if err != nil {
			msg = err.Error()
		}
		settings.LastCheckAt = report.CheckedAt
		settings.LastError = msg
		_, _ = s.SaveSettings(ctx, settings)
		return report, fmt.Errorf("%s", msg)
	}
	report.CatalogSource = string(loadRes.Source)

	installed, err := s.themeRepo.List(ctx)
	if err != nil {
		return report, err
	}

	for _, row := range installed {
		if row == nil {
			continue
		}
		report.Checked++
		item := ThemeAutoUpdateItem{ThemeID: row.ThemeID, From: row.Version}

		if settings.OnlyActive && !row.IsActive {
			item.Reason = "not active"
			report.Skipped = append(report.Skipped, item)
			continue
		}
		if settings.OnlyMarketplace {
			src := strings.ToLower(strings.TrimSpace(row.Source))
			if src != "marketplace" && src != "external" {
				item.Reason = "source is " + row.Source
				report.Skipped = append(report.Skipped, item)
				continue
			}
			if strings.TrimSpace(row.ExternalURL) == "" {
				item.Reason = "no externalUrl"
				report.Skipped = append(report.Skipped, item)
				continue
			}
		}
		if row.IsActive && !settings.IncludeActive {
			item.Reason = "active theme excluded by settings"
			report.Skipped = append(report.Skipped, item)
			continue
		}

		entry, ok := loadRes.Catalog.FindBySlug(row.ThemeID)
		if !ok {
			// try match by themeId field
			for i := range loadRes.Catalog.Themes {
				if loadRes.Catalog.Themes[i].ThemeID == row.ThemeID {
					entry = &loadRes.Catalog.Themes[i]
					ok = true
					break
				}
			}
		}
		if !ok || entry == nil {
			item.Reason = "not in catalog"
			report.Skipped = append(report.Skipped, item)
			continue
		}
		item.Slug = entry.Slug

		if !themecatalog.VersionIsNewer(entry.Latest.Version, row.Version) {
			item.Reason = "already up to date"
			item.To = entry.Latest.Version
			report.Skipped = append(report.Skipped, item)
			continue
		}

		if !apply {
			item.To = entry.Latest.Version
			item.Reason = "update available"
			report.Updated = append(report.Updated, item) // listed as would-update
			continue
		}

		res, iErr := s.installer.Install(ctx, ThemeInstallRequest{
			Slug:     entry.Slug,
			Version:  entry.Latest.Version,
			Activate: false, // never change which theme is active; only package pointer
		})
		if iErr != nil {
			item.Reason = iErr.Error()
			item.To = entry.Latest.Version
			report.Errors = append(report.Errors, item)
			continue
		}
		item.To = res.ToVer
		if res.ToVer == "" && res.Theme != nil {
			item.To = res.Theme.Version
		}
		item.Reason = "updated"
		report.Updated = append(report.Updated, item)
	}

	settings.LastCheckAt = report.CheckedAt
	settings.LastError = ""
	if apply && len(report.Updated) > 0 {
		settings.LastApplyAt = report.CheckedAt
	}
	settings.LastReport = report
	if _, sErr := s.SaveSettings(ctx, settings); sErr != nil {
		s.logger.Warn("persist theme auto-update report failed", "err", sErr)
	}
	return report, nil
}

// Export VersionIsNewer for auto-update package tests via thin wrapper if needed —
// themecatalog has unexported versionIsNewer; add exported alias there.
