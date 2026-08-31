"""QC snapshots: render a scene at specific times to PNGs.
Usage: python3 snap.py scene.html timeline.json outdir t1 t2 t3 ...  (or 'beats' = each event midpoint)"""
import json
import os
import sys

from playwright.sync_api import sync_playwright

from common import CHROME, CHROME_ARGS


def snap(scene, timeline_path, outdir, times, size=(1920, 1080)):
    tl = json.load(open(timeline_path))
    if times == ["beats"]:
        times = [round((e["t0"] + e["t1"]) / 2, 2) for e in tl["events"]]
    os.makedirs(outdir, exist_ok=True)
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=CHROME, args=CHROME_ARGS)
        pg = b.new_page(viewport={"width": size[0], "height": size[1]})
        pg.add_init_script("window.TIMELINE = " + json.dumps(tl))
        pg.goto("file://" + os.path.abspath(scene))
        pg.wait_for_function("window.ready === true && document.fonts.status === 'loaded'")
        for t in times:
            pg.evaluate(f"window.seek({t})")
            pg.screenshot(path=os.path.join(outdir, f"t{float(t):07.2f}.png"))
        b.close()
    print("snapped", times)


if __name__ == "__main__":
    sz = (1920, 1080)
    args = sys.argv[1:]
    if args[0].startswith("--size"):
        sz = tuple(map(int, args.pop(0).split("=")[1].split("x")))
    snap(args[0], args[1], args[2], args[3:] if len(args) > 3 else ["beats"], sz)
