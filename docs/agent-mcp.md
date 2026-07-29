# Inkless MCP Server（本地 Agent）

把多站内容维护暴露给 Cursor / Claude Code 等宿主，协议目标 **MCP `2026-07-28`（无状态核心）**。

| 文档 | 说明 |
|------|------|
| [design-inkless-mcp.md](design-inkless-mcp.md) | 设计与阶段 |
| [agent-access.md](agent-access.md) | Admin API + Fleet + CLI |
| [agent-fleet.schema.json](agent-fleet.schema.json) | Fleet schema |

## 安装

```bash
cd backend
go install ./cmd/inkless/
# 或
go build -o inkless ./cmd/inkless/
```

## 启动

```bash
# 多站
inkless mcp serve --fleet ~/.config/inkless/fleet.json

# 单站
export INKLESS_BASE_URL='https://yx.ink'
export INKLESS_API_KEY='ink_…'
inkless mcp serve
```

**注意：** stdio 模式下不要往 stdout 打日志；密钥放 env / fleet `api_key_env`。

## 宿主配置示例

### Cursor / Claude Desktop 风格

```json
{
  "mcpServers": {
    "inkless": {
      "command": "inkless",
      "args": ["mcp", "serve", "--fleet", "/Users/you/.config/inkless/fleet.json"],
      "env": {
        "INKLESS_KEY_PERSONAL": "ink_…",
        "INKLESS_KEY_OPS": "ink_…"
      }
    }
  }
}
```

## Tools（M1）

| Tool | 说明 |
|------|------|
| `list_sites` | Fleet 站点列表 |
| `resolve_site` | 解析 baseUrl（key 掩码） |
| `whoami` | `GET /admin/agent/whoami` + baseUrl 校验 |
| `list_articles` | 列文章；`missing_seo` 过滤 |
| `get_article` | 拉文章 JSON |
| `apply_article_patch` | 合并补丁；**默认 dry_run**；返回 `preview_handle` |

### apply 工作流

1. `apply_article_patch` + `patch` + `dry_run=true` → 得到 `previewHandle`  
2. 人工/模型审阅 `mergedBody`  
3. 同 `site_id`/`id` + `dry_run=false` + `preview_handle` → 真正 PUT  

## 与无状态 MCP 的关系

- 不依赖协议 session；每条 tool 自带 `site_id` 与鉴权解析  
- 跨步状态仅用短 TTL **preview_handle**（进程内 map）  
- SDK `go-sdk` v1.7+ 支持 `2026-07-28`  

## 安全

- 输出不含 `ink_` 明文  
- 写操作默认 dry-run  
- 写前 whoami（可用 `--no-verify` 关闭，不推荐）  
- CMS 侧仍是 RBAC ∩ key scope + 审计  

## 后续

- pages tools、publish + MRTR  
- Tasks 扩展批量 SEO  
- Streamable HTTP / Remote OAuth  
