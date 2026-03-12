package s3storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MissingEndpoint(t *testing.T) {
	_, err := New(Config{
		Bucket:          "test-bucket",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

func TestNew_MissingBucket(t *testing.T) {
	_, err := New(Config{
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket is required")
}

func TestNew_MissingCredentials(t *testing.T) {
	_, err := New(Config{
		Endpoint: "http://localhost:9000",
		Bucket:   "test-bucket",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials are required")
}

func TestNew_Valid(t *testing.T) {
	p, err := New(Config{
		Endpoint:        "http://localhost:9000",
		Bucket:          "test-bucket",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewFromSettings(t *testing.T) {
	settings := map[string]string{
		"endpoint":         "http://localhost:9000",
		"bucket":           "my-bucket",
		"access_key_id":    "key",
		"secret_access_key": "secret",
		"region":           "us-east-1",
		"use_path_style":   "true",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.Equal(t, "my-bucket", p.config.Bucket)
	assert.True(t, p.config.UsePathStyle)
}

func TestURL_WithBaseURL(t *testing.T) {
	p, err := New(Config{
		Endpoint:        "http://internal-minio:9000",
		Bucket:          "assets",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		BaseURL:         "https://cdn.example.com",
	})
	require.NoError(t, err)

	url := p.URL("images/photo.jpg")
	assert.Equal(t, "https://cdn.example.com/images/photo.jpg", url)
}

func TestURL_WithoutBaseURL(t *testing.T) {
	p, err := New(Config{
		Endpoint:        "http://localhost:9000",
		Bucket:          "assets",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		UsePathStyle:    true,
	})
	require.NoError(t, err)

	url := p.URL("docs/file.pdf")
	assert.Contains(t, url, "assets")
	assert.Contains(t, url, "docs/file.pdf")
}

func TestSave_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	path, err := p.Save(context.Background(), "test.txt", strings.NewReader("hello"), 5)
	require.NoError(t, err)
	assert.Equal(t, "test.txt", path)
}

func TestSave_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	_, err = p.Save(context.Background(), "test.txt", strings.NewReader("hello"), 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGet_Success(t *testing.T) {
	content := "file content here"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	rc, err := p.Get(context.Background(), "test.txt")
	require.NoError(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	_, err = p.Get(context.Background(), "missing.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDelete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	err = p.Delete(context.Background(), "test.txt")
	require.NoError(t, err)
}

func TestDelete_NotFound_IsOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	// Deleting a non-existent file should not return an error
	err = p.Delete(context.Background(), "ghost.txt")
	require.NoError(t, err)
}

func TestExists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	exists, err := p.Exists(context.Background(), "test.txt")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExists_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p, err := New(Config{
		Endpoint:        srv.URL,
		Bucket:          "test",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		UsePathStyle:    true,
	})
	require.NoError(t, err)
	p.httpClient = srv.Client()

	exists, err := p.Exists(context.Background(), "missing.txt")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestManifest(t *testing.T) {
	err := Manifest.Validate()
	require.NoError(t, err)
}

func TestDeriveSigningKey(t *testing.T) {
	// Just verify it doesn't panic and returns non-nil
	key := deriveSigningKey("wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "20150830", "us-east-1")
	assert.NotNil(t, key)
	assert.Len(t, key, 32) // SHA-256 output is 32 bytes
}
