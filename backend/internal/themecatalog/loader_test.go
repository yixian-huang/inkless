package themecatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_NoRemote_UsesEmbedded(t *testing.T) {
	l := NewLoader("")
	res, err := l.Load(context.Background(), false)
	require.NoError(t, err)
	require.NotNil(t, res.Catalog)
	assert.Equal(t, SourceEmbedded, res.Source)
	assert.Empty(t, res.Warning)
	_, ok := res.Catalog.FindBySlug("product-first")
	assert.True(t, ok)
}

func TestLoader_CacheHit_WithoutNetwork(t *testing.T) {
	l := NewLoader("https://inkless.run/marketplace/v1/themes.json", WithCacheTTL(time.Hour))
	embedded, err := LoadEmbedded()
	require.NoError(t, err)
	l.setCache(embedded, time.Now())

	res, err := l.Load(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, SourceCache, res.Source)
	_, ok := res.Catalog.FindBySlug("product-first")
	assert.True(t, ok)
}

func TestLoader_RefreshBypassesCache_FallsBackOnError(t *testing.T) {
	// Unreachable path on a real public host → network or 404 error.
	l := NewLoader("https://inkless.run/marketplace/v1/themes-does-not-exist-404.json",
		WithHTTPClient(&http.Client{Timeout: 2 * time.Second}),
		WithCacheTTL(time.Hour),
	)
	embedded, err := LoadEmbedded()
	require.NoError(t, err)
	l.setCache(embedded, time.Now())

	res, err := l.Load(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, SourceCache, res.Source)
	assert.Contains(t, res.Warning, "remote catalog fetch failed")
	_, ok := res.Catalog.FindBySlug("blog-first")
	assert.True(t, ok)
}

func TestLoader_RemoteFail_NoCache_UsesEmbedded(t *testing.T) {
	l := NewLoader("https://inkless.run/marketplace/v1/themes-missing.json",
		WithHTTPClient(&http.Client{Timeout: 2 * time.Second}),
	)
	res, err := l.Load(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, SourceEmbedded, res.Source)
	assert.Contains(t, res.Warning, "embedded")
	require.NotEmpty(t, l.LastError())
}

func TestValidateCatalogURL(t *testing.T) {
	require.NoError(t, validateCatalogURL("https://inkless.run/marketplace/v1/themes.json"))
	require.Error(t, validateCatalogURL("http://inkless.run/x"))
	require.Error(t, validateCatalogURL("https://127.0.0.1/x"))
	require.Error(t, validateCatalogURL("https://localhost/x"))
}

func TestLoader_FetchRemote_SuccessViaRoundTrip(t *testing.T) {
	var hits int
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		hits++
		assert.Equal(t, "https://catalog.example/themes.json", req.URL.String())
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		_, _ = rec.WriteString(`{
		  "schemaVersion": 1,
		  "themes": [{
		    "slug": "from-remote",
		    "themeId": "from-remote",
		    "name": "From Remote",
		    "contractVersion": "1",
		    "latest": {"version": "3.0.0", "umdUrl": "https://github.com/org/repo/theme.umd.js"},
		    "official": true
		  }]
		}`)
		return rec.Result(), nil
	})
	client := &http.Client{Transport: rt, Timeout: 2 * time.Second}
	l := NewLoader("https://catalog.example/themes.json", WithHTTPClient(client), WithCacheTTL(time.Minute))

	res, err := l.Load(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, SourceRemote, res.Source)
	assert.Empty(t, res.Warning)
	e, ok := res.Catalog.FindBySlug("from-remote")
	require.True(t, ok)
	assert.Equal(t, "3.0.0", e.Latest.Version)
	assert.Equal(t, 1, hits)

	// Second load without refresh → cache (no extra hit)
	res2, err := l.Load(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, SourceCache, res2.Source)
	assert.Equal(t, 1, hits)

	// refresh forces another GET
	res3, err := l.Load(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, SourceRemote, res3.Source)
	assert.Equal(t, 2, hits)
}

func TestLoader_FetchRemote_InvalidJSON_FallsBack(t *testing.T) {
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		_, _ = rec.WriteString(`{"schemaVersion":1,"themes":[]}`)
		return rec.Result(), nil
	})
	client := &http.Client{Transport: rt}
	l := NewLoader("https://catalog.example/themes.json", WithHTTPClient(client))

	res, err := l.Load(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, SourceEmbedded, res.Source)
	assert.NotEmpty(t, res.Warning)
}

func TestLoader_ExpiredCache_Refetches(t *testing.T) {
	var hits int
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		hits++
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		_, _ = rec.WriteString(`{
		  "schemaVersion": 1,
		  "themes": [{
		    "slug": "ttl-theme",
		    "themeId": "ttl-theme",
		    "name": "TTL",
		    "contractVersion": "1",
		    "latest": {"version": "1.0.0", "umdUrl": "https://github.com/org/repo/theme.umd.js"},
		    "official": true
		  }]
		}`)
		return rec.Result(), nil
	})
	client := &http.Client{Transport: rt}
	l := NewLoader("https://catalog.example/themes.json",
		WithHTTPClient(client),
		WithCacheTTL(20*time.Millisecond),
	)

	_, err := l.Load(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, 1, hits)

	time.Sleep(40 * time.Millisecond)
	res, err := l.Load(context.Background(), false)
	require.NoError(t, err)
	assert.Equal(t, SourceRemote, res.Source)
	assert.Equal(t, 2, hits)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
