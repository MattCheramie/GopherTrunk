#!/usr/bin/env python3
"""Thumbnails (pipeline doc §4): 1280x720, dark bg, big term left, simple
figure right, domain stripe. Writes PNGs to video/brand/thumbs/.

Usage: thumbs.py <outdir>
"""
import base64, sys, tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CHROME = sorted(Path("/opt/pw-browsers").glob("chromium-*/chrome-linux/chrome"))[-1]

THUMBS = [
    ("GT-TR-01", "TRUNKED RADIO", "the Field Guide course", "pool"),
    ("GT-TR-01.01", "TRUNKED RADIO", "sharing channels on demand", "pool"),
    ("GT-TR-01.02", "CONTROL CHANNEL", "the stream that never speaks", "stripe"),
    ("GT-TR-01.03", "TALKGROUP", "the channel that doesn't exist", "hop"),
    ("GT-TR-01.04", "CHANNEL GRANT", "the message that starts every call", "arrow"),
    ("GT-TR-01.05", "FDMA & TDMA", "two ways to share the air", "slots"),
]

FIGS = {
    "pool": """<rect x="40" y="30" width="420" height="64" rx="10" fill="#58a6ff" fill-opacity="0.14" stroke="#58a6ff" stroke-width="3"/>
      <text x="250" y="70" text-anchor="middle" font-size="30" fill="#e6edf3">control</text>
      <rect x="40" y="210" width="120" height="80" rx="10" fill="none" stroke="#30363d" stroke-width="3"/>
      <rect x="190" y="210" width="120" height="80" rx="10" fill="none" stroke="#30363d" stroke-width="3"/>
      <rect x="340" y="210" width="120" height="80" rx="10" stroke="#58a6ff" fill="#58a6ff" fill-opacity="0.2" stroke-width="3"/>
      <line x1="250" y1="100" x2="390" y2="200" stroke="#58a6ff" stroke-width="5" stroke-dasharray="12 9"/>""",
    "stripe": """<rect x="30" y="120" width="440" height="80" rx="12" fill="#58a6ff" fill-opacity="0.15" stroke="#58a6ff" stroke-width="3"/>
      <rect x="50" y="135" width="100" height="50" rx="6" fill="#161b22" stroke="#58a6ff" stroke-width="2"/>
      <rect x="170" y="135" width="100" height="50" rx="6" fill="#161b22" stroke="#58a6ff" stroke-width="2"/>
      <rect x="290" y="135" width="100" height="50" rx="6" fill="#161b22" stroke="#58a6ff" stroke-width="2"/>
      <text x="250" y="260" text-anchor="middle" font-size="28" fill="#8b96a3">data, never voice</text>""",
    "hop": """<line x1="30" y1="90" x2="470" y2="90" stroke="#30363d" stroke-width="2"/>
      <line x1="30" y1="170" x2="470" y2="170" stroke="#30363d" stroke-width="2"/>
      <line x1="30" y1="250" x2="470" y2="250" stroke="#30363d" stroke-width="2"/>
      <rect x="50" y="60" width="110" height="60" rx="8" fill="#58a6ff" fill-opacity="0.25" stroke="#58a6ff" stroke-width="3"/>
      <rect x="200" y="220" width="110" height="60" rx="8" fill="#58a6ff" fill-opacity="0.25" stroke="#58a6ff" stroke-width="3"/>
      <rect x="350" y="140" width="110" height="60" rx="8" fill="#58a6ff" fill-opacity="0.25" stroke="#58a6ff" stroke-width="3"/>""",
    "arrow": """<rect x="30" y="110" width="160" height="90" rx="10" fill="#58a6ff" fill-opacity="0.15" stroke="#58a6ff" stroke-width="3"/>
      <rect x="330" y="110" width="150" height="90" rx="10" fill="none" stroke="#30363d" stroke-width="3"/>
      <line x1="200" y1="155" x2="310" y2="155" stroke="#58a6ff" stroke-width="6"/>
      <path d="M 300 135 L 330 155 L 300 175 Z" fill="#58a6ff"/>""",
    "slots": """<rect x="30" y="100" width="105" height="130" fill="#58a6ff" fill-opacity="0.35" stroke="#58a6ff" stroke-width="3"/>
      <rect x="145" y="100" width="105" height="130" fill="#3fb950" fill-opacity="0.2" stroke="#3fb950" stroke-width="3"/>
      <rect x="260" y="100" width="105" height="130" fill="#58a6ff" fill-opacity="0.35" stroke="#58a6ff" stroke-width="3"/>
      <rect x="375" y="100" width="105" height="130" fill="#3fb950" fill-opacity="0.2" stroke="#3fb950" stroke-width="3"/>
      <text x="255" y="290" text-anchor="middle" font-size="30" fill="#8b96a3">time →</text>""",
}


def main():
    out = Path(sys.argv[1]); out.mkdir(parents=True, exist_ok=True)
    logo = base64.b64encode((ROOT.parent / "docs/assets/gophertrunk-logo.png").read_bytes()).decode()
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=str(CHROME), args=["--no-sandbox", "--hide-scrollbars"])
        pg = b.new_page(viewport={"width": 1280, "height": 720})
        for vid, term, sub, fig in THUMBS:
            big = 96 if len(term) <= 13 else 74
            html = f"""<!doctype html><html><body style="margin:0;width:1280px;height:720px;
background:#0d1117;font-family:'DejaVu Sans',sans-serif;overflow:hidden">
<div style="position:absolute;top:0;left:0;width:1280px;height:10px;background:#58a6ff"></div>
<div style="position:absolute;left:70px;top:180px;width:620px">
<div style="color:#8b96a3;font-size:26px;letter-spacing:6px">GOPHERTRUNK FIELD GUIDE</div>
<div style="color:#e6edf3;font-size:{big}px;font-weight:700;line-height:1.05;margin-top:18px">{term}</div>
<div style="color:#58a6ff;font-size:34px;margin-top:22px">{sub}</div></div>
<svg style="position:absolute;right:60px;top:200px" width="500" height="320">{FIGS[fig]}</svg>
<img src="data:image/png;base64,{logo}" style="position:absolute;left:70px;bottom:50px;width:90px">
<div style="position:absolute;right:70px;bottom:52px;color:#8b96a3;font-size:24px;font-family:'DejaVu Sans Mono'">{vid}</div>
</body></html>"""
            with tempfile.NamedTemporaryFile("w", suffix=".html", delete=False) as f:
                f.write(html); path = f.name
            pg.goto(f"file://{path}")
            pg.screenshot(path=str(out / f"{vid}-thumb.png"))
            Path(path).unlink()
            print(f"thumb {vid}")
        b.close()


if __name__ == "__main__":
    main()
