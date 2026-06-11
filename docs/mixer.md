---
layout: page
title: Mixer plot
description: Channel-baseband power spectrum, raw vs tuned — GopherTrunk's take on OP25's Raw Mixer / Tuned Mixer tabs
nav_group: Operate
---

# Mixer plot

The **Mixer plot** (web `/plots/mixer`, or `/mixer`) shows the **power
spectrum of the channel baseband** two ways — GopherTrunk's take on
OP25's **Raw Mixer** and **Tuned Mixer** tabs. It answers "where is the
carrier sitting, and is the receiver pulling it onto centre?"

The daemon runs a parallel P25 receiver on the selected channel (the same
engine the other [Plots](plots.html) use) and FFTs its channelized
baseband, so the plot reflects exactly what the production demod is
working with — not a re-implementation.

## What you see

Two stacked spectra, both **Power (dBFS) vs Frequency**, spanning the
channel baseband (±½ the DDC output rate, ~±24 kHz on P25):

- **Raw mixer** — the channelized baseband as the receiver first sees it.
  The carrier sits wherever its residual frequency offset leaves it; the
  **amber marker** flags the loop's carrier-offset estimate.
- **Tuned mixer** — the *same* window re-mixed by that carrier-offset
  estimate, so a locked loop pulls the carrier onto the **centre (0 Hz)
  line**.

Comparing the two is the whole point: the shift from the raw peak to the
centred tuned peak *is* the carrier-recovery correction.

| Symptom | Reading |
| --- | --- |
| Raw peak off-centre, **tuned peak centred** | Healthy lock — the loop is correcting the offset |
| Raw peak near centre already | Little tuner error on this channel |
| **Tuned peak still off-centre** | Loop hasn't acquired — weak signal or wrong Mode |
| No peak, just noise floor | No active signal at this Offset, or wrong channel |

A large, *steady* raw offset is your tuner's PPM error (a 420 MHz /
50 ppm RTL-SDR can sit ~20 kHz off); the tuned view should still recentre
it once the loop locks.

## Controls

Pick the receiver with **Mode** (C4FM or CQPSK — match the site), and use
the **Offset / Hold / Centre** controls (shared with the other scopes) to
point the receiver at a locked control or voice channel. With **Hold**
off, the Offset follows the newest active call on the SDR. See the
[Constellation panel](constellation.html) for the DC-spike explanation of
why off-centre channels are mixed down before channelizing.

## How it works

- The panel opens
  `WS /api/v1/diag/mixer?device=...&proto=<…>&offset=<hz>`. The daemon
  runs a [symbolscope](plots.html) engine with its FFT taps enabled.
- Each frame carries two FFT-shifted dBFS arrays: `raw_bins` (the DDC
  output) and `tuned_bins` (that window multiplied by
  `exp(-j·2π·f_off·n/Fs)`, where `f_off` is the receiver's own
  `carrier_offset_hz`). Reconstructing the tuned view from the loop's
  estimate needs no extra receiver tap and works identically for C4FM and
  CQPSK.
- The FFT math is the same `spectrum.PowerDB` helper the wideband
  [Spectrum](spectrum.html) waterfall uses, so the two views are
  normalized consistently.

The numeric side of the same loop state — the carrier-error trend, AGC
and clock meters — lives on the [Tuning panel](tuning.html); the Mixer
plot is its graphical FFT counterpart.

## Implementation

| Path | Role |
| --- | --- |
| `internal/dsp/spectrum/spectrum.go` | `PowerDB` — shared windowed-FFT dBFS helper |
| `internal/scanner/symbolscope/mixer.go` | Baseband accumulator → raw/tuned FFT frames |
| `internal/scanner/symbolscope/scope.go` | Feeds the accumulator with the channelized baseband + carrier estimate |
| `internal/api/mixer.go` | `MixerProvider` interface + `WS /api/v1/diag/mixer` handler |
| `cmd/gophertrunk/mixer_provider.go` | Daemon provider over the iqtap broker |
| `web/src/panels/Mixer.tsx` | Two stacked FFT line plots (raw + tuned) |
