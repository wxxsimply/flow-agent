# 阶段 E 完成说明（Produce：TTS + 出图 + FFmpeg 成片）

对应路线图：[IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §6。  
**完成总结与手动操作清单**：[PHASE_E_SUMMARY.md](PHASE_E_SUMMARY.md)

---

## 1. 阶段目标

| 编号 | 目标 | 状态 |
|------|------|------|
| E1 | 火山豆包 TTS | 已实现 OpenSpeech HTTP；无 Key 时回退百炼 CosyVoice 或静音轨 |
| E2 | 万相出图 9:16 + errgroup 并行 | 已实现 |
| E3 | 可灵图生视频（仅 ai_video_budget） | 接口 + 回退 Ken Burns（真实 Kling API 待接） |
| E4 | timeline.json 由 storyboard 驱动 | 已实现 |
| E5 | FFmpeg Ken Burns + 铺 TTS + 烧字幕 | 已实现 |
| E6 | 无 FFmpeg 友好提示 | 已实现 stderr 提示 + 占位文件 |
| E7 | 并发限流 rate.Limiter | 已实现（出图 2 QPS  burst） |
| E8 | TTS 时长 vs 目标 ±5% warning | 已实现 ffprobe + slog.Warn |

---

## 2. 主要文件

| 模块 | 路径 |
|------|------|
| Producer | `internal/agent/producer.go` |
| Timeline | `pkg/artifacts/timeline.go` |
| TTS 百炼 | `internal/provider/tts/dashscope.go` |
| TTS 火山 | `internal/provider/tts/volcengine.go` |
| 万相出图 | `internal/provider/image/dashscope.go` |
| 可灵 | `internal/provider/video/kling.go` |
| FFmpeg 合成 | `internal/compose/ffmpeg/ffmpeg.go` |
| Provider 集合 | `internal/provider/bundle.go` |

---

## 3. 如何验证

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
go test ./...

# dry-run produce（需先有 storyboard.json）
.\bin\flowagent.exe resume --run-id <run_id> --from-stage produce --dry-run --auto-gate -v

# 真实 produce（需 dashscope Key；volcengine/kling 可选）
.\bin\flowagent.exe resume --run-id <run_id> --from-stage produce --auto-gate -v
```

检查产物：

```powershell
dir runs\<run_id>\artifacts\assets
Test-Path runs\<run_id>\artifacts\narration.mp3
Test-Path runs\<run_id>\artifacts\timeline.json
Test-Path runs\<run_id>\artifacts\master.mp4
Get-Content -Encoding UTF8 runs\<run_id>\artifacts\timeline.json | Select-Object -First 20
```

**有 FFmpeg 时**：`master.mp4` 为真实竖屏视频（Ken Burns + 旁白轨）。  
**无 FFmpeg 或 dry-run**：`master.mp4` 为文本占位（预期行为）。

---

## 4. API Key 说明

| 能力 | 配置 | 必需 |
|------|------|------|
| 出图 + 百炼 TTS 回退 | `dashscope.api_key` | 非 dry-run 推荐 |
| 火山 TTS 优先 | `volcengine.access_key`（作 token）+ `app_id` | 可选 |
| 可灵 | `kling.api_key` | 可选（无则 Ken Burns） |

---

## 5. 下一阶段（F）

- Compliance 违禁词扫描
- `compliance-report.json` + `no_block_issues` 门禁
