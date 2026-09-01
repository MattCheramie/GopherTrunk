#!/usr/bin/env python3
"""Deterministic scene renderer (pipeline doc §8): Chromium screenshots piped
straight into ffmpeg — no frame files on disk.

Usage: render.py <timeline.json> <narration.wav> <out.mp4> [--vert] [--hook "text"]

Wide mode renders 1920x1080; --vert renders 1080x1920 with burned captions,
replaces a leading title block with a 2 s text hook, and appends a 2 s end
slate (narration is padded to match).
"""
import base64, json, subprocess, sys, tempfile
from pathlib import Path

import imageio_ffmpeg

FF = imageio_ffmpeg.get_ffmpeg_exe()
CHROME = sorted(Path("/opt/pw-browsers").glob("chromium-*/chrome-linux/chrome"))[-1]
ROOT = Path(__file__).resolve().parent.parent  # video/
FPS = 30
VERT_SLATE = 2.0


def build_html(tl, mode, hook):
    logo = base64.b64encode((ROOT.parent / "docs/assets/gophertrunk-logo.png").read_bytes()).decode()
    css = (ROOT / "brand/brand.css").read_text()
    js = (ROOT / "brand/engine.js").read_text()
    render = {"mode": mode, "timeline": tl, "captions": mode == "vert",
              "logo": "data:image/png;base64," + logo}
    return f"""<!doctype html><html><head><meta charset="utf-8">
<style>{css}</style></head><body><div id="stage"></div>
<script>window.RENDER = {json.dumps(render)};</script>
<script>{js}</script></body></html>"""


def main():
    args = sys.argv[1:]
    vertical = "--vert" in args
    hook = None
    if "--hook" in args:
        hook = args[args.index("--hook") + 1]
    tl_path, wav, out = args[0], args[1], args[2]
    tl = json.loads(Path(tl_path).read_text())

    dur = tl["dur"]
    if vertical:
        evs = tl["events"]
        if evs and evs[0]["visual"] == "title":
            evs[0] = {**evs[0], "visual": "hook", "arg": hook or tl["title"]}
        evs.append({"b": 999, "t": round(dur, 3), "dur": VERT_SLATE,
                    "visual": "endslate", "variant": "vert", "arg": ""})
        dur += VERT_SLATE

    W, H = (1080, 1920) if vertical else (1920, 1080)
    total = int(round(dur * FPS))

    with tempfile.NamedTemporaryFile("w", suffix=".html", delete=False) as f:
        f.write(build_html(tl, "vert" if vertical else "wide", hook))
        scene = f.name

    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=str(CHROME),
                              args=["--no-sandbox", "--force-color-profile=srgb", "--hide-scrollbars"])
        pg = b.new_page(viewport={"width": W, "height": H})
        pg.goto(f"file://{scene}")
        pg.wait_for_function("window.ready")
        afilt = ["-af", f"apad=pad_dur={VERT_SLATE},aresample=48000,pan=stereo|c0=c0|c1=c0"] if vertical \
            else ["-af", "aresample=48000,pan=stereo|c0=c0|c1=c0"]
        enc = subprocess.Popen(
            [FF, "-y", "-loglevel", "error",
             "-f", "image2pipe", "-vcodec", "png", "-r", str(FPS), "-i", "-",
             "-i", wav,
             "-c:v", "libx264", "-preset", "medium", "-crf", "20", "-pix_fmt", "yuv420p",
             *afilt,
             "-c:a", "aac", "-b:a", "192k", "-ar", "48000",
             "-t", f"{dur:.3f}", out],
            stdin=subprocess.PIPE)
        for i in range(total):
            pg.evaluate(f"window.seek({i / FPS:.5f})")
            enc.stdin.write(pg.screenshot(type="png"))
            if i % 300 == 0:
                print(f"  {out.rsplit('/',1)[-1]}: frame {i}/{total}", flush=True)
        enc.stdin.close()
        rc = enc.wait()
        b.close()
    Path(scene).unlink()
    if rc:
        raise SystemExit(f"ffmpeg failed rc={rc}")
    print(f"rendered {out} ({dur:.1f}s)")


if __name__ == "__main__":
    main()
