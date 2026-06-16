---
slug: frequency-modulation
title: Frequency modulation (FM)
entry_type: technology
category: modulation
description: Frequency modulation (FM) encodes information by varying a carrier's frequency; it resists amplitude noise and is used for broadcast and analog two-way voice.
keywords: frequency modulation, FM, deviation, capture effect, narrowband FM, broadcast
aka: [frequency modulation]
autolink: true
infobox:
  - { label: Type, value: Analog modulation }
  - { label: Varies, value: Carrier frequency (deviation) }
  - { label: Used for, value: FM broadcast, analog two-way voice }
see_also: [modulation, amplitude-modulation, single-sideband, frequency-shift-keying]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/analog-modulation/ }
external:
  - { title: "Frequency modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency_modulation }
---

**Frequency modulation** (**FM**) encodes information by varying a
[carrier](/reference/carrier-wave/)'s [frequency](/reference/frequency/) while its
[amplitude](/reference/amplitude/) stays constant. The amount of swing is the
*deviation*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A carrier whose cycle spacing tightens and loosens with the message, at constant amplitude." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q4 -28 8 0 q4 -28 8 0 q6 -28 12 0 q7 -28 14 0 q8 -28 16 0 q7 -28 14 0 q6 -28 12 0 q4 -28 8 0 q4 -28 8 0 q4 -28 8 0 q6 -28 12 0 q7 -28 14 0 q8 -28 16 0 q7 -28 14 0 q6 -28 12 0 q4 -28 8 0 q4 -28 8 0 q4 -28 8 0 q6 -28 12 0 q7 -28 14 0 q8 -28 16 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="20" y="105" font-size="10" fill="currentColor">constant amplitude — information is in the spacing (frequency)</text>
</svg>
<figcaption>FM varies the carrier's frequency while amplitude stays constant, which is why it shrugs off amplitude noise.</figcaption>
</figure>

## How it works

Because the information lives in frequency, not amplitude, an FM receiver can ignore
amplitude noise, giving clean audio. FM also shows a *capture effect* — the strongest
signal dominates a channel.

## Relevance to SDR

FM broadcast (wide deviation) and narrowband FM two-way voice are everywhere; the
latter is the analog cousin of the digital [FSK](/reference/frequency-shift-keying/)
voice modes GopherTrunk decodes.
