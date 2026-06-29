# PhysVid / PAR 式 Negative Physics（forbidden_physics + i2v negative）

来源：[PhysVid CVPR 2026](https://github.com/5aurabhpathak/PhysVid)、[PAR+SDG](https://arxiv.org/abs/2509.24702) — 用**反事实 prompt** 描述「违反局部物理定律的轨迹」，与正向 `physics_cues` 成对，在 inference 时 steer 模型远离 implausible 解。

## 通用负向（每镜可组合 3–5 项）

- 人物身体穿透墙壁、桌椅、其他角色
- 物体无支撑悬浮在空中
- 角色瞬间位移超过合理一步距离（teleport）
- 重力方向错误，人物或物体向上飘浮
- 液体向上流动或倒流
- 刚体如橡胶般无碰撞、直接互穿
- 手指数量错误或手穿过杯柄/门把
- 面部五官融化或帧间剧烈形变
- 背景纹理呼吸式蠕动、边缘闪烁
- 光影与光源方向不一致、阴影缺失或跳变
- 镜头内角色/道具数量突变
- 未接触即发生形变或状态改变（因果颠倒）

## 按 PhyT2V 规则域

| 规则域 | 负向 forbidden 示例 |
|--------|---------------------|
| 重力/支撑 | 无支撑悬浮、向上飘、坐空椅、脚踩空 |
| 运动/惯性 | 滑步、瞬移、运动方向突变无过渡 |
| 碰撞/接触 | 穿模、肢体融合、无碰撞反馈 |
| 因果 | 先开后触、未倒水液面已满 |
| 材质/数量 | 克隆人、双剑、道具凭空出现 |
| 流体 | 雨横飞、水倒流、无涟漪 |
| 热/光 | 烟雾向下、光斑随机跳、热雾无呼气 |

## 按场景类型

### 行走/奔跑

- 脚不触地滑步
- 转身时躯干扭转超过关节极限
- 影子与脚步不同步

### 持物/交互

- 手穿过杯身、门把手、对方身体
- 放下物体时无轻微惯性滞后
- 握持时手指穿透物体表面

### 雨/水

- 雨滴水平飞行
- 干衣在雨中无湿痕突变
- 水面无接触涟漪、物体浮空于水面

### 风/布料

- 衣摆与头发飘向不一致
- 布料像金属板硬折、无惯性滞后
- 静止无风时衣摆大幅乱舞

### 光/烟/火

- 烟向下沉、火焰静止无摇曳
- 霓虹反光与光源方向矛盾
- 呼气无白雾却出现热雾（或相反）

## 正向对照表（写入 physics_cues，勿与 forbidden 重复）

| 负向避免 | 正向 physics_cues |
|----------|-------------------|
| 穿模 | 指尖停在玻璃表面，无穿透 |
| 悬浮 | 双脚踩在湿沥青上，有压痕 |
| 瞬移 | 向右迈一小步，重心前移 |
| 液体倒流 | 液流沿杯口向下，液面缓升 |
| 因果颠倒 | 先触门把再推门，接触后运动 |
| 数量突变 | 镜头内始终仅一名主角 |

## i2v 用法

- **分镜**：`forbidden_physics` 从本清单选 3–5 条 + 场景定制，必须与 `physics_cues` 对立
- **Produce**：`MotionPromptBlock()` 自动抽取短句追加到 motion negative
- **BoN**：WMReward 选 surprise 最低候选时，优先无本清单项的 clip

## 参考

- [PhysVid](https://github.com/5aurabhpathak/PhysVid) — negative physics prompts + chunk CFG
- [PAR + SDG](https://arxiv.org/abs/2509.24702) — counterfactual + synchronized decoupled guidance
- [WMReward](https://github.com/facebookresearch/WMReward) — 物理 surprise 选优
