# Ops lesson: domain ≠ process (site isolation)

**Severity:** high (mutated site A content while intending site B cutover)

## What went wrong

An operator (or agent) treated **DNS / reverse-proxy hostnames** as the site
boundary and rewrote a single shared SQLite DB + theme for “product branding.”
At that moment two public hostnames reverse-proxied to **one** process and
**one** database. Changing theme / published content therefore corrupted the
other site. Recovery required restoring a pre-change DB backup.

## Root cause (mental model error)

| Wrong assumption | Correct model |
|---|---|
| “Domain = site” | **Process + data dir + port + unit** = site |
| “Same CMS binary → same instance is fine” | Binary may be shared; **runtime state must not** |
| “Brand upgrade = rewrite published config in place” | New product site = **new env / new tree / new DB** |
| “Proxy can host many names → one app is OK” | Multiple names on one upstream means **one site** until split |

Inkless is **single-instance / single-logical-site** in application code. Two
public brands require **two processes** (or two machines), not two DNS records
on one DB. See [`docs/adr/0001-single-instance-single-site.md`](../adr/0001-single-instance-single-site.md).

## Correct topology (example)

| Site role | systemd (example) | Port | Data tree (example) |
|---|---|---|---|
| Personal / blog | `inkless.service` | `8088` | `/opt/inkless/data/` |
| Product marketing | `inkless-ops.service` | `8089` | `/opt/inkless-ops/data/` |

Reverse proxy must route each hostname to its own upstream. Code artifacts
**may** be shared via symlink (`backend/current`, `frontend/current`); **never**
share `data/`, `uploads/`, `.env`, JWT secrets, or systemd `EnvironmentFile`.

## Hard rules for operators and agents

1. **Before any write** (DB, theme, env, proxy, deploy) on a public host, confirm
   unit name, listen port, `DB_DSN`, `BASE_URL`, and reverse-proxy target for
   **each** domain involved.
2. **Never** run product cutover scripts against the personal-site tree when the
   intent is the product site (use the dedicated tree / env only).
3. **Never** cross-set `BASE_URL` / CORS between independent sites without an
   explicit multi-tenant design (this product does not multi-tenant domains in
   one process).
4. **New domain for a new brand** ⇒ assume **new process + new DB** until the
   operator explicitly says “same instance.”
5. **Backup before DB mutation**; verify restore path.
6. After changes, verify **each** site still shows the expected identity
   (`/public/bootstrap` theme + identity).

## Recovery sketch

```bash
systemctl stop <unit>
cp -a /path/to/data/inkless.db.bak-<timestamp> /path/to/data/inkless.db
rm -f /path/to/data/inkless.db-wal /path/to/data/inkless.db-shm
systemctl start <unit>
# verify /public/bootstrap for that site
```

Product-only helpers (paths are examples):

```bash
INKLESS_DB=/opt/inkless-ops/data/inkless.db python3 scripts/ops-product-site-cutover.py
bash scripts/ops-bootstrap-inkless-run.sh
```

## Related

- Brand migration notes: `docs/inkless-brand-migration.md`
- Site config example: `ops/inkless-site-config.example.json`
