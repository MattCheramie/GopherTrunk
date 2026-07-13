---
slug: lms-algorithm
title: Least mean squares (LMS)
entry_type: algorithm
category: equalization
description: The least-mean-squares (LMS) algorithm adapts filter taps by a stochastic-gradient rule w += mu·e·x*, the low-cost workhorse for adaptive equalization and cancellation.
keywords: LMS, least mean squares, stochastic gradient, adaptive filter, step size mu, normalized LMS, NLMS, adaptive equalizer, Widrow-Hoff, convergence, misadjustment
aka: [LMS, least mean squares, LMS algorithm, Widrow-Hoff LMS]
autolink: true
infobox:
  - { label: Type, value: Stochastic-gradient adaptive algorithm }
  - { label: Update, value: "w += μ·e·x*" }
  - { label: Complexity, value: O(N) per sample }
see_also: [adaptive-filter, rls-algorithm, cma-equalizer, mmse-equalizer, fir-filter, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Least_mean_squares_filter
  - https://en.wikipedia.org/wiki/Bernard_Widrow
---

The **least-mean-squares (LMS)** algorithm is the workhorse
[adaptive-filter](/reference/adaptive-filter/) update rule: it adjusts the filter taps by
a small step *down the instantaneous gradient* of the squared error, using the simple
recursion `w ← w + μ·e·x*`.[^wiki] Introduced by Bernard Widrow and Ted Hoff in 1960, it
trades the slower convergence of an exact least-squares solution for an update that costs
only a handful of multiply-accumulates per sample, which is why it appears in equalizers,
echo cancellers, and noise cancellers everywhere.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A bowl-shaped error surface with a path of small steps descending in a jagged zig-zag toward the minimum, illustrating stochastic-gradient descent." xmlns="http://www.w3.org/2000/svg">
  <g fill="none" stroke="currentColor">
    <path d="M40 30 Q 230 200 420 30" stroke-width="1.3"/>
    <path d="M70 55 Q 230 175 390 55" stroke-width="0.8" stroke-opacity="0.5"/>
    <polyline points="70,52 120,92 150,86 185,116 205,110 228,128 245,124 258,134" stroke-width="1.4" stroke-opacity="0.9"/>
    <circle cx="258" cy="134" r="3" fill="currentColor"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="70" y="45">start w₀</text>
    <text x="268" y="138">minimum</text>
    <text x="150" y="150" text-anchor="middle">each step: w += μ·e·x*</text>
  </g>
</svg>
<figcaption>LMS descends the mean-square-error bowl in noisy little steps of size μ; a larger step converges faster but rattles more around the minimum (misadjustment).</figcaption>
</figure>

## How it works

LMS approximates true gradient descent on the mean-square error by using a single
sample's error in place of a statistical average:

- Form the filter output `y = wᵀx` from the current tap vector **w** and the recent input
  samples **x**.
- Compute the error `e = d − y` against the desired signal `d` (a training reference, a
  past decision, or — in blind variants — a target property).
- Update every tap: `w ← w + μ·e·x*`, where `μ` (the *step size*) sets how far each
  sample nudges the taps and `x*` is the complex conjugate of the input for I/Q data.

Because the gradient estimate is noisy, the taps never sit exactly at the optimum; they
hover around it. That residual jitter, called **misadjustment**, grows with `μ`.

## Convergence and stability

The single knob `μ` governs the whole trade-off:

- **Too large** — the taps overshoot and the algorithm diverges. Stability requires
  roughly `0 < μ < 2/λ_max`, where `λ_max` is the largest eigenvalue of the input
  autocorrelation (in practice `μ < 2/(N·P)` for `N` taps of input power `P`).
- **Too small** — stable and low-misadjustment, but slow to converge and slow to track a
  moving channel.
- Convergence speed depends on the input's **eigenvalue spread** (ratio of largest to
  smallest autocorrelation eigenvalue): highly coloured inputs converge slowly, a known
  weakness LMS shares and [RLS](/reference/rls-algorithm/) largely fixes.

**Normalized LMS (NLMS)** removes the dependence on input power by scaling the step by the
current input energy, `μ/(ε + ‖x‖²)`, making the choice of `μ` far less sensitive to
signal level — the form used in most practical echo and equalizer designs.

## Relevance to SDR

LMS and NLMS are the default engines behind adaptive channel
**equalization**, acoustic and line **echo cancellation**, and adaptive interference
cancellers. In a radio receiver an LMS equalizer trims residual
[multipath](/reference/multipath-propagation/), tightening the
[constellation](/reference/constellation-diagram/) and lifting the effective
[SNR](/reference/signal-to-noise-ratio/) at the decision slicer, often running
decision-directed once a training sequence has pulled it near lock. The blind
[CMA](/reference/cma-equalizer/) equalizer is an LMS-style stochastic-gradient rule with a
constant-modulus cost instead of a reference error. GopherTrunk's land-mobile decoders
(P25, DMR, NXDN) lean on [matched filtering](/reference/matched-filter/) and timing/carrier
recovery rather than a full LMS equalizer, so LMS is best understood here as the general
adaptive-filter primitive the wider RF world runs on.

## Sources

[^wiki]: [Least mean squares filter](https://en.wikipedia.org/wiki/Least_mean_squares_filter) — Wikipedia, on the LMS update rule, step-size stability bounds, and NLMS.
