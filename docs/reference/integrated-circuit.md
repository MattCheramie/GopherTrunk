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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 232" role="img" aria-label="On the left, a chip package with rows of pins; a zoom expands its interior on the right to reveal a single silicon die holding labelled logic and memory blocks above a dense field of tiny squares standing in for millions of transistors, all etched together." xmlns="http://www.w3.org/2000/svg">
  <rect x="46" y="76" width="104" height="82" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="98" y="112" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">chip</text>
  <text x="98" y="126" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">package</text>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7">
    <line x1="40" y1="88" x2="46" y2="88"/><line x1="40" y1="104" x2="46" y2="104"/><line x1="40" y1="120" x2="46" y2="120"/><line x1="40" y1="136" x2="46" y2="136"/>
    <line x1="150" y1="88" x2="156" y2="88"/><line x1="150" y1="104" x2="156" y2="104"/><line x1="150" y1="120" x2="156" y2="120"/><line x1="150" y1="136" x2="156" y2="136"/>
  </g>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.45" stroke-dasharray="4 3" fill="none">
    <line x1="150" y1="76" x2="230" y2="34"/><line x1="150" y1="158" x2="230" y2="198"/>
  </g>
  <text x="190" y="26" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.7">zoom in</text>
  <rect x="230" y="34" width="190" height="164" rx="4" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.5"/>
  <text x="325" y="50" text-anchor="middle" font-size="8" fill="currentColor" font-weight="600">one silicon die</text>
  <rect x="242" y="58" width="80" height="40" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="282" y="82" text-anchor="middle" font-size="8" fill="currentColor">logic</text>
  <rect x="330" y="58" width="78" height="40" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="369" y="82" text-anchor="middle" font-size="8" fill="currentColor">SRAM</text>
  <g fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="0.5" stroke-opacity="0.5">
    <rect x="242" y="112" width="12" height="12"/><rect x="262" y="112" width="12" height="12"/><rect x="282" y="112" width="12" height="12"/><rect x="302" y="112" width="12" height="12"/><rect x="322" y="112" width="12" height="12"/><rect x="342" y="112" width="12" height="12"/><rect x="362" y="112" width="12" height="12"/><rect x="382" y="112" width="12" height="12"/>
    <rect x="242" y="132" width="12" height="12"/><rect x="262" y="132" width="12" height="12"/><rect x="282" y="132" width="12" height="12"/><rect x="302" y="132" width="12" height="12"/><rect x="322" y="132" width="12" height="12"/><rect x="342" y="132" width="12" height="12"/><rect x="362" y="132" width="12" height="12"/><rect x="382" y="132" width="12" height="12"/>
    <rect x="242" y="152" width="12" height="12"/><rect x="262" y="152" width="12" height="12"/><rect x="282" y="152" width="12" height="12"/><rect x="302" y="152" width="12" height="12"/><rect x="322" y="152" width="12" height="12"/><rect x="342" y="152" width="12" height="12"/><rect x="362" y="152" width="12" height="12"/><rect x="382" y="152" width="12" height="12"/>
  </g>
  <text x="325" y="186" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.8">millions of transistors, etched together</text>
</svg>
<figcaption>A chip is a package wrapped around a single sliver of silicon — the die. Zoom in and the whole circuit is there: logic and memory blocks plus millions or billions of transistors, all etched and wired together in one fabrication step. That integration is what made computers small, cheap, and reliable.</figcaption>
</figure>

## Overview

Instead of wiring individual [transistors](/reference/transistor/), resistors, and other parts by hand, an IC etches them and their connections onto one *die* in a single manufacturing process. Independently invented by Jack Kilby and Robert Noyce around 1958–1959, this packed whole circuits into a tiny, reliable, cheap chip. Successive process generations shrank the features and multiplied the transistor count — the trend captured by [Moore's law](/reference/moores-law/) — until a single chip could hold billions of switches.

## Where it fits

The integrated circuit is what made computers small and affordable: a [CPU](/reference/central-processing-unit/), memory, or an entire [system-on-a-chip](/reference/system-on-a-chip/) is one IC. The [logic gates](/reference/logic-gate/) of digital design are built from the transistors inside it. In SDR hardware, ICs do everything from the analog-to-digital conversion in a dongle to the dedicated chips that handle wideband sampling before GopherTrunk's software takes over.

## Sources

[^wiki]: [Integrated circuit](https://en.wikipedia.org/wiki/Integrated_circuit) — Wikipedia, on ICs, their invention, and fabrication.
