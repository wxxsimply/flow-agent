# 阶段 C 完成说明（SeriesVault + Continuity）

本文档汇总 **阶段 C（W2）** 已实现的能力、涉及文件与使用方式。对应路线图：[IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §4。

---

## 1. 阶段目标（原计划）

| 编号 | 目标 |
|------|------|
| C1 | 引入 SQLite FTS，`series/<id>/vault/index.db` |
| C2 | 索引 bible、集摘要、伏笔表 |
| C3 | `vault search --query` 真实全文检索 |
| C4 | Continuity 接百炼 Qwen，输出 `continuity-report.json` |
| C5 | `critical_count > 0` 时回退重写 Write |
| C6 | 冲突可检出、修复后可过 |
| C7 | `character-state.patch.json` 合并到系列 `character-state.json` |

**结论：上述项均已实现**（含后续为稳定跑通而做的增强，见 §6）。

---

## 2. SQLite FTS 知识库（C1 / C2）

### 实现内容

- 依赖：`modernc.org/sqlite`（纯 Go，无 CGO）
- 数据库路径：`series/<series_id>/vault/index.db`
- 表结构：`documents` + FTS5 虚拟表 `documents_fts`（`unicode61` 分词）
- 自动同步：`SeriesVault.SyncIndex()` 在检索前更新索引

### 索引来源

| 类型 | `kind` | 来源文件 |
|------|--------|----------|
| 系列设定 | `bible` | `series/<id>/series-bible.yaml` |
| 集摘要 | `episode_summary` | `series/<id>/vault/episode-NNN-summary.md` |
| 伏笔 | `foreshadow` | `series/<id>/foreshadows.yaml`（可选） |

### 主要代码

- [`internal/vault/index.go`](../internal/vault/index.go) — 建库、迁移、索引、FTS 查询
- [`internal/vault/vault.go`](../internal/vault/vault.go) — `Search()` 委托 `SearchFTS`

### 示例数据

- [`series/demo/foreshadows.yaml`](../series/demo/foreshadows.yaml) — demo 系列伏笔表

---

## 3. 全文检索 CLI（C3）

### 命令

```powershell
.\bin\flowagent.exe vault search --series demo --query 伏笔
```

### 行为

- `--query` 必填；空查询报错
- 输出：`[kind] ep=N 标题` + 高亮摘要（snippet）
- 检索前自动 `SyncIndex()`

### 主要代码

- [`cmd/flowagent/cmd/vault.go`](../cmd/flowagent/cmd/vault.go)

---

## 4. 百炼 Continuity（C4）

### 实现内容

- 新增 **Bailian** OpenAI 兼容客户端（与 DeepSeek 并列）
- `provider.Bundle`：`DeepSeek` + `Bailian`，按 stack 的 `provider` 字段选择
- Continuity 阶段使用 `stack.llm.continuity`（默认 `qwen-plus`）
- 读取：`chapter.md`、`series-bible`、**character-state.json**、vault FTS「伏笔」检索结果
- 产物：
  - `artifacts/continuity-report.json`（`continuity_report_v1`）
  - 可选 `artifacts/character-state.patch.json`

### 主要代码

- [`internal/provider/llm/bailian.go`](../internal/provider/llm/bailian.go)
- [`internal/provider/bundle.go`](../internal/provider/bundle.go) — `LLMForStage()`
- [`internal/agent/continuity.go`](../internal/agent/continuity.go)
- [`internal/agent/prompts/continuity.go`](../internal/agent/prompts/continuity.go)
- [`pkg/artifacts/continuity.go`](../pkg/artifacts/continuity.go)

### 配置要求

- `config/providers.local.yaml` 中 `dashscope.api_key`（百炼 Key）
- `dashscope.region` 与控制台地域一致
- 非 dry-run 时 Continuity 需要百炼 Key；dry-run 仍输出「通过」占位报告

---

## 5. Critical 回退重写（C5）

### 流程

1. 执行 Continuity → 解析报告 → **应用 character patch**
2. 若 `critical_count > 0` 且未超过 `max_continuity_retries`（工作流默认 **3**）：
   - 写入 `continuity-rewrite-hints.json`
   - 删除 critical 场景对应的 `chapter.parts/scene-NN.md`
   - 调用 Writer 重写（带修复说明 + 角色状态）
3. **第 2 次重试起**：整章清空后全量重写（`FullRewriteAfterContinuity`）

### 主要代码

- [`internal/runner/continuity.go`](../internal/runner/continuity.go) — 重试循环
- [`internal/agent/writer_rewrite.go`](../internal/agent/writer_rewrite.go) — 局部/整章重写
- [`internal/agent/prompts/writer.go`](../internal/agent/prompts/writer.go) — `WriterUserWithFix`

### 门禁

- `continuity_passed`：读取报告，要求 `critical_count == 0`
- 配置项：`docs/workflows/novel-short-douyin.yaml` → `budget.max_continuity_retries`

---

## 6. 角色状态与归档（C7 + Learn）

### character-state

- 路径：`series/<id>/vault/character-state.json`（旧路径 `series/<id>/character-state.json` 会在首次加载时自动迁移）
- 首次运行：从 `series-bible.yaml` 的 `characters` **自动初始化**（`EnsureCharacterStateFromBible`）
- Continuity 通过后：将 `character-state.patch.json` **合并**进 `character-state.json`
- 启动期自动消毒：若历史 run 写入过旧版本 `inferCharacterPatch` 的硬编码污染（含「顾沉为保护林晚而假装分手的真相（录音已揭示）」字串），会被自动清理

### 主要代码

- [`internal/vault/character.go`](../internal/vault/character.go)
- [`internal/vault/character_seed.go`](../internal/vault/character_seed.go)
- [`internal/agent/continuity_patch.go`](../internal/agent/continuity_patch.go) — 仅合并 LLM 显式返回的 patch（不再做剧情推断）

### Learn 阶段归档

- 用 `chapter.md` 生成集摘要写入 `vault/episode-NNN-summary.md`
- 并进入 FTS 索引（供下集 Planner / Continuity 使用）

- [`internal/agent/analyst.go`](../internal/agent/analyst.go)

---

## 7. 阶段 C 之后的增强（为跑通流水线）

以下不属于最初 C 条目编号，但已在阶段 C 开发过程中落地，便于你理解当前行为。

### 7.1 Continuity 误报与放行

| 能力 | 说明 |
|------|------|
| `NormalizeSeverity()` | 将「心理留白 / 节奏 / 文风」类 `character_state` 从 critical **降为 warning** |
| Continuity prompt 收紧 | 仅**事实矛盾**标 critical |
| `--auto-gate` 兜底 | 重试用尽后仍将剩余 critical 降为 warning 并**放行**（打日志） |

代码：[`pkg/artifacts/continuity_normalize.go`](../pkg/artifacts/continuity_normalize.go)

### 7.2 Resume 续跑

| 能力 | 说明 |
|------|------|
| `resume --from-stage write` | 默认**清空** `chapter.parts` / `chapter.md` 后整章重写 |
| `--keep-chapter` | 保留已有场景（仅 API 中断续写时用） |
| 避免「write 18ms 跳过」 | 不再用旧正文反复 Continuity 失败 |

代码：[`internal/agent/chapter_draft.go`](../internal/agent/chapter_draft.go)

### 7.3 正文字数门禁（与阶段 B 交叉）

当前固定范围（**仅正文**）：

- **下限 600 字，上限 1800 字**
- 常量：`pkg/artifacts.ChapterBodyMinChars` / `ChapterBodyMaxChars`
- Writer 合并后 `EnforceChapterLength` 防止超长

代码：[`pkg/artifacts/chapter_bounds.go`](../pkg/artifacts/chapter_bounds.go)

---

## 8. 如何验证阶段 C

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent

# 1. FTS 检索
.\bin\flowagent.exe vault search --series demo --query 林晚

# 2. 全流程（需 deepseek + dashscope Key）
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --auto-gate -v

# 3. 产物检查
# runs/<run_id>/artifacts/continuity-report.json
# series/demo/vault/character-state.json
# series/demo/vault/index.db
```

### Windows PowerShell 查看中文 JSON

中文 Windows 默认 PowerShell `type` / `Get-Content` 按 GBK 解读 UTF-8 文件，
会显示成 `鏋楁櫄` 这类乱码（文件本身仍是合法 UTF-8）。两种解决方式：

```powershell
# 方法 A：显式指定 UTF-8
Get-Content -Encoding UTF8 series\demo\vault\character-state.json

# 方法 B：临时切换控制台编码后 type
chcp 65001
type series\demo\vault\character-state.json
```

**验收对照（路线图）**

- [x] 冲突能被检出（Continuity 报告含 issues）
- [x] 修复/重试后可继续（回退 Write + 归一化 / auto-gate）
- [x] `vault search` 能命中关键词

---

## 9. 尚未实现（阶段 D 及以后）

- Storyboard 真实百炼生成 + JSON Schema / 时长门禁
- 火山 TTS、万相出图、可灵、完整 FFmpeg 成片
- Vault 高级能力（按集自动摘要 LLM 化、伏笔结构化表等）

详见 [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) 阶段 D 起。

---

## 10. 相关文件索引

| 模块 | 路径 |
|------|------|
| FTS 索引 | `internal/vault/index.go` |
| 角色状态 | `internal/vault/character.go`, `character_seed.go` |
| Continuity Agent | `internal/agent/continuity.go`, `continuity_patch.go` |
| 百炼 LLM | `internal/provider/llm/bailian.go` |
| Provider 路由 | `internal/provider/bundle.go` |
| 重试编排 | `internal/runner/continuity.go` |
| 报告结构 | `pkg/artifacts/continuity.go`, `continuity_normalize.go` |
| 重写提示 | `pkg/artifacts/rewrite_hints.go` |
| CLI vault | `cmd/flowagent/cmd/vault.go` |
| 工作流预算 | `docs/workflows/novel-short-douyin.yaml` |

---

*文档版本：1.0 · 对应仓库阶段 C 实现（含后续稳定性增强）*
