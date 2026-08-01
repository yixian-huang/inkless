# 本地 Agent 接入 Inkless（Admin API）

用长期 **API Key**（`ink_…`）让本地 Agent（Claude Code / Cursor / Codex / 脚本）通过 **Admin REST API** 做 SEO、页面编辑、内容维护——**不要直连数据库**。

鉴权模型与 [PicGo 媒体上传](picgo.md) 相同：`Authorization: Bearer ink_…`，有效权限 = **用户 RBAC ∩ key scope**。

| 项 | 说明 |
|----|------|
| 适用 | 单站实例运维、SEO 补齐、草稿维护、批量巡检 |
| 多站 | **Fleet registry**（本机清单）+ 每站独立 key；见 [§4](#4-多站-fleet) |
| 不适用 | 绕过发布流程的静默写库；跨实例共用一把 key；核心 `site_id` 多租户 |
| 相关 | 站内 AI 元数据见 [article-ai-meta-seo.md](article-ai-meta-seo.md)；架构见 [adr/0001-single-instance-single-site.md](adr/0001-single-instance-single-site.md) |

---

## 1. 创建 API Key

1. 用有目标权限的账号登录后台（例如 editor / admin）。  
2. **设置 → API Key**（`/admin/api-keys`）→ **新建 Key**。  
3. 名称示例：`content-agent`。  
4. 权限预设建议：
   - **内容 Agent（推荐）**：`articles:read|update`、`pages:read|update`、`media:create`、`categories:read`、`tags:read`（**不含 publish**）
   - **内容 Agent + 发布**：在上式基础上加 `create` / `publish`（仅高信任环境）
   - **PicGo 上传**：仅 `media:create`（见 [picgo.md](picgo.md)）
5. 复制明文 `ink_…`（仅显示一次），写入本机环境变量，**勿提交到 git**。

也可用 session JWT 创建（`ink_` 不能自管 Key）：

```bash
ACCESS=$(curl -sS -X POST 'https://YOUR_HOST/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"..."}' | jq -r '.accessToken // .token // .data.accessToken')

curl -sS -X POST 'https://YOUR_HOST/admin/api-keys' \
  -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "content-agent",
    "scopes": [
      "articles:read", "articles:update",
      "pages:read", "pages:update",
      "media:create",
      "categories:read", "tags:read"
    ]
  }' | jq .
```

响应中的 `token` 仅此一次。列表 `GET /admin/api-keys`；吊销 `DELETE /admin/api-keys/:id`。

### 允许的 scope 白名单（创建时）

| Scope | 用途 |
|-------|------|
| `media:create` / `media:read` | 上传 / 列媒体 |
| `articles:read\|create\|update\|publish\|delete` | 文章 CRUD 与发布 |
| `pages:read\|create\|update\|publish\|delete` | 统一页面 draft / 发布 |
| `categories:read\|create\|update` | 分类 |
| `tags:read\|create\|update` | 标签 |

**刻意不开放**：`settings:manage`、用户/角色/备份/系统管理等。全站配置仍用浏览器 session。

---

## 2. 环境变量（单站）

```bash
export INKLESS_BASE_URL='https://YOUR_HOST'   # 不要尾斜杠；注意双实例端口/域名
export INKLESS_API_KEY='ink_…'
```

请求头：

```http
Authorization: Bearer ink_…
Content-Type: application/json
```

Swagger（本机）：`http://localhost:8088/swagger/index.html`。

多站请用 Fleet registry（§4），不要共用同一对 env。

---

## 3. 实例探针：`GET /admin/agent/whoami`

写内容前建议先探针，确认 **baseUrl / scopes / 能力** 与目标站一致（防打错实例）。

- 鉴权：JWT 或任意有效 `ink_…` key（**不要求**额外 RBAC 资源权限）  
- 路径：`GET /admin/agent/whoami`

```bash
curl -sS "$INKLESS_BASE_URL/admin/agent/whoami" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .
```

响应示例：

```json
{
  "baseUrl": "https://ops.example.com",
  "version": "0.1.0-alpha.2",
  "authMethod": "api_key",
  "apiKeyId": 42,
  "scopes": ["articles:read", "articles:update", "media:create"],
  "user": { "id": 7, "username": "alice", "role": "editor" },
  "permissions": ["articles:read", "articles:update", "…"],
  "capabilities": {
    "articles": true,
    "pages": true,
    "themeContent": true,
    "themeContentKeys": ["home"],
    "activeThemeId": "product-first",
    "activeThemeVersion": "0.1.9",
    "contentSlots": ["home"],
    "pageTemplates": ["product-first/home@1"],
    "postTemplate": "product-first/post",
    "mediaUpload": true,
    "aiArticleMeta": true,
    "publish": false
  }
}
```

| 字段 | 用途 |
|------|------|
| `baseUrl` | 实例 `BASE_URL`（canonical）；须与 fleet 的 `base_url` 一致 |
| `authMethod` | `api_key` 或 `session` |
| `scopes` | key 声明的 scope；session 为空数组 |
| `capabilities` | **RBAC ∩ scopes** 的摘要，便于 agent 短路 |
| `capabilities.pages` | 能否走 `/admin/pages`（**推荐**一级页写路径） |
| `capabilities.pageTemplates` | 当前主题 page 模板 key 列表（theme-as-templates T4） |
| `capabilities.postTemplate` | 默认 post chrome 模板 key |
| `capabilities.themeContent` | 能否走 `/admin/content/:pageKey`（**迁移期**；prefer pages） |
| `capabilities.themeContentKeys` | 有主题 slots 时 = slots 的 pageKey；否则 Host 白名单 |
| `capabilities.activeThemeId` | 当前激活主题 id |
| `capabilities.contentSlots` | 主题声明的 contentSlots pageKey 列表（**deprecated**；prefer pageTemplates） |

Agent 规则：若 `baseUrl` 与任务站点 profile 不一致 → **中止写操作**。

---

## 4. 多站 Fleet

### 4.1 原则（ADR-0001）

| 规则 | 说明 |
|------|------|
| 一站一实例 | 独立 `BASE_URL`、DB、JWT、uploads、API Key |
| 一站一 key | **禁止** 一把 `ink_` 打多个 host |
| Fleet 在客户端 | 站点清单放本机 / 私有 ops，不进 CMS 核心 `site_id` |
| 内容只走 Admin API | 部署控制面（npc 等）不写文章/页面 |

```text
┌─────────────────────────────────────┐
│  Fleet（本机 registry + agent）      │
│  site_id → base_url + key_env       │
└──────────────┬──────────────────────┘
               │ 每站独立 HTTPS + ink_ key
     ┌─────────┼─────────┐
     ▼         ▼         ▼
  Instance A  B          C
  (单站 CMS)  (单站 CMS)  (单站 CMS)
```

### 4.2 Registry 格式

- **JSON Schema**：[agent-fleet.schema.json](agent-fleet.schema.json)  
- **示例**：[examples/agent-fleet.example.json](examples/agent-fleet.example.json)

建议路径（任选）：

```text
~/.config/inkless/fleet.json
# 或仓库私有（勿提交 key）：
./.inkless/fleet.json   # 加入 .gitignore
```

字段摘要：

| 字段 | 说明 |
|------|------|
| `version` | 固定 `1` |
| `default_site` | 可选默认 `site_id` |
| `sites.<id>.base_url` | 实例 origin，无尾斜杠 |
| `sites.<id>.api_key_env` | 存放 `ink_…` 的环境变量名 |
| `sites.<id>.api_key_file` | 或密钥文件路径（二选一） |
| `sites.<id>.scopes_expected` | 期望 scopes；连上后与 whoami 比对 |
| `sites.<id>.publish_policy` | `never` \| `manual` \| `allow` |
| `sites.<id>.verify.whoami` | 写前探针（默认建议 true） |

密钥：

```bash
export INKLESS_KEY_PERSONAL='ink_…'
export INKLESS_KEY_OPS='ink_…'
# 每个 api_key_env 对应一把 key，绝不复用
```

校验示例（需本机有 JSON Schema 工具时）：

```bash
# 若安装了 ajv-cli / check-jsonschema 等
check-jsonschema --schemafile docs/agent-fleet.schema.json docs/examples/agent-fleet.example.json
```

### 4.3 Agent 解析流程

1. 任务必须带 **`site_id`**（或使用 `default_site`；否则先问用户）。  
2. 从 registry 取 `base_url` + 解析 `api_key_env` / `api_key_file`。  
3. `GET {base_url}/admin/agent/whoami`：  
   - `baseUrl` 规范化后等于 profile（去尾斜杠）  
   - 可选：`scopes` ⊇ `scopes_expected`  
4. 执行读/写 Admin API。  
5. 若 `publish_policy=never`，拒绝 `*/publish`。  
6. 验收该站公开 URL 或 `/public/bootstrap`。

### 4.4 批量任务

| 任务 | 策略 |
|------|------|
| N 站缺 SEO 巡检 | 可并行只读；输出 `{site_id, article_id, gaps}` |
| 批量补 SEO | **按站串行**写 draft；默认不 publish |
| 跨站复制内容 | 显式 export→import，不做自动双向同步 |

### 4.5 单站兼容

只有一站时：

- 仍可用 §2 的 `INKLESS_BASE_URL` + `INKLESS_API_KEY`；或  
- fleet 里只放一项，`default_site` 指向它。

---

## 5. 推荐工作流（SEO / 内容维护）

```text
本地 Agent
  │  resolve site_id → base_url + key
  │  GET /admin/agent/whoami  （防打错站）
  │  Bearer ink_…（最小 scope）
  ▼
Admin API（目标实例）
  │  校验 / 版本 / 审计（actor 含 api_key:id）
  ▼
CMS DB  ← agent 禁止直连
```

### 5.1 巡检缺 SEO 的文章

```bash
curl -sS "$INKLESS_BASE_URL/admin/articles?page=1&pageSize=50" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .
```

字段名以实际响应为准；关注 `zhSeoTitle` / `enSeoTitle` / `zhMetaDescription` / `enMetaDescription` / `slug`。

### 5.2 读取单篇

```bash
curl -sS "$INKLESS_BASE_URL/admin/articles/ID" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .
```

### 5.3（可选）站内 AI 生成元数据

需 scope 含 `articles:update`，且实例已配置 AI provider：

```bash
curl -sS -X POST "$INKLESS_BASE_URL/admin/ai/article-meta" \
  -H "Authorization: Bearer $INKLESS_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "sourceLang": "zh",
    "zhBody": "…正文…",
    "fields": ["titles", "slug", "seo", "meta"],
    "mode": "fill_empty",
    "titleCount": 3
  }' | jq .
```

生成结果应**预览后**再写入；不要静默覆盖已发布 slug。

### 5.4 写回文章（更新 SEO）

```bash
curl -sS -X PUT "$INKLESS_BASE_URL/admin/articles/ID" \
  -H "Authorization: Bearer $INKLESS_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "zhTitle": "…",
    "enTitle": "…",
    "slug": "…",
    "zhSeoTitle": "…",
    "enSeoTitle": "…",
    "zhMetaDescription": "…",
    "enMetaDescription": "…",
    "zhBody": "…",
    "enBody": "…"
  }' | jq .
```

`PUT` 请求体字段需与后台编辑器一致（缺失字段可能被置空——先 `GET` 再合并修改项）。

### 5.5 页面草稿（unified `/p/*`）

```bash
# 读草稿
curl -sS "$INKLESS_BASE_URL/admin/pages/ID/draft" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .

# 写草稿（scope: pages:update）
curl -sS -X PUT "$INKLESS_BASE_URL/admin/pages/ID/draft" \
  -H "Authorization: Bearer $INKLESS_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"config":{…},"changeNote":"agent: seo copy"}' | jq .

# 发布（需 pages:publish；默认 agent key 不要开）
curl -sS -X POST "$INKLESS_BASE_URL/admin/pages/ID/publish" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .
```

页面 SEO 元数据也在 `PUT /admin/pages/:id` 的 `zhMetaDescription` / `enMetaDescription` 等字段。

### 5.5b 主题绑定内容（product-first `home` 等）

**真源：** `content_documents.page_key`（如 `home`），**不是** articles，也**不在** `pages list`。  
**API：** `GET/PUT /admin/content/:pageKey/draft` + `POST .../publish`（设计见 [design-theme-content-admin-api.md](design-theme-content-admin-api.md)）。  
**权限：** 读 `pages:read`；写 `pages:update`；发布 `pages:publish`。  
**MediaRef：** `url` / `alt` / `caption` 必须是 **string**（禁止 `{zh,en}`，否则 400 / 白屏）。

CLI（推荐）：

```bash
inkless site whoami --site SITE   # pageTemplates + themeContentKeys
inkless templates list --site SITE              # 推荐：主题 templates 发现
inkless templates get product-first/home@1 --site SITE --json

# 迁移期别名（仍可用；stderr deprecation）
inkless content slots --site SITE   # 含 templatesProjection
inkless content schema home --site SITE
inkless content keys --site SITE    # 含 pageTemplates

inkless media upload ./shot.png --site SITE --json   # → .url

# 生产写路径（推荐）：pages 绑定 templateKey
inkless pages list --site SITE
inkless pages get-draft ID --site SITE --json
inkless pages put-draft ID --site SITE --from-file home.json --dry-run

# 迁移期 content 写（bridge → Page + Deprecation 头）
inkless content apply home --site SITE --from-file home.json --dry-run --validate-schema
inkless content apply home --site SITE --from-file home.json --validate-schema
inkless content publish home --site SITE   # honors publish_policy

inkless content get home --site SITE --public --locale zh --json
inkless content versions home --site SITE
```

Swagger：`/api-docs` → tags **Content (Admin)**、**Themes**（`/admin/themes/active/templates`）。

curl 等价：

```bash
curl -sS "$INKLESS_BASE_URL/admin/content/home/draft" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .

curl -sS -X PUT "$INKLESS_BASE_URL/admin/content/home/draft" \
  -H "Authorization: Bearer $INKLESS_API_KEY" \
  -H "Content-Type: application/json" \
  -H "If-Match: $VERSION" \
  -d @home.json   # body: {"config":{...}} 或 CLI 自动包一层

curl -sS "$INKLESS_BASE_URL/public/content/home?locale=zh" | jq .
```

无 content API 的旧实例：**禁止写库**；应升级 host 到含 M1 的版本，或人工在 SPA 编辑（若 UI 已接同一 API）。

### 5.5c 迁移到 Page（theme-as-templates T3）

生产写路径目标为 **unified pages**。一次性迁移：

```bash
inkless content migrate-to-pages --site SITE
# 强制用 content_documents 覆盖 Page（慎用）:
inkless content migrate-to-pages --site SITE --force

inkless pages list --site SITE   # 应见 slug=home
```

迁移后 `content get/apply/publish home` 仍可用：有 Page 时 **桥接到 Page**，并 dual-write content_documents 供公开双读；响应带 `Deprecation` / `X-Inkless-Prefer` 头。  
新 skill 请优先 `pages *`。

### 5.5d 主题模板发现（theme-as-templates T4）

| 任务 | 命令 / API |
|------|------------|
| 列模板 | `inkless templates list` → `GET /admin/themes/active/templates` |
| 取 schema | `inkless templates get <key>` → `GET /admin/themes/active/template?key=` |
| whoami | `capabilities.pageTemplates` / `postTemplate` |
| 兼容 | `content slots` 返回 `templatesProjection`；`content schema` 可带 `templateKey` |

无原生 `templates[]` 时 Host 将 **contentSlots 投影** 为 page 模板 + 默认 `{themeId}/post`。权限：`pages:read`。

### 5.6 验收

- 公开页与 `/public/bootstrap`（主题 / identity）  
- 审计日志中 actor 形如 `username (api_key:N)`，details 含 `auth_method=api_key`  
- whoami 的 `baseUrl` 与任务 site 一致  

---

## 6. 安全

1. **一把 key 只打一个实例**。个人站与产品运营站必须隔离。见 [ops-lessons-site-isolation.md](internal/ops-lessons-site-isolation.md)。  
2. 泄露立即在后台 **吊销**。  
3. 默认不要给 `*:publish` / `*:delete`；发布留给人审（`publish_policy: never|manual`）。  
4. 不要用无鉴权反代绕过 admin。  
5. Agent 规则里写死：**禁止** `sqlite3` / `psql` 直写生产 DSN。  
6. key 明文只放本机 env / secret manager / `api_key_file`，不写仓库；fleet 文件可提交时务必只含 `api_key_env` 名。  
7. 写前 whoami：`baseUrl` 不匹配则中止。

---

## 7. Agent Skill 片段（可粘贴）

### 单站

```markdown
## Inkless content agent (single site)

- Base: $INKLESS_BASE_URL
- Auth: Authorization: Bearer $INKLESS_API_KEY
- Before writes: GET /admin/agent/whoami and confirm baseUrl.
- Prefer Admin API over database.
- Default: draft/update only; never publish unless explicitly asked.
- SEO: merge fields after GET; do not blank unspecified body fields.
- After writes: verify public URL or /public/bootstrap for the intended instance.
- Never use the same key across different Inkless instances.
```

### 多站

```markdown
## Inkless content agent (fleet)

- Prefer official CLI: `inkless site whoami --site <id>`, `inkless articles … --site <id>`.
- Load fleet registry (JSON Schema: docs/agent-fleet.schema.json).
- Every task must name site_id (or use default_site); never invent base_url.
- Resolve {base_url, api_key} from that profile only.
- GET {base_url}/admin/agent/whoami; abort if baseUrl mismatch.
- Honor publish_policy: never | manual | allow.
- Parallel multi-site only for read/audit; writes serialize per site unless user asks batch.
- Never reuse key A against host B.
```

---

## 8. 与站内 AI 的边界

| 能力 | 路径 | 说明 |
|------|------|------|
| 元数据补齐 | `POST /admin/ai/article-meta` | 需实例 AI 配置 + `articles:update` |
| 翻译 | `POST /admin/translate*` | 需 `articles:update` |
| 通用 chat / summarize | `POST /admin/ai/*` | 多数要 `settings:manage`，**API Key 拿不到**；用本地模型或先在编辑器里生成再 PUT |

本地 Agent 可自行调用外部 LLM，再把结果经 Admin API 写回——不必依赖站内 AI。

---

## 9. 故障排查

| 现象 | 可能原因 |
|------|----------|
| 401 | key 错误 / 已吊销 / 前缀不是 `ink_` |
| 403 Permission denied | 用户 RBAC 不足 |
| 403 API key scope denied | key 未包含该 `resource:action` |
| 403 Use a session JWT… | 试图用 `ink_` 管理 api-keys |
| whoami `baseUrl` 与预期不符 | 打错实例 / 反代 host / `BASE_URL` 未配 |
| 写了字段但前台无变化 | 改的是草稿未 publish；或打错实例 |
| 多站任务写到 A 站内容出现在 B | key/env 串用；检查 fleet 与 whoami |

---

## 10. 官方 CLI（`inkless site` / `articles` / `pages`）

二进制：`backend/cmd/inkless`（与 migrate/serve 同一 CLI）。

```bash
cd backend && go install ./cmd/inkless/
# 或
go build -o inkless ./cmd/inkless/
```

### 10.1 常用命令

| 命令 | 说明 |
|------|------|
| `inkless site list [--fleet path]` | 列出 fleet 中的站 |
| `inkless site resolve --site <id>` | 解析 baseUrl / 掩码 key / policy |
| `inkless site whoami --site <id>` | 探针 `/admin/agent/whoami` |
| `inkless articles list --site <id> [--missing-seo]` | 列文章；可筛缺 SEO |
| `inkless articles get <id> --site <id>` | 拉全文 JSON |
| `inkless articles apply <id> --from-file patch.json --site <id> [--dry-run]` | GET→merge→PUT |
| `inkless pages list\|get\|get-draft\|put-draft\|publish` | 页面维护（publish 尊重 policy） |
| `inkless pages presets` | 列出 Host 页面配方（doc-simple / doc-guide / landing-use-cases） |
| `inkless pages create --slug … [--preset doc-guide]` | 新建 composable 页（可带配方多区块） |
| `inkless pages apply-preset <id> --preset doc-guide` | 用配方覆盖草稿（需再 publish） |

公共 flags：`--fleet`、`--site`、`--base-url`、`--api-key`、`--json`、`--no-verify`、`--timeout`。

### 10.2 单站

```bash
export INKLESS_BASE_URL='https://YOUR_HOST'
export INKLESS_API_KEY='ink_…'
inkless site whoami
inkless articles list --missing-seo --json
```

### 10.3 多站

```bash
# ~/.config/inkless/fleet.json  （见 examples/agent-fleet.example.json）
export INKLESS_KEY_OPS='ink_…'
inkless site list
inkless site whoami --site product-ops
inkless articles list --site product-ops --missing-seo
inkless articles apply 12 --site product-ops --from-file ./seo-patch.json --dry-run
inkless articles apply 12 --site product-ops --from-file ./seo-patch.json
```

`publish_policy=never` 时 `pages publish` 会拒绝；`manual` 需 `--force`。

### 10.4 写操作安全

- 默认会 whoami 校验 `baseUrl`（可用 `--no-verify` 关闭，不推荐）  
- `articles apply` 合并补丁，避免 PUT 清空未出现的字段  
- 优先 `--dry-run` 再写  

---

## 11. MCP（宿主 Agent）

本地 MCP 服务器：`inkless mcp serve`（stdio，规格 `2026-07-28`）。

- 设计：[design-inkless-mcp.md](design-inkless-mcp.md)  
- 用法：[agent-mcp.md](agent-mcp.md)  

与 CLI 共享 fleet / agentcli；不引入 CMS 多租户。

---

## 12. 相关文档

- [agent-fleet.schema.json](agent-fleet.schema.json) — Fleet JSON Schema  
- [examples/agent-fleet.example.json](examples/agent-fleet.example.json) — 多站示例  
- [agent-mcp.md](agent-mcp.md) — MCP 服务器  
- [picgo.md](picgo.md) — 媒体上传  
- [article-ai-meta-seo.md](article-ai-meta-seo.md) — 文章 AI SEO  
- [adr/0001-single-instance-single-site.md](adr/0001-single-instance-single-site.md) — 单实例单站  
- [api-spec.md](api-spec.md) — REST 概览  
- [product-roadmap.md](product-roadmap.md) — 能力地图  
