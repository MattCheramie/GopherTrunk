---
slug: fir-filter
title: FIR filter
entry_type: algorithm
category: filtering-multirate
description: A finite impulse response (FIR) filter sums weighted recent input samples — the workhorse SDR digital filter, always stable and capable of exactly linear phase.
keywords: FIR filter, finite impulse response, tapped delay line, linear phase, digital filter, convolution, windowing, Parks-McClellan, Remez, polyphase, taps
aka: [FIR, "finite impulse response filter"]
autolink: true
infobox:
  - { label: Type, value: Non-recursive digital filter }
  - { label: Feature, value: Always stable, exact linear phase }
  - { label: Design, value: Windowing / Parks–McClellan }
see_also: [iir-filter, window-function, polyphase-filter-bank, overlap-add-overlap-save, digital-filter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Finite_impulse_response
  - https://en.wikipedia.org/wiki/Parks%E2%80%93McClellan_filter_design_algorithm
---

A **FIR** (**finite impulse response**) filter produces each output sample as a
**weighted sum of the most recent input samples** — a tapped delay line multiplied by a
fixed set of coefficients (taps) and added up.[^wiki] It is the most common
[digital filter](/reference/digital-filter/) in [SDR](/reference/software-defined-radio/)
because, having no feedback, it is **unconditionally stable** and can be made to have
**exactly linear phase** — a flat group delay that preserves pulse shapes, which matters a
great deal for digital modulation where waveform timing carries the data.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A tapped delay line: the input passes through a chain of unit delays; each tap is multiplied by a coefficient and all products are summed to form the output." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none"><rect x="40" y="28" width="40" height="26"/><rect x="100" y="28" width="40" height="26"/><rect x="160" y="28" width="40" height="26"/><rect x="220" y="28" width="40" height="26"/></g>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="60" y="46">z⁻¹</text><text x="120" y="46">z⁻¹</text><text x="180" y="46">z⁻¹</text><text x="240" y="46">z⁻¹</text></g>
  <g stroke="currentColor" stroke-width="1"><line x1="60" y1="54" x2="60" y2="84"/><line x1="120" y1="54" x2="120" y2="84"/><line x1="180" y1="54" x2="180" y2="84"/><line x1="240" y1="54" x2="240" y2="84"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="60" y="97">×a₀</text><text x="120" y="97">×a₁</text><text x="180" y="97">×a₂</text><text x="240" y="97">×a₃</text></g>
  <circle cx="300" cy="92" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="300" y="96" text-anchor="middle" font-size="11" fill="currentColor">Σ</text>
  <g stroke="currentColor" stroke-width="1"><line x1="60" y1="100" x2="289" y2="92"/><line x1="120" y1="100" x2="289" y2="92"/><line x1="180" y1="100" x2="289" y2="92"/><line x1="240" y1="100" x2="289" y2="92"/></g>
  <line x1="312" y1="92" x2="360" y2="92" stroke="currentColor" marker-end="url(#firar)"/><text x="382" y="96" font-size="9" fill="currentColor">out</text>
  <text x="20" y="20" font-size="8" fill="currentColor">input →</text>
  <defs><marker id="firar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A FIR filter delays the input, scales each tap by a coefficient, and sums the products — a direct implementation of convolution with the filter's impulse response.</figcaption>
</figure>

## How it works

The output is the **convolution** of the input with the tap set: `y[n] = Σ aₖ·x[n−k]`. The
coefficients *are* the filter's impulse response, and their discrete Fourier transform is its
frequency response — so choosing taps is choosing exactly which frequencies pass and which
are rejected. There is no recursion and no internal state beyond the delay line, so a
bounded input always yields a bounded output: a FIR can never oscillate or blow up.

Its signature property is **linear phase**. If the taps are symmetric (or antisymmetric)
about their centre, every frequency is delayed by the same amount, so the filter shifts the
whole signal in time without distorting its shape. That is essential for pulse-shaped digital
signals, where a frequency-dependent delay would smear symbols into one another. The price is
cost: a sharp transition band needs many taps, and each output costs one multiply-accumulate
per tap, so a selective FIR is far more arithmetic than an equivalent
[IIR filter](/reference/iir-filter/).

## Variants: designing and speeding up FIRs

- **Windowing.** Start from the ideal (infinite) impulse response — e.g. a `sinc` for a
  brick-wall low-pass — truncate it, and taper the ends with a [window
  function](/reference/window-function/) (Hamming, Blackman, Kaiser) to trade main-lobe width
  against stop-band ripple. Simple and intuitive.
- **Parks–McClellan (Remez / equiripple).** An optimal algorithm that spreads the approximation
  error evenly across the band, giving the shortest FIR that meets a given ripple and
  transition spec — the standard tool for demanding channel filters.[^pm]
- **Polyphase decomposition.** When a FIR is combined with rate change, splitting its taps
  into sub-filters lets it skip the arithmetic for samples that will be discarded, forming the
  [polyphase filter bank](/reference/polyphase-filter-bank/) at the heart of efficient
  decimators, interpolators, and channelizers.
- **Fast convolution.** For very long FIRs, block-processing methods such as
  [overlap-add / overlap-save](/reference/overlap-add-overlap-save/) perform the convolution
  through the FFT, cutting the cost per output sample dramatically.

## In practice

FIR filters do the selective work everywhere in an SDR chain: channel selection, anti-alias
filtering before [decimation](/reference/decimation/), receive pulse-shaping, and CIC droop
compensation. When phase linearity and stability matter more than raw efficiency — which in a
demodulator they usually do — the FIR is the default choice.

## Relevance to SDR

FIR filtering is pervasive in GopherTrunk's DSP: the channelizer's channel-select and
anti-alias stages, half-band decimation, and receive matched/pulse-shaping filters are all
FIRs, chosen for their linear phase and guaranteed stability. Their polyphase forms are what
make simultaneous multi-channel decoding affordable on a CPU.

## Sources

[^wiki]: [Finite impulse response](https://en.wikipedia.org/wiki/Finite_impulse_response) — Wikipedia, on the non-recursive, always-stable filter, its convolution form, and linear phase.
[^pm]: [Parks–McClellan filter design algorithm](https://en.wikipedia.org/wiki/Parks%E2%80%93McClellan_filter_design_algorithm) — Wikipedia, on the equiripple/Remez method for optimal FIR design.
