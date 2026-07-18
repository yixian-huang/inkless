# ADR-0001：单实例单逻辑站点

- 状态：Accepted
- 日期：2026-07-18
- 作用范围：Impress 核心内容模型、权限边界、运行配置与部署拓扑

## 背景

Impress 曾加入 `Site`、`SiteUser`、`SiteResolver`、`SiteScope` 和 `/admin/sites` 管理骨架，但核心文章、页面、媒体、菜单、审计与调度链路没有按站点隔离。继续补齐共享数据库多租户需要给全部核心模型和查询路径引入租户键、默认站点迁移、跨站访问防护与长期兼容成本。

现有生产能力实际围绕单个逻辑站点工作。`SiteConfig` 已承载当前实例的全局配置、主题、功能开关、邮件与安装流程，它不是多租户 `Site` 模型，不能随多租户骨架删除。

## 决策

一个 Impress 实例只服务一个逻辑站点。

- `BASE_URL` 是该实例唯一的 canonical origin，用于 canonical URL、Sitemap、RSS 和对外链接生成。
- 同一逻辑站点可以有多个域名别名。默认由 Nginx、Caddy 或负载均衡器把别名以 301 重定向到 `BASE_URL`；无法重定向时，页面仍必须输出指向 `BASE_URL` 的 canonical。
- 多个独立站点通过部署多个 Impress 实例实现。实例之间必须隔离数据库、端口、上传目录、插件目录、插件数据目录、备份位置、JWT secret 与后台会话。
- 同一逻辑站点的高可用副本可以共享数据库和对象存储；这属于一个逻辑站点的横向副本，不构成多站点。
- 核心内容模型不引入 `site_id`，业务 Repository 不承担租户 Scope。
- 保留 `SiteConfig`，其术语含义是“当前实例的全局站点配置”。
- 删除多租户 `Site`、`SiteUser`、`SiteResolver`、`SiteScope`、`/admin/sites` 与无效的站点级 RBAC 语义。
- `site_admin` 角色名暂时保留以兼容现有数据库和客户端，但其语义改为“当前实例管理员”。

## 被否决的方案

### 共享数据库多租户

不采用在所有核心表增加 `site_id`、按请求解析域名并向每个 Repository 注入 Scope 的方案。该方案会扩大数据泄露风险、迁移与回滚复杂度，并让插件、调度、审计、媒体和备份都承担租户边界，而当前产品没有共享数据库多租户的明确需求。

### 应用内域名别名 CRUD

本阶段不建设域名别名管理页面或数据库模型。别名属于部署层入口，由反向代理或负载均衡器管理。

## 后果

- `/admin/sites` 前后端入口和 API 直接移除并返回标准 404；该能力此前为 experimental，不新增兼容 API。
- `sites`、`site_users`、`roles.site_id` 与 `user_roles.site_id` 在本轮只停止使用，不执行不可逆 schema DROP。旧版本二进制在回滚窗口内仍可使用原 schema。
- 后续清理历史表和列必须是显式 migration：先统计行数、生成数据库备份、确认回滚窗口结束，并分别验证 SQLite 与 PostgreSQL。
- 多实例运维成为正式支持路径，需要每实例独立的配置、systemd unit、数据目录和健康检查。
- Fleet、跨实例统一登录、跨实例内容同步和共享业务数据库均不在本决策范围内。

## 不变量检查

- 生产代码不得重新引入 `SiteResolver`、`RequireSiteContext`、`SiteScope` 或核心内容 `site_id`。
- `SiteConfig`、`/admin/site-config`、主题、Bootstrap、邮件与功能开关必须持续回归。
- 任何文档不得把本地已验证改动描述为已经合并、部署或发布；发布事实仍以远端提交、CI 和部署证据为准。
