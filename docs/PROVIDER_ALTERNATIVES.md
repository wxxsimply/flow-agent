# 模型与 API 替代方案（DashScope / 百炼）

当「找不到 DashScope」或不想用阿里云时，按本表调整。  
**首选结论：DashScope 已并入 [阿里云百炼](https://bailian.console.aliyun.com/)，不必换平台，只需换控制台申请 Key。**

---

## 1. 名称对照（避免找错站）

| 你看到的旧名 | 现在叫什么 | 控制台 |
|--------------|------------|--------|
| DashScope 灵积 | 百炼 Model Studio | https://bailian.console.aliyun.com/ |
| 通义千问 API | 百炼 → Qwen 系列 | 同上 |
| 通义万相 | 百炼 → 万相 wan2.x | 同上 |
| dashscope.aliyun.com | 仍可能跳转，**以百炼为准** | — |

**API 按地域分域名**（Key 必须与地域一致）：

| 地域 | OpenAI 兼容 LLM |
|------|-----------------|
| 华北2 北京 | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| 新加坡（国际） | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` |
| 美国 | `https://dashscope-us.aliyuncs.com/compatible-mode/v1` |
| 中国香港 | `https://cn-hongkong.dashscope.aliyuncs.com/compatible-mode/v1` |

项目里用 `dashscope.region` 或 `flowagent config test-api` 自动选择。

---

## 2. 本项目各阶段 ↔ 服务映射

| 阶段 | 原计划 | 百炼（推荐） | 当前默认 / 替代 |
|------|--------|--------------|-----------------|
| Plan / Write / Compliance | DeepSeek | 不变 | **`deepseek-v4-flash`**（`config/stacks/*.yaml`） |
| Continuity / Storyboard | Qwen-plus | 百炼 `qwen-plus` | **`deepseek-v4-flash`**（百炼欠费时可全走 DeepSeek Key） |
| 出图 | 万相 | 百炼 `wan2.6-t2i` | **火山 Seedream**（`micro-movie-seedance` stack + `volcengine.api_key`） |
| TTS | 火山豆包 | 不变 | 阿里 CosyVoice（百炼）、Azure |
| 图生视频 | 可灵（短剧 `video-native-short`） | **微电影默认：百炼万相 i2v** | **火山 Seedance**（`micro-movie-seedance`）；或万相 `wan2.6-i2v-flash` |

**说明**：全部 LLM 文本阶段已统一为 `deepseek-v4-flash`；**万相 t2i/i2v 仍需要** `config/providers.local.yaml` 中的 `dashscope.api_key`。

---

## 3. 方案 A：继续标准档（仅改申请入口）— 推荐

**改动最小**：只去百炼拿 Key，代码与 `dashscope` 配置项不变。

```yaml
# config/providers.local.yaml
dashscope:
  api_key: "sk-百炼控制台复制的Key"
  region: cn-beijing   # 国际控制台 Key 用 intl
  base_url: ""
```

```yaml
# config/stacks/standard-tier.yaml 建议更新出图模型名
image:
  provider: dashscope
  model: wan2.6-t2i    # 替代旧 wanx-v1
```

| 项目 | 值 |
|------|-----|
| LLM 兼容 Base URL | 见上表，或 `region` + 留空 `base_url` |
| 分镜/校验模型 | `qwen-plus` 或 `qwen-max` |
| 文生图模型 | `wan2.6-t2i`（竖屏 9:16 见官方尺寸说明） |
| 文档 | [百炼 API Key](https://help.aliyun.com/zh/model-studio/user-guide/apikey)、[万相 API](https://help.aliyun.com/zh/model-studio/text-to-image-v2-api-reference) |

---

## 4. 方案 B：去掉百炼，全用 DeepSeek + 火山

适合：已有 DeepSeek + 火山，暂时不注册阿里云。

| 能力 | 调整 |
|------|------|
| Continuity / Storyboard | `standard-tier.yaml` 中 `provider: deepseek`，`model: deepseek-v4-flash` |
| 出图 | 火山「即梦」或视觉 API（需单独实现 `provider/image/volcengine`） |
| 配置 | `dashscope` 可留空；`config check` 会显示未配置（阶段 C 前可忽略） |

**缺点**：分镜 JSON 稳定性可能弱于 Qwen；出图需新写适配器。

---

## 5. 方案 C：其他国内 LLM 平台

| 平台 | 适合替换 | 说明 |
|------|----------|------|
| **智谱 AI** | Qwen 角色 | https://open.bigmodel.cn/ OpenAI 兼容 |
| **Moonshot（Kimi）** | 长文本 Continuity | https://platform.moonshot.cn/ |
| **百度千帆** | Qwen 角色 | 文心一言 API |
| **腾讯混元** | Qwen 角色 | 混元大模型 API |

接入方式：在 `internal/provider/llm/` 增加对应客户端，或统一走 OpenAI 兼容 `base_url` + `api_key`。

---

## 6. 文生图替代（不用万相时）

| 服务 | 特点 | 申请 |
|------|------|------|
| **百炼万相 wan2.6** | 与 Qwen 同 Key，竖屏 | 百炼控制台 |
| **火山即梦** | 贴抖音审美，常网页+API | 火山引擎 |
| **可灵** | 也能出图+视频，按张/秒计费 | klingai.com |
| **SiliconFlow** | 托管 Flux 等，开发者友好 | siliconflow.cn |

项目 `standard-tier.yaml` 中 `image.alternative_provider: jimeng` 即预留即梦。

---

## 7. 微电影 / 测试期视频（万相默认）

与可灵单独 AK/SK 相比，**百炼万相图生视频**与万相出图、Qwen 分镜共用 `dashscope.api_key`：

| 模型 | 用途 | 参考单价（中国内地，以控制台为准） |
|------|------|-----------------------------------|
| `wan2.6-t2i` | 每镜关键帧 | 约 ¥0.2/张 |
| **`wan2.6-i2v-flash`** | **测试 / 默认动效** | **约 ¥0.1/秒** |
| `wan2.6-i2v` | 质量档（勿在省钱档默认开启） | 720P 约 ¥0.6/秒 |

新用户常有约 **50 秒** 图生视频免费额度。代码侧待实现：`internal/provider/video/wan.go`（规划见 [MICRO_MOVIE_AGENT_PLAN.md](./MICRO_MOVIE_AGENT_PLAN.md)）。

Stack 规划名：`config/stacks/micro-movie-wan-flash.yaml`。现网短剧仍用可灵，互不冲突。

---

## 8. 建议你怎么选

```text
还能注册阿里云？
  └─ 是 → 方案 A（百炼），一个 sk- 搞定 Qwen + 万相
  └─ 否 → 方案 B（DeepSeek 包办 LLM）+ 火山即梦/可灵出图
```

---

## 9. 相关文件（实施替代时需改）

| 文件 | 作用 |
|------|------|
| `config/providers.local.yaml` | API Key |
| `config/stacks/standard-tier.yaml` | 模型名与 provider |
| `internal/provider/llm/dashscope.go` | 待实现：百炼兼容模式客户端 |
| `internal/provider/image/` | 万相 / 即梦实现 |
| `internal/provider/video/wan.go` | 万相图生视频（微电影，待实现） |
| `config/stacks/micro-movie-wan-flash.yaml` | 微电影默认栈（待实现） |
| `docs/API_KEYS_SETUP.md` | 配置步骤 |

---

## 10. Gemini Veo 3.1 Lite（国外 i2v 备选）

适合：已有 Google AI Studio Key、需评估国外 i2v 质量（PropLock / 首尾帧衔接）时。

| 项目 | 值 |
|------|-----|
| Stack | `micro-movie-veo-lite` |
| 出图 | 仍用火山 Seedream（`volcengine.api_key`） |
| 图生视频 | `gemini.api_key` → `veo-3.1-lite-generate-preview` |
| 单镜 PoC | `flowagent test-shot --stack micro-movie-veo-lite` |
| PropLock 对比 | `flowagent compare-proplock --out ./tmp/proplock-compare` |
| 降级 | stack 内 `ken_burns_fallback: true` |

```yaml
gemini:
  api_key: "AIza..."
  base_url: ""   # 默认 generativelanguage.googleapis.com/v1beta
```

单价约 **$0.05–0.08/秒**（Lite 档）；Veo 单段时长为 **4/6/8 秒**（非 5 秒），`clip_duration_sec: 6` 较接近现有样片节奏。

---

## 11. OpenAI Sora 2（正片 i2v stack）

| 项目 | 值 |
|------|-----|
| Stack | `micro-movie-sora` |
| 出图 | Seedream `720x1280`（与 Sora `size` 一致） |
| 图生视频 | `openai.api_key` → `sora-2`（高质量 `sora-2-pro` + `1080x1920`） |
| 单镜测试 | `flowagent test-shot --stack micro-movie-sora --duration 8` |
| 正片 | `flowagent run micro-movie --plot "..." --stack micro-movie-sora --auto-gate` |
| 降级 | `ken_burns_fallback: true`（i2v 失败时仍出片） |

```yaml
openai:
  api_key: "sk-..."
```

注意：Sora **可能拒绝带人脸的 input_reference**；单镜时长为 **4/8/12 秒**；Videos API 计划 **2026-09-24 下线**。

---

*文档版本：1.0*
