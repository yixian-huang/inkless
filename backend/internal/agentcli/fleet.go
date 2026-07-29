// Package agentcli implements local multi-site fleet resolution and Admin API
// client helpers for the inkless CLI (content agents).
package agentcli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fleet is the multi-site registry (docs/agent-fleet.schema.json).
type Fleet struct {
	Version     int                  `json:"version"`
	DefaultSite string               `json:"default_site"`
	Sites       map[string]SiteEntry `json:"sites"`
}

// SiteEntry is one Inkless instance profile.
type SiteEntry struct {
	Label           string   `json:"label"`
	BaseURL         string   `json:"base_url"`
	APIKeyEnv       string   `json:"api_key_env"`
	APIKeyFile      string   `json:"api_key_file"`
	ScopesExpected  []string `json:"scopes_expected"`
	PublishPolicy   string   `json:"publish_policy"`
	Brand           string   `json:"brand"`
	LocaleDefault   string   `json:"locale_default"`
	Notes           string   `json:"notes"`
	Verify          *Verify  `json:"verify"`
}

// Verify controls pre-write probes.
type Verify struct {
	Whoami     *bool  `json:"whoami"`
	PublicPath string `json:"public_path"`
}

// Endpoint is a resolved remote target for Admin API calls.
type Endpoint struct {
	SiteID        string
	Label         string
	BaseURL       string
	APIKey        string
	PublishPolicy string
	VerifyWhoami  bool
	ScopesExpect  []string
}

// ResolveOptions selects how to resolve an endpoint.
type ResolveOptions struct {
	FleetPath string // empty → search defaults
	SiteID    string // empty → default_site or single site
	BaseURL   string // single-site override
	APIKey    string // single-site override
}

// DefaultFleetPaths returns candidate fleet registry paths in search order.
func DefaultFleetPaths() []string {
	var paths []string
	if v := strings.TrimSpace(os.Getenv("INKLESS_FLEET")); v != "" {
		paths = append(paths, v)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".config", "inkless", "fleet.json"))
	}
	paths = append(paths, filepath.Join(".inkless", "fleet.json"))
	return paths
}

// LoadFleet reads and lightly validates a fleet registry JSON file.
func LoadFleet(path string) (*Fleet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fleet %s: %w", path, err)
	}
	var f Fleet
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse fleet %s: %w", path, err)
	}
	if f.Version != 0 && f.Version != 1 {
		return nil, fmt.Errorf("unsupported fleet version %d (want 1)", f.Version)
	}
	if f.Version == 0 {
		f.Version = 1
	}
	if len(f.Sites) == 0 {
		return nil, fmt.Errorf("fleet %s has no sites", path)
	}
	for id, site := range f.Sites {
		if strings.TrimSpace(site.BaseURL) == "" {
			return nil, fmt.Errorf("site %q: base_url is required", id)
		}
		if strings.TrimSpace(site.APIKeyEnv) == "" && strings.TrimSpace(site.APIKeyFile) == "" {
			return nil, fmt.Errorf("site %q: api_key_env or api_key_file is required", id)
		}
	}
	return &f, nil
}

// FindFleetPath returns the first existing fleet path from opts or defaults.
func FindFleetPath(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		p := strings.TrimSpace(explicit)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("fleet not found: %s", p)
		}
		return p, nil
	}
	for _, p := range DefaultFleetPaths() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("no fleet registry found (tried --fleet, INKLESS_FLEET, ~/.config/inkless/fleet.json, ./.inkless/fleet.json)")
}

// NormalizeBaseURL trims space and trailing slashes.
func NormalizeBaseURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

// ResolveEndpoint resolves a single remote endpoint from fleet and/or env overrides.
//
// Priority:
//  1. Fleet when --site / INKLESS_SITE / --fleet / INKLESS_FLEET is set
//  2. Single-site when BaseURL + APIKey are set (flags or INKLESS_BASE_URL / INKLESS_API_KEY)
//  3. Fleet default_site / sole site if a registry is discoverable
func ResolveEndpoint(opts ResolveOptions) (*Endpoint, error) {
	base := NormalizeBaseURL(firstNonEmpty(opts.BaseURL, os.Getenv("INKLESS_BASE_URL")))
	key := strings.TrimSpace(firstNonEmpty(opts.APIKey, os.Getenv("INKLESS_API_KEY")))
	siteID := strings.TrimSpace(firstNonEmpty(opts.SiteID, os.Getenv("INKLESS_SITE")))
	fleetExplicit := strings.TrimSpace(opts.FleetPath) != "" || strings.TrimSpace(os.Getenv("INKLESS_FLEET")) != ""
	wantFleet := siteID != "" || fleetExplicit

	if !wantFleet && base != "" && key != "" {
		return &Endpoint{
			SiteID:        "default",
			Label:         "single-site",
			BaseURL:       base,
			APIKey:        key,
			PublishPolicy: "manual",
			VerifyWhoami:  true,
		}, nil
	}

	fleetPath, ferr := FindFleetPath(opts.FleetPath)
	if ferr != nil {
		if base != "" && key != "" {
			return &Endpoint{
				SiteID:        firstNonEmpty(siteID, "default"),
				Label:         "single-site",
				BaseURL:       base,
				APIKey:        key,
				PublishPolicy: "manual",
				VerifyWhoami:  true,
			}, nil
		}
		return nil, ferr
	}

	fleet, err := LoadFleet(fleetPath)
	if err != nil {
		return nil, err
	}

	if siteID == "" {
		siteID = strings.TrimSpace(fleet.DefaultSite)
	}
	if siteID == "" && len(fleet.Sites) == 1 {
		for id := range fleet.Sites {
			siteID = id
		}
	}
	if siteID == "" {
		return nil, fmt.Errorf("site id required: pass --site, set INKLESS_SITE, or set default_site in %s", fleetPath)
	}

	entry, ok := fleet.Sites[siteID]
	if !ok {
		return nil, fmt.Errorf("site %q not found in fleet %s (known: %s)", siteID, fleetPath, joinSiteIDs(fleet))
	}

	resolvedKey, err := loadAPIKey(entry)
	if err != nil {
		return nil, fmt.Errorf("site %q: %w", siteID, err)
	}
	// Flag/env key overrides fleet for this site if provided together with site id
	if key != "" {
		resolvedKey = key
	}
	resolvedBase := NormalizeBaseURL(entry.BaseURL)
	if base != "" {
		resolvedBase = base
	}

	policy := strings.TrimSpace(entry.PublishPolicy)
	if policy == "" {
		policy = "manual"
	}
	verifyWhoami := true
	if entry.Verify != nil && entry.Verify.Whoami != nil {
		verifyWhoami = *entry.Verify.Whoami
	}

	return &Endpoint{
		SiteID:        siteID,
		Label:         firstNonEmpty(entry.Label, siteID),
		BaseURL:       resolvedBase,
		APIKey:        resolvedKey,
		PublishPolicy: policy,
		VerifyWhoami:  verifyWhoami,
		ScopesExpect:  append([]string(nil), entry.ScopesExpected...),
	}, nil
}

// ListSites returns fleet sites without secrets (for `site list`).
func ListSites(fleetPath string) (path string, fleet *Fleet, err error) {
	path, err = FindFleetPath(fleetPath)
	if err != nil {
		return "", nil, err
	}
	fleet, err = LoadFleet(path)
	return path, fleet, err
}

func loadAPIKey(entry SiteEntry) (string, error) {
	if env := strings.TrimSpace(entry.APIKeyEnv); env != "" {
		v := strings.TrimSpace(os.Getenv(env))
		if v == "" {
			return "", fmt.Errorf("environment variable %s is empty (api_key_env)", env)
		}
		return v, nil
	}
	path := strings.TrimSpace(entry.APIKeyFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read api_key_file %s: %w", path, err)
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", fmt.Errorf("api_key_file %s is empty", path)
	}
	// First line only
	if i := strings.IndexByte(v, '\n'); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v, nil
}

func joinSiteIDs(f *Fleet) string {
	ids := make([]string, 0, len(f.Sites))
	for id := range f.Sites {
		ids = append(ids, id)
	}
	return strings.Join(ids, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// MaskSecret shows a short prefix for display.
func MaskSecret(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 12 {
		return s[:min(4, len(s))] + "…"
	}
	return s[:12] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
