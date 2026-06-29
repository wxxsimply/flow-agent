# 阶段 G 完成总结（Publish 发布包）

> 对应路线图 [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §8（W6 Publish）

**前置阶段**：A～F（含合规通过的 `master.mp4`）  
**下一阶段**：H（成本账本）

---

## 1. 项目阶段总览（截至 2026-05）

| 阶段 | 名称 | 状态 | 说明 |
|------|------|------|------|
| **A** | 环境与联调基线 | ✅ 完成 | build、config、resume、日志 |
| **B** | Plan + Write | ✅ 完成 | DeepSeek SSE、字数门禁 |
| **C** | Vault + Continuity | ✅ 完成 | FTS、critical 回退 |
| **D** | Storyboard | ✅ 完成 | 百炼分镜、duration_ok |
| **E** | Produce | ✅ 基本完成 | TTS/万相/FFmpeg；可灵待接；TTS 时长待优化 |
| **F** | Compliance | ✅ 完成 | 词库 + no_block_issues |
| **G** | Publish | ✅ 本次完成 | 发布文案、封面、人工门禁 |
| **H** | 成本 ROI | ✅ 已完成 | 见 [PHASE_H_SUMMARY.md](PHASE_H_SUMMARY.md) |
| **I** | Learn | ⏳ 骨架 | analyst 占位 |
| **J** | 串联验收 | ⏳ 待做 | hooks、CI、连续 3 集 |

---

## 2. 阶段 G 任务清单

| 编号 | 任务 | 状态 |
|------|------|------|
| G1 | LLM/模板生成 publish-pack | ✅ |
| G2 | FFmpeg 抽封面 cover.jpg | ✅ |
| G3 | 人工门禁 outline / final_cut / publish_authorized | ✅ |
| G4 | 抖音 API | ⏸ 未实现 |
| G5 | 相对路径 video_path / cover_path | ✅ |

---

## 3. 实现要点

- **G1**：优先 DeepSeek（与 planner 同配置）生成 JSON；失败则用 brief + hook-plan 模板
- **G2**：真实 MP4（`ftyp` 头）时在第 1 镜时间点截帧；占位 mp4 则跳过并 warning
- **G3**：`PromptHumanGates` 在 `CheckGates` 前 stdin 确认；`--auto-gate` 仍自动放行

---

## 4. 手动验证

```powershell
$runId = "<已有 master.mp4 的 run>"
$runDir = Join-Path (Get-Location) "runs\$runId"

.\bin\flowagent.exe resume --run-id $runId --from-stage publish --auto-gate -v

Get-Content -Encoding UTF8 (Join-Path $runDir "artifacts\publish-pack.json")
Test-Path (Join-Path $runDir "artifacts\cover.jpg")
```

---

## 5. 相关文档

- [PHASE_G_COMPLETED.md](PHASE_G_COMPLETED.md)
- [PHASE_F_SUMMARY.md](PHASE_F_SUMMARY.md)
