---
slug: modulation
title: Modulation
entry_type: term
category: rf-fundamentals
description: Modulation is the process of varying a carrier wave's amplitude, frequency, or phase to encode information for transmission over radio.
keywords: modulation, AM FM PSK FSK, carrier, encoding information
infobox:
  - { label: Type, value: Signal-processing concept }
  - { label: Varies, value: Amplitude, frequency, or phase }
  - { label: Families, value: Analog (AM/FM/SSB), digital (FSK/PSK/QAM) }
see_also: [carrier-wave, amplitude-modulation, frequency-modulation, phase-shift-keying, frequency-shift-keying, quadrature-amplitude-modulation]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/analog-modulation/ }
external:
  - { title: "Modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Modulation }
---

**Modulation** is the process of varying a property of a
[carrier wave](/reference/carrier-wave/) — its [amplitude](/reference/amplitude/),
[frequency](/reference/frequency/), or [phase](/reference/phase/) — in step with a
message so that information can travel over radio.

## How it works

Analog modulation varies a property continuously: [AM](/reference/amplitude-modulation/),
[FM](/reference/frequency-modulation/), and [SSB](/reference/single-sideband/). Digital
modulation switches the carrier between discrete states (symbols):
[FSK](/reference/frequency-shift-keying/), [PSK](/reference/phase-shift-keying/), and
[QAM](/reference/quadrature-amplitude-modulation/).

## Relevance to SDR

Choosing the matching demodulator for a signal's modulation is the heart of decoding.
The same three carrier properties reappear as the axes of the
[IQ](/reference/iq-data/) plane.
