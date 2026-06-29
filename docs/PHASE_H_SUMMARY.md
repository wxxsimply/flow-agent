# 阶段 H（成本账本）完成总结

## 实现内容

| 任务 | 状态 | 说明 |
|------|------|------|
| H1 Provider 用量 | ✅ | LLM `usage` 字段或字符估算；Produce 记录 TTS 字符、出图张数、视频秒数 |
| H2 CostRecorder | ✅ | `internal/cost` + `config/stacks/*` 的 `unit_prices_cny` |
| H3 阶段同步 | ✅ | `RecordLLM/TTS/Image/Video` → `manifest.cost` + `cost-ledger.json` |
| H4 对照预算 | ✅ | `CompareTargets` + `flowagent cost` 打印区间；超预算时 workflow WARN |
| H5 CLI | ✅ | `flowagent cost --run-id <uuid>` |

## 关键路径

- `internal/cost/recorder.go`、`rates.go`
- `internal/runctx/context.go` — `InitCostRecorder`、`Record*`
- `pkg/artifacts/manifest.go` — ledger 含用量字段
- `config/stacks/standard-tier.yaml` — `unit_prices_cny`

## 验证

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
go test ./...

.\bin\flowagent.exe cost --run-id <run_id>
```

非 dry-run 跑完后 `cost-ledger.json` 中 `llm_cny`、`tts_cny`、`image_cny` 等应非零（有真实 API 调用时）。

完整说明见 [PHASE_H_COMPLETED.md](PHASE_H_COMPLETED.md)。

## 下一阶段

**阶段 I**：Learn + metrics 回流（`metrics-snapshot.json`、Planner 引用上集数据）。
