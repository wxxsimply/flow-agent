# 阶段 J 完成说明

**总结**：[PHASE_J_SUMMARY.md](PHASE_J_SUMMARY.md)

## 代码变更

| 路径 | 作用 |
|------|------|
| `internal/workflow/hooks.go` | 解析 YAML hooks |
| `internal/runner/hooks.go` | 执行 before/after hooks |
| `internal/runner/condition.go` | 门禁 `condition` 表达式 |
| `.github/workflows/ci.yml` | CI：`go test` + `go build` |
| `scripts/accept-series-e2e.ps1` | ep1–3 串联验收 |

## J1 验收标准

每集 run 目录应含：

- `artifacts/episode-brief.md`
- `artifacts/chapter.md`
- `artifacts/storyboard.json`
- `artifacts/master.mp4`
- `artifacts/publish-pack.json`
- `artifacts/metrics-snapshot.json`

`manifest.stage` 为 `finished`。

## J2 验收标准

`flowagent cost --run-id` 的 `total` 落在 `standard-tier` 的 `cost_targets_cny.total`（¥25–60）附近；超出会有 workflow WARN。

## 已知限制

- `condition` 仅支持 YAML 中已有的四类模式（continuity / compliance / duration / chapter length）。
- `on_scene_complete` 等流式 hook 仍在 Writer 内部，未在 runner 逐步触发。

## PowerShell 说明

FFmpeg 的 banner 会写到 stderr。`flowagent` 已不再把子进程 stderr 接到控制台；验收脚本使用 `$ErrorActionPreference = Continue`，仅以 `$LASTEXITCODE` 判断失败。
