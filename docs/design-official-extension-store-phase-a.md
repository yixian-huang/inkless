# Design: 官方扩展商店 Phase A — 主题一键安装

**Status:** Proposed (ready for implementation)  
**Date:** 2026-07-29  
**Audience:** product + host/theme maintainers  
**Related:**

- [`docs/adr/0002-theme-host-boundary.md`](adr/0002-theme-host-boundary.md) — Host / Theme 边界  
- [`docs/theme-contract.md`](theme-contract.md) — 契约与 UMD  
- [`docs/product-roadmap.md`](product-roadmap.md) — Marketplace 现状  
- Existing code: `MarketplaceService`, `InstalledTheme` handler, `ThemeManager.loadExternal`, admin `ThemeManagementModal`

---

## 1. 问题与目标

### 1.1 问题

用户期望「浏览市场 → 一键安装 → 可用」。现状是：

| 路径 | 能力 | 缺口 |
|------|------|------|
| `POST /admin/marketplace/items/:slug/install` | 登记 + 下载计数 + 返回 `downloadUrl` | **不执行**主题/插件安装 |
| 主题管理 UI「从 URL 安装」 | `loadExternal` + `POST /admin/themes` | 需用户自己找 UMD URL |
| 内置主题 | seed + activate | 不经市场；升级 pin 靠 host 发版 |
| 插件 zip | `POST /admin/plugins/install` | 与 marketplace 未对接 |

### 1.2 Phase A 目标（In）

1. **官方精选主题目录**可浏览（列表 / 详情 / 版本 / 预览图 / 标签）。  
2. **一键安装官方主题**：校验契约 → 写入 `installed_themes` → 可选激活 + page seed。  
3. **与已安装主题列表统一**：市场安装结果出现在「已安装」中，可激活/卸载。  
4. **信任模型简单明确**：仅官方 allowlist 源；不做开放第三方上架。  
5. **不破坏 ADR-0002**：市场只分发「形态」；不引入第二套内容模型。

### 1.3 非目标（Out of Phase A）

| 不做 | 原因 |
|------|------|
| 开放第三方插件/主题上架 | 信任与审核未就绪 |
| 插件 zip 经市场一键安装 | Phase B；风险更高 |
| 包签名 PKI / 公证体系 | Phase B 与官方插件一起做 |
| 远程多 registry / 镜像 CDN 产品化 | A 可用单一官方 index |
| 主题包热更新到生产 artifact 内置 pin | 仍靠 host 发版；市场管外置/登记 |
| 自动覆盖用户已编辑 page content | 遵守 ADR-0002 切换策略 |

### 1.4 成功标准

- 管理端「扩展 → 主题市场」可列出 ≥ 官方主题（product-first / blog-first / editorial-firm 等有 UMD 的）。  
- 点击「安装」后无需手填 URL，主题进入已安装且 `source=marketplace`（或 `external` + 元数据标明来源）。  
- 点击「安装并激活」后前台呈现该主题 IA（bootstrap `activeTheme` + theme pages）。  
- 契约不兼容时安装失败并给出可读错误（contractVersion / minHostVersion）。  
- 卸载非内置、非当前激活主题成功；激活主题不可卸。

---

## 2. 概念模型

```text
┌─────────────────────────────────────────────────────────┐
│  Official catalog (远程 index，官方托管)                  │
│  themes: [{ slug, themeId, versions[], umdUrl, … }]      │
└──────────────────────────┬──────────────────────────────┘
                           │ fetch (admin / server)
┌──────────────────────────▼──────────────────────────────┐
│  Host Marketplace catalog cache (optional DB rows)        │
│  marketplace_items type=theme                             │
└──────────────────────────┬──────────────────────────────┘
                           │ InstallThemeFromCatalog
┌──────────────────────────▼──────────────────────────────┐
│  installed_themes  (runtime truth for this instance)      │
│  + ThemeManager.loadExternal(umdUrl) on SPA               │
└─────────────────────────────────────────────────────────┘
```

| 概念 | 职责 |
|------|------|
| **Catalog entry** | 「有什么可装」— 元数据 + 包 URL + 契约要求 |
| **Installed theme** | 「本实例已装什么」— 激活、settings、externalUrl |
| **Built-in theme** | 随 host 打包；市场可「标记已内置 / 打开设置」而非重复下载 |

---

## 3. Catalog 规格（官方 index）

### 3.1 托管方式（Phase A 选定）

**单一官方 JSON index**（推荐 GitHub raw / `inkless.run` 静态文件）：

```text
https://inkless.run/marketplace/v1/themes.json
# 或
https://raw.githubusercontent.com/yixian-huang/inkless-marketplace/main/themes.json
```

实例可通过 env 覆盖：

```bash
INKLESS_THEME_CATALOG_URL=https://...
```

默认内置 fallback：host 仓库内嵌一份 **精选清单**（`backend/internal/marketplace/official_themes.json` 或 embed），离线/无网时仍可装「指向已知 CDN/GitHub release 的 UMD」。

### 3.2 `themes.json` schema（v1）

```json
{
  "schemaVersion": 1,
  "updatedAt": "2026-07-29T00:00:00Z",
  "themes": [
    {
      "slug": "product-first",
      "themeId": "product-first",
      "name": "Product First",
      "nameZh": "产品优先",
      "description": "Software product landing…",
      "descriptionZh": "软件产品介绍站…",
      "author": "Inkless CMS",
      "category": "product",
      "tags": ["product", "landing", "oss"],
      "iconUrl": "https://…/icon.png",
      "previewUrl": "https://…/preview.png",
      "repoUrl": "https://github.com/yixian-huang/inkless-theme-product-first",
      "contractVersion": "1",
      "minHostVersion": "0.1.0-alpha.2",
      "latest": {
        "version": "0.1.5",
        "umdUrl": "https://cdn.example/product-first/0.1.5/theme.umd.js",
        "changelog": "…",
        "sha256": "optional-but-recommended",
        "publishedAt": "2026-07-20T12:20:31Z"
      },
      "versions": [
        {
          "version": "0.1.5",
          "umdUrl": "…",
          "changelog": "…",
          "sha256": "…",
          "publishedAt": "…"
        }
      ],
      "defaultFeaturesHint": {
        "publicPages": {
          "home": true,
          "blog": false,
          "contact": true,
          "about": false
        }
      },
      "official": true
    }
  ]
}
```

**规则**

- `themeId` 必须与主题包 `inkless.theme.json#id` / `ThemePlugin.manifest.id` 一致。  
- `contractVersion` 必须被 host `THEME_CONTRACT_SUPPORTED` 接受。  
- `umdUrl` 必须 HTTPS；host 仅允许 **allowlist host**（配置项，默认含官方 CDN / GitHub releases / `inkless.run`）。  
- `sha256` Phase A **推荐**；若存在则服务端或浏览器安装前校验（优先服务端代理下载校验，见 §5）。  
- `official: true` 是 Phase A 唯一允许安装的条目；非 official 忽略或只展示「即将支持」。

### 3.3 与现有 `marketplace_items` 表的关系

两种实现路径（二选一，推荐 **A**）：

| 方案 | 做法 | 利弊 |
|------|------|------|
| **A. Catalog 直连（推荐）** | 安装时读远程/内嵌 JSON；DB 仅写 `installed_themes` | 实现快；少同步状态 |
| B. Sync 进 `marketplace_items` | 定时/手动 sync index → 表；install 仍走现表 | 复用 list API；多一层一致性 |

Phase A 选 **A**；现有 marketplace Admin API 保留给后续 registry 运营，**不阻塞**主题一键装。  
可选：安装成功后写一条 `marketplace_items` 下载计数（best-effort），不作为安装真相源。

---

## 4. 安装状态机

```text
[Catalog] --install--> Validating --> Registering --> (optional) Activating --> Ready
                           |                |                  |
                           v                v                  v
                        Failed           Failed            Failed (theme installed, not active)
```

| 状态 | 含义 | 用户可见 |
|------|------|----------|
| not_installed | 目录有、本机无 | 「安装」/「安装并激活」 |
| installed | `installed_themes` 有行，未激活 | 「激活」「卸载」 |
| active | `is_active=1` | 「当前主题」「打开设置」 |
| builtin | host 已 registerBuiltIn | 「内置 · 激活」；无下载 |
| update_available | 已装 version &lt; catalog latest | 「更新」（Phase A 可做简单覆盖 externalUrl+version） |
| incompatible | contract / minHost 不满足 | 「不兼容」禁用按钮 + 原因 |

### 4.1 安装步骤（Install）

1. 解析 catalog entry + 目标 version（默认 `latest`）。  
2. **Validate**  
   - `official == true`  
   - `contractVersion` compatible  
   - `minHostVersion` ≤ 当前 host version（semver 宽松比较；无法解析则仅 warn）  
   - `umdUrl` host allowlist  
   - 可选 sha256  
3. **Probe UMD（推荐）**  
   - HEAD/GET 前若干字节或完整下载校验；失败则 Abort。  
   - 浏览器侧：可先 `themeManager.loadExternal(umdUrl)` 试加载；失败不写库。  
4. **Register** `installed_themes`：  
   - 若 `theme_id` 已存在：更新 `version` / `external_url` / 元数据（**不**改 `config` 用户设置，除非显式「重置」）  
   - 若不存在：`Create`  
   - `source = "marketplace"`（若列枚举紧，可用 `external` + config/provenance 字段；推荐扩展 source 允许 `marketplace`）  
5. **不**默认改 Features；可选展示「建议 Features」让用户确认（Phase A：仅提示，不自动写）。  
6. 返回 installed theme DTO。

### 4.2 安装并激活（InstallAndActivate）

在 Install 成功后：

1. `SetActive(themeId)`  
2. `SeedThemePages`（与现 `AdminActivate` 相同）  
3. **不覆盖**已有 unified/content 用户数据（ADR-0002）  
4. Invalidate bootstrap cache  
5. 前端 `refetchBootstrap` + 若已 loadExternal 则切换 active

### 4.3 更新（Update）

Phase A 最小实现：

- 已装 marketplace/external 主题：更新 `externalUrl` + `version` 到 catalog latest。  
- 若当前激活：提示刷新前台；可选自动 `loadExternal` 新 URL。  
- **不做**自动 DB 内容迁移。

### 4.4 卸载

- 沿用现规则：不可卸 built-in、不可卸 active。  
- marketplace 来源与 external 相同 soft-delete / delete。

---

## 5. API 设计（Phase A）

在现有 admin 路由旁新增（命名可微调，语义固定）：

### 5.1 目录

```http
GET /admin/extensions/themes/catalog
```

Query: `refresh=1` 强制拉远程 index。

Response:

```json
{
  "schemaVersion": 1,
  "source": "remote|embedded|cache",
  "updatedAt": "…",
  "items": [
    {
      "slug": "product-first",
      "themeId": "product-first",
      "…catalog fields…",
      "installState": "not_installed|installed|active|builtin|incompatible",
      "installedVersion": "0.1.5",
      "incompatibleReason": null
    }
  ]
}
```

`installState` 由服务端合并 `installed_themes` + built-in 列表计算。

### 5.2 安装

```http
POST /admin/extensions/themes/install
Content-Type: application/json

{
  "slug": "product-first",
  "version": "0.1.5",   // optional, default latest
  "activate": true        // optional, default false
}
```

Success `200`:

```json
{
  "theme": { "/* InstalledTheme */" },
  "activated": true,
  "warning": null
}
```

Errors：

| HTTP | 场景 |
|------|------|
| 400 | 校验失败、URL 不在 allowlist、契约不兼容 |
| 404 | slug 不在 catalog |
| 502 | 远程 index / UMD 不可达 |
| 403 | 无 `themes` manage 权限 |

### 5.3 与旧 marketplace install 的关系

- **保留** `POST /admin/marketplace/items/:slug/install` 行为（下载计数 + URL），文档标注 **deprecated for one-click**。  
- 新 UI **只**调 `/admin/extensions/themes/*`。  
- Phase B 再让 marketplace install 转发到真实 plugin/theme installer。

### 5.4 权限

与主题管理一致：`themes` + `manage`（或现有等价 RBAC）。仅登录管理员。

---

## 6. 前端 UX

### 6.1 信息架构

```text
管理后台
└── 外观 / 扩展
    ├── 主题（已安装）          ← 现有 ThemeManagementModal 增强
    │     · 画廊：激活 / 设置 / 卸载
    │     · 从 URL 安装（高级，保留）
    └── 主题市场（Phase A 新增）
          · 官方精选卡片
          · 安装 / 安装并激活 / 更新
          · 不兼容说明
```

Phase A 可将「市场」做成：

- **Tab**：在现有主题弹窗增加「市场」；或  
- **独立路由** `/admin/extensions/themes`（更利于后续插件市场并列）

推荐 **独立路由**，避免弹窗塞满；弹窗保留快捷入口「浏览市场」。

### 6.2 卡片字段

- 预览图、中英文名、一句话描述、作者、版本、标签  
- 状态 badge：内置 / 已安装 / 使用中 / 可更新 / 不兼容  
- 主按钮随状态变化  

### 6.3 安装反馈

- Loading：「校验包… / 注册… / 激活…」  
- 成功：toast + 跳转已安装或自动关闭并 refetch  
- 失败：展示服务端 `error` 原文（中文 message）

### 6.4 高级：URL 安装

保留现有流程，标注「开发者 / 自定义 UMD」，与市场分流。

---

## 7. 安全与信任（Phase A 底线）

| 控制 | Phase A |
|------|---------|
| 仅 `official: true` | 必须 |
| HTTPS umdUrl | 必须 |
| Host allowlist | 必须（env 可配） |
| sha256 | 有则校验；无则仅 official+allowlist |
| 不执行服务端任意 JS | 主题 UMD 仅浏览器加载（与现 external 相同） |
| SSRF | 服务端若代理下载：禁止私网 IP / metadata；超时与 size limit |
| 权限 | 仅管理员 |
| 开放上传主题 zip 到市场 | **不做** |

说明：外置主题 UMD 与今天「填 URL 安装」**同等信任模型**；Phase A 用 official catalog **收紧**来源，不扩大攻击面。

---

## 8. 与内置主题的关系

| 主题 | Phase A 市场行为 |
|------|------------------|
| 已 `registerBuiltIn` 且无 external | 显示「内置」；按钮「激活」调现有 activate API |
| 内置 + 市场有更新 UMD | 可「安装外置版覆盖路径」— **默认不做静默**；文案：「使用市场版（external）需安装并激活，与内置包并行登记需同 themeId 策略」 |

**themeId 冲突策略（锁定）：**

- 同一 `themeId` 全实例唯一一行 `installed_themes`。  
- 内置已存在时：市场「安装」= 更新该行的 `externalUrl`+`version`+`source=marketplace`，**激活后** SPA 优先 `loadExternal`（与现 `source===external'` 逻辑对齐，需支持 `marketplace` 同源加载）。  
- 若不想覆盖内置元数据：禁止安装并提示「请先使用内置激活；市场版同 id 将升级登记行」——产品选 **允许升级登记行**，简单可运维。

`ThemeManagerContext` 加载条件扩展：

```ts
if ((source === "external" || source === "marketplace") && externalUrl) {
  await themeManager.loadExternal(externalUrl);
}
```

---

## 9. 官方 catalog 初始条目（建议）

| slug / themeId | UMD 来源 | 备注 |
|----------------|----------|------|
| product-first | theme repo release `dist/theme.umd.js` | 产品站 |
| blog-first | 同上 | 个人博客；常与 builtin 重叠 |
| editorial-firm | 同上 | 机构/杂志 |
| corporate-classic | monorepo package；可仅 builtin 条目 | 若无独立 UMD，市场只展示内置激活 |
| minimal-starter | builtin only | demo |

发布流程（主题维护者）：

1. 主题仓 `pnpm build` → 上传 release asset  
2. 更新官方 `themes.json` latest + versions  
3. 实例市场「刷新」可见更新  

---

## 10. 实现任务拆分（可执行）

### P0 — 契约与 catalog

| ID | 任务 | 产出 | 验收 |
|----|------|------|------|
| A1 | 定稿 allowlist env + 内嵌 fallback JSON | `official_themes.json` + config | 无网可 list 内嵌条目 |
| A2 | Catalog 拉取与缓存（内存 TTL 5–15m） | Go service | `refresh=1` 绕过缓存；失败回落 embedded |
| A3 | 合并 installState 算法 | unit tests | builtin/installed/active/incompatible 用例 |

### P0 — API

| ID | 任务 | 产出 | 验收 |
|----|------|------|------|
| A4 | `GET /admin/extensions/themes/catalog` | handler + RBAC | 鉴权 401/403；结构符合 §5.1 |
| A5 | `POST /admin/extensions/themes/install` | validate + upsert installed_themes | 非法 URL/契约失败 400 |
| A6 | `activate=true` 走现有 SetActive + SeedThemePages | 复用 | 与手动 activate 行为一致 |
| A7 | source=`marketplace` + 前端 loadExternal 识别 | model/migration if needed | 激活后前台加载 UMD |

### P0 — 前端

| ID | 任务 | 产出 | 验收 |
|----|------|------|------|
| A8 | API client `extensionsThemes.ts` | typed fetch | |
| A9 | 主题市场页/Tab UI | 卡片列表 + 安装按钮 | 安装成功出现在已安装 |
| A10 | 安装并激活 + bootstrap refetch | | 前台 IA 变化 |
| A11 | 不兼容/错误态文案 | | |
| A12 | 入口：主题管理「浏览市场」+ 侧栏 | | |

### P1 — 体验与运维

| ID | 任务 | 产出 | 验收 |
|----|------|------|------|
| A13 | 更新到 catalog latest | | version/url 更新 |
| A14 | sha256 校验（若 catalog 提供） | | 篡改 UMD 失败 |
| A15 | 官方 themes.json 仓或 inkless.run 静态发布 | ops runbook | |
| A16 | E2E：市场安装 product-first 冒烟 | playwright/node | CI 可选 |

### P2 — 显式不做（登记）

- 插件市场一键装  
- 第三方上架  
- 支付/商业主题  

---

## 11. 测试计划

| 层 | 用例 |
|----|------|
| Unit | installState 合并；allowlist；contract 不兼容 |
| API | install 幂等（同 slug 再装更新 version）；activate 种子页 |
| Frontend | mock catalog 安装流 |
| Manual | 真 UMD URL（product-first release）装到本地 dev 实例 |

---

## 12. 文档与发布

1. 更新 [`docs/product-roadmap.md`](product-roadmap.md) Marketplace 条目：Phase A 主题商店状态。  
2. 主题仓 README 增加「发布到官方 catalog」一小节。  
3. `docs-site` 可选「安装官方主题」用户文档（实现后）。  
4. ADR-0002 不变；本设计是分发通道，不改边界。

---

## 13. Phase B 预告（不在本范围实现）

1. `POST …/install` 对 **plugin** 真正调 `InstallPackage`。  
2. 签名 + publisher allowlist。  
3. 统一「扩展市场」IA：主题 | 插件 tab。  
4. 旧 marketplace 表作为运营 registry 真相源。  

---

## 14. 决策摘要（锁定给实现）

| 项 | 选择 |
|----|------|
| Phase A 范围 | **仅官方主题**一键安装 |
| Catalog | 远程 JSON + 内嵌 fallback；env 可覆盖 URL |
| 安装真相源 | `installed_themes` |
| 同 themeId | 单行 upsert；marketplace 可升级 externalUrl |
| 默认激活 | API `activate` 可选；UI 提供「安装并激活」 |
| Features | 不自动改；可提示 |
| 内容 seed | 仅 activate 时 SeedThemePages；不覆盖用户 content |
| 旧 marketplace install | 保留但不用于新 UI |
| 插件 | Phase B |

---

## 15. 建议实现顺序（一条主线）

```text
A1 embed catalog → A2 fetch → A3 state merge
    → A4 list API → A5/A6/A7 install+activate
    → A8–A12 UI
    → A15 发布官方 themes.json
    → A16 冒烟
```

预估（熟悉代码库的单人）：**P0 约 3–5 人日**；含官方 release UMD 对齐与联调另计 1–2 日。
