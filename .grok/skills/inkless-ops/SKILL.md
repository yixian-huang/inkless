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
# expect: themeContent, activeThemeId, contentSlots, themeContentKeys

inkless-ops articles list --missing-seo --json
inkless-ops articles get ID --json
# patch.json = partial fields only (zhSeoTitle, zhMetaDescription, …)
inkless-ops articles apply ID --from-file patch.json --dry-run
inkless-ops articles apply ID --from-file patch.json

inkless-ops pages list
inkless-ops pages get-draft ID

# product-first home (theme content slot — NOT pages list)
inkless content slots --site inkless-ops          # activeTheme + contentSlots
inkless content schema home --site inkless-ops    # schemaId / mediaRefPaths
inkless content keys --site inkless-ops
inkless content get home --site inkless-ops --json
inkless media upload ./shot.png --site inkless-ops --json
inkless content apply home --site inkless-ops --from-file home.json --dry-run --validate-schema
# dry-run: diff.paths / localMediaIssues / validate.schemaSource=theme
inkless content apply home --site inkless-ops --from-file home.json --validate-schema
inkless content get home --site inkless-ops --public --locale zh --json
inkless content versions home --site inkless-ops
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

## Product-first home（主题内容槽）

| 真源 | 说明 |
|------|------|
| `content_documents.page_key=home` | product-first 落地页文案/配图 |
| **不是** articles | `articles list` 看不到 home |
| **不是** unified pages | `pages list` 通常为空/无 home |

**最小闭环：**

1. `whoami` → `themeContent: true`
2. `media upload` → 拿到 `url`
3. 编辑 `home.json`：`hero/showcase/features/howItWorks/install/bottomCta`；**MediaRef 的 url/alt/caption 必须是 string**（禁止 `{zh,en}`）
4. `content apply home --from-file home.json --dry-run` → 看 diff + validate
5. `content apply home --from-file home.json` → 写 draft
6. `content get home --public` + 浏览器 smoke（防 React #31）
7. `publish_policy=never` → **不要** `content publish`；人工在后台发，或改 policy 后再发

无 content API 的旧 host：**禁止写生产库**；请升级到含 theme content Admin API 的版本。

设计：`docs/design-theme-content-admin-api.md`。

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
