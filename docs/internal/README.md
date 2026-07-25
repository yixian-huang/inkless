# Internal operator notes

This directory holds **maintainer-only** runbooks and incident notes for people
who operate public Inkless instances (product site, demos, migrations).

## Rules

1. **Do not put secrets here** — no API keys, JWT values, passwords, or private keys.
2. **Prefer inventory from the control plane** — use NoPanel / Quick-Box handoff
   (`npc server list`, `npc server handoff brief …`) instead of hard-coding IPs,
   project UUIDs, or hostnames in git.
3. **End-user docs stay outside** — public deployment docs live in
   [`docs/deployment.md`](../deployment.md), [`docs/docker-setup.md`](../docker-setup.md),
   and the VitePress site under [`docs-site/`](../../docs-site/).
4. **Customer-specific procedures** may live here temporarily; strip client names
   and addresses before sharing screenshots or third-party forks.

## Contents

| File | Purpose |
|------|---------|
| [`operator-runbook.md`](operator-runbook.md) | Maintainer deploy topology (placeholders only) |
| [`ops-lessons-site-isolation.md`](ops-lessons-site-isolation.md) | Multi-site isolation incident lesson |
| [`ops-theme-demo.md`](ops-theme-demo.md) | Theme demo instance notes |
| [`runbook-legacy-customer-upgrade.md`](runbook-legacy-customer-upgrade.md) | Generic legacy-host upgrade checklist |

If this repository is used as a pure upstream fork, you can delete `docs/internal/`
entirely; nothing in the product runtime depends on it.
