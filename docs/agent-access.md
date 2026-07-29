# 本地 Agent 接入 Inkless（Admin API）

用长期 **API Key**（`ink_…`）让本地 Agent（Claude Code / Cursor / Codex / 脚本）通过 **Admin REST API** 做 SEO、页面编辑、内容维护——**不要直连数据库**。

鉴权模型与 [PicGo 媒体上传](picgo.md) 相同：`Authorization: Bearer ink_…`，有效权限 = **用户 RBAC ∩ key scope**。

| 项 | 说明 |
|----|------|
| 适用 | 单站实例运维、SEO 补齐、草稿维护、批量巡检 |
| 不适用 | 绕过发布流程的静默写库；跨实例共用一把 key |
| 相关 | 站内 AI 元数据见 [article-ai-meta-seo.md](article-ai-meta-seo.md) |

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

## 2. 环境变量

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

---

## 3. 推荐工作流（SEO / 内容维护）

```text
本地 Agent
  │  Bearer ink_…（最小 scope）
  ▼
Admin API（目标实例）
  │  校验 / 版本 / 审计（actor 含 api_key:id）
  ▼
CMS DB  ← agent 禁止直连
```

### 3.1 巡检缺 SEO 的文章

```bash
curl -sS "$INKLESS_BASE_URL/admin/articles?page=1&pageSize=50" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .
```

字段名以实际响应为准；关注 `zhSeoTitle` / `enSeoTitle` / `zhMetaDescription` / `enMetaDescription` / `slug`。

### 3.2 读取单篇

```bash
curl -sS "$INKLESS_BASE_URL/admin/articles/ID" \
  -H "Authorization: Bearer $INKLESS_API_KEY" | jq .
```

### 3.3（可选）站内 AI 生成元数据

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

### 3.4 写回文章（更新 SEO）

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

### 3.5 页面草稿

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

### 3.6 验收

- 公开页与 `/public/bootstrap`（主题 / identity）  
- 审计日志中 actor 形如 `username (api_key:N)`，details 含 `auth_method=api_key`

---

## 4. 安全与双实例

1. **一把 key 只打一个实例**。个人站与产品运营站必须隔离（不同 port / data / JWT / key）。见 [ops-lessons-site-isolation.md](internal/ops-lessons-site-isolation.md)。  
2. 泄露立即在后台 **吊销**。  
3. 默认不要给 `*:publish` / `*:delete`；发布留给人审。  
4. 不要用无鉴权反代绕过 admin。  
5. Agent 规则里写死：**禁止** `sqlite3` / `psql` 直写生产 DSN。  
6. key 明文只放本机 env / secret manager，不写仓库。

---

## 5. Agent Skill 片段（可粘贴）

```markdown
## Inkless content agent

- Base: $INKLESS_BASE_URL
- Auth: Authorization: Bearer $INKLESS_API_KEY
- Prefer Admin API over database.
- Default: draft/update only; never publish unless explicitly asked.
- SEO: merge fields after GET; do not blank unspecified body fields.
- After writes: verify public URL or /public/bootstrap for the intended instance.
- Never use the same key across inkless vs inkless-ops instances.
```

---

## 6. 与站内 AI 的边界

| 能力 | 路径 | 说明 |
|------|------|------|
| 元数据补齐 | `POST /admin/ai/article-meta` | 需实例 AI 配置 + `articles:update` |
| 翻译 | `POST /admin/translate*` | 需 `articles:update` |
| 通用 chat / summarize | `POST /admin/ai/*` | 多数要 `settings:manage`，**API Key 拿不到**；用本地模型或先在编辑器里生成再 PUT |

本地 Agent 可自行调用外部 LLM，再把结果经 Admin API 写回——不必依赖站内 AI。

---

## 7. 故障排查

| 现象 | 可能原因 |
|------|----------|
| 401 | key 错误 / 已吊销 / 前缀不是 `ink_` |
| 403 Permission denied | 用户 RBAC 不足 |
| 403 API key scope denied | key 未包含该 `resource:action` |
| 403 Use a session JWT… | 试图用 `ink_` 管理 api-keys |
| 写了字段但前台无变化 | 改的是草稿未 publish；或打错实例 |

---

## 8. 相关文档

- [picgo.md](picgo.md) — 媒体上传  
- [article-ai-meta-seo.md](article-ai-meta-seo.md) — 文章 AI SEO  
- [api-spec.md](api-spec.md) — REST 概览  
- [product-roadmap.md](product-roadmap.md) — 能力地图  
