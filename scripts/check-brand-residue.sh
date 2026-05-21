#!/usr/bin/env bash
# Fails (exits 1) if "印迹|Blotting|blotting" appears in product code.
# Allowed locations (excluded from scan): docs/, .long-agent/, backups/, go.mod,
# scripts/, swagger/, .git/, vendor/, .zip files, .claude/, demo seed content,
# infrastructure config (docker-compose, .env.*.example), Go import paths
# (module rename is out of scope for S1 per spec ambition B).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PATTERN='印迹\|Blotting\|blotting'

# git grep -l (list files with matches), then filter via :! excludes.
# Notes:
# - docs/ keeps historical specs / migration guides
# - swagger/ is generated
# - backend/go.mod and backend/scripts/migrate-sqlite-to-postgres.go reference the old
#   module name and DSN intentionally (out of scope for S1 per spec ambition B)
# - All Go source files contain "blotting-consultancy/" as the module path in imports;
#   renaming the Go module is out of scope for S1 (ambition B). We filter those lines
#   out of the match below and only flag lines that are NOT mere import paths.
# - backups/ and .long-agent/ are agent-runtime artifacts
# - seed.go's DemoSiteSeed (and the data it inserts) is deliberately the consultancy
#   demo dataset; that's its purpose and must remain available via SEED_MODE=demo
# - migrations/ may contain historical SQL referring to old table/column names
# - Infrastructure config files (docker-compose*, .env.*.example, nginx.conf,
#   package.json, .gitignore, CONTRIBUTING.md, Makefile) carry infra-level names
#   (DB names, container names, binary names) that are renamed as part of the
#   infrastructure migration (out of scope for S1 content-layer debranding)
# - scripts/ contains deployment tooling that references the binary name blotting-api
#   and DB credentials; these are renamed in the infra migration (out of scope S1)
# - CLAUDE.md is project documentation and references the project history
EXCLUDES=(
  ':!docs/**'
  ':!**/swagger/**'
  ':!backend/go.mod'
  ':!backend/scripts/**'
  ':!backend/internal/seed/seed.go'
  ':!backend/internal/db/migrations/**'
  ':!**/backups/**'
  ':!**/.long-agent/**'
  ':!**/vendor/**'
  ':!**/*.zip'
  ':!**/.claude/**'
  ':!docker-compose*.yml'
  ':!.env.*.example'
  ':!.env.example'
  ':!nginx.conf'
  ':!package.json'
  ':!.gitignore'
  ':!CONTRIBUTING.md'
  ':!Makefile'
  ':!CLAUDE.md'
  ':!scripts/**'
)

# Two-pass check:
# 1. Get all matching lines (not just file names) excluding the above paths
# 2. Further filter out lines that are only Go module import paths
#    (i.e. lines matching `"blotting-consultancy/...`) or proto go_package option
hits=$(git grep -n "$PATTERN" -- ${EXCLUDES[@]} 2>/dev/null \
  | grep -v '"blotting-consultancy/' \
  | grep -v "option go_package = \"blotting-consultancy" \
  | grep -v "^Binary file" \
  || true)

if [ -n "$hits" ]; then
  echo "ERROR: brand residue found in product code:"
  echo "$hits" | sed 's/:.*//' | sort -u
  echo ""
  echo "Details:"
  echo "$hits"
  exit 1
fi
echo "OK: no brand residue in product code"
