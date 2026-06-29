# FlowAgent Go 实施指南

本文档是 [`NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md`](NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md) 的 **Go 语言实现附录**：环境、依赖、目录、接口与开发顺序。AI 服务搭配（标准档）仍以计划书 **§9** 为准，Go 只负责编排与调用。

---

## 1. 可以用 Go 吗？

**可以，且推荐。**

| 模块 | Go 负责 | 不负责 |
|------|---------|--------|
| 工作流状态机、产物契约、门禁 | ✅ | — |
| 调 DeepSeek / Qwen / 火山 / 万相 / 可灵 HTTP API | ✅ | — |
| 流式读 LLM、按 scene 写 `chapter.parts/` | ✅ | — |
| SeriesVault（SQLite FTS） | ✅ | — |
| 并行出图、TTS、轮询可灵任务 | ✅ `errgroup` | — |
| 调用 FFmpeg 合成 | ✅ `os/exec` | — |
| 模型推理、画图算法本身 | — | 云端 API |
| 剪映精细字幕/BGM | — | 人工剪映（MVP） |

---

## 2. 本机需要安装的软件

| 软件 | 版本 | 用途 | Windows 安装 |
|------|------|------|----------------|
| **Go** | **1.22+** | 编译运行 FlowAgent | https://go.dev/dl/ 或 `winget install GoLang.Go` |
| **Git** | 最新 | 版本管理 | https://git-scm.com/ |
| **FFmpeg** | 6.x+ | 视频合成 | `winget install Gyan.FFmpeg` |
| **剪映专业版** | 最新 | 终审字幕/BGM | 官网 |
| **抖音（手机）** | — | 发布 | 应用商店 |

**不需要**：Python、Node（MVP 阶段）、CUDA、Docker。

### 2.1 验证安装

```powershell
go version    # go version go1.22.x windows/amd64
ffmpeg -version
git --version
```

### 2.2 环境变量（API Key）

```powershell
$env:DEEPSEEK_API_KEY = "sk-..."
$env:DASHSCOPE_API_KEY = "sk-..."
$env:VOLCENGINE_ACCESS_KEY = "..."
$env:VOLCENGINE_SECRET_KEY = "..."
$env:KLING_API_KEY = "..."
```

或复制 `config/providers.local.yaml.example` → `config/providers.local.yaml`（已 gitignore）。

---

## 3. Go 模块与核心依赖

**模块名**（`go.mod`）：`github.com/flow-agent/flow-agent`（可按你的 Git 远程修改）。

| 用途 | 推荐库 | 说明 |
|------|--------|------|
| CLI | `github.com/spf13/cobra` | 子命令 `run`、`vault`、`resume` |
| YAML | `gopkg.in/yaml.v3` | 加载 `docs/workflows/*.yaml` |
| HTTP 客户端 | 标准库 `net/http` | 各云 API；可加 `golang.org/x/time/rate` 限流 |
| 并发 | `golang.org/x/sync/errgroup` | Produce 并行出图/TTS |
| SQLite | `modernc.org/sqlite` | 纯 Go，免 CGO，FTS5 |
| JSON Schema | `github.com/santhosh-tekuri/jsonschema/v6` | 产物 gate |
| 配置 | 标准库 + yaml | 可选 `github.com/joho/godotenv` 读 `.env` |
| 日志 | `log/slog` | 标准库结构化日志 |
| UUID | `github.com/google/uuid` | `run_id` |

**不引入重型 AI SDK**：DeepSeek 用 OpenAI 兼容 JSON；DashScope/火山/可灵用自有 REST 封装，便于换供应商。

---

## 4. 项目目录与包职责

```text
cmd/flowagent/                 → CLI（run / resume / vault）
internal/runctx/                → Context、Store、产物读写（避免 import cycle）
internal/runner/                → RunWorkflow、gates
internal/workflow/              → YAML 加载
internal/stage/                 → 阶段编排 → agent
internal/agent/                 → Planner、Writer、Storyboarder…
internal/vault/                 → SeriesVault
internal/provider/            → LLM/TTS/图/视频
internal/compose/ffmpeg/        → 合成
internal/adapter/douyin/        → 发布包
pkg/artifacts/                  → Manifest、Storyboard 类型
```

### 4.1 核心接口（示意）

```go
// internal/runner/runner.go
type RunContext struct {
    RunID, SeriesID string
    EpisodeNo       int
    Workflow        string
    StackProfile    string
    TargetDuration  int
    CostBudgetCNY   float64
    RunDir          string
    Stage           string
}

func RunWorkflow(ctx context.Context, rc *RunContext) error
```

```go
// internal/provider/llm/client.go
type Client interface {
    Complete(ctx context.Context, req CompletionRequest) (string, error)
    Stream(ctx context.Context, req CompletionRequest, onChunk func(string) error) error
}
```

```go
// internal/stage/stage.go
type Stage interface {
    ID() string
    Run(ctx context.Context, rc *runner.RunContext) error
}
```

各阶段结束后调用 `runner.EnsureArtifacts` + `workflow.CheckGates`。

---

## 5. CLI 设计（与 YAML 对齐）

```powershell
# 跑一集（标准档 + 3 分钟）
flowagent run novel-short-douyin `
  --series ceo-rebirth-001 `
  --episode 1 `
  --stack standard-tier

# 从某阶段续跑（API 中断后）
flowagent resume --run-id <uuid> --from-stage storyboard

# 检索系列圣经 / 历史摘要
flowagent vault search --series ceo-rebirth-001 --query "伏笔"
```

工作流文件：`docs/workflows/novel-short-douyin.yaml`（路径可通过 `--workflow-dir` 覆盖）。

---

## 6. 标准档 Provider 映射（Go 包）

| 阶段 | Go 包 / 文件 | 外部服务 |
|------|----------------|----------|
| Plan, Write, Comply | `provider/llm/deepseek` | DeepSeek API |
| Continuity, Storyboard | `provider/llm/dashscope` | 通义 qwen-plus |
| Produce · TTS | `provider/tts/volcengine` | 豆包语音 |
| Produce · 静图 | `provider/image/dashscope` | 万相 |
| Produce · 动效 | `provider/video/kling` | 可灵 Turbo |
| Produce · 合成 | `compose/ffmpeg` | 本地 FFmpeg |
| Publish | `adapter/douyin` | 导出包（MVP） |

配置来源：`config/stacks/standard-tier.yaml` + `config/providers.local.yaml`。

---

## 7. 本地构建与运行

```powershell
cd D:\Code\flow-agent
copy config\providers.local.yaml.example config\providers.local.yaml
# 编辑填入 API Key

go mod tidy
go build -o bin\flowagent.exe .\cmd\flowagent

.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --stack standard-tier
```

**Makefile**（可选）：

```makefile
build:
	go build -o bin/flowagent ./cmd/flowagent
test:
	go test ./...
lint:
	go vet ./...
```

---

## 8. 流式写作（StreamSupervisor）Go 实现要点

1. `Writer` 阶段读取 `episode-brief.md` 中的 `scene_list`。
2. 对每个 scene 调用 `llm.Client.Stream`，`bufio.Scanner` 或 SSE reader 累加。
3. 每完成一 scene：`os.WriteFile` → `artifacts/chapter.parts/scene-NN.md`，并 `vault.AppendSceneSummary`。
4. 合并为 `artifacts/chapter.md`。
5. `continuity` 阶段失败时：`runner.RollbackTo("write")`，仅重写标记的 scene。

---

## 9. Produce 并行示例

```go
g, ctx := errgroup.WithContext(ctx)
for _, shot := range storyboard.Shots {
    shot := shot
    g.Go(func() error {
        switch shot.VisualType {
        case "ai_video":
            return kling.ImageToVideo(ctx, shot)
        default:
            return wanx.Generate(ctx, shot)
        }
    })
}
if err := g.Wait(); err != nil { return err }
return ffmpeg.Build(ctx, timeline, runDir)
```

注意：对 API 加 `rate.Limiter`，避免万相/可灵并发超限。

---

## 10. 与计划书阶段的开发顺序（8 周）

| 周 | Go 交付物 |
|----|-----------|
| W1 | `go.mod`、`cmd/flowagent`、`runner`、`workflow` 解析 YAML、`deepseek` 流式 Write |
| W2 | `vault` SQLite FTS、`stage/continuity` |
| W3 | `dashscope` LLM、`pkg/artifacts/storyboard.go` + schema |
| W4 | `volcengine` TTS、`compose/ffmpeg`、`stage/produce` |
| W5 | `stage/comply`、词库文件 `config/compliance/words.txt` |
| W6 | `stage/publish`、`adapter/douyin/export` |
| W7 | `cost-ledger` 写入 `manifest.json` |
| W8 | `stage/learn`、`cmd/flowagent/web`（可选 `net/http` 静态页） |

---

## 11. `.gitignore` 建议

```gitignore
/bin/
/runs/
*.exe
config/providers.local.yaml
.env
series/*/vault/*.db
```

---

## 12. 常见问题

**Q：还要学 Python 吗？**  
A：不需要。仅当你要用别人写的 Python 脚本做一次性实验时可临时用，主项目统一 Go。

**Q：OpenAI 官方 Go SDK 能用吗？**  
A：DeepSeek 兼容 OpenAI 格式时可用 `github.com/openai/openai-go` 指向 DeepSeek `base_url`；Qwen/火山/可灵建议自写薄封装。

**Q：Windows 上 SQLite CGO 麻烦吗？**  
A：用 `modernc.org/sqlite` 免 CGO。

**Q：定时日更？**  
A：Windows 任务计划程序执行 `flowagent run ...`；或 Linux 上 cron。

---

## 13. 相关文档

- 产品与架构：[`NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md`](NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md)
- 标准档 AI：同上 §9
- 工作流：[`workflows/novel-short-douyin.yaml`](workflows/novel-short-douyin.yaml)
- 栈配置：[`../config/stacks/standard-tier.yaml`](../config/stacks/standard-tier.yaml)

---

*文档版本：1.0 · 实现语言：Go 1.22+*
