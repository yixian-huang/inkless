package agentcli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestClientContentDraftApplyPublish(t *testing.T) {
	var lastIfMatch string
	var published map[string]any
	draft := map[string]any{
		"pageKey": "home",
		"version": 2,
		"config": map[string]any{
			"hero": map[string]any{"title": map[string]any{"zh": "旧", "en": "Old"}},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/content/home/draft", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ink_test" {
			http.Error(w, "unauthorized", 401)
			return
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(draft)
		case http.MethodPut:
			lastIfMatch = r.Header.Get("If-Match")
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			draft["version"] = 3
			draft["config"] = body["config"]
			_ = json.NewEncoder(w).Encode(map[string]any{"pageKey": "home", "version": 3})
		default:
			http.Error(w, "method", 405)
		}
	})
	mux.HandleFunc("/admin/content/home/validate", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "errors": []any{}})
	})
	mux.HandleFunc("/admin/content/home/publish", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&published)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pageKey": "home", "publishedVersion": 1,
		})
	})
	mux.HandleFunc("/public/content/home", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pageKey": "home", "version": 1, "locale": r.URL.Query().Get("locale"),
			"config": draft["config"],
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewClient(&Endpoint{BaseURL: srv.URL, APIKey: "ink_test"})

	got, err := c.GetContentDraft(context.Background(), "home")
	if err != nil {
		t.Fatal(err)
	}
	if ContentDraftVersion(got) != 2 {
		t.Fatalf("version=%v", got["version"])
	}

	cfg := map[string]any{
		"hero": map[string]any{"title": map[string]any{"zh": "新", "en": "New"}},
		"install": map[string]any{"code": "curl | sh"},
	}
	diff := ShallowConfigDiff(got["config"].(map[string]any), cfg)
	changed, _ := diff["changed"].([]string)
	added, _ := diff["added"].([]string)
	if len(changed) == 0 || len(added) == 0 {
		t.Fatalf("diff=%v", diff)
	}

	_, err = c.PutContentDraft(context.Background(), "home", 2, cfg, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if lastIfMatch != "2" {
		t.Fatalf("If-Match=%q", lastIfMatch)
	}

	pub, err := c.PublishContent(context.Background(), "home", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if pub["publishedVersion"] != float64(1) {
		t.Fatalf("pub=%v", pub)
	}
	if published["expectedDraftVersion"] != float64(3) {
		t.Fatalf("body=%v", published)
	}

	public, err := c.GetPublicContent(context.Background(), "home", "zh")
	if err != nil {
		t.Fatal(err)
	}
	if public["locale"] != "zh" {
		t.Fatalf("public=%v", public)
	}
}

func TestContentConfigFromFileBody(t *testing.T) {
	cfg, err := ContentConfigFromFileBody(map[string]any{
		"config":     map[string]any{"hero": map[string]any{}},
		"changeNote": "x",
	})
	if err != nil || cfg["hero"] == nil {
		t.Fatalf("%v %v", cfg, err)
	}
	bare, err := ContentConfigFromFileBody(map[string]any{"hero": map[string]any{"x": 1}})
	if err != nil || bare["hero"] == nil {
		t.Fatalf("%v %v", bare, err)
	}
}

func TestClientUploadMedia(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/media/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ink_test" {
			http.Error(w, "unauthorized", 401)
			return
		}
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		defer f.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "url": "/uploads/" + hdr.Filename, "filename": hdr.Filename,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir() + "/shot.png"
	// minimal valid-ish file content
	if err := os.WriteFile(tmp, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewClient(&Endpoint{BaseURL: srv.URL, APIKey: "ink_test"})
	out, err := c.UploadMedia(context.Background(), tmp)
	if err != nil {
		t.Fatal(err)
	}
	if out["url"] != "/uploads/shot.png" {
		t.Fatalf("%v", out)
	}
}
