---
slug: rls-algorithm
title: Recursive least squares (RLS)
entry_type: algorithm
category: equalization
description: Recursive least squares (RLS) is an adaptive filter that converges far faster than LMS by recursively tracking the inverse input-correlation matrix, at higher computational cost.
keywords: RLS, recursive least squares, adaptive filter, inverse correlation matrix, forgetting factor lambda, Kalman gain, matrix inversion lemma, fast convergence, adaptive equalizer
aka: [RLS, recursive least squares, RLS algorithm]
autolink: true
infobox:
  - { label: Type, value: Least-squares adaptive algorithm }
  - { label: Tracks, value: Inverse input-correlation matrix }
  - { label: Complexity, value: O(N²) per sample }
see_also: [adaptive-filter, lms-algorithm, mmse-equalizer, kalman-filter, fir-filter, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Recursive_least_squares_filter
  - https://en.wikipedia.org/wiki/Kalman_filter
---

**Recursive least squares (RLS)** is an [adaptive-filter](/reference/adaptive-filter/)
algorithm that, at each new sample, updates the taps to be the *exact* least-squares
solution over all data seen so far, achieving much faster and more uniform convergence
than [LMS](/reference/lms-algorithm/) — at the price of markedly higher
computation.[^wiki] Instead of taking a small gradient step, RLS recursively maintains an
estimate of the **inverse input-correlation matrix** and uses it to make an optimally
scaled correction to every tap.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Two convergence curves of error versus samples: RLS drops steeply to a low floor within a few samples while LMS decays gradually over many samples." xmlns="http://www.w3.org/2000/svg">
  <g fill="none" stroke="currentColor">
    <line x1="45" y1="20" x2="45" y2="130" stroke-width="1.1"/>
    <line x1="45" y1="130" x2="435" y2="130" stroke-width="1.1"/>
    <path d="M50 30 C 70 120, 90 122, 435 124" stroke-width="1.6"/>
    <path d="M50 30 C 160 40, 300 110, 435 118" stroke-width="1.2" stroke-opacity="0.6" stroke-dasharray="4 3"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="30" y="28" text-anchor="end">error</text>
    <text x="430" y="145" text-anchor="end">samples →</text>
    <text x="105" y="118">RLS</text>
    <text x="300" y="95" fill-opacity="0.8">LMS</text>
  </g>
</svg>
<figcaption>RLS reaches its error floor in roughly as many samples as it has taps, while LMS decays gradually over many more — the payoff for RLS's O(N²) cost per sample.</figcaption>
</figure>

## How it works

RLS minimises an *exponentially weighted* sum of past squared errors,
`Σ λ^(n−i)·|e(i)|²`, and updates the solution recursively rather than re-solving from
scratch:

- It keeps a running inverse-correlation matrix **P** (the inverse of the weighted input
  autocorrelation). Applying the **matrix inversion lemma** lets **P** be updated from the
  previous **P** without an explicit matrix inverse each sample.
- From **P** and the new input it forms a **gain vector** **k** (the RLS analogue of a
  Kalman gain) that says how strongly, and in which tap directions, to react to the
  latest error.
- The taps update as `w ← w + k·e`, where `e` is the *a priori* error (computed with the
  old taps). Because **k** already encodes the input statistics, RLS effectively
  de-correlates the input, so — unlike LMS — its convergence is nearly independent of the
  input's eigenvalue spread.

The upshot is convergence in roughly `2N` samples for `N` taps, versus the many multiples
of `N` that LMS needs on coloured inputs.

## The forgetting factor

The **forgetting factor** `λ` (0 ≪ λ ≤ 1) sets how fast old data is discounted:

- `λ = 1` weights all history equally — best for a *stationary* channel, lowest steady
  error, but no ability to track change.
- `λ < 1` (typically 0.95–0.999) gives the filter a finite memory of about `1/(1−λ)`
  samples, letting it **track** a time-varying channel at the cost of slightly higher
  steady-state error. Choosing `λ` trades tracking agility against noise immunity, the way
  `μ` does in LMS.

## In practice

RLS's speed comes with real costs: O(N²) work and storage per sample, and numerical
sensitivity — the recursively propagated **P** can lose positive-definiteness in
finite precision and cause divergence. Practical systems use square-root / QR-decomposition
or *fast RLS* (FTF, lattice) variants that restore numerical stability or cut the cost back
toward O(N). RLS is closely related to the [Kalman filter](/reference/kalman-filter/), of
which it is essentially a special case for a stationary weight vector.

## Relevance to SDR

RLS is chosen when a channel must be equalized *quickly* from a short training burst —
fast-fading HF/microwave links, burst modems, and initial acquisition where LMS would not
converge inside the preamble. It sharpens the
[constellation](/reference/constellation-diagram/) and restores
[SNR](/reference/signal-to-noise-ratio/) with fewer training symbols than
[LMS](/reference/lms-algorithm/), which is valuable when training overhead is scarce.
GopherTrunk's steady-rate P25/DMR/NXDN decode path does not deploy a full RLS equalizer —
its narrowband, [RRC-shaped](/reference/root-raised-cosine-filter/) signals are handled by
[matched filtering](/reference/matched-filter/) and synchronisation — so RLS is presented
here as a general adaptive-filtering tool of the broader RF world.

## Sources

[^wiki]: [Recursive least squares filter](https://en.wikipedia.org/wiki/Recursive_least_squares_filter) — Wikipedia, on the RLS recursion, the gain vector, and the forgetting factor.
