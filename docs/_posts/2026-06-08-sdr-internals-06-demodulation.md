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
[The demodulation pipeline]({{ '/learn/rf-sdr/demodulation-pipeline/' | relative_url }})
gives the visual intuition.

<figure class="lab-figure">
<svg viewBox="0 0 650 110" width="650" height="110" role="img" aria-label="The shared demodulation pipeline: baseband IQ enters an FM discriminator front-end, whose discriminated signal passes through a matched filter, then a 4-level slicer, producing soft symbols on the plus or minus three, plus or minus one alphabet.">
  <rect x="8" y="34" width="108" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="62" y="54" text-anchor="middle" fill="currentColor" font-size="11">IQ samples</text>
  <text x="62" y="68" text-anchor="middle" fill="var(--fg-muted)" font-size="8">baseband</text>
  <line x1="116" y1="56" x2="134" y2="56" stroke="currentColor"/><polygon points="134,52 144,56 134,60" fill="currentColor"/>
  <rect x="136" y="34" width="108" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="190" y="54" text-anchor="middle" fill="var(--accent)" font-size="11">FM discriminator</text>
  <text x="190" y="68" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shared front-end</text>
  <line x1="244" y1="56" x2="262" y2="56" stroke="currentColor"/><polygon points="262,52 272,56 262,60" fill="currentColor"/>
  <rect x="264" y="34" width="108" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="318" y="54" text-anchor="middle" fill="currentColor" font-size="11">matched filter</text>
  <text x="318" y="68" text-anchor="middle" fill="var(--fg-muted)" font-size="8">RRC / Gaussian</text>
  <line x1="372" y1="56" x2="390" y2="56" stroke="currentColor"/><polygon points="390,52 400,56 390,60" fill="currentColor"/>
  <rect x="392" y="34" width="108" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="446" y="54" text-anchor="middle" fill="currentColor" font-size="11">4-level slicer</text>
  <text x="446" y="68" text-anchor="middle" fill="var(--fg-muted)" font-size="8">to symbol alphabet</text>
  <line x1="500" y1="56" x2="518" y2="56" stroke="currentColor"/><polygon points="518,52 528,56 518,60" fill="currentColor"/>
  <rect x="530" y="34" width="112" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="586" y="54" text-anchor="middle" fill="currentColor" font-size="11">soft symbols</text>
  <text x="586" y="68" text-anchor="middle" fill="var(--fg-muted)" font-size="8">&#177;3, &#177;1</text>
</svg>
<figcaption>The C4FM demod pipeline: IQ through the shared FM discriminator, a root-raised-cosine matched filter, and a 4-level slicer to soft symbols &#8212; each stage a Part 4 primitive.</figcaption>
</figure>

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

<figure class="lab-figure">
<svg viewBox="0 0 660 196" width="660" height="196" role="img" aria-label="One shared FM discriminator front-end fans out to four matched-filter-and-slicer variants: C4FM with a root-raised-cosine filter and 4-level slicer, GFSK with a Gaussian matched filter, FFSK for audio-band signaling, and pi-over-4 DQPSK for TETRA.">
  <rect x="8" y="80" width="70" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="43" y="102" text-anchor="middle" fill="currentColor" font-size="10">IQ</text>
  <line x1="78" y1="98" x2="96" y2="98" stroke="currentColor"/><polygon points="96,94 106,98 96,102" fill="currentColor"/>
  <rect x="106" y="74" width="120" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="166" y="94" text-anchor="middle" fill="var(--accent)" font-size="11">FM discriminator</text>
  <text x="166" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">shared front-end</text>
  <line x1="226" y1="90" x2="300" y2="30" stroke="currentColor"/><polygon points="296,26 306,29 300,38" fill="currentColor"/>
  <line x1="226" y1="96" x2="300" y2="76" stroke="currentColor"/><polygon points="296,72 306,75 299,84" fill="currentColor"/>
  <line x1="226" y1="104" x2="300" y2="122" stroke="currentColor"/><polygon points="299,116 306,124 296,125" fill="currentColor"/>
  <line x1="226" y1="110" x2="300" y2="168" stroke="currentColor"/><polygon points="298,161 306,170 294,169" fill="currentColor"/>
  <rect x="306" y="14" width="340" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="318" y="34" fill="currentColor" font-size="10">C4FM &#8212; RRC matched filter + 4-level slicer (P25/DMR/NXDN)</text>
  <rect x="306" y="60" width="340" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="318" y="80" fill="currentColor" font-size="10">GFSK &#8212; Gaussian matched filter (EDACS)</text>
  <rect x="306" y="106" width="340" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="318" y="126" fill="currentColor" font-size="10">FFSK &#8212; audio-band FSK (MPT 1327, POCSAG)</text>
  <rect x="306" y="152" width="340" height="32" rx="5" fill="none" stroke="currentColor"/>
  <text x="318" y="172" fill="currentColor" font-size="10">&#960;/4-DQPSK &#8212; differential QPSK (TETRA)</text>
</svg>
<figcaption>The demod family: one FM discriminator feeds four matched-filter-and-slicer variants, each a small single-responsibility type composed from shared primitives.</figcaption>
</figure>

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
