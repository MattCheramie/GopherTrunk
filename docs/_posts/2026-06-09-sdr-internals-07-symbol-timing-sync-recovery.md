---
title: "SDR in Pure Go, Part 7: Symbol Timing & Sync Recovery"
description: How GopherTrunk recovers the symbol clock from oversampled signals with a Mueller-Muller timing loop and finds frame boundaries with a sync correlator — feedback state machines implemented in pure Go.
category: deep-dives
tags: [sdr, go, dsp, clock-recovery, synchronization, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 7
---

*Part 7 of **SDR Internals**. The demodulator gives us soft symbols, but at the
wrong rate and out of phase. This post is about finding the exact instant to
sample each symbol, and the frame boundary that gives those symbols meaning.*

## In this post

- Why **clock recovery** is necessary even after demodulation.
- The **Mueller-Muller** timing loop and the **sync correlator**.
- The **feedback-state-machine** design that carries sub-sample phase across
  every chunk.

## What timing and sync recovery do

The transmitter and receiver don't share a clock. After demodulation you have,
say, 10 samples per symbol, but the *best* sampling instant drifts continuously
as the two clocks slide against each other. **Symbol timing recovery** is a
feedback loop that tracks that optimal instant and outputs exactly one sample per
symbol. ([clock recovery]({{ '/reference/clock-recovery/' | relative_url }}),
[eye diagram]({{ '/reference/eye-diagram/' | relative_url }}))

<figure class="lab-figure">
<svg viewBox="0 0 660 180" width="660" height="180" role="img" aria-label="The Mueller-Muller symbol-timing recovery loop: oversampled input feeds an interpolator, whose output goes to a timing-error detector, then a loop filter; the filtered error drives an NCO phase accumulator that feeds the fractional sample phase mu back into the interpolator, and one symbol per period is emitted.">
  <rect x="8" y="30" width="96" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="56" y="48" text-anchor="middle" fill="currentColor" font-size="10">oversampled</text>
  <text x="56" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">~10 sps</text>
  <line x1="104" y1="51" x2="122" y2="51" stroke="currentColor"/><polygon points="122,47 132,51 122,55" fill="currentColor"/>
  <rect x="132" y="30" width="96" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="180" y="48" text-anchor="middle" fill="var(--accent)" font-size="11">interpolator</text>
  <text x="180" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reads at mu</text>
  <line x1="228" y1="51" x2="246" y2="51" stroke="currentColor"/><polygon points="246,47 256,51 246,55" fill="currentColor"/>
  <rect x="256" y="30" width="124" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="318" y="48" text-anchor="middle" fill="currentColor" font-size="10">timing-error detector</text>
  <text x="318" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Mueller-Muller</text>
  <line x1="380" y1="51" x2="398" y2="51" stroke="currentColor"/><polygon points="398,47 408,51 398,55" fill="currentColor"/>
  <rect x="408" y="30" width="96" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="456" y="55" text-anchor="middle" fill="currentColor" font-size="10">loop filter</text>
  <line x1="504" y1="51" x2="522" y2="51" stroke="currentColor"/><polygon points="522,47 532,51 522,55" fill="currentColor"/>
  <rect x="532" y="30" width="112" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="588" y="48" text-anchor="middle" fill="currentColor" font-size="10">one symbol</text>
  <text x="588" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">per period</text>
  <rect x="256" y="118" width="124" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="318" y="136" text-anchor="middle" fill="currentColor" font-size="10">NCO / phase acc.</text>
  <text x="318" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">carries mu across chunks</text>
  <line x1="456" y1="72" x2="456" y2="138" stroke="currentColor"/><line x1="456" y1="138" x2="382" y2="138" stroke="currentColor"/><polygon points="386,134 380,138 386,142" fill="currentColor"/>
  <line x1="256" y1="138" x2="180" y2="138" stroke="currentColor"/><line x1="180" y1="138" x2="180" y2="72" stroke="currentColor"/><polygon points="176,80 180,72 184,80" fill="currentColor"/>
  <text x="205" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">mu feedback</text>
</svg>
<figcaption>The timing loop as feedback state machine: interpolate &#8594; timing-error detector &#8594; loop filter &#8594; NCO, with the fractional phase <code>mu</code> fed back and carried across every chunk.</figcaption>
</figure>

Even with perfect symbols you still don't know where a *frame* begins. **Sync
recovery** slides a known pattern (a sync word) across the symbol stream and
fires when it correlates strongly — that's your frame boundary.

The learn-path lesson
[Clock recovery & symbol timing]({{ '/learn/rf-sdr/clock-recovery/' | relative_url }})
has the intuition; this post is the implementation.

## How GopherTrunk implements it in Go

`internal/dsp/sync` holds both pieces.

**Mueller-Muller** is a decision-directed timing loop for real PAM signals (all
the C4FM protocols). It maintains a fractional sample phase `mu` and nudges it
each symbol to minimize a timing-error term, interpolating to read the signal
*between* samples:

```go
// internal/dsp/sync — Mueller-Muller (shape)
type MuellerMuller struct {
    sps   float64 // samples per symbol
    mu    float64 // sub-sample phase, carried across chunks
    last  float32 // previous symbol decision
    gain  float64 // loop gain
}

func (m *MuellerMuller) Process(dst, in []float32) []float32 { /* ... */ }
```

Because `mu` and `last` live in the struct, the loop is **continuous across chunk
boundaries** — feed it 6 ms at a time and it tracks the clock as if it saw the
whole signal at once. ([reference]({{ '/reference/mueller-muller-timing-recovery/' | relative_url }}))

**The correlator** is a sliding inner-product matcher. Give it a sync pattern; it
reports a correlation strength at each position, and the protocol layer declares
frame sync when the score crosses a threshold. The same primitive finds the P25
NID, the DMR burst sync, and the NXDN frame sync — only the pattern changes.

A typical receiver chains them: `FM → matched filter → MuellerMuller → slicer →
correlator → dibits`, exactly as the NXDN receiver does in
`internal/radio/nxdn/receiver`.

<figure class="lab-figure">
<svg viewBox="0 0 660 190" width="660" height="190" role="img" aria-label="An eye diagram: overlaid symbol transitions form an open eye; a dashed vertical accent line marks the optimal sampling instant at mu where the eye is widest, with decision dots at the high and low symbol levels and a muted horizontal decision threshold.">
  <line x1="40" y1="25" x2="40" y2="160" stroke="var(--fg-muted)"/>
  <line x1="40" y1="160" x2="620" y2="160" stroke="var(--fg-muted)"/>
  <line x1="40" y1="85" x2="620" y2="85" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="34" y="48" text-anchor="end" fill="var(--fg-muted)" font-size="9">+sym</text>
  <text x="34" y="132" text-anchor="end" fill="var(--fg-muted)" font-size="9">&#8722;sym</text>
  <text x="34" y="89" text-anchor="end" fill="var(--fg-muted)" font-size="8">thr</text>
  <path d="M40,85 Q80,55 120,85" fill="none" stroke="currentColor"/>
  <path d="M120,85 Q210,42 300,85" fill="none" stroke="currentColor"/>
  <path d="M120,85 Q210,128 300,85" fill="none" stroke="currentColor"/>
  <path d="M300,85 Q390,42 480,85" fill="none" stroke="currentColor"/>
  <path d="M300,85 Q390,128 480,85" fill="none" stroke="currentColor"/>
  <path d="M480,85 Q560,55 600,85" fill="none" stroke="currentColor"/>
  <line x1="210" y1="28" x2="210" y2="150" stroke="var(--accent)" stroke-dasharray="5 3"/>
  <circle cx="210" cy="52" r="4" fill="var(--accent)"/>
  <circle cx="210" cy="118" r="4" fill="var(--accent)"/>
  <text x="210" y="20" text-anchor="middle" fill="var(--accent)" font-size="10">sample at mu</text>
  <line x1="120" y1="172" x2="300" y2="172" stroke="var(--fg-muted)"/>
  <line x1="120" y1="168" x2="120" y2="176" stroke="var(--fg-muted)"/>
  <line x1="300" y1="168" x2="300" y2="176" stroke="var(--fg-muted)"/>
  <text x="210" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one symbol period</text>
</svg>
<figcaption>The eye diagram the loop is chasing: the timing-error detector nudges <code>mu</code> toward the instant of maximum eye opening, where the margin between symbol levels is greatest.</figcaption>
</figure>

## The design principle: feedback state machines

Both timing recovery and sync detection are **feedback state machines**: they
hold an evolving internal estimate (clock phase, correlation window) and update
it from each new sample. They can't be pure functions — the whole point is
memory of the past.

### How that principle shaped the Go code

- **State is the object.** `MuellerMuller` *is* its phase accumulator and loop
  state. There's no global timing context; each instance owns one signal's clock.
- **Chunk-independence is a hard requirement.** Because IQ arrives in chunks
  ([Part 3]({{ '/blog/deep-dives/sdr-internals-03-sdr-pool-streaming-concurrency/' | relative_url }})),
  the loop must produce identical output regardless of chunk size. Keeping `mu`
  in the struct guarantees that — the boundary is invisible.
- **One owner, no locks.** Like the other DSP primitives, a timing loop is driven
  by exactly one goroutine, so its mutable state needs no synchronization.
- **Separation of concerns.** The loop recovers *timing*; the slicer makes
  *decisions*; the correlator finds *frames*. Each is a separate small type, so a
  protocol can mix and match (e.g., a different correlator pattern with the same
  timing loop).

## Where this goes next

Timing recovery is one of the richest topics in DSP — loop-gain tuning, Gardner
vs. Mueller-Muller ([reference]({{ '/reference/gardner-timing-recovery/' | relative_url }})),
interpolation methods, and lock-acquisition behavior all deserve their own
treatment. A future deep dive will plot eye diagrams as the loop converges. Next,
recovered symbols meet the FEC that makes them trustworthy.

## FAQ

**Why can't we just sample every Nth sample?**
Because the transmitter's clock and yours drift apart continuously. A fixed
decimation would slowly walk off the optimal instant and the error rate would
climb. A feedback loop tracks the drift in real time.

**What's the difference between timing recovery and frame sync?**
Timing recovery decides *when* within a symbol period to sample. Frame sync
decides *where* in the symbol stream a message starts. You need both: clean
symbols, correctly grouped.

**Why Mueller-Muller specifically?**
It's a decision-directed loop that works well on real-valued PAM signals like
C4FM and needs only one sample per symbol at steady state, which keeps it cheap.

## Series navigation

**Part 7 of 14** · ←
[Part 6]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
· Next →
[Part 8: Equalization, diversity & FFT]({{ '/blog/deep-dives/sdr-internals-08-equalization-diversity-fft/' | relative_url }})
