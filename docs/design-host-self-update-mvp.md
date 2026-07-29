# 设计：Host 自更新 MVP（实例内感知 Release + 一键升级）

| 字段 | 值 |
|------|-----|
| 状态 | Draft → 待评审实现 |
| 日期 | 2026-07-29 |
| 范围 | **Inkless 本体**（Go API + SPA），**不是**主题 UMD / catalog |
| 相关 | [ADR-0001 单实例单站点](adr/0001-single-instance-single-site.md)、[站点隔离教训](internal/ops-lessons-site-isolation.md)、[OPS 公开摘要](../OPS.md)、[artifact 激活](quick-box-artifact-deploy-method.md)、[release.yml](../.github/workflows/release.yml)、主题侧 [Phase A 可选自动更新](design-official-extension-store-phase-a.md) §4.3.1 |

---

## 1. 问题与目标

### 1.1 问题

运维侧同时有多实例（例：`yx.ink` / `inkless.run` / `imgli.com`），各自进程、数据、JWT 独立，但 **host 版本容易分叉**。日常跟版目前依赖控制面：

```bash
npc deploy <project> <env> --ref <git-ref> --wait
```

对「已装好、只想升到刚打的 `v*` Release」来说过重；主题已能 catalog 探测/一键升，**host 体验不对称**。

### 1.2 目标（MVP）

1. **探测**：管理后台展示本实例版本 vs 官方 Release 最新版。  
2. **一键升级（本实例）**：下载官方 artifact → 校验 → 写入 **本实例** `RELEASE_ROOT` → 重启 **本** unit → health。  
3. **边界清晰**：不替代 npc；不跨实例；不自动默升 major（可配置策略，默认仅提示）。  
4. **可回滚**：保留 `previous` symlink / 上一 version 目录，一键或脚本回退。

### 1.3 非目标（MVP 明确不做）

| 不做 | 原因 |
|------|------|
| 默认开启的「无人值守自动升级 host」 | 风险远高于主题 UMD；先人工一键 |
| 跨机/集群编排、金丝雀多副本 | 属控制面（npc/QB） |
| 在应用内改 Caddy/DNS/端口/新装机 | 基建职责 |
| 共享 `current` 的多站一次点升三站 | 违反站点隔离；见 §6 |
| cosign / 公证签名（可预留） | MVP 用 SHA256 + 来源 allowlist 即可 |
| 在线编译源码 | 必须吃预构建 artifact |
| 主题市场混排 host 更新 | 入口放「系统状态 / 关于与更新」 |

---

## 2. 角色与部署拓扑前提

### 2.1 实例 = 更新单元

与 ADR-0001 一致：**一个进程 = 一个逻辑站点 = 一次自更新的作用域**。

| 逻辑站（例） | unit（例） | 端口（例） | `RELEASE_ROOT`（例） | 数据 |
|--------------|------------|------------|----------------------|------|
| 个人 / yx.ink | `inkless` | 8088 | `/opt/inkless` | `/opt/inkless/data` |
| 产品 / inkless.run | `inkless-ops` | 8089 | `/opt/inkless-ops` | `/opt/inkless-ops/data` |
| 业务 / imgli.com | （独立 unit） | （独立） | **独立树** | **独立树** |

**硬前提（MVP 启用条件）**：

1. 每实例有 **独立** `RELEASE_ROOT`（至少 `backend/current`、`frontend/current` 不与其他逻辑站共享 inode 目标，除非运维明确接受「同码同升」——**产品默认禁止**）。  
2. 每实例有 **独立** systemd unit + `EnvironmentFile` + `DB_DSN` + JWT。  
3. 进程能分辨「自己是谁」：`INKLESS_RELEASE_ROOT`、`INKLESS_SYSTEMD_UNIT`（或等价）写入 env。

> 若现网仍是「多 unit 共用同一 `frontend/current` symlink」：  
> **先做 install-root 拆分，再开一键更新。** 否则一点更新 = 三站同时换码，隔离失败且无法按站金丝雀。

### 2.2 与 npc 的边界

```text
┌─────────────────────────────────────────────────────────────┐
│ 控制面 npc / QB artifact                                    │
│  · 首装、换机、改 env/端口/代理                             │
│  · 破坏性 major / 数据迁移窗口                              │
│  · 多环境编排、回滚困难时的人工门禁                         │
│  · 共享树拆分、防火墙、密钥轮换                             │
└─────────────────────────────────────────────────────────────┘
                              │ 产出可发布 Release
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ GitHub Release (v*) + sha256 artifacts                      │
│  · frontend-{ver}.tar.gz (+ .sha256)                        │
│  · backend-{ver}.tar.gz  (+ .sha256)                        │
└─────────────────────────────────────────────────────────────┘
                              │ 各实例独立拉取
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ 实例管理后台「关于与更新」                                   │
│  · 探测 latest / channel                                    │
│  · 一键升级 **本 RELEASE_ROOT + 本 unit**                    │
│  · 本实例 previous 回滚                                     │
└─────────────────────────────────────────────────────────────┘
```

| 场景 | 走谁 |
|------|------|
| 日常 patch / 小版本跟版（三站可不同步） | **站内一键** |
| 首装、新域名、新 unit | **npc** |
| 怀疑 artifact 坏、health 不过 | **npc rollback / 站内 previous** |
| 只升产品站、故意留个人站旧版 | **站内各自决定**（这是自更新的核心收益） |
| 改 `DB_DSN`、拆共享 symlink | **npc + 人** |

**原则**：npc 发布 **可升级包**；站内消费 **已发布包**。站内 **不** `git pull && go build`。

---

## 3. 探测源（Probe）

### 3.1 权威源（MVP 默认）

**GitHub Releases API**（公开仓库无需 token；私有则 env token）：

```http
GET https://api.github.com/repos/{owner}/{repo}/releases/latest
GET https://api.github.com/repos/{owner}/{repo}/releases?per_page=10
```

| 配置项 | 默认建议 | 说明 |
|--------|----------|------|
| `INKLESS_UPDATE_REPO` | `yixian-huang/inkless` | owner/repo |
| `INKLESS_UPDATE_CHANNEL` | `stable` | 见下 |
| `INKLESS_UPDATE_API_BASE` | `https://api.github.com` | 可换 GHES |
| `INKLESS_UPDATE_CHECK_TTL` | `15m` | 服务端缓存，防刷 API |
| `INKLESS_GITHUB_TOKEN` | 空 | 提高 rate limit；可选 |

### 3.2 Channel 语义

| Channel | 规则 |
|---------|------|
| `stable` | 仅 `prerelease=false` 的最新 tag；忽略 draft |
| `latest` | 含 prerelease 的时间序最新（自用金丝雀） |
| `pinned` | 不自动比最新；只显示当前 + 手动填目标 version |

MVP UI：下拉 channel（默认 stable）；高级可输入目标 tag。

### 3.3 可选镜像（增强，非阻塞）

为降低 GitHub API 依赖 / 国内可达性，可在产品站静态托管：

```text
https://inkless.run/releases/v1/channel-stable.json
```

形状示例：

```json
{
  "schemaVersion": 1,
  "channel": "stable",
  "updatedAt": "2026-07-29T12:00:00Z",
  "latest": {
    "version": "v0.2.0",
    "publishedAt": "2026-07-29T11:00:00Z",
    "notesUrl": "https://github.com/yixian-huang/inkless/releases/tag/v0.2.0",
    "assets": [
      {
        "name": "backend-v0.2.0.tar.gz",
        "url": "https://github.com/yixian-huang/inkless/releases/download/v0.2.0/backend-v0.2.0.tar.gz",
        "sha256": "…"
      },
      {
        "name": "frontend-v0.2.0.tar.gz",
        "url": "https://github.com/yixian-huang/inkless/releases/download/v0.2.0/frontend-v0.2.0.tar.gz",
        "sha256": "…"
      }
    ]
  }
}
```

优先级：`INKLESS_UPDATE_MANIFEST_URL`（若设）→ 否则 GitHub API。  
CI 在 `release.yml` 成功后可同步写该 JSON（后续任务，MVP 可只做 GH API）。

### 3.4 本机版本

已有：`GET /admin/system/status` → `application.version`（构建注入）。

补强（MVP 建议一并做）：

| 字段 | 来源 |
|------|------|
| `application.version` | 编译 ldflags / 现有 `appVersion` |
| `application.releasedAt` | 可选，build-info |
| `application.releaseRoot` | env `INKLESS_RELEASE_ROOT`（脱敏展示路径 basename 即可） |
| `application.updateCapable` | bool：是否满足 §2.1 前提 |
| `application.updateBlockedReason` | 如 `shared_release_root` / `missing_unit` / `disabled` |

版本比较：规范化 tag（去 `v` 前缀）后用 **semver**（与主题 `VersionIsNewer` 同类工具，可复用 `golang.org/x/mod/semver`）。  
非 semver 的 `git describe` 脏版本：只展示、不自动判定「可升级」，引导走 npc 或指定 tag。

---

## 4. 产物格式（Artifacts）

### 4.1 复用现有 Release 契约（强制）

与 [`.github/workflows/release.yml`](../.github/workflows/release.yml) / `scripts/build-*.sh` **完全一致**，不发明第三套包：

| 文件 | 内容 |
|------|------|
| `backend-{version}.tar.gz` | 含 `inkless-api-*` 二进制的 backend 包 |
| `backend-{version}.tar.gz.sha256` | 校验 |
| `frontend-{version}.tar.gz` | SPA `out/` |
| `frontend-{version}.tar.gz.sha256` | 校验 |

可选（与 qb activate 对齐，增强）：

| 文件 | 内容 |
|------|------|
| `artifact-manifest.json` | version + components[{path,sha256}] |
| `build-info.json` | commit、时间、go/node 版本 |

MVP **最少**：两个 tar.gz + 两个 sha256；若 Release 带 manifest 则优先用 manifest。

### 4.2 激活布局（与 qb-artifact-activate 同构）

目标树（每实例自己的 `RELEASE_ROOT`）：

```text
$RELEASE_ROOT/
├── backend/
│   ├── versions/{version}/
│   ├── current -> versions/{version}
│   └── previous -> versions/{old}     # 可选但 MVP 强烈建议
├── frontend/
│   ├── versions/{version}/
│   ├── current -> versions/{version}
│   └── previous -> …
├── data/          # 绝不被更新器改写
├── uploads/       # 绝不被更新器改写
└── .env           # 绝不被更新器改写
```

激活逻辑 **优先 shell-out 到仓库脚本**（行为已验证）：

```bash
QB_ARTIFACT_INCOMING=… QB_VERSION=… QB_RELEASE_ROOT=… \
  QB_SYSTEMD_UNIT=… bash scripts/qb-artifact-activate.sh
```

站内更新器负责：下载到 incoming → 调 activate → 重启 unit → health。  
**不要**在 Go 里重写一套解压/symlink 规则（减少双实现漂移）。

### 4.3 下载 URL 解析

从 Release assets 按 **文件名精确匹配**：

- `backend-{version}.tar.gz`
- `frontend-{version}.tar.gz`
- 对应 `.sha256`（或从 `*.sha256` 文件内容解析）

`version` 必须与 tag 一致（含或不含 `v` 与构建脚本约定统一；**以 Release tag 为权威**，文件名与 tag 对齐）。

---

## 5. 安全校验

### 5.1 威胁模型（MVP 覆盖）

| 威胁 | 缓解 |
|------|------|
| 管理员会话被盗 → 任意 RCE | 高权限门禁 + 确认文案 + 审计日志；可选二次确认输入版本号 |
| 中间人篡改包 | **仅 HTTPS** + **SHA256 必过** |
| SSRF（更新器当代理） | 下载 URL host **allowlist** |
| 指到错误实例目录 | `RELEASE_ROOT` 仅来自 **进程 env**，API **禁止** body 传入任意路径 |
| 跨站升错机 | 无集群 API；无横向凭证 |
| 降级/回滚到恶意旧包 | 回滚只允许 **本机已存在的** `versions/*`，不从网上下旧包（除非显式「安装指定 tag」且仍走校验） |

### 5.2 Allowlist

默认允许下载 host（可 env 扩）：

- `github.com`
- `objects.githubusercontent.com`
- `release-assets.githubusercontent.com`
- （若用镜像）`inkless.run` 仅 `/releases/` 路径前缀

拒绝：IP 字面量、非 HTTPS、元数据地址、内网 CIDR。

### 5.3 校验流水线（必须全过才 activate）

1. 解析目标 version + asset 列表  
2. URL host/path allowlist  
3. 下载到 `$RELEASE_ROOT/var/updates/incoming/{jobId}/`（或 `/var/tmp/inkless-update/{instance}/`）  
4. 对每个 tar：计算 sha256，与 `.sha256` 或 manifest **恒等**  
5. 可选：tar 内路径穿越检查（拒绝 `..`）  
6. 调 activate（脚本内再次 checksum，双保险）  
7. `systemctl restart $UNIT`（或 `SIGHUP` 若未来支持；MVP 用 restart）  
8. Health：轮询 `http://127.0.0.1:$PORT/…`（本地，不走公网）直到 timeout  
9. 失败：**不**删 previous；标记 job failed；UI 提供「回滚到 previous」

### 5.4 权限与审计

| API | 权限 |
|-----|------|
| 检查更新 | `system:manage`（与 system status 同级，或新 `system:update` 若要细分） |
| 执行升级 / 回滚 | **仅** `system:manage`（建议未来独立 `system:update`） |

审计字段：actor、fromVersion、toVersion、jobId、result、error、duration、client IP。

### 5.5 功能开关

```bash
INKLESS_SELF_UPDATE_ENABLED=true   # 默认 false 更安全；自托管文档说明开启条件
```

未开启时：API 返回 `updateCapable=false`，UI 只读展示版本 +「请用控制面部署」。

### 5.6 后续（非 MVP）

- cosign / minisign 签名  
- 升级前自动 `backup` 模块打 DB 快照  
- 维护模式页（升级窗口 503）

---

## 6. 站点隔离检查清单（上线前必过）

在 **每一个** 目标域名上执行（yx.ink / inkless.run / imgli.com 各做一遍）。

### 6.1 拓扑确认

- [ ] 该域名的反代 upstream **端口** 是多少？  
- [ ] 对应 **systemd unit** 名称？  
- [ ] unit 的 `EnvironmentFile` / `WorkingDirectory`？  
- [ ] `DB_DSN` 指向哪棵 data 树？（与其他域名 **不得** 相同文件）  
- [ ] `JWT_SECRET` 是否独立？  
- [ ] `BASE_URL` 是否等于该站 canonical？  
- [ ] `FRONTEND_DIR` / `RELEASE_ROOT` 是否 **本站专用**？

### 6.2 共享代码风险（自更新阻断项）

- [ ] `readlink -f backend/current` 与另一站是否同一路径？  
- [ ] `readlink -f frontend/current` 是否共享？  
- [ ] 若共享：**禁止** 对该站开启 `INKLESS_SELF_UPDATE_ENABLED`，先拆：

```text
/opt/inkless/       → yx.ink
/opt/inkless-ops/   → inkless.run
/opt/imgli/         → imgli.com   # 示例名
```

每树自有 `backend/versions`、`frontend/versions`、`current`、`previous`。  
二进制 **内容** 可以相同（各下一份），**路径** 必须独立。

### 6.3 更新器 env（每 unit）

- [ ] `INKLESS_SELF_UPDATE_ENABLED=true`（仅拆分完成后）  
- [ ] `INKLESS_RELEASE_ROOT=/opt/<this-site>`  
- [ ] `INKLESS_SYSTEMD_UNIT=<this-unit>`  
- [ ] `INKLESS_UPDATE_REPO=yixian-huang/inkless`  
- [ ] `INKLESS_UPDATE_CHANNEL=stable`  
- [ ] 进程用户对 `$RELEASE_ROOT/backend|frontend|var` **可写**  
- [ ] 进程用户具备 **仅本 unit** 的 restart 权限（polkit / sudoers 白名单，忌 NOPASSWD ALL）

### 6.4 验收（每站）

- [ ] 升级前：`GET /admin/system/status` 记录 version  
- [ ] 探测：显示最新 Release 且 sha 列表非空  
- [ ] 一键升级成功；health 绿  
- [ ] **只**该站 version 变；另两站 version **不变**  
- [ ] `/public/bootstrap` 主题与站点身份仍正确  
- [ ] 回滚 previous 成功（演练一次）

### 6.5 禁止操作

- [ ] 禁止在 A 站后台升级时手动改 B 站 `current`  
- [ ] 禁止用「产品站 DB」验证个人站  
- [ ] 禁止把 `RELEASE_ROOT` 指到共享目录「图省事」

---

## 7. 运行时架构（实现要点）

### 7.1 为何需要 helper

正在跑的 `inkless-api` **不能可靠地覆盖自己的 inode 后继续跑**。MVP 采用：

```text
Admin API (job 状态机)
    → 下载 + 校验（API 进程内可做）
    → exec / 调用 activate 脚本
    → systemctl restart 本 unit
    → 新进程起来后 job 状态写在 disk（SQLite 旁路文件或 data/updates/jobs.json）
```

Job 状态必须落盘（` $RELEASE_ROOT/var/updates/jobs/{id}.json` 或 data 下），因 restart 会杀掉旧进程。

### 7.2 API 草案

均在 `/admin/system/…`，鉴权 `system:manage`。

```http
GET  /admin/system/update          # 当前版本 + 缓存的 latest + capable 标志
POST /admin/system/update/check    # 强制刷新探测（绕过 TTL）
POST /admin/system/update/apply    # { "version": "v0.2.0" }  省略则 latest on channel
GET  /admin/system/update/jobs/:id
POST /admin/system/update/rollback # { "to": "previous" | "v0.1.9" } 仅本地 versions
```

`GET /admin/system/update` 响应示例：

```json
{
  "enabled": true,
  "capable": true,
  "currentVersion": "v0.1.8",
  "channel": "stable",
  "latest": {
    "version": "v0.2.0",
    "publishedAt": "…",
    "notesUrl": "…",
    "newer": true
  },
  "lastCheckAt": "…",
  "lastJob": { "id": "…", "status": "success", "toVersion": "v0.1.8" },
  "blockedReason": null
}
```

### 7.3 UI

- 入口：**系统状态** 页增加「关于与更新」卡片（或设置中心「更新」）  
- 展示：当前版本、最新版本、changelog 链接、capable 提示  
- 按钮：检查更新 / 升级到 vX / 回滚 previous  
- 进行中：job 进度（下载中 / 校验 / 激活 / 重启 / 健康检查）  
- **文案明确**：只升级本实例；主题更新请去主题市场  

### 7.4 与主题自动更新的关系

| | Host 自更新 | 主题自动更新 |
|--|-------------|--------------|
| 配置 | env + 可选 site config | `site_configs.system.themeAutoUpdate` |
| 默认 | **关** | **关** |
| 重启 | 要 | 不要 |
| 入口 | 系统状态 | 主题市场 |
| 探测源 | GitHub Release | themes catalog |

二者独立开关，互不阻塞。

---

## 8. MVP 分期

### Phase H0 — 可观测（0.5–1d）

- status 暴露 `updateCapable` / `releaseRoot` 是否配置  
- `GET …/update` + `POST …/check`（只读探测，不写盘）  
- UI：显示当前 vs latest，无升级按钮或按钮 disabled + 原因  

**验收**：三站都能看到各自 version 与「有新版」提示。

### Phase H1 — 一键应用（2–4d）

- 下载 + sha256 + 调 `qb-artifact-activate.sh`  
- job 落盘 + restart + health  
- 回滚 previous  
- 审计  
- **前提**：§6 隔离清单通过  

**验收**：产品站从 vN → vN+1 不经 npc；个人站版本不变。

### Phase H2 — 体验（可选）

- `inkless.run/releases/v1/channel-*.json` 镜像  
- 升级前触发 backup API  
- 维护模式  
- channel 策略：major 仅提示不 apply  

### Phase H3 — 明确不做直到需要

- 定时自动 apply host  
- 多实例编排仪表盘  
- 插件式「远程更新代理」

---

## 9. 失败与回滚

| 阶段失败 | 行为 |
|----------|------|
| 下载/校验失败 | 不碰 current；job=failed |
| activate 失败 | 不 restart；保留 current |
| restart 后 health 失败 | UI 提示「回滚 previous」；可自动 rollback（配置项，默认 **手动**） |
| 回滚失败 | 引导 npc / 主机手操 |

回滚命令语义对齐 `scripts/qb-artifact-rollback.sh`（若存在）或 activate 的 previous symlink。

---

## 10. 配置总表

| 变量 | 默认 | 说明 |
|------|------|------|
| `INKLESS_SELF_UPDATE_ENABLED` | `false` | 总开关 |
| `INKLESS_RELEASE_ROOT` | 空 | 空则 incapable |
| `INKLESS_SYSTEMD_UNIT` | 空 | 空则不可 restart（可仍允许只换 frontend，不推荐） |
| `INKLESS_UPDATE_REPO` | `yixian-huang/inkless` | |
| `INKLESS_UPDATE_CHANNEL` | `stable` | |
| `INKLESS_UPDATE_MANIFEST_URL` | 空 | 可选镜像 |
| `INKLESS_UPDATE_CHECK_TTL` | `15m` | |
| `INKLESS_UPDATE_ALLOW_HOSTS` | GH 默认列表 | 逗号分隔 |
| `INKLESS_GITHUB_TOKEN` | 空 | 可选 |
| `INKLESS_UPDATE_HEALTH_URL` | `http://127.0.0.1:$PORT/public/bootstrap` | 本地 health |
| `INKLESS_UPDATE_HEALTH_TIMEOUT` | `60s` | |

---

## 11. 测试计划

| 层 | 用例 |
|----|------|
| Unit | semver 比较；allowlist；sha256 匹配/失败；job 状态机 |
| Integration | mock HTTP Release + 临时 RELEASE_ROOT 解压 symlink（不真 systemctl） |
| Manual | 预发实例 H0→H1；双实例并行确认互不影响 |
| Chaos | 校验失败、磁盘满、health timeout、权限不足 |

---

## 12. 文档与发布

实现后更新：

1. [`OPS.md`](../OPS.md) — 自更新段落 + 与 npc 分工  
2. [`docs/internal/ops-lessons-site-isolation.md`](internal/ops-lessons-site-isolation.md) — 链到本设计 §6  
3. [`docs/deployment.md`](deployment.md) — env 表  
4. 用户文档（docs-site）：「检查更新 / 一键升级」  
5. Release notes：要求 artifact 命名与 sha256 齐全，否则站内升级不可用  

---

## 13. 决策摘要（锁定给实现）

| 项 | 选择 |
|----|------|
| 更新对象 | Host binary + SPA，非主题 |
| 探测源 | GitHub Releases（可选 channel manifest 镜像） |
| 产物 | 现有 `frontend/backend-*.tar.gz` + `.sha256` |
| 激活 | 复用 `qb-artifact-activate` 布局与脚本 |
| 安全 | HTTPS + allowlist + 强制 SHA256 + env 固定 RELEASE_ROOT |
| 默认 | **关闭**；无自动 apply |
| 作用域 | **单实例 / 单 unit / 单 RELEASE_ROOT** |
| npc | 首装、基建、隔离拆分、困难回滚；日常 patch 可站内 |
| 前置 | 三站 **禁止** 共享 `current` 再开 H1 |
| UI | 系统状态「关于与更新」，非主题市场 |

---

## 14. 建议实现顺序（给你三站）

1. **盘点**：三站 unit / port / `RELEASE_ROOT` / 是否共享 current（§6 清单）。  
2. **若共享**：先 npc/手操拆树（不写自更新代码也值）。  
3. **打/确认** Release 带齐 4 文件（两 tar + 两 sha256）。  
4. **实现 H0** 上线，三站只读看版本。  
5. **实现 H1**，先 **inkless-ops（产品）** 金丝雀，再 yx.ink / imgli。  

主题市场验收 **不依赖** 本设计；本设计解决的是 **host 多站跟版** 的长期冗余。

### 14.1 现网执行盘点（gomami · 2026-07-29）

完整勾选表与实测 symlink：[`docs/internal/ops-isolation-inventory-gomami-2026-07-29.md`](internal/ops-isolation-inventory-gomami-2026-07-29.md)。

摘要：

| 站 | unit | port | data/JWT | `backend/current` realpath |
|----|------|------|----------|----------------------------|
| yx.ink | `inkless` | 8088 | 独立 | `/opt/inkless/backend/versions/main-…`（权威树） |
| inkless.run | `inkless-ops` | 8089 | 独立 | **→ 同上（共享）** |
| imgli.com | `inkless-imgli` | 8090 | 独立 | **→ 同上（共享）** |

**H1 阻断：** ops / imgli 的 current 是指向 personal 树的 symlink。须先拆树再开站内 Host 升级。
