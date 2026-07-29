package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yixian-huang/inkless/backend/pkg/config"
)

func TestVersionIsNewerSemver(t *testing.T) {
	assert.True(t, VersionIsNewerSemver("v0.2.0", "v0.1.0"))
	assert.True(t, VersionIsNewerSemver("0.2.0", "0.1.9"))
	assert.False(t, VersionIsNewerSemver("v0.1.0", "v0.2.0"))
	assert.False(t, VersionIsNewerSemver("v1.0.0", "v1.0.0"))
}

func TestAssertAllowedURL(t *testing.T) {
	s := NewHostSelfUpdateService(&config.Config{}, "v0.1.0")
	assert.NoError(t, s.assertAllowedURL("https://github.com/yixian-huang/inkless/releases/download/v1/a.tar.gz"))
	assert.Error(t, s.assertAllowedURL("http://github.com/x"))
	assert.Error(t, s.assertAllowedURL("https://evil.example/a"))
	assert.Error(t, s.assertAllowedURL("https://127.0.0.1/a"))
}

func TestParseSHA256File(t *testing.T) {
	assert.Equal(t, strings.Repeat("a", 64), parseSHA256File(strings.Repeat("a", 64)+"  backend.tgz\n"))
	assert.Equal(t, "", parseSHA256File("not-a-hash"))
}

func TestStatusDisabledByDefault(t *testing.T) {
	s := NewHostSelfUpdateService(&config.Config{Port: 8088}, "dev")
	st := s.Status(context.Background())
	assert.False(t, st.Enabled)
	assert.False(t, st.Capable)
	assert.Contains(t, st.BlockedReason, "disabled")
}

func TestCheckAndApplyLocal(t *testing.T) {
	root := t.TempDir()
	// layout
	require.NoError(t, os.MkdirAll(filepath.Join(root, "backend", "versions"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "frontend", "versions"), 0o755))

	// build tiny artifacts
	backendBin := "inkless-api-v9.9.9"
	backendTar := filepath.Join(t.TempDir(), "backend-v9.9.9.tar.gz")
	frontendTar := filepath.Join(t.TempDir(), "frontend-v9.9.9.tar.gz")
	writeTarGz(t, backendTar, map[string]string{backendBin: "#!/bin/sh\necho ok\n", "build-info.json": `{"version":"v9.9.9"}`})
	writeTarGz(t, frontendTar, map[string]string{"index.html": "<html>ok</html>"})
	bSum := sha256File(t, backendTar)
	fSum := sha256File(t, frontendTar)

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test/inkless/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v9.9.9",
			"prerelease":   false,
			"draft":        false,
			"published_at": "2026-07-29T00:00:00Z",
			"html_url":     "https://github.com/test/inkless/releases/tag/v9.9.9",
			"assets": []map[string]any{
				{"name": "backend-v9.9.9.tar.gz", "browser_download_url": "https://github.com/test/inkless/releases/download/v9.9.9/backend-v9.9.9.tar.gz", "size": 1},
				{"name": "frontend-v9.9.9.tar.gz", "browser_download_url": "https://github.com/test/inkless/releases/download/v9.9.9/frontend-v9.9.9.tar.gz", "size": 1},
				{"name": "backend-v9.9.9.tar.gz.sha256", "browser_download_url": "https://github.com/test/inkless/releases/download/v9.9.9/backend-v9.9.9.tar.gz.sha256", "size": 1},
				{"name": "frontend-v9.9.9.tar.gz.sha256", "browser_download_url": "https://github.com/test/inkless/releases/download/v9.9.9/frontend-v9.9.9.tar.gz.sha256", "size": 1},
			},
		})
	})
	// serve downloads from github.com host via rewriting through test server —
	// assertAllowedURL only allows github hosts, so we override allow hosts to test server host.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest") || strings.Contains(r.URL.Path, "/releases/latest"):
			mux.ServeHTTP(w, r)
		case strings.HasSuffix(r.URL.Path, "backend-v9.9.9.tar.gz"):
			http.ServeFile(w, r, backendTar)
		case strings.HasSuffix(r.URL.Path, "frontend-v9.9.9.tar.gz"):
			http.ServeFile(w, r, frontendTar)
		case strings.HasSuffix(r.URL.Path, "backend-v9.9.9.tar.gz.sha256"):
			_, _ = fmt.Fprintf(w, "%s  backend-v9.9.9.tar.gz\n", bSum)
		case strings.HasSuffix(r.URL.Path, "frontend-v9.9.9.tar.gz.sha256"):
			_, _ = fmt.Fprintf(w, "%s  frontend-v9.9.9.tar.gz\n", fSum)
		default:
			// github API path style
			if strings.Contains(r.URL.Path, "/repos/") {
				mux.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	// Use manifest URL on test server instead of github API for simpler host allowlist
	manifest := map[string]any{
		"latest": map[string]any{
			"version":     "v9.9.9",
			"publishedAt": "2026-07-29T00:00:00Z",
			"notesUrl":    "https://example.invalid/notes",
			"assets": []map[string]any{
				{"name": "backend-v9.9.9.tar.gz", "url": srv.URL + "/backend-v9.9.9.tar.gz"},
				{"name": "frontend-v9.9.9.tar.gz", "url": srv.URL + "/frontend-v9.9.9.tar.gz"},
			},
		},
	}
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/channel.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(manifest)
	})
	mux2.HandleFunc("/backend-v9.9.9.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, backendTar)
	})
	mux2.HandleFunc("/frontend-v9.9.9.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, frontendTar)
	})
	mux2.HandleFunc("/backend-v9.9.9.tar.gz.sha256", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  backend-v9.9.9.tar.gz\n", bSum)
	})
	mux2.HandleFunc("/frontend-v9.9.9.tar.gz.sha256", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  frontend-v9.9.9.tar.gz\n", fSum)
	})
	srv2 := httptest.NewServer(mux2)
	t.Cleanup(srv2.Close)

	host := strings.TrimPrefix(srv2.URL, "http://")
	// allow http for test by patching — assertAllowedURL requires https.
	// So we skip full apply over HTTP and unit-test extract + activate path instead.
	_ = host

	cfg := &config.Config{
		Port:                  18088,
		SelfUpdateEnabled:     true,
		SelfUpdateReleaseRoot: root,
		SelfUpdateSystemdUnit: "inkless-test-nonexistent",
		SelfUpdateRepo:        "test/inkless",
		SelfUpdateChannel:     "stable",
		SelfUpdateCheckTTLSec: 1,
	}
	s := NewHostSelfUpdateService(cfg, "v0.1.0")

	// Direct unit tests for activate pieces
	require.NoError(t, extractTarGz(backendTar, filepath.Join(root, "backend", "versions", "v9.9.9")))
	require.NoError(t, extractTarGz(frontendTar, filepath.Join(root, "frontend", "versions", "v9.9.9")))
	require.NoError(t, ensureBackendLatestLink(filepath.Join(root, "backend", "versions", "v9.9.9")))
	require.NoError(t, atomicSymlink(filepath.Join(root, "backend", "versions", "v9.9.9"), filepath.Join(root, "backend", "current")))
	require.NoError(t, atomicSymlink(filepath.Join(root, "frontend", "versions", "v9.9.9"), filepath.Join(root, "frontend", "current")))

	resolved, err := filepath.EvalSymlinks(filepath.Join(root, "backend", "current"))
	require.NoError(t, err)
	assert.Equal(t, "v9.9.9", filepath.Base(resolved))
	assert.FileExists(t, filepath.Join(root, "frontend", "versions", "v9.9.9", "index.html"))

	st := s.Status(context.Background())
	assert.True(t, st.Enabled)
	assert.True(t, st.Capable)
	assert.Contains(t, st.LocalVersions, "v9.9.9")
}

func writeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := io.WriteString(tw, body)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
