---
slug: maximal-ratio-combining
title: Maximal-ratio combining (MRC)
entry_type: algorithm
category: estimation-array
description: Maximal-ratio combining (MRC) co-phases the branches of a diversity receiver and weights each by its own signal quality before summing, making the output SNR the sum of the branch SNRs.
keywords: maximal ratio combining, MRC, diversity combining, coherent combining, branch weights, channel estimate, receive diversity, antenna diversity, equal gain combining, selection combining
aka: [MRC, maximum ratio combining, ratio squarer]
autolink: true
infobox:
  - { label: Type, value: Coherent diversity combiner }
  - { label: Output SNR, value: Sum of the branch SNRs }
  - { label: Needs, value: A complex channel estimate per branch }
  - { label: Optimal in, value: Additive white noise (no interference) }
see_also: [antenna-diversity, interference-rejection-combining, coherence, channel-estimation, beamforming, rayleigh-fading, mimo]
related_reading:
  - { title: "The Analog Edge, Part 11: Two Antennas — Diversity & MRC From the Operator's Seat", url: /blog/tutorials/analog-edge-11-diversity-mrc/ }
  - { title: "Weak-Signal Engineering, Part 11: Tracking MRC", url: /blog/deep-dives/weak-signal-engineering-11-tracking-mrc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Maximal-ratio_combining
  - https://en.wikipedia.org/wiki/Diversity_combining
---

**Maximal-ratio combining** (**MRC**) is the optimal way to merge the branches of a
[diversity](/reference/antenna-diversity/) receiver when the impairment is additive noise:
rotate each branch so the copies add in phase, weight each by its own signal quality, and
sum.[^wiki] The result is stronger than any single branch — the combined
[SNR](/reference/signal-to-noise-ratio/) is the *sum* of the branch SNRs — and a branch deep
in a [fade](/reference/rayleigh-fading/) still contributes its share instead of being thrown
away, which is what separates MRC from simply selecting the best antenna.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Two receiver branches with different phases and amplitudes are each multiplied by the conjugate of their channel estimate, then summed, producing one output whose SNR is the sum of the branch SNRs." xmlns="http://www.w3.org/2000/svg">
  <text x="30" y="48" font-size="9" fill="currentColor">x₀</text>
  <text x="30" y="118" font-size="9" fill="currentColor">x₁</text>
  <line x1="45" y1="44" x2="115" y2="44" stroke="currentColor" stroke-width="1.2" marker-end="url(#mrcar)"/>
  <line x1="45" y1="114" x2="115" y2="114" stroke="currentColor" stroke-width="1.2" marker-end="url(#mrcar)"/>
  <rect x="118" y="28" width="88" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="162" y="48" font-size="9" fill="currentColor" text-anchor="middle">× conj(h₀)</text>
  <rect x="118" y="98" width="88" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="162" y="118" font-size="9" fill="currentColor" text-anchor="middle">× conj(h₁)</text>
  <line x1="206" y1="44" x2="280" y2="72" stroke="currentColor" stroke-width="1.2" marker-end="url(#mrcar)"/>
  <line x1="206" y1="114" x2="280" y2="86" stroke="currentColor" stroke-width="1.2" marker-end="url(#mrcar)"/>
  <circle cx="295" cy="79" r="14" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="295" y="83" font-size="12" fill="currentColor" text-anchor="middle">Σ</text>
  <line x1="309" y1="79" x2="375" y2="79" stroke="currentColor" stroke-width="1.3" marker-end="url(#mrcar)"/>
  <text x="415" y="76" font-size="9" fill="currentColor" text-anchor="middle">SNR = γ₀ + γ₁</text>
  <defs><marker id="mrcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each branch is multiplied by the conjugate of its channel estimate — co-phasing it and scaling it by its own strength — before the sum, so every branch contributes in proportion to its quality.</figcaption>
</figure>

## How it works

If branch *k* receives `x_k = h_k·s + n_k` — the transmitted signal `s` through a complex
channel gain `h_k`, plus independent noise of equal power — the MRC output is

`y = Σ conj(h_k)·x_k / Σ |h_k|²`.

Multiplying by `conj(h_k)` does two jobs at once: it rotates every branch to a common phase
(so the signal copies add coherently, amplitude-on-amplitude) and it weights each branch by
`|h_k|` (so a strong branch counts more than a weak one). Signal amplitudes then add
linearly while the independent noises add only in power, and the combined SNR works out to
exactly `γ₀ + γ₁ + …` — the "maximal ratio" that no other linear weighting can beat under
additive white noise. Two equal branches gain 3 dB even with no fading; under
[Rayleigh fading](/reference/rayleigh-fading/) the real win is the diversity order, because
both branches must fade *simultaneously* before the output does.

The price is the `h_k` themselves: MRC needs a complex
[channel estimate](/reference/channel-estimation/) per branch, which means either a training
sequence or a blind estimator, and a full coherent receiver chain per antenna. The cheaper
combining rules — selection, switched, equal-gain — exist precisely to avoid that cost; see
[antenna diversity](/reference/antenna-diversity/) for the family.

## Limits worth knowing

- **MRC assumes the impairment is noise.** Against a directional co-channel interferer it
  is actively wrong — it weights the branch where the interferer is *loudest* the most.
  Nulling interference instead requires
  [interference rejection combining](/reference/interference-rejection-combining/).
- **A single complex weight is frequency-flat.** Combining a wideband stream with one
  scalar per branch is exact only if the branches differ by a frequency-flat constant.
  Antennas metres apart give each carrier its own phase difference, so a scalar aligns
  whichever carrier dominates the cross-power and partially cancels the rest; the fix is
  combining per channel, after the [DDC](/reference/digital-down-converter/).
- **A delay is not a gain.** A constant inter-branch timing skew dilutes broadband
  [coherence](/reference/coherence/) even though per-frequency coherence stays near 1, and
  no scalar weight can represent it — the branches must be time-aligned first (see
  [fractional-delay filter](/reference/fractional-delay-filter/)).

## Relevance to SDR

MRC is the receive-side workhorse of Wi-Fi, cellular and [MIMO](/reference/mimo/) systems,
and it is exactly what a dual-channel SDR front end (USRP with two daughterboards, a
dual-channel AD9361) makes possible in software. GopherTrunk implements wideband MRC in its
SoapyRemote driver (`internal/dsp/diversity/mrc.go`, enabled with `diversity: mrc` or
`mrc-static` on a two-channel device): branch gains are estimated by least squares against a
reference branch, gated on measured [coherence](/reference/coherence/) rather than absolute
level, and either frozen once (`mrc-static`, right for shared-LO front ends) or tracked
continuously (`mrc`, right for independent-PLL daughterboards). The reference branch's
weight is pinned to `1+0j`, which anchors the output phase and keeps the combiner safe ahead
of [differential decoders](/reference/differential-decoding/). The practical
lessons — coherence gates, skew alignment, and when MRC cannot help — are collected in
[MRC diversity gotchas](/reference/mrc-diversity-gotchas/).

## Sources

[^wiki]: [Maximal-ratio combining](https://en.wikipedia.org/wiki/Maximal-ratio_combining) — Wikipedia, on conjugate-weight combining and the summed-SNR optimality result.
