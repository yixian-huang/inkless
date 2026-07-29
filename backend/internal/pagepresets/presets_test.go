package pagepresets

import (
	"testing"
)

func TestParseID(t *testing.T) {
	id, err := ParseID("doc-guide")
	if err != nil || id != DocGuide {
		t.Fatalf("ParseID doc-guide: %v %v", id, err)
	}
	if _, err := ParseID("nope"); err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

func TestBuildDocSimple(t *testing.T) {
	cfg, err := Build(DocSimple, Options{ZhTitle: "政策", EnTitle: "Policy"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg["layout"] != "reading" {
		t.Fatalf("layout=%v", cfg["layout"])
	}
	sections, ok := cfg["sections"].([]map[string]any)
	if !ok || len(sections) != 1 {
		t.Fatalf("sections=%v", cfg["sections"])
	}
	if sections[0]["type"] != "rich-text" {
		t.Fatalf("type=%v", sections[0]["type"])
	}
}

func TestBuildDocGuideHasHero(t *testing.T) {
	cfg, err := Build(DocGuide, Options{ZhTitle: "上手", EnTitle: "Start"})
	if err != nil {
		t.Fatal(err)
	}
	sections := cfg["sections"].([]map[string]any)
	if len(sections) < 2 || sections[0]["type"] != "hero" {
		t.Fatalf("expected hero first, got %#v", sections)
	}
	if sections[0]["variant"] != "compact" {
		t.Fatalf("variant=%v", sections[0]["variant"])
	}
}

func TestAllNonEmpty(t *testing.T) {
	if len(All()) != 3 {
		t.Fatalf("want 3 presets, got %d", len(All()))
	}
}
