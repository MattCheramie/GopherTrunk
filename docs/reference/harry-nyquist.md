---
slug: harry-nyquist
title: Harry Nyquist
entry_type: person
category: people
description: Harry Nyquist (1889–1976) was a Swedish-American engineer at Bell Labs whose work on sampling and signalling underlies the sampling theorem central to all digital radio.
keywords: Harry Nyquist, Nyquist rate, sampling theorem, Bell Labs, signal processing
aka: [Harry Nyquist, Nyquist]
autolink: true
infobox:
  - { label: Lived, value: "1889–1976" }
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: Sampling and signalling theory }
see_also: [nyquist-theorem, sample-rate, claude-shannon, aliasing]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
external:
  - { title: "Harry Nyquist (Wikipedia)", url: https://en.wikipedia.org/wiki/Harry_Nyquist }
---

**Harry Nyquist** (1889–1976) was a Swedish-American engineer at Bell Labs whose work on
the maximum signalling rate of a channel underlies the
[sampling theorem](/reference/nyquist-theorem/) at the heart of digital radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A sine wave sampled at just over two points per cycle, illustrating the Nyquist sampling rate." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 60 C 70 20, 130 20, 180 60 S 290 100, 340 60 S 440 30, 440 60" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="50" cy="40" r="3"/><circle cx="115" cy="32" r="3"/><circle cx="180" cy="60" r="3"/><circle cx="245" cy="88" r="3"/><circle cx="310" cy="80" r="3"/><circle cx="375" cy="48" r="3"/></g>
  <text x="230" y="108" text-anchor="middle" font-size="9" fill="currentColor">sample at ≥ 2× the highest frequency</text>
</svg>
<figcaption>Nyquist established the sampling limit at the heart of all digital radio; the Nyquist rate bears his name.</figcaption>
</figure>

## Life and work

Nyquist studied how fast pulses could be sent over a channel without interference,
establishing the relationship between [bandwidth](/reference/bandwidth/) and
[sample rate](/reference/sample-rate/) later formalised with
[Claude Shannon](/reference/claude-shannon/).

## Contribution

The Nyquist rate — sampling at twice the bandwidth — tells engineers how fast an
[ADC](/reference/analog-to-digital-converter/) must run.

## Legacy

His name marks the boundary every SDR respects to avoid [aliasing](/reference/aliasing/).
