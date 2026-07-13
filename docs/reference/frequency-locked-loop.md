---
slug: frequency-locked-loop
title: Frequency-locked loop (FLL)
entry_type: algorithm
category: synchronization
description: A frequency-locked loop tracks a signal's frequency (not its phase) using a frequency discriminator, giving wide pull-in for coarse carrier acquisition before a PLL takes over.
keywords: frequency-locked loop, FLL, frequency discriminator, AFC, coarse frequency acquisition, pull-in range, carrier acquisition, FLL-assisted PLL, dot-product discriminator
aka: [FLL, frequency-lock loop]
autolink: true
infobox:
  - { label: Type, value: Feedback tracking loop }
  - { label: Locks, value: Frequency (not phase) }
  - { label: Used for, value: Coarse carrier acquisition / AFC }
see_also: [phase-locked-loop, automatic-frequency-control, costas-loop, numerically-controlled-oscillator, ppm-frequency-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-locked_loop
  - https://en.wikipedia.org/wiki/Automatic_frequency_control
---

A **frequency-locked loop (FLL)** is a feedback loop that drives an oscillator to
match the **frequency** of an input signal, without caring about its absolute
phase.[^wiki] Where a [phase-locked loop](/reference/phase-locked-loop/) nulls a
*phase* error, an FLL nulls a *frequency* error measured by a frequency
discriminator. Ignoring phase makes the loop far more tolerant of large initial
offsets, so an FLL is the tool of choice for **coarse carrier acquisition** and
[automatic frequency control](/reference/automatic-frequency-control/) before a
narrower PLL is handed the residual.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A loop where a frequency discriminator estimates the frequency error between input and oscillator, a loop filter smooths it, and a controlled oscillator is retuned to close the loop." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fllar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="60" y="42" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="105" y="56">frequency</text><text x="105" y="67">discriminator</text>
    <rect x="196" y="42" width="72" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="232" y="56">loop</text><text x="232" y="67">filter</text>
    <rect x="314" y="42" width="82" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="355" y="56">NCO</text><text x="355" y="67">oscillator</text>
    <line x1="14" y1="59" x2="59" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#fllar)"/><text x="36" y="51">in</text>
    <line x1="150" y1="59" x2="195" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#fllar)"/><text x="173" y="51">Δf</text>
    <line x1="268" y1="59" x2="313" y2="59" stroke="currentColor" stroke-width="1.1" marker-end="url(#fllar)"/>
    <path d="M355 76 V 112 H 105 V 77" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#fllar)"/>
    <text x="235" y="126">frequency feedback</text>
  </g>
</svg>
<figcaption>An FLL measures the rate of phase change (Δf) rather than absolute phase, smooths it, and retunes the oscillator — giving a wide pull-in range that a PLL lacks.</figcaption>
</figure>

## How it works

The key difference from a PLL is the error detector. A **frequency discriminator**
estimates how fast the phase between input and local oscillator is *changing* — the
frequency error — typically from two successive complex samples. A common
digital form is the **cross-product (dot-product) discriminator**: take the current
and previous down-converted samples and combine them so the output is proportional
to the phase advance per sample, i.e. the residual frequency. That error passes
through a loop filter and retunes a
[numerically-controlled oscillator](/reference/numerically-controlled-oscillator/),
exactly as in a PLL, but the quantity being zeroed is Δ*f*, not Δ*φ*.

Because a small phase discriminator (a PLL) produces an ambiguous, wrapping error
once the offset exceeds a fraction of a cycle per sample, its pull-in range is
narrow. A frequency discriminator instead gives a monotonic error over a much wider
span, so the FLL can **acquire** signals that a PLL could never pull in from cold.

## In practice: FLL-assisted PLL

The classic arrangement — used in GPS receivers, satellite modems, and burst radio —
is a two-stage acquisition:

1. **FLL first.** With a wide loop bandwidth, the FLL slews the oscillator to within
   a few tens of hertz of the true carrier, tolerating Doppler and reference error.
2. **Hand off to a PLL.** Once the frequency error is small enough to sit inside the
   PLL's pull-in range, control transfers (or an FLL-assisted-PLL discriminator
   blends both), and the PLL locks *phase* for coherent demodulation.

This gives both robust acquisition and clean tracking. An FLL alone cannot support
coherent [PSK](/reference/phase-shift-keying/) demodulation — it leaves an unknown,
drifting phase — which is why it is usually a front-end to a PLL or
[Costas loop](/reference/costas-loop/) rather than a standalone demodulator. For
slowly-varying offsets an FLL is functionally an [AFC](/reference/automatic-frequency-control/)
mechanism.

## Relevance to SDR

Any receiver facing appreciable tuner error or Doppler benefits from an FLL front-end:
GNSS, low-Earth-orbit telemetry, and burst-mode data links all rely on it to acquire
before tracking. In land-mobile trunking the offsets are smaller, but coarse
frequency correction still matters when a cheap [RTL-SDR](/reference/rtl-sdr/) tuner is
off by several kHz. GopherTrunk performs coarse frequency estimation/correction to
centre a channel before its phase-tracking loops engage; that acquisition role is the
FLL's natural niche even where the implementation is a block estimator rather than a
continuous loop.

## Sources

[^wiki]: [Frequency-locked loop](https://en.wikipedia.org/wiki/Frequency-locked_loop) — Wikipedia, on frequency-discriminator feedback and its wider pull-in versus a PLL; see also [Automatic frequency control](https://en.wikipedia.org/wiki/Automatic_frequency_control).
