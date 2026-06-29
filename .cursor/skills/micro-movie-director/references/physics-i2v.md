# 图生视频物理（简表）



完整规范见 **[physics-logic.md](physics-logic.md)**；论文句式库见 **physics-realism** skill。



## 快速字段



- **physics_cues（正向）**：PhyT2V 七类中选 ≥2 — 重力、支撑、碰撞、因果、流体、材质、光热

- **forbidden_physics（负向）**：PhysVid 反事实，与 cues 对立，3–5 条



## 分镜 / 审查



- 正向库：`physics-realism/references/phyt2v-positive-rules.md`

- 负向库：`physics-realism/references/physvid-negative.md`

- 材质交互：`physics-realism/references/videophy-material-cues.md`



## produce



- 短约束：`produce-motion-checklist.md` → `MotionPromptBlock()`

- 多候选 BoN：`physics-realism/references/wmreward-bon.md`


