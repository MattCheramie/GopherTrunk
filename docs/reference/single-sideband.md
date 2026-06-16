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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A double-sideband AM spectrum with carrier and two sidebands on the left, an arrow, and an SSB spectrum with only one sideband on the right." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.4">
    <line x1="40" y1="100" x2="170" y2="100" stroke-opacity="0.4"/>
    <line x1="105" y1="100" x2="105" y2="40"/>
    <rect x="70" y="70" width="20" height="30" fill="currentColor" fill-opacity="0.2"/>
    <rect x="120" y="70" width="20" height="30" fill="currentColor" fill-opacity="0.2"/>
  </g>
  <text x="105" y="118" text-anchor="middle" font-size="9" fill="currentColor">AM: carrier + 2 sidebands</text>
  <line x1="195" y1="70" x2="235" y2="70" stroke="currentColor" marker-end="url(#ssbar)"/>
  <g stroke="currentColor" stroke-width="1.4">
    <line x1="280" y1="100" x2="430" y2="100" stroke-opacity="0.4"/>
    <rect x="360" y="70" width="20" height="30" fill="currentColor" fill-opacity="0.3"/>
  </g>
  <text x="355" y="118" text-anchor="middle" font-size="9" fill="currentColor">SSB: one sideband only</text>
  <defs><marker id="ssbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SSB removes the carrier and one redundant sideband — half the bandwidth, all the power on the information.</figcaption>
</figure>

## How it works

With the carrier and one sideband gone, SSB uses about **half the bandwidth** and puts
all power into the information, so modest transmitters reach across continents on
[HF](/reference/frequency-bands/). The cost is that the receiver must tune precisely or
voices sound distorted.

## Relevance to SDR

SSB is the backbone of long-distance HF voice; receiving it needs an HF-capable SDR and
accurate tuning to reinsert the missing carrier.
