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

## How it works

SNR = signal level − noise-floor level (both in [dBm](/reference/dbm/)). A signal at
−85 dBm over a −105 dBm floor has 20 dB of SNR. Each digital mode has a minimum SNR
below which [demodulation](/reference/demodulation/) starts dropping symbols.

## Relevance to SDR

Improving SNR — better antenna, placement, correct gain — is usually what moves a
marginal signal from un-decodable to clean.
