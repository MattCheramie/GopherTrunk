#!/usr/bin/env python3
"""Shorts clips (pipeline doc §11): cut the vertical at scripted [CLIP] bounds,
burn a hook/setup card over the first 2.2 s, loudnorm to -13 LUFS.

The imageio-ffmpeg static build has NO drawtext filter (see §14 2026-09-01), so
the hook card is a Chromium-rendered transparent PNG composited with `overlay`.

Usage: shorts.py <timelines_dir> <verticals_dir> <out_dir>
"""
import json, subprocess, sys, tempfile
from pathlib import Path

import imageio_ffmpeg
from assemble import loudnorm, probe_dur

FF = imageio_ffmpeg.get_ffmpeg_exe()
CHROME = sorted(Path("/opt/pw-browsers").glob("chromium-*/chrome-linux/chrome"))[-1]

HOOKS = {
    "GT-TR-01.01c1": "Why 30 radio channels sit silent",
    "GT-TR-01.01c2": "Radio channels that exist for one call",
    "GT-TR-01.02c1": "The radio channel that never speaks",
    "GT-TR-01.02c2": "Joining a call you never heard start",
    "GT-TR-01.03c1": "This radio channel doesn't exist",
    "GT-TR-01.03c2": "Two numbers identify every radio call",
    "GT-TR-01.04c1": "One message moves 100 radios at once",
    "GT-TR-01.04c2": "Inside the message that starts every call",
    "GT-TR-01.05c1": "The simplest way radios share the air",
    "GT-TR-01.05c2": "How 4 calls fit on one frequency",
}


def hook_png(pg, text, path):
    html = f"""<!doctype html><html><body style="margin:0;width:1080px;height:1920px;
background:transparent;font-family:'DejaVu Sans',sans-serif;overflow:hidden">
<div style="position:absolute;top:260px;left:50%;transform:translateX(-50%);
max-width:920px;padding:22px 36px;background:rgba(13,17,23,.88);
border:1px solid rgba(48,54,61,.9);border-radius:18px;color:#fff;
font-size:52px;font-weight:700;text-align:center;line-height:1.25">{text}</div>
</body></html>"""
    with tempfile.NamedTemporaryFile("w", suffix=".html", delete=False) as f:
        f.write(html); tmp = f.name
    pg.goto(f"file://{tmp}")
    pg.screenshot(path=path, omit_background=True)
    Path(tmp).unlink()


def main():
    tld, vd, od = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
    od.mkdir(parents=True, exist_ok=True)
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=str(CHROME), args=["--no-sandbox", "--hide-scrollbars"])
        pg = b.new_page(viewport={"width": 1080, "height": 1920})
        for tlf in sorted(tld.glob("GT-TR-01.0[1-5].timeline.json")):
            tl = json.loads(tlf.read_text())
            seg = tl["seg"]
            vsrc = vd / f"{seg}-vertical.mp4"
            for cname, bnd in tl.get("clips", {}).items():
                cid = f"{seg}{cname}"
                start, end = bnd["start"], min(bnd["end"] + 0.4, probe_dur(vsrc))
                png = od / f"{cid}.hook.png"
                hook_png(pg, HOOKS.get(cid, tl["title"]), str(png))
                raw = od / f"{cid}.raw.mp4"
                subprocess.run([FF, "-y", "-loglevel", "error",
                                "-ss", f"{start:.3f}", "-to", f"{end:.3f}", "-i", str(vsrc),
                                "-i", str(png),
                                "-filter_complex", "[0:v][1:v]overlay=0:0:enable='lt(t,2.2)'",
                                "-c:v", "libx264", "-preset", "medium", "-crf", "20",
                                "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "192k",
                                str(raw)], check=True)
                loudnorm(raw, od / f"{cid}.mp4", I=-13.0)
                raw.unlink(); png.unlink()
                print(f"{cid}: {end-start:.1f}s")
        b.close()


if __name__ == "__main__":
    main()
