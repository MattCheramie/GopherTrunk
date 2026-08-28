---
slug: interference-rejection-combining
title: Interference rejection combining (IRC)
entry_type: algorithm
category: estimation-array
description: Interference rejection combining (IRC) is MMSE diversity combining that uses the measured noise-plus-interference covariance to steer a spatial null at a directional interferer that MRC would amplify.
keywords: interference rejection combining, IRC, MMSE-IRC, noise covariance, spatial null, co-channel interference, diversity combining, MRC, channel estimation, LTE receiver
aka: [IRC, MMSE-IRC, interference rejection combiner]
autolink: true
infobox:
  - { label: Type, value: MMSE diversity combiner }
  - { label: Weights, value: "w = R_nn⁻¹ h (normalised)" }
  - { label: Beats MRC when, value: A directional interferer dominates }
  - { label: Catch, value: Needs a clean channel estimate — blind IRC fails }
see_also: [maximal-ratio-combining, antenna-diversity, mmse-equalizer, beamforming, channel-estimation, coherence]
cite_urls:
  - https://en.wikipedia.org/wiki/Diversity_combining
  - https://en.wikipedia.org/wiki/Beamforming
---

**Interference rejection combining** (**IRC**) is the diversity combiner to reach for when
the enemy is not noise but another transmitter: instead of weighting branches by signal
strength like [maximal-ratio combining](/reference/maximal-ratio-combining/), it measures
the spatial covariance of everything that is *not* the wanted signal and inverts it, which
steers a null toward a directional interferer while keeping gain on the wanted
direction.[^wiki] It is the standard uplink combiner in LTE and 5G base stations, where
co-channel interference from neighbouring cells, not thermal noise, sets the floor.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A two-antenna receiver pattern with a main lobe pointed at the wanted signal and a deep notch steered toward an interferer arriving from a different direction." xmlns="http://www.w3.org/2000/svg">
  <path d="M80 120 C 120 20 210 20 240 78 C 250 96 244 110 232 116 C 260 104 300 96 340 118" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="80" y1="120" x2="340" y2="120" stroke="currentColor" stroke-opacity="0.35"/>
  <circle cx="150" cy="120" r="3" fill="currentColor"/><circle cx="270" cy="120" r="3" fill="currentColor"/>
  <line x1="160" y1="98" x2="160" y2="34" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#ircar)"/>
  <text x="160" y="24" font-size="9" fill="currentColor" text-anchor="middle">wanted signal</text>
  <line x1="237" y1="98" x2="237" y2="120" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/>
  <line x1="237" y1="60" x2="237" y2="96" stroke="currentColor" stroke-width="1.1" marker-end="url(#ircar2)"/>
  <text x="237" y="50" font-size="9" fill="currentColor" text-anchor="middle">interferer → null</text>
  <text x="210" y="140" font-size="8.5" fill="currentColor" text-anchor="middle">two antennas, one null to spend</text>
  <defs>
    <marker id="ircar" markerWidth="8" markerHeight="8" refX="3" refY="6" orient="auto"><path d="M0 6 L3 0 L6 6 z" fill="currentColor"/></marker>
    <marker id="ircar2" markerWidth="8" markerHeight="8" refX="3" refY="6" orient="0"><path d="M0 0 L3 6 L6 0 z" fill="currentColor"/></marker>
  </defs>
</svg>
<figcaption>With N antennas the combiner has N−1 spatial degrees of freedom to spend: IRC spends one on a null toward the interferer, which MRC would instead amplify along with the branch it is loudest on.</figcaption>
</figure>

## How it works

Model branch *k* as `x_k = h_k·s + n_k`, where `n_k` now contains both noise *and* the
interference. Collect the interference-plus-noise covariance matrix `R_nn` across branches
(the interferer makes its off-diagonal terms large and structured, unlike white noise) and
form the [MMSE](/reference/mmse-equalizer/) weights

`w = R_nn⁻¹ h`, scaled so `wᴴh = 1`.

When `R_nn` is diagonal — pure independent noise — this collapses to exactly MRC, so IRC is
a strict generalisation, not a rival. When one interferer dominates, `R_nn⁻¹` de-emphasises
the spatial direction the interferer arrives from: the weights place a null there, the same
mathematics as adaptive [beamforming](/reference/beamforming/) with N−1 degrees of freedom
for N antennas. A small diagonal loading term keeps the inversion stable when the estimate
window is short.

## Why blind IRC fails

The formula hides a dependency that decides whether IRC works at all: `R_nn` is the
covariance of the *residual* after the wanted signal is removed, and removing the wanted
signal requires knowing `h` — a clean
[channel estimate](/reference/channel-estimation/). With a co-channel interferer present, a
blind least-squares estimate of `h` is contaminated: the cross-correlation between branches
contains `h_wanted·P_signal + h_interf·P_interf`, so the estimator returns a power-weighted
*blend* of the two directions, and the null gets steered at a mixture that is neither.
GopherTrunk measured this directly on a synthetic two-branch co-channel scene
(`internal/dsp/diversity/irc_test.go`): a true branch gain of 0.95∠40° read back as
0.32∠1°, blind IRC gained **0.0 dB** over MRC, and the *identical* code handed a training
sequence gained **+23.6 dB**. Cellular IRC works because every uplink burst carries known
reference symbols; a blind wideband combiner has none. There is also a subtle degeneracy to
avoid: estimating the residual against the reference *branch* rather than the combined
output forces that branch's residual to zero by construction, zeroing a row and column of
`R_nn` so no null can form.

## Relevance to SDR

For an SDR listener, IRC is the answer to a specific and recognisable failure: a strong
directional interferer (a paging transmitter, an adjacent site) that gets *worse* when MRC
diversity is enabled, because MRC faithfully weights the branch the interferer is loudest
on. The catch is the training requirement — in a trunking receiver, known symbols (a
[training sequence](/reference/tetra-training-sequences/) or
[frame sync](/reference/frame-synchronization/)) exist only per narrowband channel, after
the [DDC](/reference/digital-down-converter/), which is also where per-channel combining has
to live anyway. GopherTrunk therefore ships `IRCCalibrator`
(`internal/dsp/diversity/irc.go`) as an offline replay-harness arm for capture analysis, not
as a live driver mode: the measured blind-vs-trained gap above is the documented reason.

## Sources

[^wiki]: [Diversity combining](https://en.wikipedia.org/wiki/Diversity_combining) — Wikipedia, on combining schemes and the MMSE/optimum combining generalisation of MRC under interference.
