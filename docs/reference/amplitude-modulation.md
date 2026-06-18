---
slug: amplitude-modulation
title: Amplitude modulation (AM)
entry_type: technology
category: modulation
description: Amplitude modulation (AM) encodes information by varying a carrier's amplitude; it is simple, prone to noise, and still used for shortwave broadcast and aviation voice.
keywords: amplitude modulation, AM, carrier, sidebands, aviation airband, shortwave
aka: [amplitude modulation]
autolink: true
infobox:
  - { label: Type, value: Analog modulation }
  - { label: Varies, value: Carrier amplitude }
  - { label: Used for, value: Shortwave broadcast, aviation airband }
see_also: [modulation, frequency-modulation, single-sideband, carrier-wave]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
external:
  - { title: "Amplitude modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Amplitude_modulation }
---

**Amplitude modulation** (**AM**) encodes information by varying the
[amplitude](/reference/amplitude/) of a [carrier wave](/reference/carrier-wave/) while
its [frequency](/reference/frequency/) stays fixed. It is the oldest and simplest
[modulation](/reference/modulation/) scheme.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A carrier whose amplitude envelope follows a slower message waveform." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 65 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M20 65 C 80 18, 140 18, 200 65 S 320 112, 380 65 S 440 30, 440 65" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="20" y="118" font-size="10" fill="currentColor">the envelope carries the message</text>
</svg>
<figcaption>AM varies the carrier's amplitude in step with the message; the dashed envelope is the audio.</figcaption>
</figure>

## How it works

Louder audio produces larger swings in carrier height, creating two mirror-image
sidebands around the carrier. Because noise is also an amplitude variation, AM is
relatively noise-prone.

## Relevance to SDR

AM survives where simplicity or a useful property matters — shortwave broadcast, and
aviation VHF airband (where overlapping transmissions beat together so a controller
notices). An SDR demodulates AM by tracking the envelope.
