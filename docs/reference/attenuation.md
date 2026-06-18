---
slug: attenuation
title: Attenuation
entry_type: term
category: rf-fundamentals
description: Attenuation is the reduction in signal strength as it passes through a medium, cable, or obstacle, expressed in decibels.
keywords: attenuation, loss, dB, coax loss, signal weakening
aka: [attenuation]
autolink: true
infobox:
  - { label: Type, value: Signal loss }
  - { label: Unit, value: Decibels (dB) }
  - { label: Causes, value: Distance, cable, obstacles, filters }
see_also: [path-loss, decibel, radio-propagation, antenna]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
external:
  - { title: "Attenuation (Wikipedia)", url: https://en.wikipedia.org/wiki/Attenuation }
---

**Attenuation** is the reduction of signal strength as energy passes through a medium,
cable, connector, or obstacle. It is expressed in [decibels](/reference/decibel/) and
subtracts directly from a power budget.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A sine wave whose amplitude shrinks steadily from left to right as it is attenuated." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="65" x2="440" y2="65" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 65 q15 -40 30 0 t30 0 q15 -30 30 0 t30 0 q15 -20 30 0 t30 0 q15 -12 30 0 t30 0 q15 -7 30 0 t30 0 q15 -4 30 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="30" y="115" font-size="10" fill="currentColor">transmitter</text>
  <text x="430" y="115" font-size="10" fill="currentColor" text-anchor="end">weaker at receiver</text>
</svg>
<figcaption>Attenuation is the loss of signal strength as energy spreads out and is absorbed along the path.</figcaption>
</figure>

## How it works

Every metre of coax, every connector, and every wall or tree adds loss — generally more
at higher [frequencies](/reference/frequency/). Free-space spreading is a specific kind
of attenuation called [path loss](/reference/path-loss/).

## Relevance to SDR

Keeping cable runs short and connectors clean minimises attenuation between
[antenna](/reference/antenna/) and receiver, preserving [SNR](/reference/signal-to-noise-ratio/).
