"""Render a scene HTML + timeline.json to H.264 MP4 (video-only by default).

Usage:
  python3 render.py scene.html timeline.json out.mp4 [--size=1920x1080] [--audio=narr.wav] [--from=0] [--to=end]
"""
import json
import math
import os
import subprocess
import sys

from playwright.sync_api import sync_playwright

from common import CHROME, CHROME_ARGS, FF, FPS


def render(scene, timeline_path, out, size=(1920, 1080), audio=None, t_from=0.0, t_to=None):
    tl = json.load(open(timeline_path))
    dur = tl["duration"] if t_to is None else t_to
    n0, n1 = int(round(t_from * FPS)), int(math.ceil(dur * FPS))
    cmd = [FF, "-y", "-f", "image2pipe", "-vcodec", "png", "-r", str(FPS), "-i", "-"]
    if audio:
        cmd += ["-ss", str(t_from), "-i", audio]
    cmd += ["-c:v", "libx264", "-preset", "fast", "-crf", "20", "-pix_fmt", "yuv420p"]
    if audio:
        cmd += ["-c:a", "aac", "-b:a", "192k", "-ar", "48000", "-ac", "2", "-shortest"]
    cmd += [out]
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=CHROME, args=CHROME_ARGS)
        pg = b.new_page(viewport={"width": size[0], "height": size[1]})
        pg.add_init_script("window.TIMELINE = " + json.dumps(tl))
        pg.goto("file://" + os.path.abspath(scene))
        pg.wait_for_function("window.ready === true")
        pg.wait_for_function("document.fonts.status === 'loaded'")
        enc = subprocess.Popen(cmd, stdin=subprocess.PIPE,
                               stderr=subprocess.DEVNULL if os.environ.get("QUIET") else None)
        for i in range(n0, n1):
            pg.evaluate(f"window.seek({i / FPS})")
            enc.stdin.write(pg.screenshot(type="png"))
            if i % 300 == 0:
                print(f"  frame {i}/{n1}", flush=True)
        enc.stdin.close()
        rc = enc.wait()
        b.close()
    if rc != 0:
        raise SystemExit("ffmpeg failed")
    print("wrote", out)


if __name__ == "__main__":
    pos = [a for a in sys.argv[1:] if not a.startswith("--")]
    opt = dict(a[2:].split("=", 1) for a in sys.argv[1:] if a.startswith("--"))
    size = tuple(map(int, opt.get("size", "1920x1080").split("x")))
    render(pos[0], pos[1], pos[2], size=size, audio=opt.get("audio"),
           t_from=float(opt.get("from", 0)), t_to=float(opt["to"]) if "to" in opt else None)
