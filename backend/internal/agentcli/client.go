package agentcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client talks to one Inkless Admin API instance.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient builds a client for an endpoint.
func NewClient(ep *Endpoint) *Client {
	return &Client{
		BaseURL: NormalizeBaseURL(ep.BaseURL),
		APIKey:  ep.APIKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		UserAgent: "inkless-cli-agent",
	}
}

// Whoami is GET /admin/agent/whoami.
type Whoami struct {
	BaseURL      string         `json:"baseUrl"`
	Version      string         `json:"version"`
	AuthMethod   string         `json:"authMethod"`
	APIKeyID     *uint          `json:"apiKeyId"`
	Scopes       []string       `json:"scopes"`
	User         WhoamiUser     `json:"user"`
	Permissions  []string       `json:"permissions"`
	Capabilities map[string]any `json:"capabilities"`
}

// WhoamiUser is the authenticated principal.
type WhoamiUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// DoJSON performs an authenticated JSON request.
func (c *Client) DoJSON(ctx context.Context, method, path string, body any, out any) error {
	return c.DoJSONWithHeaders(ctx, method, path, body, nil, out)
}

// DoJSONWithHeaders is DoJSON plus extra request headers (e.g. If-Match).
func (c *Client) DoJSONWithHeaders(ctx context.Context, method, path string, body any, extraHeaders map[string]string, out any) error {
	if c == nil || c.BaseURL == "" {
		return fmt.Errorf("client base URL is empty")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API key is empty")
	}
	u := c.BaseURL + ensureLeadingSlash(path)
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for k, v := range extraHeaders {
		if k != "" && v != "" {
			req.Header.Set(k, v)
		}
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 500 {
			msg = msg[:500] + "…"
		}
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, msg)
	}
	if out == nil || len(data) == 0 || string(data) == "null" {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// GetWhoami calls GET /admin/agent/whoami.
func (c *Client) GetWhoami(ctx context.Context) (*Whoami, error) {
	var w Whoami
	if err := c.DoJSON(ctx, http.MethodGet, "/admin/agent/whoami", nil, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// VerifyWhoami ensures remote baseUrl matches the expected endpoint.
func (c *Client) VerifyWhoami(ctx context.Context, expectedBase string) (*Whoami, error) {
	w, err := c.GetWhoami(ctx)
	if err != nil {
		return nil, err
	}
	got := NormalizeBaseURL(w.BaseURL)
	want := NormalizeBaseURL(expectedBase)
	if got != "" && want != "" && !strings.EqualFold(got, want) {
		return w, fmt.Errorf("whoami baseUrl mismatch: got %q want %q (wrong instance or BASE_URL)", got, want)
	}
	return w, nil
}

// ListArticlesQuery is query params for GET /admin/articles.
type ListArticlesQuery struct {
	Page     int
	PageSize int
	Status   string
	Q        string
}

// ListResult is a generic admin list envelope.
type ListResult struct {
	Items    json.RawMessage `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// ListArticles calls GET /admin/articles.
func (c *Client) ListArticles(ctx context.Context, q ListArticlesQuery) (*ListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	vals := url.Values{}
	vals.Set("page", strconv.Itoa(q.Page))
	vals.Set("pageSize", strconv.Itoa(q.PageSize))
	if q.Status != "" {
		vals.Set("status", q.Status)
	}
	if q.Q != "" {
		vals.Set("q", q.Q)
	}
	path := "/admin/articles?" + vals.Encode()
	var out ListResult
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetArticle GET /admin/articles/:id
func (c *Client) GetArticle(ctx context.Context, id uint) (map[string]any, error) {
	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, fmt.Sprintf("/admin/articles/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PutArticle PUT /admin/articles/:id with a full body map (caller merges fields).
func (c *Client) PutArticle(ctx context.Context, id uint, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodPut, fmt.Sprintf("/admin/articles/%d", id), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatePage POST /admin/pages
func (c *Client) CreatePage(ctx context.Context, body map[string]any) (map[string]any, error) {
	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodPost, "/admin/pages", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListPages GET /admin/pages
func (c *Client) ListPages(ctx context.Context) (json.RawMessage, error) {
	var out map[string]json.RawMessage
	if err := c.DoJSON(ctx, http.MethodGet, "/admin/pages", nil, &out); err != nil {
		return nil, err
	}
	if items, ok := out["items"]; ok {
		return items, nil
	}
	// some handlers may return array root — unlikely
	raw, _ := json.Marshal(out)
	return raw, nil
}

// GetPage GET /admin/pages/:id
func (c *Client) GetPage(ctx context.Context, id uint) (map[string]any, error) {
	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, fmt.Sprintf("/admin/pages/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPageDraft GET /admin/pages/:id/draft
func (c *Client) GetPageDraft(ctx context.Context, id uint) (map[string]any, error) {
	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodGet, fmt.Sprintf("/admin/pages/%d/draft", id), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PutPageDraft PUT /admin/pages/:id/draft with If-Match from current draft version.
// body may be the draft config map, or already wrapped as {"draftConfig": ...}.
func (c *Client) PutPageDraft(ctx context.Context, id uint, body map[string]any) (map[string]any, error) {
	draft, err := c.GetPageDraft(ctx, id)
	if err != nil {
		return nil, err
	}
	ver := 0
	switch v := draft["draftVersion"].(type) {
	case float64:
		ver = int(v)
	case int:
		ver = v
	case json.Number:
		n, _ := v.Int64()
		ver = int(n)
	}
	if ver <= 0 {
		return nil, fmt.Errorf("page %d: missing draftVersion for If-Match", id)
	}

	payload := body
	if _, ok := body["draftConfig"]; !ok {
		payload = map[string]any{"draftConfig": body}
	}

	var out map[string]any
	if err := c.DoJSONWithHeaders(ctx, http.MethodPut, fmt.Sprintf("/admin/pages/%d/draft", id), payload, map[string]string{
		"If-Match": strconv.Itoa(ver),
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PublishPage POST /admin/pages/:id/publish
func (c *Client) PublishPage(ctx context.Context, id uint) (map[string]any, error) {
	var out map[string]any
	if err := c.DoJSON(ctx, http.MethodPost, fmt.Sprintf("/admin/pages/%d/publish", id), map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ArticleMissingSEO returns true if common SEO fields look empty.
func ArticleMissingSEO(item map[string]any) bool {
	keys := []string{"zhSeoTitle", "enSeoTitle", "zhMetaDescription", "enMetaDescription"}
	empty := 0
	for _, k := range keys {
		if strings.TrimSpace(asString(item[k])) == "" {
			empty++
		}
	}
	// Missing if at least two SEO fields empty, or both meta empty
	metaEmpty := strings.TrimSpace(asString(item["zhMetaDescription"])) == "" &&
		strings.TrimSpace(asString(item["enMetaDescription"])) == ""
	return empty >= 2 || metaEmpty
}

// MergeArticleUpdate builds a PUT body from current article + patch map.
// Only known writable fields are included; patch keys override current values.
func MergeArticleUpdate(current, patch map[string]any) map[string]any {
	fields := []string{
		"slug", "status",
		"zhTitle", "enTitle", "zhBody", "enBody",
		"coverImage",
		"zhSeoTitle", "enSeoTitle", "zhMetaDescription", "enMetaDescription",
		"ogImage", "author", "visibility",
		"categoryId", "categoryIds", "tagIds",
		"autoSummary", "allowComments", "pinned", "metadata",
		"baseUpdatedAt", "scheduledAt",
	}
	out := map[string]any{}
	for _, k := range fields {
		if v, ok := patch[k]; ok {
			out[k] = v
			continue
		}
		if v, ok := current[k]; ok {
			out[k] = v
		}
	}
	// Accept updatedAt as baseUpdatedAt for optimistic concurrency when present
	if _, ok := out["baseUpdatedAt"]; !ok {
		if v, ok := current["updatedAt"]; ok {
			out["baseUpdatedAt"] = v
		}
	}
	return out
}

func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}
