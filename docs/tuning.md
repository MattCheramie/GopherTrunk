---
layout: page
title: Tuning panel
description: Live receiver-state meters — carrier-frequency error, AGC, symbol-clock and equalizer state — GopherTrunk's take on OP25's Mixer / Tuner (FLL) tabs
nav_group: Operate
---

# Tuning panel

The **Tuning panel** (web `/tuning`) surfaces the live demodulator's
internal loop state — GopherTrunk's take on OP25's **Mixer** and
**Tuner (FLL)** tabs. It answers "is the receiver actually locking onto
this channel, and how much tuner error is it fighting?" while the other
scopes answer "what does the signal look like."

The daemon runs a parallel P25 receiver on the selected channel and
stamps its loop state onto every symbol frame, so this is the real state
of the production demod, not a re-implementation.

## What you see

### Carrier-error trend (the FLL view)

A rolling line of the receiver's **residual carrier-frequency-offset
estimate** in Hz:

- **C4FM** — the AFC's estimate of the true carrier offset.
- **CQPSK / LSM** — the carrier-recovery loop's estimate (coarse seed
  plus the fine Costas residual).

On a healthy lock the trace **converges toward 0 Hz and holds**. A
steady non-zero value is your tuner's PPM error (a 420 MHz / 50 ppm
RTL-SDR can sit ~20 kHz off); a trace that wanders or never settles is a
loop that hasn't acquired — usually a too-weak signal or the wrong demod
mode for the site.

### Loop-state meters

| Meter | Meaning | Healthy |
| --- | --- | --- |
| **Carrier error** | Current residual offset (Hz) | Near 0, steady |
| **AGC level** (C4FM) | Symbol-AGC mean\|x\|, vs its target | Tracks target |
| **AGC gain** (CQPSK) | Matched-filter AGC gain | Settles, not pinned at 1.0 |
| **Clock μ** | Symbol-clock sub-sample phase + nominal sps | Cycles steadily, no monotonic drift |
| **CMA error** (CQPSK) | Blind-equalizer convergence proxy | Trends toward 0 |

A **Clock μ** that drifts monotonically means the nominal samples-per-
symbol disagrees with the stream's actual baud (a sample-rate / PPM
issue); a **CMA error** stuck large means the equalizer never opened the
constellation.

## Controls

Pick the receiver with **Mode** (C4FM or CQPSK — match the site), and use
the **Offset / Hold / Centre** controls (shared with the other scopes) to
point the receiver at a locked control or voice channel. See the
[Constellation panel](constellation.html) for the DC-spike explanation.

## How it works

- The panel opens
  `WS /api/v1/diag/symbols?device=...&proto=<…>&offset=<hz>` — the same
  per-channel receiver the Constellation, Symbol scope and Eye diagram
  use, so all four are views of one decode.
- Each frame carries the receiver's state getters
  (`carrier_offset_hz`, `agc_level`, `agc_target`, `clock_mu`,
  `clock_sps`, `cma_error`), read live from the demod. The getters are
  demod-specific (Mueller-Müller vs Gardner, AFC vs carrier recovery);
  the daemon routes by the selected mode.

## Implementation

| Path | Role |
| --- | --- |
| `internal/radio/p25/phase1/receiver/receiver.go` | The state getters (`AFCOffsetHz`, `AGCLevel`, `MMClockMu`, `CQPSKCarrierOffsetHz`, `CMAError`, …) |
| `internal/scanner/symbolscope/scope.go` | `stampMetrics` reads the getters into each frame |
| `internal/api/symbols.go` | Metric fields on the symbol WS frame |
| `web/src/panels/Tuning.tsx` | Carrier-error trend (Chart.js) + loop-state meters |
