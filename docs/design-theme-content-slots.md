# 设计：Theme Content Slots（主题可发现内容契约 + CLI 校验）

| 字段 | 值 |
|------|-----|
| 状态 | **S1–S4 implemented (MVP)** — slots/schema API + path validate + CLI + product-first embed |
| 日期 | 2026-08-01 |
| 范围 | Host：schema 发现 API + validate 挂主题契约；CLI：`--validate-schema`；主题：`inkless.theme.json` / 包内 schema |
| 相关 | [ADR-0002](adr/0002-theme-host-boundary.md)、[theme-contract](theme-contract.md)、[theme-content-admin-api](design-theme-content-admin-api.md)、[product-first](design-product-first-theme.md)、[agent-access](agent-access.md) |

---

## 1. 问题

| 事实 | 痛点 |
|------|------|
| 不同主题 home 形态不同（product-first vs blog-first vs corporate） | 字段、MediaRef 路径、布局语义不一致 |
| Host 已提供 **主题无关** 槽位 API：`/admin/content/:pageKey` + CLI apply/diff | Agent 能「写进库」，但不保证「写对某主题」 |
| Host validate 用 **启发式** 猜 `product-first` / `corporate` | 新主题/改版主题会误判或漏检 |
| 若 CLI 为每个主题加子命令 | 主题升级 = CLI 分叉，不可维护 |

**目标：** 在 **不改 CLI 主命令形态** 的前提下，让「当前激活主题」的内容形状 **可发现、可校验、可 dry-run**，Host/CLI 对主题 **插拔兼容**。

**非目标：** 主题私有写库协议；CLI `inkless product-first …` 专属命令；服务端全量 locale flatten；统一把所有主题页迁 unified pages。

---

## 2. 决策（锁定）

| 项 | 选择 |
|----|------|
| 内容存储 | 继续 **content_documents**（C 类主题槽）+ 已有 Admin API（M1） |
| 形状真理 | **主题包声明** content contract，不在 Host 硬编码各主题 JSON |
| CLI 形态 | 仍是 `content get/apply/…`；增加 **schema 发现** 与 **可选 schema 校验** |
| 硬护栏 | Host 全局 **MediaRef 叶子 string** 保留（跨主题安全底线） |
| 启发式 | 现有 `schemaKind` 猜测 **降级为 fallback**（无主题契约时）；有契约则以契约为准 |
| 契约载体 | `inkless.theme.json` 的 `contentSlots[]` + 包内 JSON Schema 文件（或 inline schema） |

**分层口诀：**

```text
Host CLI  = 管道（槽位、版本、媒体、diff、鉴权）
Theme     = 形状（slots + JSON Schema + MediaRef/Localized 路径）
Skill     = 某站运营知识（golden home.json、文案策略）
```

---

## 3. 主题声明：`contentSlots`

### 3.1 `inkless.theme.json` 扩展（additive，contract major 可仍为 `"1"`）

```json
{
  "id": "product-first",
  "version": "0.1.9",
  "contractVersion": "1",
  "contentSlots": [
    {
      "pageKey": "home",
      "schemaId": "product-first/home@1",
      "title": { "zh": "产品首页", "en": "Product home" },
      "description": "Landing: hero / showcase / features / howItWorks / install / bottomCta",
      "schema": "schemas/content/home.schema.json",
      "mediaRefPaths": [
        "hero.media",
        "showcase.items[]",
        "features.items[].media"
      ],
      "localizedPaths": [
        "hero.title",
        "hero.subtitle",
        "hero.eyebrow",
        "features.items[].title",
        "features.items[].description",
        "install.caption",
        "bottomCta.title"
      ],
      "stringPaths": [
        "install.code",
        "hero.primaryCta.href",
        "hero.secondaryCta.href"
      ]
    }
  ]
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `pageKey` | ✓ | 必须是 Host `ValidPageKeys` 中允许写的 key（通常 `home`，可扩） |
| `schemaId` | ✓ | 稳定 id + 版本，如 `themeId/slot@major`；breaking 升 major |
| `schema` | ✓* | 相对主题包根路径的 JSON Schema；或 `schemaInline` 二选一 |
| `mediaRefPaths` | 推荐 | JSONPath 简化版：`.` 与 `[]`；Host 校验这些节点的 url/alt/caption 为 string |
| `localizedPaths` | 推荐 | 允许 `{zh,en}` 的文案路径 |
| `stringPaths` | 可选 | 禁止 bilingual bag 的纯 string 路径（如 install.code） |
| `title` / `description` | 可选 | Agent/Admin 展示 |

**规则：**

1. 未声明 `contentSlots` 的主题 = **legacy**：行为与今相同（MediaRef 硬规则 + 可选启发式）。  
2. 同一 `pageKey` 在一个主题内 **最多一条** slot。  
3. `pageKey` 不在 Host 白名单 → 激活/校验时报 warning，**不**自动开放新 DB key（防主题私开模型）。扩展白名单属 Host 发版。  
4. Schema 只描述 **可运营 JSON**；不描述路由/chrome/tokens。

### 3.2 主题包布局示例

```text
inkless-theme-product-first/
  inkless.theme.json
  schemas/content/home.schema.json    # draft-07 或 2020-12
  src/pages/home.tsx                  # 消费同一形状
```

主题 CI：schema 与 TypeScript 类型 / golden fixture 对齐（主题仓义务）。

---

## 4. Host 发现与校验 API

### 4.1 激活主题解析

Host 已有：active installed theme id + 主题包/catalog 元数据。  
实现时从 **当前激活主题** 的 manifest 读 `contentSlots`（内置主题、UMD 安装包均需能解析 `inkless.theme.json`）。

### 4.2 新/扩展端点

| Method | Path | 权限 | 行为 |
|--------|------|------|------|
| GET | `/admin/content/slots` | pages:read | 当前激活主题的 slots 摘要（无完整 schema 体，省 token） |
| GET | `/admin/content/:pageKey/schema` | pages:read | 该 pageKey 的 slot + **完整 JSON Schema** + paths 提示 |
| POST | `/admin/content/:pageKey/validate` | pages:update | **增强**：若存在主题 schema → 按 schema 校验；否则 fallback 启发式 |

**`GET …/slots` 响应示例：**

```json
{
  "activeThemeId": "product-first",
  "activeThemeVersion": "0.1.9",
  "slots": [
    {
      "pageKey": "home",
      "schemaId": "product-first/home@1",
      "title": { "zh": "产品首页", "en": "Product home" },
      "hasSchema": true
    }
  ],
  "hostPageKeys": ["home", "about", "contact", "…"]
}
```

**`GET …/home/schema` 响应示例：**

```json
{
  "pageKey": "home",
  "activeThemeId": "product-first",
  "schemaId": "product-first/home@1",
  "mediaRefPaths": ["hero.media", "showcase.items[]"],
  "localizedPaths": ["hero.title", "…"],
  "stringPaths": ["install.code"],
  "jsonSchema": { "$schema": "…", "type": "object", "properties": { … } },
  "source": "theme" 
}
```

`source`: `theme` | `host-fallback`（无 slots 时返回空 schema + 说明）。

### 4.3 Validate 管道（有序）

```text
1. Host hard — MediaRef 叶子 string（全局 walk，已有）
2. Theme paths — mediaRefPaths / stringPaths / localizedPaths 类型约束
3. Theme JSON Schema — 若 jsonSchema 非空（required/类型/枚举）
4. Host fallback shape — 仅当 source=host-fallback 时跑现有 product-first/corporate 启发式
5. Publish gate — CanPublish：schema errors 或 missing/stale（策略按 schema 或 fallback）
```

响应扩展（兼容现有字段）：

```json
{
  "valid": false,
  "schemaKind": "product-first",
  "schemaId": "product-first/home@1",
  "schemaSource": "theme",
  "errors": [
    { "path": "hero.media.caption", "code": "MEDIAREF_TYPE", "message": "…" },
    { "path": "install.code", "code": "TYPE", "message": "expected string" }
  ],
  "translationStatus": { },
  "warnings": [
    { "path": "showcase.items", "code": "RECOMMENDED", "message": "prefer ≤3 items" }
  ]
}
```

- `errors` → 阻断 publish；PUT draft 对 **MediaRef / stringPaths** 仍 400（与 M1 一致）；纯 schema recommended 可仅 warning。  
- MVP：JSON Schema 校验可用轻量库（如 `santhosh-tekuri/jsonschema`）或先只做 **paths 类型规则**，schema 完整校验作 M3b。

### 4.4 whoami 扩展（小）

```json
"capabilities": {
  "themeContent": true,
  "themeContentKeys": ["home", "…"],
  "activeThemeId": "product-first",
  "contentSlots": ["home"]
}
```

`themeContentKeys` 可改为：**Host 白名单 ∩ 主题 slots.pageKey**（有 slots 时）；无 slots 时仍返回全 Host 白名单（现状）。

---

## 5. CLI

| 命令 | 行为 |
|------|------|
| `inkless content slots` | GET `/admin/content/slots` |
| `inkless content schema <pageKey>` | GET `…/schema`（可 `--out schema.json`） |
| `inkless content apply <key> --from-file f --dry-run` | 已有 deep diff + validate；**默认**带服务端 validate（含主题契约） |
| `inkless content apply … --validate-schema` | 显式要求 `schemaSource=theme` 且 valid；否则非 0 退出（CI/agent 严格模式） |
| `inkless content apply … --no-schema` | 仅 Host hard + 写入（应急；文档标明风险） |

**不**新增 `inkless product-first` 命名空间。

Dry-run 输出增量：

```json
{
  "diff": { "summary": {}, "paths": [] },
  "localMediaIssues": [],
  "validate": {
    "schemaId": "product-first/home@1",
    "schemaSource": "theme",
    "valid": true
  }
}
```

---

## 6. 与现有子系统关系

```text
┌─────────────────────────────────────────────────────────┐
│ Theme package                                            │
│  inkless.theme.json#contentSlots + schemas/*.json        │
│  pages/home.tsx 渲染同一形状                              │
└───────────────────────────┬─────────────────────────────┘
                            │ install / activate
                            ▼
┌─────────────────────────────────────────────────────────┐
│ Host                                                     │
│  content_documents | Admin content API | validate 管道    │
│  GET slots/schema | whoami contentSlots                  │
└───────────────────────────┬─────────────────────────────┘
                            │ Bearer ink_
                            ▼
┌─────────────────────────────────────────────────────────┐
│ inkless CLI                                              │
│  content apply/diff/schema  （主题无关命令面）              │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│ Skill（ops/imgli）                                        │
│  某站 golden home.json、文案策略、publish_policy           │
└─────────────────────────────────────────────────────────┘
```

| 内容类型 | 工具 |
|----------|------|
| 主题 C 类槽（home） | `content *` + 本设计 schema |
| D 类 `/p/*` | `pages *` + host presets（不在本设计） |
| 文章 | `articles *` |

---

## 7. 分期

| 阶段 | 交付 | 完成标准 |
|------|------|----------|
| **S0（文档）** | 本文 + theme-contract 指针 | **done** |
| **S1 Host** | 解析激活主题 `contentSlots`；`GET slots` / `GET :pageKey/schema` | **done**（registry + installed config override） |
| **S2 校验** | paths + **JSON Schema 执行**（santhosh-tekuri/jsonschema v6）+ schemaId/schemaSource | **done** |
| **S3 CLI** | `content slots/schema`、`--validate-schema` / `--no-schema` | **done** |
| **S4 官方主题** | product-first + **blog-first** contentSlots；host embed 同步 | **done** |
| **S5 whoami** | `activeThemeId` / `contentSlots` / keys 随 slots 收窄 | **done** |

**S1 最小可用：** 即使暂不跑完整 JSON Schema，只要 **mediaRefPaths / stringPaths / localizedPaths** 进 validate，已能替代大部分启发式。

---

## 8. 风险与兼容

| 风险 | 缓解 |
|------|------|
| 主题未带 schema 文件 | `hasSchema=false`；fallback 启发式；CLI 警告 |
| Schema 与前端类型漂移 | 主题 CI：schema ↔ TS type ↔ golden JSON |
| Host 解析不到 UMD 包内 schema | 安装主题时把 manifest+schemas 拷入 data dir 或 DB blob；内置主题走源码/embed |
| pageKey 膨胀 | 仅允许 ValidPageKeys；新 key 走 Host 发版 |
| Agent token 过大 | `slots` 摘要默认不带 jsonSchema；`schema` 按需拉取 |

**向后兼容：**

- 无 `contentSlots` → 行为 = 今 M1–M3  
- 有 `contentSlots` → validate 更严，旧脏数据 publish 可能 422（符合预期；draft 可修）

---

## 9. 验收故事

```bash
inkless site whoami --site imgli
# activeThemeId=product-first, contentSlots includes home

inkless content slots --site imgli
inkless content schema home --site imgli --json | jq .schemaId
# product-first/home@1

# 故意 bilingual caption → dry-run / validate 失败
inkless content apply home --from-file bad.json --dry-run --validate-schema
# exit ≠ 0

inkless content apply home --from-file good.json --validate-schema
inkless content get home --public --locale zh
```

换主题到未声明 slots 的包：

```bash
inkless content slots   # slots: []
inkless content apply home --from-file x.json --dry-run
# schemaSource=host-fallback；仅 MediaRef + 启发式
```

---

## 10. 决议摘要

1. **主题** 用 `contentSlots` 声明「吃哪些 pageKey、什么 JSON 形状」。  
2. **Host** 提供 slots/schema 发现，并把主题契约接入 validate；**不**为每主题写死分支。  
3. **CLI** 保持通用 `content *`，用 `--validate-schema` 做严格模式。  
4. **MediaRef string 硬规则** 永远在 Host；主题 paths 细化位置。  
5. **Skill** 继续承载站点运营模板，不替代契约。

→ 多主题兼容性 = **契约可发现 + 管道主题无关**，而不是 CLI 主题分叉。
