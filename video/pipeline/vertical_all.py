"""Vertical re-edits (1080×1920) + Shorts clips for each pilot segment.

Per segment: render vertical.html (scene band + hook + burned captions + slate),
mix narration (offset past the skipped title card) + music bed, loudnorm to
-13 LUFS, export segments-9x16/, then cut the [CLIP] spans → clips/.
"""
import json
import math
import os
import subprocess
import sys

import music
from assemble import SLUGS, loudnorm2, probe_dur, run
from common import CHROME, CHROME_ARGS, FF, FPS, RENDER

from playwright.sync_api import sync_playwright

G = os.path.join(os.path.dirname(__file__), "..", "GT-RF-01", "graphics")
OFFSET = 2.0
HOOKS = {
    "GT-RF-01.01": "A radio wave is a <span style='color:#58a6ff'>ripple</span> — not in water",
    "GT-RF-01.02": "The one number everybody knows about a signal",
    "GT-RF-01.03": "A perfect radio wave says <span style='color:#58a6ff'>nothing</span>",
    "GT-RF-01.04": "Spectrum is <span style='color:#58a6ff'>real estate</span>",
    "GT-RF-01.05": "Radio numbers are <span style='color:#58a6ff'>ridiculous</span>",
    "GT-RF-01.06": "The number that decides <span style='color:#58a6ff'>everything</span>",
}


def render_vertical(seg):
    d = os.path.join(RENDER, seg)
    tl = json.load(open(os.path.join(d, "timeline.json")))
    dur = tl["duration"] - OFFSET
    out_v = os.path.join(d, "vert_video.mp4")
    scene = os.path.abspath(os.path.join(G, seg + ".html"))
    vcfg = {"scene": "file://" + scene, "term": SLUGS[seg][1], "hook": HOOKS[seg]}
    n1 = int(math.ceil(dur * FPS))
    enc = None
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=CHROME, args=CHROME_ARGS)
        pg = b.new_page(viewport={"width": 1080, "height": 1920})
        pg.add_init_script("window.TIMELINE = " + json.dumps(tl) +
                           "; window.VCFG = " + json.dumps(vcfg))
        pg.goto("file://" + os.path.abspath(os.path.join(os.path.dirname(__file__), "vertical.html")))
        pg.wait_for_function("window.ready === true")
        pg.wait_for_function("document.fonts.status === 'loaded'")
        enc = subprocess.Popen([FF, "-y", "-f", "image2pipe", "-vcodec", "png", "-r", str(FPS),
                                "-i", "-", "-c:v", "libx264", "-preset", "fast", "-crf", "20",
                                "-pix_fmt", "yuv420p", out_v], stdin=subprocess.PIPE,
                               stderr=subprocess.DEVNULL)
        for i in range(n1):
            pg.evaluate(f"window.seek({i / FPS})")
            enc.stdin.write(pg.screenshot(type="png"))
            if i % 600 == 0:
                print(f"  {seg} vert frame {i}/{n1}", flush=True)
        enc.stdin.close()
        enc.wait()
        b.close()
    # audio: narration from OFFSET + bed, loudnorm -13
    bed = os.path.join(d, "vbed.wav")
    music.generate(dur, bed, seed=17, gain_db=-34.0)
    mixed = os.path.join(d, "vert_full.mp4")
    run([FF, "-y", "-i", out_v, "-ss", str(OFFSET), "-i", os.path.join(d, "narration.wav"),
         "-i", bed, "-filter_complex",
         "[1:a]aresample=48000:resampler=soxr,aformat=channel_layouts=stereo,apad[vo];"
         "[vo][2:a]amix=inputs=2:duration=shortest:normalize=0[a]",
         "-map", "0:v", "-map", "[a]", "-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
         "-shortest", mixed])
    slug = SLUGS[seg][0]
    outdir = os.path.join(RENDER, "exports", "segments-9x16")
    os.makedirs(outdir, exist_ok=True)
    final = os.path.join(outdir, f"{seg}-{slug}-vertical.mp4")
    loudnorm2(mixed, final, target_i=-13.0)
    # clips
    clipdir = os.path.join(RENDER, "exports", "clips")
    os.makedirs(clipdir, exist_ok=True)
    for cid, span in tl.get("clips", {}).items():
        a = max(0, span["t0"] - OFFSET)
        bnd = min(dur, span["t1"] - OFFSET + 0.4)
        cut = os.path.join(clipdir, f"{seg}{cid}.mp4")
        run([FF, "-y", "-ss", str(a), "-to", str(bnd), "-i", final,
             "-c:v", "libx264", "-preset", "fast", "-crf", "20", "-pix_fmt", "yuv420p",
             "-c:a", "aac", "-b:a", "160k", cut])
    print(f"done {seg}")


if __name__ == "__main__":
    segs = sys.argv[1:] or list(SLUGS)
    for s in segs:
        render_vertical(s)
