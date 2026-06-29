# PhyT2V 正向物理规则（physics_cues 写法库）

来源：[PhyT2V CVPR 2025](https://github.com/pittisl/PhyT2V) — 用 LLM 从用户 prompt **提取物体 + 应遵守的物理规则**，再写入细化 prompt。flow-agent 在分镜阶段把规则写入 `physics_cues`，在 i2v 阶段与 motion 拼接。

## 七类核心规则（每镜至少覆盖 2 类）

| 类别 | 英文 | physics_cues 必写要素 | 示例 |
|------|------|----------------------|------|
| 重力与支撑 | gravity, support | 重力方向；足底/臀/背/手与支撑面接触 | 重力向下，双足踩实湿地面，鞋底有轻微压痕 |
| 运动与惯性 | motion, inertia | 起止速度；急停时发梢/衣摆滞后 | 向右迈一小步，重心前移，袍摆滞后半拍 |
| 碰撞与接触 | collision, contact | 接触点；反作用力；轻微形变 | 手扶扶手受力，指节弯曲，木扶手无穿透 |
| 因果时序 | causality | 先因后果；动作顺序不可颠倒 | 先抬手再触玻璃，接触后才有白雾 |
| 材质与数量 | material, quantity | 刚/软/液/气态；物体数量不变 | 单手持一剑，剑刃金属反光，无第二把 |
| 流体 | fluid dynamics | 流向、液面、涟漪、扩散 | 雨竖直下落，触地溅起同心涟漪 |
| 热与光 | thermal, optics | 热致形变；光路、折射、阴影 | 呼气遇冷成白雾；侧光在鼻颊投阴影 |

## 按 VideoPhy / PhyGenBench 交互类型（选 1 条写入）

### 固-固（solid-solid）

- 滚动物体沿斜面**向下**加速，接触面有摩擦
- 推门绕**铰链**旋转，门扇不穿透门框
- 叠放物体上层压下层，下层微沉、无穿插

### 固-液（solid-fluid）

- 物体入水：先入后溅，水花向上、波纹向外
- 倒水/倒酒：液流沿容器口向下，液面**上升**
- 湿衣贴肤、发梢滴水，重力下拉

### 液-液（fluid-fluid）

- 墨/汁滴入清水：慢速扩散、浓度渐变
- 油浮于水：分层界面清晰，不瞬间混合

## 人体专项（叙事镜高频）

- **关节**：肘膝只向正确方向弯；转头时颈肩联动
- **重心**：行走时左右交替支撑；坐下先屈髋再落座
- **持物**：指节包裹杯柄/剑柄，手腕有承重微沉
- **布料**：迈步时摆幅大于静止；坐下时袍摆因重力铺开

## 时间块写法（PhysVid chunk / action_beats 对齐）

长镜拆成 2–3 个物理块，每块**一种主导现象**：

```
0–3s：脚迈出，重心前移，鞋底触地
3–7s：伸手触玻璃，指尖停于表面，呼气白雾
7–10s：收手，肩放松，仍保持站立支撑
```

## 与 forbidden_physics 成对

| 正向 physics_cues | 对应 forbidden（勿重复写正向里） |
|-------------------|----------------------------------|
| 足底踩实地面 | 穿模、悬浮、滑步 |
| 液面随倒入上升 | 液体倒流、无接触即变满 |
| 门绕铰链打开 | 门中轴错误、门穿墙 |
| 单镜内角色数量不变 | 克隆人、1 人变 2 人 |

## Checklist（分镜师）

- [ ] 列出本镜**主物体**（人/道具/流体）
- [ ] 从七类规则中选 **≥2 类** 写具体 cues
- [ ] 若有液体/碰撞/光热，补 VideoPhy 交互类型一句
- [ ] action_beats 三步与物理块时间一致
- [ ] forbidden 为 cues 的**反事实**（见 physvid-negative.md）

## 参考

- [pittisl/PhyT2V](https://github.com/pittisl/PhyT2V) — CVPR 2025, CoT + step-back prompt refinement
- [PhyGenBench](https://github.com/PhyGenBench/PhyGenBench) — 力学/光学/热学评测集
- [Hritikbansal/videophy](https://github.com/Hritikbansal/videophy) — 固-固/固-液/液-液 commonsense
