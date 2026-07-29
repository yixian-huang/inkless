package themecatalog

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// InstallState is the catalog row lifecycle for the admin theme market UI.
// See docs/design-official-extension-store-phase-a.md §4.
type InstallState string

const (
	InstallStateNotInstalled   InstallState = "not_installed"
	InstallStateInstalled      InstallState = "installed"
	InstallStateActive         InstallState = "active"
	InstallStateBuiltin        InstallState = "builtin"
	InstallStateUpdateAvailable InstallState = "update_available"
	InstallStateIncompatible   InstallState = "incompatible"
)

// HostSupportedContracts is the default contract majors the host accepts (lockstep with frontend THEME_CONTRACT_SUPPORTED).
var HostSupportedContracts = []string{"1"}

// InstalledRef is a minimal view of installed_themes for state merge (avoids model import).
type InstalledRef struct {
	ThemeID     string
	Version     string
	Source      string // built-in | external | marketplace
	ExternalURL string
	IsActive    bool
}

// MergeOptions configures install-state evaluation.
type MergeOptions struct {
	// BuiltinIDs are theme IDs registered via host registerBuiltIn.
	BuiltinIDs map[string]struct{}
	// SupportedContracts lists accepted ThemePlugin contract majors (default HostSupportedContracts).
	SupportedContracts []string
	// HostVersion is the running Inkless version (e.g. git describe / "0.1.0-alpha.2"). Empty = skip minHost hard fail.
	HostVersion string
	// AllowHosts for UMD URL checks (default DefaultUMDAllowHosts).
	AllowHosts []string
}

// CatalogItemStatus is one catalog entry plus instance install state (API DTO core).
type CatalogItemStatus struct {
	Entry              ThemeEntry   `json:"entry"`
	InstallState       InstallState `json:"installState"`
	InstalledVersion   string       `json:"installedVersion,omitempty"`
	InstalledSource    string       `json:"installedSource,omitempty"`
	IncompatibleReason string       `json:"incompatibleReason,omitempty"`
	// UpdateAvailable is true when the instance has the theme and catalog latest is newer.
	// Also reflected as InstallStateUpdateAvailable when not active; when active, state stays active.
	UpdateAvailable bool `json:"updateAvailable"`
}

// MergeInstallState evaluates a single catalog entry against instance + host facts.
func MergeInstallState(entry ThemeEntry, installed *InstalledRef, opts MergeOptions) CatalogItemStatus {
	out := CatalogItemStatus{Entry: entry}
	if installed != nil {
		out.InstalledVersion = installed.Version
		out.InstalledSource = installed.Source
	}

	allow := opts.AllowHosts
	if len(allow) == 0 {
		allow = DefaultUMDAllowHosts
	}
	contracts := opts.SupportedContracts
	if len(contracts) == 0 {
		contracts = HostSupportedContracts
	}
	builtin := opts.BuiltinIDs
	if builtin == nil {
		builtin = map[string]struct{}{}
	}

	if reason := incompatibleReason(entry, opts.HostVersion, contracts, allow); reason != "" {
		out.InstallState = InstallStateIncompatible
		out.IncompatibleReason = reason
		// Still surface update info if installed, but install/update actions should be disabled.
		if installed != nil && versionIsNewer(entry.Latest.Version, installed.Version) {
			out.UpdateAvailable = true
		}
		return out
	}

	isHostBuiltin := isBuiltinThemeID(entry.ThemeID, builtin)

	if installed == nil {
		if entry.BuiltinOnly || isHostBuiltin {
			out.InstallState = InstallStateBuiltin
			return out
		}
		out.InstallState = InstallStateNotInstalled
		return out
	}

	// Has install row.
	marketplaceOverride := isMarketplaceLike(installed) && strings.TrimSpace(installed.ExternalURL) != ""
	update := versionIsNewer(entry.Latest.Version, installed.Version) &&
		(marketplaceOverride || (!entry.BuiltinOnly && strings.TrimSpace(entry.Latest.UMDURL) != ""))
	out.UpdateAvailable = update

	if installed.IsActive {
		out.InstallState = InstallStateActive
		return out
	}

	if update {
		out.InstallState = InstallStateUpdateAvailable
		return out
	}

	// Pure built-in (no external marketplace URL): surface as builtin for UI.
	if !marketplaceOverride && (entry.BuiltinOnly || isHostBuiltin || strings.EqualFold(installed.Source, "built-in") || installed.Source == "builtin") {
		out.InstallState = InstallStateBuiltin
		return out
	}

	out.InstallState = InstallStateInstalled
	return out
}

// MergeCatalogStatuses maps every catalog theme through MergeInstallState.
func MergeCatalogStatuses(cat *Catalog, installed []InstalledRef, opts MergeOptions) []CatalogItemStatus {
	if cat == nil {
		return nil
	}
	byID := make(map[string]*InstalledRef, len(installed))
	for i := range installed {
		id := strings.ToLower(strings.TrimSpace(installed[i].ThemeID))
		if id == "" {
			continue
		}
		// Prefer active row if duplicates (should not happen).
		if prev, ok := byID[id]; ok && prev.IsActive && !installed[i].IsActive {
			continue
		}
		ref := installed[i]
		byID[id] = &ref
	}

	out := make([]CatalogItemStatus, 0, len(cat.Themes))
	for _, entry := range cat.Themes {
		var ref *InstalledRef
		if r, ok := byID[strings.ToLower(strings.TrimSpace(entry.ThemeID))]; ok {
			ref = r
		}
		out = append(out, MergeInstallState(entry, ref, opts))
	}
	return out
}

// BuiltinIDSet builds a set from theme ID strings (case preserved; lookup is case-insensitive in merge).
func BuiltinIDSet(ids ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			m[id] = struct{}{}
		}
	}
	return m
}

func isMarketplaceLike(inst *InstalledRef) bool {
	if inst == nil {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(inst.Source))
	return s == "marketplace" || s == "external"
}

func isBuiltinThemeID(themeID string, builtin map[string]struct{}) bool {
	if len(builtin) == 0 {
		return false
	}
	if _, ok := builtin[themeID]; ok {
		return true
	}
	for id := range builtin {
		if strings.EqualFold(id, themeID) {
			return true
		}
	}
	return false
}

func incompatibleReason(entry ThemeEntry, hostVersion string, contracts []string, allowHosts []string) string {
	if !entry.Official {
		return "theme is not marked official (Phase A only allows official themes)"
	}
	cv := strings.TrimSpace(entry.ContractVersion)
	if cv == "" {
		return "missing contractVersion"
	}
	if !contractSupported(cv, contracts) {
		return fmt.Sprintf("contractVersion %q is not supported by this host", cv)
	}

	if min := strings.TrimSpace(entry.MinHostVersion); min != "" && strings.TrimSpace(hostVersion) != "" {
		if ok, known := hostMeetsMin(hostVersion, min); known && !ok {
			return fmt.Sprintf("requires host >= %s (running %s)", min, hostVersion)
		}
	}

	if entry.BuiltinOnly {
		return ""
	}
	// Installability of catalog latest UMD
	if err := ValidateUMDURL(entry.Latest.UMDURL, allowHosts); err != nil {
		return err.Error()
	}
	return ""
}

func contractSupported(version string, supported []string) bool {
	v := strings.TrimSpace(version)
	for _, s := range supported {
		if s == v {
			return true
		}
	}
	return false
}

// hostMeetsMin returns whether hostVersion >= minVersion.
// known=false means comparison was skipped (unparseable) — caller should not hard-fail.
func hostMeetsMin(hostVersion, minVersion string) (ok bool, known bool) {
	h := normalizeSemver(hostVersion)
	m := normalizeSemver(minVersion)
	if !semver.IsValid(h) || !semver.IsValid(m) {
		return true, false
	}
	return semver.Compare(h, m) >= 0, true
}

// versionIsNewer reports whether catalogVer is strictly newer than installedVer.
func versionIsNewer(catalogVer, installedVer string) bool {
	catalogVer = strings.TrimSpace(catalogVer)
	installedVer = strings.TrimSpace(installedVer)
	if catalogVer == "" || installedVer == "" {
		return false
	}
	if catalogVer == installedVer {
		return false
	}
	c := normalizeSemver(catalogVer)
	i := normalizeSemver(installedVer)
	if semver.IsValid(c) && semver.IsValid(i) {
		return semver.Compare(c, i) > 0
	}
	// Fallback: different non-semver strings → treat as update available if not equal.
	return catalogVer != installedVer
}

func normalizeSemver(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	// golang.org/x/mod/semver expects leading "v".
	if !strings.HasPrefix(v, "v") && !strings.HasPrefix(v, "V") {
		v = "v" + v
	}
	// Strip build metadata quirks; leave pre-release as-is if present.
	return v
}
