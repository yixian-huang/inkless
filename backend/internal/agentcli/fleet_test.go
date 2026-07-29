package agentcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFleetAndResolve(t *testing.T) {
	dir := t.TempDir()
	fleetPath := filepath.Join(dir, "fleet.json")
	content := `{
	  "version": 1,
	  "default_site": "ops",
	  "sites": {
	    "ops": {
	      "label": "Ops",
	      "base_url": "https://ops.example.com/",
	      "api_key_env": "TEST_INKLESS_KEY_OPS",
	      "publish_policy": "never",
	      "scopes_expected": ["articles:read"]
	    },
	    "blog": {
	      "base_url": "https://blog.example.com",
	      "api_key_file": "` + filepath.ToSlash(filepath.Join(dir, "blog.key")) + `"
	    }
	  }
	}`
	if err := os.WriteFile(fleetPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "blog.key"), []byte("ink_blogsecret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_INKLESS_KEY_OPS", "ink_opssecret")

	f, err := LoadFleet(fleetPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Sites) != 2 {
		t.Fatalf("sites=%d", len(f.Sites))
	}

	ep, err := ResolveEndpoint(ResolveOptions{FleetPath: fleetPath})
	if err != nil {
		t.Fatal(err)
	}
	if ep.SiteID != "ops" || ep.BaseURL != "https://ops.example.com" {
		t.Fatalf("ep=%+v", ep)
	}
	if ep.APIKey != "ink_opssecret" || ep.PublishPolicy != "never" {
		t.Fatalf("key/policy=%+v", ep)
	}

	ep2, err := ResolveEndpoint(ResolveOptions{FleetPath: fleetPath, SiteID: "blog"})
	if err != nil {
		t.Fatal(err)
	}
	if ep2.APIKey != "ink_blogsecret" || ep2.BaseURL != "https://blog.example.com" {
		t.Fatalf("blog=%+v", ep2)
	}
}

func TestResolveSingleSiteEnv(t *testing.T) {
	t.Setenv("INKLESS_SITE", "")
	t.Setenv("INKLESS_FLEET", "")
	ep, err := ResolveEndpoint(ResolveOptions{
		BaseURL: "https://solo.example.com/",
		APIKey:  "ink_solo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ep.BaseURL != "https://solo.example.com" || ep.APIKey != "ink_solo" || ep.SiteID != "default" {
		t.Fatalf("%+v", ep)
	}
}

func TestNormalizeAndMask(t *testing.T) {
	if NormalizeBaseURL("https://x.com/") != "https://x.com" {
		t.Fatal(NormalizeBaseURL("https://x.com/"))
	}
	m := MaskSecret("ink_0123456789abcdef")
	if m == "ink_0123456789abcdef" || m == "" {
		t.Fatalf("mask=%q", m)
	}
}

func TestLoadFleetValidation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(p, []byte(`{"version":1,"sites":{"a":{"base_url":"https://a.com"}}}`), 0o600)
	if _, err := LoadFleet(p); err == nil {
		t.Fatal("want error for missing key source")
	}
}
