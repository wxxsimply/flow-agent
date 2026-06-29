# VideoPhy 材质交互与反事实（审查 / forbidden 选用）

来源：[VideoPhy](https://github.com/Hritikbansal/videophy) — 688 条 caption 覆盖 **solid-solid / solid-fluid / fluid-fluid** 交互；人类评测 **SA（语义 adherence）+ PC（physical commonsense）** 双维。用于 `review_storyboard` 与 `forbidden_physics` 选材。

## 评测维度（审查时对照）

| 维度 | PC=1 应观察到 | PC=0 典型失败 |
|------|---------------|---------------|
| 语义 SA | 动作与 caption 一致 | 说「倒水」却未出现液流 |
| 重力 | 物体下落、跳落缓冲 | 向上飘、无落地 |
| 固-固 | 滚/滑/撞有合理反应 | 穿模、无碰撞反馈 |
| 固-液 | 液面变化、溅射方向正确 | 干容器突然满水 |
| 液-液 | 混合/分层符合直觉 | 不相溶液体瞬间均匀 |
| 时序因果 | 先接触后反应 | 未触物已移动 |

## 材质交互 — 正向 cues / 负向 forbidden

### 固-固

| 场景 | physics_cues（正向） | forbidden_physics（负向） |
|------|----------------------|---------------------------|
| 滚珠/弹珠 | 沿斜面滚下，越滚越快 | 滚上斜面、中途悬浮 |
| 推门 | 绕铰链打开，门扇不穿透框 | 门穿墙、中轴漂移 |
| 叠放 | 上层压下层，接触面稳定 | 物体互相穿透 |
| 行走 | 交替支撑，影子与步同步 | 滑步、脚不触地 |

### 固-液

| 场景 | physics_cues | forbidden_physics |
|------|--------------|-------------------|
| 倒水 | 液流向下，液面升高 | 液体倒流、无嘴出液 |
| 入水 | 先入后溅，涟漪同心扩散 | 无水花、物体浮空于水面 |
| 雨 | 竖直下落，触地微溅 | 雨横飞、干地无湿痕 |
| 洗手 | 水流过指缝向下 | 水向上爬、手穿水流 |

### 液-液

| 场景 | physics_cues | forbidden_physics |
|------|--------------|-------------------|
| 墨滴清水 | 慢扩散、浓度渐变 | 瞬间整杯变色 |
| 油水 | 分层界面，油浮上 | 瞬间乳化均匀 |
| 咖啡加奶 | 奶线向下，漩涡慢混 | 未倒奶已变色 |

## 反事实 prompt 写法（PhysVid / PAR）

`forbidden_physics` 宜描述**本镜若违反物理会是什么样**（供 i2v negative steer），而非抽象词堆砌：

- 反事实：「若违反重力，人物会离地飘浮」→ 写入：`禁止无支撑悬浮、向上飘`
- 反事实：「若违反碰撞，手会穿过杯壁」→ 写入：`禁止手穿杯、穿模`
- 反事实：「若违反因果，未触门已打开」→ 写入：`禁止未接触即形变`

每镜选 **3–5 条** 与本镜 `physics_cues` **直接对立** 的 forbidden。

## 审查 patch 触发（review-rubric 物理维）

出现以下任一项 → 必须 patch `physics_cues` + `forbidden_physics`：

- cues 只有「符合物理」等空词
- 有液体/碰撞描写但 cues 未写流向或接触
- forbidden 与 cues 无对立关系（重复同一意思）
- action_beats 顺序违反因果（先结果后因）

## 参考

- [VideoPhy paper](https://arxiv.org/abs/2406.03520) — SA + PC 人类评测
- [PhysVid](https://github.com/5aurabhpathak/PhysVid) — counterfactual local negative prompts
- [Enhancing Physical Plausibility (PAR+SDG)](https://arxiv.org/abs/2509.24702) — implausibility reasoning
