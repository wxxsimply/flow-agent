# 方案 B：整片动效镜头（video-native-short）

**每一镜**都产出可灵 **mp4 动效片段**（默认 **图生视频**：万相关键帧 → 可灵 `image2video`），FFmpeg **只拼接 mp4**，**禁止 Ken Burns / 静图幻灯片**。

工作流 `novel-short-douyin` 已默认 `stack_profile: video-native-short`。

## 启用

1. 配置可灵 AK/SK（`config/providers.local.yaml`，`base_url` 建议 `https://api.klingai.com` 若国际站 Key）  
2. 验证：`flowagent config test-kling`  
3. 运行：

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent
.\bin\flowagent.exe run novel-short-douyin --series demo --episode 1 --auto-gate -v
```

从分镜重跑：

```powershell
.\bin\flowagent.exe resume --run-id <id> --from-stage storyboard --stack video-native-short --auto-gate -v
```

## 与 standard-tier 的区别

| 项目 | standard-tier | video-native-short |
|------|---------------|-------------------|
| 画面 | 万相静图 + Ken Burns | 可灵 **text2video** 全镜 |
| 分镜 | 10～18 镜，全 ken_burns | 9～15 镜，全 ai_video |
| 可灵 | 关闭 | **必须**配置 |
| 失败回退 | N/A | 默认 **图生视频**（需万相出图）；可关 `fallback_image2video` |

## 配置项（`config/stacks/video-native-short.yaml`）

- `video.strategy`: `text2video` | `image2video`  
- `video.skip_image`: `true` 时默认不先出图（仅 t2v 或回退时出图）  
- `video.require_video`: `true` 时任一镜无 mp4 则 Produce 失败（禁止 Ken Burns）  

## 可灵模型名（重要）

文生视频与图生视频 **model_name 不同**：

| API | 推荐 model_name | 勿用 |
|-----|-----------------|------|
| `text2video` | **`kling-v1-6`**（国际站常见）、`kling-v1-5`、`kling-v2-6` | `kling-v2-1`、`kling-v2-5`（多数账号 1201 invalid） |
| `image2video` | `kling-v2-1` | — |

探测本账号可用型号：

```powershell
.\bin\flowagent.exe config test-kling-text
```

`video-native-short` 默认 `text_model: kling-v1-6`；Produce 时若 model 被拒会自动按列表回退。

## 限制（预期管理）

- 可灵单片段时长一般为 **5s 或 10s**，90s 约需 **9～15 次** API，单集 Produce 可能 **30～90 分钟**。  
- 跨镜人物一致性仍弱于专业剧组；可在 `visual_prompt` 中重复角色外观描述。  
- **口型与 TTS 不对齐**；对口型需另接数字人/唇形 API（未实现）。  

## 成本粗算

`unit_prices_cny.video_per_second × 90` ≈ **¥25+**（仅视频），见 stack `cost_targets_cny.video`。
