# Produce 加速与提质 — Stack 使用指南

本文说明 `micro-movie-wan-*` 栈在 **assemble → produce** 阶段的耗时杠杆与推荐组合。

## Stack 对照

| Stack | 定位 | `keyframe_mode` | BoN | 并行 | 典型片长 |
|-------|------|-----------------|-----|------|----------|
| `micro-movie-wan-quick` | **5 分钟出片** | `single` | 关 | 4 | ~45s（3–4 镜） |
| `micro-movie-wan-fast` | 调试/预览 | `single` | 关 | 2 | ~90s（6–12 镜） |
| `micro-movie-wan-flash` | 默认平衡 | `multi` | 全开×3 | 1 | ~150s |
| `micro-movie-wan-premiere` | 样片提质 | `tiered` | 仅 hero | 2 | ~150s |

```bash
# 目标墙钟 ~5 分钟（取决于万相 API 延迟）
flowagent run micro-movie --plot-file ./plot.md --stack micro-movie-wan-quick --target-duration-sec 45

flowagent run micro-movie --plot-file ./plot.md --stack micro-movie-wan-fast --target-duration-sec 90
flowagent run micro-movie --plot-file ./plot.md --stack micro-movie-wan-premiere
```

## 主要配置项（`stack.video`）

| 字段 | 说明 |
|------|------|
| `keyframe_mode` | `single`（1 t2i + 1 i2v/镜）、`multi`（3 t2i + 2 i2v/镜）、`tiered`（按镜 hero/standard） |
| `inter_shot_delay_sec` | 串行 produce 时镜间等待（秒）；并行时忽略。flash 默认 1（替代原硬编码 8s） |
| `max_parallel_shots` | produce 并行度，注意百炼 QPS |
| `wmreward_bon.enabled` / `candidates` | 每段 i2v 多候选选优，耗时可 ×candidates |
| `wmreward_bon.hero_only` | tiered 时仅 hero 镜 BoN |
| `use_quality_model_for` | `hero`：hero 镜用 `quality_model`（如 `wan2.6-i2v`） |
| `hero.resolution` / `hero.bon_candidates` | hero 镜 1080P 与 BoN 候选数 |
| `clip_duration_sec` | **分镜默认单镜时长**，影响扩写镜数估算；**不再**作为 i2v API 硬顶。i2v 按 TTS 实测旁白时长生成（万相上限 10s） |

`assemble.llm_review: false` 可跳过 LLM 分镜审查（仍保留规则层 `ReviewStoryboard`）。

## 末段定格（已修复）

若曾出现「每镜最后几秒画面不动、字幕/旁白还在走」：

- 原因：旁白 8–10s，万相 i2v 仅 5s，合成时用 FFmpeg 静帧拉长。
- 现行为：i2v 按 `AudioDurationSec` 请求；合成时短视频不再静帧硬拉，不足部分用关键帧 **Ken Burns 补尾**。
- 诊断：查看 `artifacts/timeline.json` 的 `video_duration_sec` vs `audio_duration_sec`。

## 推荐组合

**开发迭代（验证分镜与 prompt）**

- Stack：`micro-movie-wan-fast`
- `--target-duration-sec 90`
- 查看 `artifacts/produce-prompts.json`、`artifacts/produce-timing.json`

**对外样片**

- Stack：`micro-movie-wan-premiere`
- tiered：多数镜 single+flash，2–3 镜 hero multi+BoN+HD

**不建议同时开启**：`multi` + `BoN×3` + 15 镜 + 高 `inter_shot_delay_sec` — 墙钟可达数小时。

## 产物

| 文件 | 内容 |
|------|------|
| `artifacts/produce-prompts.json` | 每镜 t2i/i2v prompt 与 BoN 开关 |
| `artifacts/produce-timing.json` | 每镜 wall_ms、tier、multi_keyframe、bon 配置 |
| `artifacts/produce-timing-summary.txt` | 汇总镜数与近似 BoN job 数 |

## 单镜冒烟

```bash
flowagent test-shot --stack micro-movie-wan-fast --duration 5 --out ./tmp/shot-test
```

仅 1 次 t2i + 1 次 i2v，用于确认密钥与 prompt，不跑全片。
