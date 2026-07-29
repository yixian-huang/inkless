package agentcli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientWhoamiAndArticles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/agent/whoami", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ink_test" {
			http.Error(w, "unauthorized", 401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"baseUrl":    "https://ops.example.com",
			"authMethod": "api_key",
			"scopes":     []string{"articles:read"},
			"user":       map[string]any{"id": 1, "username": "a", "role": "editor"},
			"capabilities": map[string]any{
				"articles": true,
			},
		})
	})
	mux.HandleFunc("/admin/articles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": 1, "slug": "a", "zhTitle": "A", "zhSeoTitle": "", "enSeoTitle": "", "zhMetaDescription": "", "enMetaDescription": ""},
				{"id": 2, "slug": "b", "zhTitle": "B", "zhSeoTitle": "ok", "enSeoTitle": "ok", "zhMetaDescription": "d", "enMetaDescription": "d"},
			},
			"total": 2, "page": 1, "pageSize": 20,
		})
	})
	mux.HandleFunc("/admin/articles/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			body["id"] = 1
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "slug": "a", "zhTitle": "A", "enTitle": "", "zhBody": "x", "enBody": "",
			"zhSeoTitle": "", "enSeoTitle": "", "zhMetaDescription": "", "enMetaDescription": "",
			"status": "draft", "updatedAt": "2026-01-01T00:00:00Z",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(&Endpoint{BaseURL: srv.URL, APIKey: "ink_test"})
	w, err := c.VerifyWhoami(context.Background(), "https://ops.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if w.AuthMethod != "api_key" {
		t.Fatalf("%+v", w)
	}

	// mismatch
	if _, err := c.VerifyWhoami(context.Background(), "https://other.example.com"); err == nil {
		t.Fatal("want mismatch error")
	}

	list, err := c.ListArticles(context.Background(), ListArticlesQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(list.Items, &items); err != nil {
		t.Fatal(err)
	}
	if !ArticleMissingSEO(items[0]) || ArticleMissingSEO(items[1]) {
		t.Fatalf("missing-seo filter wrong: %+v %+v", items[0], items[1])
	}

	cur, err := c.GetArticle(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	merged := MergeArticleUpdate(cur, map[string]any{"zhSeoTitle": "SEO A", "zhMetaDescription": "meta"})
	if merged["zhSeoTitle"] != "SEO A" || merged["slug"] != "a" {
		t.Fatalf("merge=%v", merged)
	}
	out, err := c.PutArticle(context.Background(), 1, merged)
	if err != nil {
		t.Fatal(err)
	}
	if out["zhSeoTitle"] != "SEO A" {
		t.Fatalf("put=%v", out)
	}
}
