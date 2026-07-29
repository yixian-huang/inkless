---
name: inkless-ops
description: >
  Dogfood inkless.run product-ops content via local inkless CLI + fleet.
  Triggers: inkless.run, inkless-ops, ops SEO, dogfood, /inkless-ops.
---

# inkless-ops (project)

Same as user skill `~/.grok/skills/inkless-ops`. Prefer CLI over MCP.

```bash
source ~/.config/inkless/dogfood.sh
inkless-ops whoami
inkless-ops articles list --missing-seo --json
```

Rules: site `inkless-ops` only; whoami before write; dry-run first; no DB; `publish_policy=never`.

Full guide: `docs/agent-access.md`, host skill body under `~/.grok/skills/inkless-ops/SKILL.md`.

## Dynamic pages (presets)

Prefer multi-section host presets over a single HTML blob:

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
Publish is blocked by `publish_policy=never` on ops — draft only unless policy changes.

Content-type choice: theme page vs dynamic page vs article → omni `arch/inkless/content-type-decision-tree`.
