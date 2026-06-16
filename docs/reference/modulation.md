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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A message wave, then an AM version whose height follows the message, then an FM version whose spacing follows the message." xmlns="http://www.w3.org/2000/svg">
  <text x="6" y="22" font-size="10" fill="currentColor">message</text>
  <path d="M70 20 Q120 0 170 20 T270 20 T370 20 T440 20" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="6" y="78" font-size="10" fill="currentColor">AM</text>
  <path d="M70 75 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="6" y="140" font-size="10" fill="currentColor">FM</text>
  <path d="M70 137 q4 -16 8 0 q4 -16 8 0 q6 -16 12 0 q7 -16 14 0 q8 -16 16 0 q7 -16 14 0 q6 -16 12 0 q4 -16 8 0 q4 -16 8 0 q4 -16 8 0 q6 -16 12 0 q7 -16 14 0 q8 -16 16 0 q7 -16 14 0 q6 -16 12 0 q4 -16 8 0 q4 -16 8 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
</svg>
<figcaption>Modulation encodes a message by varying the carrier — its amplitude (AM), frequency (FM), or phase.</figcaption>
</figure>

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
