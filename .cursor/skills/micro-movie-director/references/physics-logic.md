# 物理逻辑规范（Physics Logic）

**[扩写]** 叙事中因果；**[分镜]** `physics_cues` / `forbidden_physics`；**[i2v]** motion + negative。

本规范整合 [PhyT2V](https://github.com/pittisl/PhyT2V)、[PhysVid](https://github.com/5aurabhpathak/PhysVid)、[VideoPhy](https://github.com/Hritikbansal/videophy) 的 prompt 设计原则。

## 1. PhyT2V 三问（写每镜前先答）

1. **主物体是谁**？（人 / 道具 / 流体）
2. **适用哪些物理规则**？（重力、碰撞、流体、因果…见 phyt2v-positive-rules.md）
3. **若违反规则会怎样**？→ 写入 `forbidden_physics`（反事实 negative）

## 2. 七类规则速查

| 类型 | physics_cues 必写 | forbidden 示例 |
|------|-------------------|----------------|
| 重力/支撑 | 重力向下 + 接触面 | 悬浮、向上飘 |
| 运动/惯性 | 起止速度、衣发滞后 | 瞬移、滑步 |
| 碰撞/接触 | 接触点、反作用力 | 穿模、无碰撞反馈 |
| 因果 | 先因后果顺序 | 未触即动 |
| 材质/数量 | 刚软液、数量锁定 | 克隆、双道具 |
| 流体 | 流向、液面、涟漪 | 倒流、无溅射 |
| 热/光 | 白雾、阴影、折射 | 光斑乱跳 |

## 3. VideoPhy 材质交互（有液体/碰撞时必写一类）

- **固-固**：滚/滑/撞/叠放 — 摩擦、铰链、反作用力
- **固-液**：倒/泼/入水/雨 — 液面变化、溅射向上、涟漪向外
- **液-液**：滴/混/分层 — 慢扩散、界面清晰

详见 `physics-realism/references/videophy-material-cues.md`。

## 4. 人体关节

- 肘膝弯曲方向正确；勿反关节
- 转头时颈肩联动；勿头身分离旋转
- 握持时指节包裹物体，非穿透

## 5. PhysVid 分块（chunk）与 action_beats

长镜拆时间块，每块**一种**物理主导现象：

1. 0–3s：脚迈出，重心前移，鞋底触地
2. 3–7s：伸手触玻璃，停顿，呼气白雾
3. 7–10s：收手，肩放松

`action_beats` 三步须与块对齐。

## 6. forbidden_physics 写法（反事实 negative）

- 描述**违反物理的轨迹**，不是空泛「不好」
- 与 `physics_cues` **成对对立**，每镜 3–5 条
- 句式库：`physvid-negative.md`、`video-generation-forbidden.md`

## 7. Produce / BoN

- i2v 短约束：`produce-motion-checklist.md` → `MotionPromptBlock()`
- 多候选：WMReward surprise 越低越好（支撑稳、无闪烁、无穿模）
- 配置：`video.wmreward_bon`

## 8. 审查与回归

- 分镜审查：SA（语义）+ PC（物理 commonsense），见 VideoPhy
- 回归：`physics-iq-checklist.md` + `scripts/eval-physics-iq.ps1`

## Checklist

- [ ] 每镜 physics_cues 含重力+支撑+（碰撞或流体或因果）
- [ ] forbidden 为 cues 的反事实，非空泛
- [ ] 有液体/碰撞时标明固-液/固-固类型
- [ ] action_beats 与物理块时间一致

## 参考

- [PhyT2V](https://github.com/pittisl/PhyT2V) — CVPR 2025, LLM CoT prompt refinement
- [PhysVid](https://github.com/5aurabhpathak/PhysVid) — CVPR 2026, negative physics prompts
- [VideoPhy](https://github.com/Hritikbansal/videophy) — physical commonsense benchmark
- [WMReward](https://github.com/facebookresearch/WMReward) — inference-time physics alignment
- [physics-IQ-benchmark](https://github.com/google-deepmind/physics-IQ-benchmark)
- [Awesome-Physics-Cognition](https://github.com/minnie-lin/Awesome-Physics-Cognition-based-Video-Generation)
