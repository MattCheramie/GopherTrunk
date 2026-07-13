---
slug: adaptive-filter
title: Adaptive filter
entry_type: algorithm
category: equalization
description: An adaptive filter automatically updates its own coefficients from an error signal to track a changing channel, used for equalization, echo cancellation, and interference removal.
keywords: adaptive filter, LMS, RLS, CMA, adaptive equalizer, error signal, tap update, echo cancellation, interference cancellation, system identification, blind equalization
aka: [adaptive filter, adaptive equalizer]
autolink: true
infobox:
  - { label: Type, value: Self-adjusting digital filter }
  - { label: Driven by, value: Error signal (measured or blind) }
  - { label: Used for, value: Equalization, echo/interference cancellation }
see_also: [lms-algorithm, rls-algorithm, cma-equalizer, fir-filter, mmse-equalizer, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Adaptive_filter
  - https://ieeexplore.ieee.org/document/1457259
---

An **adaptive filter** is a digital [filter](/reference/fir-filter/) whose coefficients
(taps) are not fixed but are continuously updated by an algorithm that minimises an
**error signal**, letting it track a channel or interference that changes over
time.[^wiki] Where a conventional [FIR filter](/reference/fir-filter/) is designed once
and left alone, an adaptive filter re-optimises itself sample by sample, which makes it
the core building block of equalizers, echo cancellers, and adaptive noise/interference
cancellers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An input signal passes through an adjustable filter whose output is subtracted from a desired reference to form an error, which an update algorithm feeds back to adjust the filter taps." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="afar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="90" y="30" width="86" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="133" y="44">adjustable</text><text x="133" y="55">filter w</text>
    <circle cx="250" cy="47" r="12" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="250" y="51" font-size="12">−</text>
    <rect x="90" y="112" width="86" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="133" y="126">update</text><text x="133" y="136">algorithm</text>
    <line x1="20" y1="47" x2="89" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/><text x="45" y="40">x</text>
    <line x1="176" y1="47" x2="237" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/><text x="205" y="40">y</text>
    <line x1="262" y1="47" x2="430" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/><text x="360" y="40">error e</text>
    <line x1="250" y1="15" x2="250" y2="34" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/><text x="285" y="20">desired d</text>
    <path d="M330 47 V 127 H 177" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/>
    <path d="M133 112 V 65" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/>
  </g>
</svg>
<figcaption>The adaptive-filter loop: the difference between the filter output y and a desired signal d forms an error e that an update rule uses to nudge the taps w toward the optimum.</figcaption>
</figure>

## How it works

An adaptive filter has three parts: a tapped-delay-line filter with weight vector **w**,
an error computation, and a coefficient-update rule.

- The filter forms its output as a weighted sum of recent input samples,
  `y = wᵀx` — the same convolution an ordinary FIR filter performs.
- An **error** `e = d − y` is computed against a *desired* response `d`. In
  training-based operation `d` is a known reference (a training/sync sequence); in
  *decision-directed* operation `d` is the receiver's own hard symbol decision; in
  *blind* operation there is no explicit `d` at all and the algorithm instead enforces a
  statistical property of the wanted signal.
- The **update rule** moves **w** to shrink the error. Most rules follow the
  gradient of a cost function — typically the mean-square error — so the taps descend
  toward the configuration that best matches the desired output.

Because the update runs every sample, the filter converges from an arbitrary start and
then *tracks* slow changes in the channel (fading, Doppler, drift) rather than needing a
fresh design.

## Variants

The three classic algorithm families trade convergence speed against cost:

- **[LMS](/reference/lms-algorithm/)** (least mean squares) — a stochastic-gradient
  update, `w += μ·e·x*`. Cheap (O(N) per sample) and robust, but converges slowly and
  its speed depends on the input's eigenvalue spread.
- **[RLS](/reference/rls-algorithm/)** (recursive least squares) — recursively tracks the
  inverse input-correlation matrix for much faster, spread-independent convergence at
  O(N²) cost and greater numerical fragility.
- **[CMA](/reference/cma-equalizer/)** (constant modulus algorithm) — a *blind* update
  that needs no reference, driving the output toward a constant envelope; the standard
  choice when no training sequence is available.

The same machinery, pointed at a different desired signal, also performs **echo
cancellation** (model the echo path, subtract it), **adaptive interference/noise
cancellation** (subtract a correlated noise reference), and **system identification**
(the converged taps *are* an estimate of the unknown system).

## In practice

Stability hinges on the step size or forgetting factor: too aggressive and the taps
diverge or chatter; too gentle and the filter lags a moving channel. Practical designs
also worry about tap count (enough to span the channel's delay spread), fixed-point
precision, and, in decision-directed mode, error propagation when early decisions are
wrong.

## Relevance to SDR

Adaptive filtering underlies channel **equalization** in nearly every high-rate digital
radio: it collapses [multipath](/reference/multipath-propagation/)-smeared
[constellations](/reference/constellation-diagram/) back to tight clusters and improves
the effective [SNR](/reference/signal-to-noise-ratio/) at the slicer. GSM receivers
equalize with an [MLSE](/reference/maximum-likelihood-sequence-estimation/) or DFE;
cable, DSL, and microwave links lean on LMS/RLS equalizers; and blind
[CMA](/reference/cma-equalizer/) equalizers rescue signals that carry no training
sequence. In the land-mobile world GopherTrunk targets — P25, DMR, NXDN — the modest
symbol rates and root-raised-cosine [pulse shaping](/reference/root-raised-cosine-filter/)
mean a matched filter plus timing/carrier recovery usually suffices, so full adaptive
equalizers are more the exception than the rule; GopherTrunk relies on
[matched filtering](/reference/matched-filter/) and synchronisation rather than a general
adaptive equalizer in its steady-state decode path.

## Sources

[^wiki]: [Adaptive filter](https://en.wikipedia.org/wiki/Adaptive_filter) — Wikipedia, overview of adaptive filter structures, the error-driven update loop, and applications.
