---
slug: enob
title: Effective number of bits (ENOB)
entry_type: term
category: sdr-dsp
description: "ENOB is a converter's real resolution measured from its SINAD: it reports how many ideal bits the ADC actually delivers once noise and distortion are counted."
keywords: ENOB, effective number of bits, SINAD, ADC resolution, real bits, quantization, dynamic range, converter performance, 6.02 dB per bit
aka: [effective number of bits]
autolink: true
infobox:
  - { label: Type, value: Converter performance metric }
  - { label: Formula, value: "(SINAD − 1.76) / 6.02" }
  - { label: Unit, value: Bits }
see_also: [quantization, dynamic-range, analog-to-digital-converter, dbfs, sinad, dither]
cite_urls:
  - https://en.wikipedia.org/wiki/Effective_number_of_bits
  - https://en.wikipedia.org/wiki/Signal-to-noise_and_distortion_ratio
---

The **effective number of bits** (**ENOB**) is a measure of how many bits of real
resolution a converter delivers, derived from its measured
[SINAD](/reference/sinad/) rather than its nominal bit count.[^wiki] A converter may be
sold as "12-bit," but once its own noise, distortion, and nonlinearity are folded in, it
might behave like a perfect 10-bit part. ENOB puts a single, honest number on that gap by
asking: *how many ideal bits would give the signal quality this converter actually
achieves?*

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A bar comparing a converter's nominal twelve bits against a shorter bar of about ten effective bits, with the difference labelled as loss from noise and distortion." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="40" font-size="9" fill="currentColor">nominal</text>
  <rect x="90" y="28" width="330" height="20" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/>
  <text x="425" y="43" font-size="9" fill="currentColor">12</text>
  <text x="20" y="85" font-size="9" fill="currentColor">ENOB</text>
  <rect x="90" y="73" width="270" height="20" fill="currentColor" fill-opacity="0.35" stroke="currentColor"/>
  <text x="365" y="88" font-size="9" fill="currentColor">≈10</text>
  <rect x="360" y="73" width="60" height="20" fill="none" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="140" y="120" font-size="8" fill="currentColor">lost to noise + distortion (measured via SINAD)</text>
</svg>
<figcaption>ENOB is the fraction of the nominal resolution that survives real-world noise and distortion, read out from the converter's SINAD.</figcaption>
</figure>

## How it works

For an ideal *N*-bit uniform [quantizer](/reference/quantization/), a full-scale sine wave
has a signal-to-quantization-noise ratio of 6.02·*N* + 1.76 dB. ENOB simply inverts that
relationship using the converter's *measured* SINAD — the ratio of signal power to the sum
of all noise **and** distortion power:

> ENOB = (SINAD − 1.76 dB) / 6.02

Because SINAD counts thermal noise, harmonic distortion, clock jitter, and quantization
error together, ENOB captures everything that degrades the converter below its ideal, not
just the step size. A 14-bit ADC with a SINAD of 74 dB has an ENOB of about 12 — its two
"missing" bits are consumed by real impairments.

Two consequences matter:

- **ENOB is not a fixed number.** It falls as the input frequency rises (aperture jitter
  and settling errors grow with frequency) and can change with amplitude and sample rate,
  so it is quoted at specific test conditions.
- **The 6 dB-per-bit rule works both ways.** Every 6 dB of SINAD lost to real-world
  effects costs one effective bit; that is why datasheets show ENOB rolling off across a
  converter's usable band.

## In practice

ENOB is the number to compare when the nominal bit counts mislead. A 16-bit audio codec
running near DC may deliver close to its full resolution, while a fast 12-bit RF converter
sampling at hundreds of megahertz might show an ENOB of 9 or 10 at the top of its band.
Techniques like [dither](/reference/dither/) and [oversampling](/reference/oversampling/)
can raise the *in-band* ENOB above the nominal count by moving quantization noise out of
the band of interest, which is why a well-designed system sometimes measures more
effective bits than its converter nominally has.

## Relevance to SDR

For an SDR, ENOB — not the advertised bit depth — sets the real
[dynamic range](/reference/dynamic-range/) and therefore how well a weak signal survives
next to a strong one across the captured band. A receiver claiming 16 bits but delivering
an ENOB of 11 has roughly 30 dB less usable range than the headline suggests, which shows
up directly as a higher [dBFS](/reference/dbfs/) noise floor and worse blocking
performance. When comparing SDR front ends for wideband trunking work, ENOB at the
operating frequency is the honest figure of merit.

GopherTrunk decodes whatever samples the device delivers, so a capture's usable weak-signal
margin is bounded by the source's ENOB long before the software sees the data — another
reason the decode chain is only as good as the front end feeding it.

## Sources

[^wiki]: [Effective number of bits](https://en.wikipedia.org/wiki/Effective_number_of_bits) — Wikipedia, on deriving real converter resolution from measured SINAD.
