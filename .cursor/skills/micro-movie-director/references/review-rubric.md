# 分镜审查量表（Review Rubric）

**[审查]** `ReviewStoryboard()` 规则层 + `ApplyLLMStoryboardReview()` LLM 层。

## 审查顺序（animation-craft）

1. 故事 — s01 锚定、剧情推进、旁白服务叙事
2. Staging — 走位、前中后景、道具逻辑
3. 节奏 — 景别变化、10 秒单镜一主动作
4. 表演 — action_beats 四拍、微表情具体
5. 连续性 — 锁定词、屏幕方向、光线
6. 物理可信度 — physics_cues / forbidden_physics
7. 画面精致度 — 光影材质、抗伪影
8. 字幕/旁白 — 不重复、长度 12–45 字

## 维度 1：故事（Story）

| 级别 | 标准 |
|------|------|
| error | s01 偏离用户第一镜核心画面 |
| error | narration 与 visual 矛盾 |
| warn | 镜间剧情跳跃无过渡 |

## 维度 2：Staging

| warn | 连续 3 镜同一景别且无理由 |
| warn | 道具凭空出现/消失 |

## 维度 3：节奏（Timing）

| warn | action_beats < 3 |
| error | 单镜多主动作描述 |

## 维度 4：表演（Performance）

| warn | action_beats 含空泛情绪词 |
| warn | 无微表情/眼神描述 |

## 维度 5：连续性（Continuity）

| error | 锁定词跨镜不一致 |
| warn | 跳轴无说明 |

## 维度 6：物理（Physics）

| error | physics_cues 为空（规则层自动补） |
| warn | forbidden_physics 为空 |
| warn | 违反重力/支撑描述 |

## 维度 7：画面精致度（Polish）

| warn | visual_prompt < 80 字或缺光影/材质 |
| warn | 仅「电影感」「精致」等空词 |

## 维度 8：旁白/字幕（Narration）

| error | narration 为空 |
| warn | 与其他镜 narration 重复 |

## 自动修复（规则层）

- 补全 action_beats（3 条）
- 补全 physics_cues / forbidden_physics 默认值
- DedupeNarrations

## LLM 审查输出（patches）

仅允许修改字段：`visual_prompt`, `narration`, `physics_cues`, `forbidden_physics`, `action_beats`

## 参考

- animation-craft — `references/review-rubric.md`, animation-review-template
