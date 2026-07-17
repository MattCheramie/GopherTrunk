---
slug: arm-holdings
title: Arm Holdings
entry_type: organization
category: hw-organizations
description: Arm Holdings is a British company that designs the Arm processor architecture and licenses it to chipmakers, making it the dominant CPU design in phones and embedded devices.
keywords: Arm, Arm Holdings, ARM architecture, processor IP, licensing, Cortex, RISC, fabless
aka: [Arm, ARM Ltd, Advanced RISC Machines]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor IP company }
  - { label: Founded, value: "1990" }
  - { label: HQ, value: Cambridge, England, UK }
  - { label: Makes, value: Processor architecture and core designs (IP) }
see_also: [arm-architecture, sophie-wilson, central-processing-unit, qualcomm, tsmc, risc-v]
cite_urls:
  - https://www.arm.com/
  - https://en.wikipedia.org/wiki/Arm_Holdings
---

**Arm Holdings** is a British company that designs the
[Arm processor architecture](/reference/arm-architecture/) and licenses it to other
chipmakers rather than manufacturing chips itself.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A diagram of Arm's licensing business model. Arm designs the architecture and Cortex cores as intellectual property, licenses that IP to chip vendors such as Apple, Qualcomm, and Samsung, who integrate it into their own systems-on-chip, which are then physically manufactured by foundries like TSMC. Arm itself builds no chips." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.12" stroke-width="1.2">
    <rect x="18" y="42" width="96" height="44" rx="4"/>
    <rect x="182" y="42" width="96" height="44" rx="4"/>
    <rect x="346" y="42" width="96" height="44" rx="4"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <line x1="114" y1="64" x2="176" y2="64"/>
    <line x1="278" y1="64" x2="340" y2="64"/>
    <polygon points="176,64 168,60 168,68" fill="currentColor"/>
    <polygon points="340,64 332,60 332,68" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="9">
    <text x="66" y="60" font-weight="600">Arm</text>
    <text x="66" y="74" font-size="8">designs IP / Cortex cores</text>
    <text x="230" y="60" font-weight="600">SoC vendors</text>
    <text x="230" y="74" font-size="8">Apple, Qualcomm, Samsung</text>
    <text x="394" y="60" font-weight="600">foundry</text>
    <text x="394" y="74" font-size="8">TSMC fabricates</text>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8" fill-opacity="0.9">
    <text x="145" y="36">licenses</text>
    <text x="309" y="36">makes silicon</text>
  </g>
  <text x="230" y="112" font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.9">Arm never builds a chip — it sells the design that everyone else builds around</text>
</svg>
<figcaption>Arm's pure-IP model: it designs the architecture and Cortex cores, licenses them to SoC vendors like Apple, Qualcomm, and Samsung, who in turn have foundries such as TSMC fabricate the silicon.</figcaption>
</figure>

## Overview

Arm grew out of Acorn Computers, where the original ARM (Acorn RISC Machine) design was
created in the 1980s by a team including [Sophie Wilson](/reference/sophie-wilson/). The
company spun out in 1990 and adopted a pure licensing model: it sells the right to use its
architecture and its ready-made Cortex core designs, and partners such as Apple,
[Qualcomm](/reference/qualcomm/), and Samsung build the actual silicon.[^home]

Because Arm cores are energy-efficient, they dominate battery-powered devices. The vast
majority of smartphones, tablets, and microcontrollers — and a growing share of laptops and
servers — run Arm-based processors.

## Two ways to license

Arm's business rests on two kinds of licence, which suit very different customers:

| Licence type | What the customer gets |
|--------------|------------------------|
| Core licence | A ready-made Cortex design to drop into a chip |
| Architecture licence | The right to design a custom core (e.g. Apple's) that runs Arm code |

## Where it fits

Arm's licensing model spread one CPU architecture across nearly the entire mobile and
embedded world, and its main open-standard rival is now the royalty-free
[RISC-V](/reference/risc-v/). The single-board computers and microcontrollers used as
GopherTrunk capture nodes — Raspberry Pis and many MCU boards — are built around Arm cores,
so the same instruction set runs from the antenna node up to the phones in your pocket.

## Sources

[^home]: [Arm](https://www.arm.com/) — the company's official site, for its architecture and licensing.
[^wiki]: [Arm Holdings](https://en.wikipedia.org/wiki/Arm_Holdings) — Wikipedia, for the company's history and business model.
