# ADR-0002：Host / Theme 边界与主题规范

- 状态：Accepted
- 日期：2026-07-29
- 作用范围：主题契约、官方主题族、路由与内容归属、`@inkless/theme-host` 演进
- 相关：[`docs/theme-contract.md`](../theme-contract.md)、[`docs/design-product-first-theme.md`](../design-product-first-theme.md)、ADR-0001

## 背景

Inkless 定位为**多方向、多主题的综合建站工具**（对标 Halo）：官方需提供博客、官网、个人站、产品运营站等不同形态的官方主题。

若不划分清晰边界，会出现两类失败模式：

1. **Host 被某一形态绑架**：例如 demo seed 默认企业七页矩阵，平台看起来像「咨询公司 CMS」。
2. **主题越权**：主题源码写死品牌营销长文、第二套内容模型、或深依赖 host 内部 `@/…`。

已有实践（blog-first / product-first）方向正确：主题拥有首页信息架构与 chrome；内容与品牌走 site config / content；文章等实体属 host。本 ADR 把该划分写成**全平台不变量**。

## 决策

### 一句话分层

| 层 | 职责 | 类比 |
|----|------|------|
| **Host** | 内容实体、权限、发布、媒体、SEO 基建、Public/Admin API、可复用渲染原语 | 站点操作系统 + 能力模块 |
| **Theme** | 信息架构、Chrome、首页叙事、tokens、可选页面组件 / section 类型 | 站点形态（形状） |
| **Site config / content** | 品牌、文案、主题 settings、页面 published config | 实例数据 |
| **Features** | 实例启用哪些 host 能力面 | 功能开关，非主题私货 |

**铁律**

1. 主题不得拥有「写入 DB 的业务模型」（文章、用户、权限、迁移…）。
2. Host 不得拥有「某一种站点类型的叙事长文 / 固定 IA」作为平台默认。
3. 实例数据不得写死在主题源码（禁止主题包内官方营销长文）。

### 判定树（任何页面 / 字段 / 组件 / 路由）

1. **换主题后用户仍期望数据在？**  
   - 是 → Host 能力 + 内容模型  
   - 否 → 主题呈现
2. **换站点类型（博客 → 产品站 → 咨询官网）是否必须跟着变？**  
   - 是 → 主题（或主题 `pages[]`）  
   - 否 → Host
3. **多个官方主题会复用同一套交互与数据？**  
   - 是 → Host 原语（`@inkless/theme-host`）  
   - 否 → 主题包内

### 四类对象（勿混成一种）

| 类 | 说明 | 示例 |
|----|------|------|
| **A. Content types** | 平台实体，主题只读 | Article / Media / Category / Tag / Comment |
| **B. Site config** | 跨主题身份 | identity / brand / author / seo / features |
| **C. Theme surface** | 主题声明的形态 | `pages[]` + `layoutChrome` + tokens + `settingSchema`；可选主题页 content schema |
| **D. Generic pages** | 用户扩展页 | unified_pages / sections / `/p/*` |

### 路由归属

| 路由模式 | 所有者 | 规则 |
|----------|--------|------|
| `/` | **主题** `pages[home]`（或显式让位） | 站点类型第一印象由主题定 |
| 主题声明 slug（`/features`、`/author`…） | **主题** | 激活时 seed；切换策略见下 |
| `/blog`、`/blog/:slug`、分类/标签 | **Host** | 内容系统标准入口；主题只包壳/样式 |
| `/admin/*`、`/setup`、`/auth` | **Host** | 主题永不碰 |
| `/p/*`、用户 unified page | **Host + sections** | 主题可贡献 section，不拥有页面生命周期 |
| 外链（Docs、GitHub） | **配置** | theme settings 或 site config，不是 Inkless 路由 |

### Host 正交能力模块（主题只组合）

| 模块 | Host 提供 | 主题如何用 |
|------|-----------|------------|
| Identity | 站名、标语、Logo、locale | `useBranding` / `SeoHead` |
| Publishing | 草稿 / 版本 / 定时 / 可见性 | 只读 published |
| Articles | CRUD、列表、详情、分类标签 | `ArticleList` 等 + 主题壳 |
| Pages | 统一页面 + template/sections | 注册 section 或 hardcode 页 |
| Media | 上传与 URL | config 中的 media ref |
| Navigation | theme pages + menu | `useThemePages` |
| Features | 能力开关 | 响应开关（如是否渲染 blog 区） |
| SEO / 搜索 / RSS | 基建 | `SeoHead`；开关在 features |
| Auth / Admin / Plugin | 运营面 | 主题零参与 |

**刻意不是 Host 内置「站点类型」的东西**（咨询七页矩阵、产品 Features 目录、作者主页叙事）→ 分别由 corporate / product / blog 主题拥有。

### 主题契约五面（规范抽象）

1. **形态声明（IA）** — `ThemePlugin.pages[]` + backend `pages.json` seed  
2. **视觉与 Chrome** — `defaultTokens` + `layoutChrome` + `tokenPresets`  
3. **配置面** — `settingSchema` → `installed_themes.config`（CTA/docsUrl 等；不是全站 SEO）  
4. **内容 schema** — hardcode 页的 schema 由主题文档约定、数据存 host；**必须可空降级**  
5. **Host 原语白名单** — 仅 `@inkless/theme-host` + public API；破坏性变更 bump `THEME_CONTRACT_VERSION`

### 官方主题族（垂直模板，不是皮肤）

| 主题 | 站点方向 | 主题拥有的 IA（示意） | 默认依赖的 Host Features |
|------|----------|----------------------|---------------------------|
| blog-first | 个人/作者 | `/`、`/author` | articles / blog 面 |
| product-first | 软件产品运营 | `/`、`/features`、轻量 contact | identity；blog 可选作 changelog |
| corporate-classic | 企业服务/咨询 | 企业页矩阵或等价 | 企业 publicPages（可关） |
| editorial-firm | 机构/杂志 | 少量页 + `ef-*` sections | unified pages / sections |
| minimal-starter | 扩展 demo | 最小 | 演示路径 |

每个官方主题 README 应固定四段：**受众**、**routes/pages**、**default Features**、**content schema（若有）**。

### 切换主题时的保留策略

| 保留 | 随主题变化 | 策略 |
|------|------------|------|
| 文章、媒体、identity/brand | 主题 pages 结构与默认 nav | seed 新主题页；**不覆盖**用户已有 page content（对齐 editorial-firm 先例） |
| Features 用户显式设置 | 主题建议的 default Features | 首次激活可写默认；再次切换不静默改用户开关（推荐） |
| 旧主题专用 section | — | 降级为空白/fallback section，不删数据 |

### 十条不变量（工程检查清单）

1. Host = 数据与能力；Theme = 形态与呈现；Config = 实例。  
2. 主题不得引入第二套内容模型。  
3. 主题不得依赖 Admin / 私密 API。  
4. 主题只通过 `@inkless/theme-host` 与公开 public API 读数据。  
5. `/` 与主题声明页属主题；`/blog*`、媒体、账户属 Host。  
6. identity / brand / seo 默认属 site config，跨主题保留。  
7. 主题 optional content schema 必须可空降级（占位骨架 / identity fallback）。  
8. Features 开关属实例；主题只响应不发明能力。  
9. 可复用展示积木进 theme-host；单形态 UI 留在主题包。  
10. 破坏性 facade 变更必须 bump `THEME_CONTRACT_VERSION` 并更新 inventory。

## 被否决的方案

### Host 内置「超级站点类型」开关

不采用在 Host 内用大量 feature flag 拼出 blog/corporate/product 全套 IA。会导致平台默认叙事混乱，且第三方主题无法平等接入。

### 主题自带文章/页面存储

不采用主题私有 content store。切换主题会丢数据，违背「换形状不丢内容」。

### 主题深依赖 host `@/` 内部模块

不采用。唯一稳定面是 `@inkless/theme-host` 与 contract inventory。

## 后果

- 新官方主题必须按本 ADR 的 Theme profile（受众 / pages / Features / schema）交付。  
- `docs/theme-contract.md` 的 Ownership 章节与本 ADR 对齐；细节 API 仍以 contract 为准。  
- Demo seed 若使用 corporate 内容，文档须标明「企业示例」，空白站默认仍为 blog-first（既有决策）。  
- 同 `pageKey`（如 `home`）多形态 schema 互踩是已知债：后续应引入 schemaId/version 或切换时的显式 seed 策略（P1，不阻塞本 ADR）。  
- 主题 live-dev（`THEME_*_PATH`）只影响开发解析路径，不改变运行时边界。  
- 官方扩展商店（主题一键安装）是分发通道，不改变本 ADR 边界；设计见 [`docs/design-official-extension-store-phase-a.md`](../design-official-extension-store-phase-a.md)。

## 不变量检查

- 生产代码不得让主题包 `import` host 内部非 facade 路径。  
- 新增 theme-host 导出必须更新 inventory + contract 文档；删除/改语义必须 bump contract major。  
- 激活主题的 page seed 不得无提示地覆盖用户已编辑的 published content（除非运维脚本显式声明 destructive）。  
- 任何「站点类型」默认 IA 必须落在主题 seed，不得成为无主题时的全局硬编码。

## 产品表述

对用户：

> Inkless 提供建站所需的内容与运营能力；官方主题提供博客、官网、产品站等不同「站点形状」。换主题换的是形状与首页叙事，不是丢掉文章和品牌。

对内：

> 能力正交、形态可插拔、实例可配置。
