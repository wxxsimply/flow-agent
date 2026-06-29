# 阶段 I 完成说明

**总结**：[PHASE_I_SUMMARY.md](PHASE_I_SUMMARY.md)

## 新增路径

| 路径 | 作用 |
|------|------|
| `pkg/artifacts/metrics.go` | `PublishMetrics` 结构与序列化 |
| `internal/vault/metrics.go` | `publish-metrics/`、`vault/episode-*-next-hints.md` |
| `cmd/flowagent/cmd/metrics.go` | `metrics set` / `show` / `list` |
| `internal/agent/metrics_hints.go` | `metrics set` 后生成下集 hints |
| `internal/agent/planner_context.go` | Planner 注入上下文 |

## 推荐工作流（个人 MVP）

1. 跑完 ep1 全流程（含 learn）。
2. 视频发布约 24h 后录入指标：`metrics set ...`。
3. `metrics set` 会自动写入 `vault/episode-002-next-hints.md`；若需合并 run 内 hook/brief，可 `resume --from-stage learn`。
4. 跑 ep2 `run ... --episode 2`，检查 brief 是否引用指标。

## 文件布局示例

```text
series/demo/publish-metrics/episode-001.json
series/demo/vault/episode-002-next-hints.md
series/demo/vault/episode-001-summary.md
```
