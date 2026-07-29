package themecatalog

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultCacheTTL is the in-memory cache lifetime for a successful remote fetch.
	DefaultCacheTTL = 10 * time.Minute
	// DefaultFetchTimeout bounds a single remote catalog GET.
	DefaultFetchTimeout = 8 * time.Second
	// DefaultMaxBodyBytes rejects oversized catalog payloads.
	DefaultMaxBodyBytes = 2 << 20 // 2 MiB
)

// Loader loads the official theme catalog from a remote URL with TTL cache,
// falling back to the embedded catalog when the remote is unset or fails.
type Loader struct {
	remoteURL  string
	client     *http.Client
	ttl        time.Duration
	maxBody    int64
	userAgent string

	mu        sync.Mutex
	cached    *Catalog
	cachedAt  time.Time
	lastError string
}

// LoaderOption configures a Loader.
type LoaderOption func(*Loader)

// WithHTTPClient overrides the HTTP client (tests).
func WithHTTPClient(c *http.Client) LoaderOption {
	return func(l *Loader) {
		if c != nil {
			l.client = c
		}
	}
}

// WithCacheTTL overrides cache TTL (must be > 0).
func WithCacheTTL(d time.Duration) LoaderOption {
	return func(l *Loader) {
		if d > 0 {
			l.ttl = d
		}
	}
}

// WithMaxBodyBytes overrides max response body size.
func WithMaxBodyBytes(n int64) LoaderOption {
	return func(l *Loader) {
		if n > 0 {
			l.maxBody = n
		}
	}
}

// NewLoader creates a catalog loader. remoteURL may be empty (embedded only).
func NewLoader(remoteURL string, opts ...LoaderOption) *Loader {
	l := &Loader{
		remoteURL: strings.TrimSpace(remoteURL),
		client: &http.Client{
			Timeout: DefaultFetchTimeout,
		},
		ttl:        DefaultCacheTTL,
		maxBody:    DefaultMaxBodyBytes,
		userAgent:  "inkless-theme-catalog/1",
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoadResult is the catalog plus provenance for API responses.
type LoadResult struct {
	Catalog   *Catalog
	Source    Source
	FetchedAt time.Time
	// Warning is set when remote was requested but embedded/cache was used instead.
	Warning string
}

// LastError returns the last remote fetch error string (empty if none).
func (l *Loader) LastError() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastError
}

// Load returns a catalog. refresh bypasses a still-fresh cache and re-fetches
// remote when configured. On remote failure, returns embedded catalog with
// SourceEmbedded (or cache if available and refresh did not produce a newer result).
//
// Only returns an error if even the embedded catalog cannot be parsed.
func (l *Loader) Load(ctx context.Context, refresh bool) (*LoadResult, error) {
	if l == nil {
		cat, err := LoadEmbedded()
		if err != nil {
			return nil, err
		}
		return &LoadResult{Catalog: cat, Source: SourceEmbedded, FetchedAt: time.Now()}, nil
	}

	// No remote configured → always embedded.
	if l.remoteURL == "" {
		cat, err := LoadEmbedded()
		if err != nil {
			return nil, err
		}
		return &LoadResult{Catalog: cat, Source: SourceEmbedded, FetchedAt: time.Now()}, nil
	}

	now := time.Now()
	if !refresh {
		if cat, ok := l.getFreshCache(now); ok {
			return &LoadResult{Catalog: cat, Source: SourceCache, FetchedAt: l.cachedAt}, nil
		}
	}

	cat, err := l.fetchRemote(ctx)
	if err == nil {
		l.setCache(cat, now)
		return &LoadResult{Catalog: cat, Source: SourceRemote, FetchedAt: now}, nil
	}

	l.mu.Lock()
	l.lastError = err.Error()
	// Prefer stale cache over embedded when we have one (refresh failed).
	stale := l.cached
	staleAt := l.cachedAt
	l.mu.Unlock()

	if stale != nil {
		return &LoadResult{
			Catalog:   stale,
			Source:    SourceCache,
			FetchedAt: staleAt,
			Warning:   fmt.Sprintf("remote catalog fetch failed, using cached catalog: %v", err),
		}, nil
	}

	embedded, embErr := LoadEmbedded()
	if embErr != nil {
		return nil, fmt.Errorf("remote catalog failed (%v) and embedded fallback failed: %w", err, embErr)
	}
	return &LoadResult{
		Catalog:   embedded,
		Source:    SourceEmbedded,
		FetchedAt: now,
		Warning:   fmt.Sprintf("remote catalog fetch failed, using embedded catalog: %v", err),
	}, nil
}

func (l *Loader) getFreshCache(now time.Time) (*Catalog, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cached == nil {
		return nil, false
	}
	if now.Sub(l.cachedAt) > l.ttl {
		return nil, false
	}
	return l.cached, true
}

func (l *Loader) setCache(cat *Catalog, at time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cached = cat
	l.cachedAt = at
	l.lastError = ""
}

func (l *Loader) fetchRemote(ctx context.Context) (*Catalog, error) {
	if err := validateCatalogURL(l.remoteURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.remoteURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", l.userAgent)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog GET status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, l.maxBody+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read catalog body: %w", err)
	}
	if int64(len(raw)) > l.maxBody {
		return nil, fmt.Errorf("catalog body exceeds %d bytes", l.maxBody)
	}

	return ParseCatalog(raw)
}

// validateCatalogURL enforces HTTPS and blocks IP / loopback / link-local hosts (SSRF baseline).
func validateCatalogURL(raw string) error {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid catalog URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("catalog URL must use https, got %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("catalog URL missing host")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("catalog URL host must not be localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("catalog URL must not target private/link-local IP")
		}
		// Still block public IP literals for catalog — require DNS names for official index.
		return fmt.Errorf("catalog URL host must not be an IP address")
	}
	return nil
}
