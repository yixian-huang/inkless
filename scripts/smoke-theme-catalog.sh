#!/usr/bin/env bash
# Smoke: official theme catalog + install API (Phase A).
# Usage:
#   API_BASE=http://localhost:8088 ADMIN_USER=admin ADMIN_PASS=admin123 ./scripts/smoke-theme-catalog.sh
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8088}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"

echo "== login =="
LOGIN=$(curl -sS -X POST "$API_BASE/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}")
TOKEN=$(echo "$LOGIN" | python3 -c 'import sys,json; print(json.load(sys.stdin)["accessToken"])')
test -n "$TOKEN"
echo "ok token"

echo "== catalog =="
CAT=$(curl -sS "$API_BASE/admin/extensions/themes/catalog" \
  -H "Authorization: Bearer $TOKEN")
echo "$CAT" | python3 -c '
import sys,json
d=json.load(sys.stdin)
assert "items" in d and len(d["items"])>=1, d
print("source=", d.get("source"), "n=", len(d["items"]))
for it in d["items"]:
  print(" -", it.get("slug"), it.get("installState"), it.get("latest",{}).get("version"))
'

echo "== install product-first (no activate) =="
INS=$(curl -sS -X POST "$API_BASE/admin/extensions/themes/install" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"slug":"product-first","activate":false}')
echo "$INS" | python3 -c '
import sys,json
d=json.load(sys.stdin)
assert "theme" in d, d
t=d["theme"]
print("themeId=", t.get("themeId"), "source=", t.get("source"), "version=", t.get("version"), "created=", d.get("created"))
assert t.get("themeId")=="product-first"
assert t.get("source") in ("marketplace","built-in","external")
'

echo "== catalog after install =="
curl -sS "$API_BASE/admin/extensions/themes/catalog" \
  -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
d=json.load(sys.stdin)
pf=next(i for i in d["items"] if i["slug"]=="product-first")
print("product-first state=", pf.get("installState"), "installedVersion=", pf.get("installedVersion"))
'

echo "SMOKE OK"
