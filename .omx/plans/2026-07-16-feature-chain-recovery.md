# Impress 功能断链恢复计划

日期：2026-07-16
基线：`main@67fd7d8` / `v0.1.0-alpha.1`
目标：先恢复“功能描述与真实行为一致、核心链路可验证、权限边界可信”的可发布状态，再推进平台能力。

## 0. 当前执行状态（2026-07-16）

当前分支：`codex/wave0-trustworthy-admin`

Wave 0 已完成：

- 登录成功回跳已从失效的 `/admin/content` 修正为 `/admin`。
- 仪表盘“编辑首页”已指向统一页面入口 `/admin/pages`。
- migration 已接入路由、导航和 `system:manage` 权限。
- 后台路由权限元数据已统一供 sidebar 与 route guard 使用；实验性能力默认不出现在导航。
- `/auth/me` 已返回后端实际执行的 RBAC 权限；旧角色用户保留兼容推导。
- 主要 `/admin` API 及 backup/comment/form_submission/qa 模块路由已按
  `read/create/update/delete/publish/manage` 收口。
- editor 的旧角色兼容权限已禁止页面/文章发布，内置 Editor 角色 seed 同步移除发布权限。
- 已增加前端权限单测、后端 RBAC 边界测试和 Playwright 管理端发布链路 E2E。
- CI 已增加 Chromium 安装和管理端 E2E 门禁。

Wave 1-A 已完成：

- `unified_pages` 已成为可编辑的公开路由和自动导航事实源；Bootstrap 只下发已发布页面的路由/导航事实，不下发页面正文。
- 新建页面发布后可直接访问 `/{slug}`；`showInNav` 可驱动页头和页脚自动导航。
- 旧主题页面按 slug 作为迁移兼容补位；相同 slug 始终由 unified page 接管，避免部分迁移时旧路由整体消失。
- 主题 manifest 只负责为匹配 slug 选择 hardcoded 渲染组件；其他 unified page 使用动态区块渲染。
- 页面内容草稿与页面信息已拆分保存：内容发布/回滚/下线只更新发布字段，不会覆盖并发修改的线上路径和导航信息；页面信息保存明确标注为立即生效。
- slug 已限制为小写字母、数字和连字符，并拒绝后台、认证、公开 API、健康检查、指标等保留根路径。
- 页面父子关系支持显式清空，并拒绝不存在、自引用和循环 parent。
- 发布、下线、回滚、删除和页面信息更新都会失效 Bootstrap、公开列表、旧/新 slug 页面缓存。
- 已增加 Bootstrap 契约测试、统一页面发布工作流集成测试、路由/导航兼容单测和状态化 Playwright E2E。
- E2E 已证明：新建页面 → 发布 → `/{slug}` 可访问 → 导航出现 → 页面信息立即更新 → 导航消失 → 下线 → 路由 404。
- 独立 code-reviewer 最终结论为 `APPROVE`，architect 最终状态为 `CLEAR`。

Wave 0 尚余的测试债务：

- 将 `backend/cmd/server/integration_test.go` 中跳过的旧 `/admin/content` 用例迁移为
  unified page 发布、回滚、并发冲突测试。
- 为页面内发布、删除、配置等按钮补充 action 级显隐测试；当前路由和 API 边界已生效，
  但部分页面仍依赖后端 403 作为最终保护。

当前发布判断：

- 管理端登录、主导航、迁移入口、API 权限边界和统一页面公开发布闭环已恢复到可验证状态。
- 当前可承诺统一页面的立即发布、公开路由、自动导航、页面信息即时更新和下线一致性。
- AI 向导、QA、翻译、远端存储、多站点继续按实验性能力隐藏，不纳入本轮发布承诺。
- 下一主工作包为 Wave 1-B：发布权限、审计与事件闭环；随后推进 Wave 1-C 定时发布和 Wave 1-D 系统状态/迁移正式开放。

## 1. 已确认的功能断链

### P0：影响当前发布可信度

1. **登录与后台快捷入口指向已删除路由**
   - 登录成功跳转 `/admin/content`。
   - 仪表盘“编辑首页”跳转 `/admin/content/editor/home`。
   - 路由配置已明确删除旧 content editor，当前入口应落到统一页面系统。

2. **RBAC 只完成了局部接线**
   - 前端路由权限表仅覆盖部分后台页面，其他页面可通过直达 URL 绕过前端守卫。
   - 后端大多数 `/admin` API 只经过 `RequireAdminOrEditor`；只有 analytics、audit、users、roles 等少数路由使用细粒度权限。
   - editor 可能直接调用站点、存储、邮件、AI、市场、迁移、页面发布等超出其角色边界的 API。

3. **统一页面内容、路由与导航不是同一个事实源**
   - 后台页面编辑器读写 `unified_pages`。
   - 前台动态路由和导航仍由旧 `pages` 主题页生成。
   - 新建并发布统一页面后，`/{slug}` 路由和 `showInNav` 不会自然生效；`/p/*` 只是旁路。
   - 后台显示的 URL 语义与真实可访问路径不一致。

4. **定时发布链路只覆盖旧页面模型**
   - 调度器更新 `articles` 和旧 `pages`。
   - `unified_pages` 的 `scheduledAt/status` 未进入调度器。
   - 文章编辑器也没有排期交互；当前定时发布更接近“后端字段存在”而非完整功能。

5. **审计日志只有读取面，没有稳定写入面**
   - `DbWriter` 初始化后未注入 handler、service 或 event bus。
   - 审计日志页面和查询 API 存在，但生产操作不会形成完整审计事件。

6. **质量门禁不能证明关键用户旅程**
   - CI 有 lint、typecheck、单元测试、Go race test 和 build。
   - 没有 Playwright 配置和登录→编辑→发布→前台读取等浏览器 E2E。
   - 部分后端集成测试仍引用已删除的 `/admin/content`，且关键发布/回滚测试存在 skip。

### P1：界面存在，但能力未真正闭环

7. **迁移功能孤岛**
   - 前端 migration 页面、API 客户端和后端任务/SSE API 已存在。
   - 没有路由和导航入口。

8. **AI、向导、QA、翻译形成“可见但不可用”链路**
   - AI 配置只保存在内存，重启丢失。
   - QA 和 Wizard 在启动时捕获 `nil` AI provider，运行时更新 Registry 不会更新它们。
   - Wizard 仍向旧 `pages` 写数据，不能进入统一页面编辑/发布链路。
   - Translation 固定使用 noop provider，成功响应可返回空翻译。

9. **存储配置存在三层断链**
   - 前端发送 `access_key/secret_key/base_path`，后端接收 `accessKey/secretKey/basePath`。
   - “连接测试”只校验字段完整性，不连接远端。
   - 上传运行时始终注册本地存储，保存 S3/OSS 配置不会切换实际上传行为。

10. **System status 后端能力无前端入口**
    - `/admin/system/status` 已存在，但没有页面、API 客户端和导航入口。

### P2：平台能力目前是骨架或占位

11. **多站点仅有管理壳**
    - Site CRUD、SiteResolver、SiteScope 已存在。
    - Resolver/Scope 未接入主路由和内容查询。
    - 核心内容模型没有完整 `site_id`，无法实现真实数据隔离。

12. **Marketplace/Plugin 不是安装系统**
    - marketplace install 只增加下载计数并返回 URL。
    - Plugin Manager 未接入 server 生命周期。
    - gRPC DirectClient 多数方法返回 `not implemented`。
    - CLI `impress plugin create` 是 stub；没有管理前端。

13. **Roadmap 与当前代码真相脱节**
    - `docs/product-roadmap.md` 的“已有/缺失能力”仍停留在 2026-02-18。
    - 多项“缺失”能力已经有实现，多项“已完成”能力实际上只是骨架。

## 2. 发布策略

在修复前按以下原则处理功能暴露：

- **继续开放**：文章、统一页面基础编辑/立即发布、媒体、主题、菜单、全局配置、功能开关、备份。
- **修复后开放**：迁移、系统状态、审计日志、定时发布、细粒度 RBAC。
- **暂时隐藏或标记实验性**：AI 向导、QA、翻译、远端存储、多站点、Marketplace/Plugin。
- API 端同样需要权限或 feature flag；只隐藏菜单不算完成。

## 3. Subagent 工作包

### Wave 0：建立可信基线

#### WP0-A 路由与入口修复

- 角色：`executor-frontend`
- 范围：
  - 修正登录后跳转。
  - 修正仪表盘“编辑首页”入口。
  - 为 migration 增加路由和受权限控制的导航。
  - 删除或标记未使用的旧 pages API/themePages 直连客户端。
- 验收：
  - 登录后进入有效后台页面。
  - 所有后台可见链接均能匹配路由。
  - migration 可从导航进入且刷新后仍可访问。
- 测试：
  - React Router 单元测试。
  - 链接目标与路由表一致性测试。
- 依赖：无。

#### WP0-B RBAC 后端收口

- 角色：`executor-backend-security`
- 范围：
  - 建立 admin API → resource/action 权限矩阵。
  - 给内容、媒体、菜单、主题、邮件、AI、迁移、站点、存储、翻译、页面、模板、市场等路由增加权限中间件。
  - 区分 read/create/update/delete/publish/manage。
  - 为 module routes 提供一致的权限注册方式。
- 验收：
  - editor 不能执行未授权的系统和发布操作。
  - super admin/site admin/editor/author/viewer 的行为与 seed 权限一致。
  - 直接调用 API 与前端菜单权限结果一致。
- 测试：
  - 表驱动权限矩阵测试。
  - 401/403/200 集成测试。
- 依赖：无。

#### WP0-C 前端权限矩阵对齐

- 角色：`executor-frontend-security`
- 范围：
  - 从单一权限元数据生成 sidebar、route guard 和页面操作权限。
  - 补齐 roles、sites、storage、email-settings、translation、qa、wizard、features、site-config、comments、migration 等路由。
  - 发布、删除、配置等按钮使用 action 级权限，不只使用页面级资源权限。
- 验收：
  - 菜单、直达路由、按钮和后端权限矩阵一致。
  - 无权限用户不会看到入口，直达会稳定返回安全落点。
- 测试：
  - 角色路由矩阵测试。
  - 按钮显隐测试。
- 依赖：与 WP0-B 共享权限命名，接口契约先对齐；实现可并行。

#### WP0-D 关键 E2E 与陈旧测试清理

- 角色：`test-engineer`
- 范围：
  - 配置 Playwright。
  - 建立 setup/login/admin navigation 冒烟测试。
  - 清理或重写引用 `/admin/content` 的集成测试。
  - 取消关键 publish/rollback 测试的 skip，或用统一页面链路替代。
- 验收：
  - CI 能证明：登录成功、后台主导航无 404、权限拒绝正确。
  - 不再存在针对已删除 API 的“绿色”测试。
- 依赖：WP0-A；权限 E2E 依赖 WP0-B/WP0-C。

### Wave 1：恢复 CMS 核心闭环

#### WP1-A 统一页面成为页面事实源

- 角色：`architect` + `executor-backend-pages` + `executor-frontend-pages`
- 决策：
  - `unified_pages` 作为内容、状态、路由元数据、导航元数据的唯一可编辑事实源。
  - 旧 `pages` 仅保留为主题清单/迁移兼容层，或完成一次性迁移后删除；禁止继续双写扩散。
- 范围：
  - Bootstrap 返回已发布统一页面的路由/导航信息。
  - 新页面发布后自动获得 `/{slug}` 路由。
  - `showInNav` 进入导航生成。
  - 处理 slug 冲突、保留路由和主题 hardcoded 页面覆盖规则。
  - 明确 `/p/*` 的兼容和废弃策略。
- 验收：
  - 新建 `test-page`→发布→访问 `/test-page` 成功。
  - 勾选导航后出现在目标导航；取消后消失。
  - 回滚/下线后前台和路由状态立即一致。
- 测试：
  - 后端 bootstrap/public pages 集成测试。
  - 前端动态路由测试。
  - Playwright 新建→发布→前台读取→下线。
- 依赖：WP0-B/C；架构决策先于代码并行。

#### WP1-B 发布权限、审计与事件闭环

- 角色：`executor-backend-audit`
- 范围：
  - 将 audit writer 接入登录、内容创建/更新/发布/回滚/删除、权限、配置、迁移、备份恢复等高风险操作。
  - 统一 actor、resource、result、request metadata。
  - 审计写入失败不破坏主业务，但必须可观测。
- 验收：
  - 执行发布/回滚后审计页可检索对应事件。
  - 失败操作也记录 result=failure 和原因摘要。
- 测试：
  - handler/service 审计事件测试。
  - 审计 API 过滤与分页测试。
- 依赖：WP0-B；可与 WP1-A 主体并行。

#### WP1-C 定时发布完整实现

- 角色：`executor-backend-scheduler` + `executor-frontend-editor`
- 范围：
  - 调度统一页面与文章。
  - 通过 service 发布，复用版本、缓存、搜索、事件、审计逻辑，避免直接 SQL 改状态。
  - 增加文章和统一页面排期/取消/修改 UI。
  - 增加待发布队列和失败可观测性。
- 验收：
  - 文章和统一页面均可排期并在到点后公开。
  - 发布会生成正确版本、刷新缓存并写审计。
  - 取消排期后不会发布。
- 测试：
  - fake clock/service 测试。
  - SQLite/PostgreSQL 兼容测试。
  - Playwright 排期表单与队列测试。
- 依赖：WP1-A、WP1-B 的发布 service/事件契约。

#### WP1-D 系统状态与迁移正式开放

- 角色：`executor-frontend-ops` + `test-engineer`
- 范围：
  - 增加 system status 页面和 API 客户端。
  - 完善 migration 入口、空状态、失败重试、SSE 断线恢复。
  - 为迁移增加权限、审计和结果摘要。
- 验收：
  - 管理员可查看版本、数据库、存储等状态。
  - 可完成一个 Markdown 小样导入并看到结果。
- 依赖：WP0-A/B/C、WP1-B。

### Wave 2：兑现已暴露配置

#### WP2-A AI Provider 生命周期

- 角色：`architect-ai` + `executor-backend-ai`
- 范围：
  - 持久化 provider 配置，密钥加密/脱敏。
  - Registry 改为可订阅或请求时解析 provider，避免 QA/Wizard 捕获旧实例。
  - 增加 provider health check。
  - Wizard 改写 `unified_pages`，应用结果进入统一页面编辑器。
- 验收：
  - 配置后无需重启即可使用；重启后仍有效。
  - QA、Wizard、直接 AI API 使用同一 provider。
  - Wizard 创建的页面可在统一页面后台编辑并发布。
- 依赖：WP1-A。

#### WP2-B 翻译真实化

- 角色：`executor-backend-translation` + `executor-frontend-translation`
- 范围：
  - 使用动态 AI TranslationProvider 或明确的外部翻译 provider。
  - 未配置 provider 时返回明确的 503/功能未配置，不返回“成功+空翻译”。
  - 增加预览、批量确认、覆盖保护。
- 验收：
  - 未配置时不会产生空内容写入。
  - 配置后单条、批量、文章翻译可验证。
- 依赖：WP2-A。

#### WP2-C 存储配置真实化

- 角色：`executor-backend-storage` + `executor-frontend-storage`
- 范围：
  - 修正 camelCase API 契约和响应类型。
  - 实现真实 S3/OSS 连接测试。
  - 根据持久配置构建并热切换 StorageProvider。
  - 媒体上传、分片上传、备份远端统一使用 Registry storage。
- 验收：
  - 保存凭证后读取状态正确。
  - 测试会真实访问目标存储。
  - 上传文件实际落到所选 provider，切换和失败回退规则明确。
- 依赖：WP0-B；可与 WP2-A/B 并行。

### Wave 3：平台能力

#### WP3-A 多站点数据隔离

- 角色：`architect-multisite` + `executor-backend-migration` + `test-engineer`
- 范围：
  - 定义租户边界和默认站点兼容策略。
  - 为核心内容、配置、媒体、菜单、评论、表单、审计等增加 `site_id`。
  - 接入 SiteResolver、RequireSiteContext 和 repository scope。
  - 增加跨站访问防护和迁移回滚方案。
- 验收：
  - 两个站点同 slug 内容互不泄露。
  - 域名/子路径能解析到正确站点。
  - 用户只能访问授权站点。
- 依赖：WP1-A、WP0-B；属于高风险串行迁移。

#### WP3-B Plugin/Marketplace 最小真实闭环

- 角色：`architect-plugin` + `executor-backend-plugin` + `executor-cli` + `executor-frontend-plugin`
- 范围：
  - 先限定一个最小 provider 类型完成真实 POC。
  - 正确实现 go-plugin gRPC client 连接。
  - server 接入 Manager 启停、恢复、健康检查。
  - marketplace 实现下载、校验、解包、安装、启用、禁用、卸载。
  - 完成 CLI scaffold 和管理 UI。
- 验收：
  - 外部示例插件可安装、启动、调用、禁用、重启恢复、卸载。
  - 安装失败不会留下伪“已安装”记录。
- 依赖：WP2-C 可作为第一个真实 provider；不得与多站点迁移共享核心文件并行。

## 4. 并行执行图

```text
Wave 0:
  WP0-A ───────┐
  WP0-B ──┬────┼──> WP0-D
  WP0-C ──┘    │
               └──> 发布门槛 R0

Wave 1:
  WP1-A ──┬────────> WP1-C
  WP1-B ──┘
  WP1-D 可与 WP1-A/WP1-B 后半段并行
  完成后达到发布门槛 R1

Wave 2:
  WP2-A ───────> WP2-B
  WP2-C 可独立并行

Wave 3:
  WP3-A 与 WP3-B 分支独立，但均需单独设计评审和迁移/回滚演练
```

## 5. 发布门槛

### R0：可信后台补丁版

- 无失效后台入口。
- API 和前端权限矩阵一致。
- 登录、导航、权限 E2E 通过。
- 实验功能默认隐藏。

### R1：可信 CMS Alpha

- 新建统一页面可通过真实 URL 发布和下线。
- 文章/页面立即发布、回滚、审计可验证。
- migration/system status 正式可达。
- 定时发布若未完成则继续隐藏，不允许半成品入口。

### R2：配置即行为

- AI、翻译、远端存储的保存配置会真实改变运行时行为。
- 未配置或失败时返回明确错误，不出现“成功但无效果”。

### R3：平台 Beta

- 多站点完成数据隔离和迁移演练。
- 至少一个外部插件跑通完整生命周期。

## 6. 文档与真相维护

每个 Wave 完成时同步：

- 更新 `docs/product-roadmap.md` 的现状、已完成、受限、未开始状态。
- 建立“能力状态”枚举：`production` / `beta` / `experimental` / `stub` / `hidden`。
- 更新 API、权限矩阵、数据模型和运维手册。
- 将仓库事实同步到 Omni MCP 的 Impress 独立域。
