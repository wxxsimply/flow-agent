#!/usr/bin/env python3
"""WMReward 选优桩脚本：stdout 打印 surprise（越低越好）。

接入 facebookresearch/WMReward 时，将本文件替换为仓库中的
compute_vjepa_surprise 包装，或通过 FLOWAGENT_WMREWARD_SCRIPT 指向真实实现。

用法:
  python scripts/wmreward/compute_surprise.py path/to/clip.mp4
"""
from __future__ import annotations

import sys


def main() -> None:
    if len(sys.argv) < 2:
        print("usage: compute_surprise.py <video.mp4>", file=sys.stderr)
        sys.exit(2)
    # 桩：固定返回中等 surprise；真实实现应调用 V-JEPA world model
    print("12.5")


if __name__ == "__main__":
    main()
