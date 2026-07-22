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
  [waterfall]({{ '/learn/rf-sdr/fft-and-waterfall/' | relative_url }}) and for carrier
  detection.

## How GopherTrunk implements it in Go

**Equalizers** live in `internal/dsp/equalizer`: `LMS` (trained against known
reference symbols), `CMA` (blind — drives the output magnitude toward a constant,
no reference needed), and a fractionally-spaced variant for sub-symbol taps. Each
is an adaptive FIR updated by stochastic gradient descent. Crucially, an
equalizer **wraps** the demod chain — it sits between decimation and the
discriminator and improves the symbols without the demodulator knowing it's
there.

<figure class="lab-figure">
<svg viewBox="0 0 660 180" width="660" height="180" role="img" aria-label="An adaptive equalizer inserted as a decorator between decimation and the discriminator: decimated IQ passes through the equalizer's adaptive FIR taps to the discriminator and out as soft symbols; an error signal drives an LMS or CMA update that adjusts the tap weights each sample.">
  <rect x="8" y="30" width="104" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="60" y="55" text-anchor="middle" fill="currentColor" font-size="10">decimation</text>
  <line x1="112" y1="51" x2="130" y2="51" stroke="currentColor"/><polygon points="130,47 140,51 130,55" fill="currentColor"/>
  <rect x="140" y="24" width="170" height="54" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="225" y="45" text-anchor="middle" fill="var(--accent)" font-size="11">equalizer</text>
  <text x="225" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="8">adaptive FIR: w0 w1 w2 w3</text>
  <line x1="310" y1="51" x2="328" y2="51" stroke="currentColor"/><polygon points="328,47 338,51 328,55" fill="currentColor"/>
  <rect x="338" y="30" width="120" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="398" y="55" text-anchor="middle" fill="currentColor" font-size="10">discriminator</text>
  <line x1="458" y1="51" x2="476" y2="51" stroke="currentColor"/><polygon points="476,47 486,51 476,55" fill="currentColor"/>
  <rect x="486" y="30" width="120" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="546" y="55" text-anchor="middle" fill="currentColor" font-size="10">soft symbols</text>
  <rect x="140" y="120" width="170" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="225" y="139" text-anchor="middle" fill="var(--accent)" font-size="10">adapt taps</text>
  <text x="225" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="8">LMS / CMA (SGD)</text>
  <line x1="546" y1="72" x2="546" y2="139" stroke="currentColor"/><line x1="546" y1="139" x2="312" y2="139" stroke="currentColor"/><polygon points="316,135 310,139 316,143" fill="currentColor"/>
  <text x="430" y="133" text-anchor="middle" fill="var(--fg-muted)" font-size="9">error signal</text>
  <line x1="225" y1="120" x2="225" y2="78" stroke="currentColor"/><polygon points="221,86 225,78 229,86" fill="currentColor"/>
  <text x="262" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">new weights</text>
</svg>
<figcaption>The equalizer as a decorator: it wraps the chain between decimation and the discriminator, and a per-sample LMS/CMA update adapts its FIR taps to reverse the channel &#8212; transparent to the demodulator.</figcaption>
</figure>

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

<figure class="lab-figure">
<svg viewBox="0 0 660 180" width="660" height="180" role="img" aria-label="A power spectrum from the rate-limited FFT producer: dBFS on the vertical axis, frequency bins across the bottom, a flat muted noise floor of short bars, and one tall accent bar marking a detected carrier peak; a label notes the ten frames per second cap.">
  <line x1="56" y1="24" x2="56" y2="150" stroke="var(--fg-muted)"/>
  <line x1="56" y1="150" x2="612" y2="150" stroke="var(--fg-muted)"/>
  <text x="50" y="30" text-anchor="end" fill="var(--fg-muted)" font-size="8">0 dBFS</text>
  <text x="50" y="150" text-anchor="end" fill="var(--fg-muted)" font-size="8">&#8722;80</text>
  <text x="334" y="170" text-anchor="middle" fill="var(--fg-muted)" font-size="9">frequency bins &#8594;</text>
  <rect x="66" y="136" width="16" height="14" fill="var(--fg-muted)"/>
  <rect x="90" y="130" width="16" height="20" fill="var(--fg-muted)"/>
  <rect x="114" y="138" width="16" height="12" fill="var(--fg-muted)"/>
  <rect x="138" y="128" width="16" height="22" fill="var(--fg-muted)"/>
  <rect x="162" y="134" width="16" height="16" fill="var(--fg-muted)"/>
  <rect x="186" y="126" width="16" height="24" fill="var(--fg-muted)"/>
  <rect x="210" y="132" width="16" height="18" fill="var(--fg-muted)"/>
  <rect x="234" y="122" width="16" height="28" fill="var(--fg-muted)"/>
  <rect x="258" y="112" width="16" height="38" fill="var(--fg-muted)"/>
  <rect x="282" y="44" width="16" height="106" fill="var(--accent)"/>
  <rect x="306" y="108" width="16" height="42" fill="var(--fg-muted)"/>
  <rect x="330" y="130" width="16" height="20" fill="var(--fg-muted)"/>
  <rect x="354" y="136" width="16" height="14" fill="var(--fg-muted)"/>
  <rect x="378" y="128" width="16" height="22" fill="var(--fg-muted)"/>
  <rect x="402" y="134" width="16" height="16" fill="var(--fg-muted)"/>
  <rect x="426" y="138" width="16" height="12" fill="var(--fg-muted)"/>
  <rect x="450" y="130" width="16" height="20" fill="var(--fg-muted)"/>
  <rect x="474" y="135" width="16" height="15" fill="var(--fg-muted)"/>
  <rect x="498" y="132" width="16" height="18" fill="var(--fg-muted)"/>
  <rect x="522" y="138" width="16" height="12" fill="var(--fg-muted)"/>
  <rect x="546" y="134" width="16" height="16" fill="var(--fg-muted)"/>
  <rect x="570" y="139" width="16" height="11" fill="var(--fg-muted)"/>
  <line x1="290" y1="44" x2="290" y2="30" stroke="var(--accent)"/>
  <text x="290" y="24" text-anchor="middle" fill="var(--accent)" font-size="10">carrier</text>
  <text x="604" y="34" text-anchor="end" fill="var(--fg-muted)" font-size="9">10 fps cap</text>
</svg>
<figcaption>One dBFS <code>Frame</code> from the rate-limited spectrum producer: a flat noise floor with a single accent carrier peak &#8212; the same frames that feed the waterfall, TUI panel, and carrier detector.</figcaption>
</figure>

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
