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
