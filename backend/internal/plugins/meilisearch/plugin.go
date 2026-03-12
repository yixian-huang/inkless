// Package meilisearch provides a Meilisearch-backed full-text search plugin for Impress CMS.
// It implements the provider.SearchProvider interface using the Meilisearch REST API.
package meilisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"blotting-consultancy/internal/plugin"
	"blotting-consultancy/internal/provider"
)

// Manifest describes this plugin's metadata.
var Manifest = plugin.PluginMeta{
	ID:            "mls-search",
	Name:          "Meilisearch Plugin",
	NameZh:        "Meilisearch 搜索插件",
	Version:       "1.0.0",
	Description:   "Replaces built-in search with Meilisearch for fast, typo-tolerant full-text search.",
	Author:        "Impress CMS",
	License:       "MIT",
	MinAppVersion: "1.0.0",
	Permissions:   []plugin.Permission{plugin.PermNetworkOutbound},
	Providers: []plugin.ProviderDecl{
		{Type: "search", Name: "meilisearch"},
	},
}

// Config holds the Meilisearch plugin configuration.
type Config struct {
	// Host is the Meilisearch server URL (e.g. "http://localhost:7700").
	Host string

	// APIKey is the Meilisearch API key (master key or search/write key).
	APIKey string

	// IndexPrefix is prepended to all index names (e.g. "cms_" produces "cms_articles").
	// Default: "impress_"
	IndexPrefix string
}

// indexedDoc is the document shape stored in Meilisearch.
type indexedDoc struct {
	ID          string  `json:"id"`          // "<type>_<id>" e.g. "article_42"
	NumericID   uint    `json:"numeric_id"`
	Type        string  `json:"type"`        // "article" or "page"
	Locale      string  `json:"locale"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	Slug        string  `json:"slug"`
	Score       float64 `json:"_rankingScore,omitempty"`
}

// searchRequest is the payload sent to Meilisearch /indexes/{index}/search.
type searchRequest struct {
	Q                    string   `json:"q"`
	Limit                int      `json:"limit"`
	Offset               int      `json:"offset"`
	AttributesToRetrieve []string `json:"attributesToRetrieve"`
	AttributesToHighlight []string `json:"attributesToHighlight,omitempty"`
	Filter               string   `json:"filter,omitempty"`
	ShowRankingScore     bool     `json:"showRankingScore"`
}

// searchResponse is the response from Meilisearch /indexes/{index}/search.
type searchResponse struct {
	Hits               []indexedDoc `json:"hits"`
	EstimatedTotalHits int64        `json:"estimatedTotalHits"`
	Query              string       `json:"query"`
}

// Plugin implements provider.SearchProvider using Meilisearch.
type Plugin struct {
	config      Config
	httpClient  *http.Client
	indexPrefix string
}

// New creates a new Meilisearch search plugin with the provided configuration.
func New(cfg Config) (*Plugin, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("meilisearch: host is required")
	}

	prefix := cfg.IndexPrefix
	if prefix == "" {
		prefix = "impress_"
	}

	return &Plugin{
		config:      cfg,
		indexPrefix: prefix,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

// NewFromSettings creates a Plugin from a string settings map (used by plugin manager).
func NewFromSettings(settings map[string]string) (*Plugin, error) {
	cfg := Config{
		Host:        settings["host"],
		APIKey:      settings["api_key"],
		IndexPrefix: settings["index_prefix"],
	}
	return New(cfg)
}

// indexName returns the full index name for a content type and locale.
func (p *Plugin) indexName(contentType, locale string) string {
	return fmt.Sprintf("%s%s_%s", p.indexPrefix, contentType, locale)
}

// doRequest performs an HTTP request against the Meilisearch API.
func (p *Plugin) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("meilisearch: failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	reqURL := strings.TrimRight(p.config.Host, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("meilisearch: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("meilisearch: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("meilisearch: failed to read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

// Search performs a full-text search query against the appropriate Meilisearch index.
func (p *Plugin) Search(ctx context.Context, query, locale, contentType string, page, pageSize int) (*provider.SearchResponse, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Determine which index(es) to search
	var indexesToSearch []string
	switch contentType {
	case "article":
		indexesToSearch = []string{p.indexName("articles", locale)}
	case "page":
		indexesToSearch = []string{p.indexName("pages", locale)}
	default:
		// Search both
		indexesToSearch = []string{
			p.indexName("articles", locale),
			p.indexName("pages", locale),
		}
	}

	var allResults []provider.SearchResult
	var totalHits int64

	for _, indexName := range indexesToSearch {
		sreq := searchRequest{
			Q:                    query,
			Limit:                pageSize,
			Offset:               offset,
			AttributesToRetrieve: []string{"id", "numeric_id", "type", "locale", "title", "body", "slug"},
			AttributesToHighlight: []string{"title", "body"},
			ShowRankingScore:     true,
		}

		respData, statusCode, err := p.doRequest(ctx, http.MethodPost,
			fmt.Sprintf("/indexes/%s/search", url.PathEscape(indexName)), sreq)
		if err != nil {
			return nil, err
		}
		if statusCode == http.StatusNotFound {
			// Index doesn't exist yet - not an error, just no results
			continue
		}
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("meilisearch: search returned status %d: %s", statusCode, string(respData))
		}

		var sresp searchResponse
		if err := json.Unmarshal(respData, &sresp); err != nil {
			return nil, fmt.Errorf("meilisearch: failed to parse search response: %w", err)
		}

		totalHits += sresp.EstimatedTotalHits
		for _, hit := range sresp.Hits {
			snippet := hit.Body
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			allResults = append(allResults, provider.SearchResult{
				ID:      hit.NumericID,
				Type:    hit.Type,
				Title:   hit.Title,
				Snippet: snippet,
				URL:     "/" + hit.Type + "s/" + hit.Slug,
				Locale:  hit.Locale,
				Score:   hit.Score,
			})
		}
	}

	return &provider.SearchResponse{
		Results:  allResults,
		Total:    totalHits,
		Page:     page,
		PageSize: pageSize,
		Query:    query,
	}, nil
}

// Suggest returns autocomplete suggestions for a given prefix.
func (p *Plugin) Suggest(ctx context.Context, prefix, locale string, limit int) ([]string, error) {
	// Use search with small limit to extract title suggestions
	resp, err := p.Search(ctx, prefix, locale, "", 1, limit)
	if err != nil {
		return nil, err
	}

	suggestions := make([]string, 0, len(resp.Results))
	seen := make(map[string]struct{})
	for _, r := range resp.Results {
		if _, ok := seen[r.Title]; !ok {
			suggestions = append(suggestions, r.Title)
			seen[r.Title] = struct{}{}
		}
	}
	return suggestions, nil
}

// IndexArticle adds or updates an article document in the Meilisearch index.
func (p *Plugin) IndexArticle(ctx context.Context, id uint, locale, title, body, slug string) error {
	return p.indexDocument(ctx, p.indexName("articles", locale), indexedDoc{
		ID:        fmt.Sprintf("article_%d", id),
		NumericID: id,
		Type:      "article",
		Locale:    locale,
		Title:     title,
		Body:      body,
		Slug:      slug,
	})
}

// IndexPage adds or updates a page document in the Meilisearch index.
func (p *Plugin) IndexPage(ctx context.Context, id uint, locale, title, body, slug string) error {
	return p.indexDocument(ctx, p.indexName("pages", locale), indexedDoc{
		ID:        fmt.Sprintf("page_%d", id),
		NumericID: id,
		Type:      "page",
		Locale:    locale,
		Title:     title,
		Body:      body,
		Slug:      slug,
	})
}

// indexDocument upserts a document into a Meilisearch index.
func (p *Plugin) indexDocument(ctx context.Context, indexName string, doc indexedDoc) error {
	docs := []indexedDoc{doc}
	respData, statusCode, err := p.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/indexes/%s/documents", url.PathEscape(indexName)), docs)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("meilisearch: index document returned status %d: %s", statusCode, string(respData))
	}
	return nil
}

// RemoveFromIndex removes a document from the Meilisearch index.
func (p *Plugin) RemoveFromIndex(ctx context.Context, contentType string, id uint) error {
	docID := fmt.Sprintf("%s_%d", contentType, id)

	// We need to remove from all locale-specific indexes. For simplicity,
	// try common locales. A production implementation would query all indexes.
	for _, locale := range []string{"zh", "en"} {
		var indexName string
		if contentType == "article" {
			indexName = p.indexName("articles", locale)
		} else {
			indexName = p.indexName("pages", locale)
		}

		path := fmt.Sprintf("/indexes/%s/documents/%s",
			url.PathEscape(indexName), url.PathEscape(docID))
		_, statusCode, err := p.doRequest(ctx, http.MethodDelete, path, nil)
		if err != nil {
			return err
		}
		// 404 means the index or document didn't exist - that's fine
		if statusCode != http.StatusNotFound && (statusCode < 200 || statusCode >= 300) {
			return fmt.Errorf("meilisearch: remove from index returned status %d", statusCode)
		}
	}
	return nil
}

// RebuildIndex recreates all indexes. In Meilisearch this means deleting and re-creating them.
// The actual re-indexing of content is handled by the caller (CMS core), which should
// call IndexArticle/IndexPage for each piece of content after this returns.
func (p *Plugin) RebuildIndex(ctx context.Context) error {
	indexesToReset := []string{
		p.indexName("articles", "zh"),
		p.indexName("articles", "en"),
		p.indexName("pages", "zh"),
		p.indexName("pages", "en"),
	}

	for _, name := range indexesToReset {
		// Delete the index if it exists
		_, statusCode, err := p.doRequest(ctx, http.MethodDelete,
			fmt.Sprintf("/indexes/%s", url.PathEscape(name)), nil)
		if err != nil {
			return fmt.Errorf("meilisearch: failed to delete index %s: %w", name, err)
		}
		if statusCode != http.StatusAccepted && statusCode != http.StatusNotFound {
			return fmt.Errorf("meilisearch: delete index %s returned status %d", name, statusCode)
		}

		// Re-create the index with "id" as the primary key
		createBody := map[string]string{
			"uid":        name,
			"primaryKey": "id",
		}
		respData, statusCode, err := p.doRequest(ctx, http.MethodPost, "/indexes", createBody)
		if err != nil {
			return fmt.Errorf("meilisearch: failed to create index %s: %w", name, err)
		}
		if statusCode < 200 || statusCode >= 300 {
			return fmt.Errorf("meilisearch: create index %s returned status %d: %s",
				name, statusCode, string(respData))
		}
	}
	return nil
}

// Ensure Plugin implements SearchProvider at compile time.
var _ provider.SearchProvider = (*Plugin)(nil)
