#!/usr/bin/env python3
"""One-shot: cut over production DB from personal blog-first to product ops site."""
from __future__ import annotations

import json
import shutil
import sqlite3
import time
from datetime import datetime, timezone
from pathlib import Path

DB_PATH = Path(__import__("os").environ.get("INKLESS_DB", "/opt/inkless-ops/data/inkless.db"))


def main() -> None:
    if not DB_PATH.exists():
        raise SystemExit(f"db not found: {DB_PATH}")
    bak = DB_PATH.with_name(f"{DB_PATH.name}.bak-product-cutover-{int(time.time())}")
    shutil.copy2(DB_PATH, bak)
    print("backup", bak)
    print("db", DB_PATH)

    con = sqlite3.connect(str(DB_PATH))
    con.row_factory = sqlite3.Row
    cur = con.cursor()
    now = datetime.now(timezone.utc).isoformat()

    themes = [
        {
            "theme_id": "corporate-classic",
            "name": "Corporate Classic",
            "name_zh": "企业经典",
            "description": "专业产品官网，含首页、关于、优势、服务、案例、专家、联系",
            "preview": "linear-gradient(135deg, #1a5f8f 0%, #8bc34a 100%)",
        },
        {
            "theme_id": "minimal-starter",
            "name": "Minimal Starter",
            "name_zh": "极简起步",
            "description": "最简内置主题",
            "preview": "linear-gradient(135deg, #374151 0%, #9ca3af 100%)",
        },
        {
            "theme_id": "blog-first",
            "name": "Blog First",
            "name_zh": "博客优先",
            "description": "极简个人博客",
            "preview": "linear-gradient(135deg, #44403c 0%, #a8a29e 100%)",
        },
    ]
    for t in themes:
        row = cur.execute(
            "select id from installed_themes where theme_id=?", (t["theme_id"],)
        ).fetchone()
        if row:
            cur.execute(
                "update installed_themes set name=?, name_zh=?, description=?, is_active=0, updated_at=? where theme_id=?",
                (t["name"], t["name_zh"], t["description"], now, t["theme_id"]),
            )
            print("theme exists", t["theme_id"])
        else:
            cur.execute(
                """insert into installed_themes
                (theme_id, name, name_zh, description, author, version, source, external_url, is_active, preview, config, created_at, updated_at)
                values (?,?,?,?,?,?,?,?,?,?,?,?,?)""",
                (
                    t["theme_id"],
                    t["name"],
                    t["name_zh"],
                    t["description"],
                    "Inkless CMS",
                    "1.0.0",
                    "built-in",
                    "",
                    0,
                    t["preview"],
                    "{}",
                    now,
                    now,
                ),
            )
            print("theme created", t["theme_id"])

    cur.execute("update installed_themes set is_active=0")
    cur.execute(
        "update installed_themes set is_active=1, updated_at=? where theme_id=?",
        (now, "corporate-classic"),
    )
    print("active", cur.execute("select theme_id,is_active from installed_themes").fetchall())

    pages = [
        ("home", "home", 0, {"zh": "首页", "en": "Home"}, {"showInHeader": True, "showInFooter": True}),
        ("about", "about", 1, {"zh": "产品", "en": "Product"}, {"showInHeader": True, "showInFooter": True}),
        ("advantages", "advantages", 2, {"zh": "能力", "en": "Capabilities"}, {"showInHeader": True, "showInFooter": True}),
        ("core-services", "core-services", 3, {"zh": "方案", "en": "Solutions"}, {"showInHeader": True, "showInFooter": True}),
        ("cases", "cases", 4, {"zh": "案例", "en": "Cases"}, {"showInHeader": True, "showInFooter": False}),
        ("experts", "experts", 5, {"zh": "团队", "en": "Team"}, {"showInHeader": False, "showInFooter": False}),
        ("contact", "contact", 6, {"zh": "联系", "en": "Contact"}, {"showInHeader": True, "showInFooter": True}),
    ]
    for slug, content_key, sort_order, title, nav in pages:
        title_b = json.dumps(title, ensure_ascii=False)
        nav_b = json.dumps(nav, ensure_ascii=False)
        existing = cur.execute(
            "select id from pages where slug=? and deleted_at is null", (slug,)
        ).fetchone()
        if existing:
            cur.execute(
                """update pages set theme_id=?, content_key=?, render_mode=?, is_theme_page=1,
                   title=?, nav_config=?, status=?, sort_order=?, updated_at=? where id=?""",
                (
                    "corporate-classic",
                    content_key,
                    "hardcoded",
                    title_b,
                    nav_b,
                    "published",
                    sort_order,
                    now,
                    existing["id"],
                ),
            )
            print("page reassigned", slug)
        else:
            cur.execute(
                """insert into pages
                (slug, parent_id, title, template, config, status, sort_order, seo_title, seo_description, keywords,
                 theme_id, content_key, render_mode, is_theme_page, nav_config, cover_image, auto_summary, allow_comments,
                 pinned, visibility, metadata, created_at, updated_at)
                values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)""",
                (
                    slug,
                    None,
                    title_b,
                    "default",
                    "{}",
                    "published",
                    sort_order,
                    "{}",
                    "{}",
                    "{}",
                    "corporate-classic",
                    content_key,
                    "hardcoded",
                    1,
                    nav_b,
                    "",
                    0,
                    0,
                    0,
                    "public",
                    "{}",
                    now,
                    now,
                ),
            )
            print("page created", slug)

    features = {
        "publicPages": {
            "home": True,
            "blog": False,
            "contact": True,
            "about": True,
            "experts": False,
            "coreServices": True,
            "advantages": True,
            "cases": True,
        },
        "blog": {"comments": False, "rss": False},
    }
    features_s = json.dumps(features, ensure_ascii=False, separators=(",", ":"))
    row = cur.execute("select id from site_configs where key=?", ("features",)).fetchone()
    if row:
        cur.execute(
            "update site_configs set draft_config=?, published_config=?, draft_version=draft_version+1, published_version=published_version+1, updated_at=? where key=?",
            (features_s, features_s, now, "features"),
        )
    else:
        cur.execute(
            "insert into site_configs (key, draft_config, draft_version, published_config, published_version, created_at, updated_at) values (?,?,?,?,?,?,?)",
            ("features", features_s, 1, features_s, 1, now, now),
        )
    print("features updated")

    theme_tokens = {
        "colors": {
            "primary": "#111827",
            "primaryDark": "#0b1220",
            "accent": "#14b8a6",
            "accentHover": "#0d9488",
            "surface": "#ffffff",
            "surfaceAlt": "#f8fafc",
            "onPrimary": "#ffffff",
            "onSurface": "#111827",
            "onSurfaceMuted": "#64748b",
            "border": "#e2e8f0",
        },
        "fonts": {
            "sans": 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif',
            "heading": 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif',
            "mono": "ui-monospace, SF Mono, Menlo, Monaco, Consolas, monospace",
        },
        "layout": {
            "maxWidth": "1200px",
            "borderRadius": "0.5rem",
            "contentPadding": "1.5rem",
            "sectionSpacing": "4rem",
            "contentGap": "2rem",
        },
    }
    theme_s = json.dumps(theme_tokens, ensure_ascii=False)
    row = cur.execute("select id from site_configs where key=?", ("theme",)).fetchone()
    if row:
        cur.execute(
            "update site_configs set draft_config=?, published_config=?, draft_version=draft_version+1, published_version=published_version+1, updated_at=? where key=?",
            (theme_s, theme_s, now, "theme"),
        )
    else:
        cur.execute(
            "insert into site_configs (key, draft_config, draft_version, published_config, published_version, created_at, updated_at) values (?,?,?,?,?,?,?)",
            ("theme", theme_s, 1, theme_s, 1, now, now),
        )
    print("theme tokens updated")

    global_cfg = {
        "identity": {
            "name": {"zh": "Inkless", "en": "Inkless"},
            "tagline": {
                "zh": "现代可扩展的内容管理系统",
                "en": "A modern, extensible content management system",
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
            "ogImage": "/brand/og-default.png",
            "primaryColor": "#111827",
            "accentColor": "#14b8a6",
        },
        "author": {
            "name": "Inkless CMS",
            "avatar": "",
            "bio": {
                "zh": "Inkless 是面向团队的内容与站点运营平台：主题、页面、媒体与发布工作流一站完成。",
                "en": "Inkless is a content and site operations platform for teams.",
            },
            "socials": [
                {"kind": "github", "url": "https://github.com/yixian-huang/inkless"},
            ],
        },
        "header": {
            "brandMode": "logo",
            "showSocials": True,
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
                "zh": "Inkless 是现代、可扩展的 CMS：用主题与配置驱动的页面，快速运营产品官网与内容站。",
                "en": "Inkless is a modern, extensible CMS for product sites and content operations.",
            },
            "twitterHandle": "",
        },
    }
    global_s = json.dumps(global_cfg, ensure_ascii=False)
    row = cur.execute(
        "select page_key, published_version, draft_version from content_documents where page_key=?",
        ("global",),
    ).fetchone()
    if row:
        cur.execute(
            "update content_documents set draft_config=?, published_config=?, draft_version=?, published_version=?, updated_at=? where page_key=?",
            (global_s, global_s, row["draft_version"] + 1, row["published_version"] + 1, now, "global"),
        )
        print("global updated", row["published_version"] + 1)
    else:
        cur.execute(
            "insert into content_documents (page_key, draft_config, draft_version, published_config, published_version, updated_at) values (?,?,?,?,?,?)",
            ("global", global_s, 1, global_s, 1, now),
        )
        print("global created")

    content_docs = {
        "home": {
            "hero": {
                "title": {"zh": "用 Inkless 运营你的产品站", "en": "Run your product site with Inkless"},
                "subtitle": {
                    "zh": "主题驱动的 CMS：页面、内容、媒体与发布流程一体化，适合产品官网与运营站点。",
                    "en": "Theme-driven CMS for product websites and content operations.",
                },
                "backgroundImage": {
                    "url": "/images/hero-bg.jpg",
                    "alt": {"zh": "Inkless", "en": "Inkless"},
                },
            },
            "about": {
                "title": {"zh": "为什么是 Inkless", "en": "Why Inkless"},
                "descriptions": [
                    {
                        "zh": "不是个人博客模板，而是可扩展的内容平台：内置主题、插件接口、双语与运营后台。",
                        "en": "Not a personal blog template — an extensible content platform.",
                    }
                ],
                "image": {"url": "/images/about.jpg", "alt": {"zh": "产品", "en": "Product"}},
                "cta": {
                    "label": {"zh": "了解产品", "en": "Learn more"},
                    "href": "/about",
                    "target": "_self",
                },
            },
            "advantages": {
                "title": {"zh": "核心能力", "en": "Capabilities"},
                "cards": [
                    {
                        "title": {"zh": "主题系统", "en": "Themes"},
                        "description": {
                            "zh": "corporate / blog / 自定义主题一键切换",
                            "en": "Swap corporate, blog, or custom themes",
                        },
                    },
                    {
                        "title": {"zh": "内容运营", "en": "Content ops"},
                        "description": {
                            "zh": "页面、文章、媒体与发布版本管理",
                            "en": "Pages, articles, media, versioned publish",
                        },
                    },
                    {
                        "title": {"zh": "可扩展", "en": "Extensible"},
                        "description": {
                            "zh": "插件与模块化 API，适配你的业务",
                            "en": "Plugins and modular APIs",
                        },
                    },
                ],
            },
            "coreServices": {
                "title": {"zh": "适用场景", "en": "Use cases"},
                "items": [
                    {
                        "title": {"zh": "产品官网", "en": "Product marketing"},
                        "description": {
                            "zh": "落地页、能力页、案例与联系转化",
                            "en": "Landing, capabilities, cases, contact",
                        },
                    },
                    {
                        "title": {"zh": "文档与博客", "en": "Docs & blog"},
                        "description": {
                            "zh": "可选开启内容频道，不绑架站点形态",
                            "en": "Optional content channels",
                        },
                    },
                    {
                        "title": {"zh": "自托管运营", "en": "Self-hosted ops"},
                        "description": {
                            "zh": "artifact 部署、systemd、自有域名",
                            "en": "Artifact deploy, systemd, your domain",
                        },
                    },
                ],
            },
        },
        "about": {
            "hero": {
                "title": {"zh": "产品介绍", "en": "Product"},
                "backgroundImage": {
                    "url": "/images/about-hero.jpg",
                    "alt": {"zh": "Inkless", "en": "Inkless"},
                },
            },
            "companyProfile": {
                "title": {"zh": "Inkless CMS", "en": "Inkless CMS"},
                "content": {
                    "zh": "Inkless 是开源 CMS：用配置驱动页面与主题，帮助团队快速上线产品运营站点，并保留自托管与扩展能力。",
                    "en": "Inkless is an open-source CMS for product and content sites.",
                },
            },
            "blocks": [],
        },
        "advantages": {"hero": {"title": {"zh": "产品能力", "en": "Capabilities"}}, "blocks": []},
        "core-services": {
            "hero": {"title": {"zh": "方案与接入", "en": "Solutions"}},
            "services": [
                {
                    "title": {"zh": "开源自托管", "en": "Open source self-host"},
                    "description": {
                        "zh": "GitHub: yixian-huang/inkless",
                        "en": "GitHub: yixian-huang/inkless",
                    },
                },
                {
                    "title": {"zh": "域名 inkless.run", "en": "inkless.run"},
                    "description": {
                        "zh": "本站即产品运营入口",
                        "en": "This site is the product ops entry",
                    },
                },
            ],
        },
        "cases": {"hero": {"title": {"zh": "案例", "en": "Cases"}}, "cases": []},
        "experts": {
            "hero": {"title": {"zh": "团队", "en": "Team"}},
            "sectionTitle": {"zh": "团队", "en": "Team"},
            "experts": [],
        },
        "contact": {
            "hero": {"title": {"zh": "联系与试用", "en": "Contact"}},
            "contactInfo": {"email": "hello@inkless.run", "phone": ""},
        },
    }
    for key, cfg in content_docs.items():
        s = json.dumps(cfg, ensure_ascii=False)
        row = cur.execute(
            "select page_key, published_version, draft_version from content_documents where page_key=?",
            (key,),
        ).fetchone()
        if row:
            cur.execute(
                "update content_documents set draft_config=?, published_config=?, draft_version=?, published_version=?, updated_at=? where page_key=?",
                (s, s, row["draft_version"] + 1, row["published_version"] + 1, now, key),
            )
            print("content updated", key)
        else:
            cur.execute(
                "insert into content_documents (page_key, draft_config, draft_version, published_config, published_version, updated_at) values (?,?,?,?,?,?)",
                (key, s, 1, s, 1, now),
            )
            print("content created", key)

    con.commit()
    print("VERIFY active", cur.execute("select theme_id,is_active from installed_themes where is_active=1").fetchone())
    print(
        "VERIFY pages",
        cur.execute(
            "select slug,theme_id,status from pages where deleted_at is null order by sort_order"
        ).fetchall(),
    )
    g = json.loads(
        cur.execute(
            "select published_config from content_documents where page_key=?", ("global",)
        ).fetchone()[0]
    )
    print("VERIFY title", g["seo"]["defaultTitle"])
    print(
        "VERIFY features",
        cur.execute("select published_config from site_configs where key=?", ("features",)).fetchone()[0][
            :180
        ],
    )
    con.close()
    print("DONE")


if __name__ == "__main__":
    main()
