# 阶段 F 完成总结（Compliance 合规扫描）

> 汇总文档 · 对应路线图 [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §7（W5 Compliance）  
> 技术细节与逐步验证见 [PHASE_F_COMPLETED.md](PHASE_F_COMPLETED.md)

**完成时间**：2026-05  
**前置阶段**：E（Produce 已产出 `master.mp4` 等）  
**下一阶段**：G（Publish 发布包与封面）

---

## 1. 阶段目标回顾

在内容发布前，用**本地词库**扫描剧本、分镜与（可选）发布文案，生成 `compliance-report.json`；若存在 **block** 级违禁词，工作流在 **comply** 阶段失败，避免带病进入 publish。

---

## 2. 任务完成清单

| 编号 | 路线图任务 | 完成状态 | 说明 |
|------|------------|----------|------|
| **F1** | 加载 `config/compliance/words.txt` + 平台词库 | ✅ 已完成 | 合并加载 `words.txt` 与 `platform-douyin.txt`，支持 `#` 注释、`block:` / `warn:` 前缀 |
| **F2** | 扫描 chapter、storyboard 字幕、publish 标题描述等 | ✅ 已完成 | 见下文「扫描范围」；publish-pack 仅在产物已存在时扫描 |
| **F3** | 输出 `compliance-report.json`（`blocked` / `warnings`） | ✅ 已完成 | schema `compliance_report_v1`；含 `blocks`、`warnings`、计数与 `checked_at` |
| **F4** | 含 block 级词时 `no_block_issues` 失败 | ✅ 已完成 | `internal/runner/gates.go` 读取报告真实校验，不再仅检查文件是否存在 |
| **F5** | DeepSeek 二次抽检（可选） | ⏸ 未实现 | 当前以词库规则为准；断网/无 Key 不影响 comply |

---

## 3. 实现摘要

### 3.1 处理流程

```text
comply 阶段启动
  → 从 config/compliance/ 加载词库
  → 收集 run 内文本源（chapter / brief / ssml / storyboard / publish-pack）
  → 子串匹配（中文原文；ASCII 不区分大小写）
  → 写入 artifacts/compliance-report.json
  → 门禁 no_block_issues：blocked == false 才通过
```

### 3.2 扫描范围（F2）

| 产物 | 报告中的 `source` 字段 |
|------|------------------------|
| `artifacts/chapter.md` | `chapter.md` |
| `artifacts/episode-brief.md` | `episode-brief.md` |
| `artifacts/narration.ssml` | `narration.ssml` |
| `artifacts/storyboard.json` 各镜 | `storyboard.shots[N].subtitle` / `.narration` |
| `artifacts/publish-pack.json`（若存在） | `publish-pack.title` / `.description` / `.hashtags[N]` |

> **说明**：默认工作流中 comply 在 publish **之前**，首次跑到 comply 时通常还没有 `publish-pack.json`；若需测发布文案，需先跑过 publish 或手动放置该文件。

### 3.3 词库与严重级别（F1）

| 文件 | 用途 |
|------|------|
| `config/compliance/words.txt` | 自定义黑名单（示例含 `block:赌博`、`warn:香烟`） |
| `config/compliance/platform-douyin.txt` | 抖音平台敏感词骨架（毒品、枪支等） |

- **`block:`**（或无前缀）：写入 `blocks`，`blocked=true`，触发门禁失败  
- **`warn:`**：仅写入 `warnings`，**不**阻断工作流  

### 3.4 门禁行为（F4）

- 门禁 ID：`no_block_issues`（YAML 条件：`compliance-report.json.blocked == false`）
- 有 block 命中：comply 阶段报错退出，`exit≠0`（与是否加 `--auto-gate` 无关）
- 仅有 warning：可通过 comply

---

## 4. 新增与修改的文件

| 类型 | 路径 |
|------|------|
| Agent | `internal/agent/compliance.go`（替换原骨架占位） |
| 词库加载 | `internal/compliance/wordlist.go` |
| 扫描器 | `internal/compliance/scanner.go` |
| 报告契约 | `pkg/artifacts/compliance.go` |
| 门禁 | `internal/runner/gates.go` — `checkNoBlockIssues()` |
| 词库配置 | `config/compliance/words.txt`、`config/compliance/platform-douyin.txt` |
| 单元测试 | `pkg/artifacts/compliance_test.go`、`internal/compliance/scanner_test.go`、`internal/runner/gates_test.go` |
| 阶段说明 | `docs/PHASE_F_COMPLETED.md` |
| 项目状态 | `README.md`（阶段 F 勾选为已完成） |

---

## 5. 产物契约：`compliance-report.json`

**干净通过示例**：

```json
{
  "episode_no": 1,
  "blocked": false,
  "block_count": 0,
  "warning_count": 0,
  "warnings": [],
  "checked_at": "2026-05-20T12:00:00Z"
}
```

**拦截示例**（block 词「赌博」出现在 chapter）：

```json
{
  "episode_no": 1,
  "blocked": true,
  "block_count": 1,
  "warning_count": 0,
  "blocks": [
    {
      "severity": "block",
      "word": "赌博",
      "source": "chapter.md",
      "snippet": "…去了赌博网站…"
    }
  ],
  "warnings": [],
  "checked_at": "2026-05-20T12:00:00Z"
}
```

---

## 6. 本地验证结论

| 验证项 | 结果 |
|--------|------|
| `go test ./...` | 通过（含 compliance / gates / artifacts 相关测试） |
| 干净内容 `resume --from-stage comply` | 通过，`blocked: false` |
| chapter 临时加入 block 词「赌博」 | `gate no_block_issues` 失败，`exit=1` |
| 仅加入 warn 词「香烟」 | 写入 `warnings`，工作流仍可通过 |

**注意**：PowerShell 测试前需设置完整路径变量，例如：

```powershell
$runId = "<你的 run_id>"
$runDir = Join-Path (Get-Location) "runs\$runId"
$chapter = Join-Path $runDir "artifacts\chapter.md"
```

未设置 `$runDir` 时，`"$runDir\artifacts\..."` 会解析到 `D:\artifacts\...` 导致找不到文件。

---

## 7. 常用命令（速查）

```powershell
cd D:\Code\flow-agent
go build -o bin\flowagent.exe .\cmd\flowagent

$runId = "<run_id>"
.\bin\flowagent.exe resume --run-id $runId --from-stage comply --auto-gate -v

Get-Content -Encoding UTF8 (Join-Path (Get-Location) "runs\$runId\artifacts\compliance-report.json")
```

逐项验证步骤见 [PHASE_F_COMPLETED.md](PHASE_F_COMPLETED.md) §4。

---

## 8. 已知限制

| 项 | 说明 |
|----|------|
| F5 LLM 抽检 | 未接入；敏感判断完全依赖词库维护 |
| 匹配算法 | 简单子串匹配，无分词/变形/谐音 |
| publish-pack | 默认 comply 时往往尚未生成，需单独场景验证 |
| dry-run | comply 仍会扫描磁盘上已有产物（与 continuity 类似） |

---

## 9. 与路线图 / 工作流的关系

- 工作流阶段：`produce` → **`comply`** → `publish` → `learn`（`docs/workflows/novel-short-douyin.yaml`）
- 标准档 stack 中 `compliance` 仍配置为 deepseek（`config/stacks/standard-tier.yaml`），**当前实现未调用**，仅保留配置位供后续 F5 扩展

---

## 10. 下一阶段（G）预览

| 项 | 内容 |
|----|------|
| G1 | Publisher 根据 storyboard / 系列风格生成标题、描述、话题 |
| G2 | 从 `master.mp4` 抽封面 → `cover.jpg` |
| G3 | 人工门禁 `final_cut_approved` / `publish_authorized` |

---

## 11. 相关文档

| 文档 | 用途 |
|------|------|
| [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) | 全项目阶段顺序 |
| [PHASE_F_COMPLETED.md](PHASE_F_COMPLETED.md) | F 阶段验证步骤与词库格式 |
| [PHASE_E_COMPLETED.md](PHASE_E_COMPLETED.md) | 上一阶段 Produce |
| [README.md](../README.md) | 项目状态总览 |
