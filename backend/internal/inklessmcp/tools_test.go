package inklessmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestListSitesAndApplyDryRun(t *testing.T) {
	dir := t.TempDir()
	fleetPath := filepath.Join(dir, "fleet.json")
	keyEnv := "INKLESS_MCP_TEST_KEY"
	t.Setenv(keyEnv, "ink_testkey")

	var putBody map[string]any
	var serverURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/agent/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"baseUrl":    serverURL,
			"authMethod": "api_key",
			"scopes":     []string{"articles:read", "articles:update"},
			"user":       map[string]any{"id": 1, "username": "a", "role": "editor"},
		})
	})
	mux.HandleFunc("/admin/articles/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			_ = json.NewEncoder(w).Encode(putBody)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 7, "slug": "hello", "zhTitle": "Hi", "enTitle": "",
			"zhBody": "body", "enBody": "", "status": "draft",
			"zhSeoTitle": "", "enSeoTitle": "", "zhMetaDescription": "", "enMetaDescription": "",
			"updatedAt": "2026-01-01T00:00:00Z",
		})
	})
	mux.HandleFunc("/admin/articles", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": 7, "slug": "hello", "zhTitle": "Hi", "zhSeoTitle": "", "enSeoTitle": "",
					"zhMetaDescription": "", "enMetaDescription": ""},
			},
			"total": 1, "page": 1, "pageSize": 20,
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	serverURL = ts.URL

	fleet := map[string]any{
		"version": 1, "default_site": "ops",
		"sites": map[string]any{
			"ops": map[string]any{
				"label": "Ops", "base_url": ts.URL, "api_key_env": keyEnv, "publish_policy": "never",
			},
		},
	}
	raw, _ := json.Marshal(fleet)
	if err := os.WriteFile(fleetPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Options{FleetPath: fleetPath, Version: "test"})

	_, out, err := s.toolListSites(context.Background(), nil, emptyArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if out.(map[string]any)["defaultSite"] != "ops" {
		t.Fatalf("%v", out)
	}

	_, out, err = s.toolListArticles(context.Background(), nil, listArticlesArgs{MissingSEO: true})
	if err != nil {
		t.Fatal(err)
	}
	items := out.(map[string]any)["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}

	dry := true
	_, out, err = s.toolApplyArticlePatch(context.Background(), nil, applyArticleArgs{
		ID: 7, DryRun: &dry,
		Patch: map[string]any{"zhSeoTitle": "SEO Title", "zhMetaDescription": "meta desc here"},
	})
	if err != nil {
		t.Fatal(err)
	}
	handle, _ := out.(map[string]any)["previewHandle"].(string)
	if handle == "" {
		t.Fatalf("no handle: %v", out)
	}
	if putBody != nil {
		t.Fatal("should not PUT on dry run")
	}

	f := false
	_, _, err = s.toolApplyArticlePatch(context.Background(), nil, applyArticleArgs{
		ID: 7, DryRun: &f, PreviewHandle: handle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if putBody["zhSeoTitle"] != "SEO Title" {
		t.Fatalf("put=%v", putBody)
	}

	if s.MCP() == nil {
		t.Fatal("nil mcp server")
	}
}
