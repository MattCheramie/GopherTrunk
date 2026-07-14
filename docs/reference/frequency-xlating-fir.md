---
slug: frequency-xlating-fir
title: Frequency-translating FIR filter (xlating FIR)
entry_type: algorithm
category: sdr-app-building
description: A frequency-translating FIR filter combines mixing, low-pass filtering, and decimation into one block that selects and channelises a single channel from a wideband IQ stream.
keywords: frequency xlating fir, frequency translating fir filter, xlating fir, channel select, mix filter decimate, digital down converter, GNU Radio xlating filter, channelizer, decimating fir
aka: [xlating FIR, frequency xlating FIR filter, translating FIR]
autolink: true
infobox:
  - { label: Type, value: DSP algorithm }
  - { label: Combines, value: Mix + FIR filter + decimate }
  - { label: Used for, value: Single-channel select }
see_also: [digital-down-converter, fir-filter, decimation, channelizer, numerically-controlled-oscillator, baseband]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_down_converter
  - https://en.wikipedia.org/wiki/Finite_impulse_response
---

A **frequency-translating FIR filter** (an *xlating FIR*) does three jobs in one block: it
mixes a chosen channel down to [baseband](/reference/baseband/), low-pass filters it with an
[FIR](/reference/fir-filter/), and [decimates](/reference/decimation/) to a lower sample
rate.[^ddc] It is the workhorse of channel selection in an SDR receiver — the compact,
efficient realisation of a [digital down converter](/reference/digital-down-converter/) that
picks one narrow channel out of a wide IQ capture.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A wideband IQ stream is multiplied by a complex oscillator, filtered by a FIR, and decimated, producing one baseband channel; the trick is folding the mixer into the filter taps." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="xfirar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="8" y="52" font-size="7" fill="currentColor">wide IQ</text>
  <circle cx="80" cy="50" r="12" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="53" font-size="9" fill="currentColor" text-anchor="middle">×</text>
  <path d="M80 90 q6 -10 12 0 q6 -10 12 0" fill="none" stroke="currentColor" stroke-width="1"/><text x="96" y="108" font-size="6.5" fill="currentColor" text-anchor="middle">e^-jωn</text>
  <line x1="80" y1="78" x2="80" y2="62" stroke="currentColor" stroke-width="1" marker-end="url(#xfirar)"/>
  <line x1="48" y1="50" x2="67" y2="50" stroke="currentColor" stroke-width="1" marker-end="url(#xfirar)"/>
  <rect x="120" y="38" width="70" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="155" y="53" font-size="8" fill="currentColor" text-anchor="middle">FIR LPF</text>
  <line x1="92" y1="50" x2="119" y2="50" stroke="currentColor" stroke-width="1" marker-end="url(#xfirar)"/>
  <rect x="210" y="38" width="60" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="240" y="53" font-size="8" fill="currentColor" text-anchor="middle">↓ M</text>
  <line x1="190" y1="50" x2="209" y2="50" stroke="currentColor" stroke-width="1" marker-end="url(#xfirar)"/>
  <line x1="270" y1="50" x2="330" y2="50" stroke="currentColor" stroke-width="1" marker-end="url(#xfirar)"/>
  <text x="360" y="53" font-size="7" fill="currentColor">1 channel</text>
  <rect x="112" y="30" width="166" height="42" rx="5" fill="none" stroke="currentColor" stroke-width="0.7" stroke-dasharray="3 3"/>
  <text x="195" y="86" font-size="6.5" fill="currentColor" text-anchor="middle">mixer foldable into taps</text>
</svg>
<figcaption>Mix, filter, and decimate fused into one block; the mixer can be absorbed into the FIR coefficients.</figcaption>
</figure>

## How it works

Naively, three operations run in sequence. The input `x[n]` is multiplied by a complex
exponential `e^{-jω₀n}` from a [numerically-controlled oscillator](/reference/numerically-controlled-oscillator/)
to slide the channel at frequency `ω₀` down to 0 Hz; the result passes through a low-pass FIR
that keeps only the wanted channel's bandwidth; and the filtered stream is decimated by `M`,
keeping one sample in `M` since the bandwidth is now small enough that a lower rate is legal.

The algorithm's value is that these can be *fused*. Multiplying before the filter is the same
as filtering with **frequency-shifted taps**: pre-rotate the FIR coefficients by `e^{jω₀k}`,
and a single complex FIR both mixes and filters. Better still, because decimation throws away
`M−1` of every `M` outputs, a decimating FIR never computes them — the mix-filter-decimate
block costs only as much as the output rate demands, not the input rate. That efficiency is
why it is the standard channel-select primitive.

## Variants

- **Rotate-input vs rotate-taps.** Rotating the input needs a running oscillator but keeps the
  taps real; rotating the taps bakes the frequency in at design time (fixed offset only).
- **Polyphase decimating form.** Reorganising the taps into a
  [polyphase](/reference/polyphase-filter-bank/) structure computes only the retained outputs,
  the efficient realisation used in practice.
- **Bank of xlating FIRs → channelizer.** Running many at different `ω₀` extracts several
  channels at once; when the offsets are uniform this becomes an FFT-based
  [channelizer](/reference/channelizer/) that shares work across all channels.

## Relevance to SDR

The xlating FIR is how GNU Radio's `freq_xlating_fir_filter` and most SDR channelisers tune
within a captured band — you set a centre offset and a decimation, and it hands back one
baseband channel. It is the natural first block of a [receiver chain](/reference/receiver-chain/).

**GopherTrunk** applies exactly this idea, though its two down-converters realise it
differently. The single-channel `Downconverter` in `internal/scanner/ccdecoder/ddc.go` mixes
with an NCO and decimates to a fixed channel rate (48 kHz, or 144 kHz for TETRA), while the
wideband `DDCBank` in `internal/dsp/tuner` extracts many taps at once — the multi-channel,
channelizer-style form of the same mix-filter-decimate operation. Both are the frequency-
translating FIR at work, selecting a channel out of a wide capture before demodulation.

## Sources

[^ddc]: [Digital down converter](https://en.wikipedia.org/wiki/Digital_down_converter) — Wikipedia, on combining a complex mixer, low-pass filter, and decimator to select a channel.
[^fir]: [Finite impulse response](https://en.wikipedia.org/wiki/Finite_impulse_response) — Wikipedia, on the FIR filter whose taps can be frequency-shifted to absorb the mixer.
