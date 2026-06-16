---
slug: iir-filter
title: IIR filter
entry_type: algorithm
category: sdr-dsp
description: An infinite impulse response (IIR) filter feeds back past outputs, achieving a sharp response with few coefficients — efficient but without the exact linear phase of a FIR filter.
keywords: IIR filter, infinite impulse response, recursive filter, feedback, biquad, digital filter
aka: [IIR, "infinite impulse response filter"]
autolink: true
see_also: [fir-filter, digital-filter, automatic-gain-control]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/filtering-decimation/ }
external:
  - { title: "Infinite impulse response (Wikipedia)", url: https://en.wikipedia.org/wiki/Infinite_impulse_response }
---

An **IIR** (**infinite impulse response**) filter computes each output from both recent
**inputs and past outputs** — it has **feedback**. That recursion achieves a sharp
frequency response with far fewer coefficients than a [FIR filter](/reference/fir-filter/),
at the cost of nonlinear phase and a need to watch stability.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A filter block with a feedback path from output back to input, characteristic of an IIR filter." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="50" x2="70" y2="50" stroke="currentColor"/><text x="30" y="42" font-size="9" fill="currentColor">in</text>
  <circle cx="85" cy="50" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="85" y="54" text-anchor="middle" font-size="11" fill="currentColor">Σ</text>
  <rect x="130" y="36" width="80" height="28" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="170" y="54" text-anchor="middle" font-size="9" fill="currentColor">delays</text>
  <line x1="97" y1="50" x2="129" y2="50" stroke="currentColor" marker-end="url(#iirar)"/>
  <line x1="210" y1="50" x2="300" y2="50" stroke="currentColor" marker-end="url(#iirar)"/><text x="320" y="54" font-size="9" fill="currentColor">out</text>
  <path d="M260 50 V 95 H 85 V 63" fill="none" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#iirar)"/>
  <text x="170" y="110" text-anchor="middle" font-size="8.5" fill="currentColor">feedback path</text>
  <defs><marker id="iirar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An IIR filter feeds past outputs back into the input, giving a sharp response from very few terms.</figcaption>
</figure>

## Overview

IIR designs (often built from cascaded *biquad* sections) suit narrowband tasks like
DC blocking and the loop filters in [AGC](/reference/automatic-gain-control/) and
[timing recovery](/reference/clock-recovery/), where efficiency matters more than phase
linearity.
