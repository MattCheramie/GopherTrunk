---
layout: page
title: Symbol histogram panel
description: Recovered-symbol distribution and a derived signal-quality (balance + MER/SNR) readout off the live demod
nav_group: Operate
---

# Symbol histogram panel

The **Symbol histogram panel** (web `/histogram`) shows the distribution
of the recovered symbols off the live demod, plus a derived
signal-quality readout. It's the quickest "is the slicer healthy?" check:
a P25 control channel is scrambled, so its symbols are evenly spread, and
a lopsided histogram immediately flags a problem.

## What you see

### The histogram

A bar per symbol level (4 for both P25 C4FM and CQPSK). Because the
on-air stream is scrambled, a healthy channel spreads the symbols
**evenly — each bin near 25%**. Departures are diagnostic:

| Histogram | Meaning |
| --- | --- |
| Four bins ≈ 25% | Healthy slicer, eye open |
| Two bins dominant | Slicer collapsed to outer (±3) symbols — level / AFC trouble |
| One bin dominant | No lock / unmodulated carrier / wrong offset |
| Empty | No symbols decoding on this channel |

### Quality meters

- **Balance** — the largest deviation of any bin from the 25% ideal.
  Low is good; a high value means the distribution is skewed (collapsed
  eye, biased slicer).
- **SNR (MER)** — for **C4FM** only, an estimate from the soft samples:
  the separation between the four level means versus the spread *within*
  each level (a modulation-error-ratio-style figure). Higher dB is
  cleaner. CQPSK has no real soft track here, so this reads "—".
- **Symbols** — the number of symbols in the current window.

## Controls

Pick the receiver with **Mode**, and use the **Offset / Hold / Centre**
controls (shared with the other scopes) to point the demod at a locked
channel. See the [Constellation panel](constellation.html) for the
DC-spike explanation.

## How it works

- The panel opens
  `WS /api/v1/diag/symbols?device=...&proto=<…>&offset=<hz>` — the same
  per-channel receiver the other scopes use.
- The histogram and quality figures are computed **client-side** over a
  rolling window of the streamed `dibits` (and, for C4FM, the aligned
  `soft` samples) — no extra backend work; it's a different view of the
  symbol stream the Symbol scope and Constellation already consume.

## Implementation

| Path | Role |
| --- | --- |
| `internal/scanner/symbolscope/scope.go` | Streams the `dibits` / `soft` the histogram is computed from |
| `web/src/panels/Histogram.tsx` | Bar chart (Chart.js) + balance / MER computation |
