package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"archive/tar"
	"compress/gzip"

	"golang.org/x/mod/semver"

	"github.com/yixian-huang/inkless/backend/pkg/config"
)

// Default hosts allowed for host self-update downloads.
var defaultSelfUpdateAllowHosts = []string{
	"github.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
	"api.github.com",
	"inkless.run",
}

// HostSelfUpdateService probes GitHub Releases and optionally applies
// versioned artifacts into INKLESS_RELEASE_ROOT (H0 + H1).
type HostSelfUpdateService struct {
	cfg        *config.Config
	appVersion string
	httpClient *http.Client
	logger     *slog.Logger

	mu         sync.Mutex
	cached     *HostUpdateProbeResult
	cachedAt   time.Time
	applyMu    sync.Mutex // single apply at a time
}

// HostUpdateStatus is GET /admin/system/update.
type HostUpdateStatus struct {
	Enabled         bool                   `json:"enabled"`
	Capable         bool                   `json:"capable"`
	BlockedReason   string                 `json:"blockedReason,omitempty"`
	CurrentVersion  string                 `json:"currentVersion"`
	Channel         string                 `json:"channel"`
	ReleaseRoot     string                 `json:"releaseRoot,omitempty"`
	SystemdUnit     string                 `json:"systemdUnit,omitempty"`
	Repo            string                 `json:"repo,omitempty"`
	Latest          *HostReleaseInfo       `json:"latest,omitempty"`
	LastCheckAt     string                 `json:"lastCheckAt,omitempty"`
	LastJob         *HostUpdateJob         `json:"lastJob,omitempty"`
	LocalVersions   []string               `json:"localVersions,omitempty"`
	HasPrevious     bool                   `json:"hasPrevious"`
	Checks          map[string]bool        `json:"checks"`
}

// HostReleaseInfo is a remote release summary.
type HostReleaseInfo struct {
	Version     string            `json:"version"`
	PublishedAt string            `json:"publishedAt,omitempty"`
	NotesURL    string            `json:"notesUrl,omitempty"`
	Newer       bool              `json:"newer"`
	Assets      []HostReleaseAsset `json:"assets,omitempty"`
	Prerelease  bool              `json:"prerelease,omitempty"`
}

// HostReleaseAsset is one downloadable file.
type HostReleaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
	Size   int64  `json:"size,omitempty"`
}

// HostUpdateProbeResult is the outcome of a check.
type HostUpdateProbeResult struct {
	CheckedAt string           `json:"checkedAt"`
	Channel   string           `json:"channel"`
	Source    string           `json:"source"` // github | manifest
	Latest    *HostReleaseInfo `json:"latest,omitempty"`
	Error     string           `json:"error,omitempty"`
}

// HostUpdateJob tracks apply/rollback.
type HostUpdateJob struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // apply | rollback
	Status      string `json:"status"` // pending|running|success|failed|pending_restart
	FromVersion string `json:"fromVersion,omitempty"`
	ToVersion   string `json:"toVersion,omitempty"`
	StartedAt   string `json:"startedAt"`
	FinishedAt  string `json:"finishedAt,omitempty"`
	Error       string `json:"error,omitempty"`
	Phase       string `json:"phase,omitempty"`
	Message     string `json:"message,omitempty"`
}

func NewHostSelfUpdateService(cfg *config.Config, appVersion string) *HostSelfUpdateService {
	return &HostSelfUpdateService{
		cfg:        cfg,
		appVersion: appVersion,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		logger:     slog.Default(),
	}
}

func (s *HostSelfUpdateService) Status(ctx context.Context) HostUpdateStatus {
	st := HostUpdateStatus{
		Enabled:        s.enabled(),
		CurrentVersion: s.appVersion,
		Channel:        s.channel(),
		ReleaseRoot:    s.releaseRoot(),
		SystemdUnit:    s.unit(),
		Repo:           s.repo(),
		Checks:         map[string]bool{},
	}

	root := s.releaseRoot()
	st.Checks["enabled"] = s.enabled()
	st.Checks["releaseRootSet"] = root != ""
	st.Checks["unitSet"] = s.unit() != ""
	if root != "" {
		st.Checks["releaseRootExists"] = dirExists(root)
		st.Checks["versionsWritable"] = canWriteDir(filepath.Join(root, "backend", "versions")) ||
			canWriteDir(filepath.Join(root, "var")) ||
			canWriteDir(root) ||
			canWriteDir(filepath.Join(root, "var", "updates"))
		st.LocalVersions = listVersionDirs(filepath.Join(root, "backend", "versions"))
		st.HasPrevious = pathExists(filepath.Join(root, "backend", "previous"))
	}
	st.Checks["systemctlPresent"] = commandExists("systemctl")

	capable, reason := s.capable(st.Checks)
	st.Capable = capable
	st.BlockedReason = reason

	if cached := s.loadCachedProbe(); cached != nil && cached.Latest != nil {
		st.Latest = cached.Latest
		st.Latest.Newer = VersionIsNewerSemver(cached.Latest.Version, s.appVersion)
		st.LastCheckAt = cached.CheckedAt
	}
	if job := s.loadLastJob(); job != nil {
		st.LastJob = job
	}
	return st
}

func (s *HostSelfUpdateService) capable(checks map[string]bool) (bool, string) {
	if !checks["enabled"] {
		return false, "self-update disabled (set INKLESS_SELF_UPDATE_ENABLED=true)"
	}
	if !checks["releaseRootSet"] {
		return false, "INKLESS_RELEASE_ROOT not set"
	}
	if !checks["releaseRootExists"] {
		return false, "release root does not exist"
	}
	if !checks["unitSet"] {
		return false, "INKLESS_SYSTEMD_UNIT not set"
	}
	if !checks["versionsWritable"] {
		return false, "release root not writable by process (expand ReadWritePaths / ownership)"
	}
	return true, ""
}

// Check refreshes remote latest (H0). Always allowed for manage users;
// does not require apply capability.
func (s *HostSelfUpdateService) Check(ctx context.Context, force bool) (*HostUpdateProbeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ttl := time.Duration(s.cfg.SelfUpdateCheckTTLSec) * time.Second
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if !force && s.cached != nil && time.Since(s.cachedAt) < ttl {
		return s.cached, nil
	}

	res, err := s.probeRemote(ctx)
	if err != nil {
		res = &HostUpdateProbeResult{
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
			Channel:   s.channel(),
			Error:     err.Error(),
		}
		s.cached = res
		s.cachedAt = time.Now()
		_ = s.persistProbe(res)
		return res, err
	}
	if res.Latest != nil {
		res.Latest.Newer = VersionIsNewerSemver(res.Latest.Version, s.appVersion)
	}
	s.cached = res
	s.cachedAt = time.Now()
	_ = s.persistProbe(res)
	return res, nil
}

// Apply downloads and activates target version (H1). version empty → latest on channel.
func (s *HostSelfUpdateService) Apply(ctx context.Context, version string) (*HostUpdateJob, error) {
	st := s.Status(ctx)
	if !st.Capable {
		return nil, fmt.Errorf("not update-capable: %s", st.BlockedReason)
	}
	if !s.applyMu.TryLock() {
		return nil, fmt.Errorf("another update job is running")
	}
	defer s.applyMu.Unlock()

	probe, err := s.Check(ctx, true)
	if err != nil {
		return nil, err
	}
	target := strings.TrimSpace(version)
	var rel *HostReleaseInfo
	if target == "" {
		if probe.Latest == nil {
			return nil, fmt.Errorf("no latest release found")
		}
		rel = probe.Latest
		target = rel.Version
	} else {
		rel, err = s.fetchReleaseByTag(ctx, target)
		if err != nil {
			return nil, err
		}
	}
	if !VersionIsNewerSemver(target, s.appVersion) && normalizeVer(target) == normalizeVer(s.appVersion) {
		return nil, fmt.Errorf("already on version %s", s.appVersion)
	}

	job := &HostUpdateJob{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:        "apply",
		Status:      "running",
		FromVersion: s.appVersion,
		ToVersion:   target,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		Phase:       "download",
	}
	_ = s.saveJob(job)

	if err := s.applyRelease(ctx, job, rel); err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveJob(job)
		return job, err
	}
	job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if job.Status == "" || job.Status == "running" {
		job.Status = "success"
	}
	_ = s.saveJob(job)
	return job, nil
}

// Rollback switches current to previous or a local versions/* directory.
func (s *HostSelfUpdateService) Rollback(ctx context.Context, to string) (*HostUpdateJob, error) {
	st := s.Status(ctx)
	if !st.Capable {
		return nil, fmt.Errorf("not update-capable: %s", st.BlockedReason)
	}
	if !s.applyMu.TryLock() {
		return nil, fmt.Errorf("another update job is running")
	}
	defer s.applyMu.Unlock()

	root := s.releaseRoot()
	to = strings.TrimSpace(to)
	if to == "" {
		to = "previous"
	}

	job := &HostUpdateJob{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:        "rollback",
		Status:      "running",
		FromVersion: s.appVersion,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		Phase:       "rollback",
	}

	var targetBackend, targetFrontend string
	if to == "previous" {
		targetBackend = filepath.Join(root, "backend", "previous")
		targetFrontend = filepath.Join(root, "frontend", "previous")
		if !pathExists(targetBackend) {
			job.Status = "failed"
			job.Error = "no backend/previous symlink"
			job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
			_ = s.saveJob(job)
			return job, fmt.Errorf("%s", job.Error)
		}
		job.ToVersion = basenameResolved(targetBackend)
	} else {
		// only local version dir name
		safe := filepath.Base(to)
		if safe != to || strings.Contains(to, "..") {
			return nil, fmt.Errorf("invalid version id")
		}
		targetBackend = filepath.Join(root, "backend", "versions", safe)
		targetFrontend = filepath.Join(root, "frontend", "versions", safe)
		if !dirExists(targetBackend) {
			return nil, fmt.Errorf("local backend version not found: %s", safe)
		}
		job.ToVersion = safe
	}

	_ = s.saveJob(job)

	// Swap: current -> becomes previous only if rolling to a versions/* path
	curB := filepath.Join(root, "backend", "current")
	curF := filepath.Join(root, "frontend", "current")
	if err := atomicSymlink(mustResolveOr(targetBackend, targetBackend), curB); err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveJob(job)
		return job, err
	}
	if dirExists(targetFrontend) || pathExists(targetFrontend) {
		_ = atomicSymlink(mustResolveOr(targetFrontend, targetFrontend), curF)
	}

	job.Phase = "restart"
	if err := s.restartUnit(ctx); err != nil {
		job.Status = "pending_restart"
		job.Message = "symlinks switched; restart failed: " + err.Error()
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveJob(job)
		return job, nil
	}
	job.Phase = "health"
	if err := s.waitHealth(ctx); err != nil {
		job.Status = "failed"
		job.Error = "health after rollback: " + err.Error()
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.saveJob(job)
		return job, fmt.Errorf("%s", job.Error)
	}
	job.Status = "success"
	job.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	_ = s.saveJob(job)
	return job, nil
}

func (s *HostSelfUpdateService) GetJob(id string) (*HostUpdateJob, error) {
	path := filepath.Join(s.jobsDir(), id+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var job HostUpdateJob
	if err := json.Unmarshal(b, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// --- config helpers ---

func (s *HostSelfUpdateService) enabled() bool {
	return s.cfg != nil && s.cfg.SelfUpdateEnabled
}
func (s *HostSelfUpdateService) releaseRoot() string {
	if s.cfg == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.SelfUpdateReleaseRoot)
}
func (s *HostSelfUpdateService) unit() string {
	if s.cfg == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.SelfUpdateSystemdUnit)
}
func (s *HostSelfUpdateService) channel() string {
	if s.cfg == nil || s.cfg.SelfUpdateChannel == "" {
		return "stable"
	}
	return s.cfg.SelfUpdateChannel
}
func (s *HostSelfUpdateService) repo() string {
	if s.cfg == nil || s.cfg.SelfUpdateRepo == "" {
		return "yixian-huang/inkless"
	}
	return s.cfg.SelfUpdateRepo
}
func (s *HostSelfUpdateService) apiBase() string {
	if s.cfg == nil || s.cfg.SelfUpdateAPIBase == "" {
		return "https://api.github.com"
	}
	return strings.TrimRight(s.cfg.SelfUpdateAPIBase, "/")
}
func (s *HostSelfUpdateService) allowHosts() []string {
	if s.cfg != nil && len(s.cfg.SelfUpdateAllowHosts) > 0 {
		return s.cfg.SelfUpdateAllowHosts
	}
	return defaultSelfUpdateAllowHosts
}

func (s *HostSelfUpdateService) updatesDir() string {
	return filepath.Join(s.releaseRoot(), "var", "updates")
}
func (s *HostSelfUpdateService) jobsDir() string {
	return filepath.Join(s.updatesDir(), "jobs")
}

// VersionIsNewerSemver reports whether remote is strictly newer than current.
func VersionIsNewerSemver(remote, current string) bool {
	r := ensureV(normalizeVer(remote))
	c := ensureV(normalizeVer(current))
	if !semver.IsValid(r) || !semver.IsValid(c) {
		// fall back to string inequality for non-semver (git describe)
		return remote != "" && normalizeVer(remote) != normalizeVer(current) && remote != current
	}
	return semver.Compare(r, c) > 0
}

func normalizeVer(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	// strip common deploy prefixes like main-sha
	return v
}

func ensureV(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// --- filesystem helpers ---

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
func pathExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
func canWriteDir(dir string) bool {
	if !dirExists(dir) {
		return false
	}
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
func listVersionDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out
}

func basenameResolved(p string) string {
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Base(p)
	}
	return filepath.Base(r)
}

func mustResolveOr(p, fallback string) string {
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return fallback
	}
	return r
}

func atomicSymlink(target, link string) error {
	// target should be absolute path for versions dir
	absTarget, err := filepath.Abs(target)
	if err != nil {
		absTarget = target
	}
	dir := filepath.Dir(link)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := link + ".tmp-" + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.Symlink(absTarget, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, link)
}

func (s *HostSelfUpdateService) saveJob(job *HostUpdateJob) error {
	if s.releaseRoot() == "" {
		return nil
	}
	dir := s.jobsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(job, "", "  ")
	path := filepath.Join(dir, job.ID+".json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	// last pointer
	_ = os.WriteFile(filepath.Join(s.updatesDir(), "last_job.json"), b, 0o644)
	return nil
}

func (s *HostSelfUpdateService) loadLastJob() *HostUpdateJob {
	if s.releaseRoot() == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(s.updatesDir(), "last_job.json"))
	if err != nil {
		return nil
	}
	var job HostUpdateJob
	if json.Unmarshal(b, &job) != nil {
		return nil
	}
	return &job
}

func (s *HostSelfUpdateService) persistProbe(res *HostUpdateProbeResult) error {
	if s.releaseRoot() == "" {
		return nil
	}
	if err := os.MkdirAll(s.updatesDir(), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(res, "", "  ")
	return os.WriteFile(filepath.Join(s.updatesDir(), "last_check.json"), b, 0o644)
}

func (s *HostSelfUpdateService) loadCachedProbe() *HostUpdateProbeResult {
	if s.cached != nil {
		return s.cached
	}
	if s.releaseRoot() == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(s.updatesDir(), "last_check.json"))
	if err != nil {
		return nil
	}
	var res HostUpdateProbeResult
	if json.Unmarshal(b, &res) != nil {
		return nil
	}
	s.cached = &res
	return &res
}

// --- HTTP allowlist ---

func (s *HostSelfUpdateService) assertAllowedURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only https downloads allowed")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if ip := net.ParseIP(host); ip != nil {
		return fmt.Errorf("IP download hosts not allowed")
	}
	ok := false
	for _, h := range s.allowHosts() {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if host == h || strings.HasSuffix(host, "."+h) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("host %q not in allowlist", host)
	}
	// path guard for inkless.run
	if host == "inkless.run" || strings.HasSuffix(host, ".inkless.run") {
		if !strings.HasPrefix(u.Path, "/releases/") && !strings.HasPrefix(u.Path, "/marketplace/") {
			// allow /releases only for host update mirror
			if !strings.Contains(u.Path, "/releases/") {
				return fmt.Errorf("inkless.run path not allowed for host update: %s", u.Path)
			}
		}
	}
	return nil
}

func (s *HostSelfUpdateService) doJSON(ctx context.Context, method, rawURL string, out any) error {
	if err := s.assertAllowedURL(rawURL); err != nil {
		// api.github.com is in allowlist
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "inkless-host-self-update")
	if s.cfg != nil && s.cfg.SelfUpdateGitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.SelfUpdateGitHubToken)
	}
	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	return json.Unmarshal(body, out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// --- probe ---

type ghRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

func (s *HostSelfUpdateService) probeRemote(ctx context.Context) (*HostUpdateProbeResult, error) {
	res := &HostUpdateProbeResult{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Channel:   s.channel(),
	}
	if s.cfg != nil && s.cfg.SelfUpdateManifestURL != "" {
		rel, err := s.probeManifest(ctx, s.cfg.SelfUpdateManifestURL)
		if err == nil && rel != nil {
			res.Source = "manifest"
			res.Latest = rel
			return res, nil
		}
		s.logger.Warn("manifest probe failed, fallback github", "err", err)
	}
	rel, err := s.probeGitHub(ctx)
	if err != nil {
		return res, err
	}
	res.Source = "github"
	res.Latest = rel
	return res, nil
}

func (s *HostSelfUpdateService) probeManifest(ctx context.Context, rawURL string) (*HostReleaseInfo, error) {
	if err := s.assertAllowedURL(rawURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "inkless-host-self-update")
	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("manifest HTTP %d", res.StatusCode)
	}
	var doc struct {
		Latest *HostReleaseInfo `json:"latest"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	if doc.Latest == nil || doc.Latest.Version == "" {
		return nil, fmt.Errorf("manifest missing latest")
	}
	return doc.Latest, nil
}

func (s *HostSelfUpdateService) probeGitHub(ctx context.Context) (*HostReleaseInfo, error) {
	ch := s.channel()
	repo := s.repo()
	if ch == "latest" {
		// list releases and pick first non-draft
		var list []ghRelease
		u := fmt.Sprintf("%s/repos/%s/releases?per_page=15", s.apiBase(), repo)
		if err := s.doJSON(ctx, http.MethodGet, u, &list); err != nil {
			return nil, err
		}
		for i := range list {
			if list[i].Draft {
				continue
			}
			return s.releaseFromGH(&list[i])
		}
		return nil, fmt.Errorf("no releases found")
	}
	// stable: /releases/latest (non-prerelease)
	var one ghRelease
	u := fmt.Sprintf("%s/repos/%s/releases/latest", s.apiBase(), repo)
	if err := s.doJSON(ctx, http.MethodGet, u, &one); err != nil {
		// fallback list filter
		var list []ghRelease
		u2 := fmt.Sprintf("%s/repos/%s/releases?per_page=20", s.apiBase(), repo)
		if err2 := s.doJSON(ctx, http.MethodGet, u2, &list); err2 != nil {
			return nil, err
		}
		for i := range list {
			if list[i].Draft || list[i].Prerelease {
				continue
			}
			return s.releaseFromGH(&list[i])
		}
		return nil, fmt.Errorf("no stable release: %w", err)
	}
	return s.releaseFromGH(&one)
}

func (s *HostSelfUpdateService) fetchReleaseByTag(ctx context.Context, tag string) (*HostReleaseInfo, error) {
	tag = strings.TrimSpace(tag)
	u := fmt.Sprintf("%s/repos/%s/releases/tags/%s", s.apiBase(), s.repo(), url.PathEscape(tag))
	var one ghRelease
	if err := s.doJSON(ctx, http.MethodGet, u, &one); err != nil {
		// try with v prefix
		if !strings.HasPrefix(tag, "v") {
			u = fmt.Sprintf("%s/repos/%s/releases/tags/%s", s.apiBase(), s.repo(), url.PathEscape("v"+tag))
			if err2 := s.doJSON(ctx, http.MethodGet, u, &one); err2 != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return s.releaseFromGH(&one)
}

func (s *HostSelfUpdateService) releaseFromGH(r *ghRelease) (*HostReleaseInfo, error) {
	if r == nil || r.TagName == "" {
		return nil, fmt.Errorf("empty release")
	}
	info := &HostReleaseInfo{
		Version:     r.TagName,
		PublishedAt: r.PublishedAt,
		NotesURL:    r.HTMLURL,
		Prerelease:  r.Prerelease,
	}
	// map assets + pair sha256 files
	byName := map[string]string{}
	shaFiles := map[string]string{}
	sizes := map[string]int64{}
	for _, a := range r.Assets {
		name := a.Name
		if strings.HasSuffix(name, ".sha256") {
			shaFiles[name] = a.BrowserDownloadURL
			continue
		}
		byName[name] = a.BrowserDownloadURL
		sizes[name] = a.Size
	}
	// expected artifact names
	ver := r.TagName
	want := []string{
		fmt.Sprintf("backend-%s.tar.gz", ver),
		fmt.Sprintf("frontend-%s.tar.gz", ver),
	}
	// also try without rewriting if tag has no v
	for _, name := range want {
		urlStr, ok := byName[name]
		if !ok {
			// try alternate
			alt := strings.Replace(name, ver, strings.TrimPrefix(ver, "v"), 1)
			urlStr, ok = byName[alt]
			name = alt
		}
		if !ok {
			continue
		}
		asset := HostReleaseAsset{Name: name, URL: urlStr, Size: sizes[name]}
		if shaURL, ok := shaFiles[name+".sha256"]; ok {
			asset.SHA256 = shaURL // temporarily store URL; resolved on download
			// We'll fetch content during apply; keep URL in a parallel convention:
			// put checksum URL as "sha256:<url>" prefix? Better: second pass.
			_ = shaURL
		}
		info.Assets = append(info.Assets, asset)
	}
	// Attach sha256 download URLs as synthetic assets is awkward; store map via re-fetch on apply.
	// Enrich SHA256 field with file content when short hex already present — for now leave empty;
	// apply will download .sha256 sibling.
	if len(info.Assets) == 0 {
		// include all tar.gz for debugging
		for name, u := range byName {
			if strings.HasSuffix(name, ".tar.gz") {
				info.Assets = append(info.Assets, HostReleaseAsset{Name: name, URL: u, Size: sizes[name]})
			}
		}
	}
	return info, nil
}

// --- apply pipeline ---

func (s *HostSelfUpdateService) applyRelease(ctx context.Context, job *HostUpdateJob, rel *HostReleaseInfo) error {
	root := s.releaseRoot()
	ver := rel.Version
	incoming := filepath.Join(s.updatesDir(), "incoming", job.ID)
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return err
	}
	defer func() {
		// keep failed incoming for debug; clean success lightly
		if job.Status == "success" {
			_ = os.RemoveAll(incoming)
		}
	}()

	backendName := fmt.Sprintf("backend-%s.tar.gz", ver)
	frontendName := fmt.Sprintf("frontend-%s.tar.gz", ver)

	// Resolve download URLs from release assets or reconstruct from GitHub
	bURL, fURL, err := s.resolveArtifactURLs(ctx, rel, backendName, frontendName)
	if err != nil {
		return err
	}

	job.Phase = "download"
	_ = s.saveJob(job)

	bPath := filepath.Join(incoming, backendName)
	fPath := filepath.Join(incoming, frontendName)
	if err := s.downloadFile(ctx, bURL, bPath); err != nil {
		return fmt.Errorf("download backend: %w", err)
	}
	if err := s.downloadFile(ctx, fURL, fPath); err != nil {
		return fmt.Errorf("download frontend: %w", err)
	}

	job.Phase = "verify"
	_ = s.saveJob(job)
	if err := s.verifySHA256(ctx, bURL, bPath); err != nil {
		return fmt.Errorf("backend checksum: %w", err)
	}
	if err := s.verifySHA256(ctx, fURL, fPath); err != nil {
		return fmt.Errorf("frontend checksum: %w", err)
	}

	job.Phase = "activate"
	_ = s.saveJob(job)

	bVerDir := filepath.Join(root, "backend", "versions", ver)
	fVerDir := filepath.Join(root, "frontend", "versions", ver)
	if err := os.MkdirAll(filepath.Dir(bVerDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fVerDir), 0o755); err != nil {
		return err
	}
	// remove partial
	_ = os.RemoveAll(bVerDir)
	_ = os.RemoveAll(fVerDir)
	if err := extractTarGz(bPath, bVerDir); err != nil {
		return fmt.Errorf("extract backend: %w", err)
	}
	if err := extractTarGz(fPath, fVerDir); err != nil {
		return fmt.Errorf("extract frontend: %w", err)
	}
	if err := ensureBackendLatestLink(bVerDir); err != nil {
		return err
	}

	// previous = current (if any)
	curB := filepath.Join(root, "backend", "current")
	curF := filepath.Join(root, "frontend", "current")
	prevB := filepath.Join(root, "backend", "previous")
	prevF := filepath.Join(root, "frontend", "previous")
	if pathExists(curB) {
		if resolved, err := filepath.EvalSymlinks(curB); err == nil {
			_ = atomicSymlink(resolved, prevB)
		}
	}
	if pathExists(curF) {
		if resolved, err := filepath.EvalSymlinks(curF); err == nil {
			_ = atomicSymlink(resolved, prevF)
		}
	}
	if err := atomicSymlink(bVerDir, curB); err != nil {
		return err
	}
	if err := atomicSymlink(fVerDir, curF); err != nil {
		return err
	}

	job.Phase = "restart"
	_ = s.saveJob(job)
	if err := s.restartUnit(ctx); err != nil {
		job.Status = "pending_restart"
		job.Message = "artifacts activated; restart failed: " + err.Error() +
			"; run: systemctl restart " + s.unit()
		// not a hard fail of code switch
		return nil
	}

	job.Phase = "health"
	_ = s.saveJob(job)
	if err := s.waitHealth(ctx); err != nil {
		return fmt.Errorf("health check failed after restart: %w (consider rollback)", err)
	}
	job.Status = "success"
	job.Message = "updated to " + ver
	return nil
}

func (s *HostSelfUpdateService) resolveArtifactURLs(ctx context.Context, rel *HostReleaseInfo, backendName, frontendName string) (string, string, error) {
	var bURL, fURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case backendName:
			bURL = a.URL
		case frontendName:
			fURL = a.URL
		}
		// alternate without v
		if bURL == "" && strings.Contains(a.Name, "backend-") && strings.HasSuffix(a.Name, ".tar.gz") && !strings.HasSuffix(a.Name, ".sha256") {
			bURL = a.URL
			backendName = a.Name
		}
		if fURL == "" && strings.Contains(a.Name, "frontend-") && strings.HasSuffix(a.Name, ".tar.gz") && !strings.HasSuffix(a.Name, ".sha256") {
			fURL = a.URL
			frontendName = a.Name
		}
	}
	if bURL == "" || fURL == "" {
		// reconstruct standard GitHub release download URLs
		ver := rel.Version
		repo := s.repo()
		if bURL == "" {
			bURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, ver, backendName)
		}
		if fURL == "" {
			fURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, ver, frontendName)
		}
	}
	_ = ctx
	if err := s.assertAllowedURL(bURL); err != nil {
		return "", "", err
	}
	if err := s.assertAllowedURL(fURL); err != nil {
		return "", "", err
	}
	return bURL, fURL, nil
}

func (s *HostSelfUpdateService) downloadFile(ctx context.Context, rawURL, dest string) error {
	if err := s.assertAllowedURL(rawURL); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "inkless-host-self-update")
	if s.cfg != nil && s.cfg.SelfUpdateGitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.SelfUpdateGitHubToken)
	}
	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d downloading %s", res.StatusCode, rawURL)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	// limit 512MB
	_, err = io.Copy(f, io.LimitReader(res.Body, 512<<20))
	return err
}

func (s *HostSelfUpdateService) verifySHA256(ctx context.Context, artifactURL, path string) error {
	sumURL := artifactURL + ".sha256"
	// Also try filename.sha256 next to asset on GitHub
	if err := s.assertAllowedURL(sumURL); err != nil {
		// try alternate construction
		return fmt.Errorf("checksum url not allowed: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sumURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "inkless-host-self-update")
	if s.cfg != nil && s.cfg.SelfUpdateGitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.SelfUpdateGitHubToken)
	}
	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("missing checksum file HTTP %d (refusing to apply without sha256)", res.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	expected := parseSHA256File(string(body))
	if expected == "" {
		return fmt.Errorf("could not parse sha256 file")
	}
	got, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, got) {
		return fmt.Errorf("sha256 mismatch: expected %s got %s", expected, got)
	}
	return nil
}

func parseSHA256File(content string) string {
	// formats: "hex  filename" or just hex
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return ""
	}
	hexSum := fields[0]
	if len(hexSum) != 64 {
		return ""
	}
	for _, c := range hexSum {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return ""
		}
	}
	return strings.ToLower(hexSum)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractTarGz(src, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := hdr.Name
		if name == "" || strings.Contains(name, "..") {
			return fmt.Errorf("refusing unsafe tar path: %s", name)
		}
		// strip absolute
		name = strings.TrimPrefix(name, "/")
		target := filepath.Join(dest, name)
		// ensure under dest
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), filepath.Clean(dest)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(dest) {
			return fmt.Errorf("tar path escapes dest: %s", name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0o200)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, io.LimitReader(tr, 512<<20)); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			// skip symlinks in untrusted tars for safety
			continue
		default:
			continue
		}
	}
	return nil
}

func ensureBackendLatestLink(verDir string) error {
	latest := filepath.Join(verDir, "inkless-api-latest")
	if pathExists(latest) {
		_ = os.Chmod(latest, 0o755)
		// if it's a file, ok; if symlink ok
		return nil
	}
	entries, err := os.ReadDir(verDir)
	if err != nil {
		return err
	}
	var bin string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "inkless-api-") && name != "inkless-api-latest" && !e.IsDir() {
			bin = name
			break
		}
	}
	if bin == "" {
		return fmt.Errorf("no inkless-api binary in %s", verDir)
	}
	_ = os.Chmod(filepath.Join(verDir, bin), 0o755)
	return os.Symlink(bin, latest)
}

func (s *HostSelfUpdateService) restartUnit(ctx context.Context) error {
	unit := s.unit()
	if unit == "" {
		return fmt.Errorf("no systemd unit configured")
	}
	// only allow simple unit names
	if strings.ContainsAny(unit, "/\\ \t\n") || strings.Contains(unit, "..") {
		return fmt.Errorf("invalid unit name")
	}
	cmd := exec.CommandContext(ctx, "systemctl", "restart", unit)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, truncate(string(out), 300))
	}
	return nil
}

func (s *HostSelfUpdateService) waitHealth(ctx context.Context) error {
	urlStr := ""
	if s.cfg != nil {
		urlStr = strings.TrimSpace(s.cfg.SelfUpdateHealthURL)
	}
	if urlStr == "" {
		port := 8088
		if s.cfg != nil && s.cfg.Port > 0 {
			port = s.cfg.Port
		}
		urlStr = fmt.Sprintf("http://127.0.0.1:%d/health", port)
	}
	// only allow loopback health
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("health URL must be loopback")
	}
	timeout := 60 * time.Second
	if s.cfg != nil && s.cfg.SelfUpdateHealthTimeout > 0 {
		timeout = time.Duration(s.cfg.SelfUpdateHealthTimeout) * time.Second
	}
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 3 * time.Second}
	var last error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		res, err := client.Do(req)
		if err == nil {
			res.Body.Close()
			if res.StatusCode >= 200 && res.StatusCode < 300 {
				return nil
			}
			last = fmt.Errorf("status %d", res.StatusCode)
		} else {
			last = err
		}
		time.Sleep(1500 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("timeout")
	}
	return last
}
