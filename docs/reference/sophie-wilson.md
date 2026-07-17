---
slug: sophie-wilson
title: Sophie Wilson
entry_type: person
category: hw-people
description: Sophie Wilson (born 1957) is a British computer scientist who designed the BBC Micro's BASIC and the original ARM instruction set, the architecture now found in most of the world's mobile and embedded processors.
keywords: Sophie Wilson, ARM, ARM architecture, BBC Micro, BBC BASIC, Acorn, instruction set, RISC
aka: [Sophie Wilson]
autolink: true
infobox:
  - { label: Born, value: "1957" }
  - { label: Field, value: Computer science / chip design }
  - { label: Known for, value: ARM instruction set }
see_also: [arm-architecture, arm-holdings, central-processing-unit, instruction-set-architecture]
cite_urls:
  - https://en.wikipedia.org/wiki/Sophie_Wilson
---

**Sophie Wilson** (born 1957) is a British computer scientist who designed the original
[ARM](/reference/arm-architecture/) instruction set, the processor architecture now at the
heart of most smartphones and a vast range of embedded devices.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A career timeline of Sophie Wilson: she wrote BBC BASIC for the BBC Micro at Acorn in 1981, designed the original ARM instruction set in 1983, saw the first ARM silicon run in 1985, and the architecture was spun out into Arm Holdings in 1990." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="66" x2="440" y2="66" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="60" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="185" cy="66" r="6" fill="currentColor"/>
    <circle cx="310" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="420" cy="66" r="5" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="60" y="50" font-size="9" font-weight="600">1981</text>
    <text x="60" y="86" font-size="8">BBC BASIC</text>
    <text x="185" y="50" font-size="9" font-weight="600">1983</text>
    <text x="185" y="86" font-size="8">ARM ISA</text>
    <text x="310" y="50" font-size="9" font-weight="600">1985</text>
    <text x="310" y="86" font-size="8">first ARM silicon</text>
    <text x="420" y="50" font-size="9" font-weight="600">1990</text>
    <text x="420" y="86" font-size="8">Arm spun out</text>
    <text x="235" y="112" font-size="8" fill-opacity="0.9">a simple, low-power RISC design &#8212; now tens of billions of cores a year</text>
  </g>
</svg>
<figcaption>Wilson's line runs from BBC BASIC to the compact ARM instruction set she designed at Acorn — a low-power RISC idea that grew into the most widely shipped processor architecture in the world.</figcaption>
</figure>

## Life and work

At Acorn Computers, Wilson wrote the BBC BASIC language for the BBC Micro, a hugely influential
British educational computer of the 1980s. When Acorn set out to build its own processor, she
designed the [instruction set](/reference/instruction-set-architecture/) for the Acorn RISC
Machine — ARM — while colleague Steve Furber led the chip implementation. Their goal was a
simple, efficient, low-power [CPU](/reference/central-processing-unit/), and the resulting RISC
design was remarkably compact.[^wiki] The architecture was later spun out into what became
[Arm Holdings](/reference/arm-holdings/).[^wiki]

| Year | Milestone |
|------|-----------|
| 1981 | Writes BBC BASIC for the BBC Micro |
| 1983 | Designs the original ARM instruction set |
| 1985 | First ARM silicon runs |
| 1990 | ARM spun out into Arm Holdings |

## Why they matter

The [ARM architecture](/reference/arm-architecture/) Wilson defined emphasised doing more with
less power, which made it ideal as battery-powered devices proliferated. ARM cores now ship in
the tens of billions per year, powering phones, single-board computers, and embedded
controllers — including many of the small boards used to run SDR capture and decode tasks.[^wiki]

## Legacy

The instruction set she created decades ago still underlies the dominant mobile and embedded
processor family, making it one of the most widely used designs in computing history — and the
same ISA that runs GopherTrunk when it is cross-compiled for a low-power ARM capture node.

## Sources

[^wiki]: [Sophie Wilson](https://en.wikipedia.org/wiki/Sophie_Wilson) — Wikipedia, for biography, the BBC Micro, and ARM.
