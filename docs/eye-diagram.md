---
layout: page
title: Eye diagram panel
description: Live P25 C4FM eye diagram (OP25's datascope) — fold the oversampled baseband over the symbol period to judge symbol timing and SNR
nav_group: Operate
---

# Eye diagram panel

The **Eye diagram panel** (web `/eye`) is GopherTrunk's take on OP25's
**datascope**: it folds the oversampled, recovered C4FM baseband over the
symbol period and overlays the windows, so the four-level *eye* is
visible. It's the view for answering "is my symbol timing and signal
quality good enough to decode?" — the question the 1-sample-per-symbol
[Symbol scope](symbol-scope.html) can hint at but can't show directly.

## What you see

Each trace is a two-symbol window of the matched-filter output, drawn
faintly and stacked on top of every other window. On a **healthy P25
C4FM** channel they pile up into **four open horizontal bands** — the
±1 / ±3 decision rails — with clear vertical gaps between them at the
centre line (the symbol-decision instant). The wider and cleaner those
gaps, the more margin the slicer has.

A **closed eye** — bands smearing together, no gap at the centre, or a
band collapsing — means trouble:

| Symptom | Likely cause |
| --- | --- |
| All four bands present, gaps narrow | Low SNR / weak signal |
| Bands smeared horizontally, centre fuzzy | Symbol-timing (clock) error |
| Outer bands merge into inner | AGC / level miscalibration, compression |
| Only two bands | Slicer collapsed to outer symbols (offset / AFC) |
| Vertical eye walls slope / drift | Sample-rate vs baud mismatch (PPM) |

The four rails are auto-scaled to the data, so the eye fills the plot
regardless of front-end gain; the vertical line marks the symbol centre.

## C4FM only

The eye is a property of the FM-discriminated 4-level baseband, so this
panel runs the **P25 C4FM** receiver. CQPSK / LSM (simulcast) has no
4-level eye — its quality view is the complex **constellation** (four
clusters), shown on the [Constellation panel](constellation.html).

## Controls

The **Offset / Hold / Centre** controls work exactly like the other
scopes: dial the offset onto a locked control or voice channel (or leave
Hold off to follow the newest active call) so the eye reflects a real
transmission rather than the centre DC spike. See the
[Constellation panel](constellation.html#getting-a-usable-picture-the-dc-spike-problem)
for the DC-spike explanation.

## How it works

- The panel opens
  `WS /api/v1/diag/symbols?device=...&proto=p25-c4fm&offset=<hz>` — the
  same per-channel receiver the Symbol scope and Constellation use.
- The receiver exposes an **eye tap**: the oversampled matched-filter
  output (`eye_sps` samples per symbol, e.g. 10 at the 48 kHz channel
  rate), scaled by the symbol-AGC gain so its rails line up with the
  slicer levels. It is the same buffer the symbol-clock loop reads, so
  the eye reflects exactly what the slicer decides on.
- Each frame carries a bounded recent window (`eye_soft`) rather than the
  full oversampled stream — the oversampled rate is `eye_sps`× the symbol
  rate, so it is capped to keep the WebSocket payload modest while still
  carrying enough windows for a stable eye.
- The panel keeps a rolling buffer and the chart folds it over `eye_sps`,
  overlaying two-symbol windows.

## Implementation

| Path | Role |
| --- | --- |
| `internal/radio/p25/phase1/receiver/receiver.go` | `EyeSink` — emits the AGC-scaled oversampled matched-filter output (C4FM) |
| `internal/scanner/symbolscope/scope.go` | Carries the bounded eye window (`EyeSoft`/`EyeSPS`) on each frame |
| `internal/api/symbols.go` | `eye_soft` / `eye_sps` on the symbol WS frame |
| `web/src/components/EyeDiagramChart.tsx` | Canvas eye renderer (folds + overlays windows) |
| `web/src/panels/EyeDiagram.tsx` | Panel: SDR + offset controls, C4FM stream |
