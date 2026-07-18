# S1 · 去品牌化与可配置基线 (Generic-ize impress)

> 历史设计更新：本文关于“保留 sites 脚手架”的判断已被 2026-07-18 单实例单站点 ADR 取代；`SiteConfig` 继续保留，但多租户 `Site`/`SiteUser` 脚手架撤销。参见 `docs/adr/0001-single-instance-single-site.md`。

**Status:** Draft — pending user review
**Date:** 2026-05-20
**Owner:** Isian
**Scope tag:** S1 of 4 (S2 博客向主题/section、S3 插件 SDK、S4 我的博客站 应用实例)

## 1. 背景与动机

`impress` 项目当前以"印迹法规咨询 (Blotting Consultancy)"为主体硬编码上线。我希望以它为底座搭建**自己的个人博客 / 个人站点**，并在此之上探索博客方向的插件、通用主题与 section 设计。

把这个目标一锅炖是个 multi-spec 项目，本 spec 仅覆盖**第一个子项目 S1**：把 impress 从"咨询公司站点"演进为"任意人 / 任意品牌可部署的通用站点基线"。S2 / S3 / S4 不在本 spec 范围内，仅在末尾给出衔接说明。

## 2. 目标与非目标

### 2.1 目标

- 站点身份（站名、Logo、品牌色、联系方式、版权、SEO 默认）**全部由 `site_config.global` 驱动**
- 语言模式可在站点级配置（`mono-zh` / `mono-en` / `bilingual`），不再强制双语
- 前端 `i18n` 资源严格只承载 **UI 字符串**，业务文案一律迁出
- 咨询专用页面（about / experts / advantages / cases / core-services）默认下线但代码保留，由 `site_config.features.publicPages` 开关控制
- 默认 SQLite DSN 重命名为 `impress.db`
- 全部用户可见面（公开页面 + Admin 后台 + SEO meta + document.title）**0 "印迹/Blotting" 字串残留**
- 现有印迹 demo 数据通过 `SEED_MODE=demo` 仍可重现（不丢失能力）

### 2.2 非目标

- **不**做真正的插件运行时加载（属于 S3）
- **不**做博客向新 section / 新主题（属于 S2）
- **不**重命名 `go.mod` 模块（`blotting-consultancy` 保留——接口外不可见，重命名代价巨大且收益低）
- **不**引入多语言扩展（>2 种）；i18n 资源结构保持 `zh` + `en`
- **不**引入多作者模型升级；`Article.Author` 仍是 string
- **不**做自定义 CSS 注入 / 自定义域名 / 邮件模板品牌化
- **不**实现 RSS（仅在 features 留占位开关，实际实装属于 S2/S3）

## 3. 上游决定（已与用户确认）

| 决定 | 选定方案 | 影响 |
|---|---|---|
| 多租户态度 | 单租户优先，API 向多租户对齐 | `site_config` 保持 global 单例语义；`sites` 脚手架保留但不启用；`Article/Page` 不引入 `site_id` |
| 语言模式 | 站点级可配（`mono-zh`/`mono-en`/`bilingual`） | 数据模型保留双语字段但变可选；UI 行为按 `localeMode` 切换 |
| i18n 资源职责 | 严格拆分：仅 UI 字符串，业务文案迁出 | `i18n/local/{zh,en}/common.ts` 全面瘦身 |
| Ambition 级别 | B · 用户可见面全覆盖 | Header/Footer/SEO/Admin/Blog 全洗；咨询页 features 下线；go.mod 不动 |

## 4. 数据契约

### 4.1 `site_config.global` 完整 schema

```ts
type LocaleMode = "mono-zh" | "mono-en" | "bilingual";
type Locale = "zh" | "en";
type LocalizedString = { zh?: string; en?: string };

interface SiteConfigGlobal {
  identity: {
    name: LocalizedString;            // 站名；mono 模式只读主语言字段
    tagline?: LocalizedString;        // 副标题 / slogan
    localeMode: LocaleMode;
    defaultLocale: Locale;            // bilingual 模式下的回退；mono 下与主语言一致
  };
  brand: {
    logo: { light: string; dark?: string };  // media URL；dark 缺则复用 light
    favicon: string;
    ogImage: string;                  // 全局 SEO 默认 OG 图
    primaryColor: string;             // 覆盖 theme token --color-primary
    accentColor?: string;             // 覆盖 --color-accent，缺省按主题包默认
  };
  author: {                            // 个人站点 owner profile
    name: string;
    avatar?: string;
    bio?: LocalizedString;
    location?: string;
    socials: Array<{
      kind: "github" | "twitter" | "email" | "rss" | "linkedin" | "custom";
      url: string;
      label?: string;                 // kind=custom 时显示这个
    }>;
  };
  footer: {
    copyright?: LocalizedString;      // 空则按 © {year} {identity.name[locale]} 自动生成
    icp?: string;                     // 中国大陆 ICP；空字符串视为隐藏整段
    extraLinks?: Array<{ label: LocalizedString; url: string }>;
  };
  seo: {
    defaultTitle?: LocalizedString;   // 空则 fallback identity.name
    titleTemplate?: string;           // 默认 "{page} | {site}"
    defaultDescription?: LocalizedString;
    twitterHandle?: string;
  };
}
```

**存储（实际现状）**：现有代码把 global config 存在 **`content_documents` 表，`PageKey = "global"`**（不是 `site_configs.global` — 后者在 model 中定义但无 handler 写入）。bootstrap `globalConfig` 字段由 `contentDocRepo.FindByPageKey("global")` 读取；本 spec 不迁移存储，仅在写入路径上加 schema 校验。

**已发现的实现差距**：legacy `/admin/content/:pageKey/draft|publish` 路由在 content_documents → unified_pages 迁移后被移除，但 `global` PageKey 没有同步迁到 unified_pages，**当前 admin 无法写 global 配置**。S1 需补回 `/admin/global-config` 端点。

**验证**：在写入 `PageKey=global` 时进入新增的 `validateGlobalConfig(JSONMap) error`（接收 JSONMap → 反序列化为 `SiteConfigGlobal` 结构体 → validate `localeMode` 合法值、required 字段不空）。validate 失败返回 400。

同样，**`site_configs.features` 也没有 admin 写入端点**（只在 bootstrap 被读、由 seed 初始化）；S1 需补回 `/admin/features` 端点（写 `site_configs.features`）。

### 4.2 `site_config.features` 子开关

```ts
interface SiteConfigFeatures {
  publicPages: {
    home: boolean;          // 默认 true   — route /
    blog: boolean;          // 默认 true   — routes /blog, /blog/:slug
    contact: boolean;       // 默认 true   — route /contact
    about: boolean;         // 默认 false  — route /about
    experts: boolean;       // 默认 false  — route /experts
    coreServices: boolean;  // 默认 false  — route /core-services
    advantages: boolean;    // 默认 false  — route /advantages
    cases: boolean;         // 默认 false  — route /cases
  };
  blog: {
    comments: boolean;      // 默认 true
    rss: boolean;           // 默认 true（仅占位，S2/S3 实装；当前 toggled on 不会产生任何额外行为，只是为 S2/S3 留位）
  };
}
```

**键名约定**：`publicPages` 的字段名一律使用 JS camelCase（`coreServices`），路由 URL 使用 kebab-case（`/core-services`）。`<FeatureGate>` 接收的是 JS dot-path（`"publicPages.coreServices"`），不是 URL。路由 ↔ 配置键映射在 `frontend/src/router/featureMap.ts`（新增文件）集中声明。

**与现有 features 的兼容**：`site_configs.features` 已存在但 schema 未规范化；本 spec 把它定义清楚。已部署站点首次启动若 `publicPages.*` 缺省，按"旧行为兼容"——所有页面默认开启（保留现状）。新部署 / `BlankSiteSeed` 走"个人博客默认"——只开 home/blog/contact。

**Admin 写入路径**：S1 新增 `GET/PUT /admin/features` 端点（写 `site_configs.features.DraftConfig` / `PublishedConfig`，draft → publish 两步）。

### 4.3 LocalizedString 读取契约

新增前端 helper：

```ts
// src/lib/locale.ts
export function pickLocaleValue(
  value: LocalizedString | undefined,
  mode: LocaleMode,
  defaultLocale: Locale,
  currentLocale: Locale,
): string {
  if (!value) return "";
  if (mode === "mono-zh") return value.zh ?? "";
  if (mode === "mono-en") return value.en ?? "";
  // bilingual：先 currentLocale 再 defaultLocale 再另一语言
  return value[currentLocale] || value[defaultLocale] || value.zh || value.en || "";
}
```

**调用约定**：所有读 `LocalizedString` 的地方走它，**禁止**散落的 `value.zh || value.en` 模式。lint 规则不强制但 PR review 检查。

## 5. 模块设计

### 5.1 前端

#### 5.1.1 `GlobalConfigContext` 类型升级

现有 `GlobalConfigContext` 已 fetch `site_config.global` published；本 spec 扩展其 TypeScript 类型为 §4.1 的 `SiteConfigGlobal`。运行时行为不变。

#### 5.1.2 新 hooks

```ts
// src/hooks/useLocaleMode.ts
function useLocaleMode(): {
  localeMode: LocaleMode;
  defaultLocale: Locale;
  currentLocale: Locale;        // 受 localeMode 收窄
  available: Locale[];          // mono → 单元素；bilingual → ["zh","en"]
  isMono: boolean;
};

// src/hooks/useSEODefaults.ts
function useSEODefaults(): {
  defaultTitle: string;
  titleTemplate: string;        // 含 {page} 和 {site} 占位符
  defaultDescription: string;
  defaultOgImage: string;
  buildTitle(pageTitle: string): string;  // 应用 titleTemplate
};

// src/hooks/useBranding.ts
function useBranding(): {
  siteName: string;
  logo: { light: string; dark?: string };
  favicon: string;
  primaryColor: string;
  author: { name: string; avatar?: string; bio: string; socials: Social[] };
  footer: { copyright: string; icp?: string; extraLinks: ExtraLink[] };
};
```

#### 5.1.3 `<FeatureGate feature="publicPages.about">`

```tsx
function FeatureGate({ feature, children, fallback = null }: {
  feature: string;        // dot path into SiteConfigFeatures
  children: ReactNode;
  fallback?: ReactNode;
}): JSX.Element;
```

用于：
- 公开页面路由出口（`<FeatureGate feature="publicPages.about"><AboutPage /></FeatureGate>`，关闭时渲染 `<NotFound />`）
- 导航菜单项（关闭时不渲染该项）

#### 5.1.4 `<LanguageSwitch>` 行为

读 `useLocaleMode()`，`isMono === true` 时不渲染。`bilingual` 时维持现有切换行为。`i18next-browser-languagedetector` 启动时若 `localeMode !== bilingual`，强制覆盖为 `available[0]`。

#### 5.1.5 Header / Footer / Admin Layout

- `components/feature/Header.tsx` → 读 `useBranding()` + `useLocaleMode()`，删除硬编码 `'Blotting Consultancy'` fallback
- `components/feature/Footer.tsx` + `theme/layouts/ThemedFooter.tsx` → 读 `useBranding().footer` 与 `.author.socials`；ICP 空字符串则整段隐藏；删除硬编码 `readdy.ai` 链接
- `pages/admin/AdminLayout.tsx` → 标题与文案读 `useBranding().siteName`
- `useDocumentTitle(pageTitle)` → 内部走 `useSEODefaults().buildTitle()`，调用点不再传 suffix

### 5.2 后端

#### 5.2.1 `pkg/config` 默认 DSN

```go
const defaultSQLiteDSN = "file:./data/impress.db?cache=shared&mode=rwc"
```

兼容：若 `DB_DSN` 环境变量已显式设置 `blotting.db`，**不**做自动迁移，按现值走。

#### 5.2.2 `internal/seo/meta.go` 默认值

删除 `defaultTitle` / `defaultDescription` 硬编码常量。改为从 `site_config.global.seo` 与 `site_config.global.identity` 读，启动时通过 `SiteConfigService` cache；config 变更通过现有 eventbus（`pages` 已经在用）失效。

#### 5.2.3 `internal/seed` 拆分

- 现有 `Seed()` 重命名为 `DemoSiteSeed()`（保留所有印迹示例数据）
- 新增 `BlankSiteSeed()`：
  - 一个 admin user（账号读环境变量 `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`，缺省 `admin@local` / 随机口令并打印到日志）
  - `site_config.global` 默认值：
    ```json
    {
      "identity": {
        "name": { "zh": "My Site" },
        "localeMode": "mono-zh",
        "defaultLocale": "zh"
      },
      "brand":   { "logo": { "light": "" }, "favicon": "", "ogImage": "", "primaryColor": "#1e40af" },
      "author":  { "name": "", "socials": [] },
      "footer":  { },
      "seo":     { }
    }
    ```
  - `site_config.features` 默认按 §4.2 中"个人博客默认"（home/blog/contact 开，其余关；blog.comments/rss 都 true）
  - 不插入任何文章 / 媒体 / 评论
- 入口由环境变量 `SEED_MODE` 控制：
  - `SEED_MODE=blank`（推荐默认）→ `BlankSiteSeed`
  - `SEED_MODE=demo` → `DemoSiteSeed`（兼容 CI/集成测试）
  - `SEED_MODE=none` → 不 seed
  - 未设置时遵循"已有数据库不重 seed"原则

#### 5.2.4 `site_config` validate

在新增的 `internal/handler/global_config/handler.go`（或 service 层）增加 `validateGlobalConfig(JSONMap) error`：
- `identity.name` 至少一种语言非空
- `identity.localeMode` ∈ {mono-zh, mono-en, bilingual}
- `identity.defaultLocale` ∈ {zh, en}，且 mono 模式下与主语言一致
- `brand.logo.light`、`brand.favicon`、`brand.ogImage` 可空但若非空必须是合法 URL/相对路径
- `footer.icp` 长度上限 100

写入操作通过 `contentDocRepo.UpsertByPageKey(ctx, model.PageKeyGlobal, draft, published)` 落入既有表，不引入新表。

### 5.3 主题包元数据

```ts
// frontend/src/theme/packages/default/index.ts
author: "impress",   // 原 "Blotting Consultancy"
description: "经典蓝绿色调，专业沉稳",   // 原 "印迹咨询经典配色..."

// modern-dark/index.ts
author: "impress",

// warm-earth/index.ts
author: "impress",
```

不引入主题包的 i18n 化（成本高、收益低）。

## 6. PR 切分（执行顺序）

每个 PR 自带测试，过 `pnpm lint && pnpm type-check && go vet && go test -race ./...`。

### PR-1 `feat(global-config): schema + admin endpoint + raw JSON editor`
- 后端：定义 `SiteConfigGlobal` Go struct + `validateGlobalConfig`；新增 `internal/handler/global_config/handler.go` 提供 `GET /admin/global-config`、`PUT /admin/global-config/draft`、`POST /admin/global-config/publish`；写入走 `contentDocRepo` (PageKey="global")
- 前端：`GlobalConfigContext` TypeScript 类型扩展；`pickLocaleValue` helper + 单元测试；新增 `pages/admin/site-config/page.tsx` 简易 JSON 编辑器（textarea + 校验提示），允许填写真值供后续 PR 测试
- **不**包含 `useBranding` / `useLocaleMode` / `useSEODefaults` —— 它们随其消费者落入 PR-2 / PR-3 / PR-5
- 公开页行为无变化（GlobalConfigContext 兼容旧 schema 与新 schema）

### PR-2 `refactor(i18n): UI-only boundary, move business strings out`
- 删除 `i18n/local/{zh,en}/common.ts` 中所有业务文案
- 保留 UI label / button / placeholder / error
- 残留业务 key 改为通用占位（"My Site" / "Welcome"）
- **新增** `useBranding()` hook
- Header / Footer / Admin Layout 接入 `useBranding()`
- 删除硬编码 `'Blotting Consultancy'` fallback 与 `readdy.ai` 推广链接

### PR-3 `feat(locale): localeMode + LanguageSwitch behavior`
- **新增** `useLocaleMode()` hook
- `<LanguageSwitch>` mono 模式不渲染
- `i18next` detector 在 mono 模式强制覆盖
- 单元测试覆盖三种模式

### PR-4 `feat(routing): features.publicPages gates + admin endpoint`
- 后端：新增 `internal/handler/features/handler.go` 提供 `GET /admin/features`、`PUT /admin/features/draft`、`POST /admin/features/publish`；写 `site_configs.features`
- 前端：`<FeatureGate>` 组件 + 单元测试
- 新增 `frontend/src/router/featureMap.ts`（route ↔ feature key 集中映射）
- 路由出口 + 菜单接入
- Admin 增加 features toggles UI（最简列表+开关）

### PR-5 `feat(seo): seo defaults from global-config + form-based admin editor`
- **新增** `useSEODefaults()` hook；所有 `useDocumentTitle` 调用收口（删 suffix 参数）
- 后端 `seo/meta.go` 读 `content_documents.global.seo` + eventbus 失效（global 写入时发 invalidate 事件）
- 前端：把 PR-1 的 raw JSON 编辑器升级为 tabbed form（Identity / Brand / Author / Footer / SEO）
- 测试：mock global config → SEO meta 输出正确

### PR-6 `chore: rename DSN, blank seed, cleanup`
- `pkg/config/config.go` 默认 DSN 改 `impress.db`
- `internal/seed` 拆分为 `BlankSiteSeed` / `DemoSiteSeed`，`SEED_MODE` 入口
- 主题包元数据 author 字段更新
- 文档（`frontend/src/FRONTEND_RENDERING.md` 开头介绍段）更新
- 运行 `scripts/check-brand-residue.sh` 验证 0 残留

## 7. 测试策略

| 层级 | 覆盖 |
|---|---|
| Frontend unit | `pickLocaleValue` 三种 mode 行为；`useSEODefaults` fallback 链；`<FeatureGate>` 各开关组合；mock GlobalConfig 缺失字段的兜底 |
| Frontend integration | Header / Footer 在 mono-zh / mono-en / bilingual 下渲染快照（Vitest + happy-dom） |
| Frontend regression | 现有 `frontend/src/test/pages.regression.test.tsx` 必须更新以反映新默认（咨询页下线） |
| Backend unit | `SiteConfigGlobal.Validate` 拒绝非法 localeMode、required 字段缺失等 |
| Backend integration | `BlankSiteSeed` 后调用公开 API：`GET /public/site-config/global` 返回默认 schema；`POST /admin/site-config/global` 接受合法 schema，拒绝非法 |
| Backend regression | `DemoSiteSeed` 行为完全等同旧 `Seed()`，现有 backend 测试 (`seed_test.go`, `handler_test.go`) 不引入新失败 |
| 脚本 | `scripts/check-brand-residue.sh`：grep `印迹|blotting|Blotting` 范围限定 `frontend/src/**/*.{ts,tsx}` 和 `backend/internal/**/*.go` 与 `backend/pkg/**/*.go`；PR-6 后命中数为 0 |

## 8. 验收标准 (Definition of Done)

1. 全新部署 + `SEED_MODE=blank` 后，可登录 admin、编辑 `site_config.global`、点击发布，公开页面立刻反映新配置
2. 切换 `localeMode=mono-zh` 后整站无英文字段、Header 无语言切换器、SEO 无 `hreflang` 标签
3. Header / Footer / SEO defaults / Admin layout / `document.title` 在任何路由下都无"印迹/Blotting"字样
4. `/about` `/experts` `/advantages` `/cases` `/core-services` 默认不出现在导航；直接访问 URL 返回 NotFound（路由代码保留可复活）
5. `/blog` `/blog/[slug]` 仍可正常列出与查看文章（demo seed 下）
6. `pnpm lint && pnpm type-check && go vet && go test -race ./...` 全绿
7. `scripts/check-brand-residue.sh` 在 `frontend/src` 与 `backend/internal` `backend/pkg` 0 命中
8. `SEED_MODE=demo` 仍能复现印迹示例数据，作为 S4 决定"复活咨询站"或回归测试用

## 9. 风险与缓解

| 风险 | 缓解 |
|---|---|
| 现有 dev DB `blotting.db` 用户内容丢失 | DSN 改为 `impress.db` 是**默认值**而非强制；旧 DSN 若被 `DB_DSN` 显式设置则继续生效。提供 `scripts/migrate-content.go` 可选导出 |
| i18n 大改触发回归 | PR-2 单独成 PR；保留旧 key 一个 PR 周期作 deprecation；后续 PR 验证后再彻底删除 |
| `site_config.features` 旧部署 schema 缺失 | 后端读取时若 `publicPages.*` 缺省则按"全开"兼容（保留现状）；新 `BlankSiteSeed` 走"个人博客默认" |
| Admin 后台改动可能误伤现有用户 | Admin layout 改动作为 PR-2 一部分，集中 review；保留对 `globalConfig` 缺失的兜底显示 |
| SEO 变更影响搜索引擎索引 | 默认仍输出有效 meta；只是数据源从硬编码改为 config；首次部署强制要求填 `identity.name` |

## 10. 衔接 — 后续 S2 / S3 / S4

S1 完成后，三个后续子项目同时被解锁：

- **S2 · 博客向 section 库与主题包**：在 S1 干净基线上设计 article-list / TOC / related / author-card / newsletter / code / callout / gallery 等 section；产出 1 个"阅读优先"主题包替代 default。`useBranding().author` 可直接被 author-card section 复用。
- **S3 · 插件 SDK 与扩展点**：在 S1 之上设计前端 section/route/widget 扩展契约 + 后端 hook/route 契约 + 安装/卸载生命周期。`site_config` 模型可平移到 plugin settings（`PluginSetting` 表已存在）。
- **S4 · "我的博客站"应用实例**：S1 后可立即上线"朴素版"；叠加 S2 为"美观版"；叠加 S3 为"可扩展版"。每一档都可独立部署，无强依赖。

各自独立 spec 待 S1 完成后再 brainstorm。

## 11. 不在本 spec 范围

明确划开：

- 真正的插件运行时加载与扩展点（→ S3）
- 博客向新 section / 新主题包（→ S2）
- 多作者 / 完整用户档案系统升级
- `go.mod` 模块路径重命名
- 多语言 (>2 种) 扩展
- 自定义 CSS 注入 / 邮件模板品牌化 / 自定义域名
- RSS 实际实装（仅留 `features.blog.rss` 占位）
- 移除 `sites` / `installed_theme` / `site_users` 脚手架（保留以备未来多租户）
