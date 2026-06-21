---
slug: amplitude
title: Amplitude
entry_type: term
category: rf-fundamentals
description: Amplitude is the magnitude or strength of a wave; in radio it corresponds to signal power and is the quantity varied by amplitude modulation.
keywords: amplitude, signal strength, magnitude, power, AM
infobox:
  - { label: Type, value: Wave property }
  - { label: Represents, value: Strength / power }
  - { label: Reported as, value: Power level (dBm / dBFS) }
see_also: [phase, decibel, dbm, amplitude-modulation, signal-to-noise-ratio]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Amplitude
---

**Amplitude** is the magnitude — the "height" — of a wave.[^wiki] For a
[radio wave](/reference/radio-wave/) it corresponds to signal strength, which a
receiver reports as a power level in [dBm](/reference/dbm/) or
[dBFS](/reference/dbfs/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A larger-amplitude sine wave and a smaller-amplitude sine wave sharing a centre line." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 70 q35 -48 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="2"/>
  <path d="M20 70 q35 -16 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="1.6" stroke-opacity="0.6"/>
  <line x1="90" y1="70" x2="90" y2="22" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="98" y="40" font-size="11" fill="currentColor">larger amplitude = stronger signal</text>
</svg>
<figcaption>Amplitude is the height of the wave; greater amplitude delivers more power to the receiver.</figcaption>
</figure>

## How it works

A larger amplitude carries more power. As a signal spreads from a transmitter and
passes obstacles, its amplitude falls ([path loss](/reference/path-loss/)), which is
why distant signals are weaker.

## Relevance to SDR

Varying amplitude is the basis of [amplitude modulation](/reference/amplitude-modulation/)
and part of what an [IQ](/reference/iq-data/) sample encodes (its distance from the
origin). A signal's amplitude relative to the [noise floor](/reference/noise-floor/)
sets its [SNR](/reference/signal-to-noise-ratio/).

## Sources

[^wiki]: [Amplitude](https://en.wikipedia.org/wiki/Amplitude) — Wikipedia, on the magnitude of a wave's oscillation.
