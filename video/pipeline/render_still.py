"""Render a brand card config to PNG via cards.html.

Usage: python3 render_still.py out.png '{"kind":"title","term":"Decibel (dB)","segid":"GT-RF-01.05"}' [WxH]
"""
import base64
import json
import os
import sys

from playwright.sync_api import sync_playwright

from common import CHROME, CHROME_ARGS, VIDEO

SIZES = {"title": (1920, 1080), "recap": (1920, 1080), "endslate": (1920, 1080),
         "thumb": (1280, 720), "thumbvert": (1080, 1920)}


def render(out_png, cfg, size=None):
    w, h = size or SIZES[cfg["kind"]]
    url = "file://" + os.path.join(VIDEO, "brand", "cards.html") + \
        "?cfg=" + base64.b64encode(json.dumps(cfg).encode()).decode()
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=CHROME, args=CHROME_ARGS)
        pg = b.new_page(viewport={"width": w, "height": h})
        pg.goto(url)
        pg.wait_for_function("document.fonts.status === 'loaded'")
        pg.screenshot(path=out_png, type="png")
        b.close()


if __name__ == "__main__":
    out = sys.argv[1]
    cfg = json.loads(sys.argv[2])
    size = tuple(map(int, sys.argv[3].split("x"))) if len(sys.argv) > 3 else None
    render(out, cfg, size)
    print("wrote", out)
