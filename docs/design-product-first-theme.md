# Design: **product-first** theme（软件产品介绍站）

**Status:** decisions locked (2026-07-20)  
**Audience:** product + theme authors  
**Related:** [`docs/theme-contract.md`](theme-contract.md), blog-first package, inkless.run ops site

## Decisions (locked)

| Item | Choice |
|------|--------|
| Theme id | **`product-first`** |
| Docs | **External URL only** (`docsUrl` in theme settings / site config). Other products host docs content & service. |
| Changelog surface | Host **`Features.publicPages.blog`** → `/blog` (product release notes optional; default **off** until content exists) |
| Package home | **Independent GitHub repo** (same pattern as `inkless-theme-blog-first`) |
| Layout | Host **`contentProfile: "wide"`** + theme tokens **`maxWidth: 72rem`**; **no** new `product` profile in v1. Product shell = theme-local `ProductPageShell`. |

---

## 0. 问题诊断（当前 inkless.run）

| 现状 | 问题 |
|------|------|
| 激活主题 **`corporate-classic`** | 信息架构是 **咨询/公司官网**：关于我们、优势、核心服务、案例、专家团队 |
| 首页组件 | 复用 host `@/pages/home` 等 **企业 CMS 区块**（hero 背景图 + about + advantages + services） |
| Chrome | `CorporateHeader` / 宽栏「企业站」导航语义 |
| Features 开启 about / advantages / coreServices / cases | 像服务公司，不像 **软件产品 landing** |

**产品运营站要讲的是：** 这是什么软件、解决什么问题、能力边界、如何开始、在哪里看文档/源码/更新。  
**不是：** 公司简介、专家团队、服务套餐。

`blog-first` 方向正确的部分：主题拥有 `/` 信息架构 + chrome + tokens，内容来自 site config / 专用页组件，**不**强绑企业 page 矩阵。  
我们应 **镜像 blog-first 的接入规范**，做平行主题 **`product-first`**。

---

## 1. 目标与非目标

### 目标

1. 新内置主题 **`product-first`**（`@inkless/theme-product-first`），contract **v1**。
2. 适合 **开源/自托管软件产品** 运营站（inkless.run 首发），文案与区块语义是 product，不是 corporate。
3. 接入路径与 blog-first 对齐：`ThemePlugin` + `layoutChrome` + `pages[]` + `inkless.theme.json` + UMD register + backend `pages.json` seed。
4. 激活主题后 Features 默认只开 product 路由；企业路由关闭。
5. **禁止**在主题源码写死「Inkless 营销长文」——identity/CTA 走 site config + 可选 home content schema；fallback 可用 host `PRODUCT_BRAND` 短字段。

### 非目标（v1）

- 不做完整 docs 引擎 / 定价计算器 / 用户账号体系。
- 不替代 Admin；不 fork CMS 业务逻辑。
- 不删除 `corporate-classic`（咨询客户仍可用）。
- v1 不强制独立 GitHub 仓库（可 monorepo `packages/theme-product-first`，提取路径与 blog-first 相同）。

---

## 2. 与现有主题对照

| | **blog-first** | **corporate-classic** | **product-first（本设计）** |
|--|----------------|------------------------|-----------------------------|
| 受众 | 个人作者读者 | 企业服务买方 | **软件用户 / 开发者 / 试用者** |
| 首页 | 作者 + 文章列表 | 企业 hero + 服务区块 | **产品 hero + 能力 + 上手 + CTA** |
| 布局 | reading ~42rem | wide 1200px | **wide + maxWidth 72rem**（v1 不新增 contentProfile） |
| 主导航 | Home, Blog, About | 关于/优势/服务/案例/专家 | **Product, Features, Docs*, Changelog*, Contact** |
| 内容源 | Author + Articles | CMS 企业 page keys | **Site config + product home schema**（+ 可选 blog） |
| Chrome | 阅读向 header | 企业 sticky | **产品 logo + 主 CTA 按钮** |

\* Docs 可为外链或后续 host 路由；Changelog 可映射 host `/blog`。

---

## 3. 信息架构（v1）

### 3.1 路由归属

| 路由 | 所有者 | 说明 |
|------|--------|------|
| `/` | **Theme** `pages[home]` | 产品 landing（hardcoded 页组件） |
| `/features` | **Theme** `pages[features]` | 能力详表（hardcoded；可与 home 共用 schema 片段） |
| `/contact` | Theme **或** host contact | v1 建议 theme 轻量页：邮箱 + GitHub + 社区链接 |
| `/blog`, `/blog/:slug` | **Host** | 可选 Changelog / 发布说明（Features.blog） |
| Docs | **外链 only**（theme setting `docsUrl`） | 文档内容与服务由其他项目提供；Header/Footer 链出去 |
| `/p/*` | Host | 任意补充页 |
| corporate 页 about/advantages/… | **不注册** | Features 默认 off |

### 3.2 首页区块（上→下）

```text
[Header: logo | nav | Docs | GitHub | Get started CTA]
[Hero: product name · tagline · primary CTA · secondary CTA · optional version/badge]
[Logo/social proof strip — optional, settings toggle]
[Features grid: 3–6 cards — title + short description + optional icon key]
[How it works: 3 steps]
[Code / install snippet — optional pre block from settings or home config]
[Latest updates — optional ArticleList if Features.blog on]
[Bottom CTA band]
[Footer: product links · legal · powered by]
```

### 3.3 导航默认

| Nav | Header | Footer |
|-----|--------|--------|
| Home | ✓ | ✓ |
| Features | ✓ | ✓ |
| Blog（若 Features.blog） | ✓ | ✓ |
| Docs（外链） | ✓ | ✓ |
| Contact | ✓ | ✓ |
| GitHub（外链） | utility | ✓ |

---

## 4. 内容所有权（对齐 blog-first）

| 区域 | 配置位置 | 主题行为 |
|------|----------|----------|
| 产品显示名、标语 | Site Config → Identity | Hero 标题 fallback |
| Logo / favicon / OG | Site Config → Brand | Header `BrandMark` |
| 主 CTA URL、文档 URL、GitHub | **Theme settings** + 可选 home config 覆盖 | Header + Hero |
| Hero 标题/副文案 | **content_documents `home`** 专用 schema（见下） | 空则用 identity / `PRODUCT_BRAND` |
| Feature cards / steps | `home` published config | 空则渲染内置 **占位骨架**（短、中性，非公司话术） |
| 文章更新 | Articles + Features.blog | 可选首页「最近更新」 |
| 联系方式 | Site Config / contact content | Contact 页 |

### 4.1 `home` content schema（product-first）

```ts
// conceptual — not corporate hero.backgroundImage-first
type ProductHomeConfig = {
  hero?: {
    eyebrow?: Localized;      // e.g. "Open source CMS"
    title?: Localized;
    subtitle?: Localized;
    primaryCta?: { label: Localized; href: string };
    secondaryCta?: { label: Localized; href: string };
    badge?: Localized;        // e.g. version line
  };
  features?: {
    title?: Localized;
    items?: Array<{ title: Localized; description: Localized; icon?: string }>;
  };
  howItWorks?: {
    title?: Localized;
    steps?: Array<{ title: Localized; description: Localized }>;
  };
  install?: {
    title?: Localized;
    code?: string;            // plain install snippet
    caption?: Localized;
  };
  bottomCta?: {
    title?: Localized;
    subtitle?: Localized;
    primaryCta?: { label: Localized; href: string };
  };
};
```

**明确：** product-first **不读取** corporate 的 `about` / `advantages` / `coreServices` 企业字段。切换主题不自动兼容旧企业 JSON（可另做迁移工具，v1 不做）。

---

## 5. ThemePlugin 接入规范（照抄 blog-first 形状）

### 5.1 包布局（独立仓库）

Repo: **`yixian-huang/inkless-theme-product-first`**（对齐 `inkless-theme-blog-first`）

Host 消费：`pnpm` git dependency `github:yixian-huang/inkless-theme-product-first#main`（或 pinned tag）。

```text
inkless-theme-product-first/
  inkless.theme.json
  package.json                         # @inkless/theme-product-first
  src/
    index.ts                           # ThemePlugin export
    register.ts                        # UMD: __INKLESS_THEME_REGISTER__
    chrome/
      ProductHeader.tsx
      ProductFooter.tsx
      resolveProductCtas.ts
    pages/
      home.tsx
      features.tsx
      contact.tsx
    shell/
      ProductPageShell.tsx             # theme-local; not host contract v1
  types/theme-host-shim.d.ts
  vite.config.ts                       # UMD + ESM like blog-first
  README.md
```

### 5.2 `inkless.theme.json`（契约镜像）

```json
{
  "id": "product-first",
  "name": "Product First",
  "nameZh": "产品优先",
  "version": "1.0.0",
  "contractVersion": "1",
  "tags": ["product", "saas", "oss", "landing"],
  "umd": "dist/theme.umd.js",
  "esm": "dist/theme.es.js",
  "entry": {
    "builtin": "src/index.ts",
    "register": "src/register.ts"
  },
  "peerHost": {
    "sharedGlobal": "__INKLESS_SHARED__",
    "hostGlobal": "InklessThemeHost",
    "registerGlobal": "__INKLESS_THEME_REGISTER__"
  }
}
```

### 5.3 插件字段（必备）

```ts
export const PRODUCT_FIRST_THEME_ID = "product-first";
export const PRODUCT_FIRST_CONTRACT_VERSION = "1";

export const productFirstTheme: ThemePlugin = {
  manifest: {
    id: PRODUCT_FIRST_THEME_ID,
    name: "Product First",
    nameZh: "产品优先",
    description: "Software product landing: hero, features, install CTA, optional changelog",
    descriptionZh: "软件产品介绍站：主视觉、能力、安装引导、可选更新日志",
    type: "theme",
    tags: ["product", "landing"],
    // ...
  },
  contractVersion: "1",
  defaultTokens: productFirstTokens,
  settingSchema: [ /* header CTAs, docsUrl, githubUrl, showInstall, showChangelog */ ],
  tokenPresets: [ /* inkless-dark, clean-light, … */ ],
  pages: [
    { slug: "home", renderMode: "hardcoded", lazyComponent: () => import("./pages/home"), contentKey: "home", nav: {…} },
    { slug: "features", renderMode: "hardcoded", lazyComponent: () => import("./pages/features"), contentKey: "features", nav: {…} },
    { slug: "contact", renderMode: "hardcoded", lazyComponent: () => import("./pages/contact"), contentKey: "contact", nav: {…} },
  ],
  defaultLayout: PRODUCT_DEFAULT_LAYOUT, // contentProfile: "product" | wide-ish
  layoutChrome: { Header: ProductHeader, Footer: ProductFooter },
};
```

### 5.4 Host 注册

1. `themeManager.registerBuiltIn(productFirstTheme)` in `ThemeManagerContext`（与 blog-first 并列）。
2. `BUILTIN_THEME_IDS.PRODUCT_FIRST = "product-first"`。
3. Backend `builtinthemes/constants.go` + `pages.json` 增加 `product-first` 页定义。
4. Seed：可选 `ProductSiteDefaultThemeID = product-first` **仅**用于产品站 bootstrap 文档/脚本；**不要**改 blank-site 默认（仍为 blog-first）。

### 5.5 Host facade 依赖（v1 尽量不 bump contract）

只用现有 `@inkless/theme-host`：

- `BaseSiteHeader`, `BrandMark`, `HeaderUtilities`, `useHeaderSettings`, `useBranding`
- `SeoHead`, `useGlobalConfig`, `useSEODefaults`, `useLocaleMode`, `pickLocaleValue`
- `getPublicArticles`, `ArticleList`（changelog 区）
- `PRODUCT_BRAND`, `ProductPoweredBy`

**若**需要产品壳布局，优先 theme 内 `ProductPageShell`（对标 `BlogPageShell`），v1 **不必**进 host；稳定后再提升到 theme-host 并更新 inventory。

### 5.6 Tokens（产品向，非咨询蓝绿、非阅读衬线）

```ts
// defaultLayout
{
  type: "default",
  contentProfile: "wide",   // host already understands wide vs reading
  header: { style: "sticky" },
  footer: { style: "minimal" }, // product footer is denser via chrome, not "full corporate"
}

// defaultTokens.layout
{
  maxWidth: "72rem",        // ~1152px: landing comfort without full-bleed corporate 1200+ noise
  borderRadius: "0.5rem",
  contentPadding: "1.5rem",
  sectionSpacing: "4.5rem",
  contentGap: "2rem",
}
// colors: near-ink neutrals + one accent
// fonts: sans UI (not Georgia reading)
```

**Layout 决议（v1）：** 不新增 `contentProfile: "product"`。理由见下文「Layout 说明」。Hero 可全宽背景，**内容列**仍受 `maxWidth` 约束（`ProductPageShell`）。

---

## 6. Chrome 行为

### ProductHeader

- 左：`BrandMark`（默认 **logo**，fallback text = identity.name）
- 中：theme pages nav + Features 门控 blog
- 右：`Docs`（外链）、`GitHub`（外链）、**Primary CTA**（Get started / 试用 / 文档）
- **不**默认强调 RSS/作者头像（setting 可关）；博客站的 avatar 模式不是默认

### ProductFooter

- 三列：Product · Resources · Community
- Copyright from site config
- `ProductPoweredBy` 可选（运营站自身可关闭）

---

## 7. 与 Features / 激活流程

激活 `product-first` 时（Admin 或 ops 脚本）：

1. `SetActive("product-first")` + `SeedThemePages`
2. 建议同步 Features（产品预设，可手写 ops 脚本，不必改 seed 全局 blank）：

```json
{
  "publicPages": {
    "home": true,
    "blog": false,
    "contact": true,
    "about": false,
    "advantages": false,
    "coreServices": false,
    "cases": false,
    "experts": false
  },
  "blog": { "comments": false, "rss": false }
}
```

**Features.blog 说明：** 这是 host 的「公开文章频道」开关（路由 `/blog`、`/blog/:slug` + RSS/评论子开关），**不是** theme 内置的 “Feature.log”。  
用于产品站时 = **可选发布说明 / Changelog**；默认 **关**，有内容再开。主题 setting `showChangelog` 仅在 Features.blog 为 true 时在首页挂「最近更新」。

---

## 8. inkless.run 落地（双进程教训内）

| 项 | 值 |
|----|-----|
| 进程 | **`inkless-ops`** only（`/opt/inkless-ops`） |
| 禁止 | 触碰 `/opt/inkless`（yx.ink） |
| 激活 | 产品库 `installed_themes` → `product-first` |
| 配置 | product home schema + identity Inkless + brand assets |

---

## 9. 实施分期

| Phase | 交付 | 验收 |
|-------|------|------|
| **P0** | 包骨架 + tokens + chrome + home 页 + registerBuiltIn + pages.json + 单测/smoke | Admin 可激活；`/` 非企业话术 |
| **P1** | features 页、install 区块、header CTA settings、changelog 条 | inkless.run 切到 product-first |
| **P2** | UMD 构建 + host `theme:umd:smoke` 纳入 product-first | 与 blog-first 同级可外置安装 |
| **P3** | Host 提升 `ProductPageShell` / 更强 section 原语 | 多产品复用 |

---

## 10. 成功标准

1. 产品站激活 **product-first** 后，主导航 **不再出现** 专家/案例/核心服务等企业语义。
2. 首页信息架构符合 §3.2，空配置仍可读（中性占位 + PRODUCT_BRAND）。
3. 主题 **零** `@/` 深依赖；只走 `@inkless/theme-host`（与 blog-first 一致）。
4. 切换回 corporate / blog-first 不破坏对方站点数据（分库或分实例已满足 inkless.run / yx.ink）。
5. 文档：`docs-site/guide/product-first.md` + theme-layout 表增加一行。

---

## 11. Layout 说明（为何这样选）

| 方案 | 优点 | 缺点 | v1 |
|------|------|------|-----|
| **A. `wide` + tokens.maxWidth 72rem** | 零 host contract 改动；与 corporate 共用 profile，靠 tokens 区分密度 | 语义上 “wide” 略泛 | **采用** |
| B. 新增 `contentProfile: "product"` | 语义清晰；可绑 product 专用 spacing | 动 host layout + 兼容面 | 推迟 |
| C. 全站 `reading` | 实现省事 | landing 过窄，像博客 | 否 |

**分层：**

1. **Viewport 全宽**：Header、Hero 背景可 `w-full`。
2. **内容栏 72rem**：标题、特性、安装块、底 CTA 包在 theme-local `ProductPageShell`。
3. **栅格**：特性 1→2→3 列；How-it-works 3 步；避免企业服务卡片墙。
4. **主 CTA 钉在 sticky Header 右侧**，转化路径优先。

## 12. Next

按 **P0 → P1** 开工：独立仓 `inkless-theme-product-first` → host 依赖与 `registerBuiltIn` → 仅 **inkless-ops** 库激活。
