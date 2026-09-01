#!/usr/bin/env python3
"""Brand-kit card generators (playbook §1.2): title card, recap card,
lower-third, corner bug composite, thumbnail, map card.

Every card keeps its must-read content inside the centered 9:16 crop zone so
term segments crop to vertical with zero re-layout (playbook §2.1 tech notes).

CLI:
  python3 gen_cards.py demo              # render one of each into renders/
  python3 gen_cards.py title "Term" GT-RF-01.03 out.png
"""
import sys, os
from PIL import Image, ImageDraw, ImageFont

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import brand as B


def _font(path, size):
    return ImageFont.truetype(path, size)


def _center_zone(w, h):
    zone_w = int(h * 9 / 16)
    x0 = (w - zone_w) // 2
    return x0, x0 + zone_w


def _wrap(draw, text, font, max_w):
    words, lines, cur = text.split(), [], ""
    for w in words:
        t = (cur + " " + w).strip()
        if draw.textlength(t, font=font) <= max_w:
            cur = t
        else:
            if cur:
                lines.append(cur)
            cur = w
    if cur:
        lines.append(cur)
    return lines


def _base(size=B.CANVAS_4K, bg=B.BG):
    img = Image.new("RGB", size, B.hex_rgb(bg))
    return img, ImageDraw.Draw(img)


def _stripe(draw, size, color, height_frac=0.012):
    w, h = size
    draw.rectangle([0, h - int(h * height_frac), w, h], fill=B.hex_rgb(color))


def _logo(img, scale_h, pos, opacity=255):
    try:
        logo = Image.open(B.LOGO_PNG).convert("RGBA")
    except FileNotFoundError:
        return
    r = scale_h / logo.height
    logo = logo.resize((int(logo.width * r), scale_h))
    if opacity < 255:
        a = logo.getchannel("A").point(lambda v: v * opacity // 255)
        logo.putalpha(a)
    img.paste(logo, pos, logo)


def title_card(term, seg_id, out, size=B.CANVAS_4K, domain="rf-sdr", subtitle=None):
    img, d = _base(size)
    w, h = size
    s = h / 2160.0  # scale factor vs 4K
    zx0, zx1 = _center_zone(w, h)
    zone_w = zx1 - zx0

    eyebrow_f = _font(B.FONT_SANS_BOLD, int(64 * s))
    d.text((zx0 + 40 * s, h * 0.30), B.EYEBROW, font=eyebrow_f,
           fill=B.hex_rgb(B.ACCENT), features=None)

    # term, wrapped inside the 9:16 zone
    tsize = int(220 * s)
    tf = _font(B.FONT_SANS_BOLD, tsize)
    lines = _wrap(d, term, tf, zone_w - 80 * s)
    while len(lines) > 3 and tsize > int(120 * s):
        tsize = int(tsize * 0.85)
        tf = _font(B.FONT_SANS_BOLD, tsize)
        lines = _wrap(d, term, tf, zone_w - 80 * s)
    y = h * 0.38
    for ln in lines:
        d.text((zx0 + 40 * s, y), ln, font=tf, fill=B.hex_rgb(B.TEXT))
        y += tsize * 1.12

    if subtitle:
        sf = _font(B.FONT_SANS, int(72 * s))
        d.text((zx0 + 40 * s, y + 30 * s), subtitle, font=sf, fill=B.hex_rgb(B.MUTED))

    idf = _font(B.FONT_MONO, int(56 * s))
    d.text((w - 60 * s - d.textlength(seg_id, font=idf), h - 140 * s), seg_id,
           font=idf, fill=B.hex_rgb(B.MUTED))
    _stripe(d, size, B.DOMAIN_COLORS.get(domain, B.ACCENT))
    _logo(img, int(120 * s), (int(60 * s), int(60 * s)))
    img.save(out)


def recap_card(term, bullets, seg_id, out, size=B.CANVAS_4K, domain="rf-sdr"):
    img, d = _base(size)
    w, h = size
    s = h / 2160.0
    zx0, zx1 = _center_zone(w, h)
    zone_w = zx1 - zx0

    hf = _font(B.FONT_SANS_BOLD, int(110 * s))
    d.text((zx0 + 40 * s, h * 0.16), term, font=hf, fill=B.hex_rgb(B.TEXT))
    d.line([zx0 + 40 * s, h * 0.16 + 150 * s, zx1 - 40 * s, h * 0.16 + 150 * s],
           fill=B.hex_rgb(B.ACCENT), width=int(8 * s))

    bf = _font(B.FONT_SANS, int(80 * s))
    marks = ["①", "②", "③"]  # ①②③
    y = h * 0.30
    for i, btxt in enumerate(bullets[:3]):
        lines = _wrap(d, btxt, bf, zone_w - 220 * s)
        d.text((zx0 + 40 * s, y), marks[i], font=bf, fill=B.hex_rgb(B.ACCENT))
        for ln in lines:
            d.text((zx0 + 170 * s, y), ln, font=bf, fill=B.hex_rgb(B.TEXT))
            y += 100 * s
        y += 60 * s

    ptr = "Full write-up: gophertrunk.org/reference"
    pf = _font(B.FONT_SANS, int(60 * s))
    d.text((zx0 + 40 * s, h * 0.85), ptr, font=pf, fill=B.hex_rgb(B.MUTED))
    idf = _font(B.FONT_MONO, int(56 * s))
    d.text((w - 60 * s - d.textlength(seg_id, font=idf), h - 140 * s), seg_id,
           font=idf, fill=B.hex_rgb(B.MUTED))
    _stripe(d, size, B.DOMAIN_COLORS.get(domain, B.ACCENT))
    img.save(out)


def lower_third(text, out, size=B.CANVAS_4K):
    """Transparent PNG strip for overlaying on footage."""
    w, h = size
    s = h / 2160.0
    img = Image.new("RGBA", size, (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    f = _font(B.FONT_SANS_BOLD, int(84 * s))
    tw = d.textlength(text, font=f)
    x, y = int(180 * s), int(h * 0.82)
    pad = int(36 * s)
    d.rounded_rectangle([x, y, x + tw + 2 * pad, y + int(150 * s)],
                        radius=int(B.RADIUS * s), fill=(*B.hex_rgb(B.BG_ELEV), 235))
    d.text((x + pad, y + int(28 * s)), text, font=f, fill=B.hex_rgb(B.TEXT))
    d.line([x + pad, y + int(132 * s), x + pad + tw, y + int(132 * s)],
           fill=B.hex_rgb(B.ACCENT), width=int(8 * s))
    img.save(out)


def corner_bug(out, size=B.CANVAS_4K):
    """Transparent overlay: gopher mark top-right, OUTSIDE the 9:16 center zone
    (deliberately excluded from vertical crops), low opacity."""
    w, h = size
    s = h / 2160.0
    img = Image.new("RGBA", size, (0, 0, 0, 0))
    _logo(img, int(140 * s), (int(w - 240 * s), int(60 * s)), opacity=110)
    img.save(out)


def thumbnail(term, out, size=(1280, 720), domain="rf-sdr", kicker=None):
    img, d = _base(size)
    w, h = size
    f = _font(B.FONT_SANS_BOLD, 150)
    lines = _wrap(d, term, f, int(w * 0.60))
    while len(lines) > 3:
        f = _font(B.FONT_SANS_BOLD, int(f.size * 0.85))
        lines = _wrap(d, term, f, int(w * 0.60))
    y = int(h * 0.22)
    for ln in lines:
        d.text((60, y), ln, font=f, fill=B.hex_rgb(B.TEXT))
        y += int(f.size * 1.1)
    if kicker:
        kf = _font(B.FONT_SANS_BOLD, 48)
        d.text((60, int(h * 0.10)), kicker.upper(), font=kf, fill=B.hex_rgb(B.ACCENT))
    d.rectangle([0, h - 14, w, h], fill=B.hex_rgb(B.DOMAIN_COLORS.get(domain, B.ACCENT)))
    _logo(img, 90, (w - 160, 24))
    img.save(out)


def map_card(nodes, done_idx, current_idx, out, size=B.CANVAS_4K,
             title="Radio Fundamentals — the course map"):
    """Course diagram with per-node highlight states (playbook §4.2).
    nodes: list of (act_number, label). done nodes lit, current pulses (bright)."""
    img, d = _base(size)
    w, h = size
    s = h / 2160.0
    tf = _font(B.FONT_SANS_BOLD, int(90 * s))
    d.text((w * 0.05, h * 0.05), title, font=tf, fill=B.hex_rgb(B.TEXT))

    acts = {}
    for i, (act, label) in enumerate(nodes):
        acts.setdefault(act, []).append((i, label))
    col_w = w * 0.30
    nf = _font(B.FONT_SANS, int(52 * s))
    af = _font(B.FONT_SANS_BOLD, int(60 * s))
    for a, (act, items) in enumerate(sorted(acts.items())):
        x = w * 0.05 + a * col_w
        d.text((x, h * 0.16), f"Act {act}", font=af, fill=B.hex_rgb(B.ACCENT))
        y = h * 0.22
        for i, label in items:
            if i == current_idx:
                fill, dot = B.TEXT, B.ACCENT
                d.rounded_rectangle(
                    [x - 20 * s, y - 12 * s,
                     x + d.textlength(label, font=nf) + 90 * s, y + 70 * s],
                    radius=int(B.RADIUS * s), outline=B.hex_rgb(B.ACCENT),
                    width=int(6 * s))
            elif i <= done_idx:
                fill, dot = B.TEXT, B.DIAG[3]
            else:
                fill, dot = B.MUTED, B.BORDER
            d.ellipse([x, y + 10 * s, x + 36 * s, y + 46 * s], fill=B.hex_rgb(dot))
            d.text((x + 60 * s, y), label, font=nf, fill=B.hex_rgb(fill))
            y += int(102 * s)
    _stripe(d, size, B.DOMAIN_COLORS["rf-sdr"])
    img.save(out)


GT_RF_01_NODES = [
    (1, "Electromagnetic spectrum"), (1, "Radio wave"), (1, "Frequency"),
    (1, "Wavelength"), (1, "Amplitude"), (1, "Phase"), (1, "Frequency bands"),
    (2, "Carrier wave"), (2, "Modulation"), (2, "Bandwidth"), (2, "Decibel (dB)"),
    (2, "Noise floor"), (2, "Signal-to-noise ratio"), (2, "Attenuation"),
    (2, "Path loss"),
    (3, "Impedance"), (3, "Resonance"), (3, "Harmonics"), (3, "Phase noise"),
    (3, "ERP & EIRP"), (3, "Link budget"), (3, "Shannon capacity"),
]


def demo():
    here = os.path.join(os.path.dirname(os.path.abspath(__file__)), "renders")
    os.makedirs(here, exist_ok=True)
    title_card("Decibel (dB)", "GT-RF-01.11", f"{here}/title-decibel.png")
    recap_card("Decibel (dB)",
               ["dB = a ratio, on a log scale",
                "+3 dB ≈ ×2 · +10 dB = ×10",
                "A suffix makes it absolute (dBm, dBFS)"],
               "GT-RF-01.11", f"{here}/recap-decibel.png")
    lower_third("Signal-to-noise ratio · SNR", f"{here}/lower-third.png")
    corner_bug(f"{here}/corner-bug.png")
    thumbnail("Radio Fundamentals", f"{here}/thumb-pillar.png",
              kicker="The complete course")
    map_card(GT_RF_01_NODES, done_idx=10, current_idx=11, out=f"{here}/map-card.png")
    print("demo renders written to", here)


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "demo":
        demo()
    elif len(sys.argv) == 5 and sys.argv[1] == "title":
        title_card(sys.argv[2], sys.argv[3], sys.argv[4])
    else:
        print(__doc__)
