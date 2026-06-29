# 阶段 K2 动画：可灵配置与验收

本文说明如何启用 **K2 图生视频**（`ai_video_budget` 镜头生成真实 mp4，而非 Ken Burns 静图）。

## 1. 申请密钥（必须手动）

1. 打开 [可灵开放平台](https://klingai.com/global/dev)（国内站也可）
2. 创建 **Access Key** 与 **Secret Key**（一对密钥，不是网页登录密码，也不是单个 `api_key`）
3. 确认账户有图生视频额度（控制台可见余额/资源包）

## 2. 写入配置

编辑 `config/providers.local.yaml`：

```yaml
kling:
  access_key: "你的 Access Key"
  secret_key: "你的 Secret Key"
  base_url: "https://api-beijing.klingai.com"   # 国内默认
  # 若 test-kling 仅国际域名通过，改为:
  # base_url: "https://api.klingai.com"
```

也可用环境变量（优先级更高）：

```powershell
$env:KLING_ACCESS_KEY = "..."
$env:KLING_SECRET_KEY = "..."
```

## 3. 一键验证（推荐）

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
.\bin\flowagent.exe config test-kling
```

成功示例：

```text
[x] https://api-beijing.klingai.com
    认证成功
    余额(积分): 120.00

建议 base_url: https://api-beijing.klingai.com
```

若出现 `code=1002 access key not found`：

- 检查 **Access Key / Secret Key 是否填反**
- 确认密钥来自开放平台「API 密钥」，不是别的代理商 Key
- 运行 `test-kling` 看 **国际域名** `api.klingai.com` 是否通过，通过则改 `base_url`

## 4. 跑流水线（K2 自动生效）

无需改工作流；`produce` 阶段会对 `ai_video_budget=true` 的镜头调用可灵：

```powershell
.\scripts\accept-series-e2e.ps1 -LiveRun -AutoGate
```

日志中应出现：

```text
level=INFO msg="kling clips ready" count=4 model=kling-v2-5-turbo
```

不应再整集都是 `kling skipped, ken_burns fallback`（除非余额不足或单镜失败）。

## 5. 验收清单

| 检查项 | 路径 / 现象 |
|--------|-------------|
| 可灵认证 | `flowagent config test-kling` 通过 |
| 成片镜头 | `artifacts/assets/s01.mp4` 等 ≥4 个，可播放且有运动 |
| 时间轴 | `timeline.json` 中对应镜 `visual_type: ai_video` |
| 成本 | `flowagent cost` 中 `video_cny` > 0 |

## 6. 模型说明

`config/stacks/standard-tier.yaml` 中 `video.model: kling-v2-5-turbo` 会自动映射为可灵官方 API 名 **`kling-v2-1`**（图生视频稳定型号）。如需专业模式可在 stack 中设 `mode: pro`。

## 7. 其它阶段依赖（K1/K3）

| 项 | 说明 |
|----|------|
| **百炼 DashScope** | 按镜 TTS + 万相出图 |
| **FFmpeg** | `ffmpeg/bin` 或系统 PATH |

## 8. 可选 K4 BGM

```yaml
# config/stacks/standard-tier.yaml
compose:
  bgm_enabled: true
  bgm_path: assets/bgm/default.mp3
```

将 180s 可循环 mp3 放到 `assets/bgm/default.mp3`。

## 成本提示

- 可灵按镜 × 时长计费，标准档建议 4～6 镜/集（与分镜 `ai_video_budget` 一致）
- 单镜失败会自动重试 1 次，仍失败则该镜回退 Ken Burns，不阻断整集
