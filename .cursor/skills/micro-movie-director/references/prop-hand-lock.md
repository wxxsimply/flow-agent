# 手-道具锁定（Prop Hand Lock）

用于 `held_props`、`prop_refs`、`[PROP_LOCK]`、`[PROP_VIEW_LOCK]`、i2v motion negative。解决 AI 视频常见失败：**左手瞬移到右手、道具凭空消失、武器扭曲变形、物体突然换成别的**。

## held_props 格式（必填）

```
右手：匕首；左手：空
右手：长剑；左手：圆盾
右手：空；左手：伞
```

- 每镜独立写入 `held_props`，外观锁定在 `prop-sheets.json`（与 character-sheets 分离）
- produce 按镜注入：
  - `[PROPS]` 全局物体外观块
  - `[PROP_VIEW_LOCK]` 按视角锁定物体三视图
  - `[PROP_LOCK] 右手：匕首（全程保持，禁止换到另一手/消失/变形）`

## prop_refs 与三视图

- assemble / prop 阶段从分镜提取手持物 + 跨镜 hero 场景物，生成 `artifacts/prop-sheets.json` 与 `artifacts/assets/props/{id}-{front|side|back}.png`
- 每镜 `prop_refs` 绑定物体 ID（如 `p01-dagger`），produce 时与 `[PROP_VIEW_LOCK]` 耦合
- **跨镜禁止同物异名**：始终用 canonical 名称（「匕首」≠「短刀」）
- Hero 场景物（王座、信封等）须在多镜 `visual_prompt` 中重复同一称谓

## 正向（physics_cues / motion）

- 指节包裹握柄，道具全程可见
- 同一手内仅持一物，形状刚性不变
- 持物手位移最小，小幅运镜
- action_beats 每条重复当前握持手
- 物体外观须与 prop-sheets 三视图一致

## 负向（forbidden_physics / i2v）

- 禁止道具从左手瞬移到右手（prop hand swap / 道具换手）
- 禁止道具凭空消失（prop vanish / 道具消失）
- 禁止武器/杯/伞扭曲变形（prop morph / 武器变形）
- 禁止同手双物（same hand two objects）
- 禁止未写「放下/接过」的单镜内换手
- 禁止物体突然切换成别的物体（prop identity swap）

## 分镜规则

| 情况 | 做法 |
|------|------|
| 单镜持物 | 三 beat 均写「仍握于右手」 |
| 需换手 | 拆成两镜：A 镜放下 → B 镜接过 |
| 需转刀尖 | 写「仍握于右手，仅调整刀尖朝向」 |
| 跨镜同场景 | 继承上一镜 held_props 与 prop_refs |
| 换物体 | 必须拆镜并写放下/接过，否则 continuity review 报错 |

## 关键帧对齐

有 held_props 时，关键帧 beat 须与 visual_prompt 握姿一致（优先匹配 beat，否则用收势 beat）。

## 参考

- Motion-I2V / 分镜 staging：左右手分工，换手持须写放下/接过
- flow-agent：`pkg/artifacts/prop_lock.go`、`pkg/artifacts/prop_sheet.go`、`prop_registry.go`、`shotPropLockSuffix`
