# Design: Inkless MCP Server (local agent)

> Status: **M1 implemented** (stdio MVP)  
> Spec target: **MCP `2026-07-28`** (stateless core)  
> Related: [`agent-access.md`](agent-access.md), [`agent-fleet.schema.json`](agent-fleet.schema.json), CLI `inkless site|articles|pages`

## Goals

Expose Inkless multi-site content operations to host agents (Cursor / Claude Code / …) via **MCP**, without:

- CMS-core multi-tenant `site_id`
- Protocol-level sessions
- Direct DB access

## Non-goals (M1)

- Remote/hosted MCP + OAuth/CIMD  
- Tasks extension batch jobs  
- MRTR publish confirmation  
- MCP Apps UI  

## Architecture

```text
Host agent
  │  stdio MCP (2026-07-28, no initialize session required by app logic)
  ▼
inkless mcp serve   (local process)
  │  fleet.json + env keys
  │  internal/agentcli
  ▼
Inkless Admin API (per instance)
```

- **State at protocol layer:** none (SDK may negotiate versions; app does not pin sessions).  
- **State at app layer:** optional short-TTL `preview_handle` for dry-run apply (in-process map).  
- **Multi-site:** `site_id` on tools + fleet registry (same as CLI).

## Tools (M1)

| Tool | R/W | Notes |
|------|-----|--------|
| `list_sites` | R | Fleet inventory, no secrets |
| `resolve_site` | R | baseUrl + masked key meta |
| `whoami` | R | Admin whoami + baseUrl check |
| `list_articles` | R | optional `missing_seo` |
| `get_article` | R | full JSON |
| `apply_article_patch` | W | default `dry_run=true`; preview handle |

## Security

- Keys never returned in tool output (only masked prefix).  
- Mutating tools: `OpenWorldHint`, `DestructiveHint` when not dry-run.  
- Default dry-run for apply.  
- whoami verify before writes (unless `no_verify`).  

## Implementation

- Package: `backend/internal/inklessmcp`  
- CLI: `inkless mcp serve`  
- SDK: `github.com/modelcontextprotocol/go-sdk` v1.7+ (supports 2026-07-28)

## Later

- M2: pages tools, MRTR publish, preview store persistence  
- M3: Tasks extension for fleet-wide SEO scan  
- M4: Streamable HTTP bind + remote auth  
