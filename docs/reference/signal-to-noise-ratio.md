---
slug: signal-to-noise-ratio
title: Signal-to-noise ratio (SNR)
entry_type: term
category: rf-fundamentals
description: Signal-to-noise ratio is the difference in decibels between a signal's power and the noise floor; digital modes require a minimum SNR to decode reliably.
keywords: SNR, signal to noise ratio, noise floor, dB, decode threshold, SINAD
aka: [signal-to-noise ratio, SNR]
autolink: true
infobox:
  - { label: Symbol, value: SNR }
  - { label: Unit, value: Decibels (dB) }
  - { label: Formula, value: "signal (dBm) − noise floor (dBm)" }
see_also: [noise-floor, decibel, dbm, demodulation]
related_lessons:
  - { title: "Decibels & signal power", url: /learn/decibels/ }
external:
  - { title: "Signal-to-noise ratio (Wikipedia)", url: https://en.wikipedia.org/wiki/Signal-to-noise_ratio }
---

**Signal-to-noise ratio** (**SNR**) is the gap, in [decibels](/reference/decibel/),
between a signal's power and the [noise floor](/reference/noise-floor/). It is the
single best predictor of whether a signal will decode.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A spectrum with a noisy baseline labelled noise floor and a tall peak labelled signal, with the gap between them labelled SNR." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 120 L70 124 L110 116 L150 122 L190 119 L240 121 L300 118 L360 121 L420 119" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <line x1="30" y1="120" x2="430" y2="120" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.5"/>
  <text x="350" y="135" font-size="10" fill="currentColor">noise floor</text>
  <path d="M210 120 L224 45 L238 120 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.5"/>
  <text x="224" y="38" text-anchor="middle" font-size="10" fill="currentColor">signal</text>
  <line x1="260" y1="45" x2="260" y2="120" stroke="currentColor"/>
  <text x="268" y="86" font-size="11" fill="currentColor">SNR</text>
</svg>
<figcaption>SNR is how far a signal rises above the noise floor; digital modes need a minimum SNR to decode.</figcaption>
</figure>

## How it works

SNR = signal level − noise-floor level (both in [dBm](/reference/dbm/)). A signal at
−85 dBm over a −105 dBm floor has 20 dB of SNR. Each digital mode has a minimum SNR
below which [demodulation](/reference/demodulation/) starts dropping symbols.

## Relevance to SDR

Improving SNR — better antenna, placement, correct gain — is usually what moves a
marginal signal from un-decodable to clean.
