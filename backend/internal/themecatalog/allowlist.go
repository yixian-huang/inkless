package themecatalog

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// DefaultUMDAllowHosts are hostnames permitted for official theme UMD URLs
// when INKLESS_THEME_UMD_ALLOW_HOSTS is unset.
//
// Keep aligned with docs/design-official-extension-store-phase-a.md §7.
var DefaultUMDAllowHosts = []string{
	"github.com",
	"raw.githubusercontent.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
	"cdn.jsdelivr.net",
	"inkless.run",
	"www.inkless.run",
}

// ParseAllowHosts splits a comma-separated host list; empty input → defaults.
func ParseAllowHosts(csv string) []string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return append([]string(nil), DefaultUMDAllowHosts...)
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		h := normalizeHost(p)
		if h != "" {
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), DefaultUMDAllowHosts...)
	}
	return out
}

func normalizeHost(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	if i := strings.Index(h, "/"); i >= 0 {
		h = h[:i]
	}
	if i := strings.LastIndex(h, "@"); i >= 0 {
		h = h[i+1:]
	}
	return strings.TrimSpace(h)
}

// ValidateUMDURL checks HTTPS + allowlisted host. Empty url is invalid unless
// the caller skips for builtinOnly themes.
func ValidateUMDURL(raw string, allowHosts []string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("umdUrl is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid umdUrl: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("umdUrl must use https scheme, got %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("umdUrl missing host")
	}
	// Block literal IPs (SSRF / non-CDN) even if somehow allowlisted.
	if ip := net.ParseIP(host); ip != nil {
		return fmt.Errorf("umdUrl host must not be an IP address")
	}
	if len(allowHosts) == 0 {
		allowHosts = DefaultUMDAllowHosts
	}
	if !hostAllowed(host, allowHosts) {
		return fmt.Errorf("umdUrl host %q is not on the allowlist", host)
	}
	return nil
}

func hostAllowed(host string, allowHosts []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, a := range allowHosts {
		a = normalizeHost(a)
		if a == "" {
			continue
		}
		if host == a {
			return true
		}
		// Allow subdomains of an allowlisted registrable-ish host:
		// e.g. allow "githubusercontent.com" would match "objects.githubusercontent.com"
		// We only do suffix match when allow entry has a dot (avoid matching "com").
		if strings.Contains(a, ".") && strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}

// ValidateEntryInstallable checks Phase A install preconditions for a catalog row
// at a resolved version: official, umd allowlist (unless builtinOnly).
func ValidateEntryInstallable(entry *ThemeEntry, ver *ThemeVersion, allowHosts []string) error {
	if entry == nil {
		return fmt.Errorf("theme entry is nil")
	}
	if !entry.Official {
		return fmt.Errorf("theme %q is not official (Phase A only installs official themes)", entry.Slug)
	}
	if entry.BuiltinOnly {
		return nil
	}
	if ver == nil {
		return fmt.Errorf("version is nil")
	}
	return ValidateUMDURL(ver.UMDURL, allowHosts)
}
