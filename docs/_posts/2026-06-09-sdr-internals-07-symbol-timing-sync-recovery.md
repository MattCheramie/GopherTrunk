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
