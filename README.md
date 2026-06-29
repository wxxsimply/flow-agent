# flow-agent

流式小说 → 分镜 → 短视频 → 多平台发布的工作流 Agent（**Go**）。

## 文档

- [docs/NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md](docs/NOVEL_STREAM_VIDEO_PUBLISH_PROPOSAL.md) — 产品与架构
- [docs/GO_IMPLEMENTATION.md](docs/GO_IMPLEMENTATION.md) — Go 环境与包说明
- [docs/IMPLEMENTATION_ROADMAP.md](docs/IMPLEMENTATION_ROADMAP.md) — **后续实现与调试顺序（推荐先看）**
- [docs/PHASE_E_SUMMARY.md](docs/PHASE_E_SUMMARY.md) — 阶段 E（Produce）完成总结与手动验证
- [docs/PHASE_F_SUMMARY.md](docs/PHASE_F_SUMMARY.md) — 阶段 F（Compliance）完成总结
- [docs/PHASE_G_SUMMARY.md](docs/PHASE_G_SUMMARY.md) — 阶段 G（Publish）完成总结
- [docs/PHASE_H_SUMMARY.md](docs/PHASE_H_SUMMARY.md) — 阶段 H（成本账本）完成总结
- [docs/PHASE_H_COMPLETED.md](docs/PHASE_H_COMPLETED.md) — 阶段 H 验收与 `flowagent cost` 用法
- [docs/PHASE_I_SUMMARY.md](docs/PHASE_I_SUMMARY.md) — 阶段 I（Learn + metrics 回流）完成总结
- [docs/PHASE_J_SUMMARY.md](docs/PHASE_J_SUMMARY.md) — 阶段 J（串联验收）完成总结
- [docs/PHASE_K_MEDIA_PIPELINE.md](docs/PHASE_K_MEDIA_PIPELINE.md) — **下一阶段 K**：音画同步、动画、全量字幕、BGM

## 快速开始

```powershell
# 依赖：Go 1.22+，可选 FFmpeg
go mod tidy
go build -o bin/flowagent.exe ./cmd/flowagent

# 占位流水线（不调 API）
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --dry-run --auto-gate -v

# 真实 Plan + Write + Continuity + Storyboard（需 deepseek + dashscope Key）
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --auto-gate -v
```

产物目录：`runs/<run_id>/artifacts/`。

## 命令

| 命令 | 说明 |
|------|------|
| `flowagent run <workflow> --series <id> --episode N` | 执行工作流 |
| `flowagent run micro-movie --plot "..."` | 微电影（默认 **万相 i2v-flash**，stack 见下） |
| `flowagent run micro-movie --stack micro-movie-wan-flash --auto-gate` | **推荐**：万相 t2i + wan2.6-i2v-flash |
| `flowagent run micro-movie --stack micro-movie-seedance --auto-gate` | 火山 Seedream + Seedance（需方舟余额） |
| `flowagent test-shot` | **单镜测试**：stack 配置的 t2i + i2v 一段 mp4 |
| `flowagent test-shot --stack micro-movie-wan-flash` | 万相单镜冒烟（默认 stack） |
| `flowagent test-shot --stack micro-movie-veo-lite` | Veo 3.1 Lite i2v PoC（需 `gemini.api_key`） |
| `flowagent compare-proplock` | 同关键帧对比 Seedance vs Veo Lite 的 PropLock 成片 |
| `flowagent config test-wan-video` | 同 test-shot（默认 wan-flash stack），输出到临时目录 |
| `flowagent resume --run-id <uuid> --from-stage <stage>` | 从某阶段续跑 |
| `flowagent vault search --series <id> --query <关键词>` | FTS 检索系列知识库 |
| `flowagent config check` | 检查 API Key |
| `flowagent version` | 版本 |

常用参数：`--stack standard-tier`、`--dry-run`、`--auto-gate`（开发用，跳过人工门禁）。

## 项目结构

```text
cmd/flowagent/          CLI
internal/
  config/               配置与 stack
  workflow/             YAML 工作流
  runner/               run_workflow、RunStore
  stage/                阶段编排
  agent/                Plan/Write/Continuity/Storyboard/Produce/Compliance 均已接 API
  vault/                SeriesVault
  provider/             LLM/TTS/图/视频（DeepSeek 已实现，其余待接）
  compose/ffmpeg/       视频合成
  adapter/douyin/       发布包
pkg/artifacts/          manifest、storyboard 类型
config/stacks/          standard-tier.yaml
docs/workflows/         novel-short-douyin.yaml
series/                 系列数据
runs/                   运行产物（gitignore）
```

## 配置 API Key

```powershell
.\bin\flowagent.exe config init    # 生成 providers.local.yaml
# 编辑 config/providers.local.yaml 填入密钥
.\bin\flowagent.exe config check   # 检查是否填好
```

详细步骤：[docs/API_KEYS_SETUP.md](docs/API_KEYS_SETUP.md)（**DashScope 已并入阿里云百炼**）  
替代方案：[docs/PROVIDER_ALTERNATIVES.md](docs/PROVIDER_ALTERNATIVES.md)

## 状态

- [x] 工作流 YAML 加载、阶段机、产物落盘、CLI
- [x] 阶段 B：DeepSeek Plan + Write（SSE 分场景、断点续写、字数门禁）
- [x] 阶段 C：SQLite FTS Vault + 百炼 Continuity（critical 回退重写、角色 patch）
- [x] 阶段 D：百炼 Storyboard + Validate + duration_ok 门禁（4～6 镜 ai_video）
- [x] 阶段 E：Produce TTS + 万相出图 + FFmpeg Ken Burns 合成（可灵 API 待接）
- [x] 阶段 F：Compliance 违禁词扫描 + `no_block_issues` 门禁
- [x] 阶段 G：Publish 发布包 + 封面抽帧 + 人工门禁（`--auto-gate` 可跳过）
- [x] 阶段 H：真实 cost-ledger（用量记账 + `flowagent cost`）
- [x] 阶段 I：`metrics set` + Learn 归档 + Planner 引用上集数据
- [x] 阶段 J：hooks + condition 门禁 + CI + 三连跑验收脚本
