# 设计：Theme as Templates（主题管显示 · Page/Post 管数据）

| 字段 | 值 |
|------|-----|
| 状态 | **Proposed（产品方向锁定）** |
| 日期 | 2026-08-01 |
| 范围 | 内容真源收敛到 Page（unified_pages）+ Post（articles）；主题提供模板/默认 seed/section；迁移 content_documents 与硬编码 C 页 |
| 相关 | [ADR-0002](adr/0002-theme-host-boundary.md)（将修订）、[theme-contract](theme-contract.md)、[theme-content-slots](design-theme-content-slots.md)（演进）、[theme-content-admin-api](design-theme-content-admin-api.md)、[agent-access](agent-access.md) |

---

## 1. 问题与目标

### 1.1 现状摩擦

| 现状 | 后果 |
|------|------|
| 一级产品页多为 **C 类硬编码**（`ThemePlugin.pages[]`） | 观感好，但运营/Agent 写路径分裂 |
| 可运营 JSON 在 **content_documents**（contentSlots） | 与 `pages list` / unified 发布心智不一致 |
| D 类 `/p/*` 是「扩展面」 | Agent 误以为站上「没有页」 |
| 主题换 = 整棵 IA 换 | 与「数据应跨主题保留」 intuitively 冲突 |

### 1.2 产品倾向（本设计锁定）

> **主题管怎么显示；数据靠 Page / Post；主题可以提供 Page/Post 的默认模板（及一次性 seed），但不拥有第二套内容模型。**

对齐：经典 CMS（Ghost / WP 模板）+ 现代「模板绑定实体」；**不**对齐「营销主题把整站写死在 React 组件里当唯一真源」。

### 1.3 目标

1. **一级可运营页（含首页）** 的真源是 **Host Page 实体**（unified_pages）。  
2. **文章** 真源仍是 **articles（Post）**。  
3. **主题** 声明 **templates[]**（渲染契约 + 默认 draft 结构 + schema），可选 **seed 默认 Page**。  
4. **Agent / CLI** 对一级页只走 **pages**（+ articles）；content API 进入兼容/迁移层。  
5. **换主题**：Page/Post 数据保留；换可用模板与 chrome；未知模板 **降级** 为 Host 通用 composable 渲染。

### 1.4 非目标（本设计不一次做完）

- 完整可视化 page builder / 拖拽自由画布  
- 废除主题 React 组件能力（模板仍可由主题组件渲染）  
- 多租户 / 跨站共用 Page  
- 立刻删除 content_documents 表（先双读迁移）

---

## 2. 决策（锁定）

| 项 | 选择 |
|----|------|
| Page 实体 | **unified_pages**（已有 draft/publish/versions/CLI） |
| Post 实体 | **articles**（已有） |
| 首页 | **必须是 Page**（slug 约定见 §4）；不再以 content_documents.home 为生产写路径 |
| 主题职责 | chrome、tokens、**templates**、section 类型、可选 seed；**不**作为运营写路径 |
| 模板绑定 | Page：`mode` + `templateKey`（字符串，主题命名空间）；见 §3（扩展现有 `templateId`） |
| 默认模板 | 主题 `templates[]` + `defaultSeed`；激活时 **仅当目标 slug 不存在或空 draft 时** 写入 |
| 硬编码 `pages[]` | **过渡保留** 作渲染壳 / 未迁移站点 fallback；新主题 **禁止** 把定稿营销文案当唯一数据源（强化 ADR 铁律 3） |
| contentSlots / content API | **演进为模板 schema 的来源之一**，再废弃生产写；见 §7 |
| MediaRef 护栏 | 保留（section/config 全局 string 叶子） |

**分层口诀（修订）：**

```text
Post / Page  = 数据真源（Host 实体 + 发布生命周期）
Theme        = 显示（chrome / templates / sections / tokens）
Seed         = 主题提供的默认 Page/Post 结构（可跳过、不覆盖用户已发布内容）
CLI / Agent  = pages + articles（统一）
```

---

## 3. 主题声明：`templates[]`

### 3.1 `inkless.theme.json`（additive）

```json
{
  "id": "product-first",
  "version": "0.2.0",
  "contractVersion": "1",
  "templates": [
    {
      "key": "product-first/home",
      "appliesTo": "page",
      "title": { "zh": "产品首页", "en": "Product home" },
      "description": "Hero + features + install CTA",
      "routeHint": { "slug": "home", "path": "/", "nav": true, "sortOrder": 0 },
      "schema": "schemas/templates/home.schema.json",
      "defaultSeed": "seeds/home.default.json",
      "renderer": "theme-page",
      "mediaRefPaths": ["sections[type=pf-hero].props.media", "…"],
      "sectionTypes": ["pf-hero", "pf-feature-grid", "pf-install", "pf-bottom-cta"]
    },
    {
      "key": "product-first/features",
      "appliesTo": "page",
      "routeHint": { "slug": "features", "path": "/features", "nav": true },
      "schema": "schemas/templates/features.schema.json",
      "defaultSeed": "seeds/features.default.json",
      "renderer": "theme-page"
    },
    {
      "key": "product-first/post",
      "appliesTo": "post",
      "title": { "zh": "文章", "en": "Post" },
      "renderer": "theme-post",
      "description": "Single post chrome; body is article fields"
    }
  ],
  "defaultTemplates": {
    "page": "product-first/home",
    "post": "product-first/post",
    "home": "product-first/home"
  }
}
```

| 字段 | 说明 |
|------|------|
| `key` | 全局唯一：`{themeId}/{name}`；Page 上存此字符串（见 §3.2） |
| `appliesTo` | `page` \| `post` |
| `routeHint` | seed 用：建议 slug/path/nav；**不**等于强制占路由（冲突规则见 ADR 附录 D） |
| `schema` / `defaultSeed` | 相对主题包路径；Host 安装/内置时可读 |
| `renderer` | `theme-page`（主题组件）\| `composable`（Host section 栈）\| `theme-post` |
| `sectionTypes` | 本模板期望的 section type（校验/编辑器提示） |

**与 contentSlots 关系：**  
`contentSlots` 视为 **v0 契约**；迁移期 Host 可将 `contentSlots` **投影**为「单一 flat config 模板」的 schema。新主题只写 `templates[]`。

### 3.2 Page 上的绑定字段

现有 `unified_pages` 已有：

- `mode`: `template` | `composable`  
- `templateId`: *uint（page_templates 表）*

**本设计补充（二选一，推荐 B 为主、A 兼容）：**

| 方案 | 做法 |
|------|------|
| **A. 沿用 TemplateID** | 主题激活时把 `templates[]` 同步进 `page_templates` 行，Page.templateId 指 FK |
| **B. templateKey 字符串（推荐）** | Page 增加 `template_key`（或 metadata）：`product-first/home`；不依赖 DB 行也可渲染 |

**决议倾向：B + A 可选同步。**  
- 渲染与校验以 **templateKey** 为准（主题卸载后 key 仍在，走 fallback）。  
- Admin 可选把主题模板 **镜像** 到 page_templates 便于列表 UI。

`mode=composable`：忽略主题专用 renderer，纯 Host section 栈（政策页、活动页）。  
`mode=template`：必须有 templateKey；按主题 renderer 画；**config 仍是 Page 的 draft/published JSON**。

### 3.3 Post 模板

- 文章不强制 templateKey；缺省用主题 `defaultTemplates.post`。  
- 模板只影响 **chrome/排版**（字号栏宽、上下篇、作者卡），**不**把正文拆进另一套 JSON（正文仍 `zhBody`/`enBody`）。

---

## 4. 路由与首页约定

### 4.1 公开路径优先级（沿用 ADR 附录 D，修订 C）

1. Host 系统路由（`/admin`、API…）  
2. Host 内容标准入口（`/blog`、`/blog/:slug`…）  
3. **已发布 Page 的公开 path**（pretty 或 `/p/:slug`——见 Open）  
4. 主题声明但尚未落到 Page 的 fallback 壳（**仅迁移期**）  
5. 404  

**修订点：** 生产站「一级 IA」以 **已发布 Page** 为准，不再以「仅主题 pages[] 无数据」为长期常态。

### 4.2 首页

| 约定 | 值 |
|------|----|
| 规范 slug | `home`（或空/`index`，实现选一个 **canonical**，推荐 **`home`**） |
| 公开 path | `/` 映射到 slug=`home` 的 published Page |
| 创建 | 主题 seed 或 setup wizard；Agent：`pages create --slug home --template product-first/home` |
| 写内容 | `pages get-draft` / `put-draft` / `publish`（及 apply） |

### 4.3 主题激活 seed 规则（硬）

对每个 `templates[]` 中带 `routeHint.slug` 的 page 模板：

1. 若 **不存在** 该 slug 的 Page → 创建 draft（或直接 published 仅限 blank site setup，需配置）并填 `defaultSeed`。  
2. 若存在且 **draft 与 published 均为空/从未用户编辑** → 可更新 seed（需 marker，如 `metadata.seededFrom`）。  
3. 若存在 **任何用户编辑痕迹或 publishedVersion>0** → **禁止覆盖**。  
4. 切换主题：不删 Page；若新主题有同名 template 映射则提示「建议更换 templateKey」；否则 composable fallback。

---

## 5. 渲染管道

```text
Request /
  → 解析 Page slug=home (published)
  → 读 templateKey + publishedConfig
  → 若 template.renderer == theme-page 且主题仍安装
        → 主题模板组件(config, siteConfig, articles…)
  → 否则 Host composable SectionRenderer(config.sections)
  → 未知 section type → 降级占位（ADR 附录 B）
```

**主题组件** 从「自拉 `/public/content/home`」改为「吃 Page 的 published config（Host 注入）」。  
过渡期：组件可读 **Page 优先，content_documents 回退**。

---

## 6. API / CLI / Agent

### 6.1 生产写路径（目标态）

| 任务 | 命令 / API |
|------|------------|
| 列一级页 | `inkless pages list` |
| 改首页 | `inkless pages apply <id> --from-file` / draft API |
| 发文章 | `inkless articles *` |
| 上传图 | `inkless media upload` |
| 发现模板 | `GET /admin/themes/active/templates` 或 bootstrap |

### 6.2 兼容层（迁移期）

| 旧 | 行为 |
|----|------|
| `GET/PUT /admin/content/home/*` | 读写映射到 slug=home 的 Page draft/publish；或 410 + 文档 |
| `inkless content apply home` | 内部转 pages apply；打印 deprecation |
| contentSlots schema | 投影为 template schema |

### 6.3 whoami（目标）

```json
"capabilities": {
  "pages": true,
  "articles": true,
  "activeThemeId": "product-first",
  "pageTemplates": ["product-first/home", "product-first/features"],
  "postTemplate": "product-first/post"
}
```

`themeContent` / `contentSlots` 标记 deprecated，等同 pages+templates。

---

## 7. 迁移：content_documents 与 C 页

### 7.1 数据

```text
content_documents.page_key=home (published)
  → unified_pages.slug=home
     mode=template
     template_key=product-first/home
     published_config = 原 flat JSON 或包进 { "sections": … } / { "legacy": flat }
```

- **形态 A（快）：** published_config 仍为 **flat product-first 形状**；主题模板继续吃 flat。  
- **形态 B（净）：** 转为 section 栈（`pf-hero`…）；需一次性转换器。  

**MVP 推荐形态 A**，降低迁移风险。

### 7.2 主题页 `pages` 表（isThemePage）

- 长期：导航改为 **已发布 Page 的 showInNav** + 菜单；theme page 行仅作遗留。  
- 中期：bootstrap `themePages` 与 Page 列表 **合并去重**（slug 优先 Page）。

### 7.3 官方主题改造顺序

1. **product-first**：home（+ features 等一级页）→ Page + templateKey；组件改 props 来源。  
2. **blog-first**：home 以 articles + site config 为主；可选 Page 仅 SEO/intro；post 模板绑定。  
3. **corporate-classic**：企业多页 → 多个 Page + 模板或 composable。

---

## 8. 与 ADR-0002 的关系（待修订摘要）

| 原表述 | 修订方向 |
|--------|----------|
| C = 主题 pages[] 拥有一级 IA | C = 主题 **templates + chrome**；一级 IA 的 **数据** 在 Page |
| `/` 所有者 = 主题 pages[home] | `/` = **published Page(home)** 的渲染，模板来自主题 |
| D = 扩展面、观感不保证 | D 升格为 **默认可运营页模型**；主题模板保证「同级观感」 |
| content schema 在主题页 content | schema 挂在 **template**，数据在 Page config |

新 ADR 补丁可标题：**0002-bis Theme as Templates**（或修订 0002 正文）。

---

## 9. 分期

| 阶段 | 交付 | 完成标准 |
|------|------|----------|
| **T0** | 本文 + ADR 修订草案 + 弃用说明 | 方向书面锁定 |
| **T1** | Page：`template_key`（或等价）；bootstrap `/` 优先 Page(home) | 有 Page 时不再只靠硬编码空壳 |
| **T2** | product-first：seed home Page；模板渲染吃 Page config；双读 content_documents | dogfood 站 pages list 见 home；CLI pages 可改首页 |
| **T3** | 迁移工具 content→page；content API deprecation | 新写路径只走 pages |
| **T4** | `templates[]` 发现 API；contentSlots 只读投影；CLI 别名 | Agent skill 改写完成 |
| **T5** | 硬编码 pages[] 薄壳化 / 文档删除生产 content 写路径 | 模型干净 |

**T1–T2 为关键路径**；未完成前 content API 保持可用。

---

## 10. 风险

| 风险 | 缓解 |
|------|------|
| 换主题后首页「不好看」 | 主题提供高质量 defaultSeed；未知模板 fallback composable + 提示换 templateKey |
| seed 覆盖用户内容 | §4.3 硬规则 + 审计 |
| flat config vs sections 两套 | MVP 锁 flat（形态 A）；sections 后置 |
| 与已实现 contentSlots 投资 | 投影为 template schema，不推倒 CLI 能力 |
| pretty URL `/` vs `/p/home` | 明确 home slug 映射 `/`；其它 Page 仍 `/p/:slug` 或后续 pretty |

---

## 11. 验收故事（目标态）

```bash
inkless site whoami --site imgli
# pages: true, pageTemplates includes product-first/home

inkless pages list --site imgli
# 含 slug=home, status=published, templateKey=product-first/home

inkless pages get-draft HOME_ID --json
# 可编辑 hero/features…（flat 或 sections）

inkless pages apply HOME_ID --from-file home.json --dry-run
inkless pages apply HOME_ID --from-file home.json
# publish_policy 允许时 publish

# 浏览器 / 与 pages 数据一致；无需 content apply home
```

迁移期：

```bash
inkless content apply home …   # 仍可用，stderr deprecation → pages
```

---

## 12. 决议摘要

1. **数据：Page + Post；显示：Theme templates。**  
2. **首页是 Page（slug=home → `/`），不是长期 content_documents 特例。**  
3. **主题提供默认模板与 seed，禁止覆盖已运营内容。**  
4. **Agent 生产路径统一 pages + articles；content API 迁移后弃用。**  
5. **contentSlots / 硬编码 C 页是过渡，不是终态。**  
6. **落地从 T1–T2（Page 绑定 + product-first 首页迁 Page）开始。**

---

## 13. Open questions（不阻塞 T0–T1）

1. 非 home 的 Page 是否全面 pretty URL（去 `/p/`）？  
2. `template_key` 列 vs metadata vs 同步 page_templates FK——实现选型在 T1 敲定。  
3. blank site 默认主题 seed 的 Page 初始 status：draft 还是 published？  
4. blog-first 是否强制创建 home Page，还是「仅 articles + site config、无 Page 也可 `/`」？

**建议默认：** 非 home 暂保持 `/p/:slug`；template_key 独立列；blank seed **published** 最小占位；blog-first **创建** home Page（可极简 seed）以统一 Agent 路径。
