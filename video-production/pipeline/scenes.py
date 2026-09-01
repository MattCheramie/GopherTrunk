"""Reusable motion-graphic scenes (playbook: park every animation in a shared
library — one scene serves many segments).

A scene is `f(t, dur, size) -> PIL.Image` where t is seconds into the scene.
All scenes: site dark theme, line-art style, must-read content inside the
centered 9:16 zone. Easing: 0.3–0.5 s ease-ins, no gratuitous motion.
"""
import math, os, sys
import numpy as np
from PIL import Image, ImageDraw, ImageFont

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "_brand"))
import brand as B


def _f(path, size):
    return ImageFont.truetype(path, size)


def ease(x):
    x = max(0.0, min(1.0, x))
    return x * x * (3 - 2 * x)  # smoothstep


def base(size):
    img = Image.new("RGB", size, B.hex_rgb(B.BG))
    return img, ImageDraw.Draw(img)


def zone(size):
    w, h = size
    zw = int(h * 9 / 16)
    x0 = (w - zw) // 2
    return x0, x0 + zw


def _polyline(d, pts, color, width):
    d.line(pts, fill=B.hex_rgb(color), width=width, joint="curve")


# ---------------------------------------------------------------- sine wave --
def sine_scene(t, dur, size, freq=2.0, label="a radio wave", amp_mod=None):
    """A traveling sine wave drawing itself in over the first second."""
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = zone(size)
    reveal = ease(t / 1.0)
    n = int((x1 - x0) * reveal)
    pts = []
    for i in range(max(n, 2)):
        x = x0 + i
        ph = 2 * math.pi * (freq * (i / (x1 - x0)) - 0.6 * t)
        a = 0.32 * h
        if amp_mod:
            a *= amp_mod(t, i / (x1 - x0))
        pts.append((x, h * 0.48 + a * math.sin(ph)))
    _polyline(d, pts, B.DIAG[0], max(2, int(6 * s)))
    d.line([x0, h * 0.48, x1, h * 0.48], fill=B.hex_rgb(B.BORDER), width=max(1, int(2 * s)))
    lf = _f(B.FONT_SANS, int(44 * s))
    d.text((x0, h * 0.86), label, font=lf, fill=B.hex_rgb(B.MUTED))
    return img


def two_freq_scene(t, dur, size):
    """The frequency article's SVG figure, animated: low-f wave on top,
    high-f wave below, cycle counts appearing."""
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = zone(size)
    for row, (freq, label, color) in enumerate(
        [(2, "low frequency — few cycles per second", B.DIAG[0]),
         (8, "high frequency — many cycles per second", B.DIAG[1])]):
        cy = h * (0.30 + row * 0.35)
        reveal = ease((t - row * 0.6) / 1.0)
        n = int((x1 - x0) * reveal)
        pts = []
        for i in range(max(n, 2)):
            x = x0 + i
            ph = 2 * math.pi * (freq * (i / (x1 - x0)) - 0.5 * t)
            pts.append((x, cy + 0.12 * h * math.sin(ph)))
        _polyline(d, pts, color, max(2, int(6 * s)))
        if reveal > 0.99:
            lf = _f(B.FONT_SANS, int(38 * s))
            d.text((x0, cy + 0.15 * h), label, font=lf, fill=B.hex_rgb(B.MUTED))
    return img


# ------------------------------------------------------------------ spectrum --
def spectrum_scene(t, dur, size, carrier_db=-55.0, floor_db=-95.0, label=None,
                   show_snr=False):
    """Spectrum panel: noise-floor 'grass' plus one carrier spike; optional SNR
    bracket. Mirrors GopherTrunk's spectrum view styling."""
    rng = np.random.default_rng(int(t * 30))
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = zone(size)
    top, bot = h * 0.18, h * 0.78
    d.rectangle([x0, top, x1, bot], outline=B.hex_rgb(B.BORDER), width=max(1, int(2 * s)))

    def db_to_y(db):
        return top + (0 - db) / 110.0 * (bot - top)

    n = x1 - x0
    floor = floor_db + rng.normal(0, 2.0, n)
    xs = np.arange(n)
    spike_pos, bw = 0.55, 0.012
    spike = (carrier_db - floor_db) * np.exp(-0.5 * ((xs / n - spike_pos) / bw) ** 2)
    lvl = floor + spike * ease(t / 0.8)
    pts = [(x0 + int(i), db_to_y(v)) for i, v in zip(xs, lvl)]
    _polyline(d, pts, B.DIAG[0], max(1, int(3 * s)))

    gf = _f(B.FONT_MONO, int(30 * s))
    for db in (-40, -60, -80, -100):
        y = db_to_y(db)
        d.line([x0, y, x1, y], fill=B.hex_rgb(B.BG_ELEV), width=1)
        d.text((x0 - 90 * s, y - 15 * s), f"{db}", font=gf, fill=B.hex_rgb(B.MUTED))
    if show_snr and t > 1.2:
        a = ease((t - 1.2) / 0.5)
        cx = x0 + int(spike_pos * n)
        y1, y2 = db_to_y(carrier_db), db_to_y(floor_db)
        d.line([cx + 60 * s, y1, cx + 60 * s, y2], fill=B.hex_rgb(B.DIAG[3]),
               width=max(2, int(4 * s)))
        d.text((cx + 80 * s, (y1 + y2) / 2 - 20 * s),
               f"SNR {carrier_db - floor_db:.0f} dB", font=_f(B.FONT_SANS_BOLD, int(40 * s)),
               fill=tuple(int(c * a) for c in B.hex_rgb(B.DIAG[3])))
    if label:
        d.text((x0, h * 0.85), label, font=_f(B.FONT_SANS, int(40 * s)),
               fill=B.hex_rgb(B.MUTED))
    return img


# ----------------------------------------------------------------- text card --
def card_scene(t, dur, size, title=None, lines=(), mono=False, accent_first=False):
    """Generic center-safe card: title + up to 5 lines fading in one by one.
    Also the animatic fallback for paragraphs without a custom scene."""
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = zone(size)
    y = h * 0.24
    if title:
        tf = _f(B.FONT_SANS_BOLD, int(64 * s))
        d.text((x0 + 20 * s, y), title, font=tf, fill=B.hex_rgb(B.ACCENT))
        y += 110 * s
    lf = _f(B.FONT_MONO if mono else B.FONT_SANS, int(46 * s))
    for i, ln in enumerate(lines):
        a = ease((t - 0.25 * i) / 0.4)
        if a <= 0:
            continue
        col = B.ACCENT if (accent_first and i == 0) else B.TEXT
        fill = tuple(int(c * a) for c in B.hex_rgb(col))
        d.text((x0 + 20 * s, y), ln, font=lf, fill=fill)
        y += 78 * s
    return img


def storyboard_scene(t, dur, size, direction=""):
    """Animatic placeholder: renders the script's [V:] direction as a slate so
    a whole segment can be previewed before its real graphics exist."""
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = zone(size)
    d.rounded_rectangle([x0 + 10 * s, h * 0.28, x1 - 10 * s, h * 0.72],
                        radius=int(B.RADIUS * s), outline=B.hex_rgb(B.BORDER),
                        width=max(1, int(2 * s)), fill=B.hex_rgb(B.BG_ELEV))
    ef = _f(B.FONT_SANS_BOLD, int(30 * s))
    d.text((x0 + 40 * s, h * 0.31), "STORYBOARD", font=ef, fill=B.hex_rgb(B.MUTED))
    lf = _f(B.FONT_SANS, int(40 * s))
    words, cur, lines = direction.split(), "", []
    for wd in words:
        tst = (cur + " " + wd).strip()
        if d.textlength(tst, font=lf) < (x1 - x0) - 120 * s:
            cur = tst
        else:
            lines.append(cur); cur = wd
    lines.append(cur)
    y = h * 0.38
    for ln in lines[:7]:
        d.text((x0 + 40 * s, y), ln, font=lf, fill=B.hex_rgb(B.TEXT))
        y += 58 * s
    return img


# ------------------------------------------------------------------- dB ladder
def ladder_scene(t, dur, size):
    """+10 dB steps multiply power by 10; +3 dB doubles."""
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = zone(size)
    steps = [("0 dB", "×1"), ("+10 dB", "×10"), ("+20 dB", "×100"), ("+30 dB", "×1000")]
    bf = _f(B.FONT_SANS_BOLD, int(52 * s))
    mf = _f(B.FONT_MONO, int(46 * s))
    for i, (dbl, mult) in enumerate(steps):
        a = ease((t - 0.5 * i) / 0.4)
        if a <= 0:
            continue
        y = h * 0.70 - i * h * 0.15
        fill = tuple(int(c * a) for c in B.hex_rgb(B.TEXT))
        d.rounded_rectangle([x0 + 30 * s, y, x0 + 320 * s, y + 90 * s],
                            radius=int(B.RADIUS * s), fill=B.hex_rgb(B.BG_ELEV),
                            outline=B.hex_rgb(B.BORDER))
        d.text((x0 + 60 * s, y + 18 * s), dbl, font=bf, fill=fill)
        d.text((x0 + 380 * s, y + 20 * s), mult, font=mf,
               fill=tuple(int(c * a) for c in B.hex_rgb(B.DIAG[3])))
    d.text((x0 + 30 * s, h * 0.82), "+3 dB ≈ ×2   ·   −3 dB ≈ ½",
           font=_f(B.FONT_SANS_BOLD, int(48 * s)), fill=B.hex_rgb(B.ACCENT))
    return img


# ------------------------------------------------------------------ chain -----
def chain_scene(t, dur, size, hops=None):
    """Link-budget chain: boxes with dB gains/losses appearing left→right and a
    running sum. Default hops match the decibel/link-budget scripts.
    WIDE visual (uses the full 16:9 stage) — flagged for a stacked re-layout in
    the vertical export, per playbook §2.3 tech notes."""
    hops = hops or [("TX", "+37 dBm"), ("feedline", "−3 dB"), ("antenna", "+6 dB"),
                    ("path", "−120 dB"), ("RX ant", "+3 dB")]
    img, d = base(size)
    w, h = size
    s = h / 1080.0
    x0, x1 = int(w * 0.08), int(w * 0.92)
    n = len(hops)
    bw_ = (x1 - x0 - 40 * s) / n
    bf = _f(B.FONT_SANS, int(26 * s))
    vf = _f(B.FONT_MONO, int(30 * s))
    total = None
    for i, (name, val) in enumerate(hops):
        a = ease((t - 0.55 * i) / 0.4)
        if a <= 0:
            continue
        x = x0 + 20 * s + i * bw_
        y = h * 0.40
        col = B.DIAG[3] if val.startswith("+") else B.DIAG[2]
        d.rounded_rectangle([x, y, x + bw_ - 16 * s, y + 150 * s],
                            radius=int(B.RADIUS * s), fill=B.hex_rgb(B.BG_ELEV),
                            outline=B.hex_rgb(B.BORDER))
        d.text((x + 14 * s, y + 20 * s), name, font=bf, fill=B.hex_rgb(B.TEXT))
        d.text((x + 14 * s, y + 80 * s), val, font=vf,
               fill=tuple(int(c * a) for c in B.hex_rgb(col)))
        if i < n - 1 and a > 0.9:
            d.line([x + bw_ - 16 * s, y + 75 * s, x + bw_, y + 75 * s],
                   fill=B.hex_rgb(B.MUTED), width=int(4 * s))
        num = float(val.replace("dBm", "").replace("dB", "").replace("−", "-"))
        total = num if total is None else total + num
    if total is not None and t > 0.55 * n:
        d.text((x0 + 20 * s, h * 0.62), f"running sum → {total:+.0f} dBm at the receiver",
               font=_f(B.FONT_SANS_BOLD, int(44 * s)), fill=B.hex_rgb(B.ACCENT))
    return img


SCENES = {
    "sine": sine_scene,
    "two_freq": two_freq_scene,
    "spectrum": spectrum_scene,
    "card": card_scene,
    "ladder": ladder_scene,
    "chain": chain_scene,
    "storyboard": storyboard_scene,
}
