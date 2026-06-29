# 阶段 J（串联验收与优化）完成总结

## 实现内容

| 任务 | 状态 | 说明 |
|------|------|------|
| J1 | 📋 脚本 | `scripts/accept-series-e2e.ps1` 连续 ep1–3 |
| J2 | 📋 本地 | 非 dry-run 后用 `flowagent cost` 对照 ¥25–60 |
| J3 | ✅ | `workflow.ParseStageHooks` + `runner.RunHooks` |
| J4 | ✅ | `gate.condition` → `EvaluateGateCondition` |
| J5 | ✅ | workflow/hooks、gates、condition 单测（storyboard 已有） |
| J6 | ✅ | `.github/workflows/ci.yml` |

## Hooks 行为

| 动作 | 时机 | 作用 |
|------|------|------|
| `inject_l0_series_bible` | before plan | Ensure vault + 加载 bible |
| `inject_publish_metrics` | before plan | 日志上集 metrics |
| `inject_l1_episode_brief` | before write | 检查 brief 存在 |
| `inject_l2_foreshadows_if_needed` | before continuity | 检查 foreshadows.yaml |
| `archive_episode_to_series_vault` | after learn | 归档 chapter 摘要到 vault |
| stream hooks | write 内 | Writer 已处理，runner 仅 ack |

## 验收命令

```powershell
# J1 dry-run 三连跑（约 5 分钟，无 API）
.\scripts\accept-series-e2e.ps1 -AutoGate

# J1 真实 API（需 Key + FFmpeg）
.\scripts\accept-series-e2e.ps1 -LiveRun -AutoGate

# J2 单集成本
.\bin\flowagent.exe cost --run-id <run_id>
```

详见 [PHASE_J_COMPLETED.md](PHASE_J_COMPLETED.md)。
