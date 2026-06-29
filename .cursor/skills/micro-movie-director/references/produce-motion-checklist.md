# Produce 阶段 i2v 短约束清单



本文件供 `MotionPromptBlock()` 读取，每条 ≤40 字，勿写长段落。正向句来自 PhyT2V/PhysVid；负向句来自 VideoPhy PC 失败模式。



## 正向（physics_cues 压缩）



- 足底贴地或明确支撑，重力向下

- 单镜单一主动作，预备-进行-收势可见

- 重心转移连续，衣摆发梢有惯性滞后

- 接触点明确：手扶/脚踩/臀坐有反作用力

- 流体竖直下落，触地溅射或液面缓升

- 因果顺序：先接触后形变后反应

- 刚体杯门遵循铰链，碰撞有轻微反馈

- 指节包裹握持物，道具全程可见不消失
- 道具固定在同一手，形状刚性不变
- 持物手位移最小，小幅运镜

## 负向（forbidden 压缩，Produce 自动追加）

- 禁止穿模与物体穿透
- 禁止无支撑悬浮与向上飘
- 禁止瞬间位移与闪烁形变
- 禁止道具换手或左手右手瞬移
- 禁止道具消失或凭空出现
- 禁止武器/道具扭曲变形
- 禁止因果颠倒与未触即动
- 禁止液体倒流与干地瞬湿
- 禁止多余肢体与面部融化
- 禁止背景呼吸式变形
- 禁止角色或道具数量突变



## 参考



- [PhyT2V](https://github.com/pittisl/PhyT2V)

- [PhysVid](https://github.com/5aurabhpathak/PhysVid)

- [WMReward BoN](https://github.com/facebookresearch/WMReward)


