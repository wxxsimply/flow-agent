# 视频生成绝对禁止清单（i2v / 合成 negative）

用于 `forbidden_physics` 与 `MotionPromptBlock()` negative steer。依据 PhysVid counterfactual、PhyT2V 规则域、VideoPhy PC 失败模式整理。

## 手持道具（Prop Hand Lock）

- 禁止道具从左手瞬移到右手（prop hand swap）
- 禁止道具凭空消失或突然出现
- 禁止武器/杯/伞/手机扭曲变形（prop morph）
- 禁止同手双物（右手同时持剑与盾 unless held_props 写明）
- 禁止单镜内未写「放下/接过」的换手

## 空间与刚体（最高优先级）

- 禁止任何物体/人物穿透墙壁、地面、桌椅、门、其他角色
- 禁止无支撑悬浮、脚踩空、坐空椅
- 禁止刚体瞬间位移超过合理一步距离（teleport）
- 禁止物体穿模、网格穿插、肢体融合
- 禁止手指/武器穿过杯柄、门把手、对方身体
- 禁止未接触即形变（门未触已开、未倒水液已满）

## 人体与数量

- 禁止多余手指、多余肢体、五官融化变形
- 禁止同一镜头内角色数量突变（1 人变 2 人）
- 禁止克隆人、镜像复制同一角色
- 禁止道具数量与旁白不一致（说一把剑画面两把）
- 禁止关节反折、头身分离旋转

## 重力 / 运动 / 因果

- 禁止重力方向错误、人物或物体向上飘
- 禁止滑步、脚不触地、影子与步不同步
- 禁止因果颠倒：先结果后因、未触即动
- 禁止一镜内多种冲突主动作叠加

## 流体 / 材质（VideoPhy）

- 禁止雨雪水平飞、液体倒流
- 禁止干容器/干地面瞬间变湿/变满
- 禁止入水无溅射、倒水无液流
- 禁止不相溶液体瞬间均匀混合
- 禁止布料如金属硬折、无惯性滞后

## 热 / 光 / 烟 / 火

- 禁止烟雾向下沉（除非明确倒灌场景）
- 禁止火焰完全静止、无气流摇曳
- 禁止光影与光源方向矛盾、阴影随机跳变
- 禁止背景纹理呼吸式蠕动

## 时序与画面

- 禁止画面闪烁、跳帧式形变
- 禁止静态幻灯片式完全定格（除明确定帧）
- 禁止一镜内多种冲突运镜叠加

## 用法

| 阶段 | 做法 |
|------|------|
| 分镜 | 每镜 `forbidden_physics` 从本清单选 **3–5 条** + 场景定制，须与 `physics_cues` **对立** |
| Produce | 与正向 cues 成对注入 motion；详见 `produce-motion-checklist.md` |
| 审查 | 若 cues/forbidden 空泛或不对立 → patch |

## 正向成对（勿只写禁止）

每选一条 forbidden，须在 `physics_cues` 写对立正向，见 `phyt2v-positive-rules.md` 对照表。

## 参考

- [PhysVid](https://github.com/5aurabhpathak/PhysVid) — negative physics conditioning
- [PhyT2V](https://github.com/pittisl/PhyT2V) — physical rule extraction
- [VideoPhy](https://github.com/Hritikbansal/videophy) — PC failure taxonomy
