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
  - { title: "How signals travel", url: /learn/propagation/ }
external:
  - { title: "Attenuation (Wikipedia)", url: https://en.wikipedia.org/wiki/Attenuation }
---

**Attenuation** is the reduction of signal strength as energy passes through a medium,
cable, connector, or obstacle. It is expressed in [decibels](/reference/decibel/) and
subtracts directly from a power budget.

## How it works

Every metre of coax, every connector, and every wall or tree adds loss — generally more
at higher [frequencies](/reference/frequency/). Free-space spreading is a specific kind
of attenuation called [path loss](/reference/path-loss/).

## Relevance to SDR

Keeping cable runs short and connectors clean minimises attenuation between
[antenna](/reference/antenna/) and receiver, preserving [SNR](/reference/signal-to-noise-ratio/).
