# Physics-IQ 回归评测

本仓库通过 `scripts/eval-physics-iq.ps1`（Windows）或 `scripts/eval-physics-iq.sh`（Linux/CI）对**固定第一镜样例**跑 director 管线，便于对比物理相关改动前后的产物。

CI 在 PR 上运行 job **`physics-iq-dryrun`**：`go test ./internal/runner/... -run MotionQuality|AudioDuration` + dry-run 脚本（不调用真实视频 API）。

## 快速运行

```powershell
.\scripts\eval-physics-iq.ps1 -DryRun
```

真实万相调用（耗时、计费）：

```powershell
.\scripts\eval-physics-iq.ps1
```

## 与 flow-agent 的衔接

| 阶段 | 产物 | 说明 |
|------|------|------|
| assemble | `artifacts/storyboard-review.json` | animation-craft 规则审查 + `physics_cues` / `forbidden_physics` 补全 |
| produce | `clips/*.mp4` | 启用 `wmreward_bon` 时每镜多候选选优 |
| produce | `artifacts/produce-degradation.json` | `wmreward_bon` 数组：每段 i2v 各候选 score 与选中项 |
| produce | `clips/_bon_*/scores.json` | 单段 BoN 明细（与 degradation 同步） |

栈配置见 [`config/stacks/micro-movie-wan-flash.yaml`](../config/stacks/micro-movie-wan-flash.yaml) 中 `video.wmreward_bon`。

## 官方基准

1. 克隆 [google-deepmind/physics-IQ-benchmark](https://github.com/google-deepmind/physics-IQ-benchmark)
2. 按仓库 README 安装环境与权重
3. 对 `runs/physics-iq-*` 下生成的 I2V 片段提交评测，记录 Physics-IQ 分数用于回归对比

## WMReward 脚本

### 像素启发式（默认）

未配置 `script_path` 时，BoN 使用 ffmpeg 帧间像素差（YAVG）启发式，目标约 10（0–255），**score 越低越好**；`<30` 为栈内回归目标。

### 真实 V-JEPA（可选，需 GPU）

1. 克隆 [facebookresearch/WMReward](https://github.com/facebookresearch/WMReward)
2. 设置环境变量 `WMREWARD_REPO` 指向克隆目录
3. 在 stack yaml 中启用：

```yaml
video:
  wmreward_bon:
    script_path: scripts/wmreward/compute_vjepa_surprise_wrapper.py
```

或使用环境变量 `FLOWAGENT_WMREWARD_SCRIPT`。

V-JEPA surprise 典型范围 **0–2**（与启发式 `<30` 不可直接对比）；越低表示物理 surprise 越小。

### 连通性桩脚本

`scripts/wmreward/compute_surprise.py` 固定返回 `12.5`，仅用于验证 BoN 管线连通，**不代表**物理质量。
