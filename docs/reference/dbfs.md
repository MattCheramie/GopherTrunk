---
slug: dbfs
title: dBFS
entry_type: term
category: rf-fundamentals
description: dBFS is level expressed in decibels relative to digital full scale, used inside an SDR; 0 dBFS is the ADC's maximum, and exceeding it causes clipping.
keywords: dBFS, decibels full scale, ADC, clipping, headroom, digital level, overload
aka: [dBFS]
autolink: true
infobox:
  - { label: Type, value: Digital level unit }
  - { label: Reference, value: ADC full scale (0 dBFS max) }
  - { label: Risk at 0 dBFS, value: Clipping / distortion }
see_also: [decibel, dbm, analog-to-digital-converter, automatic-gain-control, dynamic-range, quantization]
related_lessons:
  - { title: "Gain, AGC & avoiding overload", url: /learn/rf-sdr/gain-and-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/DBFS
  - https://en.wikipedia.org/wiki/Clipping_(audio)
---

**dBFS** (decibels relative to **full scale**) measures level inside the digital domain
of an SDR. Here **0 dBFS is the ceiling** — the largest value the
[ADC](/reference/analog-to-digital-converter/) can represent — and all real samples sit
below it as negative numbers.[^wiki] Unlike [dBm](/reference/dbm/), it describes a
digital code range, not a physical power.

<figure class="figure" markdown="0">
<svg viewBox="0 0 320 160" role="img" aria-label="A vertical scale with 0 dBFS at the top as the ADC ceiling, a signal sitting below it with headroom above and the quantization noise floor near the bottom." xmlns="http://www.w3.org/2000/svg">
  <rect x="120" y="16" width="60" height="128" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="120" y1="26" x2="180" y2="26" stroke="currentColor" stroke-width="2"/>
  <text x="190" y="29" font-size="10" fill="currentColor">0 dBFS (clip)</text>
  <rect x="120" y="26" width="60" height="30" fill="currentColor" fill-opacity="0.12"/>
  <text x="190" y="48" font-size="10" fill="currentColor">headroom</text>
  <rect x="120" y="56" width="60" height="54" fill="currentColor" fill-opacity="0.25"/>
  <text x="190" y="86" font-size="10" fill="currentColor">signal</text>
  <line x1="120" y1="126" x2="180" y2="126" stroke="currentColor" stroke-dasharray="3 3"/>
  <text x="190" y="129" font-size="10" fill="currentColor">quantization floor</text>
  <text x="150" y="156" font-size="9" fill="currentColor" text-anchor="middle">span = dynamic range</text>
</svg>
<figcaption>dBFS is the digital scale inside the SDR; 0 dBFS is the ADC ceiling, real samples sit below it, and the span down to the quantization floor is the converter's dynamic range.</figcaption>
</figure>

## How it works

Full scale is fixed by the converter's bit depth: an N-bit ADC has a largest
representable magnitude, and 0 dBFS is defined as exactly that. Every actual sample is a
fraction of full scale, so its level in dBFS is negative — a sample at half of full scale
is −6 dBFS, at one tenth is −20 dBFS. Because it is a level referenced to a maximum, dBFS
values *cannot* be positive; a real converter simply cannot output more than full scale.

The importance of the ceiling is what happens at it. If the analog input drives the ADC
past full scale, the converter **clips**: peaks are flattened to the maximum code, and
that flat-topping is a hard nonlinearity.[^clip] Clipping does not just distort the one
strong signal — it generates harmonics and
[intermodulation](/reference/intermodulation/) products that spray across the whole
captured spectrum, creating spurs and raising the apparent
[noise floor](/reference/noise-floor/). A single overloading pager or FM broadcast carrier
can wipe out weak signals many megahertz away.

The bottom of the scale is set by [quantization](/reference/quantization/) noise — the
rounding error of representing a continuous voltage in finite bits. The distance from
0 dBFS down to that quantization floor is the converter's
[dynamic range](/reference/dynamic-range/), roughly 6 dB per bit. An 8-bit RTL-SDR
therefore has far less digital headroom than a 12- or 14-bit receiver, which is why gain
staging matters more on cheap hardware.

## In practice

The goal of setting front-end gain is to place the **strongest** signal in the passband
a comfortable margin below 0 dBFS — often 6 to 12 dB of headroom — so momentary peaks do
not clip, while still lifting the weakest signal well above the quantization floor. Too
little gain buries weak signals in quantization noise; too much gain clips on strong
ones. Software overload or "ADC clip" indicators watch for samples reaching full scale.

dBFS and dBm describe two different worlds. dBm is the physical power arriving at the
antenna; dBFS is the code level after the analog gain chain and the ADC. There is no
universal offset between them — the same −80 dBm signal can land at −40 dBFS or −10 dBFS
depending on how much gain precedes the converter. Only a calibrated receiver ties the
two scales together.

## Relevance to SDR

Overload is one of the most common causes of a receiver that "hears nothing" despite a
strong antenna: a nearby transmitter drives the ADC into clipping and the whole band
degrades. GopherTrunk's DSP operates on the sample stream well after the ADC, so it
cannot undo clipping that already happened in hardware — the fix is
[gain](/reference/automatic-gain-control/), attenuation, or filtering ahead of the
converter, keeping peaks under 0 dBFS. This is exactly the gain-staging distinction noted
in GopherTrunk's DSP notes: a symptom baked into captured samples cannot be recovered by
downstream processing.

## Sources

[^wiki]: [dBFS](https://en.wikipedia.org/wiki/DBFS) — Wikipedia, decibels relative to digital full scale and the impossibility of positive values.
[^clip]: [Clipping](https://en.wikipedia.org/wiki/Clipping_(audio)) — Wikipedia, how driving a converter past full scale flattens peaks and generates distortion products.
