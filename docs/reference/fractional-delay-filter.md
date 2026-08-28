---
slug: fractional-delay-filter
title: Fractional-delay filter
entry_type: algorithm
category: filtering-multirate
description: A fractional-delay filter delays a sampled signal by a non-integer number of samples, interpolating between samples so two streams can be aligned to a fraction of a sample period.
keywords: fractional delay, interpolating delay line, Farrow structure, sinc interpolation, sample alignment, inter-branch skew, timing alignment, Lagrange interpolation, delay estimation, parabolic interpolation
aka: [fractional delay line, interpolating delay line]
autolink: true
infobox:
  - { label: Type, value: Interpolating FIR structure }
  - { label: Delays by, value: A non-integer number of samples }
  - { label: Common forms, value: "Windowed-sinc FIR, Lagrange, Farrow" }
  - { label: Used for, value: Timing recovery, branch alignment, beam steering }
see_also: [resampler, fir-filter, clock-recovery, matched-filter, coherence, maximal-ratio-combining]
cite_urls:
  - https://en.wikipedia.org/wiki/Interpolation
  - https://en.wikipedia.org/wiki/Whittaker%E2%80%93Shannon_interpolation_formula
---

**A fractional-delay filter** delays a sampled signal by a non-integer number of samples —
2.60 samples, say — by interpolating new sample values *between* the ones that were
captured.[^wiki] An integer delay is free (read the buffer later), but the moment two
streams must be aligned to a fraction of a sample period, a filter has to reconstruct what
the signal was doing between sampling instants. The
[sampling theorem](/reference/nyquist-theorem/) guarantees this is possible for a
band-limited signal: the ideal fractional delay is a shifted sinc, and practical filters are
finite approximations of it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Two sample streams offset by a fraction of a sample period; the early stream passes through an interpolating delay filter and emerges aligned with the late one." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="42" r="2.6"/><circle cx="80" cy="30" r="2.6"/><circle cx="120" cy="46" r="2.6"/><circle cx="160" cy="34" r="2.6"/></g>
  <g fill="currentColor" fill-opacity="0.45"><circle cx="52" cy="90" r="2.6"/><circle cx="92" cy="78" r="2.6"/><circle cx="132" cy="94" r="2.6"/><circle cx="172" cy="82" r="2.6"/></g>
  <text x="100" y="18" font-size="8.5" fill="currentColor" text-anchor="middle">branch 0 (early)</text>
  <text x="112" y="112" font-size="8.5" fill="currentColor" text-anchor="middle">branch 1 (late by 0.3 samples)</text>
  <line x1="185" y1="42" x2="235" y2="58" stroke="currentColor" stroke-width="1.2" marker-end="url(#fdar)"/>
  <rect x="238" y="46" width="104" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="290" y="65" font-size="9" fill="currentColor" text-anchor="middle">delay by 0.3</text>
  <line x1="345" y1="61" x2="395" y2="61" stroke="currentColor" stroke-width="1.2" marker-end="url(#fdar)"/>
  <text x="424" y="64" font-size="9" fill="currentColor" text-anchor="middle">aligned</text>
  <defs><marker id="fdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Delaying the early stream by the measured fractional offset re-times its samples onto the other stream's grid, so the two can be summed sample-for-sample.</figcaption>
</figure>

## How it works

The ideal delay-by-D filter has impulse response `sinc(n − D)` — for integer D that
collapses to a single unit tap, for fractional D it spreads over all n and must be
truncated. Practical structures trade accuracy for cost:

- **Windowed-sinc FIR.** Take a dozen or so taps of the shifted sinc under a window;
  accurate across most of the band, with the usual FIR latency.
- **Lagrange / polynomial interpolation.** Fit a low-order polynomial through neighbouring
  samples and read it off at the fractional position. Linear interpolation is the 1st-order
  case — cheap, but it low-pass filters the signal noticeably.
- **Farrow structure.** Factors a polynomial interpolator so the fractional delay is a
  runtime *parameter* rather than baked into the taps — the standard choice inside
  [timing-recovery](/reference/clock-recovery/) loops, where the delay changes every symbol.

A fractional-delay filter is a [resampler](/reference/resampler/) evaluated at a constant
offset: same mathematics, different intent. Where a resampler changes the rate, a
fractional delay keeps the rate and moves the sampling *phase*.

## Measuring the delay to apply

Alignment jobs pair the filter with an estimator. Cross-correlating the two streams over a
range of integer lags finds the coarse offset; fitting a parabola through the correlation
peak and its two neighbours refines it to a fraction of a sample. The frequency-domain
signature of a pure delay is distinctive and worth recognising: per-frequency
[coherence](/reference/coherence/) stays near 1 while broadband coherence is diluted (each
frequency sees a different phase slope, so the wideband average partially cancels), and
group delay is flat.

## Relevance to SDR

Fractional delays are everywhere timing matters: inside every symbol-timing loop, in
beamforming (a steering delay per element), and in multi-channel alignment. GopherTrunk's
sharpest lesson came from diversity combining: on an X310 with two daughterboards, branch 0
lagged branch 1 by a *constant 2.60 samples* (13 µs at 200 kS/s), and since a single complex
gain cannot represent a delay, [MRC](/reference/maximal-ratio-combining/) on the skewed
branches decoded 22% *fewer* CRC-clean frames than the best branch alone — the combiner was
hurting. The SoapyRemote driver now measures the skew per stream (±16-lag correlation scan
with parabolic refinement, DC-removed, latched behind a coherence gate) and delays the early
branch through an interpolating delay line (`internal/sdr/soapyremote/align.go`); the
identical combine after alignment matched the best branch exactly. The diagnostic — "other
lags ref by N±f samples" — is printed by the diversity replay harness, and the wider story
lives in [MRC diversity gotchas](/reference/mrc-diversity-gotchas/).

## Sources

[^wiki]: [Whittaker–Shannon interpolation formula](https://en.wikipedia.org/wiki/Whittaker%E2%80%93Shannon_interpolation_formula) — Wikipedia, on ideal sinc reconstruction between samples, the basis of fractional-delay filtering.
