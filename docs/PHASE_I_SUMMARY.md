# 阶段 I（Learn + 指标回流）完成总结

## 实现内容

| 任务 | 状态 | 说明 |
|------|------|------|
| I1 | ✅ | `flowagent metrics set/show/list`；set 后自动刷新 `episode-NNN-next-hints.md` |
| I2 | ✅ | Learn 合并指标 → `metrics-snapshot.json`、vault、`episode-NNN-next-hints.md` |
| I3 | ✅ | Planner prompt 注入上集 publish-metrics + next-hints |
| I4 | ⏭ | Web 看板（可选，未做） |
| I5 | 📋 | 需本地 ep1→录入 metrics→ep2 plan 验收 |

## 数据流

```text
发布后在抖音看到数据
  → flowagent metrics set --series demo --episode 1 --views ... --completion ...
  → series/demo/publish-metrics/episode-001.json

ep1 learn（或 metrics set 后重跑 learn）
  → artifacts/metrics-snapshot.json
  → series/demo/vault/episode-002-next-hints.md

ep2 plan
  → brief 引用上集指标与 hints
```

## 常用命令

```powershell
# 录入第 1 集播放数据（发布 24h 后）
.\bin\flowagent.exe metrics set --series demo --episode 1 --views 24000 --completion 0.35 --keywords "爽点,雨夜"

# 查看 / 列出
.\bin\flowagent.exe metrics show --series demo --episode 1
.\bin\flowagent.exe metrics list --series demo

# 可选：同步到某次 run 产物
.\bin\flowagent.exe metrics set --series demo --episode 1 --views 24000 --completion 0.35 --run-id <run_id>

# 第 2 集策划（会读上集 metrics + hints）
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 2 --auto-gate
```

## 验收（I5）

第 2 集 `artifacts/episode-brief.md` 应出现上集播放/完播或评论热词，或明确承接 cliffhanger 的表述。

详见 [PHASE_I_COMPLETED.md](PHASE_I_COMPLETED.md)。

## 下一阶段

**阶段 J**：连续 3 集端到端、CI、hooks 等。
