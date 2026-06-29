# 阶段 F 完成说明（Compliance：违禁词扫描）

对应路线图：[IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §7。  
**完成总结**（任务清单、文件列表、验证结论）：[PHASE_F_SUMMARY.md](PHASE_F_SUMMARY.md)

---

## 1. 阶段目标

| 编号 | 目标 | 状态 |
|------|------|------|
| F1 | 加载 `config/compliance/words.txt` + 平台词库 | 已实现 |
| F2 | 扫描 chapter、storyboard 字幕/旁白、episode-brief、publish-pack（若存在） | 已实现 |
| F3 | `compliance-report.json`：`blocked` / `blocks` / `warnings` | 已实现 |
| F4 | 含 block 级词时 `no_block_issues` 门禁失败 | 已实现 |
| F5 | DeepSeek 二次抽检 | 未做（可选，词库为主） |

---

## 2. 主要文件

| 模块 | 路径 |
|------|------|
| Compliance Agent | `internal/agent/compliance.go` |
| 词库加载 | `internal/compliance/wordlist.go` |
| 扫描器 | `internal/compliance/scanner.go` |
| 报告契约 | `pkg/artifacts/compliance.go` |
| 门禁 | `internal/runner/gates.go` — `checkNoBlockIssues()` |
| 自定义词库 | `config/compliance/words.txt` |
| 抖音平台词库 | `config/compliance/platform-douyin.txt` |

---

## 3. 词库格式

```text
# 注释行以 # 开头
block:赌博          # block 可省略，默认即 block
warn:香烟           # 仅写入 warnings，不触发 blocked
```

---

## 4. 如何验证

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
go test ./...

# 正常 comply（应通过）
.\bin\flowagent.exe resume --run-id <run_id> --from-stage comply --dry-run --auto-gate -v

# 查看报告
Get-Content -Encoding UTF8 runs\<run_id>\artifacts\compliance-report.json
```

**F4 拦截测试**：在 `artifacts/chapter.md` 末尾临时加入词库中的 block 词（如「赌博」），再跑 comply（不加 `--auto-gate` 也可，comply 为自动门禁）：

```powershell
.\bin\flowagent.exe resume --run-id <run_id> --from-stage comply --auto-gate -v
# 预期：gate no_block_issues 失败，exit≠0
```

---

## 5. 报告示例

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
  "checked_at": "2026-05-20T09:00:00Z"
}
```

---

## 6. 下一阶段（G）

- Publisher 根据 storyboard / 系列风格生成标题、描述、话题
- 从 `master.mp4` 抽封面帧 → `cover.jpg`
- 人工门禁 `final_cut_approved`
