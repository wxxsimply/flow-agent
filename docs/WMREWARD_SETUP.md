# WMReward V-JEPA 部署指南

flow-agent 使用 [WMReward](https://github.com/facebookresearch/WMReward)（CVPR 2026, arXiv:2601.10553）的 V-JEPA surprise 对 BoN 候选选优。分数越低表示物理 surprise 越小。

## 1. 克隆 WMReward

```powershell
git clone https://github.com/facebookresearch/WMReward.git D:\path\to\WMReward
setx WMREWARD_REPO "D:\path\to\WMReward"
```

重启终端使环境变量生效。

## 2. 安装依赖（需 CUDA GPU）

按 WMReward 仓库 README 安装 Python 依赖与 V-JEPA 权重。典型步骤：

```powershell
cd D:\path\to\WMReward
pip install -r requirements.txt
```

## 3. 栈配置

[`config/stacks/micro-movie-wan-flash.yaml`](../config/stacks/micro-movie-wan-flash.yaml) 已默认启用：

```yaml
video:
  wmreward_bon:
    enabled: true
    hero_only: true
    candidates: 2
    script_path: scripts/wmreward/compute_vjepa_surprise_wrapper.py
```

无 GPU 时可临时改用连通性桩（固定 12.5，不代表物理质量）：

```yaml
script_path: scripts/wmreward/compute_surprise.py
```

## 4. 验证

```powershell
python scripts/wmreward/compute_vjepa_surprise_wrapper.py path\to\clip.mp4
# 应输出 0–2 范围的浮点数
```

Studio `/api/config/status` 返回 `"wmreward_ready": true` 表示环境就绪。

## 5. 评分标尺

| Scorer | 典型范围 | 越低越好 |
|--------|----------|----------|
| wmreward_script (V-JEPA) | 0–2 | 是 |
| pixel_heuristic (fallback) | 0–30 | 是 |

V-JEPA 与 pixel heuristic 不可直接对比；优先使用 V-JEPA。
