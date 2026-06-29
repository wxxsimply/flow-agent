# WMReward Best-of-N（推理时物理对齐）

## 原理

用 V-JEPA 等 world model 的 **surprise** 作为物理可信度奖励；推理时从 N 条候选选 surprise **最低**者（CVPR 2026, [WMReward](https://github.com/facebookresearch/WMReward)）。

## flow-agent 配置

```yaml
# config/stacks/micro-movie-wan-flash.yaml
video:
  wmreward_bon:
    enabled: true
    candidates: 3
    script_path: scripts/wmreward/compute_surprise.py  # 或真实 V-JEPA 包装
```

环境变量：`FLOWAGENT_WMREWARD_SCRIPT`

## 流程

1. 每镜生成 N 条 Wan i2v（prompt 变体见 `produce_wmreward.go`）
2. `scoreClipPhysics`：优先 Python 脚本 stdout 浮点；失败则帧差启发式
3. 复制最优到 `clips/sXX.mp4`，元数据在 `_bon_sXX/selected.txt`

## 候选 prompt 变体要点

- 重力与支撑、无穿模
- 预备-收势节奏
- 刚体/流体方向正确

## 成本

- `candidates: 3` ≈ 3 倍 i2v 调用；调试可 `enabled: false`

## 接入真实 WMReward

```bash
git clone https://github.com/facebookresearch/WMReward.git
set WMREWARD_REPO=D:\path\to\WMReward
# stack yaml:
# script_path: scripts/wmreward/compute_vjepa_surprise_wrapper.py
```

V-JEPA surprise 典型 0–2（越低越好），与默认 pixel heuristic 的 `<30` 标尺不同。

## 默认 pixel heuristic

未配置 script 时：ffmpeg 帧间 YAVG 差，target≈10，score = |mean−10| + maxJump×0.5，**<30 为栈内目标**。

## 参考

- arXiv:2601.10553 — Inference-time Physics Alignment
- Physics-IQ 排行榜 Wan2.2 + WMReward BoN ~44.4% I2V
