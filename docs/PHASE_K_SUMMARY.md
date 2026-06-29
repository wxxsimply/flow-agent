# 阶段 K 完成摘要

## 已实现

| 子阶段 | 内容 | 关键路径 |
|--------|------|----------|
| **K1** | 按镜 TTS、`audio_segments.json`、timeline v2（`audio_start_sec` / `audio_duration_sec`）、合成去掉 `-shortest`、`sync-report.json` | `internal/agent/produce_media.go`、`pkg/artifacts/timeline.go` |
| **K2** | 可灵 JWT（AK+SK）、提交/轮询/下载 mp4 | `internal/provider/video/kling.go` |
| **K3** | `subtitles.ass` 按镜/按句时间轴，全局烧录 | `internal/compose/subtitles/ass.go` |
| **K4** | stack `compose.bgm_*`，FFmpeg amix 混 BGM | `internal/compose/ffmpeg/ffmpeg.go` |
| **K5** | 门禁 `av_sync_ok`、工作流产物、分镜 prompt 取消 20 字限制 | `docs/workflows/novel-short-douyin.yaml` |

## 你需要手动做的

见 [`PHASE_K_PREREQUISITES.md`](PHASE_K_PREREQUISITES.md)。

- **必做**：百炼 Key + FFmpeg（与阶段 E 相同）
- **可灵动画**：`config/providers.local.yaml` 配置 `access_key` / `secret_key`
- **BGM**：将 `compose.bgm_enabled: true` 并放置 `assets/bgm/default.mp3`

## 验证

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
go test ./...
.\scripts\accept-series-e2e.ps1 -LiveRun -AutoGate
```

检查 `artifacts/sync-report.json` 中 `max_drift_sec <= 0.5`、`artifacts/subtitles.ass` 存在。
