# 单实例单站点收敛升级说明

本文说明从包含 experimental 多站点管理骨架的版本升级到单实例单逻辑站点架构时的兼容边界。当前改动只在本地 `codex/single-site-convergence` 分支实施和验证，不代表已经合并或发布。

## 升级前检查

对目标数据库运行只读预检：

```bash
cd backend
go run ./cmd/impress migrate legacy-site-status
go run ./cmd/impress migrate legacy-site-status --json
```

命令报告 `sites`、`site_users` 的表存在性与总行数，以及 `roles.site_id`、`user_roles.site_id` 的列存在性与非空行数；汇总字段为 `hasLegacyData`。它不执行 migration、DELETE 或 DROP。

若任何计数非零：

1. 先生成数据库备份。
2. 导出对应记录，保存迁移审计信息。
3. 确认这些记录只来自 experimental 多站点功能，不要把它们当作核心 CMS 内容自动折叠或删除。
4. 本轮仍可升级，因为新版本不会 DROP 旧表或旧列；如需回滚，旧版本二进制仍可读取原 schema。

## 行为变化

- `/admin/sites` 页面、导航、权限入口和 `/admin/sites*` API 被移除，请求返回标准 404。
- 新建或分配角色不再接受 `siteId`。为避免旧客户端误以为站点级授权仍有效，显式发送 `siteId` 的请求会返回 400，而不是静默忽略。
- `site_admin` 名称继续存在，但代表当前实例管理员，不再表示某个站点范围内的管理员。
- `SiteConfig` 和 `/admin/site-config` 保留，继续表示当前实例的全局配置。

## Schema 兼容与回滚

本阶段不会删除：

- `sites`
- `site_users`
- `roles.site_id`
- `user_roles.site_id`

新版本停止映射和使用这些对象，但 GORM AutoMigrate 不会主动删除它们。回滚时使用升级前备份和原版本二进制；不要在回滚窗口结束前运行任何 cleanup migration。

后续稳定版本如提供 cleanup migration，必须在执行前再次统计历史行、生成备份并要求显式确认。SQLite 清理需要重建表，不能依赖 `DROP COLUMN`；PostgreSQL 需要独立验证。

## 部署迁移

- 将 `BASE_URL` 设为唯一主域名。
- 将后台实际使用的完整 origin 放入 `CORS_ALLOWED_ORIGINS`。
- 在反向代理层把域名别名 301 到 `BASE_URL`。
- 多个独立站点部署为多个实例，并为每个实例设置唯一的 `PORT`、`DB_DSN`、`UPLOAD_DIR`、`PLUGIN_DIR`、`PLUGIN_DATA_DIR`、备份目录、JWT secret、Quick-Box release root 和 systemd unit。

完整示例见部署文档和 Quick-Box artifact 部署说明。

## 升级后验证

1. 确认 `/admin/sites` 与 `/admin/sites/*` 均为 404。
2. 确认 `/admin/site-config`、主题、Bootstrap、邮件和功能开关仍可用。
3. 重新运行旧 schema 预检，确认历史表和列未被删除或修改。
4. 验证 alias 到主域的 301 或 canonical。
5. 若同机部署两个实例，验证相同 slug 可保存不同内容，数据库、媒体、插件数据、备份、停止和重启互不影响。
