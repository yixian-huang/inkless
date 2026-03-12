package meilisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MissingHost(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestNew_Valid(t *testing.T) {
	p, err := New(Config{Host: "http://localhost:7700"})
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "impress_", p.indexPrefix)
}

func TestNew_CustomPrefix(t *testing.T) {
	p, err := New(Config{Host: "http://localhost:7700", IndexPrefix: "cms_"})
	require.NoError(t, err)
	assert.Equal(t, "cms_", p.indexPrefix)
}

func TestNewFromSettings(t *testing.T) {
	settings := map[string]string{
		"host":         "http://meilisearch:7700",
		"api_key":      "masterKey",
		"index_prefix": "test_",
	}
	p, err := NewFromSettings(settings)
	require.NoError(t, err)
	assert.Equal(t, "http://meilisearch:7700", p.config.Host)
	assert.Equal(t, "masterKey", p.config.APIKey)
	assert.Equal(t, "test_", p.indexPrefix)
}

func TestIndexName(t *testing.T) {
	p, _ := New(Config{Host: "http://localhost:7700"})
	assert.Equal(t, "impress_articles_zh", p.indexName("articles", "zh"))
	assert.Equal(t, "impress_pages_en", p.indexName("pages", "en"))
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/search")

		resp := searchResponse{
			Hits: []indexedDoc{
				{
					ID:        "article_1",
					NumericID: 1,
					Type:      "article",
					Locale:    "zh",
					Title:     "Test Article",
					Body:      "Some body content",
					Slug:      "test-article",
					Score:     0.95,
				},
			},
			EstimatedTotalHits: 1,
			Query:              "test",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	result, err := p.Search(context.Background(), "test", "zh", "article", 1, 10)
	require.NoError(t, err)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "Test Article", result.Results[0].Title)
	assert.Equal(t, "zh", result.Results[0].Locale)
	assert.Equal(t, "test", result.Query)
}

func TestSearch_IndexNotFound_ReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	result, err := p.Search(context.Background(), "test", "zh", "article", 1, 10)
	require.NoError(t, err)
	assert.Empty(t, result.Results)
	assert.Equal(t, int64(0), result.Total)
}

func TestSearch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	_, err := p.Search(context.Background(), "test", "zh", "article", 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestSearch_SnippetTruncation(t *testing.T) {
	longBody := string(make([]byte, 300))
	for i := range longBody {
		longBody = longBody[:i] + "x" + longBody[i+1:]
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := searchResponse{
			Hits: []indexedDoc{
				{
					ID:        "article_1",
					NumericID: 1,
					Type:      "article",
					Locale:    "zh",
					Title:     "Long Article",
					Body:      longBody,
					Slug:      "long",
				},
			},
			EstimatedTotalHits: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	result, err := p.Search(context.Background(), "x", "zh", "article", 1, 10)
	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.LessOrEqual(t, len(result.Results[0].Snippet), 203) // 200 + "..."
}

func TestIndexArticle_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/documents")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1}`))
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	err := p.IndexArticle(context.Background(), 42, "zh", "Title", "Body text", "my-article")
	require.NoError(t, err)
}

func TestIndexPage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":2}`))
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	err := p.IndexPage(context.Background(), 7, "en", "Page Title", "Page content", "my-page")
	require.NoError(t, err)
}

func TestRemoveFromIndex_Success(t *testing.T) {
	deleteCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteCount++
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	err := p.RemoveFromIndex(context.Background(), "article", 42)
	require.NoError(t, err)
	// Should attempt deletion from both zh and en indexes
	assert.Equal(t, 2, deleteCount)
}

func TestRebuildIndex_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{"taskUid":1}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"uid":"test","primaryKey":"id"}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	err := p.RebuildIndex(context.Background())
	require.NoError(t, err)
}

func TestSuggest_ReturnsTitles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := searchResponse{
			Hits: []indexedDoc{
				{ID: "article_1", NumericID: 1, Type: "article", Locale: "zh", Title: "Alpha", Slug: "alpha"},
				{ID: "article_2", NumericID: 2, Type: "article", Locale: "zh", Title: "Beta", Slug: "beta"},
			},
			EstimatedTotalHits: 2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := New(Config{Host: srv.URL})
	p.httpClient = srv.Client()

	suggestions, err := p.Suggest(context.Background(), "al", "zh", 5)
	require.NoError(t, err)
	assert.Contains(t, suggestions, "Alpha")
}

func TestManifest(t *testing.T) {
	err := Manifest.Validate()
	require.NoError(t, err)
}
