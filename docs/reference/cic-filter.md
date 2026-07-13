---
slug: cic-filter
title: CIC filter
entry_type: algorithm
category: filtering-multirate
description: A cascaded integrator–comb (CIC) filter is a multiplier-free filter used for efficient large-ratio decimation or interpolation in SDR front ends and channelizers.
keywords: CIC filter, cascaded integrator comb, decimation, interpolation, multiplier-free, Hogenauer, passband droop, bit growth, compensation FIR
aka: [CIC filter, cascaded integrator-comb filter, Hogenauer filter]
autolink: true
infobox:
  - { label: Type, value: Decimation/interpolation filter }
  - { label: Feature, value: No multipliers (adds/delays only) }
  - { label: Use, value: Large-ratio rate change }
see_also: [decimation, half-band-filter, digital-down-converter, polyphase-filter-bank, digital-filter, sample-rate]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Cascaded_integrator%E2%80%93comb_filter
  - https://web.archive.org/web/20120118234057/http://home.mit.bme.hu/~kollar/papers/Hogenauer.pdf
---

A **CIC filter** (cascaded integrator–comb) is a [digital filter](/reference/digital-filter/)
built from only integrators and combs — **no multipliers**, just adders and delays — which
makes it the cheapest practical way to change [sample rate](/reference/sample-rate/) by a
large ratio.[^wiki] Introduced by Eugene Hogenauer in 1981, it is the standard first stage
of [decimation](/reference/decimation/) in SDR hardware and the front of software
[channelizers](/reference/channelizer/), where an incoming stream must be brought down by
tens or hundreds to one before any multiply-heavy filtering is affordable.[^hoge]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A cascaded integrator-comb decimator: two integrator stages running at the high rate, a rate-change by R, then two comb stages running at the low rate." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="24" y="45" width="52" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="50" y="64">∫</text>
    <rect x="84" y="45" width="52" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="110" y="64">∫</text>
    <rect x="152" y="45" width="58" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="181" y="63">↓ R</text>
    <rect x="226" y="45" width="52" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="252" y="64">comb</text>
    <rect x="286" y="45" width="52" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="312" y="64">comb</text>
    <text x="90" y="28" font-size="8">integrators @ high rate</text>
    <text x="312" y="28" font-size="8">combs @ low rate</text>
    <text x="400" y="64">rate/R<tspan x="400" dy="12">output</tspan></text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" marker-end="url(#cicar)"><line x1="76" y1="60" x2="83" y2="60"/><line x1="136" y1="60" x2="151" y2="60"/><line x1="210" y1="60" x2="225" y2="60"/><line x1="278" y1="60" x2="285" y2="60"/><line x1="338" y1="60" x2="360" y2="60"/></g>
  <defs><marker id="cicar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A CIC decimator: integrators accumulate at the high rate, the sample rate drops by R, then combs difference at the low rate — all with adders and delays, no multipliers.</figcaption>
</figure>

## How it works

A single integrator is a running accumulator, `y[n] = y[n−1] + x[n]`; a single comb is a
delayed difference, `y[n] = x[n] − x[n−M]`. Cascade *N* integrators, drop the rate by *R*,
then cascade *N* combs and the two halves collapse — through the identity that an
accumulate-then-difference pair is a moving-average — into an *N*-th-order boxcar (moving
average) low-pass filter of length `R·M`. The magic is *where* each stage runs: integrators
sit **before** the rate change at the high input rate, combs sit **after** it at the low
output rate, so the expensive delay lines in the combs are short and clocked slowly. For
interpolation the structure runs in reverse (combs first, upsample, then integrators).

Because every coefficient is effectively ±1, a CIC needs **no multipliers at all** — only
adders, registers, and the decimator/expander. That is why it is baked into the digital
down-converters of nearly every SDR chip and FPGA front end: it delivers a huge,
run-time-programmable rate change for a handful of gates per stage.

- **Bit growth.** The integrators have unity-at-DC feedback, so their internal word grows
  by `N·log₂(R·M)` bits. Fixed-point implementations must size the accumulators for the
  worst case (and rely on two's-complement wrap-around cancelling correctly between the
  integrator and comb sections), or the filter overflows silently.
- **Passband droop.** A boxcar's frequency response is a `sinc` (raised to the *N*-th
  power), which sags across the passband and has only modest stop-band rejection between
  its nulls. The nulls land on the aliasing bands, which is why the shape works, but the
  droop distorts wanted signals near band edge.

## Variants and compensation

The droop is corrected downstream by a short **compensation FIR** — an inverse-`sinc`
"CIC comp" filter — usually combined with the final [half-band](/reference/half-band-filter/)
decimation stages that trim the rate the rest of the way and sharpen the transition band.
Design freedom lives in three integers: the number of stages *N* (steeper roll-off and
better alias rejection, but more droop and delay), the differential delay *M* (1 or 2,
setting null placement), and the rate change *R*. A common SDR chain is **CIC → CIC-comp
FIR → half-band cascade → channel FIR**, letting the multiplier-free CIC absorb the bulk of
the decimation so the multiply-based filters only ever run at a low rate.

## In practice

CIC filters are the workhorse of the first decimation stage in [RTL-SDR](/reference/rtl-sdr/),
Airspy, and HackRF-class receivers and in the [digital down-converters](/reference/digital-down-converter/)
of communications ASICs, precisely because a programmable, gate-cheap rate change is exactly
what a wideband front end needs before anything selective happens. Their weaknesses —
droop and finite alias rejection — are tolerable there because a following FIR fixes both.

## Relevance to SDR

Any GopherTrunk decode that starts from a wideband capture leans on CIC-style multiplier-free
decimation in the SDR hardware before samples ever reach the host. GopherTrunk's own software
down-conversion favours [polyphase](/reference/polyphase-filter-bank/) and half-band FIR
stages rather than a software CIC, but the principle — do the coarse rate reduction with the
cheapest filter, then clean up with a short FIR — is the same one that shapes its channelizer
front end.

## Sources

[^wiki]: [Cascaded integrator–comb filter](https://en.wikipedia.org/wiki/Cascaded_integrator%E2%80%93comb_filter) — Wikipedia, on Hogenauer's multiplier-free decimation/interpolation structure, bit growth, and passband droop.
[^hoge]: [An Economical Class of Digital Filters for Decimation and Interpolation](https://web.archive.org/web/20120118234057/http://home.mit.bme.hu/~kollar/papers/Hogenauer.pdf) — E. B. Hogenauer, IEEE Trans. ASSP, 1981 — the seminal CIC paper defining the register-growth and comb/integrator arrangement.
