# 文章 AI 元数据 — 黄金样本与回归跑分

配合 [`article-ai-meta-seo.md`](article-ai-meta-seo.md) Phase 1.5。用于**换模型 / 改 prompt 前后**对比「可发布率」，而不是日常线上 KPI。

---

## 1. 样本清单（建议 20–30 篇）

从真实站点抽文，覆盖：

| 维度 | 覆盖 |
|------|------|
| 长度 | 短（~200 字）、中、长（>2000 字） |
| 语言 | 中文主、英文主、中英混排 |
| 体裁 | 技术、随笔、产品说明、changelog |
| 噪声 | 大量代码块、列表、引用 |

本地 fixture（可扩展）：`backend/internal/service/testdata/article_meta_golden/`  
每篇一个 JSON：

```json
{
  "id": "sample-rust-async",
  "sourceLang": "zh",
  "body": "……纯文本正文……",
  "notes": "期望主题：Rust 异步",
  "acceptHints": {
    "mustContainAny": ["Rust", "异步", "Tokio"],
    "forbid": ["天气预报", "股票"]
  }
}
```

---

## 2. 人工打分表（0–2）

| 项 | 0 | 1 | 2 |
|----|---|---|---|
| 主题一致 | 跑题/幻觉 | 部分贴题 | 准确 |
| 信息密度 | 空话堆砌 | 尚可 | 信息充分 |
| 双语质量 | 中英错位或机翻感强 | 可读 | 自然 |
| 规范 | 长度/slug/占位失败 | 轻微超标 | 符合约束 |
| 可发布 | 不愿用 | 大改后可用 | 愿意一键用 |

**通过线（单篇）：** 平均 ≥ 1.5 且「主题一致」≠ 0。  
**套件通过率：** 通过篇数 / 总篇数；建议 ≥ 80% 再切换默认生产模型。

---

## 3. 自动质检（与产品一致）

生成后响应含 `warnings[]`（服务端），前端再按**当前勾选字段 + 标题候选**重算。

| code | 含义 |
|------|------|
| `placeholder` | 未命名 / Untitled 等 |
| `length_short` / `length_long` | Meta/SEO/标题长度 |
| `language_mismatch` | 中英字段脚本不符 |
| `low_relevance` | 与正文关键词重叠为 0 |
| `slug_format` | slug 非 kebab-case |

**注意：** 自动质检 **不替代** 人工「主题一致」分；`low_relevance` 是启发式，短英文专有名词可能漏检或误报。

跑 Go 单测（规则本身，不调 LLM）：

```bash
cd backend && go test ./internal/service/ -run 'MetaQuality|ArticleMeta' -count=1
```

前端规则单测：

```bash
cd frontend && pnpm exec vitest run \
  src/pages/admin/articles/editor/utils/aiMetaQuality.test.ts \
  src/pages/admin/articles/editor/utils/aiMetaTelemetry.test.ts
```

---

## 4. 对真实模型的手工回归步骤

1. 在 AI 配置页选中候选模型 A。  
2. 对每篇样本：打开编辑器 → 贴正文 → **AI 元数据** → 记录 suggested + warnings。  
3. 按打分表打分；记下 JSON 解析失败次数与 P95 体感延迟。  
4. 换模型 B 重复。  
5. 比较：通过率、费用、延迟；**通过率接近时选更便宜的**。

可选：把每次生成的 `suggested` 与 `warnings` 存成 `reports/<model>/<id>.json` 便于 diff。

---

## 5. 在线统计（本机）

编辑器会把事件写入 `localStorage`：

- key：`inkless.aiMeta.events`（最近 200 条）
- 类型：`open` / `generate_ok` / `generate_err` / `apply` / `dismiss` / `feedback`

控制台：

```js
__inklessAIMetaStats()
// { applyRate, applyWithWarnRate, feedback, generateOk, … }
```

| 指标 | 计算 | 解读 |
|------|------|------|
| **应用率** | applies / generate_ok | 预览后是否真的用 |
| **带警告应用率** | apply 且 warnCount>0 / applies | 偏高说明仍需人工把关（正常） |
| **反馈** | useful / needs_edit / unusable | 改 prompt 的信号 |

**修改率（字段级）：** 当前未做服务端埋点；可用「应用后到保存前是否改字段」后续在 Phase 2 接审计。现阶段以 **应用率 + 主观反馈** 为主。

---

## 6. 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-22 | Phase 1.5：黄金样本流程 + 本地 events 统计说明 |
