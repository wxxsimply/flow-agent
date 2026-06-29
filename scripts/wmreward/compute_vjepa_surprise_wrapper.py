#!/usr/bin/env python3

"""Thin wrapper for facebookresearch/WMReward V-JEPA surprise scoring.



Requires a local clone of WMReward and CUDA for vitg (see upstream README).



Setup:

  git clone https://github.com/facebookresearch/WMReward.git

  set WMREWARD_REPO=D:\\path\\to\\WMReward



flow-agent stack yaml:

  video:

    wmreward_bon:

      script_path: scripts/wmreward/compute_vjepa_surprise_wrapper.py



Or: set FLOWAGENT_WMREWARD_SCRIPT to this file path.



Usage:

  python scripts/wmreward/compute_vjepa_surprise_wrapper.py path/to/clip.mp4



Stdout: surprise float (lower = more physically plausible). Typical range ~0–2.

"""

from __future__ import annotations



import os

import sys





def main() -> None:

    if len(sys.argv) < 2:

        print("usage: compute_vjepa_surprise_wrapper.py <video.mp4>", file=sys.stderr)

        sys.exit(2)



    repo = os.environ.get("WMREWARD_REPO", "").strip()

    if repo:

        sys.path.insert(0, os.path.abspath(repo))



    try:

        from compute_wmreward import compute_vjepa_surprise  # type: ignore

    except ImportError as exc:

        print(

            "ERROR: cannot import compute_wmreward. Clone WMReward and set WMREWARD_REPO.\n"

            "  git clone https://github.com/facebookresearch/WMReward.git\n"

            f"  detail: {exc}",

            file=sys.stderr,

        )

        sys.exit(1)



    video_path = sys.argv[1]

    score = compute_vjepa_surprise(video_path)

    print(f"{float(score):.6f}")





if __name__ == "__main__":

    main()

