# 阶段 G 完成说明（Publish：发布包 / 封面 / 人工门禁）

对应路线图：[IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) §8。  
**完成总结**：[PHASE_G_SUMMARY.md](PHASE_G_SUMMARY.md)

---

## 1. 阶段目标

| 编号 | 目标 | 状态 |
|------|------|------|
| G1 | 根据 storyboard / brief 生成标题、描述、话题 | 已实现（DeepSeek + 模板回退） |
| G2 | 从 master.mp4 抽封面 → cover.jpg | 已实现 |
| G3 | 无 `--auto-gate` 时人工门禁交互 | 已实现 |
| G4 | 抖音 OpenAPI 草稿 | 未做（可选） |
| G5 | 发布包路径相对 run_dir | 已实现 |

---

## 2. 主要文件

| 模块 | 路径 |
|------|------|
| Publisher | `internal/agent/publisher.go` |
| Prompts | `internal/agent/prompts/publisher.go` |
| 封面 | `internal/compose/ffmpeg/cover.go` |
| 人工门禁 | `internal/runner/human_gate.go` |
| 契约 | `pkg/artifacts/publish_pack.go` |

---

## 3. 如何验证

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
go test ./...

# 开发模式（跳过人工门禁）
.\bin\flowagent.exe resume --run-id <run_id> --from-stage publish --auto-gate -v

# 人工门禁（会提示输入 y）
.\bin\flowagent.exe resume --run-id <run_id> --from-stage publish -v
```

检查：

```powershell
Get-Content -Encoding UTF8 runs\<run_id>\artifacts\publish-pack.json
Test-Path runs\<run_id>\artifacts\cover.jpg   # 需真实 master.mp4
```

---

## 4. 下一阶段（H）

- 真实成本记账 `cost-ledger.json`
