# Runbook：拆分共享 `backend/current` / `frontend/current`（gomami）

| 字段 | 值 |
|------|-----|
| 状态 | **Ready — 待人类批准后执行** |
| 日期 | 2026-07-29 |
| 主机 | `gomami`（`npc server` 别名） |
| 盘点依据 | [`ops-isolation-inventory-gomami-2026-07-29.md`](ops-isolation-inventory-gomami-2026-07-29.md) |
| 设计依据 | [`design-host-self-update-mvp.md`](../design-host-self-update-mvp.md) §6 / §14.1 |
| 默认策略 | **本文不自动执行生产变更**；须用户明确批准后，再经 `npc server exec` 逐步做 |

---

## 0. 目标与非目标

### 目标

把现在「三站共享 `/opt/inkless` 的 artifact」改成：

| 站 | unit | 端口 | 代码树（拆后） |
|----|------|------|----------------|
| yx.ink | `inkless` | 8088 | `/opt/inkless`（保持权威 personal 树，**本窗口可不改**） |
| inkless.run | `inkless-ops` | 8089 | `/opt/inkless-ops` **自有** `backend/versions` + `frontend/versions` |
| imgli.com | `inkless-imgli` | 8090 | `/opt/inkless-imgli` **自有** 同上 |

拆后：

```text
readlink -f /opt/inkless-ops/backend/current
# 期望: /opt/inkless-ops/backend/versions/<VER>
# 禁止: /opt/inkless/backend/versions/...

readlink -f /opt/inkless-imgli/backend/current
# 期望: /opt/inkless-imgli/backend/versions/<VER>
```

数据树（`data/`、`uploads/`、`.env`、JWT）**已独立，本 runbook 不改**。

### 非目标

- 不改 Caddy 域名与端口映射  
- 不改 DB / JWT / BASE_URL  
- 不实现 Host 自更新代码  
- 不强制三站版本永久一致（拆完后**允许**分叉）  
- 不删除 `/opt/inkless` 历史 `versions/`  
- 不动 `impress.service` / theme-demo（除非验证时顺带确认）

### 为何要拆

| 现状 | 风险 |
|------|------|
| ops/imgli 的 `current` → `/opt/inkless/.../current` | 一次 `npc deploy hk-artifact` 三站同升 |
| 无法产品站金丝雀 | 无法只升 inkless.run |
| Host 自更新 H1 | 设计明确禁止共享 `current` |

---

## 1. 角色与批准

| 角色 | 职责 |
|------|------|
| 操作者 | 按清单执行 `npc` 命令，贴出验证输出 |
| 批准人 | 确认窗口、顺序、失败是否回滚 |
| 禁止 | 未批准时改 symlink / restart 生产 unit |

**批准话术示例（给 agent）：**

> 批准执行 gomami 拆树 runbook：先 dry-run，再只做 inkless-ops；验证通过后再做 inkless-imgli。

未写清范围则 **只允许 dry-run / 只读**。

---

## 2. 变更窗口建议

| 项 | 建议 |
|----|------|
| 时长 | 每站 10–20 分钟（含验证） |
| 顺序 | **先 `inkless-ops`（产品）→ 再 `inkless-imgli`**；personal 不拆 |
| 影响 | 目标 unit `restart` 时该站约数秒不可用；**另两站应保持服务** |
| 低峰 | 按业务自选；产品站可优先工作日低流量 |

---

## 3. 前置条件（执行前全部勾选）

在 **本机**（有 npc 的环境）：

```bash
npc server list | grep -i gomami
npc server handoff brief gomami --section agentSummary -o json | head -c 500
```

在 **gomami**（只读预检）——一次跑完，保存输出：

```bash
npc server exec command gomami --timeout 60 -- 'bash -lc "
set -e
echo === units ===
systemctl is-active inkless inkless-ops inkless-imgli
echo === shared? ===
readlink -f /opt/inkless/backend/current
readlink -f /opt/inkless-ops/backend/current
readlink -f /opt/inkless-imgli/backend/current
readlink -f /opt/inkless/frontend/current
readlink -f /opt/inkless-ops/frontend/current
readlink -f /opt/inkless-imgli/frontend/current
echo === version id ===
basename \$(readlink -f /opt/inkless/backend/current)
echo === disk ===
df -h /opt | tail -1
du -sh /opt/inkless/backend/versions/\$(basename \$(readlink -f /opt/inkless/backend/current)) \
       /opt/inkless/frontend/versions/\$(basename \$(readlink -f /opt/inkless/frontend/current))
echo === health ===
for p in 8088 8089 8090; do curl -sS -m 3 -o /dev/null -w \"\$p=%{http_code} \" http://127.0.0.1:\$p/health; done; echo
"'
```

勾选：

- [ ] 三 unit `active`  
- [ ] ops/imgli backend realpath **仍指向** `/opt/inkless/...`（确认尚未拆过）  
- [ ] `/opt` 剩余空间 **> 200MB**（每站约 backend 46M + frontend 20M ≈ 70M，留余量）  
- [ ] 8088/8089/8090 health = 200  
- [ ] 已记下当前 `VER=$(basename $(readlink -f /opt/inkless/backend/current))`（例：`main-5566dbb9`）  
- [ ] 批准人已确认本次范围（仅 ops / ops+imgli）

**不要**在拆树窗口同时跑 `npc deploy`。

---

## 4. 策略说明：复制 vs 硬链

| 方式 | 做法 | 优点 | 缺点 |
|------|------|------|------|
| **A. rsync 复制（推荐）** | 把 `versions/$VER` 拷到目标树 | 真独立；删 personal 旧版不影响 ops | 多占 ~70MB/站 |
| B. hardlink `cp -al` | 同 inode | 省空间 | 同盘文件仍共享内容；覆盖写可能互相影响，**不推荐用于生产拆分目标** |

本 runbook 采用 **A（rsync -a）**。

版本目录名与 personal **保持同一 `VER` 字符串**（便于对照），但路径在目标树下。

---

## 5. 单站拆分程序（模板）

以下用变量描述，**ops** 与 **imgli** 各跑一遍。

```bash
# 在执行前于会话中设定（示例：ops）
SITE_ROOT=/opt/inkless-ops
UNIT=inkless-ops
PORT=8090   # 错！ops 是 8089 —— 见站点参数表
```

### 5.1 站点参数表

| 参数 | inkless-ops | inkless-imgli |
|------|-------------|---------------|
| `SITE_ROOT` | `/opt/inkless-ops` | `/opt/inkless-imgli` |
| `UNIT` | `inkless-ops` | `inkless-imgli` |
| `PORT` | `8089` | `8090` |
| `BASE_URL`（验证） | `https://inkless.run` | `https://imgli.com` |
| 期望 identity 关键词 | `Inkless` | `imgli` |
| 期望 themeId | `product-first` | `product-first` |

Personal **跳过**：

| | inkless (yx.ink) |
|--|------------------|
| `SITE_ROOT` | `/opt/inkless` — 已是权威树，无需复制 |

### 5.2 源版本

```bash
SRC_ROOT=/opt/inkless
VER=$(basename "$(readlink -f "${SRC_ROOT}/backend/current")")
# 必须与 frontend 一致：
test "$(basename "$(readlink -f "${SRC_ROOT}/frontend/current")")" = "$VER"
```

### 5.3 备份现有 symlink（可回滚）

```bash
TS=$(date -u +%Y%m%dT%H%M%SZ)
BACKUP_DIR="${SITE_ROOT}/.split-backup-${TS}"
mkdir -p "${BACKUP_DIR}"
cp -a "${SITE_ROOT}/backend/current" "${BACKUP_DIR}/backend.current.link" 2>/dev/null || true
cp -a "${SITE_ROOT}/frontend/current" "${BACKUP_DIR}/frontend.current.link" 2>/dev/null || true
readlink "${SITE_ROOT}/backend/current" > "${BACKUP_DIR}/backend.current.readlink.txt"
readlink "${SITE_ROOT}/frontend/current" > "${BACKUP_DIR}/frontend.current.readlink.txt"
echo "backup=${BACKUP_DIR} VER=${VER}"
```

> `cp -a` 对 symlink 复制的是 **链接本身**（不解析），正好用于回滚。

### 5.4 复制 versions（不停站）

可在 **不 restart** 时先 rsync，缩短切换窗口：

```bash
mkdir -p "${SITE_ROOT}/backend/versions" "${SITE_ROOT}/frontend/versions"

# 若目标已存在同名目录，先改名避开
if [ -d "${SITE_ROOT}/backend/versions/${VER}" ]; then
  mv "${SITE_ROOT}/backend/versions/${VER}" "${SITE_ROOT}/backend/versions/${VER}.pre-split-${TS}"
fi
if [ -d "${SITE_ROOT}/frontend/versions/${VER}" ]; then
  mv "${SITE_ROOT}/frontend/versions/${VER}" "${SITE_ROOT}/frontend/versions/${VER}.pre-split-${TS}"
fi

rsync -a --delete \
  "${SRC_ROOT}/backend/versions/${VER}/" \
  "${SITE_ROOT}/backend/versions/${VER}/"

rsync -a --delete \
  "${SRC_ROOT}/frontend/versions/${VER}/" \
  "${SITE_ROOT}/frontend/versions/${VER}/"

# 二进制可执行
chmod a+x "${SITE_ROOT}/backend/versions/${VER}/inkless-api-"* 2>/dev/null || true
# 确保 inkless-api-latest 指向真实二进制
if [ -L "${SITE_ROOT}/backend/versions/${VER}/inkless-api-latest" ] || [ -e "${SITE_ROOT}/backend/versions/${VER}/inkless-api-latest" ]; then
  ls -la "${SITE_ROOT}/backend/versions/${VER}/inkless-api-latest"
else
  BIN=$(find "${SITE_ROOT}/backend/versions/${VER}" -maxdepth 1 -type f -name 'inkless-api-*' ! -name 'inkless-api-latest' | head -1)
  ln -sfn "$(basename "$BIN")" "${SITE_ROOT}/backend/versions/${VER}/inkless-api-latest"
fi

# 目录属主与 personal 一致（root 拥有 versions 亦可，与现网一致）
chown -R root:root "${SITE_ROOT}/backend/versions/${VER}" "${SITE_ROOT}/frontend/versions/${VER}"
```

校验复制：

```bash
test -x "${SITE_ROOT}/backend/versions/${VER}/inkless-api-latest" \
  || test -e "${SITE_ROOT}/backend/versions/${VER}/inkless-api-latest"
test -f "${SITE_ROOT}/frontend/versions/${VER}/index.html"
# 内容校验（尺寸量级）
du -sh "${SITE_ROOT}/backend/versions/${VER}" "${SRC_ROOT}/backend/versions/${VER}"
```

### 5.5 原子切换 symlink + restart（短中断）

```bash
# 指向【本树】versions，禁止再指向 /opt/inkless
ln -sfn "${SITE_ROOT}/backend/versions/${VER}" "${SITE_ROOT}/backend/current"
ln -sfn "${SITE_ROOT}/frontend/versions/${VER}" "${SITE_ROOT}/frontend/current"

# 可选：记录 previous 为本站视角（若曾共享，previous 可先指同一 VER 的备份说明）
# 首次拆分可不设 previous；第二次升级时由 deploy 脚本维护

# 关键：realpath 必须在本 SITE_ROOT 下
readlink -f "${SITE_ROOT}/backend/current" | grep -q "^${SITE_ROOT}/backend/versions/" 
readlink -f "${SITE_ROOT}/frontend/current" | grep -q "^${SITE_ROOT}/frontend/versions/"

systemctl restart "${UNIT}"
sleep 2
systemctl is-active "${UNIT}"
```

### 5.6 验证（该站 + 另站未伤）

```bash
# 本站
curl -sS -m 5 -o /dev/null -w "health=%{http_code}\n" "http://127.0.0.1:${PORT}/health"
curl -sS -m 5 "http://127.0.0.1:${PORT}/public/bootstrap" | python3 -c "
import sys,json
d=json.load(sys.stdin)
g=(d.get('globalConfig') or {}).get('config') or {}
print('identity', (g.get('identity') or {}).get('name'))
print('theme', (d.get('activeTheme') or {}).get('themeId'))
print('footer', (g.get('footer') or {}).get('copyright'))
"

# 进程 cwd / 可执行文件应落在 SITE_ROOT
systemctl show "${UNIT}" -p MainPID --value | xargs -I{} sh -c 'readlink -f /proc/{}/exe; tr "\0" "\n" < /proc/{}/environ | grep -E "^(PORT|BASE_URL|FRONTEND_DIR|DB_DSN)="'

# 另两站仍 200（拆 ops 时检查 8088 与 8090；拆 imgli 时检查 8088 与 8089）
for p in 8088 8089 8090; do
  curl -sS -m 3 -o /dev/null -w "$p=%{http_code} " "http://127.0.0.1:$p/health" || echo "$p=fail"
done
echo
```

公网抽检（本机或 CI）：

```bash
curl -sS -m 15 -o /dev/null -w "%{http_code}\n" https://inkless.run/health
curl -sS -m 15 -o /dev/null -w "%{http_code}\n" https://imgli.com/health
curl -sS -m 15 -o /dev/null -w "%{http_code}\n" https://yx.ink/health
```

### 5.7 成功标准（单站）

- [ ] `readlink -f $SITE_ROOT/backend/current` **以** `$SITE_ROOT/backend/versions/` **开头**  
- [ ] frontend 同理  
- [ ] **不再** 等于 `/opt/inkless/backend/versions/...`  
- [ ] unit `active`，本站 health 200  
- [ ] bootstrap 身份 / theme 与拆前一致  
- [ ] 另两站 health 200  
- [ ] `FRONTEND_DIR` 仍指向 `$SITE_ROOT/frontend/current`（.env 已如此则无需改）

---

## 6. 推荐执行顺序（完整窗口）

### Phase 0 — Dry-run（无写）

只跑 §3 预检 + 打印将要执行的路径：

```bash
npc server exec command gomami --timeout 30 -- 'bash -lc "
SRC=/opt/inkless
VER=\$(basename \$(readlink -f \$SRC/backend/current))
for root in /opt/inkless-ops /opt/inkless-imgli; do
  echo \"WOULD rsync \$SRC/backend/versions/\$VER -> \$root/backend/versions/\$VER\"
  echo \"WOULD rsync \$SRC/frontend/versions/\$VER -> \$root/frontend/versions/\$VER\"
  echo \"WOULD ln -sfn \$root/backend/versions/\$VER \$root/backend/current\"
  echo \"current now: \$(readlink -f \$root/backend/current)\"
done
"'
```

### Phase 1 — 仅 inkless-ops

1. 设定 `SITE_ROOT=/opt/inkless-ops` `UNIT=inkless-ops` `PORT=8089`  
2. §5.3 → §5.4 → §5.5 → §5.6  
3. 失败 → §7 回滚 ops  
4. 成功 → 暂停 5–10 分钟观察，再 Phase 2  

### Phase 2 — inkless-imgli

1. `SITE_ROOT=/opt/inkless-imgli` `UNIT=inkless-imgli` `PORT=8090`  
2. 同样 §5  
3. 全站 §8 终态验收  

### Phase 3 — 记录与后续

- 更新盘点文档状态为「代码已隔离」  
- 约定新的 deploy 策略（§9）  
- **仍不要** 默认打开 `INKLESS_SELF_UPDATE_ENABLED`，直到 Host H1 实现并通过单站演练  

---

## 7. 回滚（单站）

若 restart 后 health 失败或身份错乱：

```bash
# 使用拆分前备份的 symlink 内容
BACKUP_DIR=...   # 来自 §5.3
SITE_ROOT=...
UNIT=...

# 恢复为指向 /opt/inkless 的旧链接（备份文件是 symlink）
ln -sfn "$(readlink "${BACKUP_DIR}/backend.current.link")" "${SITE_ROOT}/backend/current"
# 若 backup 存的是链接节点：
#   ln -sfn /opt/inkless/backend/current "${SITE_ROOT}/backend/current"
# 以 backup 内 readlink 文本为准：
ln -sfn "$(cat "${BACKUP_DIR}/backend.current.readlink.txt")" "${SITE_ROOT}/backend/current"
ln -sfn "$(cat "${BACKUP_DIR}/frontend.current.readlink.txt")" "${SITE_ROOT}/frontend/current"

systemctl restart "${UNIT}"
sleep 2
systemctl is-active "${UNIT}"
curl -sS -m 5 -o /dev/null -w "%{http_code}\n" "http://127.0.0.1:${PORT}/health"
```

回滚后三站再次共享代码（回到拆前模型），**可接受**为紧急态。

复制出的 `versions/$VER` 目录可先保留，便于再次切换；确认稳定后再删以省空间。

---

## 8. 终态验收清单

```bash
npc server exec command gomami --timeout 60 -- 'bash -lc "
echo === realpaths ===
for p in \
  /opt/inkless/backend/current \
  /opt/inkless-ops/backend/current \
  /opt/inkless-imgli/backend/current \
  /opt/inkless/frontend/current \
  /opt/inkless-ops/frontend/current \
  /opt/inkless-imgli/frontend/current
 do
  echo \"\$p -> \$(readlink -f \$p)\"
done
echo === uniqueness (python) ===
python3 - <<\"PY\"
import os
paths=[
 \"/opt/inkless/backend/current\",
 \"/opt/inkless-ops/backend/current\",
 \"/opt/inkless-imgli/backend/current\",
]
reals=[os.path.realpath(p) for p in paths]
print(reals)
assert len(set(reals))==3, \"FAIL still shared\"
for r in reals:
  assert r.startswith(r.split(\"/backend/\")[0]+\"/backend/versions/\") or \"/versions/\" in r
print(\"PASS three distinct backend realpaths\")
PY
echo === health ===
for p in 8088 8089 8090; do curl -sS -m 3 -o /dev/null -w \"\$p=%{http_code} \" http://127.0.0.1:\$p/health; done; echo
"'
```

- [ ] 三 backend realpath **互不相同**  
- [ ] 三 frontend realpath **互不相同**  
- [ ] ops/imgli realpath **不**落在 `/opt/inkless/` 下  
- [ ] 三站 health 200，bootstrap 身份正确  
- [ ] yx.ink 全程未 restart（或仅若误操作才需查）  

---

## 9. 拆后部署策略（必读）

拆完后 **不能** 再假设「只部署 `/opt/inkless` 就会更新产品站」。

### 短期（人工 / agent）

每次要升 host：

1. 继续 `npc deploy impress hk-artifact --ref …`（更新 personal 权威树），**或**  
2. 对需要升级的站，**额外** rsync 新 `versions/$NEW` 并切该站 `current` + `systemctl restart`  

推荐维护一个小脚本（可后补）：`scripts/ops-propagate-version-to-site.sh SITE_ROOT VER`。

### 中期（控制面）

| 方案 | 说明 |
|------|------|
| **多 env 多 releaseRoot** | npc 上为 ops/imgli 各建 artifact env，`releaseRoot=/opt/inkless-ops` 等 |
| **deploy post-hook** | 激活 personal 后 hook 同步到 ops/imgli（若希望继续锁同版本） |
| **站内 Host 自更新** | 每站自拉 Release，互不依赖（H1） |

在策略未改前，文档/agent 规则应写明：

> 部署 personal **不等于** 自动升级 inkless.run / imgli.com。

### ProtectSystem 注意

unit 使用 `ProtectSystem=strict`，`ReadWritePaths` 仅 data/uploads/…。  
**激活新版本** 需要 root（或具备写 `versions/` 的用户）在站外执行 rsync/ln，**不要**指望 inkless 进程用户写 versions——与 Host 自更新设计一致（helper + root/polkit）。

---

## 10. 一键脚本（默认 dry-run）

仓库脚本：[`scripts/ops-split-shared-current.sh`](../../scripts/ops-split-shared-current.sh)

```bash
# 在 gomami 上（经 npc 上传或 curl 仓库 raw / 粘贴）：
# 默认 DRY_RUN=1，只打印
SITE_ROOT=/opt/inkless-ops UNIT=inkless-ops PORT=8089 \
  bash scripts/ops-split-shared-current.sh

# 真正执行（需批准）
DRY_RUN=0 SITE_ROOT=/opt/inkless-ops UNIT=inkless-ops PORT=8089 \
  bash scripts/ops-split-shared-current.sh
```

脚本必须在 **目标机** 跑，且具备 root（写 `/opt`、systemctl）。

---

## 11. 风险与缓解

| 风险 | 缓解 |
|------|------|
| rsync 中途磁盘满 | §3 检查空间；rsync 先于切链 |
| 切链后 binary 不可执行 | chmod + 检查 `inkless-api-latest` |
| restart 起不来 | §7 回滚 symlink |
| 拷错版本 | `VER` 取自 live personal current；前后端 VER 一致断言 |
| 与 deploy 竞态 | 窗口内禁止 npc deploy |
| 误拆 personal | 脚本拒绝 `SITE_ROOT=/opt/inkless` |

---

## 12. 沟通模板

**开始：**

> 变更窗口：gomami 拆分 inkless-ops 代码树（数据不动）。预期 inkless.run 中断 &lt;30s。yx.ink / imgli 应不受影响。

**结束：**

> inkless-ops `current` 已指向 `/opt/inkless-ops/.../versions/<VER>`。三站 health 200。后续 host 部署需按站同步（见 runbook §9）。

---

## 13. 执行记录（填写）

| 时间 (UTC) | 阶段 | 操作者 | 结果 | 备注 |
|------------|------|--------|------|------|
| | dry-run | | | |
| | ops split | | | |
| | imgli split | | | |
| | final verify | | | |

---

## 14. 与后续 Host 自更新

| 步骤 | 依赖本 runbook |
|------|----------------|
| H0 探测 | 不强制，但拆后版本可分叉，探测才有意义 |
| H1 一键升级 | **必须** 本拆分完成 |
| 主题市场 | **不依赖** 本拆分 |

拆分完成并验收后，再实现 H0/H1。
