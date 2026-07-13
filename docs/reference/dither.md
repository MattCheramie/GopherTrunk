---
slug: dither
title: Dither
entry_type: term
category: sdr-dsp
description: "Dither is a small amount of noise added before quantization that decorrelates the rounding error, trading harmonic distortion for a smoother, more benign noise floor."
keywords: dither, dithering, quantization, decorrelation, TPDF, triangular dither, noise shaping, ADC linearity, harmonic distortion
aka: [dithering]
autolink: true
infobox:
  - { label: Type, value: Pre-quantization noise }
  - { label: Purpose, value: Decorrelate quantization error }
  - { label: Trade, value: Distortion → noise }
see_also: [quantization, analog-to-digital-converter, enob, dbfs, dynamic-range, oversampling]
cite_urls:
  - https://en.wikipedia.org/wiki/Dither
  - https://en.wikipedia.org/wiki/Quantization_(signal_processing)
---

**Dither** is a small, deliberately added amount of noise applied to a signal *before*
[quantization](/reference/quantization/), used to break the correlation between the signal
and the rounding error.[^wiki] Without it, a low-level or slowly changing signal produces
quantization error that tracks the signal and shows up as ugly harmonic
**distortion**; with it, that error is randomised into a smooth, uncorrelated noise floor.
It is a case where adding noise makes the result *more* faithful, not less.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="On the left a small signal snaps to the same quantization level producing a distorted stepped output; on the right the same signal with added dither noise crosses levels so its average follows the true value." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.25"><line x1="20" y1="40" x2="210" y2="40"/><line x1="20" y1="75" x2="210" y2="75"/><line x1="20" y1="110" x2="210" y2="110"/></g>
  <path d="M20 92 Q115 55 210 92" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <polyline points="20,75 210,75" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="30" y="130" font-size="8" fill="currentColor">no dither: stuck on one level</text>
  <g stroke="currentColor" stroke-opacity="0.25"><line x1="250" y1="40" x2="440" y2="40"/><line x1="250" y1="75" x2="440" y2="75"/><line x1="250" y1="110" x2="440" y2="110"/></g>
  <path d="M250 92 Q345 55 440 92" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <polyline points="250,75 270,40 290,75 310,40 330,40 350,75 370,40 390,75 410,40 440,75" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="258" y="130" font-size="8" fill="currentColor">dithered: average tracks the signal</text>
</svg>
<figcaption>A tiny signal that would snap to one level (left) instead flickers across levels once dither is added (right), so its average encodes the true value.</figcaption>
</figure>

## How it works

An ideal quantizer's error is only "noise-like" when the signal crosses many levels
unpredictably. When a signal sits between two levels, or moves less than one
[LSB](/reference/quantization/), plain rounding always picks the same level, so the error
is a deterministic function of the input — which is exactly what produces
signal-dependent [harmonic](/reference/harmonics/) tones. Dither adds a controlled random
perturbation of about one LSB before the rounding step. Now the chosen level flickers, and
the *probability* of rounding up versus down encodes the sub-LSB value. Averaged over many
samples (or across a band after filtering), the output tracks the true signal with
resolution finer than a single level, and the leftover error is spread into a flat,
signal-independent noise floor.

The classic choice is **triangular-PDF (TPDF) dither**, roughly two LSB peak-to-peak,
because it makes both the mean and the variance of the quantization error independent of
the input — the strongest guarantee that no distortion tone survives.

## In practice

Dither costs a little: the noise floor rises modestly because you added noise on purpose.
The payoff is that the artefacts that remain are benign broadband noise rather than
spurious tones that a listener or a spectral display would flag as signals. In many
converters the dither is effectively free — thermal noise in the analog front end already
provides an LSB or more of natural dithering, which is one reason a slightly noisy input
can quantize more cleanly than a mathematically perfect one.

Dither pairs naturally with [oversampling](/reference/oversampling/): sampling faster
spreads the same quantization-noise power over a wider band, so filtering afterward
recovers resolution, and dither guarantees that power is smooth enough to filter. Together
they can push the [effective number of bits](/reference/enob/) above the nominal count in
the band of interest.

## Relevance to SDR

Dither matters most for weak-signal work near an SDR's noise floor, where a low-level
carrier might otherwise generate spurious "birdies" from correlated quantization error
instead of appearing as a clean tone above smooth noise. On the low-bit-depth converters
in cheap SDRs, the ever-present thermal noise usually supplies enough natural dither that
the quantization stays well behaved; some receiver and audio designs add dither
explicitly. Reading a [dBFS](/reference/dbfs/) spectrum, a flat noise floor rather than a
picket fence of tiny tones is the sign that dithering (natural or deliberate) is doing its
job.

GopherTrunk decodes the samples it is given and does not dither the ADC itself, but the
principle explains why a marginally noisy capture can yield cleaner spectra — and more
reliable soft-decision inputs — than an artificially quiet one.

## Sources

[^wiki]: [Dither](https://en.wikipedia.org/wiki/Dither) — Wikipedia, on adding noise before quantization to decorrelate and linearise the quantization error.
