package contentexcerpt

import (
	"strings"
	"testing"

	"github.com/yixian-huang/inkless/backend/internal/model"
)

func TestPlainExcerpt_PrefersMeta(t *testing.T) {
	got := plainExcerpt("<p>long body text here</p>", "meta desc", 160)
	if got != "meta desc" {
		t.Fatalf("expected meta, got %q", got)
	}
}

func TestPlainExcerpt_StripsHTML(t *testing.T) {
	got := plainExcerpt("<p>Hello <strong>world</strong> &amp; friends</p>", "", 160)
	if got != "Hello world & friends" {
		t.Fatalf("got %q", got)
	}
}

func TestPlainExcerpt_Truncates(t *testing.T) {
	body := strings.Repeat("字", 200)
	got := plainExcerpt("<p>"+body+"</p>", "", 50)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis, got %q", got)
	}
	// 50 runes + "..."
	runes := []rune(strings.TrimSuffix(got, "..."))
	if len(runes) > 50 {
		t.Fatalf("too long: %d runes", len(runes))
	}
}

func TestApplyListExcerpts(t *testing.T) {
	items := []*model.Article{
		{
			ZhBody:            "<p>中文正文内容用于列表摘要展示。</p>",
			ZhMetaDescription: "",
			EnBody:            "<p>English body for excerpt.</p>",
		},
	}
	ApplyListExcerpts(items)
	if items[0].ZhBody == "" || strings.Contains(items[0].ZhBody, "<p>") {
		t.Fatalf("zh body not excerpted: %q", items[0].ZhBody)
	}
	if items[0].EnBody != "English body for excerpt." {
		t.Fatalf("en body: %q", items[0].EnBody)
	}
}
