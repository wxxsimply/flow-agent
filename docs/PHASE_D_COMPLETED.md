# 阶段 D 完成说明（Storyboard + JSON 校验 + 时长门禁）

对应路线图：[IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §5。

---

## 1. 阶段目标

| 编号 | 目标 | 状态 |
|------|------|------|
| D1 | 百炼 qwen-plus 生成 `storyboard.json` + `narration.ssml` | 已实现 |
| D2 | 使用 `Storyboard` 结构体，禁止固定 6 镜 | 已实现 |
| D3 | `Validate()` 校验 storyboard_v1 | 已实现 |
| D4 | `duration_ok` 门禁（误差 ≤3s） | 已实现 |
| D5 | chapter 与 storyboard 旁白对齐评分 | 已实现（warning，不 block） |
| D6 | `ai_video_budget` 仅 4～6 镜 | 已实现 |

---

## 2. 实现要点

### Storyboard Agent（百炼）

- [`internal/agent/storyboard.go`](../internal/agent/storyboard.go) — dry-run / live 分支
- [`internal/agent/prompts/storyboard.go`](../internal/agent/prompts/storyboard.go) — system/user prompt
- 输入：`chapter.md`、`hook-plan.json`、`series-bible`
- 输出：`artifacts/storyboard.json`、`artifacts/narration.ssml`
- 后处理：`NormalizeDurations()` 保证总时长在 target±3s

### 契约与校验

- [`pkg/artifacts/storyboard.go`](../pkg/artifacts/storyboard.go) — `Validate()`、`LoadStoryboard()`、`NormalizeDurations()`
- [`pkg/artifacts/storyboard_align.go`](../pkg/artifacts/storyboard_align.go) — `NarrationAlignmentScore()`（子串命中率，<0.8 打 warning）

### 门禁

- [`internal/runner/gates.go`](../internal/runner/gates.go) — `checkDurationOK()` 读取 storyboard 校验总时长
- 已移除 agent 内 `SetGate("duration_ok", true)` 绕过

### 共用工具

- [`internal/agent/json_extract.go`](../internal/agent/json_extract.go) — `ExtractTopLevelJSON`（planner / storyboard 共用）

---

## 3. 如何验证

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
go test ./pkg/artifacts/... ./internal/agent/... ./internal/runner/...

# dry-run（不调 API，8 镜 + 4 ai_video）
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --dry-run --auto-gate -v

# 真实分镜（需 deepseek + dashscope Key）
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --auto-gate -v
```

检查 storyboard：

```powershell
$sb = Get-Content -Encoding UTF8 runs\<run_id>\artifacts\storyboard.json | ConvertFrom-Json
$total = ($sb.shots | Measure-Object -Property duration_sec -Sum).Sum
Write-Host "shots=$($sb.shots.Count) total=${total}s target=$($sb.target_duration_sec)s"
($sb.shots | Where-Object ai_video_budget).Count   # 期望 4~6
```

**验收标准**

- `sum(duration_sec)` ∈ [target-3, target+3]（180s 目标即 177～183s）
- `ai_video_budget` 镜数 4～6
- JSON 非法或 Validate 失败时 storyboard 阶段报错退出

---

## 4. 下一阶段（E）

- 火山 TTS、万相出图、可灵图生视频
- FFmpeg 按 `timeline.json` 真实合成 `master.mp4`

详见路线图阶段 E。
