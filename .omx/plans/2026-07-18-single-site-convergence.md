# Impress 单实例单站点收敛计划

日期：2026-07-18

## 1. 决策与目标

### 架构决策

> 一个 Impress 实例服务一个逻辑站点；多个域名可以作为同一站点的别名；多个独立站点部署多个 Impress 实例。

补充约束：

- `BASE_URL` 是该实例唯一的主域名与 canonical origin；别名域名默认由反向代理 301 到主域名。
- 高可用副本可以共享同一数据库与对象存储，但它们仍属于同一个逻辑站点，不属于“多站点”。
- 独立站点不得共享数据库、上传目录、插件目录、插件数据目录、JWT secret 或后台会话。
- 核心内容表不引入 `site_id`，业务 Repository 不承担租户 Scope。
- `SiteConfig` 表示当前实例的全局站点配置，必须保留；它与多租户 `Site` / `SiteUser` 是不同概念。

### 收敛目标

1. 删除会让用户误以为 Impress 支持多租户的 UI、API、权限与运行时骨架。
2. 保留并明确单实例的 `SiteConfig`、AI `SitePlan`、站点布局等非多租户概念。
3. 用部署层解决域名别名和多个独立站点的运行问题。
4. 将 R3 从“多站点 Platform Beta”改为“插件产品化与多实例运维”。
5. 保持现有 `v0.1.0-alpha.1` 单站点数据可直接升级和回滚。

## 2. 当前事实

- `/admin/sites` 已形成前端页面与后端 CRUD/用户分配/导入导出 API，但前端仍标记为 experimental：
  - `frontend/src/router/config.tsx:31,201-204`
  - `frontend/src/router/adminAccess.ts:39`
  - `frontend/src/pages/admin/components/AdminSidebar.tsx:189-197`
  - `backend/cmd/server/routes.go:476-486`
- `Site` 与 `SiteUser` 已进入 AutoMigrate 和 server 依赖装配：
  - `backend/cmd/server/main.go:192-193,396`
- `SiteResolver`、`RequireSiteContext` 和 `SiteScope` 只有实现与单元测试，没有接入内容主路由和 Repository：
  - `backend/internal/middleware/site.go:17-86`
  - `backend/internal/service/site_service.go:29-35`
- `RBACRole.SiteID`、`UserRole.SiteID` 和 `AssignRoleToUser(... siteID)` 已进入模型/API，但实际权限判断没有按站点过滤：
  - `backend/internal/model/rbac_role.go:11-20,49-55`
  - `backend/internal/repository/role_repository.go:44-45`
  - `backend/internal/repository/role_repository_impl.go:157-164`
- 单实例配置 `SiteConfig` 被主题、功能开关、邮件、Bootstrap、安装流程等生产能力广泛使用，不能随多站点壳删除：
  - `backend/internal/model/site_config.go:9-38`
- 运行配置已经具备多实例所需的大部分隔离参数：`PORT`、`DB_DSN`、`UPLOAD_DIR`、`BASE_URL`、`PLUGIN_DIR`、`PLUGIN_DATA_DIR`：
  - `backend/pkg/config/config.go:9-22`
  - `backend/pkg/config/bootstrap.go:91-116`
- Quick-Box 脚本已支持自定义 `QB_RELEASE_ROOT`、`QB_SYSTEMD_UNIT` 与健康检查 URL，但 env 文件尚未完整持久化 `BASE_URL`、CORS 与插件目录：
  - `scripts/qb-artifact-common.sh:82-95,117-133`
  - `scripts/qb-artifact-activate.sh:112-155`

## 3. 范围边界

### 本次包含

- 撤销内建多站点产品承诺和 R3 硬门槛。
- 删除 `/admin/sites` 前后端功能链。
- 删除未接线的多租户 Resolver、Scope、Site/SiteUser 模型及站点级 RBAC 参数。
- 保留历史数据库表一个发布周期，不在第一阶段执行不可逆 DROP。
- 增加主域名/别名约定和两实例部署范例。
- 更新 Roadmap、架构文档、部署文档和 Omni KB。

### 本次不包含

- Fleet/Control Plane、跨实例统一登录、跨实例内容同步。
- 多实例共享业务数据库或共享本地上传目录。
- 应用内域名别名 CRUD；别名先由 Nginx/Caddy/负载均衡器管理。
- 将 `SiteConfig`、`SitePlan`、`withSiteLayout` 等普通“站点”命名机械重命名。

## 4. 实施步骤

### WP-S0：冻结架构语义与升级前检查

1. 新增 ADR，记录单实例单站点不变量、被否决的共享数据库多租户方案及后果。
2. 更新 `docs/product-roadmap.md` 与 `.omx/plans/2026-07-16-feature-chain-recovery.md`：
   - WP3-A 多站点数据隔离改为 cancelled/deferred-by-demand。
   - R3 不再以多站点为门槛。
   - 新增“单实例边界收口”和“多实例运维”工作包。
3. 实施前检查 `sites`、`site_users`、`roles.site_id`、`user_roles.site_id` 是否有数据。
4. 如果发现用户数据，先导出 JSON/数据库备份并写入升级说明；本轮仍不 DROP 表。

验收：

- 当前 Roadmap 不再把多站点描述为已承诺或下一主工作包。
- ADR 明确 `SiteConfig` 保留、`BASE_URL` 为 canonical、别名由代理层处理。
- 升级检查能给出旧多站点表的行数，不静默丢弃数据。

### WP-S1：撤销产品入口和公共 API

前端：

1. 删除 `frontend/src/pages/admin/sites/page.tsx` 和 `frontend/src/api/sites.ts`。
2. 从 `frontend/src/router/config.tsx` 删除 lazy import 与 `/admin/sites` 路由。
3. 从 `frontend/src/router/adminAccess.ts` 删除 experimental access 项。
4. 从 `frontend/src/pages/admin/components/AdminSidebar.tsx` 删除“站点管理”菜单。
5. 更新相关路由可见性测试，证明 `/admin/sites` 不再存在。

后端：

1. 从 `backend/cmd/server/routes.go` 删除 `/admin/sites` 全部路由和 `Handlers.Site`。
2. 从 `backend/cmd/server/main.go` 删除 Site repository/service/handler 构造与注入。
3. `/admin/sites` 直接变为 404；该功能一直标记为 experimental，不建立新的兼容 API。

验收：

- 前端 bundle 不包含 AdminSitesPage。
- `/admin/sites` 页面不可导航，直达进入标准 404。
- 所有 `/admin/sites*` API 返回 404。
- `SiteConfig`、站点设置页面、主题与 Bootstrap 回归测试保持通过。

### WP-S2：删除多租户运行时与权限语义

1. 删除：
   - `backend/internal/model/site.go`
   - `backend/internal/model/site_user.go`
   - `backend/internal/repository/site_repository.go`
   - `backend/internal/repository/site_repository_impl.go`
   - `backend/internal/service/site_service.go`
   - `backend/internal/handler/site/`
   - `backend/internal/middleware/site.go`
   - 对应多站点测试。
2. 从 AutoMigrate 删除 `model.Site`、`model.SiteUser`，但不自动 DROP 已存在的 `sites`、`site_users` 表，保证旧二进制可回滚。
3. 从 `BuiltinResources` 和 RBAC seed 删除 `sites`；已有 `sites:*` 权限记录在本阶段保持惰性，不影响升级回滚。
4. 从角色模型与 API 删除无效的站点 Scope：
   - `RBACRole.SiteID`
   - `UserRole.SiteID`
   - `AssignRoleToUser` 的 `siteID` 参数
   - 角色分配请求中的 `siteId`
5. 保留 `site_admin` 内置角色名作为兼容标识，但把语义明确为“当前实例管理员”；本轮不做角色名迁移。

验收：

- 生产代码中不存在 `SiteResolver`、`RequireSiteContext`、`SiteScope`、`SiteUser` 或 `/admin/sites` 引用。
- 新角色分配不接受或返回 `siteId`。
- 文章、页面、媒体、菜单、审计、调度等模型没有新增 `site_id`。
- 旧数据库升级后核心 CMS 数据不变，旧版本二进制仍可在 schema 未 DROP 的情况下回滚。

### WP-S3：历史 schema 清理策略

1. 当前发布只停止使用旧表和旧列，不删除：
   - `sites`
   - `site_users`
   - `roles.site_id`
   - `user_roles.site_id`
2. 在后续稳定版本提供显式 cleanup migration；执行前必须：
   - 验证旧表行数。
   - 生成数据库备份。
   - 确认回滚窗口已经结束。
3. cleanup migration 分别验证 SQLite 与 PostgreSQL；SQLite 使用重建表，不依赖 `DROP COLUMN`。

验收：

- 第一阶段升级没有不可逆 schema 操作。
- 后续 cleanup 在空库和含历史行的测试库上都有明确结果；含历史行时必须要求显式确认或备份，不可静默删除。

### WP-I1：域名别名与多实例运维

1. 明确配置约定：
   - `BASE_URL`：唯一主域名，生成 canonical、Sitemap、RSS。
   - `CORS_ALLOWED_ORIGINS`：后台允许的完整 origin 列表。
   - alias domain：由反向代理 301 到 `BASE_URL`；若不跳转，也必须输出主域 canonical。
2. 补齐 `qb_write_env_file`，为每个实例持久化：
   - `BASE_URL`
   - `CORS_ALLOWED_ORIGINS`
   - `PLUGIN_DIR`
   - `PLUGIN_DATA_DIR`
   - `ENABLE_EXTERNAL_PLUGINS`
3. 在部署文档加入两个实例同机运行范例：
   - 唯一 `QB_RELEASE_ROOT`。
   - 唯一 `QB_SYSTEMD_UNIT`。
   - 唯一 `PORT`、`DB_DSN`、`UPLOAD_DIR`、插件目录与 JWT secret。
   - Nginx/Caddy 按域名代理到各自端口。
4. 增加多实例 smoke：同时启动实例 A/B，在两边创建相同 slug、不同内容，验证数据库、媒体、插件、备份与停止/重启互不影响。

验收：

- 同一实例的别名域名跳转到主域或带正确 canonical。
- 两个实例可在同一主机同时运行，端口、数据库、媒体和插件数据无碰撞。
- A/B 可存在相同 slug 且返回不同内容。
- 停止、升级、回滚或恢复其中一个实例不影响另一个。

### WP-D1：真相同步

1. 更新产品 Roadmap、架构说明、部署手册和升级说明。
2. 将旧 Phase 4 多站点文档标记为历史设计，不删除历史记录，但不得再被当前 Roadmap 引用为待实现承诺。
3. 更新 Omni KB Impress 域的 hub、能力现状、技术债、Roadmap 和当前决策。
4. 明确术语：
   - “站点设置” = 当前实例配置。
   - “实例” = 一个独立部署单元。
   - “域名别名” = 指向同一实例的代理层入口。
   - “多个站点” = 多个独立实例。

验收：

- 当前文档和 Omni KB 中不存在“R3 被多站点阻塞”的结论。
- 搜索“多站点”时，当前文档只返回取消决策、多实例替代方案或明确标记的历史设计。

## 5. 验证矩阵

### 静态检查

```bash
rg -n 'SiteResolver|RequireSiteContext|SiteScope|SiteUser|/admin/sites|sites:read' backend frontend
rg -n 'site_id' backend/internal/model backend/internal/repository backend/internal/handler
git diff --check
```

除历史 migration/兼容说明外，第一组应无生产代码命中；第二组不得出现新的内容租户字段。

### 后端

```bash
cd backend
go test ./...
go test -race ./...
go build ./cmd/server ./cmd/impress
```

重点增加：

- `/admin/sites*` 404。
- 角色创建/分配契约不包含 `siteId`。
- `SiteConfig`、Bootstrap、主题、邮件、功能开关保持可用。
- 旧 schema 存在时新版本正常启动。

### 前端

```bash
cd frontend
pnpm lint
pnpm type-check
pnpm test
pnpm build
```

重点增加：

- 后台路由表、权限表和侧边栏不含 `/admin/sites`。
- `/admin/site-config` 仍为 production 且可访问。

### E2E / 运维

- 单实例核心 CMS 发布、定时发布、迁移、系统状态回归。
- alias → primary redirect/canonical 测试。
- 双实例同机隔离 smoke。
- 单实例备份、升级、回滚不影响另一实例。

## 6. 风险与缓解

| 风险 | 缓解 |
| --- | --- |
| 隐藏 API 已被少量用户调用 | 实施前查访问日志和旧表行数；发现数据则先导出，第一阶段不 DROP schema |
| 误删 `SiteConfig` 破坏全局配置 | 建立允许列表；只删除多租户 `Site`/`SiteUser` 链，保留 `site_configs` 全部生产调用 |
| 角色中的 `siteId` 被外部客户端发送 | 更新 OpenAPI/客户端契约；请求出现 `siteId` 时返回明确 400，不静默忽略 |
| 多实例目录或端口冲突 | 启动前校验 release root、unit、port、DB、upload/plugin dirs 唯一性 |
| alias 产生重复 SEO 内容 | 默认 301 到 `BASE_URL`；不能跳转时强制 canonical 指向主域 |
| 过早 DROP 导致无法回滚 | schema cleanup 延后一个稳定发布周期，第一阶段仅停止代码使用 |

## 7. 推荐执行顺序与停止条件

执行顺序：`WP-S0 → WP-S1 → WP-S2 → WP-I1 → WP-D1`；`WP-S3` 延后到下一个稳定发布窗口。

每个工作包都必须独立可提交、可回滚。满足以下条件后，本轮收敛完成：

1. 产品、API 和前端不再暴露多站点能力。
2. 核心运行时没有多租户 Scope 和站点级 RBAC 语义。
3. `SiteConfig` 与单站 CMS 回归全绿。
4. 一个域名别名和两个独立实例均通过验收。
5. Roadmap 与 Omni KB 已改为单实例单站点事实。

## 8. 推荐并行分工

- Lane A（后端）：删除 Site/SiteUser/API/RBAC 多租户语义，负责 schema 兼容。
- Lane B（前端）：删除 sites 页面、API、路由、权限与导航。
- Lane C（运维）：完善实例参数持久化、域名别名和双实例 smoke。
- Lane D（文档/验证）：ADR、Roadmap、升级说明、Omni KB 与最终跨 lane 验收。

Lane A 与 Lane B 可并行；Lane C 可同时推进；Lane D 在前三条 lane 合并后完成最终真相同步。
