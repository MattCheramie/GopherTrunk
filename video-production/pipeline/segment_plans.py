"""Per-segment scene plans: slug → {"scenes": [spec per narration paragraph],
"recap": [bullets]}. A spec is (scene_name, kwargs) or None for the
storyboard-slate animatic. Paragraphs without a plan render as animatic slates,
so EVERY script is previewable before its custom graphics exist.

Indices follow the script's narration paragraphs in order (the title-card [V:]
carries no narration and doesn't count). Re-check indices after any script edit.
"""

PLANS = {
    # GT-RF-01.11 — verified against scripts/GT-RF-01.11-decibel.md (9 paragraphs)
    "decibel": {
        "scenes": [
            ("card", {"lines": ["0.000 000 000 02 W", "↕  ×100,000,000,000", "2 W"],
                      "mono": True, "title": "radio numbers are ridiculous"}),
            ("card", {"title": "Decibel (dB)",
                      "lines": ["dB = 10 · log₁₀(P₁/P₂)", "", "a ratio — not an amount"]}),
            None,  # piano-octave analogy — storyboard slate until illustrated
            ("ladder", {}),
            ("card", {"title": "power vs voltage",
                      "lines": ["power:   10 · log₁₀", "voltage: 20 · log₁₀",
                                "", "because P ∝ V²"], "mono": True}),
            ("chain", {}),
            ("card", {"title": "same math, different anchor",
                      "lines": ["dB   = a ratio", "dBm  = power vs 1 mW",
                                "dBFS = level vs digital full scale"], "mono": True}),
            ("card", {"title": "each anchor has a quirk",
                      "lines": ["dBm: received signals are negative", "  −70 beats −90 (×100 power)",
                                "dBFS: 0 is the ceiling", "  0 dBFS = clipping"], "mono": True}),
            ("spectrum", {"label": "GopherTrunk: signal in dBFS, SNR in dB",
                          "show_snr": True}),
            None,  # recap paragraph → recap card (renderer supplies it)
        ],
        "recap": ["dB = a ratio, on a log scale",
                  "+3 dB ≈ ×2 · +10 dB = ×10",
                  "A suffix makes it absolute (dBm, dBFS)"],
    },
    # GT-RF-01.02 — verified against scripts/GT-RF-01.02-radio-wave.md (8 paragraphs)
    "radio-wave": {
        "scenes": [
            ("sine", {"freq": 3.0, "label": "a radio wave — 3 kHz to 300 GHz, at light speed"}),
            None,  # pond-ripple analogy — storyboard slate until illustrated
            ("sine", {"freq": 4.0, "label": "wavelength λ between crests · amplitude = height"}),
            ("card", {"title": "the only three knobs",
                      "lines": ["AMPLITUDE — its strength", "FREQUENCY — cycles per second",
                                "PHASE — position in the cycle"]}),
            ("card", {"title": "numbers worth keeping",
                      "lines": ["speed ≈ 299,792,458 m/s", "λ = c / f",
                                "received: a few microvolts"], "mono": True}),
            ("spectrum", {"label": "the wave arrives buried in noise"}),
            ("spectrum", {"label": "GopherTrunk: spectrum/waterfall + dBFS meter",
                          "show_snr": True}),
            None,  # recap paragraph → recap card
        ],
        "recap": ["EM radiation, 3 kHz–300 GHz — needs no medium",
                  "Two fields regenerating each other at light speed",
                  "Amplitude, frequency, phase — the only three knobs"],
    },
}
