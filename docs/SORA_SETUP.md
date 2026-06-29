# OpenAI Sora 接入指南

## 前提

- **ChatGPT 订阅 ≠ Sora API**。需在 [platform.openai.com](https://platform.openai.com) 创建 API Key、绑卡，并确认组织已开通 **Sora Videos API**。
- 出图使用百炼万相 `wan2.6-t2i`（`dashscope.api_key`），不依赖火山方舟余额。
- 视频使用 OpenAI `sora-2`（`openai.api_key` 或环境变量 `OPENAI_API_KEY`）。

## 1. 配置密钥

编辑 `config/providers.local.yaml`：

```yaml
openai:
  api_key: "sk-..."   # platform.openai.com → API keys
  base_url: ""        # 默认 https://api.openai.com/v1
```

或使用环境变量（优先于文件）：

```powershell
$env:OPENAI_API_KEY = "sk-..."
```

## 2. 检查

```powershell
.\bin\flowagent.exe config check
```

未配置 OpenAI 时会附加 **Sora stack 凭证检查** 报告。

## 3. 单镜冒烟（推荐先做）

```powershell
.\bin\flowagent.exe config test-sora
# 或
.\bin\flowagent.exe test-shot --stack micro-movie-sora --duration 8 --out .\tmp\sora-test
```

成功后在输出目录应有 `shot.png` + `shot.mp4`。

## 4. 整片运行

```powershell
.\bin\flowagent.exe run micro-movie `
  --plot "王殿内，国王端坐王座，香炉青烟袅袅。" `
  --stack micro-movie-sora `
  --input-mode director `
  --target-duration 45 `
  --auto-gate -v
```

## 5. 常见错误

| 错误 | 原因 | 处理 |
|------|------|------|
| `sora stack not ready: openai.api_key` | 未填 OpenAI Key | 见上文 §1 |
| `sora create: http 403` | API 未开通或权限不足 | Platform 检查 Sora 访问权限 |
| `t2i: AccessDenied` | 百炼 Key 无万相权限 | 百炼控制台开通 wan2.6-t2i |
| produce `motion_quality_ok` fail | i2v 全失败降级 Ken Burns | 先修 test-sora，再跑整片 |

Stack 配置：[`config/stacks/micro-movie-sora.yaml`](../config/stacks/micro-movie-sora.yaml)
