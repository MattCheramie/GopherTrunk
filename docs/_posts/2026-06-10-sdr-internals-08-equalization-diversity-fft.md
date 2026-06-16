---
title: "SDR in Pure Go, Part 8: Equalization, Diversity & the FFT"
description: How GopherTrunk fights simulcast distortion with adaptive equalizers, combines multiple receivers with diversity, and drives the live waterfall with a rate-limited FFT — all in pure Go.
category: deep-dives
tags: [sdr, go, dsp, equalizer, fft, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 8
---

*Part 8 of **SDR Internals**. Real-world signals arrive smeared by multipath,
split across antennas, and in need of a spectrum view. This post covers three
DSP refinements — equalization, diversity, and the FFT — and the decorator and
interface-segregation patterns behind them.*

## In this post

- **Adaptive equalizers** (LMS, CMA, fractionally-spaced) that undo simulcast
  smearing.
- **Diversity combining** (MRC, selection) across multiple receivers.
- The **FFT/spectrum** producer that feeds the waterfall.
- The **decorator** and **interface-segregation** principles tying them in.

## What these stages do

- **Equalization.** Simulcast systems transmit the same signal from multiple
  towers; the echoes cause inter-symbol interference that closes the
  [eye]({{ '/reference/eye-diagram/' | relative_url }}). An adaptive equalizer is
  a self-tuning filter that reverses the channel's smearing.
  ([CMA]({{ '/reference/cma-equalizer/' | relative_url }}))
- **Diversity.** With two antennas/dongles on the same signal, you can combine
  them for more SNR than either alone — Maximal-Ratio Combining weights each
  branch by its signal quality.
- **FFT / spectrum.** The
  [fast Fourier transform]({{ '/reference/fast-fourier-transform/' | relative_url }})
  turns IQ into a power spectrum for the live
  [waterfall]({{ '/learn/fft-and-waterfall/' | relative_url }}) and for carrier
  detection.

## How GopherTrunk implements it in Go

**Equalizers** live in `internal/dsp/equalizer`: `LMS` (trained against known
reference symbols), `CMA` (blind — drives the output magnitude toward a constant,
no reference needed), and a fractionally-spaced variant for sub-symbol taps. Each
is an adaptive FIR updated by stochastic gradient descent. Crucially, an
equalizer **wraps** the demod chain — it sits between decimation and the
discriminator and improves the symbols without the demodulator knowing it's
there.

**Diversity** lives in `internal/dsp/diversity`: `MRC` (weighted sum by per-branch
SNR) and `Selection` (pick the strongest branch).

**FFT/spectrum** lives in `internal/dsp/fft` and `internal/dsp/spectrum`. The
spectrum producer windows each block (Hann/Hamming to stop spectral leakage),
runs one FFT, and normalizes to
[dBFS]({{ '/reference/dbfs/' | relative_url }}) — but only at a bounded frame
rate (10 fps by default), so the display never steals CPU from the decoder:

```go
// internal/dsp/spectrum — rate-limited producer (shape)
type Producer struct {
    plan   *fft.Plan
    window []float32
    fps    int      // cap; skip blocks between frames
}

func (p *Producer) Push(iq []complex64) (*Frame, bool) {
    // returns a dBFS Frame only when the next frame is due
}
```

## The design principle: decorator + interface segregation

Two principles do the work here. The equalizer is a **decorator**: it adds
behavior to the signal chain by wrapping a stage, presenting the same
symbols-in/symbols-out shape, so it's optional and transparent. And the FFT
consumers rely on **interface segregation** — the spectrum producer exposes a
*tiny* surface, so each consumer depends on only the slice of functionality it
needs.

### How that principle shaped the Go code

- **Optional by composition.** Because the equalizer wraps the demod and keeps
  the same interface, simulcast handling is enabled by *inserting* a stage, not by
  threading flags through the demodulator. With it absent, the chain is identical
  minus one link.
- **Narrow interfaces, many consumers.** The same FFT frames feed the web
  waterfall, the TUI spectrum panel, and the
  [carrier detector]({{ '/reference/control-channel/' | relative_url }}) used by
  system discovery. Each consumes a minimal `Frame`/producer interface, so none
  is coupled to the others.
- **Best-effort, bounded.** Spectrum frames ride the non-blocking tap broker from
  [Part 3]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }})
  and the producer self-limits to a few frames per second — display quality never
  comes at the cost of decode reliability.
- **Adaptive state stays local.** Each equalizer owns its tap weights and updates
  them per sample, the same one-goroutine-one-state rule as every other DSP
  primitive.

## Where this goes next

Adaptive equalization is genuinely hard and worth its own series — LMS step-size
vs. convergence, why CMA works without a reference, and how fractionally-spaced
taps beat symbol-spaced ones on multipath. Diversity combining and FFT window
design are each deep topics too. With clean symbols in hand, we turn to making
them *correct*: forward error correction.

## FAQ

**What is simulcast distortion?**
When several towers transmit the same signal simultaneously, their slightly
different path delays overlap at your antenna, smearing each symbol into the next.
An equalizer learns and reverses that channel response.

**Why limit the FFT to 10 frames per second?**
A human watching a waterfall can't perceive more, and computing a full FFT on
every IQ block would waste CPU the decoder needs. Rate-limiting keeps the display
smooth and the decode path fast.

**Do I need two SDRs to benefit from this stage?**
Diversity combining needs multiple receivers, but equalization and the FFT work
on a single SDR. Most users run one dongle and still get the equalizer's
simulcast benefit.

## Series navigation

**Part 8 of 14** · ←
[Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
· Next →
[Part 9: Framing & forward error correction]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }})
