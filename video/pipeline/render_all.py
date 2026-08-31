"""Batch-render every pilot piece to video-only H.264 (16:9)."""
import json
import os
import subprocess
import sys

from common import RENDER

G = os.path.join(os.path.dirname(__file__), "..", "GT-RF-01", "graphics")

PIECES = [  # (piece dir name, scene file)
    ("GT-RF-01.P0-coldopen", "coldopen.html"),
    ("GT-RF-01.P1-intro", "mapcard.html"),
    ("GT-RF-01.01", "GT-RF-01.01.html"),
    ("GT-RF-01.T1", "mapcard.html"),
    ("GT-RF-01.02", "GT-RF-01.02.html"),
    ("GT-RF-01.T2", "mapcard.html"),
    ("GT-RF-01.03", "GT-RF-01.03.html"),
    ("GT-RF-01.T3", "mapcard.html"),
    ("GT-RF-01.04", "GT-RF-01.04.html"),
    ("GT-RF-01.T4", "mapcard.html"),
    ("GT-RF-01.05", "GT-RF-01.05.html"),
    ("GT-RF-01.T5", "mapcard.html"),
    ("GT-RF-01.06", "GT-RF-01.06.html"),
    ("GT-RF-01.P8-outro", "mapcard.html"),
    ("GT-RF-01.P9-endslate", "endslate.html"),
]


def main(only=None):
    for seg, scene in PIECES:
        if only and seg not in only:
            continue
        d = os.path.join(RENDER, seg)
        out = os.path.join(d, "video.mp4")
        tlp = os.path.join(d, "timeline.json")
        if os.path.exists(out):
            tl = json.load(open(tlp))
            print(f"skip {seg} (exists)")
            continue
        print(f"== render {seg}")
        env = dict(os.environ, QUIET="1")
        subprocess.run([sys.executable, os.path.join(os.path.dirname(__file__), "render.py"),
                        os.path.join(G, scene), tlp, out], check=True, env=env)


if __name__ == "__main__":
    main(sys.argv[1:] or None)
