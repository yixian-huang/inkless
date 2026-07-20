# Ops lesson: yx.ink ≠ inkless.run

**Incident date:** 2026-07-20  
**Host:** gomami (`103.73.220.161`)  
**Severity:** high (mutated personal production content while intending product-site cutover)

## What went wrong

An agent treated **DNS / Caddy hostnames** as the site boundary and rewrote the single shared SQLite DB + theme for “inkless.run branding.”  
At that moment both `yx.ink` and `inkless.run` reverse-proxied to **one** process (`inkless.service` on `:8088`) and **one** database (`/opt/inkless/data/inkless.db`).  
Changing theme / `content_documents` / features therefore corrupted the personal site. Recovery required restoring a pre-change DB backup.

## Root cause (mental model error)

| Wrong assumption | Correct model |
|---|---|
| “Domain = site” | **Process + data dir + port + unit** = site |
| “Same CMS binary → same instance is fine” | Binary may be shared; **runtime state must not** |
| “Brand upgrade = rewrite published config in place” | New product site = **new env / new tree / new DB** |
| “Caddy can host many names → one app is OK” | Multiple names on one upstream means **one site** until split |

Inkless is **single-instance / single-logical-site** in application code. Two public brands require **two processes** (or two machines), not two DNS records on one DB.

## Correct topology (as of recovery)

| Site | Role | systemd | Port | Data | Theme (typical) |
|---|---|---|---|---|---|
| **yx.ink** | Personal blog | `inkless.service` | `8088` | `/opt/inkless/data/inkless.db` | `blog-first` |
| **inkless.run** | Product ops | `inkless-ops.service` | `8089` | `/opt/inkless-ops/data/inkless.db` | `corporate-classic` |

Caddy must route:

- `yx.ink` → `127.0.0.1:8088`
- `inkless.run` / `www` → `127.0.0.1:8089`

Code artifacts **may** be shared via symlink (`backend/current`, `frontend/current`); **never** share `data/`, `uploads/`, `.env`, JWT secrets, or systemd `EnvironmentFile`.

## Hard rules for agents

1. **Before any write** (DB, theme, env, Caddy, deploy) on a public host, print and confirm:
   - unit name, listen port, `DB_DSN`, `BASE_URL`, reverse-proxy target for **each** domain involved.
2. **Never** run product/ops cutover scripts against `/opt/inkless` when the user means inkless.run product site. Use `/opt/inkless-ops` (or a dedicated env) only.
3. **Never** set personal-site `BASE_URL` / CORS to `inkless.run`, or product-site env to `yx.ink`, without an explicit multi-tenant design (this product does not have multi-tenant domains in one process).
4. **New domain for product** ⇒ assume **new process + new DB** until the user explicitly says “same instance.”
5. **Backup before DB mutation**; prefer timestamped copy next to the DB and verify restore path.
6. After changes, verify **both** sites still show the expected identity (title + `/public/bootstrap` `activeTheme` + `identity.name`).

## Recovery notes (if personal site is polluted again)

```bash
# example: restore yx.ink DB (personal)
systemctl stop inkless
cp -a /opt/inkless/data/inkless.db.bak-<timestamp> /opt/inkless/data/inkless.db
rm -f /opt/inkless/data/inkless.db-wal /opt/inkless/data/inkless.db-shm
systemctl start inkless
# verify https://yx.ink/public/bootstrap → blog-first / 黄逸仙
```

Product cutover helper (product DB only):

```bash
INKLESS_DB=/opt/inkless-ops/data/inkless.db python3 scripts/ops-product-site-cutover.py
```

Bootstrap second process:

```bash
bash scripts/ops-bootstrap-inkless-run.sh
```

## Related docs

- ADR: `docs/adr/0001-single-instance-single-site.md` (one process = one logical site)
- Brand migration: `docs/inkless-brand-migration.md` (product domain; does **not** authorize overwriting personal sites)
- Site config example for product: `ops/inkless-site-config.example.json`
