"""Render the pillar + per-segment thumbnails into video/brand/thumbnails/."""
import os

from render_still import render
from common import VIDEO

OUT = os.path.join(VIDEO, "brand", "thumbnails")
A = "#58a6ff"; G = "#3fb950"; V = "#bc8cff"; W = "#d29922"; T = "#e6edf3"; M = "#8b96a3"

def sine(y, amp, cyc, col, w=500, sw=5):
    import math
    pts = " ".join(f"{'M' if x == 0 else 'L'}{x} {y - amp * math.sin(2 * math.pi * cyc * x / w):.0f}"
                   for x in range(0, w + 1, 6))
    return f'<path d="{pts}" stroke="{col}" stroke-width="{sw}" fill="none"/>'

rings = ''.join(f'<circle cx="260" cy="290" r="{r}" fill="none" stroke="{A}" stroke-width="5" opacity="{o}"/>'
                for r, o in [(60, .95), (125, .6), (190, .35), (255, .16)])

THUMBS = [
    ("GT-RF-01-pillar.png", {"kind": "thumb", "term": "Radio waves, explained",
      "sub": "Radio Fundamentals — the complete course", "eyebrow": "GT-RF-01 · Field Guide course",
      "art": f'<svg viewBox="0 0 520 560">{rings}<line x1="260" y1="300" x2="260" y2="420" stroke="{T}" stroke-width="7"/></svg>'}),
    ("GT-RF-01.01-radio-wave.png", {"kind": "thumb", "term": "What IS a radio wave?",
      "sub": "Radio wave · 3 minutes", "art": f'<svg viewBox="0 0 520 560">{rings}</svg>'}),
    ("GT-RF-01.02-frequency.png", {"kind": "thumb", "term": "Frequency & wavelength",
      "sub": "The seesaw: λ = c / f · 3 minutes",
      "art": f'<svg viewBox="0 0 520 560">{sine(180, 55, 2, A)}{sine(390, 55, 8, V)}</svg>'}),
    ("GT-RF-01.03-modulation.png", {"kind": "thumb", "term": "Making a wave talk",
      "sub": "Modulation · AM, FM, and symbols · 3 minutes",
      "art": f'<svg viewBox="0 0 520 560">{sine(140, 40, 1.5, T)}{sine(300, 45, 7, G)}{sine(450, 45, 9, A)}</svg>'}),
    ("GT-RF-01.04-bandwidth.png", {"kind": "thumb", "term": "Spectrum is real estate",
      "sub": "Bandwidth · 3 minutes",
      "art": f'<svg viewBox="0 0 520 560"><path d="M40 460 C140 460 150 160 260 160 C370 160 380 460 480 460" fill="{A}" fill-opacity=".2" stroke="{A}" stroke-width="5"/><line x1="120" y1="240" x2="400" y2="240" stroke="{T}" stroke-width="4"/><text x="260" y="215" fill="{T}" font-size="34" text-anchor="middle" font-family="Inter">bandwidth</text></svg>'}),
    ("GT-RF-01.05-decibel.png", {"kind": "thumb", "term": "Do dB math in your head",
      "sub": "Decibel (dB) · +3 = ×2 · +10 = ×10 · 4 minutes",
      "art": f'<svg viewBox="0 0 520 560"><g font-family="JetBrains Mono" font-size="44" text-anchor="middle"><rect x="90" y="80" width="340" height="100" rx="16" fill="none" stroke="{G}" stroke-width="4"/><text x="260" y="143" fill="{G}">+3 dB = ×2</text><rect x="90" y="230" width="340" height="100" rx="16" fill="none" stroke="{A}" stroke-width="4"/><text x="260" y="293" fill="{A}">+10 dB = ×10</text><rect x="90" y="380" width="340" height="100" rx="16" fill="none" stroke="{W}" stroke-width="4"/><text x="260" y="443" fill="{W}">−3 dB = ½</text></g></svg>'}),
    ("GT-RF-01.06-snr.png", {"kind": "thumb", "term": "The number that decides everything",
      "sub": "Signal-to-noise ratio · 3 minutes",
      "art": f'<svg viewBox="0 0 520 560"><path d="M20 470 L80 462 L140 474 L200 466 L260 472 L320 464 L380 473 L440 466 L500 471" stroke="{W}" stroke-width="4" fill="none"/><path d="M230 470 L260 130 L290 470 Z" fill="{A}" fill-opacity=".25" stroke="{A}" stroke-width="5"/><line x1="350" y1="130" x2="350" y2="470" stroke="{G}" stroke-width="5"/><text x="375" y="300" fill="{G}" font-size="40" font-family="Inter">SNR</text></svg>'}),
    ("GT-RF-01-vertical.png", {"kind": "thumbvert", "hook": "How radio actually works",
      "art": f'<svg viewBox="0 0 800 800"><g transform="translate(140,110)">{rings}</g></svg>'}),
]

os.makedirs(OUT, exist_ok=True)
for name, cfg in THUMBS:
    render(os.path.join(OUT, name), cfg)
    print("thumb", name)
