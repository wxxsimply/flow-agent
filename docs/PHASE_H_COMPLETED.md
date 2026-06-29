# 阶段 H 完成说明

**总结**：[PHASE_H_SUMMARY.md](PHASE_H_SUMMARY.md)

## 已实现

| 编号 | 内容 |
|------|------|
| H1 | LLM `usage` / 流式估算；Produce 记录 TTS 字符、出图张数、视频秒数 |
| H2 | `internal/cost` + `unit_prices_cny` 折算人民币 |
| H3 | 各阶段 `Record*` → `manifest.cost` + `cost-ledger.json`（含用量字段，resume 可恢复） |
| H4 | `cost.CompareTargets` + `flowagent cost` 预算对照 + 工作流结束 WARN |
| H5 | `flowagent cost --run-id`（默认可读报告，`--json` 原始 JSON） |

## 验证命令

```powershell
cd D:\Code\flow-agent
go build -o bin\flowagent.exe .\cmd\flowagent

# 查看某次 run 成本（含预算区间）
.\bin\flowagent.exe cost --run-id <run_id>

# 原始 JSON
.\bin\flowagent.exe cost --run-id <run_id> --json
```

## 说明

- 在阶段 H **之前**已跑完的 run，`cost-ledger` 可能为空；需从 `plan` 或 `produce` 等阶段 `resume` 重跑才会累计用量。
- 单价为 stack 内 `unit_prices_cny` 的**估算折算**，用于 ROI 复盘，不等于云账单精确值。

## 下一阶段

**阶段 I**：`flowagent metrics set`、Learn 数据回流、Planner 引用上集 metrics。
