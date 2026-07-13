---
slug: oversampling
title: Oversampling
entry_type: term
category: sdr-dsp
description: "Oversampling samples well above the Nyquist rate, spreading quantization noise over a wider band so filtering and decimation recover resolution and ease anti-aliasing."
keywords: oversampling, processing gain, noise shaping, decimation, sigma-delta, oversampling ratio, Nyquist rate, anti-alias filter, dynamic range
aka: [oversampled conversion]
autolink: true
infobox:
  - { label: Type, value: Sampling strategy }
  - { label: Idea, value: Sample above Nyquist, then decimate }
  - { label: Gain, value: "~3 dB per doubling (more with shaping)" }
see_also: [decimation, quantization, sample-rate, nyquist-theorem, dither, analog-to-digital-converter]
cite_urls:
  - https://en.wikipedia.org/wiki/Oversampling
  - https://en.wikipedia.org/wiki/Delta-sigma_modulation
---

**Oversampling** means sampling a signal at a [rate](/reference/sample-rate/)
substantially higher than the minimum the [Nyquist theorem](/reference/nyquist-theorem/)
requires — often many times higher.[^wiki] The extra samples are not wasted: because a
converter's total [quantization](/reference/quantization/) noise is fixed in *power* but
now spread across a much wider band, only a fraction of it falls in the band of interest.
Filtering off the rest and [decimating](/reference/decimation/) back down recovers
resolution, relaxes the analog anti-alias filter, and cleans up the signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two spectra with equal total quantization noise: at the Nyquist rate the noise is packed into a narrow band, while oversampling spreads the same noise power thinly across a wide band, so the in-band portion under the signal is much smaller." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="200" y2="70" stroke="currentColor"/>
  <rect x="30" y="45" width="70" height="25" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <path d="M40 45 Q65 20 90 45" fill="none" stroke="currentColor"/>
  <text x="34" y="90" font-size="8" fill="currentColor">Nyquist rate</text>
  <text x="34" y="102" font-size="8" fill="currentColor">noise packed in-band</text>
  <line x1="250" y1="70" x2="445" y2="70" stroke="currentColor"/>
  <rect x="250" y="60" width="185" height="10" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/>
  <path d="M262 60 Q287 35 312 60" fill="none" stroke="currentColor"/>
  <line x1="250" y1="35" x2="322" y2="35" stroke="currentColor" stroke-dasharray="2 2" stroke-opacity="0.6"/>
  <text x="254" y="90" font-size="8" fill="currentColor">oversampled</text>
  <text x="254" y="102" font-size="8" fill="currentColor">same noise spread thin; keep only band</text>
</svg>
<figcaption>Equal total quantization-noise power fills the narrow Nyquist band (left) but only a sliver of the wide oversampled band (right); a low-pass filter discards the rest.</figcaption>
</figure>

## How it works

The **oversampling ratio** (OSR) is the actual sample rate divided by twice the signal
bandwidth. Quantization noise power is essentially constant regardless of rate, so
sampling faster distributes that power over a proportionally wider spectrum — the
[power spectral density](/reference/power-spectral-density/) of the noise drops. A digital
low-pass filter then keeps only the signal band and rejects the noise outside it. Each
**doubling** of the sample rate removes half the in-band noise, worth about 3 dB, or half
a bit of extra [effective resolution](/reference/enob/) — this is the **processing gain**
of oversampling. After filtering, [decimation](/reference/decimation/) throws away the now
redundant samples to return to an efficient rate.

**Noise shaping** does far better. A delta-sigma (Σ-Δ) modulator feeds back the
quantization error so that most of the noise is pushed *up* to high frequencies, away from
the signal band, before the decimation filter removes it. With shaping, each doubling of
OSR can buy far more than 3 dB — this is how a 1-bit Σ-Δ converter achieves 16 or more
effective bits at audio bandwidths.

## In practice

Beyond resolution, oversampling relaxes the analog **anti-alias filter**. At the Nyquist
rate the filter must transition from passband to full rejection within a razor-thin guard
band, demanding a steep, expensive analog design. Oversample, and the first alias sits far
away, so a gentle analog filter suffices and the sharp cutoff is done digitally where it
is cheap and precise. Oversampling also pairs with [dither](/reference/dither/): the added
noise keeps the quantization error smooth so the processing gain is genuine noise
reduction rather than smeared distortion.

The cost is throughput — more samples per second to move and process — which is why
systems oversample at the ADC and then decimate to a manageable working rate.

## Relevance to SDR

Oversampling is pervasive in SDR. Converters run far above the wanted channel bandwidth,
and the receiver's [digital down-converter](/reference/digital-down-converter/) and
decimating filters recover both processing gain and a clean channel at a lower rate. It is
also why capturing at a higher sample rate than a single channel needs can help: the wider
capture spreads quantization noise, and the channelising filter reclaims dynamic range for
the channel you keep. The same decimation chains built on
[CIC](/reference/cic-filter/) and [half-band](/reference/half-band-filter/) filters that
step a wide capture down to a channel rate are the practical machinery of oversampling.

GopherTrunk's DSP chain oversamples in exactly this sense: it takes a wide IQ capture and
decimates each channel down to the per-protocol symbol rate, so the filtering that isolates
a control channel also delivers the processing gain that oversampling promises.

## Sources

[^wiki]: [Oversampling](https://en.wikipedia.org/wiki/Oversampling) — Wikipedia, on sampling above Nyquist to spread quantization noise, relax anti-aliasing, and gain resolution by decimation.
