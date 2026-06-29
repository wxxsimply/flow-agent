# AI 视频分镜字段模板（Timed Shot Block）

与 [`ShotsFromBriefSystem`](../../../../internal/agent/prompts/shot_language_expand.go) JSON 对齐。

## 单镜 JSON 填写指南

```json
{
  "id": "s02",
  "shot_size": "medium",
  "camera_angle": "平视，缓慢推近",
  "scene_background": "雨夜霓虹巷，地面积水，左侧绿灯药房招牌",
  "character_motion": "向右行走两步后停，视线移向橱窗",
  "micro_expression": "眉微蹙，嘴角抿紧",
  "action_behavior": "收伞，抬手擦玻璃雾气",
  "narration": "他认出了倒影里的那个人。",
  "visual_prompt": "【锁定词】三十岁左右亚裔男性，短黑发微湿，深灰防水夹克…，中景，平视缓慢推近，浅景深，霓虹青橙对比，雨丝前景虚化，皮肤自然光影",
  "duration_sec": 10,
  "action_beats": [
    "预备：收伞站立，重心右脚，视线盯橱窗",
    "进行：左手擦玻璃，呼气白雾，指尖触冷玻璃",
    "收势：手放下，肩放松，仍盯倒影"
  ],
  "physics_cues": "重力向下，双足踩湿地面，手指接触玻璃无穿模，雨竖直下落",
      "held_props": "右手：匕首；左手：空",
      "forbidden_physics": "穿模，悬浮，瞬移，道具换手，道具消失，武器变形"
}
```

## visual_prompt 结构（80–180 字）

按顺序拼接：

1. **锁定词**（人物+服装，逐字固定）
2. **景别+角度+运镜**
3. **场景+光影+色调**
4. **主动作+微表情**（时间顺序）
5. **材质/景深** 一句

## action_beats 时间映射（10 秒镜）

|  beat | 时间 | 内容 |
|------|------|------|
| 1 | 0–3s | 预备姿态 |
| 2 | 3–7s | 主动作 |
| 3 | 7–10s | 收势 |

## 字段与 skill 维度对照

| 字段 | 主要 skill 文档 |
|------|-----------------|
| shot_size, camera_angle | camera-language.md |
| visual_prompt 光影材质 | visual-polish.md |
| action_beats, micro_expression | character-performance.md |
| visual_prompt 锁定词 | costume-and-identity.md |
| scene_background | staging-and-detail.md |
| physics_* | physics-logic.md |

## 禁止

- 静态海报式：「美丽的城市夜景」无动作
- 一镜多景别跳变描述
- narration 重复其他镜

## 参考

- animation-craft — `examples/ai-video-shot-template.md`
