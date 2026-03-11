package seo_test

import (
	"testing"

	"blotting-consultancy/internal/seo"
)

func TestResolveHomePage(t *testing.T) {
	meta := seo.ResolveFromPath("/", "https://example.com", "zh")
	if meta.CanonicalURL != "https://example.com/" {
		t.Errorf("expected canonical /, got %q", meta.CanonicalURL)
	}
	if meta.OgURL != "https://example.com/" {
		t.Errorf("expected og:url /, got %q", meta.OgURL)
	}
	if meta.Locale != "zh" {
		t.Errorf("expected locale zh, got %q", meta.Locale)
	}
}

func TestResolveAboutPage(t *testing.T) {
	meta := seo.ResolveFromPath("/about", "https://example.com", "en")
	if meta.CanonicalURL != "https://example.com/about" {
		t.Errorf("expected canonical /about, got %q", meta.CanonicalURL)
	}
	if meta.Locale != "en" {
		t.Errorf("expected locale en, got %q", meta.Locale)
	}
}

func TestResolveArticlePath(t *testing.T) {
	meta := seo.ResolveFromPath("/blog/my-article", "https://example.com", "zh")
	if meta.CanonicalURL != "https://example.com/blog/my-article" {
		t.Errorf("expected canonical, got %q", meta.CanonicalURL)
	}
	if meta.OgType != "article" {
		t.Errorf("expected og:type article for blog path, got %q", meta.OgType)
	}
}

func TestResolveBlogIndex(t *testing.T) {
	meta := seo.ResolveFromPath("/blog/", "https://example.com", "zh")
	if meta.OgType != "website" {
		t.Errorf("expected og:type website for blog index, got %q", meta.OgType)
	}
}

func TestResolveTrailingSlashBaseURL(t *testing.T) {
	meta := seo.ResolveFromPath("/about", "https://example.com/", "en")
	if meta.CanonicalURL != "https://example.com/about" {
		t.Errorf("expected no double slash, got %q", meta.CanonicalURL)
	}
}

func TestResolveDefaultMeta(t *testing.T) {
	meta := seo.ResolveFromPath("/", "https://example.com", "zh")
	if meta.TwitterCard != "summary_large_image" {
		t.Errorf("expected default twitter card, got %q", meta.TwitterCard)
	}
	if meta.OgType != "website" {
		t.Errorf("expected default og:type website, got %q", meta.OgType)
	}
}
