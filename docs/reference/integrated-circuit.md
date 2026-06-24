---
slug: integrated-circuit
title: Integrated circuit
entry_type: hardware
category: hw-foundations
description: An integrated circuit is a set of electronic circuits fabricated together on a single small piece of semiconductor, packing millions or billions of transistors into one chip.
keywords: integrated circuit, IC, chip, microchip, silicon, transistor, fabrication, die
aka: [IC, Microchip, Chip]
infobox:
  - { label: Type, value: Semiconductor chip }
  - { label: Made of, value: Silicon (mostly) }
  - { label: Contains, value: Up to billions of transistors }
  - { label: Invented, value: "1958–1959" }
see_also: [transistor, semiconductor, logic-gate, moores-law, central-processing-unit, system-on-a-chip]
cite_urls:
  - https://en.wikipedia.org/wiki/Integrated_circuit
---

An **integrated circuit** (**IC**, or chip) is a set of electronic circuits fabricated together on a single small piece of [semiconductor](/reference/semiconductor/) material, usually silicon.[^wiki]

## Overview

Instead of wiring individual [transistors](/reference/transistor/), resistors, and other parts by hand, an IC etches them and their connections onto one *die* in a single manufacturing process. Independently invented by Jack Kilby and Robert Noyce around 1958–1959, this packed whole circuits into a tiny, reliable, cheap chip. Successive process generations shrank the features and multiplied the transistor count — the trend captured by [Moore's law](/reference/moores-law/) — until a single chip could hold billions of switches.

## Where it fits

The integrated circuit is what made computers small and affordable: a [CPU](/reference/central-processing-unit/), memory, or an entire [system-on-a-chip](/reference/system-on-a-chip/) is one IC. The [logic gates](/reference/logic-gate/) of digital design are built from the transistors inside it. In SDR hardware, ICs do everything from the analog-to-digital conversion in a dongle to the dedicated chips that handle wideband sampling before GopherTrunk's software takes over.

## Sources

[^wiki]: [Integrated circuit](https://en.wikipedia.org/wiki/Integrated_circuit) — Wikipedia, on ICs, their invention, and fabrication.
