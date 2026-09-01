"""GopherTrunk video brand tokens — single source of truth for all generators.

Matches the site tokens in docs/assets/css/style.scss (video skews dark-theme,
per strategy §7). Fonts here are the render-environment stand-ins for the
site's system font stack; swap paths if rendering elsewhere.
"""

# ---- colors (site dark theme) ------------------------------------------------
BG = "#0d1117"          # dark bg
BG_ELEV = "#161b22"     # elevated surface
TEXT = "#e6edf3"        # primary text
MUTED = "#8b96a3"       # muted text
ACCENT = "#58a6ff"      # dark-mode accent (site light accent: #155799)
ACCENT_LIGHT = "#155799"
BORDER = "#30363d"

# The 4 diagram accent colors (diagram style sheet §1.2) — used in order:
# primary stroke, secondary, warning/loss, success/gain.
DIAG = ["#58a6ff", "#d2a8ff", "#f0883e", "#3fb950"]

# Domain color stripes (title cards, thumbnails)
DOMAIN_COLORS = {
    "rf-sdr": "#58a6ff",
    "software-dev": "#3fb950",
    "hardware-devices": "#f0883e",
}

# ---- geometry ----------------------------------------------------------------
CANVAS_4K = (3840, 2160)
CANVAS_HD = (1920, 1080)
# Vertical-safe stage: keep must-read content inside the centered 9:16 zone
# (1215 px wide at 4K); graphics build on a 1350x2160 safe stage (strategy §3.2).
SAFE_W_4K = 1215
STAGE_W_4K = 1350
RADIUS = 24  # ~0.5rem at 4K scale

# ---- fonts (render-env stand-ins for the site's system sans + mono) ----------
FONT_SANS = "/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf"
FONT_SANS_BOLD = "/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf"
FONT_MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

LOGO_PNG = "/home/user/GopherTrunk/docs/assets/gophertrunk-logo.png"

EYEBROW = "GOPHERTRUNK FIELD GUIDE"


def hex_rgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i : i + 2], 16) for i in (0, 2, 4))
