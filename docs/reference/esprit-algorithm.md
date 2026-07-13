---
slug: esprit-algorithm
title: ESPRIT algorithm
entry_type: algorithm
category: estimation-array
description: ESPRIT is a subspace DOA/frequency estimator that exploits a rotational invariance between two shifted array subarrays to solve for arrival angles directly, with no spectral search.
keywords: ESPRIT algorithm, Estimation of Signal Parameters via Rotational Invariance Techniques, direction of arrival, DOA, subspace method, rotational invariance, frequency estimation, array processing, MUSIC
aka: [ESPRIT, Estimation of Signal Parameters via Rotational Invariance Techniques]
autolink: true
infobox:
  - { label: Type, value: Subspace DOA estimator }
  - { label: Recovers, value: Arrival angles / frequencies }
  - { label: Key idea, value: Rotational invariance, no search }
see_also: [music-algorithm, beamforming, antenna, discrete-fourier-transform, signal-to-noise-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/Estimation_of_signal_parameters_via_rotational_invariance_techniques
  - https://ieeexplore.ieee.org/document/32276
---

**ESPRIT** (Estimation of Signal Parameters via Rotational Invariance Techniques) is a
subspace direction-of-arrival estimator that, like [MUSIC](/reference/music-algorithm/),
separates a signal subspace from noise — but instead of scanning a spectrum for peaks it
solves for the arrival angles *algebraically* by exploiting a shift structure built into
the array.[^wiki] Two identical subarrays separated by a fixed displacement see the same
sources with only a phase rotation between them, and that rotation encodes the angles
directly.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two overlapping subarrays offset by a fixed spacing; their signal subspaces are related by a rotation matrix whose eigenvalues, unit-modulus phasors, give the arrival angles directly without a search." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="40" cy="35" r="3" fill="currentColor"/><circle cx="75" cy="35" r="3" fill="currentColor"/><circle cx="110" cy="35" r="3" fill="currentColor"/><circle cx="145" cy="35" r="3" fill="currentColor"/>
    <line x1="30" y1="52" x2="120" y2="52" stroke="currentColor" stroke-width="1.2"/><text x="75" y="64">subarray 1</text>
    <line x1="65" y1="70" x2="155" y2="70" stroke="currentColor" stroke-width="1.2"/><text x="110" y="82">subarray 2 (shifted Δ)</text>
    <rect x="210" y="30" width="90" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="255" y="44">signal</text><text x="255" y="56">subspaces</text>
    <rect x="330" y="30" width="105" height="34" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="382" y="44">rotation Ψ</text><text x="382" y="56">eig(Ψ) → θ</text>
    <line x1="160" y1="47" x2="209" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#esar)"/>
    <line x1="300" y1="47" x2="329" y2="47" stroke="currentColor" stroke-width="1.1" marker-end="url(#esar)"/>
    <text x="300" y="112" text-anchor="middle">eigenvalue phases = arrival angles — no grid search</text>
  </g>
  <defs><marker id="esar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>ESPRIT pairs two displacement-shifted subarrays; the rotation matrix relating their signal subspaces has eigenvalues whose phases are the arrival angles, found in closed form.</figcaption>
</figure>

## How it works

Split the array into two subarrays that are identical up to a known translation `Δ`. Both
observe the same signal subspace `Eₛ` (from an eigendecomposition of the array covariance),
but the second subarray's copy is the first multiplied by a diagonal matrix of phase
factors `e^{jωΔsinθ}` — one per source. That means the two subarray slices of `Eₛ` are
linked by a single invertible **rotation matrix** `Ψ`:

- Estimate the covariance and its signal subspace `Eₛ`.
- Partition `Eₛ` into the two subarray rows and solve (typically total least squares) for
  `Ψ` such that `Eₛ₁ Ψ ≈ Eₛ₂`.
- Take the **eigenvalues** of `Ψ`. They lie on the unit circle, and each one's phase maps
  straight to an arrival angle (or, in the temporal version, a frequency).

There is no angle grid and no peak search — the parameters drop out of an eigenvalue
computation, which is both faster and free of the resolution-versus-grid-spacing trade-off
that a scanned spectrum has.

## Contrast with MUSIC

MUSIC and ESPRIT reach the same signal/noise subspace split; they differ in the last step.
MUSIC *searches* a pseudo-spectrum over all candidate angles, needs an accurately calibrated
steering vector `a(θ)` for every angle, and pays for a fine grid. ESPRIT *computes* the
angles from the shift structure, so it needs no calibrated array manifold and no search,
making it cheaper and more robust to calibration error — at the cost of requiring the array
to have that specific translational (doublet) geometry. MUSIC works with arbitrary array
shapes; ESPRIT trades that flexibility for speed and closed-form estimates.

## Relevance to SDR

ESPRIT is used for direction finding, radar and channel-parameter estimation, and its
temporal form gives high-resolution [frequency](/reference/discrete-fourier-transform/)
and delay estimates in channel sounders and OFDM channel estimation. Like MUSIC it assumes
a coherent multi-element [antenna](/reference/antenna/) array with per-element sampling —
hardware a single-front-end receiver like **GopherTrunk** does not have, so GT performs no
DOA estimation. It appears here as the search-free counterpart to
[MUSIC](/reference/music-algorithm/) in the array-processing toolkit of the wider RF world.

## Sources

[^wiki]: [ESPRIT](https://en.wikipedia.org/wiki/Estimation_of_signal_parameters_via_rotational_invariance_techniques) — Wikipedia, on rotational-invariance subspace DOA estimation without a spectral search.
