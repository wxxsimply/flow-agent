# 阶段 E 完成总结（Produce：TTS + 出图 + FFmpeg 成片）

> 汇总文档 · 对应路线图 [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §6（W4 Produce）  
> 逐步验证命令见 [PHASE_E_COMPLETED.md](PHASE_E_COMPLETED.md)

**完成时间**：2026-05  
**前置阶段**：D（已有 `storyboard.json`、`narration.ssml`）  
**下一阶段**：F（Compliance 合规扫描）

---

## 1. 阶段目标回顾

根据分镜生成竖屏成片（约 180s）：旁白 TTS、分镜配图、可选可灵短视频，经 FFmpeg 合成 `master.mp4`，并输出 `timeline.json` 与 `assets/`。

---

## 2. 任务完成清单

| 编号 | 路线图任务 | 完成状态 | 说明 |
|------|------------|----------|------|
| **E1** | 火山豆包 TTS `provider/tts/volcengine` | ✅ 已完成 | 配置了 `volcengine.access_key` + `app_id` 时优先；失败回退百炼 CosyVoice，再失败则静音轨 |
| **E2** | 万相出图 `provider/image/dashscope` 9:16 并行 | ✅ 已完成 | `errgroup` 并行；`rate.Limiter` 约 2 QPS；失败则占位 PNG |
| **E3** | 可灵图生视频（仅 `ai_video_budget`） | ⚠️ 部分 | 接口已接；**真实 Kling API 未实现**，自动回退 Ken Burns 静图动效 |
| **E4** | `timeline.json` 由 storyboard 驱动 | ✅ 已完成 | `pkg/artifacts/timeline.go` — `BuildTimeline()` |
| **E5** | FFmpeg Ken Burns + 铺 TTS + 烧字幕 | ✅ 已完成 | 静图 Ken Burns + 底部 `drawtext` 字幕（Windows 优先微软雅黑字体） |
| **E6** | 无 FFmpeg 友好提示 | ✅ 已完成 | stderr 提示 + `master.mp4` 文本占位 |
| **E7** | 并发限流防 429 | ✅ 已完成 | `golang.org/x/time/rate` |
| **E8** | TTS 时长 vs 分镜 ±5% | ✅ 已实现检测 | `ffprobe` + `slog.Warn`（不阻断）；**听感需人工确认** |

---

## 3. 实现摘要

### 3.1 处理流程

```text
produce 阶段
  → 读取 storyboard.json + narration.ssml
  → E1: TTS → artifacts/narration.mp3
  → E2: 万相并行出图 → artifacts/assets/sXX.png
  → E3: 可灵（可选，失败跳过）→ assets/sXX.mp4
  → E4: BuildTimeline → artifacts/timeline.json
  → E5: FFmpeg 按镜拼接 + 铺旁白 + 烧字幕 → artifacts/master.mp4
```

### 3.2 主要文件

| 模块 | 路径 |
|------|------|
| Producer | `internal/agent/producer.go` |
| Timeline | `pkg/artifacts/timeline.go` |
| TTS 百炼 | `internal/provider/tts/dashscope.go` |
| TTS 火山 | `internal/provider/tts/volcengine.go` |
| 万相出图 | `internal/provider/image/dashscope.go` |
| 可灵 | `internal/provider/video/kling.go`（MVP 占位） |
| FFmpeg | `internal/compose/ffmpeg/ffmpeg.go` |
| Provider 注入 | `internal/provider/bundle.go` |

### 3.3 产物清单

| 路径 | 说明 |
|------|------|
| `artifacts/narration.mp3` | 全片旁白 |
| `artifacts/assets/s01.png` … | 各镜配图 |
| `artifacts/assets/sXX.mp4` | 可灵成功时才有（当前通常无） |
| `artifacts/timeline.json` | 合成时间轴 |
| `artifacts/master.mp4` | 竖屏成片 |

---

## 4. 你需要手动完成的操作（重要）

以下步骤 **无法由代码自动代替**，需在本地执行。

### 4.1 环境

| 步骤 | 操作 | 说明 |
|------|------|------|
| ① | 安装 **FFmpeg** | 推荐项目内：`.\scripts\setup-ffmpeg.ps1` → `ffmpeg\bin/ffmpeg.exe`；或系统 PATH / `FFMPEG_PATH` |
| ② | Go 1.22+，`go build` 成功 | 项目根执行 |
| ③ | （推荐）安装支持中文的字体 | Windows 一般已有 `msyh.ttc`；无字幕可跳过 |

### 4.2 API Key（真实 produce 必需）

```powershell
cd D:\Code\flow-agent
.\bin\flowagent.exe config init
# 编辑 config/providers.local.yaml
.\bin\flowagent.exe config check
```

| 能力 | 配置项 | 真实 produce |
|------|--------|--------------|
| 万相出图 + 百炼 TTS 回退 | `dashscope.api_key` | **必需** |
| 火山 TTS 优先 | `volcengine.access_key`、`volcengine.app_id` | 可选 |
| 可灵短视频 | `kling.api_key` | 可选（当前 API 未接，无 Key 不影响） |
| Plan/Write | `deepseek.api_key` | 全流程 run 时需要；仅 resume produce 可不填 |

### 4.3 准备 run（必须有 storyboard）

produce 依赖 **D 阶段产物**。任选其一：

- 从头跑：`flowagent run novel-short-douyin --series demo --episode 1 --auto-gate -v`（到 storyboard 后再 produce）
- 续跑已有 run：

```powershell
$runId = "<已有 storyboard 的 run_id>"
.\bin\flowagent.exe resume --run-id $runId --from-stage produce --auto-gate -v
```

确认前置文件：

```powershell
$runDir = Join-Path (Get-Location) "runs\$runId"
Test-Path (Join-Path $runDir "artifacts\storyboard.json")   # 必须 True
Test-Path (Join-Path $runDir "artifacts\narration.ssml")  # 必须 True
```

### 4.4 两种验证模式

**A. dry-run（不调云 API，适合先通路）**

```powershell
.\bin\flowagent.exe resume --run-id $runId --from-stage produce --dry-run --auto-gate -v
```

- 占位 PNG / 静音或占位 mp3
- 无 FFmpeg 时 `master.mp4` 为文本占位（**不能用播放器打开**，属预期）

**B. 真实 produce（验收 E 阶段）**

```powershell
.\bin\flowagent.exe resume --run-id $runId --from-stage produce --auto-gate -v
```

检查：

```powershell
dir (Join-Path $runDir "artifacts\assets")
# PNG 应 > 几 KB（非 26 字节占位）
(Get-Item (Join-Path $runDir "artifacts\narration.mp3")).Length
# 用播放器打开 master.mp4（需已安装 FFmpeg）
```

### 4.5 E8 人工听看（仅真实 produce）

- 播放 `master.mp4`，听旁白是否与分镜节奏大致一致
- 日志若出现 `tts duration mismatch`，表示 TTS 时长与分镜目标相差 >5%，可调整 SSML 或分镜时长

### 4.6 常见问题

| 现象 | 处理 |
|------|------|
| `dashscope api_key required for produce` | 填 Key 或加 `--dry-run` |
| PNG 只有 26 字节 | 出图 API 失败，看日志；检查百炼 Key 与模型 `wan2.6-t2i` |
| `master.mp4` 打不开 | 未装 FFmpeg 或仍是 dry-run 占位 |
| 字幕乱码/方框 | Windows 确认 `C:\Windows\Fonts\msyh.ttc` 存在 |
| produce 很慢 | 万相异步轮询 + 多镜并行，正常需数分钟 |

---

## 5. 本地验证结论（开发机）

| 验证项 | 说明 |
|--------|------|
| `go test ./...` | 单元测试通过 |
| dry-run produce | 生成 timeline、占位 assets、占位 master |
| 真实 produce | 需你本机 **DashScope Key + FFmpeg** 后自行跑一遍 |

---

## 6. 已知限制

| 项 | 说明 |
|----|------|
| E3 可灵 | 仅占位，有 Key 也会回退 Ken Burns |
| 字幕 | 依赖系统字体；无中文字体时可能显示异常 |
| 成本账本 | produce 仍写入估算常数，真实用量见阶段 H |
| 剪映后期 | BGM、精细字幕安全区需人工（计划书 §9.8） |

---

## 7. 相关文档

| 文档 | 用途 |
|------|------|
| [PHASE_E_COMPLETED.md](PHASE_E_COMPLETED.md) | 命令速查 |
| [API_KEYS_SETUP.md](API_KEYS_SETUP.md) | Key 配置 |
| [PHASE_D_COMPLETED.md](PHASE_D_COMPLETED.md) | 上一阶段 Storyboard |
| [PHASE_F_SUMMARY.md](PHASE_F_SUMMARY.md) | 下一阶段 Compliance |
