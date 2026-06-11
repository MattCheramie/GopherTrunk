---
layout: page
title: Plots (signal scopes)
description: The Plots hub — one tabbed home for GopherTrunk's per-channel signal scopes, mirroring OP25's Plots tabs
nav_group: Operate
---

# Plots (signal scopes)

The **Plots hub** (web `/plots`) gathers GopherTrunk's per-channel
signal scopes under one tabbed page, mirroring OP25's Plots tabs. They
are all **views of a single live decode**: each opens the same per-channel
receiver (`WS /api/v1/diag/symbols`) on the selected SDR, so the symbol
points, eye, tuning meters and histogram all reflect exactly what the
production demodulator sees.

| Sub-tab | What it answers | Page |
| --- | --- | --- |
| **Constellation** | How clean is the modulation? (symbol clusters / vector scope) | [Constellation](constellation.html) |
| **Symbol scope** | What does the symbol waveform look like over time? | [Symbol scope](symbol-scope.html) |
| **Eye diagram** | Is the C4FM eye open (symbol timing / SNR)? | [Eye diagram](eye-diagram.html) |
| **Mixer** | Where does the carrier sit — and does the loop centre it? (raw vs tuned baseband FFT) | [Mixer](mixer.html) |
| **Tuning** | Is the receiver locking — carrier error, AGC, clock? | [Tuning](tuning.html) |
| **Histogram** | Is the symbol distribution healthy (balance / MER)? | [Histogram](histogram.html) |

Each sub-tab is the standalone panel rendered inline; the chosen view is
reflected in the URL (`/plots/<tab>`) so it's shareable and survives a
refresh, and the individual routes (`/constellation`, `/eye`, …) still
work for deep links. The wideband **[Spectrum](spectrum.html)** waterfall
stays its own top-level tab.

## Workflow

1. Pick the **SDR** and dial the **Offset** onto a locked control or voice
   channel (or leave Hold off to follow the newest active call) — the
   offset and Mode persist across the scopes.
2. **Constellation / Eye** to judge raw modulation quality.
3. **Tuning** to confirm the loops are locking (carrier error → 0 Hz).
4. **Histogram** for an at-a-glance balance / SNR sanity check.

See each scope's own page (linked above) for the detail.
