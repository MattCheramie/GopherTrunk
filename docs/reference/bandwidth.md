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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A spectrum bump occupying a span of frequency, with the width of the occupied span labelled bandwidth." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M150 100 C 190 100, 195 35, 230 35 C 265 35, 270 100, 310 100" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.6"/>
  <line x1="150" y1="55" x2="310" y2="55" stroke="currentColor" marker-start="url(#bws)" marker-end="url(#bwe)"/>
  <text x="230" y="48" text-anchor="middle" font-size="11" fill="currentColor">bandwidth</text>
  <text x="230" y="118" text-anchor="middle" font-size="10" fill="currentColor">frequency →</text>
  <defs>
    <marker id="bws" markerWidth="8" markerHeight="8" refX="2" refY="3" orient="auto"><path d="M6 0 L0 3 L6 6 z" fill="currentColor"/></marker>
    <marker id="bwe" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker>
  </defs>
</svg>
<figcaption>Bandwidth is the span of frequency a signal occupies — wider signals carry more data but use more spectrum.</figcaption>
</figure>

## How it works

Wider bandwidth can carry more information but uses more spectrum and demands a higher
[sample rate](/reference/sample-rate/) to capture (per the
[Nyquist theorem](/reference/nyquist-theorem/)). It also admits more noise, affecting
[SNR](/reference/signal-to-noise-ratio/).

## Relevance to SDR

An SDR's capture bandwidth (≈ its sample rate) sets how much spectrum you see at once.
[Filtering and decimation](/reference/decimation/) narrow a wide capture down to a
single channel's bandwidth.
