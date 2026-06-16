---
title: "SDR in Pure Go, Part 6: Demodulation — FM, C4FM, GFSK & More"
description: How GopherTrunk turns baseband IQ into symbols with pure-Go demodulators for FM, C4FM, GFSK, FFSK, and π/4-DQPSK — each a focused, single-responsibility type composed from DSP primitives.
category: deep-dives
tags: [sdr, go, dsp, demodulation, c4fm, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 6
---

*Part 6 of **SDR Internals**. Demodulation is where a radio wave becomes data.
This post covers the family of demodulators in `internal/dsp/demod` and the
single-responsibility design that keeps each one small and composable.*

## In this post

- What demodulation is and why digital radio needs several flavors.
- The pure-Go demodulators: **FM, C4FM, GFSK, FFSK, π/4-DQPSK**.
- The **single-responsibility principle**: each modulation is one focused type,
  built by composing Part 4 primitives.

## What demodulation does

[Modulation]({{ '/reference/modulation/' | relative_url }}) is how data is
written onto a carrier; demodulation reads it back. Different radio systems use
different schemes, so GopherTrunk needs a small zoo of demodulators:

- **FM** — a quadrature discriminator that recovers instantaneous frequency;
  the basis of analog voice and the front-end for several digital modes.
  ([reference]({{ '/reference/frequency-modulation/' | relative_url }}))
- **C4FM** — 4-level FSK used by P25, DMR, NXDN, dPMR, and YSF: a matched filter
  plus a 4-level slicer. ([reference]({{ '/reference/c4fm/' | relative_url }}))
- **GFSK** — Gaussian FSK (EDACS/GE-Marc).
  ([reference]({{ '/reference/gfsk/' | relative_url }}))
- **FFSK** — fast FSK for audio-band signaling (MPT 1327, POCSAG, AFSK paging).
  ([reference]({{ '/reference/ffsk/' | relative_url }}))
- **π/4-DQPSK** — differential QPSK used by TETRA.
  ([reference]({{ '/reference/pi-4-dqpsk/' | relative_url }}))

The learn-path lesson
[The demodulation pipeline]({{ '/learn/demodulation-pipeline/' | relative_url }})
gives the visual intuition.

## How GopherTrunk implements it in Go

Each demodulator is a self-contained stateful struct that takes IQ (or a
discriminated signal) and emits real-valued soft symbols. The FM discriminator is
the simplest — the angle between consecutive samples:

```go
// internal/dsp/demod — FM discriminator (shape)
func (f *FM) Process(dst []float32, iq []complex64) []float32 {
    for i, z := range iq {
        d := z * cmplx.Conj(f.prev) // phase difference
        dst[i] = float32(math.Atan2(imag(d), real(d)))
        f.prev = z
    }
    return dst
}
```

C4FM builds on top: it runs a
[root-raised-cosine matched filter]({{ '/reference/root-raised-cosine-filter/' | relative_url }})
over the discriminated signal, then a 4-level slicer maps soft values to the
symbol alphabet ±3, ±1. GFSK swaps in a
[Gaussian matched filter]({{ '/reference/matched-filter/' | relative_url }}).
There's also an `AdaptiveC4FM` that adds automatic frequency correction to track
transmitter drift. Each type is a few dozen lines because the hard work — the
filters, the oscillator — was already built in
[Part 4]({{ '/blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/' | relative_url }}).

## The design principle: single responsibility

Every demodulator does exactly one thing: convert a particular modulation into
soft symbols. It does **not** recover the symbol clock, find frame sync, or
decode FEC — those are separate stages with their own posts. This
**single-responsibility** boundary is what keeps the family small and the
pipeline composable.

### How that principle shaped the Go code

- **Small types, sharp edges.** Each demod is a struct with a `Process` method
  and nothing more. You can read, test, and reason about `C4FM` without thinking
  about timing recovery or P25 framing.
- **Composition, not configuration flags.** A C4FM demod is *built from* a
  matched filter and a slicer rather than being a giant function with a "mode"
  switch. New modulations reuse the same primitives in a new arrangement.
- **Testable in isolation.** Because a demod's only job is symbols-in from
  IQ-out, the test suite can modulate a known bit pattern, push it through, and
  assert the symbols come back — table-driven tests like
  `TestGFSKRecoversAlternatingNRZ` do exactly this.
- **Clean handoff.** Each demod outputs the same shape (a slice of soft symbols),
  so the next stage — timing recovery in
  [Part 7]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
  — doesn't care which modulation produced them.

## Where this goes next

Demodulation is deep enough for its own series — the math of the FM
discriminator, why RRC is the matched filter for C4FM, how π/4-DQPSK's
differential trick survives phase ambiguity. For now, the lesson is structural:
one modulation, one small type, composed from shared primitives.

## FAQ

**What's the difference between FM demod and C4FM demod?**
FM demod recovers a continuous signal (analog voice or a raw discriminator
output). C4FM treats that discriminated signal as four discrete frequency levels
and slices it into 2-bit symbols, after a matched filter cleans up the pulse
shape.

**Why is the same FM front-end used for several digital modes?**
C4FM, GFSK, and FFSK are all frequency-shift schemes, so a frequency
discriminator is the common first step. What differs is the matched filter and
the slicer that follow it.

**Do these demodulators recover bits directly?**
No — they output *soft symbols*. Turning those into reliable bits needs symbol
timing recovery and (usually) forward error correction, which are the next two
posts.

## Series navigation

**Part 6 of 14** · ←
[Part 5]({{ '/blog/deep-dives/sdr-internals-05-tuning-channelization/' | relative_url }})
· Next →
[Part 7: Symbol timing & sync recovery]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }})
