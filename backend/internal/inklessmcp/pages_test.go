package inklessmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPagesListDraftAndPublishMRTR(t *testing.T) {
	dir := t.TempDir()
	fleetPath := filepath.Join(dir, "fleet.json")
	keyEnv := "INKLESS_MCP_PAGES_KEY"
	t.Setenv(keyEnv, "ink_pageskey")

	var published bool
	var putDraft map[string]any
	var serverURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/agent/whoami", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"baseUrl": serverURL, "authMethod": "api_key",
			"user": map[string]any{"id": 1, "username": "a", "role": "admin"},
		})
	})
	mux.HandleFunc("/admin/pages", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{"id": 3, "slug": "home", "status": "draft", "zhTitle": "首页"}},
		})
	})
	mux.HandleFunc("/admin/pages/3", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "slug": "home", "zhMetaDescription": ""})
	})
	mux.HandleFunc("/admin/pages/3/draft", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&putDraft)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"config": map[string]any{"sections": []any{}}})
	})
	mux.HandleFunc("/admin/pages/3/publish", func(w http.ResponseWriter, r *http.Request) {
		published = true
		_ = json.NewEncoder(w).Encode(map[string]any{"published": true, "version": 2})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	serverURL = ts.URL

	fleet := map[string]any{
		"version": 1, "default_site": "ops",
		"sites": map[string]any{
			"ops": map[string]any{
				"label": "Ops", "base_url": ts.URL, "api_key_env": keyEnv, "publish_policy": "manual",
			},
		},
	}
	raw, _ := json.Marshal(fleet)
	_ = os.WriteFile(fleetPath, raw, 0o600)

	s := New(Options{FleetPath: fleetPath, Version: "test"})

	_, out, err := s.toolListPages(context.Background(), nil, siteArgs{})
	if err != nil {
		t.Fatal(err)
	}
	if out.(map[string]any)["siteId"] != "ops" {
		t.Fatalf("%v", out)
	}

	// draft dry-run then commit
	dry := true
	_, out, err = s.toolPutPageDraft(context.Background(), nil, putPageDraftArgs{
		ID: 3, DryRun: &dry,
		Body: map[string]any{"config": map[string]any{"x": 1}, "changeNote": "mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	handle := out.(map[string]any)["previewHandle"].(string)
	if putDraft != nil {
		t.Fatal("no put on dry-run")
	}
	f := false
	_, _, err = s.toolPutPageDraft(context.Background(), nil, putPageDraftArgs{
		ID: 3, DryRun: &f, PreviewHandle: handle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if putDraft["changeNote"] != "mcp" {
		t.Fatalf("putDraft=%v", putDraft)
	}

	// publish without confirm → MRTR
	res, _, err := s.toolPublishPage(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{},
	}, publishPageArgs{ID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || len(res.InputRequests) == 0 {
		t.Fatalf("want input_required, got %#v", res)
	}
	if published {
		t.Fatal("should not publish yet")
	}
	state := res.RequestState
	if state == "" {
		t.Fatal("empty requestState")
	}

	// decline
	_, _, err = s.toolPublishPage(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			RequestState: state,
			InputResponses: mcp.InputResponseMap{
				"confirm": &mcp.ElicitResult{Action: "decline"},
			},
		},
	}, publishPageArgs{ID: 3})
	if err == nil {
		t.Fatal("want decline error")
	}

	// accept via MRTR
	_, out, err = s.toolPublishPage(context.Background(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			RequestState: state,
			InputResponses: mcp.InputResponseMap{
				"confirm": &mcp.ElicitResult{
					Action:  "accept",
					Content: map[string]any{"confirm": true},
				},
			},
		},
	}, publishPageArgs{ID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("expected publish")
	}
	if out.(map[string]any)["published"] != true {
		t.Fatalf("%v", out)
	}

	// never policy
	fleetNever := map[string]any{
		"version": 1, "default_site": "ops",
		"sites": map[string]any{
			"ops": map[string]any{
				"base_url": ts.URL, "api_key_env": keyEnv, "publish_policy": "never",
			},
		},
	}
	raw, _ = json.Marshal(fleetNever)
	_ = os.WriteFile(fleetPath, raw, 0o600)
	s2 := New(Options{FleetPath: fleetPath})
	_, _, err = s2.toolPublishPage(context.Background(), nil, publishPageArgs{ID: 3, Force: true})
	if err == nil {
		t.Fatal("never policy should reject even with force")
	}
}

func TestPublishConfirmationHelpers(t *testing.T) {
	if c, d := publishConfirmation(nil, true); !c || d {
		t.Fatal("force")
	}
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{
		InputResponses: mcp.InputResponseMap{
			"confirm": &mcp.ElicitResult{Action: "accept", Content: map[string]any{"confirm": false}},
		},
	}}
	if c, d := publishConfirmation(req, false); c || !d {
		t.Fatal("confirm false should decline")
	}
}
