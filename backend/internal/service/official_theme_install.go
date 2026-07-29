package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/yixian-huang/inkless/backend/internal/cache"
	"github.com/yixian-huang/inkless/backend/internal/model"
	"github.com/yixian-huang/inkless/backend/internal/repository"
	"github.com/yixian-huang/inkless/backend/internal/themecatalog"
)

// ThemeInstallRequest installs/updates one official catalog theme.
type ThemeInstallRequest struct {
	Slug     string
	Version  string // empty / latest → catalog latest
	Activate bool
}

// ThemeInstallResult is the outcome of a catalog install/upsert.
type ThemeInstallResult struct {
	Theme     *model.InstalledTheme
	Created   bool
	Activated bool
	Warning   string
	FromVer   string
	ToVer     string
	Updated   bool // true when existing row version/url changed
}

// OfficialThemeInstaller installs themes from the official catalog into installed_themes.
type OfficialThemeInstaller struct {
	loader           *themecatalog.Loader
	themeRepo        repository.InstalledThemeRepository
	themePageService *ThemePageService
	publicCache      *cache.Cache
	hostVersion      string
	allowHosts       []string
}

func NewOfficialThemeInstaller(
	loader *themecatalog.Loader,
	themeRepo repository.InstalledThemeRepository,
	hostVersion string,
	allowHosts []string,
) *OfficialThemeInstaller {
	if loader == nil {
		loader = themecatalog.NewLoader("")
	}
	return &OfficialThemeInstaller{
		loader:      loader,
		themeRepo:   themeRepo,
		hostVersion: strings.TrimSpace(hostVersion),
		allowHosts:  allowHosts,
	}
}

func (s *OfficialThemeInstaller) WithActivation(themePage *ThemePageService, publicCache *cache.Cache) *OfficialThemeInstaller {
	if s == nil {
		return nil
	}
	s.themePageService = themePage
	s.publicCache = publicCache
	return s
}

func (s *OfficialThemeInstaller) Loader() *themecatalog.Loader {
	if s == nil {
		return themecatalog.NewLoader("")
	}
	return s.loader
}

func (s *OfficialThemeInstaller) HostVersion() string {
	if s == nil {
		return ""
	}
	return s.hostVersion
}

func (s *OfficialThemeInstaller) AllowHosts() []string {
	if s == nil {
		return themecatalog.DefaultUMDAllowHosts
	}
	if len(s.allowHosts) == 0 {
		return themecatalog.DefaultUMDAllowHosts
	}
	return s.allowHosts
}

// ListInstalledRefs maps installed_themes rows for catalog installState merge.
func (s *OfficialThemeInstaller) ListInstalledRefs(ctx context.Context) ([]themecatalog.InstalledRef, error) {
	if s == nil || s.themeRepo == nil {
		return nil, nil
	}
	themes, err := s.themeRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]themecatalog.InstalledRef, 0, len(themes))
	for _, t := range themes {
		if t == nil {
			continue
		}
		out = append(out, themecatalog.InstalledRef{
			ThemeID:     t.ThemeID,
			Version:     t.Version,
			Source:      t.Source,
			ExternalURL: t.ExternalURL,
			IsActive:    t.IsActive,
		})
	}
	return out, nil
}

// Install validates catalog entry and upserts installed_themes.
func (s *OfficialThemeInstaller) Install(ctx context.Context, req ThemeInstallRequest) (*ThemeInstallResult, error) {
	if s == nil || s.themeRepo == nil {
		return nil, errors.New("theme installer not configured")
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		return nil, fmt.Errorf("slug 不能为空")
	}

	loadRes, err := s.loader.Load(ctx, false)
	if err != nil || loadRes == nil || loadRes.Catalog == nil {
		return nil, fmt.Errorf("加载主题目录失败")
	}
	entry, ok := loadRes.Catalog.FindBySlug(slug)
	if !ok {
		return nil, errThemeNotInCatalog
	}

	ver, err := entry.ResolveVersion(req.Version)
	if err != nil {
		return nil, err
	}
	if reason := installBlockReason(entry, ver, s.hostVersion, s.AllowHosts()); reason != "" {
		return nil, fmt.Errorf("%s", reason)
	}
	if strings.TrimSpace(ver.SHA256) != "" && strings.TrimSpace(ver.UMDURL) != "" {
		if vErr := themecatalog.VerifyUMDSHA256(ctx, ver.UMDURL, ver.SHA256, s.AllowHosts()); vErr != nil {
			return nil, fmt.Errorf("主题包校验失败: %w", vErr)
		}
	}

	fromVer := ""
	if existing, eErr := s.themeRepo.FindByThemeID(ctx, entry.ThemeID); eErr == nil && existing != nil {
		fromVer = existing.Version
	}

	theme, created, err := s.upsert(ctx, entry, ver)
	if err != nil {
		return nil, err
	}

	res := &ThemeInstallResult{
		Theme:   theme,
		Created: created,
		FromVer: fromVer,
		ToVer:   ver.Version,
		Updated: !created && fromVer != ver.Version,
	}

	if req.Activate {
		if actErr := s.activate(ctx, theme.ThemeID); actErr != nil {
			res.Warning = actErr.Error()
		} else {
			res.Activated = true
			if refreshed, findErr := s.themeRepo.FindByThemeID(ctx, theme.ThemeID); findErr == nil && refreshed != nil {
				res.Theme = refreshed
			}
		}
	}
	return res, nil
}

var errThemeNotInCatalog = errors.New("主题不在官方目录中")

// IsNotInCatalog reports catalog miss for HTTP 404 mapping.
func IsNotInCatalog(err error) bool {
	return errors.Is(err, errThemeNotInCatalog)
}

func (s *OfficialThemeInstaller) upsert(
	ctx context.Context,
	entry *themecatalog.ThemeEntry,
	ver *themecatalog.ThemeVersion,
) (*model.InstalledTheme, bool, error) {
	existing, err := s.themeRepo.FindByThemeID(ctx, entry.ThemeID)
	notFound := err != nil && strings.Contains(err.Error(), "not found")
	if err != nil && !notFound {
		return nil, false, err
	}
	if notFound {
		existing = nil
	}

	source, externalURL := installSourceAndURL(entry, ver, existing)

	if existing == nil {
		theme := &model.InstalledTheme{
			ThemeID:     entry.ThemeID,
			Name:        entry.Name,
			NameZh:      entry.NameZh,
			Description: pickDescription(entry),
			Author:      entry.Author,
			Version:     ver.Version,
			Source:      source,
			ExternalURL: externalURL,
			Preview:     entry.PreviewURL,
			IsActive:    false,
			Config:      model.JSONMap{},
		}
		if err := s.themeRepo.Create(ctx, theme); err != nil {
			return nil, false, err
		}
		return theme, true, nil
	}

	existing.Name = entry.Name
	if strings.TrimSpace(entry.NameZh) != "" {
		existing.NameZh = entry.NameZh
	}
	existing.Description = pickDescription(entry)
	if strings.TrimSpace(entry.Author) != "" {
		existing.Author = entry.Author
	}
	existing.Version = ver.Version
	existing.Source = source
	existing.ExternalURL = externalURL
	if strings.TrimSpace(entry.PreviewURL) != "" {
		existing.Preview = entry.PreviewURL
	}
	if err := s.themeRepo.Update(ctx, existing); err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (s *OfficialThemeInstaller) activate(ctx context.Context, themeID string) error {
	if err := s.themeRepo.SetActive(ctx, themeID); err != nil {
		return err
	}
	cache.InvalidateThemeOrSiteConfig(s.publicCache)
	if s.themePageService != nil {
		if err := s.themePageService.SeedThemePages(ctx, themeID); err != nil {
			log.Printf("Warning: seed theme pages after marketplace install: %v", err)
			return errors.New("主题已激活，但页面同步失败: " + err.Error())
		}
	}
	return nil
}

func installSourceAndURL(
	entry *themecatalog.ThemeEntry,
	ver *themecatalog.ThemeVersion,
	existing *model.InstalledTheme,
) (source, externalURL string) {
	if entry.BuiltinOnly || strings.TrimSpace(ver.UMDURL) == "" {
		if existing != nil && (existing.Source == "built-in" || existing.Source == "builtin") {
			return existing.Source, ""
		}
		return "built-in", ""
	}
	return "marketplace", strings.TrimSpace(ver.UMDURL)
}

func pickDescription(entry *themecatalog.ThemeEntry) string {
	if entry == nil {
		return ""
	}
	if strings.TrimSpace(entry.Description) != "" {
		return entry.Description
	}
	return entry.DescriptionZh
}

func installBlockReason(
	entry *themecatalog.ThemeEntry,
	ver *themecatalog.ThemeVersion,
	hostVersion string,
	allowHosts []string,
) string {
	if entry == nil || ver == nil {
		return "无效的主题条目"
	}
	if !entry.Official {
		return "仅支持安装官方主题"
	}
	st := themecatalog.MergeInstallState(*entry, nil, themecatalog.MergeOptions{
		SupportedContracts: themecatalog.HostSupportedContracts,
		HostVersion:        hostVersion,
		AllowHosts:         allowHosts,
	})
	if st.InstallState == themecatalog.InstallStateIncompatible {
		if st.IncompatibleReason != "" {
			return st.IncompatibleReason
		}
		return "主题与当前实例不兼容"
	}
	if err := themecatalog.ValidateEntryInstallable(entry, ver, allowHosts); err != nil {
		return err.Error()
	}
	return ""
}
