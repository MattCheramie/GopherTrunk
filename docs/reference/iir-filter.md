---
slug: iir-filter
title: IIR filter
entry_type: algorithm
category: filtering-multirate
description: An infinite impulse response (IIR) filter feeds back past outputs for a sharp response from few coefficients — efficient, but without a FIR filter's linear phase.
keywords: IIR filter, infinite impulse response, recursive filter, feedback, biquad, poles and zeros, Butterworth, Chebyshev, digital filter, stability
aka: [IIR, "infinite impulse response filter", recursive filter]
autolink: true
infobox:
  - { label: Type, value: Recursive digital filter }
  - { label: Feature, value: Sharp response, few coefficients }
  - { label: Trade-off, value: Nonlinear phase, watch stability }
see_also: [fir-filter, digital-filter, automatic-gain-control, clock-recovery]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Infinite_impulse_response
  - https://en.wikipedia.org/wiki/Digital_biquad_filter
---

An **IIR** (**infinite impulse response**) filter computes each output from both recent
**inputs and past outputs** — it has **feedback**.[^wiki] That recursion lets a handful of
coefficients produce a very sharp [frequency response](/reference/digital-filter/), so an IIR
can match an [FIR filter](/reference/fir-filter/)'s selectivity at a fraction of the
arithmetic — at the cost of **nonlinear phase** and a genuine need to **watch stability**,
since the same feedback that sharpens the response can also make it oscillate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A filter with a summing junction, a bank of delays, and a dashed feedback path carrying the output back to the input — the defining structure of a recursive IIR filter." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="50" x2="70" y2="50" stroke="currentColor"/><text x="30" y="42" font-size="9" fill="currentColor">in</text>
  <circle cx="85" cy="50" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="85" y="54" text-anchor="middle" font-size="11" fill="currentColor">Σ</text>
  <rect x="130" y="36" width="90" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="175" y="54" text-anchor="middle" font-size="9" fill="currentColor">delays × aₖ,bₖ</text>
  <line x1="97" y1="50" x2="129" y2="50" stroke="currentColor" marker-end="url(#iirar)"/>
  <line x1="220" y1="50" x2="310" y2="50" stroke="currentColor" marker-end="url(#iirar)"/><text x="330" y="54" font-size="9" fill="currentColor">out</text>
  <path d="M270 50 V 100 H 85 V 63" fill="none" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#iirar)"/>
  <text x="177" y="118" text-anchor="middle" font-size="8.5" fill="currentColor">feedback path (past outputs re-enter)</text>
  <defs><marker id="iirar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An IIR filter feeds past outputs back into the summing junction; that recursion gives a sharp response from very few terms but can ring forever, so its impulse response is "infinite."</figcaption>
</figure>

## How it works

The defining recurrence mixes feed-forward and feedback terms:
`y[n] = Σ bₖ·x[n−k] − Σ aₖ·y[n−k]`. Because a past output re-enters the calculation, the
impulse response never truly ends — hence *infinite* impulse response. The behaviour is
easiest to reason about in terms of **poles and zeros** in the z-plane: the feed-forward `b`
coefficients place **zeros** (frequencies pushed toward null), and the feedback `a`
coefficients place **poles** (frequencies boosted, producing sharp resonances). Selectivity
that would need dozens of FIR taps can come from a single pole pair sitting just inside the
unit circle.

Two consequences follow directly from the poles:

- **Stability is not automatic.** Every pole must lie strictly inside the unit circle. Push
  one outside — through a design error or fixed-point coefficient rounding — and the output
  grows without bound. Unlike a FIR, an IIR can be unstable, so implementations must be
  designed and quantised with care.
- **Phase is nonlinear.** Different frequencies are delayed by different amounts, which
  distorts pulse shapes. For audio and control loops this is harmless; for pulse-shaped
  digital symbols it can smear the constellation, which is why demodulators usually reach for
  a linear-phase FIR instead.

## Variants: biquads and classic responses

Real IIR filters are almost never built as one high-order recursion — coefficient sensitivity
would make them fragile. Instead they are factored into a cascade of **biquads**: second-order
sections, each with two poles and two zeros. Cascading biquads keeps every section
numerically well-behaved and lets each be tuned independently. The pole/zero *pattern* is
chosen from a standard family — **Butterworth** (maximally flat passband), **Chebyshev**
(steeper roll-off in exchange for passband or stop-band ripple), or **elliptic** (steepest
transition for a given order) — usually derived from a proven analog prototype and mapped to
the digital domain by the bilinear transform.

## In practice

IIR designs shine in narrowband, low-latency tasks where efficiency matters more than phase
linearity: DC-blocking a stream, notching a tone, smoothing an envelope, and the **loop
filters** buried inside an [AGC](/reference/automatic-gain-control/), a PLL, or a
[timing-recovery](/reference/clock-recovery/) loop — where a simple one-pole integrator or a
biquad does the averaging with almost no computation. Where a flat group delay is required,
the job goes to a FIR instead; the two filter families are complementary tools, not rivals.

## Relevance to SDR

GopherTrunk uses IIR structures for exactly these narrowband, phase-tolerant jobs — DC
removal, envelope and error smoothing, and the recursive loop filters inside its AGC and
synchronisation loops — while leaving the selective, phase-critical channel and pulse-shaping
filtering to linear-phase FIRs. Knowing which family fits which job is a core DSP judgement in
any SDR decode chain.

## Sources

[^wiki]: [Infinite impulse response](https://en.wikipedia.org/wiki/Infinite_impulse_response) — Wikipedia, on the recursive, feedback-based filter, poles and zeros, and stability trade-offs.
[^biq]: [Digital biquad filter](https://en.wikipedia.org/wiki/Digital_biquad_filter) — Wikipedia, on the second-order sections that IIR filters are cascaded from for numerical robustness.
