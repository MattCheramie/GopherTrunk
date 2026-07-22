---
title: "SDR in Pure Go, Part 4: DSP Foundations — Filters, NCO & AGC"
description: The building blocks of GopherTrunk's signal chain in Go — FIR/CIC filters, a numerically-controlled oscillator, automatic gain control, and resamplers — and the zero-allocation streaming design behind them.
category: deep-dives
tags: [sdr, go, dsp, filters, agc, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 4
---

*Part 4 of **SDR Internals**. Before any protocol can be decoded, the raw IQ has
to be shifted, filtered, leveled, and resampled. These are the DSP primitives in
`internal/dsp` — and the Go design that keeps them fast.*

## In this post

- The four workhorses: **filters, the NCO, AGC, and the resampler**.
- Why every primitive is a **stateful streaming object**, not a pure function.
- The **zero-allocation `Process(dst, src)` convention** that keeps the hot path
  off the garbage collector.

## What the DSP foundation does

Wideband IQ off a dongle is messy: it's centered at the wrong frequency, it
spans far more bandwidth than the signal you want, and its amplitude swings with
the band. Four primitives fix that:

- **NCO (numerically-controlled oscillator)** — a software local oscillator that
  multiplies the IQ by a rotating phasor to shift a signal to baseband.
  ([reference]({{ '/reference/numerically-controlled-oscillator/' | relative_url }}))
- **Filters** — FIR, CIC, and half-band filters that remove everything outside
  the channel and let you decimate the sample rate down.
  ([FIR]({{ '/reference/fir-filter/' | relative_url }}),
  [CIC]({{ '/reference/cic-filter/' | relative_url }}))
- **AGC (automatic gain control)** — normalizes amplitude so downstream slicers
  see a consistent signal level.
  ([reference]({{ '/reference/automatic-gain-control/' | relative_url }}))
- **Resampler** — a polyphase rational L/M resampler that converts between the
  dongle's rate and a protocol's symbol clock.
  ([reference]({{ '/reference/resampler/' | relative_url }}))

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="The four DSP primitives compose left to right into one signal chain: wideband IQ enters an NCO or mixer that shifts it to baseband, then FIR or CIC filters that filter and decimate, then AGC that levels amplitude to a target, then a resampler that converts to the protocol symbol clock, emitting symbols. Every stage uses the same Process destination-source buffer-reuse convention.">
  <text x="340" y="20" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Process(dst, src) — buffers reused every chunk</text>
  <rect x="8" y="52" width="86" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="51" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="10">wideband IQ</text>
  <line x1="94" y1="74" x2="108" y2="74" stroke="currentColor"/><polygon points="108,70 116,74 108,78" fill="currentColor"/>
  <rect x="116" y="52" width="96" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="164" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">NCO / mixer</text>
  <text x="164" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shift to baseband</text>
  <line x1="212" y1="74" x2="226" y2="74" stroke="currentColor"/><polygon points="226,70 234,74 226,78" fill="currentColor"/>
  <rect x="234" y="52" width="96" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="282" y="72" text-anchor="middle" fill="currentColor" font-size="10">FIR / CIC</text>
  <text x="282" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">filter + decimate</text>
  <line x1="330" y1="74" x2="344" y2="74" stroke="currentColor"/><polygon points="344,70 352,74 344,78" fill="currentColor"/>
  <rect x="352" y="52" width="96" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="400" y="72" text-anchor="middle" fill="var(--accent)" font-size="10">AGC</text>
  <text x="400" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">level to target</text>
  <line x1="448" y1="74" x2="462" y2="74" stroke="currentColor"/><polygon points="462,70 470,74 462,78" fill="currentColor"/>
  <rect x="470" y="52" width="120" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="530" y="72" text-anchor="middle" fill="currentColor" font-size="10">resampler</text>
  <text x="530" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="8">L/M to symbol clock</text>
  <line x1="590" y1="74" x2="604" y2="74" stroke="currentColor"/><polygon points="604,70 612,74 604,78" fill="currentColor"/>
  <rect x="612" y="52" width="62" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="643" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="10">symbols</text>
</svg>
<figcaption>The four primitives compose into one chain — mix, filter, level, resample — each a stateful struct driven through the same <code>Process(dst, src)</code> signature so the hot path never allocates.</figcaption>
</figure>

The learn-path lesson
[Filtering & decimation]({{ '/learn/rf-sdr/filtering-decimation/' | relative_url }})
covers the theory; this post is about the code.

## How GopherTrunk implements it in Go

Each primitive is a small struct that **carries state across calls**. A
streaming FIR filter must remember the tail of the previous chunk to filter the
next one seamlessly, so it owns a history ring buffer:

```go
// internal/dsp — shape of a streaming primitive
type FIR struct {
    taps    []float32
    history []float32 // carries the boundary between chunks
    pos     int
}

// Process filters src into dst, reusing dst's backing array.
func (f *FIR) Process(dst, src []float32) []float32 { /* ... */ }
```

The NCO is the same idea with a single piece of state — its phase accumulator —
advanced sample by sample so the phasor stays continuous across chunk
boundaries. The AGC tracks a running magnitude estimate (an exponential moving
average) and scales toward a target level. The resampler keeps its polyphase
filter state between calls.

<figure class="lab-figure">
<svg viewBox="0 0 620 170" width="620" height="170" role="img" aria-label="A signal-level-over-time timeline showing what AGC does. The raw input amplitude sits at noise level, jumps sharply when a transmitter keys up, and later fades. The AGC output is held near a constant target level throughout, with only a brief transient at key-up and at the fade.">
  <line x1="40" y1="20" x2="40" y2="140" stroke="currentColor"/>
  <line x1="40" y1="140" x2="600" y2="140" stroke="currentColor"/>
  <text x="16" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="9" transform="rotate(-90 16 84)">level</text>
  <text x="576" y="156" text-anchor="middle" fill="var(--fg-muted)" font-size="9">time →</text>
  <line x1="40" y1="70" x2="600" y2="70" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="80" y="64" fill="var(--fg-muted)" font-size="9">target level</text>
  <polyline points="50,120 170,120 180,45 300,50 420,48 432,96 590,102" fill="none" stroke="currentColor"/>
  <polyline points="50,74 178,74 188,52 214,72 420,72 432,86 462,72 590,72" fill="none" stroke="var(--accent)"/>
  <text x="185" y="36" text-anchor="middle" fill="var(--fg-muted)" font-size="9">key-up</text>
  <text x="470" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">fade</text>
  <text x="120" y="112" fill="currentColor" font-size="9">input amplitude</text>
  <text x="500" y="64" fill="var(--accent)" font-size="9">AGC output</text>
</svg>
<figcaption>AGC tracks a running magnitude estimate and scales toward a target: the input amplitude swings as a transmitter keys up and fades, but downstream slicers see a level held steady near the target.</figcaption>
</figure>

Factory functions handle the math-heavy design step once, up front:
`LowpassKaiser(n, fc, beta)`, `RootRaisedCosine(sps, span, alpha)`, and
`Gaussian(sps, span, bt)` compute filter taps so the hot path only ever does
multiply-accumulate.

## The design principle: stateful streaming + zero-allocation reuse

Two principles work together here. First, **these are stateful objects, not pure
functions**, because continuous streaming requires memory of the past. Second,
the **`Process(dst, src)` convention reuses caller-supplied buffers** so the
streaming path allocates nothing.

### How that principle shaped the Go code

- **`Process(dst, src) []T`** is the universal signature. The caller passes a
  destination slice; the method fills and returns it, reallocating only if
  capacity is too small. In a tight loop the same buffers are reused millions of
  times and the garbage collector never sees the hot path.
- **History lives inside the object.** Because each filter owns its ring buffer,
  chunk boundaries are invisible — you can feed 6 ms at a time and get identical
  output to feeding the whole signal at once.
- **State means not concurrency-safe — by design.** One stateful primitive is
  driven by exactly one goroutine in the pipeline
  ([Part 3]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }})),
  so no locks are needed inside the DSP at all.
- **Design once, run hot.** Expensive Kaiser/RRC tap computation happens in a
  constructor; the per-sample loop is pure float multiply-add, which Go compiles
  to tight, predictable code.

## Where this goes next

Filters and oscillators are the alphabet; the rest of the DSP series spells words
with them. Channelizers ([Part 5]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }}))
are filters arranged in polyphase banks; demodulators
([Part 6]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }}))
chain a matched filter onto a discriminator. A future **"DSP in Go"** series can
go deep on Kaiser window design and fixed-vs-floating-point trade-offs.

## FAQ

**Why floating-point DSP instead of fixed-point?**
`float32` keeps the code simple and is plenty fast on modern CPUs, while
sidestepping the scaling and overflow bugs that plague fixed-point. The
zero-allocation buffer reuse matters far more for performance than the numeric
type.

**What does AGC actually protect against?**
Slicers and timing loops assume a roughly constant signal level. As a signal
fades or a transmitter keys up, AGC keeps the amplitude near a target so those
later stages keep working without re-tuning thresholds.

**Why is a CIC filter used for decimation?**
A [CIC filter]({{ '/reference/cic-filter/' | relative_url }}) decimates with only
adds and subtracts — no multiplies — which makes it the cheapest way to drop a
high sample rate before a sharper FIR cleans up the passband.

## Series navigation

**Part 4 of 14** · ←
[Part 3]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }})
· Next →
[Part 5: Tuning & channelization]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }})
