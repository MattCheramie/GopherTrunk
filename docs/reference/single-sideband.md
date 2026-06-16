---
slug: single-sideband
title: Single sideband (SSB)
entry_type: technology
category: modulation
description: Single sideband (SSB) is an efficient form of amplitude modulation that suppresses the carrier and one sideband, using half the bandwidth and concentrating power for long-distance HF voice.
keywords: single sideband, SSB, USB, LSB, suppressed carrier, HF voice, amateur
aka: [single sideband, SSB]
autolink: true
infobox:
  - { label: Type, value: Analog modulation (AM variant) }
  - { label: Sends, value: One sideband, suppressed carrier }
  - { label: Used for, value: Long-distance HF voice, amateur }
see_also: [amplitude-modulation, modulation, frequency-bands, ionospheric-propagation]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/analog-modulation/ }
external:
  - { title: "Single-sideband modulation (Wikipedia)", url: https://en.wikipedia.org/wiki/Single-sideband_modulation }
---

**Single sideband** (**SSB**) is a refined form of
[amplitude modulation](/reference/amplitude-modulation/) that removes the
[carrier](/reference/carrier-wave/) and one of the two redundant sidebands,
transmitting only **one sideband** — upper (USB) or lower (LSB).

## How it works

With the carrier and one sideband gone, SSB uses about **half the bandwidth** and puts
all power into the information, so modest transmitters reach across continents on
[HF](/reference/frequency-bands/). The cost is that the receiver must tune precisely or
voices sound distorted.

## Relevance to SDR

SSB is the backbone of long-distance HF voice; receiving it needs an HF-capable SDR and
accurate tuning to reinsert the missing carrier.
