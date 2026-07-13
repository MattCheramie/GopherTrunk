---
slug: mmse-equalizer
title: MMSE equalizer
entry_type: algorithm
category: equalization
description: An MMSE equalizer minimizes the mean-square error between its output and the transmitted symbols, balancing intersymbol-interference removal against noise enhancement.
keywords: MMSE equalizer, minimum mean square error, linear equalizer, ISI, noise enhancement, Wiener filter, zero forcing comparison, adaptive equalizer, low SNR
aka: [MMSE equalizer, minimum mean square error equalizer]
autolink: true
infobox:
  - { label: Type, value: Linear equalizer }
  - { label: Criterion, value: Minimize mean-square error (ISI + noise) }
  - { label: Strength, value: Better than ZF at low SNR }
see_also: [zero-forcing-equalizer, adaptive-filter, lms-algorithm, decision-feedback-equalizer, signal-to-noise-ratio, constellation-diagram]
cite_urls:
  - https://en.wikipedia.org/wiki/Minimum_mean_square_error
  - https://en.wikipedia.org/wiki/Wiener_filter
---

An **MMSE (minimum mean-square-error) equalizer** is a linear filter whose taps are chosen
to minimise the average squared difference between its output and the transmitted symbols,
so it trades off residual **intersymbol interference (ISI)** against **noise
enhancement** instead of eliminating one at the expense of the other.[^wiki] It is the
[Wiener-filter](/reference/kalman-filter/) solution applied to equalization, and it
outperforms the [zero-forcing](/reference/zero-forcing-equalizer/) equalizer whenever noise
is non-negligible.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Two error-versus-SNR curves: the zero-forcing equalizer has high error at low SNR while the MMSE equalizer stays lower, and the two curves converge at high SNR." xmlns="http://www.w3.org/2000/svg">
  <g fill="none" stroke="currentColor">
    <line x1="50" y1="20" x2="50" y2="135" stroke-width="1.1"/>
    <line x1="50" y1="135" x2="435" y2="135" stroke-width="1.1"/>
    <path d="M60 35 C 150 60, 300 128, 430 132" stroke-width="1.2" stroke-dasharray="4 3" stroke-opacity="0.7"/>
    <path d="M60 90 C 150 110, 300 130, 430 132" stroke-width="1.6"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="42" y="26" text-anchor="end">error</text>
    <text x="430" y="150" text-anchor="end">SNR →</text>
    <text x="120" y="52" fill-opacity="0.8">zero-forcing</text>
    <text x="120" y="100">MMSE</text>
    <text x="360" y="122" font-size="8">converge</text>
  </g>
</svg>
<figcaption>At low SNR the ZF equalizer's noise enhancement inflates its error, while MMSE stays lower by tolerating a little ISI; as SNR rises the two solutions converge.</figcaption>
</figure>

## How it works

Rather than inverting the channel, MMSE solves for the tap vector **w** that minimises the
expected squared error `E[|d − wᵀx|²]` between the equalizer output and the true symbol
`d`. The solution is the **Wiener–Hopf** equation `w = R⁻¹·p`, where `R` is the
autocorrelation of the received samples and `p` is their cross-correlation with the desired
symbol. In the frequency domain this is approximately

`C(f) = H*(f) / (‖H(f)‖² + σ²/S)`,

where `H` is the channel, `σ²` the noise power and `S` the signal power. The `+ σ²/S` term
in the denominator is the whole point: near a spectral null, where `‖H‖²` is tiny, that
term keeps the gain finite instead of exploding to `1/H` the way
[zero forcing](/reference/zero-forcing-equalizer/) does.

- **High SNR** (`σ² → 0`): the extra term vanishes and MMSE **reduces to the ZF
  solution** — full channel inversion, zero ISI.
- **Low SNR**: MMSE deliberately leaves some ISI uncorrected to avoid amplifying noise,
  minimising the *combined* penalty and delivering a cleaner
  [constellation](/reference/constellation-diagram/) and higher effective output
  [SNR](/reference/signal-to-noise-ratio/).

## In practice

The Wiener solution needs the channel and noise statistics, which are rarely known in
advance, so MMSE equalizers are almost always realised *adaptively*: an
[LMS](/reference/lms-algorithm/) or [RLS](/reference/rls-algorithm/) loop, trained on a
known sequence and then run decision-directed, converges toward the MMSE tap set without
ever explicitly inverting a matrix. The same MMSE criterion also underlies the
feed-forward section of a [decision-feedback equalizer](/reference/decision-feedback-equalizer/)
and MMSE detectors in MIMO/OFDM receivers, where it balances stream interference against
noise.

## Relevance to SDR

MMSE is the default linear equalizer in practical digital radio because it degrades
gracefully on the noisy, fading channels real receivers face — DSL, LTE/5G, Wi-Fi and
microwave links all use MMSE (or MMSE-DFE) equalization or detection. Compared with the
zero-forcing baseline it buys a meaningful low-SNR margin for essentially the same
implementation cost. GopherTrunk's narrowband P25/DMR/NXDN decoders rely on
[matched filtering](/reference/matched-filter/) and synchronisation of
[RRC-shaped](/reference/root-raised-cosine-filter/) signals rather than a general MMSE
equalizer, so MMSE is covered here as a core equalization concept from the wider RF and
communications world.

## Sources

[^wiki]: [Minimum mean square error](https://en.wikipedia.org/wiki/Minimum_mean_square_error) — Wikipedia, on the MMSE criterion, the Wiener solution, and its equalization use.
