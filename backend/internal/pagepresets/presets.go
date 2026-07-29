// Package pagepresets provides Host composable page recipes for CLI/agent
// (doc-simple, doc-guide, landing-use-cases). Mirrors frontend pagePresets.ts.
package pagepresets

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// ID is a host page preset key.
type ID string

const (
	DocSimple       ID = "doc-simple"
	DocGuide        ID = "doc-guide"
	LandingUseCases ID = "landing-use-cases"
)

// Meta describes a preset for listing.
type Meta struct {
	ID            ID     `json:"id"`
	LabelZh       string `json:"labelZh"`
	LabelEn       string `json:"labelEn"`
	DescriptionZh string `json:"descriptionZh"`
	DescriptionEn string `json:"descriptionEn"`
	Layout        string `json:"layout"`
}

// Options customize titles and body when building a preset.
type Options struct {
	ZhTitle    string
	EnTitle    string
	ZhSubtitle string
	EnSubtitle string
	ZhBody     string
	EnBody     string
}

// All returns preset metadata in stable order.
func All() []Meta {
	return []Meta{
		{
			ID: DocSimple, LabelZh: "简单文档", LabelEn: "Simple doc",
			DescriptionZh: "页眉 + 富文本，适合政策与短说明",
			DescriptionEn: "Page header + rich text for policies and short notes",
			Layout:        "reading",
		},
		{
			ID: DocGuide, LabelZh: "上手指南", LabelEn: "Guide",
			DescriptionZh: "短 Hero + 富文本 + 步骤 + CTA",
			DescriptionEn: "Compact hero + rich text + steps + CTA",
			Layout:        "landing",
		},
		{
			ID: LandingUseCases, LabelZh: "用例落地", LabelEn: "Use-cases landing",
			DescriptionZh: "Hero + 文字卡片 + 富文本 + CTA",
			DescriptionEn: "Hero + text cards + rich text + CTA",
			Layout:        "landing",
		},
	}
}

// ParseID validates a preset id.
func ParseID(s string) (ID, error) {
	id := ID(strings.TrimSpace(s))
	for _, m := range All() {
		if m.ID == id {
			return id, nil
		}
	}
	return "", fmt.Errorf("unknown page preset %q (want doc-simple|doc-guide|landing-use-cases)", s)
}

func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + "-" + hex.EncodeToString(b[:])
}

func bilingual(zh, en string) map[string]any {
	return map[string]any{"zh": zh, "en": en}
}

func defaultBody(zhTitle, enTitle string) (string, string) {
	zh := fmt.Sprintf(
		`<h2>概述</h2><p>在此编写「%s」的正文。支持标题、列表与链接。</p><ul><li>要点一</li><li>要点二</li></ul>`,
		zhTitle,
	)
	en := fmt.Sprintf(
		`<h2>Overview</h2><p>Write the body for “%s” here. Headings, lists, and links are supported.</p><ul><li>Point one</li><li>Point two</li></ul>`,
		enTitle,
	)
	return zh, en
}

// Build returns a draftConfig map: { layout, showPageHeader?, sections }.
func Build(id ID, opts Options) (map[string]any, error) {
	zhTitle := strings.TrimSpace(opts.ZhTitle)
	if zhTitle == "" {
		zhTitle = "未命名页面"
	}
	enTitle := strings.TrimSpace(opts.EnTitle)
	if enTitle == "" {
		enTitle = "Untitled page"
	}
	zhSub := strings.TrimSpace(opts.ZhSubtitle)
	enSub := strings.TrimSpace(opts.EnSubtitle)
	zhBody := strings.TrimSpace(opts.ZhBody)
	enBody := strings.TrimSpace(opts.EnBody)
	if zhBody == "" || enBody == "" {
		dz, de := defaultBody(zhTitle, enTitle)
		if zhBody == "" {
			zhBody = dz
		}
		if enBody == "" {
			enBody = de
		}
	}

	switch id {
	case DocSimple:
		return map[string]any{
			"layout":         "reading",
			"showPageHeader": true,
			"sections": []map[string]any{
				{
					"id": newID("rt"), "type": "rich-text", "variant": "default",
					"data": map[string]any{
						"content":   bilingual(zhBody, enBody),
						"alignment": "left",
					},
					"settings": map[string]any{
						"padding": "md", "maxWidth": "reading", "background": "surface",
					},
				},
				{
					"id": newID("faq"), "type": "faq", "variant": "default",
					"data": map[string]any{
						"title": bilingual("常见问题", "FAQ"),
						"items": []map[string]any{
							{
								"question": bilingual("如何修改本页？", "How do I edit this page?"),
								"answer":   bilingual("在管理端 → 页面中编辑草稿并发布。", "Edit the draft under Admin → Pages, then publish."),
							},
						},
					},
					"settings": map[string]any{
						"padding": "md", "maxWidth": "reading", "background": "surface",
					},
				},
			},
		}, nil

	case DocGuide:
		if zhSub == "" {
			zhSub = "分步说明与相关资源"
		}
		if enSub == "" {
			enSub = "Steps and related resources"
		}
		return map[string]any{
			"layout":         "landing",
			"showPageHeader": false,
			"sections": []map[string]any{
				{
					"id": newID("hero"), "type": "hero", "variant": "compact",
					"data": map[string]any{
						"title":           bilingual(zhTitle, enTitle),
						"subtitle":        bilingual(zhSub, enSub),
						"backgroundColor": "#0f172a",
					},
					"settings": map[string]any{
						"padding": "none", "maxWidth": "full", "background": "surface",
					},
				},
				{
					"id": newID("rt"), "type": "rich-text", "variant": "default",
					"data": map[string]any{
						"content": bilingual(zhBody, enBody), "alignment": "left",
					},
					"settings": map[string]any{
						"padding": "md", "maxWidth": "reading", "background": "surface",
					},
				},
				{
					"id": newID("steps"), "type": "steps", "variant": "default",
					"data": map[string]any{
						"title": bilingual("推荐步骤", "Suggested steps"),
						"steps": []map[string]any{
							{
								"title":       bilingual("确认环境", "Prepare environment"),
								"description": bilingual("准备服务器或本机开发环境", "Server or local dev environment"),
							},
							{
								"title":       bilingual("完成配置", "Configure"),
								"description": bilingual("站点身份、主题与基础开关", "Identity, theme, and feature toggles"),
							},
							{
								"title":       bilingual("验证公开页", "Verify public pages"),
								"description": bilingual("检查首页与关键路由", "Check home and key routes"),
							},
						},
					},
					"settings": map[string]any{
						"padding": "md", "maxWidth": "layout", "background": "surface-alt",
					},
				},
				{
					"id": newID("cta"), "type": "cta", "variant": "default",
					"data": map[string]any{
						"title":          bilingual("下一步", "Next steps"),
						"subtitle":       bilingual("浏览能力页或继续完善内容", "Explore features or keep refining content"),
						"primaryLabel":   bilingual("查看能力", "View features"),
						"primaryHref":    "/features",
						"secondaryLabel": bilingual("面向 Agent", "For agents"),
						"secondaryHref":  "/p/agents",
					},
					"settings": map[string]any{
						"padding": "lg", "maxWidth": "layout", "background": "surface",
					},
				},
			},
		}, nil

	case LandingUseCases:
		if zhSub == "" {
			zhSub = "适用场景与价值"
		}
		if enSub == "" {
			enSub = "Where it fits and why"
		}
		return map[string]any{
			"layout":         "landing",
			"showPageHeader": false,
			"sections": []map[string]any{
				{
					"id": newID("hero"), "type": "hero", "variant": "compact",
					"data": map[string]any{
						"title": bilingual(zhTitle, enTitle), "subtitle": bilingual(zhSub, enSub),
						"backgroundColor": "#0f172a",
					},
					"settings": map[string]any{
						"padding": "none", "maxWidth": "full", "background": "surface",
					},
				},
				{
					"id": newID("cards"), "type": "card-grid", "variant": "default",
					"data": map[string]any{
						"title":           bilingual("典型场景", "Use cases"),
						"columns":         3,
						"preferTextCards": true,
						"cards": []map[string]any{
							{
								"title":       bilingual("个人博客", "Personal blog"),
								"description": bilingual("文章流 + 轻量页面", "Article stream with light pages"),
							},
							{
								"title":       bilingual("产品运营站", "Product site"),
								"description": bilingual("主题页 + 动态页扩展", "Theme IA plus dynamic pages"),
							},
							{
								"title":       bilingual("内容团队", "Content team"),
								"description": bilingual("SEO 与 Agent 协作维护", "SEO and agent-assisted ops"),
							},
						},
					},
					"settings": map[string]any{
						"padding": "lg", "maxWidth": "layout", "background": "surface",
					},
				},
				{
					"id": newID("rt"), "type": "rich-text", "variant": "default",
					"data": map[string]any{
						"content": bilingual(zhBody, enBody), "alignment": "left",
					},
					"settings": map[string]any{
						"padding": "md", "maxWidth": "reading", "background": "surface",
					},
				},
				{
					"id": newID("cta"), "type": "cta", "variant": "default",
					"data": map[string]any{
						"title":          bilingual("开始搭建", "Start building"),
						"primaryLabel":   bilingual("快速开始", "Get started"),
						"primaryHref":    "/p/get-started",
						"secondaryLabel": bilingual("能力一览", "Features"),
						"secondaryHref":  "/features",
					},
					"settings": map[string]any{
						"padding": "lg", "maxWidth": "layout", "background": "surface",
					},
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown page preset %q", id)
	}
}
