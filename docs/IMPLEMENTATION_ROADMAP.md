# FlowAgent 后续实现与调试路线图

本文档汇总 **当前已完成内容**、**待实现功能** 与 **待调试项**，并按推荐 **搭建先后顺序** 排列。与 [`NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md`](NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md) §11（8 周 MVP）及 [`GO_IMPLEMENTATION.md`](GO_IMPLEMENTATION.md) 对齐。

---

## 0. 当前基线（2026-05）

| 类别 | 状态 | 说明 |
|------|------|------|
| Go 项目骨架 | ✅ | `cmd/`、`internal/*`、`pkg/artifacts` |
| 工作流 YAML 加载 | ✅ | `novel-short-douyin.yaml` 8 阶段 |
| `run_workflow` 阶段机 | ✅ | 产物检查 + 门禁 |
| CLI | ✅ | `run` / `resume` / `vault search` / `version` |
| Plan / Write / Continuity / Storyboard | ✅ | DeepSeek + 百炼，见阶段 B～D 文档 |
| Produce（阶段 E） | ✅ | TTS + 万相 + FFmpeg；可灵 API 待接 |
| Compliance（阶段 F） | ✅ | 词库扫描 + `no_block_issues` 门禁 |
| Publish（阶段 G） | ✅ | 发布文案 + 封面 + 人工门禁交互 |
| SeriesVault | ✅ | SQLite FTS5 |
| FFmpeg 合成 | ✅ | Ken Burns + 旁白 + 字幕；无 FFmpeg 时占位 |
| 成本账本（阶段 H） | ✅ | Provider 用量 + stack 单价折算；`flowagent cost --run-id` |
| Learn / 指标回流（阶段 I） | ✅ | `metrics set` + Planner 读 publish-metrics / next-hints |
| 串联验收（阶段 J） | ✅ | hooks + condition 门禁 + CI + `accept-series-e2e.ps1` |
| 成片增强（阶段 K） | ✅ 已实现 | 音画同步、ASS 字幕、可灵 API、可选 BGM → [**PHASE_K_SUMMARY.md**](PHASE_K_SUMMARY.md) |

**一次完整 run 产物说明**：见上文对话或 `runs/<run_id>/` 目录结构（`manifest.json`、`artifacts/*`）。

---

## 1. 推荐总顺序（一览）

```text
阶段 A  环境与联调基线          ← 先能稳定 build/run
阶段 B  W1 真实 Plan + Write    ← 内容生产核心
阶段 C  W2 Vault + Continuity   ← 连载一致性
阶段 D  W3 Storyboard + 校验    ← 可拍分镜
阶段 E  W4 Produce 量产         ← TTS + 出图 + 成片
阶段 F  W5 Compliance           ← 合规拦截
阶段 G  W6 Publish              ← 发布包 / 可选抖音 API
阶段 H  W7 成本与 ROI           ← 真实记账
阶段 I  W8 Learn + 可选 Web     ← 数据回流与看板
阶段 J  串联验收与优化          ← 连续 3 集端到端
阶段 K  成片增强（媒体管线）    ← 音画同步 / 动画 / 字幕 / BGM（见 PHASE_K_MEDIA_PIPELINE.md）
```

---

## 2. 阶段 A：环境与联调基线（优先）

**目标**：本地开发环境稳定，避免「假报错」和配置踩坑。

| 序号 | 类型 | 任务 | 涉及路径 / 命令 |
|------|------|------|-----------------|
| A1 | 调试 | 确认 `go build -o bin\flowagent.exe .\cmd\flowagent` 成功 | 项目根 |
| A2 | 调试 | 配置 API Key：`flowagent config init` → 编辑 `providers.local.yaml` → `config check` | [`API_KEYS_SETUP.md`](API_KEYS_SETUP.md) |
| A3 | 调试 | 验证 dry-run 全流程：`run ... --dry-run --auto-gate`，检查 `runs/<id>/artifacts` | CLI |
| A4 | 实现 | （已完成）日志输出到 stdout，避免 PowerShell 将 stderr INFO 标红 | `cmd/flowagent/cmd/root.go` |
| A5 | 调试 | 安装 FFmpeg 并 `ffmpeg -version`；非 dry-run 跑一次 produce | 系统 PATH |
| A6 | 实现 | `resume` 从中间阶段续跑：故意删某产物后 `--from-stage write` 验证 | `cmd/flowagent/cmd/resume.go` |
| A7 | 文档 | README 链到本文档 | `README.md` |

**验收**：`EXIT=0`，末尾出现 `OK workflow finished.`，无业务 err。

---

## 3. 阶段 B：W1 — 真实 Plan + Write（DeepSeek）

**目标**：`episode-brief.md`、`hook-plan.json`、`chapter.md` 由模型生成，Writer 支持流式分场景落盘。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| B1 | 实现 | `RunContext` / stage 注入 `provider.Bundle`（LLM 客户端） | `internal/runner`、`internal/stage` |
| B2 | 实现 | Planner：读 `series-bible.yaml` + 上集摘要 → 调 DeepSeek → 写 brief / hook-plan | `internal/agent/planner.go` |
| B3 | 实现 | Writer：按 `hook-plan` 的 scene 列表 **Stream** 写入 `chapter.parts/` | `internal/agent/writer.go` |
| B4 | 实现 | 合并 `chapter.parts` → `chapter.md`；API 中断可 resume | `internal/agent/writer.go` |
| B5 | 实现 | DeepSeek **SSE 流式**解析（替换当前整段 Complete） | `internal/provider/llm/deepseek.go` |
| B6 | 调试 | 无 Key 时明确报错；有 Key 时对比 dry-run 产物差异 | `config/providers.local.yaml` |
| B7 | 调试 | `length_in_range` 门禁：按 brief 字数/时长校验 chapter | `internal/runner/gates.go` |
| B8 | 实现 | Plan 阶段 **人工门禁**交互（无 `--auto-gate` 时暂停确认） | `internal/runner/gates.go`、CLI |

**验收**：去掉 `--dry-run`，brief 与 chapter 内容随 prompt 变化；`chapter.parts/` 有多文件。

---

## 4. 阶段 C：W2 — SeriesVault（SQLite FTS）+ Continuity

**目标**：人设/伏笔可检索；一致性失败回退 Write。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| C1 | 实现 | 引入 `modernc.org/sqlite`，`series/<id>/vault/index.db` | `go.mod`、`internal/vault/` |
| C2 | 实现 | 索引：`series-bible`、章节摘要、伏笔表（FTS5） | `internal/vault/` |
| C3 | 实现 | `vault search --query` 真实检索（替换仅列文件名） | `internal/vault/vault.go`、`cmd/.../vault.go` |
| C4 | 实现 | Continuity：调 **qwen-plus**（DashScope）读 chapter + bible → `continuity-report.json` | `internal/provider/llm/dashscope.go`（新建）、`internal/agent/continuity.go` |
| C5 | 实现 | `critical_count > 0` 时 runner **回退** 到 write（仅重写问题 scene） | `internal/runner/runner.go` |
| C6 | 调试 | 故意改 bible 与 chapter 冲突，应触发 critical | 手动测试 |
| C7 | 实现 | `character-state.patch.json` 应用后更新 `series/.../character-state.json` | `internal/vault/`、`series/` |

**验收**：冲突能被检出；修复后 continuity 通过；`vault search` 能命中关键词。

---

## 5. 阶段 D：W3 — Storyboard + JSON Schema 校验

**目标**：`storyboard.json` 与 `chapter.md` 旁白一致，总时长误差 ≤ 5%。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| D1 | 实现 | DashScope qwen-plus 生成 `storyboard.json` + `narration.ssml` | `internal/agent/storyboard.go` |
| D2 | 实现 | 使用 `pkg/artifacts.Storyboard` 结构体；禁止手写固定 6 镜 | `pkg/artifacts/storyboard.go` |
| D3 | 实现 | JSON Schema 校验（`storyboard_v1`）或关键字段 `Validate()` | `pkg/artifacts/` 或 `github.com/santhosh-tekuri/jsonschema` |
| D4 | 实现 | `duration_ok` 门禁：`abs(total - target) <= 3` | `internal/runner/gates.go` |
| D5 | 调试 | chapter 与 storyboard 旁白 diff（应为 0 或可控阈值） | 测试脚本 / 人工 |
| D6 | 实现 | `ai_video_budget` 仅 4～6 镜为 true（标准档） | `storyboard` 生成 prompt |

**验收**：180s 目标时总时长 177～183s；JSON 非法时 stage 失败。

---

## 6. 阶段 E：W4 — Produce（TTS + 出图 + FFmpeg 成片）

**目标**：可播放的竖屏 `master.mp4`（约 3 分钟）。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| E1 | 实现 | 火山豆包 TTS：`provider/tts/volcengine` | `internal/provider/tts/` |
| E2 | 实现 | 万相出图：`provider/image/dashscope`，9:16，并行 `errgroup` | `internal/provider/image/` |
| E3 | 实现 | 可灵图生视频：仅 `ai_video_budget` 镜头 | `internal/provider/video/kling` |
| E4 | 实现 | `timeline.json` 由 storyboard 驱动（非硬编码） | `internal/agent/producer.go` |
| E5 | 实现 | `compose/ffmpeg`：静图 Ken Burns + 铺 TTS + 烧字幕 | `internal/compose/ffmpeg/` |
| E6 | 调试 | 无 FFmpeg 时友好提示；有 FFmpeg 时 `master.mp4` 可用播放器打开 | 本地 |
| E7 | 调试 | 并发限流（`golang.org/x/time/rate`），避免 API 429 | `internal/agent/producer.go` |
| E8 | 调试 | 音画时长：TTS 实测与 `duration_sec` 误差 ≤ 5% | 手工听看 |

**验收**：非 dry-run 生成真实 mp4；`assets/` 含图与旁白 mp3。

---

## 7. 阶段 F：W5 — Compliance（合规）

**目标**：违禁内容在发布前 block。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| F1 | 实现 | 加载 `config/compliance/words.txt` + 平台词库 | `internal/agent/compliance.go` |
| F2 | 实现 | 扫描 chapter、storyboard 字幕、publish 标题描述 | 同上 |
| F3 | 实现 | `compliance-report.json`：`blocked` / `warnings` | 同上 |
| F4 | 调试 | 故意加入敏感词，应 `no_block_issues` 失败 | 手工 |
| F5 | 可选 | DeepSeek 二次抽检（词库为主） | `provider/llm` |

**验收**：含 block 级词时工作流在 comply 阶段失败且 exit≠0。

---

## 8. 阶段 G：W6 — Publish（发布包 / 抖音）

**目标**：`publish-pack.json` 可直接用于手机发布；可选开放平台草稿。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| G1 | 实现 | Publisher：根据 storyboard / 系列风格生成标题、描述、话题 | `internal/agent/publisher.go` |
| G2 | 实现 | 从 `master.mp4` 抽封面帧 → `artifacts/cover.jpg` | `internal/compose/ffmpeg` 或 agent |
| G3 | 实现 | 人工门禁：无 `--auto-gate` 时等待确认 `final_cut_approved` | CLI / runner |
| G4 | 可选 | `adapter/douyin` CreateDraft API | `internal/adapter/douyin/` |
| G5 | 调试 | 发布包路径均为相对 `run_dir`，手机端能找文件 | 手工 |

**验收**：剪映/抖音所需字段齐全；成片路径有效。

---

## 9. 阶段 H：W7 — 成本账本与 ROI

**目标**：`cost-ledger.json` 反映真实 API 用量。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| H1 | 实现 | 各 Provider 返回 token/字符数/张数/秒数 | `internal/provider/*` |
| H2 | 实现 | 统一 `CostRecorder` 按 stack 单价折算 CNY | `internal/runctx` 或 `pkg/artifacts` |
| H3 | 实现 | 阶段结束累加至 `manifest.cost` 并 `SaveCostLedger` | `internal/runner/runner.go` |
| H4 | 调试 | 对照 [`standard-tier.yaml`](../config/stacks/standard-tier.yaml) `cost_targets_cny` 区间 | 单集跑完 |
| H5 | 可选 | CLI `flowagent cost --run-id` 打印明细 | `cmd/flowagent/cmd/` |

**验收**：非 dry-run 一次运行后各项成本非零且 total 合理。

---

## 10. 阶段 I：W8 — Learn + 可选 Web 看板

**目标**：发布数据回流，下一集 Planner 可引用。

| 序号 | 类型 | 任务 | 涉及路径 |
|------|------|------|----------|
| I1 | 实现 | CLI `metrics set/show/list` 手填播放数据 | `cmd/flowagent/cmd/metrics.go` | ✅ |
| I2 | 实现 | Learn 写入 `metrics-snapshot.json` 并归档 vault | `internal/agent/analyst.go` | ✅ |
| I3 | 实现 | Planner 读取 `publish-metrics` / 上集 hints | `internal/agent/planner_context.go` | ✅ |
| I4 | 可选 | `cmd/flowagent/web` 只读列出 runs、artifacts | 新包 | ⏭ |
| I5 | 调试 | 连续跑 ep1→ep2，ep2 brief 应引用 ep1 数据 | 端到端 | 📋 |

**验收**：第 2 集 brief 含第 1 集 metrics 或伏笔提示。

---

## 11. 阶段 J：串联验收与优化

| 序号 | 类型 | 任务 | 说明 |
|------|------|------|------|
| J1 | 调试 | 同一 `series_id` 连续跑 **第 1～3 集** | `scripts/accept-series-e2e.ps1` | 📋 本地 |
| J2 | 调试 | 标准档单集成本 ¥25～60 | `flowagent cost` | 📋 本地 |
| J3 | 实现 | 工作流 hooks 解析与执行 | `internal/workflow/hooks.go`、`runner/hooks.go` | ✅ |
| J4 | 实现 | 门禁 `condition:` 表达式 | `internal/runner/condition.go` | ✅ |
| J5 | 优化 | 单测 workflow/hooks/gates/condition | `*_test.go` | ✅ |
| J6 | 优化 | GitHub Actions CI | `.github/workflows/ci.yml` | ✅ |

---

## 12. 已知待调试问题（来自当前骨架）

| 问题 | 现象 | 处理阶段 | 状态 |
|------|------|----------|------|
| PowerShell stderr 假报错 | 红色 NativeCommandError | A4 | ✅ 已修复（日志改 stdout） |
| dry-run 的 master.mp4 非视频 | 播放器无法打开 | E6；说明即可 | ⚠️ 预期行为 |
| 无 `--auto-gate` 时 publish/plan 卡住 | 人工门禁未实现交互 | B8、G3 | ✅ 已实现 stdin 确认 |
| hooks 未执行 | YAML 里 hooks 无效果 | J3 | ✅ before/after 已执行 |
| Learn 归档目录首次不存在 | 已通过 `Ensure()` 创建 | C | ✅ 基本可用 |
| Provider 未注入 Agent | 始终占位内容 | B1 | ✅ 已实现 |

---

## 13. 依赖与环境检查清单

实施前确认：

- [ ] Go 1.22+
- [ ] `config/providers.local.yaml`（DeepSeek、DashScope、火山、可灵）
- [ ] FFmpeg（阶段 E 必需）
- [ ] 剪映（人工字幕安全区/BGM，计划书 §9.8）
- [ ] 可选：`GOPROXY=https://goproxy.cn,direct`

---

## 14. 相关文档索引

| 文档 | 用途 |
|------|------|
| [`NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md`](NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md) | 产品架构、§9 标准档 AI 搭配 |
| [`GO_IMPLEMENTATION.md`](GO_IMPLEMENTATION.md) | Go 包职责、CLI、依赖 |
| [`workflows/novel-short-douyin.yaml`](workflows/novel-short-douyin.yaml) | 阶段与产物契约 |
| [`../config/stacks/standard-tier.yaml`](../config/stacks/standard-tier.yaml) | 模型与成本目标 |
| [`../README.md`](../README.md) | 快速开始 |

---

## 15. 建议个人开发者执行节奏

| 周次 | 阶段 | 最小可演示 |
|------|------|------------|
| 第 1 周 | A + B | 真实 AI 写出一集 chapter |
| 第 2 周 | C | 人设冲突能拦住 |
| 第 3 周 | D | 合法 storyboard.json |
| 第 4 周 | E | 能播放的 mp4 |
| 第 5 周 | F + G | 合规 + 发布包 |
| 第 6 周 | H + I | 成本 + 第 2 集引用第 1 集 |
| 第 7～8 周 | J | 连续 3 集、修 CI/门禁 |

---

*文档版本：1.0 · 基于骨架跑通后的后续路线 · 随实现进度更新 checkbox*
