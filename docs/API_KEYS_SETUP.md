# API Key 配置指南

按本指南完成 **阶段 A2**，再进入 [`IMPLEMENTATION_ROADMAP.md`](IMPLEMENTATION_ROADMAP.md) 阶段 B。

---

## 重要：DashScope 去哪了？

**没有消失。** 阿里云已将「灵积 DashScope」产品能力 **并入「大模型服务平台 · 百炼（Model Studio）」**：

| 以前叫法 | 现在入口 | API 是否还能用 |
|----------|----------|----------------|
| DashScope / 灵积 | **[百炼控制台](https://bailian.console.aliyun.com/)** | ✅ 按 Key **地域** 选不同 `dashscope*.aliyuncs.com` 端点 |
| 通义千问 API | 百炼 → 模型广场 → Qwen | ✅ OpenAI 兼容：`/compatible-mode/v1` |
| 通义万相 | 百炼 → 模型广场 → 万相 | ✅ 文生图 API（如 `wan2.6-t2i`） |

本项目配置里仍用字段名 **`dashscope`**（兼容旧文档），填的是 **百炼控制台创建的 API Key**（格式通常也是 `sk-...`）。

**不要再去** 已跳转或难找的 `dashscope.aliyun.com` 独立站，统一用百炼：

1. 打开 https://bailian.console.aliyun.com/
2. 开通百炼服务（按提示完成实名/开通模型）
3. 右上角选地域（国内一般 **华北2 北京**；国际账号常见 **新加坡**）
4. 左侧 **API Key** → 创建 → 复制 `sk-...`（**Key 与地域绑定，不能混用端点**）
5. 填入 `config/providers.local.yaml` 的 `dashscope.api_key` 与 `region`
6. 运行 `flowagent config test-api` 自动探测可用端点

官方说明：[如何获取 API Key（百炼）](https://help.aliyun.com/zh/model-studio/user-guide/apikey)

---

## 1. 一键初始化配置文件

```powershell
cd D:\Code\flow-agent
go build -o bin\flowagent.exe .\cmd\flowagent

.\bin\flowagent.exe config init
```

---

## 2. 编辑 `config/providers.local.yaml`

| 配置项 | 实际服务 | 用途 | 何时必需 | 申请入口 |
|--------|----------|------|----------|----------|
| `deepseek` | DeepSeek | 写作、规划 | ✅ 现在 | https://platform.deepseek.com/ |
| `dashscope` | **阿里云百炼** | Qwen 校验/分镜、万相出图 | 阶段 C/E | https://bailian.console.aliyun.com/ |
| `volcengine` | 火山引擎 | 豆包语音 TTS | 阶段 E | https://console.volcengine.com/ |
| `kling` | 可灵 | 图生视频 | 可选 | https://klingai.com/ |

### DeepSeek

```yaml
deepseek:
  api_key: "sk-xxxxxxxx"
  base_url: https://api.deepseek.com
```

### 阿里云百炼（原 DashScope，配置项仍叫 dashscope）

```yaml
dashscope:
  api_key: "sk-xxxxxxxx"
  region: cn-beijing   # 与控制台 Key 地域一致：cn-beijing | intl | us | hk
  base_url: ""         # 留空则按 region 自动选择官方端点
```

| region | OpenAI 兼容 base_url |
|--------|----------------------|
| `cn-beijing` | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| `intl` | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` |
| `us` | `https://dashscope-us.aliyuncs.com/compatible-mode/v1` |
| `hk` | `https://cn-hongkong.dashscope.aliyuncs.com/compatible-mode/v1` |

若以前写的北京地址 `https://dashscope.aliyuncs.com/compatible-mode/v1` 返回 401/403，多半是 **Key 属于国际/新加坡地域**，请改 `region: intl` 或运行 `flowagent config test-api`。

**百炼 Key 可调用：**

- **Qwen**（`qwen-plus` / `qwen-max`）→ Continuity、Storyboard 阶段  
- **万相**（`wan2.6-t2i` 等）→ 文生图，见 [万相文生图 API](https://help.aliyun.com/zh/model-studio/text-to-image-v2-api-reference)

### 火山引擎（语音）

豆包语音 OpenSpeech 需要 **AppID + Access Token**（不是方舟 `ark-` Key）：

```yaml
volcengine:
  access_key: "语音应用 Access Token"   # 非 ark- 开头
  secret_key: ""                        # 本项目 TTS 未使用，可留空
  app_id: "1234567890"                 # 语音控制台应用 AppID
```

若未开通语音或只有 `ark-` Key，请 **留空** `access_key` / `app_id`，produce 将自动用百炼 CosyVoice。

**万相 wan2.6-t2i**：须保证 `dashscope.region` 与百炼控制台 Key 地域一致（北京 `cn-beijing` / 国际 `intl`）。

### 可灵（可选）

```yaml
kling:
  api_key: "xxxxxxxx"
```

---

## 3. 若不想用百炼：替代方案

详见 **[PROVIDER_ALTERNATIVES.md](PROVIDER_ALTERNATIVES.md)**。摘要：

| 原能力 | 推荐替代 A（省事） | 推荐替代 B |
|--------|-------------------|------------|
| Qwen 校验/分镜 | **继续用百炼**（只是换控制台） | **DeepSeek** 同一 Key 兼任（改 `standard-tier.yaml`） |
| 万相出图 | 百炼万相 `wan2.6-t2i` | **火山即梦** / **可灵**（改 image provider） |
| 全流程国产最少账号 | 百炼 + 火山 + DeepSeek | DeepSeek + 火山（无百炼） |

个人开发者 **仍建议百炼**：一个 `sk-` 同时覆盖 Qwen + 万相，与项目标准档一致。

---

## 4. 环境变量（可选）

```powershell
$env:DEEPSEEK_API_KEY = "sk-..."
$env:DASHSCOPE_API_KEY = "sk-..."
$env:DASHSCOPE_REGION = "intl"      # 可选：cn-beijing | intl | us | hk
$env:VOLCENGINE_ACCESS_KEY = "AK..."
$env:VOLCENGINE_SECRET_KEY = "..."
$env:KLING_API_KEY = "..."
```

---

## 5. 检查配置

```powershell
.\bin\flowagent.exe config check
.\bin\flowagent.exe config test-api   # 探测百炼各地域端点（需已填 dashscope.api_key）
```

至少 **deepseek** 为 `[x]` 才能进入阶段 B；阶段 C 起需要 **dashscope（百炼）** 为 `[x]`。`test-api` 会标出当前 Key 可用的 `region`。

---

## 6. 安全提醒

- `providers.local.yaml` 已 gitignore，勿提交  
- Key 泄露后立即在百炼/DeepSeek 控制台删除并重建  

---

## 7. 下一步

| 已完成 | 下一步 |
|--------|--------|
| `config check` → deepseek `[x]` | [阶段 B 真实写作](IMPLEMENTATION_ROADMAP.md#3-阶段-bw1--真实-plan--writedeepseek) |

---

*文档版本：1.1 · dashscope 说明更新为阿里云百炼*
