package seo_test

import (
	"encoding/json"
	"testing"

	"blotting-consultancy/internal/seo"
)

func TestOrganizationJSONLD(t *testing.T) {
	ld := seo.OrganizationJSONLD("Test Site", "https://example.com", "https://example.com/logo.png")
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(ld), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["@type"] != "Organization" {
		t.Errorf("expected @type Organization, got %v", m["@type"])
	}
	if m["name"] != "Test Site" {
		t.Errorf("expected name, got %v", m["name"])
	}
}

func TestArticleJSONLD(t *testing.T) {
	ld := seo.ArticleJSONLD("Title", "Desc", "https://example.com/blog/test", "https://example.com/img.png", "2026-01-01", "Author")
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(ld), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["@type"] != "Article" {
		t.Errorf("expected @type Article, got %v", m["@type"])
	}
}

func TestBreadcrumbJSONLD(t *testing.T) {
	items := []seo.BreadcrumbItem{
		{Name: "Home", URL: "https://example.com/"},
		{Name: "Blog", URL: "https://example.com/blog"},
		{Name: "Article", URL: "https://example.com/blog/test"},
	}
	ld := seo.BreadcrumbJSONLD(items)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(ld), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["@type"] != "BreadcrumbList" {
		t.Errorf("expected @type BreadcrumbList, got %v", m["@type"])
	}
}
