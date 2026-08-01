# 设计：Theme Content Admin API（恢复 agent 可写的主题页内容槽）

| 字段 | 值 |
|------|-----|
| 状态 | **M1–M3 implemented** — Admin API + CLI + deep dry-run + versions/rollback + swagger |
| 日期 | 2026-08-01 |
| 范围 | Host：Admin API + cache + CLI/skill 文档；**不改** product-first 视觉 |
| 相关 | [ADR-0002 主题边界](adr/0002-theme-host-boundary.md)、[product-first 设计](design-product-first-theme.md)、[api-spec § content](api-spec.md)、[agent-access](agent-access.md) |

---

## 1. 问题

| 事实 | 影响 |
|------|------|
| product-first 首页内容真源是 **`content_documents.page_key=home`**（flat JSON：hero / showcase / features / …） | 主题 C 类页，**不是** articles，也**不是** unified `/p/*` |
| 公开读仍在：`GET /public/content/:pageKey`（unified miss 时 legacy `content_documents`） | 验收路径可用 |
| Admin 写路径已消失：`routes_admin` 仅有 `/admin/pages/*`；`/admin/content/*` 未挂载 | SPA 路由返回 HTML；API Key agent **无法合规写 home** |
| skill / agent-access 禁止 SSH 写库 | 运营 agent 只能违规改 DB 或卡住 |
| `MediaRef.alt/caption` 契约在 product-first 为 **string**，误写 `{zh,en}` 会 React #31 | 需校验/拍扁，属可用性 bug |

**目标一句话：** 恢复与 articles/pages **同语义** 的 draft → publish 写通道，让 content-agent 在 **不写生产库** 的前提下维护主题落地页。

---

## 2. 决策（锁定）

| 项 | 选择 | 理由 |
|----|------|------|
| 存储 | **继续用 `content_documents`**（不迁 unified_pages） | 已有表、public 读、export/backup；与 D 类 section 栈模型不同 |
| API 形态 | **恢复** `/admin/content/:pageKey/*`（对齐现有 [api-spec](api-spec.md) 与 integration tests） | 文档与测试债本就指向此路径；前端 `fetchDraftContent` 仍引用 |
| 与 `/admin/pages` | **并存、不合并 list** | pages = 用户动态页；content = 主题绑定 pageKey 槽 |
| 乐观锁 | body `expectedVersion` **或** `If-Match`（与 pages 二选一实现时优先 **与旧 content 测试一致**；若旧 handler 无 If-Match 则 body 字段） | 防并发覆盖 |
| 权限 | `pages:read` / `pages:update` / `pages:publish` | 复用 content-agent 已有 scope，不新增 `content:*` RBAC 资源（MVP） |
| MediaRef 叶子 | **强制 string**：`url` / `alt` / `caption`；拒绝 bilingual object | 防 #31；Localized 只允许文案字段 |
| 缓存 | publish / draft→影响 public 的写之后 **`InvalidatePagePublic(c, pageKey)`**（清 `content:{key}:*`） | 禁止「改完必须 restart」 |
| 非目标 | 不把 theme home 塞进 pages list；不做 DB 直写 runbook；不改主题视觉 | 见主题侧独立任务 |

**明确不做（本设计）：**

- 把 `home` 迁成 unified page + section 反解（成本高，阻断 agent 急救）
- 默认给 content-agent `settings:manage` / `themes:manage`
- 服务端对 product-first 做完整 JSON Schema 门禁（MVP 只做结构护栏 + MediaRef；完整 schema 可后续）

---

## 3. 内容类型边界（给 agent / skill）

```text
articles          → 时间流、changelog、SEO 文
/admin/pages      → D 类动态页 /p/*（presets）
/admin/content/*  → 主题 C 类绑定槽（home 等 flat config）
site-config       → identity / brand / logo（另轨；本设计可只读文档提及）
```

| pageKey（MVP 必开写） | 消费方 | 备注 |
|----------------------|--------|------|
| `home` | product-first / 部分 host 页 | **P0** |
| `contact` 等 legacy keys | corporate / public 读 | 一并恢复写（白名单 `model.ValidPageKeys` 减去仅内部用途的 key 若有） |
| `global` | 历史；bootstrap 以 site_configs 为准 | 写接口可保留但文档标明 **deprecated for brand**，优先 global-config |

`pages list` **不会**出现 `home`。发现性靠：

- 文档 + skill 写死路径；和/或
- `whoami.capabilities.themeContent: true` + 可选 `themeContentKeys: ["home", …]`

---

## 4. API 契约（MVP）

路径前缀：`/admin/content/:pageKey`  
鉴权：`Authorization: Bearer`（session JWT 或 `ink_…`），**有效权限 = RBAC ∩ key scope**。

| Method | Path | 权限 | 行为 |
|--------|------|------|------|
| GET | `.../draft` | pages:read | 返回 draft config + version |
| PUT | `.../draft` | pages:update | 乐观锁更新 draft；**不**改 published |
| POST | `.../validate` | pages:update | 结构/MediaRef 校验；可选 translation hints |
| POST | `.../publish` | pages:publish | draft → published；写 version 历史（若表仍在）；**invalidate cache** |
| GET | `.../versions` | pages:read | 列表（若已有 model） |
| POST | `.../rollback/:version` | pages:publish | 回滚 published（与旧 api-spec 对齐） |

公开读（已有，不改语义，可加文档）：

- `GET /public/content/:pageKey?locale=zh|en` — 仅 published；locale 主要用于埋点；**配置仍可为 bilingual bag**（主题 client pick）。MVP **不强制**服务端拍扁全树，避免双重 pick；**仅在 validate/publish 拒绝非法 MediaRef**。

### 4.1 请求/响应形状（与 api-spec 对齐）

```http
GET /admin/content/home/draft
→ 200 { "pageKey": "home", "version": 12, "config": { ... } }

PUT /admin/content/home/draft
If-Match: 12
{ "config": { ... } }
→ 200 { "pageKey": "home", "version": 13, "updatedAt": "..." }
→ 409 版本冲突

POST /admin/content/home/publish
{ "expectedVersion": 13 }   # 或与 draft 一致的锁字段
→ 200 { "pageKey": "home", "publishedVersion": 11, ... }
```

`publish_policy=never` 的 fleet 站点：CLI/agent **不得**调 publish；可只写 draft，由人工在 SPA 发布——**前提是 SPA 也能调同一 API**（恢复后自然满足）。

### 4.2 MediaRef 校验规则（P0）

对 config 树递归：

- 若对象形如 media 槽（键名 `media` / `backgroundImage` / `image`，或具备 `url` 且同时有 `alt`/`caption` 的叶子）：
  - `url`、`alt`、`caption` 的值若为 **object**（含 `{zh,en}`）→ **400** validate/publish（PUT draft 可 warn 或同样 400，MVP 建议 PUT 也 400 防脏草稿）。
- 文案字段 `title` / `subtitle` / `description` / `label` 等 **允许** `{zh,en}`。

与旧 [data-model](data-model.md) 中「MediaRef.alt = LocalizedText」冲突时：**以 product-first + 本设计为准**——主题 media 叶子 string；企业旧 schema 若仍用 Localized alt，validate 对 corporate keys 可分 schema 或渐进（MVP：统一 string 化要求更安全，旧数据 publish 前需拍扁）。

---

## 5. 实现要点

```text
handler (admin content)
  → service (load/save draft, publish, validate MediaRef)
  → repository ContentDocument (+ ContentVersion if present)
  → on publish/update affecting public: cache.InvalidatePagePublic(pageKey)
```

- **Wire：** `registerAdminContent` 旁挂 content 路由（或独立 `registerAdminThemeContent`），注入已有 `ContentDocumentRepository` + `publicCache`。
- **缺文档：** GET draft 若不存在可 **空 config + version 0**（便于 agent 首次写入），publish 前 validate。
- **审计：** actor 含 `api_key:id`（与 articles 一致）。
- **测试：** 复活 `integration_test` 中 `/admin/content/home/*` 用例；加 MediaRef object → 400；publish 后 public 读新值且 `X-Cache` 可 MISS。

---

## 6. CLI / whoami / skill（同迭代或紧随）

| 交付 | 说明 |
|------|------|
| `inkless content get home` | GET draft 或 public（flag 区分） |
| `inkless content apply home --from-file x.json --dry-run` | 打印 diff / 校验；再 apply PUT draft |
| `inkless media upload` | 已有 HTTP，补 CLI（agent 少猜路径） |
| whoami | `capabilities.themeContent`；可选 keys |
| skill（ops/imgli） | 最小闭环：upload → apply home → GET `/public/content/home` → 浏览器 smoke；**无 API 时禁止写库** |

---

## 7. Agent 最小合规流程（验收故事）

```bash
inkless site whoami --site imgli
# capabilities.themeContent == true

inkless media upload --site imgli ./shot.png   # → url
# 编辑 home.json：MediaRef 全 string；文案用 {zh,en}

inkless content apply home --site imgli --from-file home.json --dry-run
inkless content apply home --site imgli --from-file home.json
# publish_policy=never → 停在 draft；人工 publish 或策略允许时：
# inkless content publish home --site imgli

curl -sS "$BASE/public/content/home?locale=zh" | jq .config.hero
# 浏览器打开 / 确认无白屏、图生效（cache 无需 restart）
```

---

## 8. 分期

| 阶段 | 内容 | 完成标准 |
|------|------|----------|
| **M1** | 恢复 GET/PUT draft + POST publish + cache invalidate + MediaRef 校验 + 集成测试 | **done** |
| **M2** | CLI content/media + whoami.themeContent + agent-access/skill | **done** |
| **M3** | deep dry-run diff、local MediaRef preflight、versions/rollback CLI、schemaKind、swagger Content (Admin) | **done** |
| **Out** | home → unified 迁移、服务端全量 locale flatten、CF bot 规则 | 另案 |

---

## 9. 风险

| 风险 | 缓解 |
|------|------|
| 与「一切迁 unified pages」叙事冲突 | ADR-0002：C 类主题页内容槽 ≠ D 类 page；本 API 是 C 的宿主，不是倒退 |
| corporate MediaRef 旧数据 | validate 明确错误信息；提供一次性拍扁脚本（可选） |
| publish 权限过宽 | fleet 默认无 publish；fleet `publish_policy=never` |
| SPA 仍 404 旧 content 编辑 UI | M1 以 API 为准；Admin UI 可后补或沿用残留页 |

---

## 10. 决议摘要

1. **恢复** `content_documents` 的 Admin draft/publish API，路径与 [api-spec](api-spec.md) 一致。  
2. **权限** 挂 `pages:*`，content-agent 零 RBAC 扩资源即可写 home。  
3. **MediaRef 叶子 string-only** 为硬校验。  
4. **写后必失效** `content:{pageKey}:*` 缓存。  
5. **CLI + skill** 跟进，关闭「只能写库」的 agent 死路。
