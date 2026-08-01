---
name: inkless-ops
description: >
  Dogfood and maintain the inkless.run product site via local inkless CLI
  (not MCP, not production DB). Project-scoped to the inkless monorepo only.
  Use when the user mentions inkless.run, inkless-ops, product ops content,
  SEO for the ops site, dogfood inkless, or /inkless-ops.
---

# inkless-ops (inkless.run · this repo)

Maintain **https://inkless.run** with the official CLI + fleet. Prefer CLI over MCP for token efficiency and auditability.

This skill lives under `inkless/.grok/skills/` so Grok only loads it when the
workspace is this monorepo — not in imgli, yixian-content content hub, or unrelated trees.

## Setup (this machine)

```bash
source ~/.config/inkless/dogfood.sh
# needs ~/.config/inkless/env.ops with INKLESS_KEY_OPS=ink_…
```

| Item | Value |
|------|--------|
| Fleet site id | `inkless-ops` |
| Base URL | https://inkless.run |
| Key env | `INKLESS_KEY_OPS` |
| `publish_policy` | **never** (CLI will not publish) |
| Binary | `~/go/bin/inkless` on PATH |

Create key: login ops site → **设置 → API Key** → preset **内容 Agent（推荐）** → `~/.config/inkless/env.ops`.

## Hard rules

1. **Only** hit site `inkless-ops` / `https://inkless.run` unless the user explicitly names another site.
2. **Never** write SQLite/Postgres or SSH-edit production DB for content work.
3. **Never** reuse ops key against yx.ink / imgli.com.
4. Before writes: `inkless-ops whoami` (or `inkless site whoami --site inkless-ops --fleet ~/.config/inkless/fleet.json`).
5. Writes: always **`--dry-run` first**, then apply after review.
6. Minimize tokens: `--json` + `jq`; do not paste full article bodies unless needed.
7. Prefer CLI; do not start MCP unless the user asks.

## Commands

```bash
source ~/.config/inkless/dogfood.sh

inkless-ops whoami
# expect: pages, pageTemplates (e.g. product-first/home@1), activeThemeId

inkless-ops articles list --missing-seo --json
inkless-ops articles get ID --json
# patch.json = partial fields only (zhSeoTitle, zhMetaDescription, …)
inkless-ops articles apply ID --from-file patch.json --dry-run
inkless-ops articles apply ID --from-file patch.json

inkless-ops pages list
inkless-ops pages get-draft ID

# product-first home (theme-as-templates: prefer pages + templates)
inkless templates list --site inkless-ops
inkless content migrate-to-pages --site inkless-ops   # once: content_documents → Page home
inkless pages list --site inkless-ops                 # should include slug=home + templateKey
inkless media upload ./shot.png --site inkless-ops --json
# Preferred write path:
inkless pages get-draft ID --json
inkless pages put-draft ID --from-file home.json --dry-run
# Legacy content apply still works (bridges to Page + deprecation warning):
inkless content apply home --site inkless-ops --from-file home.json --dry-run
```

Full form:

```bash
export PATH="$HOME/go/bin:$PATH"
export INKLESS_FLEET="$HOME/.config/inkless/fleet.json"
set -a && source "$HOME/.config/inkless/env.ops" && set +a

inkless site whoami --site inkless-ops
inkless articles list --site inkless-ops --missing-seo --json
```

## SEO patch shape

```json
{
  "zhSeoTitle": "…",
  "enSeoTitle": "…",
  "zhMetaDescription": "…",
  "enMetaDescription": "…"
}
```

`articles apply` merges onto the current article (safe PUT).

## Product-first home（theme-as-templates）

| 真源 | 说明 |
|------|------|
| **`unified_pages` slug=`home`** | 推荐运营真源（`templateKey` 如 `product-first/home@1`） |
| `content_documents.page_key=home` | 迁移期 / 公开双读回退 |
| **不是** articles | `articles list` 看不到 home |

**最小闭环：**

1. `whoami` → `pages: true`，`pageTemplates` 含 home 模板
2. `templates list` / `pages list` 确认 slug=home
3. `media upload` → 拿到 `url`
4. 编辑 `home.json`：`hero/showcase/features/…`；**MediaRef 的 url/alt/caption 必须是 string**
5. `pages put-draft ID --from-file home.json --dry-run` → 再 apply
6. 迁移期可用 `content apply home`（bridge + deprecation）
7. `publish_policy=never` → **不要** CLI publish；人工后台发

设计：`docs/design-theme-as-templates.md`、`docs/design-theme-content-admin-api.md`。

## Dynamic pages (presets)

| Preset | Use |
|--------|-----|
| `doc-simple` | Policy / short note (header + rich-text) |
| `doc-guide` | Getting started (compact hero + rich-text + checklist) |
| `landing-use-cases` | Marketing cases (hero + cards + rich-text) |

```bash
inkless pages presets --json
inkless pages create --site inkless-ops --slug get-started --preset doc-guide \
  --zh-title '快速开始' --en-title 'Get started' --dry-run
inkless pages apply-preset ID --site inkless-ops --preset doc-guide \
  --zh-title '快速开始' --from-file body.json --dry-run
```

`body.json` optional keys: `zhBody`, `enBody`, `zhSubtitle`, `enSubtitle` (HTML ok).  
Publish is blocked by `publish_policy=never` — draft only unless policy changes.

Content-type choice (theme page vs dynamic page vs article) → omni `arch/inkless/content-type-decision-tree`.

## Out of scope unless asked

- Publish (policy never)
- Deploy / npc / SSH (use nopanel-npc / deploy docs)
- yx.ink or imgli.com content (other project skills: `inkless-yx`, `inkless-imgli`)

## Related local skills (other trees)

| Site | Project path | Skill name |
|------|--------------|------------|
| yx.ink | `yixian-content/.grok/skills/inkless-yx` | `inkless-yx` |
| imgli.com | `imgli/.grok/skills/inkless-imgli` | `inkless-imgli` |
