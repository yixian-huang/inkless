# 文章 AI 元数据补齐与 SEO 评估

> 状态：Phase 1 + 1.5 已实现（2026-07-22）  
> 范围：文章编辑器 AI 补齐（title / slug / SEO / meta 等）→ 可评估 → 单页 SEO 优化  
> 相关实现：`backend/internal/service/article_meta_ai.go`、`article_meta_quality.go`、`frontend/src/api/ai.ts`、`aiMetaQuality.ts`、`aiMetaTelemetry.ts`  
> 黄金样本：[`docs/article-ai-meta-golden-samples.md`](article-ai-meta-golden-samples.md)

---

## 1. 目标与原则

### 1.1 产品目标

让作者**只聚焦正文**，在需要时一键得到可审阅的：

- 中英文标题
- 英文 URL slug
- SEO 标题（中/英）
- Meta 描述（中/英）
- （可选）摘要 excerpt、标签建议

默认：**预览后应用**，不直接写库；与现有「中→英翻译」心智一致。

### 1.2 设计原则

| 原则 | 说明 |
|------|------|
| 内容优先 | 正文是源，元数据是派生 |
| 不静默覆盖 | 默认 `fill_empty`；已有字段跳过或预览勾选 |
| 一次请求多字段 | 统一 `article-meta`，避免前端串多个 AI 端点 |
| 可渐进 | 整包生成 + 字段级 ✨ |
| 双语一致 | 主语言正文驱动；副语言同次输出或后续翻译 |
| 可评估 | 生成后可自动质检；应用率/修改率可观测 |
| 降级可用 | AI 未配置时本地仍可 `slugifyTitle`；明确引导配置 |

### 1.3 与现有能力边界

| 已有 | 本功能 |
|------|--------|
| `POST /admin/ai/suggest-titles`、`summarize`、`suggest-tags` | 保留给通用工具；**文章主路径走 `article-meta`** |
| `POST /admin/translate` + 编辑器中→英 | 整文/标题翻译；不负责 SEO 结构字段 |
| AI 配置页（OpenAI 兼容 / Anthropic） | 共用 provider；本功能不新增供应商 |
| 发布检查清单 | 缺 SEO 时可 CTA「用 AI 补齐」 |

**权限：** 生成接口使用 `articles:update`（或 create），**不要**绑 `settings:manage`。配置读写仍 `settings:manage`。

---

## 2. 交互摘要（Phase 1）

### 2.1 主入口

Action Bar 标题旁：**「AI 元数据」**（非 Zen 模式）。

打开 **预览面板**（非立刻写表单）：

- 源语言：zh / en
- 模式：● 仅填空（默认） / ○ 全部重写（二次确认）
- 勾选字段：中英标题、slug、SEO 标题、Meta、（可选）标签
- 标题多候选：一次返回 3 个，面板内「换一个」本地轮换
- 字符计量：复用 `SEO_TITLE_MAX` / `SEO_DESC_MIN|MAX` 与 `CharCountMeter`
- 操作：取消 / 应用勾选项 → `touch()` + toast「已应用 N 项，未保存」

### 2.2 触发门槛

- 正文 plain text 建议 ≥ ~100 字，否则 toast 拦截
- 已发布文章：默认 **锁定 slug**（可不勾选 / UI 禁用 + 说明）

### 2.3 发布清单联动

缺标题 / meta 警告时提供 **「用 AI 补齐」** → 同一预览面板，预勾缺项。

### 2.4 明确不做（Phase 1）

- 停笔自动生成（费 token、易打断）
- 生成后自动保存
- 保证排名 / 流量类文案
- 自动改已发布 slug

---

## 3. API 与实现骨架

### 3.1 新接口

```http
POST /admin/ai/article-meta
Permission: articles:update
```

**Request（示意）：**

```json
{
  "sourceLang": "zh",
  "zhTitle": "",
  "enTitle": "",
  "zhBody": "…",
  "enBody": "",
  "existing": {
    "slug": "",
    "zhSeoTitle": "",
    "enSeoTitle": "",
    "zhMetaDescription": "",
    "enMetaDescription": ""
  },
  "fields": ["titles", "slug", "seo", "meta", "tags", "excerpts"],
  "mode": "fill_empty",
  "titleCount": 3,
  "existingTags": []
}
```

**Response（示意）：**

```json
{
  "candidates": {
    "zhTitles": ["…", "…", "…"],
    "enTitles": ["…", "…", "…"]
  },
  "suggested": {
    "zhTitle": "…",
    "enTitle": "…",
    "slug": "…",
    "zhSeoTitle": "…",
    "enSeoTitle": "…",
    "zhMetaDescription": "…",
    "enMetaDescription": "…",
    "zhExcerpt": "…",
    "enExcerpt": "…",
    "tags": []
  },
  "skipped": ["slug"],
  "model": "…",
  "usage": { "prompt_tokens": 0, "output_tokens": 0 }
}
```

### 3.2 服务端职责

1. HTML → plain text；正文截断（建议 4k–8k 字，优先开头与小标题）
2. **单次 Chat** + 严格 JSON；temperature 约 0.3–0.5
3. 后处理：slug 规范化（对齐前端 `titleToLatinSlug` 规则）、SEO 长度裁剪
4. `fill_empty`：existing 非空字段进入 `skipped`，不写入 `suggested` 对应项
5. 业务 prompt 放在 **service 层**（`article_meta_ai.go`），通过 `AIProvider.Chat` / `ChatComplete`，避免每个 provider 复制业务逻辑

### 3.3 前端落点

```
frontend/src/api/ai.ts
frontend/src/pages/admin/articles/editor/
  hooks/useArticleAIMeta.ts
  components/AIMetaPreviewDialog.tsx
  utils/applyAIMeta.ts
```

接入：`EditorActionBar`、（Phase 2 前可延后）字段级 ✨、`PublishChecklistDialog`、`page.tsx` form setters。

### 3.4 后端落点

```
backend/internal/service/article_meta_ai.go
backend/internal/service/article_meta_ai_test.go
backend/internal/handler/ai/handler.go   # ArticleMeta
backend/internal/app/routes_admin.go    # 权限 articles:update
```

---

## 4. SEO「准确性」如何评估

AI 元数据**没有唯一标准答案**。评估拆成三层：**规范（硬）→ 相关性（半自动）→ 效果（线上校准）**。

### 4.1 层 A — 硬约束（可自动判）

| 维度 | 合格标准 |
|------|----------|
| 长度 | SEO 标题约 ≤60（中文按字）；Meta 约 50–160 |
| 非空 / 非占位 | 无「未命名」「Untitled」、无空洞模板句堆砌 |
| 格式 | slug 仅 `[a-z0-9-]`、无连续 `--`、长度合理（≤80） |
| 语言 | 中文字段以中文为主；英文字段以拉丁字母为主 |
| 禁幻觉事实 | 标题/Meta **不得出现正文未出现的专有数字、产品名、承诺话术** |

**使用时机：** 生成后、应用前；不通过则标红/标黄，仍可由作者强制应用（产品可选）。

### 4.2 层 B — 相关性（半自动，最接近「准不准」）

| 方法 | 说明 | 阶段 |
|------|------|------|
| 关键词覆盖 | 正文抽 5–15 词，title/meta 是否覆盖主词；0 重叠 → 低相关 | Phase 1.5 |
| 嵌入相似度 | 正文与字段 `Embed` 后余弦相似度；与关键词组合（见下） | Phase 1.5（已实现） |
| 盲测还原 | 只给 Meta，人能否说清主题 | 离线黄金样本 |

**日常「可发布」定义：** A 全过 + B 无严重告警 + 作者约 10 秒扫一眼。

### 4.3 层 C — 效果（线上，不作日更 KPI）

| 信号 | 说明 |
|------|------|
| Search Console CTR | 改 title/meta 后同 query 点击率（滞后 2–8 周） |
| 展示量 / 平均排名 | 噪声大，季度看趋势 |
| 站内列表 CTR / 跳出 | 辅助判断标题吸引力（偏产品文案） |

层 C 只用于**季度复盘模型与 prompt**，不卡发布。

### 4.4 离线黄金样本回归

1. 固定 **20–30 篇** 真实文章（长短、中英、体裁多样）
2. 每篇人工标「可接受 / 不可接受」，或准备参考 title/meta
3. 改 prompt / 换模型时同一套重跑

**人工打分表（0–2）：**

| 项 | 含义 |
|----|------|
| 主题一致 | 是否跑题、夸大 |
| 信息密度 | 是否太空或堆砌 |
| 双语质量 | 英文是否自然 |
| 规范 | 长度、slug、无违规话术 |
| 可发布 | 作者是否愿意一键用 |

**通过线：** 平均 ≥ 1.5 且「主题一致」无 0 分。记录：模型名、温度、正文截断长度、通过率、JSON 解析成功率、P95 延迟、单次成本。

### 4.5 在线产品反馈（Phase 1 预留钩子）

应用勾选项后（或面板内）：

- 一键：`有用` / `需大改` / `完全不可用`（可选一句备注）
- 指标：
  - **应用率** = 打开预览后点应用的比例
  - **字段修改率** = 应用后保存前用户改写的字段占比 / 编辑距离

修改率持续偏高 → 优先调 prompt 或换模型，而不是加更多自动覆盖。

### 4.6 与发布门禁的关系

| 级别 | 示例 |
|------|------|
| Block（发布） | 缺中文标题、正文为空等现有规则；AI 不单独引入新 block，除非空字段仍未补 |
| Warn | Meta 过短/过长、相关度低、与站内 title 过度雷同 |
| AI 质检 | 应用前软提示；不替代发布清单 |

---

## 5. 日常模型选型

### 5.1 任务画像

短结构化输出 + 中等指令遵循；**不需要**强推理、超长上下文、联网工具。

| 能力 | 重要度 |
|------|--------|
| 稳定 JSON / 字段约束 | 极高 |
| 中英双语 | 高 |
| 低延迟（目标 2–8s） | 高 |
| 低单价 | 高 |
| 忠实给定正文 | 高 |
| 长上下文 / 强推理 | 低 |

### 5.2 推荐档位

| 档位 | 典型 | 用途 |
|------|------|------|
| **默认生产** | `gpt-4o-mini`、Claude Haiku 级、同级 OpenAI 兼容轻量模型 | 日更、批量补齐 |
| **质量敏感** | `gpt-4o` / Sonnet 级 | 黄金样本评测、旗舰文 |
| **不推荐日常** | o1/o3 等强推理旗舰 | 贵、慢，对 meta 收益小 |

**参数建议：** temperature 0.3–0.5；max tokens 小；正文截断 4k–8k 字。

**选型流程：** 黄金样本 × 2–3 候选模型，比 JSON 成功率、可发布率、费用、P95；可发布率接近时选更便宜的。

配置入口：现有 **管理端 → AI 配置**；生产默认轻量模型即可，无需为本功能单独接供应商。

---

## 6. 分阶段交付

```
Phase 1     AI 补齐元数据 + 预览应用 + 反馈钩子
    ↓
Phase 1.5   生成后自动质检 + 黄金样本回归流程
    ↓
Phase 2     单页 SEO 体检 / 优化（诊断 → 建议 → 逐项采纳）
```

### Phase 1 — MVP（补齐）

**交付：**

- [x] `POST /admin/ai/article-meta`（titles + slug + seo + meta，双语；可选 tags/excerpts）
- [x] 权限 `articles:update`
- [x] 编辑器主按钮 + `AIMetaPreviewDialog` + `fill_empty` / rewrite
- [x] 发布清单「用 AI 补齐」
- [x] `api/ai.ts` + `useArticleAIMeta` + apply 规则单测
- [x] 应用后轻量反馈钩子（localStorage 占位）
- [x] AI 设置页文案：说明文章元数据共用此配置

**验收：**

- 正文足够长时可生成预览；仅填空不覆盖已有 slug/标题
- AI 未配置返回明确错误并引导 `/admin/ai-settings`
- 应用只改表单不自动保存；`pnpm lint && pnpm type-check` + 相关 Go 测试通过

### Phase 1.5 — 可评估（质检）

**交付：**

- [x] 应用前自动：长度、语言、占位检测（服务端 `warnings` + 前端按勾选字段重算）
- [x] 关键词重叠 + **embedding 余弦相关度**（`low_relevance` / `low_relevance_embedding` / `low_relevance_weak`；Embed 失败降级）
- [x] 黄金样本清单与跑分：[`article-ai-meta-golden-samples.md`](article-ai-meta-golden-samples.md) + `testdata/article_meta_golden/`
- [x] 本地应用率/反馈统计：`localStorage` events + 控制台 `__inklessAIMetaStats()`

**验收：**

- 明显跑题或超长结果在 UI 有可见警告（字段高亮 +「仍要应用」）
- 换模型前后可用同一黄金样本对比通过率（见黄金样本文档）

### Phase 2 — 单页 SEO 优化（下一阶段产品）

与 Phase 1 **本质不同**：已有内容要「更好」，不是「从空到有」。

| | Phase 1 补齐 | Phase 2 优化 |
|--|--------------|--------------|
| 问题 | 字段空 | 已有字段/结构想更好搜到 |
| 输出 | 可应用字段包 | **诊断 + 建议 + 可选改写** |
| 风险 | 低 | 改已发布 title/URL、过度优化 |
| 评价 | 相关性 + 规范 | 相对现版是否更好 + 长期 CTR |

**建议能力：**

1. **规则体检**：标题长度、H 结构、meta 完整度、图片 alt、正文长度等  
2. **AI 建议清单（逐条采纳）**：更优 SEO 标题/Meta（diff）、H2 大纲建议、主词/长尾（基于正文）、站内内链候选  
3. **SERP / 社交卡片预览**：生成前后对比  
4. **不做**：自动改已发布 slug；保证排名话术；无 GSC 时的排名预测  

**验收（阶段目标）：**

- 单篇文章可打开 SEO 面板看到分数/清单
- 建议可逐项应用进表单；已发布 slug 默认锁定
- 与 Phase 1 共用 AI 配置与质检规则

**启动条件建议：** Phase 1 上线后有一定真实调用与修改率数据，再定 Phase 2 优先级。

---

## 7. 风险与约束

| 风险 | 缓解 |
|------|------|
| 模型幻觉（正文没有的承诺） | 硬约束 + 相关度；prompt 要求「仅基于正文」 |
| Token 成本 | 截断正文、轻量默认模型、用户主动触发 |
| 权限过宽/过窄 | 生成跟文章权限；配置跟 settings |
| 已发布 URL 变更 | UI 锁 slug；rewrite 二次确认 |
| JSON 解析失败 | 重试一次 / 抽 fenced block；失败友好错误 |
| 与翻译重复改写 | 元数据生成与正文翻译入口分离；文案区分 |

---

## 8. 文档与代码索引

| 主题 | 位置 |
|------|------|
| AI Provider 接口 | `backend/internal/provider/ai.go` |
| 现有 AI HTTP | `backend/internal/handler/ai/handler.go`、`routes_admin.go` |
| Slug 本地规则 | `frontend/.../editor/utils/slugify.ts` |
| SEO 长度常量 / 发布清单 | `frontend/.../editor/utils/publishChecklist.ts` |
| 文章字段模型 | `backend/internal/model/article.go` |
| 产品总览 | `docs/product-roadmap.md`（AI 相关条目） |
| 历史 AI 大阶段计划 | `docs/superpowers/plans/2026-03-11-phase5-ai-intelligence.md`（归档，实现以本文与代码为准） |

---

## 9. 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-22 | 初版：交互、API、评估三层、模型选型、Phase 1 / 1.5 / 2 |
| 2026-07-22 | Phase 1 实现：`POST /admin/ai/article-meta`、编辑器「AI 元数据」预览面板、发布清单入口、localStorage 反馈钩子 |
| 2026-07-22 | Phase 1.5：质量 warnings、关键词相关度、黄金样本文档/fixture、本地 events 统计 |
| 2026-07-22 | Embedding 相关度：`EvaluateArticleMetaEmbeddingRelevance` + OpenAI `EmbedBatch`；Anthropic 降级 |
