# 现网隔离盘点（执行版）— gomami · 2026-07-29

**Host:** `gomami` (`103.73.220.161`)  
**方法:** `npc server exec command gomami`（只读巡检；JWT 仅比对 fingerprint，不落明文）  
**对照设计:** [`docs/design-host-self-update-mvp.md`](../design-host-self-update-mvp.md) §6  
**对照教训:** [`ops-lessons-site-isolation.md`](ops-lessons-site-isolation.md)

---

## 1. 总判

| 维度 | 结论 | 对 Host 自更新 |
|------|------|----------------|
| **数据 / 身份隔离** | ✅ 通过 | 可安全并行跑三站 |
| **代码 `current` 隔离** | ❌ **未通过**（ops / imgli 共享 personal 的 artifact） | **禁止**开 H1 一键升级 |
| **反代域名→端口** | ✅ 正确 | 与 unit 一致 |
| **H0 只读探测** | ⚠️ 可做 | 三站会显示**同一** host 版本（因共享码） |

**一句话：** 三站是「三份数据 + 一份代码」；npc 部署 `hk-artifact` 切 `/opt/inkless/.../current` 时，**三站 host 一起变**。

---

## 2. 站点 × unit 矩阵

| 公网域名 | systemd unit | 状态 | 监听 | RELEASE 树 | 身份（bootstrap） | 主题 |
|----------|--------------|------|------|------------|-------------------|------|
| **yx.ink** | `inkless.service` | active | `*:8088` | `/opt/inkless` | Yixian / yx.ink | `blog-first` (built-in) |
| **inkless.run** | `inkless-ops.service` | active | `*:8089` | `/opt/inkless-ops` | Inkless / inkless.run | `product-first` (built-in) |
| **imgli.com** | `inkless-imgli.service` | active | `*:8090` | `/opt/inkless-imgli` | imgli | `product-first` (built-in) |
| themes.inkless.run | docker theme-demo | 8098 | `/opt/inkless-theme-demo` | （演示栈，非三站之一） |
| （遗留） | `impress.service` | **disabled** | — | `/opt/impress` | 旧 impress 树，未作生产 upstream |

### Caddy（`/etc/caddy/Caddyfile`）

| 域名 | reverse_proxy |
|------|----------------|
| `yx.ink` | `127.0.0.1:8088` |
| `inkless.run`, `www.inkless.run` | `127.0.0.1:8089` |
| `imgli.com`, `www.imgli.com` | `127.0.0.1:8090` |
| `themes.inkless.run` | （theme-demo，8098 类） |

Health：`8088` / `8089` / `8090` 均 `200`。

---

## 3. §6.1 拓扑确认（逐站）

### yx.ink → `inkless`

| 项 | 值 | 勾选 |
|----|-----|------|
| unit | `inkless.service` | ✅ |
| port | `8088` | ✅ |
| EnvironmentFile | `/opt/inkless/backend/.env` | ✅ |
| WorkingDirectory / ExecStart | `/opt/inkless/backend/current` · `inkless-api-latest` | ✅ |
| DB_DSN | `file:/opt/inkless/data/inkless.db` | ✅ |
| BASE_URL | `https://yx.ink` | ✅ |
| CORS | `https://yx.ink` | ✅ |
| FRONTEND_DIR | `/opt/inkless/frontend/current` | ✅ |
| UPLOAD_DIR | `/opt/inkless/uploads` | ✅ |
| ReadWritePaths | data + uploads | ✅ |
| JWT | 独立 fp（与另两站不同） | ✅ |
| 反代 | yx.ink → 8088 | ✅ |

### inkless.run → `inkless-ops`

| 项 | 值 | 勾选 |
|----|-----|------|
| unit | `inkless-ops.service` | ✅ |
| port | `8089` | ✅ |
| EnvironmentFile | `/opt/inkless-ops/backend/.env` | ✅ |
| DB_DSN | `file:/opt/inkless-ops/data/inkless.db` | ✅ |
| BASE_URL | `https://inkless.run` | ✅ |
| FRONTEND_DIR | `/opt/inkless-ops/frontend/current`（路径名独立） | ⚠️ 见 §4 |
| UPLOAD_DIR | `/opt/inkless-ops/uploads` | ✅ |
| JWT | 独立 | ✅ |
| `INKLESS_THEME_CATALOG_URL` | `https://inkless.run/marketplace/v1/themes.json` | ✅ 产品站已配 |
| 反代 | inkless.run → 8089 | ✅ |

### imgli.com → `inkless-imgli`

| 项 | 值 | 勾选 |
|----|-----|------|
| unit | `inkless-imgli.service` | ✅ |
| port | `8090` | ✅ |
| EnvironmentFile | `/opt/inkless-imgli/backend/.env` | ✅ |
| DB_DSN | `file:/opt/inkless-imgli/data/inkless.db` | ✅ |
| BASE_URL | `https://imgli.com` | ✅ |
| CORS | `imgli.com` + `www.imgli.com` | ✅ |
| FRONTEND_DIR | `/opt/inkless-imgli/frontend/current` | ⚠️ 见 §4 |
| UPLOAD_DIR | `/opt/inkless-imgli/uploads` | ✅ |
| JWT | 独立 | ✅ |
| 反代 | imgli.com → 8090 | ✅ |
| 主题 catalog URL | **未配置**（走内嵌 fallback） | ℹ️ 可选 |

---

## 4. §6.2 共享代码（阻断 Host 自更新 H1）

### 实测 symlink

| 路径 | 指向 | 最终 realpath |
|------|------|----------------|
| `/opt/inkless/backend/current` | `…/versions/main-5566dbb9` | **本树版本目录** |
| `/opt/inkless/frontend/current` | `…/versions/main-5566dbb9` | **本树版本目录** |
| `/opt/inkless-ops/backend/current` | **`/opt/inkless/backend/current`** | 与 personal **同一 inode 目标** |
| `/opt/inkless-ops/frontend/current` | **`/opt/inkless/frontend/current`** | 同上 |
| `/opt/inkless-imgli/backend/current` | **`/opt/inkless/backend/current`** | 同上 |
| `/opt/inkless-imgli/frontend/current` | **`/opt/inkless/frontend/current`** | 同上 |

**SHARED：**

```text
/opt/inkless/backend/versions/main-5566dbb9
  <= inkless + inkless-ops + inkless-imgli  (backend/current)

/opt/inkless/frontend/versions/main-5566dbb9
  <= inkless + inkless-ops + inkless-imgli  (frontend/current)
```

| 树 | 自有 `versions/` | 自有 `previous` |
|----|------------------|-----------------|
| `/opt/inkless` | ✅（多版本） | ✅ |
| `/opt/inkless-ops` | ❌（仅 symlink + `.env`） | ❌ |
| `/opt/inkless-imgli` | ❌（仅 symlink + `.env`） | ❌ |
| `/opt/impress` | ✅ 独立遗留 | ✅（disabled unit） |

### 影响

1. **`npc deploy impress hk-artifact`**（`releaseRoot=/opt/inkless`）会同时刷新三站运行中的 binary/SPA。  
2. 任一站若做 Host 自更新切 `current`，另两站一起变。  
3. 无法对 inkless.run 做「只升产品、个人留旧」金丝雀。  
4. `ProtectSystem=strict` + 各 unit `ReadWritePaths` **不含** 对方的 versions——即使开启自更新，ops/imgli 进程也**写不进** personal 的 versions（权限上更别扭）。

**结论：** 数据隔离 ✅；**代码隔离 ❌ → Host 自更新 H1 前置未满足。**

---

## 5. §6.3 自更新 env（现状）

三站均 **未** 配置：

- `INKLESS_SELF_UPDATE_ENABLED`
- `INKLESS_RELEASE_ROOT`
- `INKLESS_SYSTEMD_UNIT`

符合「功能未上、默认关」。在拆树前 **不要** 打开。

---

## 6. 数据侧唯一性（摘要）

| 键 | 结果 |
|----|------|
| JWT_SECRET / JWT_REFRESH | ✅ 三站 fingerprint 各异 |
| DB_DSN 路径 | ✅ 三棵 data 树 |
| PORT | ✅ 8088 / 8089 / 8090 |
| BASE_URL | ✅ 三域名 |
| UPLOAD_DIR | ✅ 三分开 |

**没有**重复 DB / 重复 JWT 的跨站污染迹象（与 2026-07 隔离事故后的正确模型一致）。

---

## 7. 部署与版本

| 项 | 值 |
|----|-----|
| 共享代码当前版本目录 | `main-5566dbb9`（含主题自动更新提交） |
| previous（仅 personal 树） | `main-290ac14c` |
| npc 环境 | `impress` / `hk-artifact` → 典型激活 `/opt/inkless` |
| 遗留 | `impress.service` disabled；`/opt/impress` 旧版本仍在盘上 |

---

## 8. Checklist 勾选汇总

### 数据 / 反代 / 身份

- [x] 三域名各绑独立 port  
- [x] 三 unit 独立 EnvironmentFile  
- [x] 三 DB 文件路径互异  
- [x] 三 JWT 互异  
- [x] 三 BASE_URL / CORS 对齐域名  
- [x] bootstrap 身份互不串味  

### 代码 / 自更新前置

- [ ] ops `backend/current` 独立 realpath  
- [ ] ops `frontend/current` 独立 realpath  
- [ ] imgli 同上  
- [ ] 各树 `versions/` + `previous`  
- [ ] 各 unit `INKLESS_RELEASE_ROOT` = 本树  
- [ ] 自更新写权限仅限本树  

### 可选产品配置

- [x] ops：`INKLESS_THEME_CATALOG_URL`  
- [ ] imgli：按需配 catalog  
- [ ] yx.ink：按需配 catalog  

---

## 9. 建议动作（优先级）

### P0 — 拆代码共享（Host 自更新 / 金丝雀前提）

目标：ops、imgli **各有一份** `backend/versions` + `frontend/versions`，`current` 指本树，**不再** → `/opt/inkless/...`。

**执行 runbook（默认不自动改生产）：**  
[`runbook-split-shared-release-roots-gomami.md`](runbook-split-shared-release-roots-gomami.md)  
脚本（默认 `DRY_RUN=1`）：[`scripts/ops-split-shared-current.sh`](../../scripts/ops-split-shared-current.sh)

之后：

- `npc deploy` 需 **按站 releaseRoot** 或部署后复制 artifact 到各树（控制面策略要改）。  
- 或：继续「只部署 personal 树 + 显式同步脚本推到 ops/imgli」——至少比 symlink 可审计。

### P1 — 再开 Host 自更新

拆完后按设计 H0→H1；每 unit：

```bash
INKLESS_SELF_UPDATE_ENABLED=true
INKLESS_RELEASE_ROOT=/opt/inkless-ops   # 本站树
INKLESS_SYSTEMD_UNIT=inkless-ops
```

### P2 — 清理

- 确认 `impress.service` 保持 disabled；评估删除 `/opt/impress` 旧树。  
- theme-demo 与三站隔离已满足，保持即可。

### 不阻塞

- **主题市场验收（产品站）**：数据隔离已够；catalog URL 已在 ops。  
- **主题自动更新**：可按站开关，与代码共享无关（写的是各站 DB + externalUrl）。

---

## 10. 复跑命令（agent / 人）

```bash
npc server list
npc server handoff brief gomami --section agentSummary -o json

# 共享 current 一眼看
npc server exec command gomami --timeout 60 -- \
  'readlink -f /opt/inkless/backend/current /opt/inkless-ops/backend/current /opt/inkless-imgli/backend/current;
   readlink -f /opt/inkless/frontend/current /opt/inkless-ops/frontend/current /opt/inkless-imgli/frontend/current'

# 端口与 unit
npc server exec command gomami --timeout 30 -- \
  'systemctl is-active inkless inkless-ops inkless-imgli; ss -lntp | grep -E ":808[89]|:8090"'
```

---

## 11. 决策建议

| 问题 | 建议 |
|------|------|
| 现在能否开站内 Host 一键升级？ | **否**（代码共享） |
| 现在能否只读「检查更新」H0？ | 能，但三站版本号会始终相同 |
| 主题市场 / 主题自动更新？ | **可以**在产品站继续用 |
| 下一步工程 | **先拆 ops/imgli 的 current 共享**，再实现 Host 自更新 H1 |

盘点完成日：2026-07-29（UTC 巡检窗口）。变更部署后请重跑 §10 命令刷新本表。
