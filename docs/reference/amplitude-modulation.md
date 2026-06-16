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
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/analog-modulation/ }
external:
  - { title: "Amplitude modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Amplitude_modulation }
---

**Amplitude modulation** (**AM**) encodes information by varying the
[amplitude](/reference/amplitude/) of a [carrier wave](/reference/carrier-wave/) while
its [frequency](/reference/frequency/) stays fixed. It is the oldest and simplest
[modulation](/reference/modulation/) scheme.

## How it works

Louder audio produces larger swings in carrier height, creating two mirror-image
sidebands around the carrier. Because noise is also an amplitude variation, AM is
relatively noise-prone.

## Relevance to SDR

AM survives where simplicity or a useful property matters — shortwave broadcast, and
aviation VHF airband (where overlapping transmissions beat together so a controller
notices). An SDR demodulates AM by tracking the envelope.
