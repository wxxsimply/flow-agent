---
name: micro-movie-director
description: 微电影导演手册：镜头语言、画面精致度、表演、服饰、场景细节、物理逻辑、连续性审查与万相 i2v 分镜模板。assemble/produce 时由 flow-agent 自动注入。
---

# Micro-Movie Director（索引）

项目内导演 skill，运行时由 [`internal/agent/skills`](../../internal/agent/skills) 按阶段加载 `references/`，无需全局 `npx skills add`。

## 八维覆盖

| 维度 | Reference |
|------|-----------|
| 镜头语言 | [camera-language.md](references/camera-language.md) |
| 画面精致度 | [visual-polish.md](references/visual-polish.md) |
| 人物表演 | [character-performance.md](references/character-performance.md) |
| 服饰身份 | [costume-and-identity.md](references/costume-and-identity.md) |
| 场景细节 | [staging-and-detail.md](references/staging-and-detail.md) |
| 物理逻辑 | [physics-logic.md](references/physics-logic.md) |
| 连续性 | [continuity-ledger.md](references/continuity-ledger.md) |
| 审查量表 | [review-rubric.md](references/review-rubric.md) |

## 辅助文档

| 文件 | 用途 |
|------|------|
| [ai-video-shot-template.md](references/ai-video-shot-template.md) | 分镜 JSON 字段模板 |
| [motion-principles.md](references/motion-principles.md) | 运镜与表演速查 |
| [produce-motion-checklist.md](references/produce-motion-checklist.md) | i2v 短约束（produce 非 LLM） |
| [physics-i2v.md](references/physics-i2v.md) | 物理字段简表 → 详见 physics-logic |

## 运行时阶段映射

| Stage | 注入的 references |
|-------|-------------------|
| `expand_brief_segment` | camera-language, costume-and-identity, staging-and-detail, character-performance |
| `expand_brief_continue` | continuity-ledger, character-performance |
| `generate_shots` | 上述 + visual-polish, physics-logic, ai-video-shot-template, review-rubric；**physics-realism** 注入 phyt2v-positive + physvid-negative |
| `review_storyboard` | review-rubric, continuity-ledger, physics-logic；**physics-realism** 注入 videophy-material-cues |
| `produce_motion` | produce-motion-checklist + video-generation-forbidden；**physics-realism** 正向/负向短句（MotionPromptBlock） |

产物：`artifacts/applied-skills.json`

## 何时人工打开本 skill

- 修改扩写/分镜 prompt 或 `shot_language_expand.go`
- 排查穿模、身份漂移、旁白重复
- 增补 reference 后跑 `go test ./internal/agent/skills/...`

## 外部参考

- [animation-craft](https://github.com/khanhhuyenngo985-sys/animation-craft)
- 内嵌兜底：`internal/agent/prompts/animation_craft.go`
