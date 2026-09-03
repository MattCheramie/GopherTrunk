#!/usr/bin/env python3
"""YouTube channel branding kit: banner (2560x1440, text inside the 1546x423
TV/desktop/mobile-safe center) and an 800x800 avatar, from the brand tokens.

Usage: channel_kit.py <outdir>
"""
import base64, sys, tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CHROME = sorted(Path("/opt/pw-browsers").glob("chromium-*/chrome-linux/chrome"))[-1]


def main():
    out = Path(sys.argv[1]); out.mkdir(parents=True, exist_ok=True)
    logo = base64.b64encode((ROOT.parent / "docs/assets/gophertrunk-logo.png").read_bytes()).decode()
    from playwright.sync_api import sync_playwright

    # Banner: 2560x1440; safe zone is 1546x423 centered (x 507..2053, y 508..931).
    stripe = "".join(
        f'<div style="position:absolute;left:{120 + i * 210}px;top:1050px;width:150px;'
        f'height:70px;border:2px solid rgba(88,166,255,{0.10 + 0.05 * (i % 3)});border-radius:10px"></div>'
        for i in range(12))
    banner = f"""<!doctype html><html><body style="margin:0;width:2560px;height:1440px;
background:#0d1117;font-family:'DejaVu Sans',sans-serif;overflow:hidden">
<div style="position:absolute;top:0;left:0;width:2560px;height:12px;background:#58a6ff"></div>
<div style="position:absolute;left:0;top:340px;width:2560px;height:8px;background:#161b22"></div>
{stripe}
<div style="position:absolute;left:507px;top:508px;width:1546px;height:423px;text-align:center">
  <img src="data:image/png;base64,{logo}" style="height:150px;margin-top:20px">
  <div style="color:#e6edf3;font-size:92px;font-weight:700;margin-top:8px">GopherTrunk</div>
  <div style="color:#58a6ff;font-size:40px;margin-top:10px">Trunked radio, decoded — the Field Guide, in video</div>
</div>
</body></html>"""
    avatar = f"""<!doctype html><html><body style="margin:0;width:800px;height:800px;
background:#0d1117;overflow:hidden">
<div style="position:absolute;left:50px;top:50px;width:700px;height:700px;border-radius:50%;
border:14px solid #58a6ff;display:flex;align-items:center;justify-content:center">
  <img src="data:image/png;base64,{logo}" style="width:460px"></div>
</body></html>"""

    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=str(CHROME), args=["--no-sandbox", "--hide-scrollbars"])
        for name, html, w, h in [("banner-2560x1440.png", banner, 2560, 1440),
                                 ("avatar-800.png", avatar, 800, 800)]:
            pg = b.new_page(viewport={"width": w, "height": h})
            with tempfile.NamedTemporaryFile("w", suffix=".html", delete=False) as f:
                f.write(html); tmp = f.name
            pg.goto(f"file://{tmp}")
            pg.screenshot(path=str(out / name))
            Path(tmp).unlink(); pg.close()
            print("wrote", out / name)
        b.close()


if __name__ == "__main__":
    main()
