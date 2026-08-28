---
slug: coherence
title: Coherence (normalized cross-correlation)
entry_type: term
category: estimation-array
description: "Coherence is the normalized cross-correlation between two signals — a scale-invariant 0-to-1 measure of how much of them is the same underlying signal, readable directly as a per-branch SNR."
keywords: coherence, normalized cross-correlation, correlation coefficient, rho, diversity branches, phase alignment, common signal, SNR estimate, DC offset, bandwidth dilution
aka: [normalized cross-correlation, complex correlation coefficient]
infobox:
  - { label: Type, value: Similarity statistic }
  - { label: Range, value: "0 (unrelated) to 1 (identical up to gain)" }
  - { label: SNR reading, value: "|ρ| = γ/(1+γ) for equal branch SNR γ" }
  - { label: Noise-only floor, value: "≈ √(π/4N) for N samples" }
see_also: [signal-to-noise-ratio, maximal-ratio-combining, channel-estimation, preamble-correlation, dc-offset, dbfs]
related_reading:
  - { title: "The Analog Edge, Part 13: Coherence, Not dBFS", url: /blog/tutorials/analog-edge-13-coherence-not-dbfs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Pearson_correlation_coefficient
  - https://en.wikipedia.org/wiki/Coherence_(signal_processing)
---

**Coherence** — the magnitude of the **normalized cross-correlation** between two complex
signals — measures how much of them is the *same underlying signal*, on a scale from 0
(statistically unrelated) to 1 (identical up to a complex gain), independent of how loud
either one is.[^wiki] For two diversity branches it is computed as

`|ρ| = |Σ x₁·conj(x₀)| / √(Σ|x₀|² · Σ|x₁|²)`

— the cross-power between the branches, normalised by each branch's own power. That
normalisation is the point: gain, attenuation, and front-end scaling all cancel, so ρ
answers "do these two streams carry one signal?" where an absolute level in
[dBFS](/reference/dbfs/) can only answer "is this stream loud?".

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A curve of coherence versus per-branch SNR rising from zero through 0.5 at 0 dB toward 1, with the noise-only floor marked near zero coherence." xmlns="http://www.w3.org/2000/svg">
  <line x1="50" y1="112" x2="430" y2="112" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="50" y1="112" x2="50" y2="20" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M50 106 C 130 104 170 92 240 66 C 300 44 370 32 430 28" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="240" y1="66" x2="240" y2="112" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="3 3"/>
  <line x1="50" y1="66" x2="240" y2="66" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="3 3"/>
  <text x="44" y="69" font-size="8" fill="currentColor" text-anchor="end">0.5</text>
  <text x="240" y="124" font-size="8" fill="currentColor" text-anchor="middle">0 dB</text>
  <text x="44" y="30" font-size="8" fill="currentColor" text-anchor="end">1.0</text>
  <text x="60" y="100" font-size="7.5" fill="currentColor">noise floor √(π/4N)</text>
  <text x="240" y="136" font-size="8.5" fill="currentColor" text-anchor="middle">per-branch SNR γ →</text>
  <text x="30" y="70" font-size="8.5" fill="currentColor" text-anchor="middle" transform="rotate(-90 30 70)">|ρ|</text>
</svg>
<figcaption>|ρ| = γ/(1+γ): coherence 0.5 means each branch sits at 0 dB SNR, and the noise-only floor √(π/4N) is where N samples of pure independent noise land by chance.</figcaption>
</figure>

## Reading the number

Three anchor facts turn ρ from an abstract statistic into an instrument:

- **It is an SNR meter.** For two branches carrying one signal at equal per-branch SNR γ,
  `|ρ| = γ/(1+γ)`: coherence 0.50 means 0 dB per branch, 0.35 means about −2.7 dB. A
  threshold on ρ is therefore a threshold on
  [signal-to-noise ratio](/reference/signal-to-noise-ratio/), stated in a form no gain knob
  can game.
- **It has a chance floor.** N samples of *independent* noise produce `|ρ| ≈ √(π/4N)` just
  from finite averaging — about 0.014 for N = 4096. A measured ρ is meaningful only well
  above that floor, and the floor falls as 1/√N, so longer windows buy resolution.
- **It carries an error bar.** The phase of ρ estimates the inter-branch phase, with a
  standard error of roughly `√((1−ρ²)/(2Nρ²))` — small-looking coherence over a long window
  can still pin phase to a few degrees, which is why gates on the *projected error* behave
  where fixed ρ thresholds do not.

## Two traps

**DC offsets fake coherence.** Both receivers of one front end share
[LO leakage](/reference/dc-offset/), and a common DC term correlates perfectly with itself:
an uncentred correlator fed two branches of independent noise plus common DC reports
`|ρ| → 1` and returns the ratio of the DC offsets as a "channel estimate" — in exactly the
weak-signal regime where the number matters most. Subtract each branch's mean before
correlating; in a correlator this is load-bearing, not hygiene.

**Bandwidth dilutes wideband coherence.** Only the hertz actually carrying the common
signal contribute cross-power, so for in-channel power fractions f₀, f₁ of each branch,
`ρ_wb ≈ ρ_ch·√(f₀·f₁)`. A perfectly coherent narrowband carrier inside a wide noisy capture
can legitimately measure ρ ≈ 0.16 wideband — and a fixed 0.5 threshold then silently makes
"coherent enough" depend on the configured capture bandwidth and each branch's noise floor.
Compare coherence measured before and after the
[DDC](/reference/digital-down-converter/) to separate "the branches disagree" from "the
bandwidth is diluting".

## Relevance to SDR

GopherTrunk's diversity combiner is built on this statistic
(`internal/dsp/diversity/crossstats.go`): [MRC](/reference/maximal-ratio-combining/)
calibration is gated on the phase error projected from ρ rather than on any absolute level,
after an absolute −40 dBFS gate proved to be a gain-staging trap — an operator once raised
front-end gain 65 → 82 dB purely to push a number past a software constant. The same
statistic diagnoses hardware: per-frequency coherence near 1 with diluted broadband ρ is the
signature of a pure inter-branch delay (fixed by a
[fractional-delay filter](/reference/fractional-delay-filter/), not a gain), and a coherence
that no tracking improves marks the wideband-scalar limit of single-gain combining. The
operator-facing symptoms are catalogued in
[MRC diversity gotchas](/reference/mrc-diversity-gotchas/).

## Sources

[^wiki]: [Pearson correlation coefficient](https://en.wikipedia.org/wiki/Pearson_correlation_coefficient) — Wikipedia, on the normalised correlation statistic, its scale invariance, and its sampling behaviour.
