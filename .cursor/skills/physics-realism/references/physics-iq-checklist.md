# Physics-IQ 回归评测清单

配合 `scripts/eval-physics-iq.ps1` 与 [physics-IQ-benchmark](https://github.com/google-deepmind/physics-IQ-benchmark)。

## 运行前

- [ ] 固定第一镜样例已记录（便于对比版本）
- [ ] stack `wmreward_bon` 配置与上次一致或已注明变更
- [ ] `artifacts/applied-skills.json` 已生成

## 生成物检查

- [ ] 每镜 `physics_cues` / `forbidden_physics` 非空
- [ ] `storyboard-review.json` 无 error 级未处理项
- [ ] clips 目录每镜有 mp4，BoN 目录有 `selected.txt`（若启用）

## Physics-IQ 维度（人工快检）

| 维度 | 观察点 |
|------|--------|
| 重力 | 下落物体、跳跃落地 |
| 碰撞 | 门、桌、人肩接触反应 |
| 液体 | 雨、倒水、水面 ripple |
| 固体 | 杯、盒不穿透、不融化 |
| 时序 | 因果顺序正确（先伸手后接触） |

## 分数趋势

- 记录：裸 Wan vs Wan+BoN vs 扩写 skill 版本
- 目标：Physics-IQ I2V 分数较裸模提升（参考 WMReward 论文 ~38%→~44%）

## 参考

- google-deepmind/physics-IQ-benchmark
- facebookresearch/WMReward
