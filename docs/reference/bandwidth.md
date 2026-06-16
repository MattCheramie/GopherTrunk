---
slug: bandwidth
title: Bandwidth
entry_type: term
category: rf-fundamentals
description: Bandwidth is the width of the frequency range a signal occupies or a receiver captures, measured in hertz; it bounds data rate and sets capture requirements.
keywords: bandwidth, Hz, channel width, occupied bandwidth, capture bandwidth
infobox:
  - { label: Symbol, value: B }
  - { label: Unit, value: Hertz (Hz) }
  - { label: Determines, value: Data capacity, capture needs }
see_also: [sample-rate, nyquist-theorem, frequency, signal-to-noise-ratio]
related_lessons:
  - { title: "Anatomy of a signal", url: /learn/signal-anatomy/ }
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/sample-rate-nyquist/ }
external:
  - { title: "Bandwidth (signal processing) (Wikipedia)", url: https://en.wikipedia.org/wiki/Bandwidth_(signal_processing) }
---

**Bandwidth** is the width, in hertz, of the [frequency](/reference/frequency/) range a
signal occupies or that a receiver captures. A narrowband voice channel may be ~12.5
kHz wide; an FM broadcast station ~200 kHz; Wi-Fi tens of megahertz.

## How it works

Wider bandwidth can carry more information but uses more spectrum and demands a higher
[sample rate](/reference/sample-rate/) to capture (per the
[Nyquist theorem](/reference/nyquist-theorem/)). It also admits more noise, affecting
[SNR](/reference/signal-to-noise-ratio/).

## Relevance to SDR

An SDR's capture bandwidth (≈ its sample rate) sets how much spectrum you see at once.
[Filtering and decimation](/reference/decimation/) narrow a wide capture down to a
single channel's bandwidth.
