#!/usr/bin/env python3
"""P0 content pack for inkless.run product ops DB only."""
from __future__ import annotations

import json
import shutil
import sqlite3
import time
from datetime import datetime, timezone
from pathlib import Path

DB = Path("/opt/inkless-ops/data/inkless.db")
DOCS_URL = "https://docs.inkless.run"


def main() -> None:
    if not DB.exists() or "inkless-ops" not in str(DB):
        raise SystemExit(f"refuse: {DB}")
    bak = DB.with_name(f"inkless.db.bak-p0-content-{int(time.time())}")
    shutil.copy2(DB, bak)
    print("backup", bak)

    con = sqlite3.connect(str(DB))
    con.row_factory = sqlite3.Row
    cur = con.cursor()
    now = datetime.now(timezone.utc).isoformat()

    def upsert_content(key: str, payload: dict) -> None:
        s = json.dumps(payload, ensure_ascii=False)
        row = cur.execute(
            "select published_version, draft_version from content_documents where page_key=?",
            (key,),
        ).fetchone()
        if row:
            cur.execute(
                "update content_documents set draft_config=?, published_config=?, "
                "draft_version=?, published_version=?, updated_at=? where page_key=?",
                (s, s, row["draft_version"] + 1, row["published_version"] + 1, now, key),
            )
            print("content", key, "->", row["published_version"] + 1)
        else:
            cur.execute(
                "insert into content_documents "
                "(page_key, draft_config, draft_version, published_config, published_version, updated_at) "
                "values (?,?,?,?,?,?)",
                (key, s, 1, s, 1, now),
            )
            print("content created", key)

    global_cfg = {
        "identity": {
            "name": {"zh": "Inkless", "en": "Inkless"},
            "tagline": {
                "zh": "无墨，内容自有形态",
                "en": "Content without constraints",
            },
            "localeMode": "bilingual",
            "defaultLocale": "zh",
        },
        "brand": {
            "logo": {
                "light": "/brand/inkless-wordmark.svg",
                "dark": "/brand/inkless-wordmark.svg",
            },
            "favicon": "/brand/favicon.svg",
            "ogImage": "/uploads/product/02-dashboard.png",
            "primaryColor": "#0f172a",
            "accentColor": "#2563eb",
        },
        "author": {
            "name": "Inkless CMS",
            "avatar": "/brand/inkless-mark.svg",
            "bio": {
                "zh": "开源自托管 CMS：主题驱动的产品站、博客与内容运营。",
                "en": "Open-source self-hosted CMS for theme-driven product sites and content ops.",
            },
            "socials": [
                {"kind": "github", "url": "https://github.com/yixian-huang/inkless"},
                {"kind": "email", "url": "mailto:hello@inkless.run"},
            ],
        },
        "header": {
            "brandMode": "logo",
            "showSocials": False,
            "showRssLink": False,
        },
        "footer": {
            "copyright": {
                "zh": "© 2026 Inkless · inkless.run",
                "en": "© 2026 Inkless · inkless.run",
            }
        },
        "seo": {
            "defaultTitle": {
                "zh": "Inkless · 内容管理系统",
                "en": "Inkless · Content Management System",
            },
            "titleTemplate": "{page} · Inkless",
            "defaultDescription": {
                "zh": "无墨，内容自有形态。开源自托管 CMS：官方主题市场一键安装，主题驱动产品站与内容运营。",
                "en": "Content without constraints. Open-source self-hosted CMS with an official theme market and theme-driven product sites.",
            },
            "twitterHandle": "",
        },
    }
    upsert_content("global", global_cfg)

    home = {
        "hero": {
            "eyebrow": {"zh": "Inkless CMS · v0.1.1", "en": "Inkless CMS · v0.1.1"},
            "title": {
                "zh": "用主题驱动你的产品站与内容运营",
                "en": "Theme-driven product sites and content ops",
            },
            "subtitle": {
                "zh": "无墨，内容自有形态。开源自托管 CMS：管理端官方主题市场一键安装；产品介绍、博客与企业站由主题决定呈现。",
                "en": "Content without constraints. Open-source self-hosted CMS — install official themes from the admin market; product, blog, or corporate shape is owned by themes.",
            },
            "primaryCta": {
                "label": {"zh": "快速开始", "en": "Get started"},
                "href": "#install",
            },
            "secondaryCta": {
                "label": {"zh": "GitHub", "en": "GitHub"},
                "href": "https://github.com/yixian-huang/inkless",
            },
            "media": {
                "url": "/uploads/product/02-dashboard.png",
                "alt": "Inkless admin dashboard",
                "caption": "",
            },
        },
        "showcase": {
            "title": {"zh": "产品界面", "en": "Product interface"},
            "items": [
                {
                    "url": "/uploads/product/03-pages.png",
                    "alt": "Pages",
                    "caption": "Pages",
                },
                {
                    "url": "/uploads/product/05-theme.png",
                    "alt": "Themes",
                    "caption": "Themes",
                },
                {
                    "url": "/uploads/product/04-articles.png",
                    "alt": "Articles",
                    "caption": "Articles",
                },
            ],
        },
        "features": {
            "title": {"zh": "核心能力", "en": "Capabilities"},
            "items": [
                {
                    "title": {"zh": "官方主题市场", "en": "Official theme market"},
                    "description": {
                        "zh": "管理端浏览官方 catalog，一键安装 product-first / blog-first 等主题，无需手填 UMD。",
                        "en": "Browse the official catalog in admin and one-click install themes — no hand-pasted UMD URLs.",
                    },
                },
                {
                    "title": {"zh": "主题形态", "en": "Theme shapes"},
                    "description": {
                        "zh": "product-first / blog-first / editorial 等契约主题，切换呈现不改内容模型。",
                        "en": "Contracted themes (product-first, blog-first, editorial, …) change presentation without a second content model.",
                    },
                },
                {
                    "title": {"zh": "内容运营", "en": "Content ops"},
                    "description": {
                        "zh": "页面、文章、媒体与发布版本，适合持续运营。",
                        "en": "Pages, articles, media, and versioned publish.",
                    },
                },
                {
                    "title": {"zh": "自托管与版本产物", "en": "Self-hosted releases"},
                    "description": {
                        "zh": "GitHub Release 提供 backend/frontend 产物；可选在系统状态检查更新并升级本实例。",
                        "en": "GitHub Releases ship backend/frontend artifacts; optionally check and apply updates for this instance in System status.",
                    },
                },
            ],
        },
        "howItWorks": {
            "title": {"zh": "如何开始", "en": "How it works"},
            "steps": [
                {
                    "title": {"zh": "部署实例", "en": "Deploy"},
                    "description": {
                        "zh": "自托管：本地 make dev-up，或使用 Release artifact / 你的部署流水线。",
                        "en": "Self-host with make dev-up, Release artifacts, or your deploy pipeline.",
                    },
                },
                {
                    "title": {"zh": "安装官方主题", "en": "Install a theme"},
                    "description": {
                        "zh": "打开管理端「主题市场」，一键安装并激活（如 product-first）。",
                        "en": "Open Theme market in admin, one-click install and activate (e.g. product-first).",
                    },
                },
                {
                    "title": {"zh": "发布内容", "en": "Publish"},
                    "description": {
                        "zh": "配置品牌与页面，对外发布；需要时再升级 host 或主题包。",
                        "en": "Configure brand and pages, publish, then upgrade host or theme packages when needed.",
                    },
                },
            ],
        },
        "install": {
            "title": {"zh": "快速开始", "en": "Quick start"},
            "code": "git clone https://github.com/yixian-huang/inkless.git\ncd inkless && make dev-up",
            "caption": {
                "zh": "本地开发默认打开 http://localhost:3000 。Release 见 GitHub v0.1.1；主题 catalog 示例 https://inkless.run/marketplace/v1/themes.json",
                "en": "Local dev serves http://localhost:3000. Releases: GitHub v0.1.1; sample theme catalog https://inkless.run/marketplace/v1/themes.json",
            },
        },
        "bottomCta": {
            "title": {"zh": "开始构建你的产品站", "en": "Build your product site"},
            "subtitle": {
                "zh": "无墨，内容自有形态。官方主题市场已开放一键安装。",
                "en": "Content without constraints. Official theme market with one-click install.",
            },
            "primaryCta": {
                "label": {"zh": "查看源码", "en": "View source"},
                "href": "https://github.com/yixian-huang/inkless",
            },
        },
    }
    upsert_content("home", home)

    theme_cfg = {
        "header": {
            "brandMode": "logo",
            "docsUrl": DOCS_URL,
            "githubUrl": "https://github.com/yixian-huang/inkless",
            "primaryCtaLabel": "Get started",
            "primaryCtaHref": "#install",
            "showRssLink": False,
            "showSocials": False,
        }
    }
    cur.execute(
        "update installed_themes set config=?, updated_at=? where theme_id=?",
        (json.dumps(theme_cfg, ensure_ascii=False), now, "product-first"),
    )
    print("docsUrl", DOCS_URL)

    tokens = {
        "colors": {
            "primary": "#0f172a",
            "primaryDark": "#020617",
            "accent": "#2563eb",
            "accentHover": "#1d4ed8",
            "surface": "#ffffff",
            "surfaceAlt": "#f8fafc",
            "onPrimary": "#f8fafc",
            "onSurface": "#0f172a",
            "onSurfaceMuted": "#64748b",
            "border": "#e2e8f0",
        },
        "fonts": {
            "sans": "ui-sans-serif, system-ui, -apple-system, \"Segoe UI\", sans-serif",
            "heading": "ui-sans-serif, system-ui, -apple-system, \"Segoe UI\", sans-serif",
            "mono": "ui-monospace, SF Mono, Menlo, Monaco, Consolas, monospace",
        },
        "layout": {
            "maxWidth": "72rem",
            "borderRadius": "0.75rem",
            "contentPadding": "1.5rem",
            "sectionSpacing": "5rem",
            "contentGap": "2rem",
        },
    }
    ts = json.dumps(tokens)
    row = cur.execute("select id from site_configs where key=?", ("theme",)).fetchone()
    if row:
        cur.execute(
            "update site_configs set draft_config=?, published_config=?, "
            "draft_version=draft_version+1, published_version=published_version+1, updated_at=? where key=?",
            (ts, ts, now, "theme"),
        )
    else:
        cur.execute(
            "insert into site_configs (key, draft_config, draft_version, published_config, published_version, created_at, updated_at) "
            "values (?,?,?,?,?,?,?)",
            ("theme", ts, 1, ts, 1, now, now),
        )
    print("theme tokens ok")
    con.commit()
    con.close()
    print("P0 content done")


if __name__ == "__main__":
    main()
