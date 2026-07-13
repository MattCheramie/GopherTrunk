---
slug: resampler
title: Resampler
entry_type: algorithm
category: filtering-multirate
description: A resampler converts a stream from one sample rate to another — essential when an SDR's native rate doesn't match the rate a decoder needs, by a rational or arbitrary ratio.
keywords: resampler, resampling, sample-rate conversion, interpolation, decimation, rational resampling, fractional resampling, arbitrary resampling, polyphase, Farrow
aka: [resampler, resampling, sample-rate converter]
autolink: true
infobox:
  - { label: Type, value: Sample-rate converter }
  - { label: Ratio, value: Rational L/M or arbitrary }
  - { label: Core, value: Interpolate → filter → decimate }
see_also: [decimation, polyphase-filter-bank, half-band-filter, sample-rate, nyquist-theorem]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
related_reading:
  - { title: "SDR Internals, Part 5: Tuning & channelization", url: /blog/deep-dives/sdr-internals-05-tuning-channelization/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Sample-rate_conversion
  - https://en.wikipedia.org/wiki/Farrow_structure
---

A **resampler** converts a sample stream from one [sample rate](/reference/sample-rate/) to
another.[^wiki] SDRs rarely produce exactly the rate a decoder wants — a
[P25](/reference/p25-phase-1/) channel needs a clean multiple of its 4800-baud symbol rate,
say, while the radio delivers 2.4 or 10 MS/s — so a resampler bridges the two. It can do so by
a whole-number ratio, a rational ratio *L/M*, or an entirely arbitrary (irrational, even
time-varying) ratio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An input stream of samples at one spacing is converted by a resampler into an output stream at a different spacing, while the underlying waveform is preserved." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 40 Q 95 12 170 40 T 320 40" fill="none" stroke="currentColor" stroke-opacity="0.35" stroke-width="1.1"/>
  <g fill="currentColor"><circle cx="30" cy="40" r="3"/><circle cx="58" cy="34" r="3"/><circle cx="86" cy="40" r="3"/><circle cx="114" cy="47" r="3"/><circle cx="142" cy="40" r="3"/><circle cx="170" cy="34" r="3"/></g>
  <text x="100" y="70" font-size="8.5" fill="currentColor" text-anchor="middle">input rate (6 samples)</text>
  <line x1="200" y1="90" x2="244" y2="90" stroke="currentColor" marker-end="url(#rsar)"/><text x="222" y="83" text-anchor="middle" font-size="8" fill="currentColor">resample</text>
  <g fill="currentColor"><circle cx="278" cy="100" r="3"/><circle cx="322" cy="100" r="3"/><circle cx="366" cy="100" r="3"/><circle cx="410" cy="100" r="3"/></g>
  <text x="344" y="122" font-size="8.5" fill="currentColor" text-anchor="middle">output rate (4 samples, same waveform)</text>
  <defs><marker id="rsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A resampler changes the sample rate — here reducing it — by reconstructing the underlying continuous waveform and re-sampling it at the new grid, never simply dropping samples.</figcaption>
</figure>

## How it works

The correct mental model is *reconstruct then re-sample*: recover the underlying
band-limited waveform implied by the input samples, then read off its value on the new
sample grid. Done naively — dropping or duplicating samples — this folds energy across the
band and creates [aliasing](/reference/aliasing/), so a resampler **always** contains an
anti-alias / anti-image [filter](/reference/digital-filter/) that limits the signal to the
lower of the two Nyquist bands.

For a **rational** rate change *L/M*, the textbook chain is:

- **Upsample by L** — insert `L−1` zeros between input samples, raising the rate.
- **Low-pass filter** — remove the spectral images the zero-stuffing created (and pre-filter
  for the coming decimation), passing only the original band.
- **Downsample by M** — keep every *M*-th filtered sample.

Because the low-pass filter runs at the high intermediate rate `L·fs`, the naive form is
wasteful: it multiplies many zeros and computes many samples that decimation discards. A
[polyphase filter bank](/reference/polyphase-filter-bank/) fixes both — it skips the zero
multiplies and computes only the outputs that survive — turning the interpolate-filter-decimate
chain into an efficient single stage. Large integer factors are peeled off first with cheap
[half-band](/reference/half-band-filter/) stages, leaving the polyphase resampler to handle
the small remaining ratio.

## Variants: rational, fractional, and arbitrary

- **Integer decimation / interpolation.** The simple *L*=1 or *M*=1 cases; one polyphase FIR
  and a rate change.
- **Rational L/M.** Any ratio expressible as a fraction (e.g. 48/44.1 for audio, or matching
  a 4800-baud symbol grid) — a single polyphase resampler with *L* sub-filter phases.
- **Arbitrary / fractional resampling.** When the ratio is irrational or drifts over time —
  as in symbol-timing recovery, where the wanted sample instant sits *between* input
  samples and moves — you need a continuously variable delay. The **Farrow structure** does
  this: it evaluates a low-order polynomial interpolation of the surrounding samples at a
  fractional offset µ, so the same hardware can produce a sample at any inter-sample position
  by adjusting µ.[^farrow] This is exactly how a timing loop pulls samples onto the symbol
  clock without changing the input rate.

## In practice

Resampling appears wherever two clocks meet: matching an SDR's native rate to a decoder's
required rate, converting between audio rates, and — via the Farrow / fractional form — inside
the interpolators of clock- and symbol-timing recovery. The engineering choice is between a
tidy rational polyphase resampler when the ratio is fixed and known, and an arbitrary Farrow
resampler when the ratio is fractional or must be steered in real time.

## Relevance to SDR

Rate conversion is unavoidable in [SDR](/reference/software-defined-radio/): the radio's
native rate almost never equals the per-protocol channel rate a decoder expects. GopherTrunk's
down-conversion normalises each capture to the per-protocol channel rate (for example 48 kHz
for the 4800-baud C4FM family, 144 kHz for TETRA), and its timing recovery interpolates
fractionally to land samples on the symbol instants — both concrete instances of resampling in
the decode chain.

## Sources

[^wiki]: [Sample-rate conversion](https://en.wikipedia.org/wiki/Sample-rate_conversion) — Wikipedia, on rational L/M resampling, the interpolate-filter-decimate chain, and polyphase efficiency.
[^farrow]: [Farrow structure](https://en.wikipedia.org/wiki/Farrow_structure) — Wikipedia, on polynomial-interpolation fractional-delay filters for arbitrary/continuously-variable resampling.
