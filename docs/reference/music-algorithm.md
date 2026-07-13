---
slug: music-algorithm
title: MUSIC algorithm
entry_type: algorithm
category: estimation-array
description: MUSIC is a subspace direction-of-arrival estimator that splits an antenna array's covariance into signal and noise subspaces, giving super-resolution angle or frequency estimates.
keywords: MUSIC algorithm, MUltiple SIgnal Classification, direction of arrival, DOA, subspace method, eigendecomposition, noise subspace, super-resolution, array processing, Schmidt
aka: [MUSIC, MUltiple SIgnal Classification]
autolink: true
infobox:
  - { label: Type, value: Subspace DOA estimator }
  - { label: Recovers, value: Arrival angles / frequencies }
  - { label: Key idea, value: Noise-subspace null spectrum }
see_also: [esprit-algorithm, beamforming, antenna, discrete-fourier-transform, signal-to-noise-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/MUSIC_(algorithm)
  - https://ieeexplore.ieee.org/document/1143830
---

**MUSIC** (MUltiple SIgnal Classification) is a subspace method that estimates the
directions of arrival of several signals impinging on an antenna array by
eigen-decomposing the array's covariance matrix and exploiting the fact that the true
arrival directions are orthogonal to the *noise subspace*.[^wiki] Because it works with
subspaces rather than a beam pattern, MUSIC resolves sources far closer together than the
array's physical beamwidth would allow — it is a **super-resolution** technique.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A pipeline from an antenna array to a covariance matrix, then an eigendecomposition splitting into signal and noise subspaces, then a null spectrum whose sharp peaks mark the arrival angles." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="25" cy="40" r="3" fill="currentColor"/><circle cx="25" cy="58" r="3" fill="currentColor"/><circle cx="25" cy="76" r="3" fill="currentColor"/><text x="25" y="98">array</text>
    <rect x="70" y="40" width="66" height="36" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="103" y="55">covariance</text><text x="103" y="67">R = E[xxᴴ]</text>
    <rect x="165" y="40" width="70" height="36" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="200" y="55">eigen-</text><text x="200" y="67">decompose</text>
    <text x="270" y="35">signal subspace</text><text x="270" y="83">noise subspace Eₙ</text>
    <line x1="30" y1="58" x2="69" y2="58" stroke="currentColor" stroke-width="1.1" marker-end="url(#muar)"/>
    <line x1="136" y1="58" x2="164" y2="58" stroke="currentColor" stroke-width="1.1" marker-end="url(#muar)"/>
    <line x1="235" y1="58" x2="248" y2="58" stroke="currentColor" stroke-width="1.1"/>
    <g transform="translate(320,110)"><line x1="0" y1="0" x2="130" y2="0" stroke="currentColor" stroke-width="1"/><line x1="0" y1="0" x2="0" y2="-40" stroke="currentColor" stroke-width="1"/><path d="M0 -2 Q 30 -4 45 -34 Q 48 -4 60 -3 Q 80 -4 95 -30 Q 100 -3 130 -2" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="14">angle θ →  P(θ)=1/(aᴴEₙEₙᴴa)</text></g>
  </g>
  <defs><marker id="muar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>MUSIC builds the array covariance, splits it into signal and noise subspaces, then scans a steering vector so its null-spectrum spikes at each true arrival angle.</figcaption>
</figure>

## How it works

Model `M` narrowband sources arriving at an `N`-element array (`N > M`). Each source
adds a rank-one term to the spatial covariance matrix `R = E[xxᴴ]`, so `R` has `M` large
eigenvalues (signal + noise) and `N − M` small ones (noise only). The eigenvectors split
into two orthogonal blocks:

- **Signal subspace** — spanned by the eigenvectors of the large eigenvalues; it contains
  the true array-response (steering) vectors.
- **Noise subspace** `Eₙ` — spanned by the remaining eigenvectors; crucially, every true
  steering vector is orthogonal to it.

MUSIC then sweeps a candidate steering vector `a(θ)` across all angles and plots the
**pseudo-spectrum** `P(θ) = 1 / (aᴴEₙEₙᴴa)`. When `θ` matches a real source, the
denominator collapses toward zero and `P(θ)` shoots up as a sharp, narrow peak. The
`M` tallest peaks are the arrival angles. The same math applied to a time-shift array
estimates closely spaced **frequencies** (spectral MUSIC).

## In practice

MUSIC needs to know (or estimate) the number of sources `M`, and it needs a reasonably
accurate covariance estimate, which means enough snapshots and a calibrated array — element
gain/phase errors and mutual coupling smear the peaks. Coherent sources (e.g. a signal and
its [multipath](/reference/multipath-propagation/) copy) collapse the signal subspace rank
and must be decorrelated by spatial smoothing first. It is more expensive than beamforming
because of the eigendecomposition plus the angle search, and its resolution advantage fades
at low [SNR](/reference/signal-to-noise-ratio/).

## Relevance to SDR

MUSIC is a workhorse of direction finding, radar, sonar, and channel sounding, and
spectral MUSIC is used for high-resolution frequency estimation. Its main rival,
[ESPRIT](/reference/esprit-algorithm/), reaches the same subspace but skips the angle
search. Both assume a multi-element phased array with coherent per-channel sampling —
hardware GopherTrunk does not have: GT is a single-front-end trunking receiver, so it does
**no** DOA estimation. MUSIC is included here as the canonical super-resolution array
algorithm in the broader RF world, the direct counterpart to the
[beamforming](/reference/beamforming/) that shares the same array data.

## Sources

[^wiki]: [MUSIC (algorithm)](https://en.wikipedia.org/wiki/MUSIC_(algorithm)) — Wikipedia, on subspace DOA estimation via the noise-subspace null spectrum.
