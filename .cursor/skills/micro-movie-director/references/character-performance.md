# 人物表演规范（Character Performance）

**[扩写]** 体态、眼神、呼吸、节奏；**[分镜]** action_beats、micro_expression；**[i2v]** 小幅连贯动作。

## 1. 表演四拍（Anticipation → Action → Reaction → Settle）

压缩到 3 条 `action_beats` 时映射为：

| 四拍 | action_beats 槽位 | 写法 |
|------|-------------------|------|
| 预备 Anticipation | 第 1 条 | 重心后移、视线锁定、吸气、肩放松 |
| 动作 Action | 第 2 条 | 单一主动作：伸手、转身、一步、开口 |
| 反应+收势 Reaction/Settle | 第 3 条 | 情绪回落、视线转移、恢复站姿 |

**禁止** 一镜内多个无关主动作（挥拳又转身又跳跃）。

## 2. 微表情（Micro-expression）

**[分镜]** `micro_expression` 字段或并入 visual_prompt：

- 眼神：瞳孔方向、眨眼频率、眼眶湿润（克制使用）
- 眉：上扬/蹙眉/单侧挑眉
- 口：抿唇、嘴角微动、咬牙
- 呼吸：胸口起伏、叹息、屏息

用**可见描述**，不用「悲伤」「愤怒」单独出现。

## 3. 体态与重心

- 站立：双脚间距、是否承重腿、骨盆微倾
- 行走：步幅、摆臂、是否跛行/疲惫
- 坐姿：前倾倾听 vs 后仰防御
- **[i2v]** 重心转移须连续，禁止瞬移站姿

## 4. 与环境互动

- 触摸：手指接触玻璃留印、雨滴滴在手背
- 握持：杯子重量感、手指扣紧杯柄
- 躲避：侧身让行人、伞沿挡雨
- **physics_cues** 须写接触点与反作用力

## 5. 对白与表演同步

- narration 说话镜：嘴型微动、喉结、手势与重音
- 沉默镜：表演代替台词，narration 宜短
- 禁止 narration 描述与 visual_prompt 动作矛盾

## 6. 群演与配角

- 配角有明确位置与 1 个动作，非模糊人影墙
- 景深：配角可虚化但轮廓稳定，防 AI 人脸漂移

## Checklist

- [ ] action_beats 三条可画、无抽象情绪词
- [ ] 单镜单一主动作
- [ ] 微表情与 narration 情绪一致
- [ ] 与环境有至少一处接触或反应

## 参考

- [animation-craft](https://github.com/khanhhuyenngo985-sys/animation-craft) — character performance, pose, gaze, breath
