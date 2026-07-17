---
slug: intel
title: Intel
entry_type: organization
category: hw-organizations
description: Intel is an American semiconductor company, the inventor of the commercial microprocessor and long the dominant maker of x86 CPUs for PCs, servers, and data centers.
keywords: Intel, x86, microprocessor, CPU, semiconductor, Core, Xeon, fab, 4004, 8086
aka: [Intel Corporation]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor company }
  - { label: Founded, value: "1968" }
  - { label: HQ, value: Santa Clara, California, USA }
  - { label: Makes, value: x86 CPUs, chipsets, FPGAs }
see_also: [central-processing-unit, x86, gordon-moore, robert-noyce, amd, moores-law, semiconductor]
cite_urls:
  - https://www.intel.com/
  - https://en.wikipedia.org/wiki/Intel
---

**Intel** is an American semiconductor company founded in 1968 that produced the first
commercial microprocessor and became the dominant supplier of
[x86](/reference/x86/) [CPUs](/reference/central-processing-unit/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A timeline of Intel milestones. It is founded in 1968 by Noyce and Moore, ships the 4004, the first commercial single-chip microprocessor, in 1971, launches the 8086 that starts the x86 line in 1978, and reaches the modern era of Core processors for PCs and Xeon processors for servers." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="66" x2="440" y2="66" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="60" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="180" cy="66" r="5" fill-opacity="0.15"/>
    <circle cx="300" cy="66" r="6" fill="currentColor"/>
    <circle cx="410" cy="66" r="5" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="60" y="50" font-size="9" font-weight="600">1968</text>
    <text x="60" y="86" font-size="8">founded</text>
    <text x="60" y="97" font-size="8">Noyce &amp; Moore</text>
    <text x="180" y="50" font-size="9" font-weight="600">1971</text>
    <text x="180" y="86" font-size="8">4004</text>
    <text x="180" y="97" font-size="8">1st microprocessor</text>
    <text x="300" y="50" font-size="9" font-weight="600">1978</text>
    <text x="300" y="86" font-size="8">8086</text>
    <text x="300" y="97" font-size="8">starts x86</text>
    <text x="410" y="50" font-size="9" font-weight="600">now</text>
    <text x="410" y="86" font-size="8">Core / Xeon</text>
  </g>
</svg>
<figcaption>Intel's arc runs from its 1968 founding by Noyce and Moore, through the 4004 — the first single-chip microprocessor — and the 8086 that launched x86, to today's Core and Xeon lines.</figcaption>
</figure>

## Overview

Intel was founded by [Robert Noyce](/reference/robert-noyce/) and
[Gordon Moore](/reference/gordon-moore/), two veterans of the early silicon industry. Its
1971 Intel 4004 was the first commercially available microprocessor on a single chip, and
the 1978 8086 launched the x86 instruction set that still underpins most desktop, laptop,
and server processors today.[^wiki]

For decades Intel built both the chip designs and the fabrication plants ("fabs") that
made them, a vertically integrated model summarised by
[Moore's law](/reference/moores-law/) — his prediction that transistor counts would roughly
double every two years. Its modern lines include Core processors for PCs and Xeon
processors for servers and data centers; it has also produced chipsets, network controllers,
and FPGAs.[^home]

## Landmark chips

A few products mark the turns in Intel's history:

| Year | Chip | Why it mattered |
|------|------|-----------------|
| 1971 | 4004 | First commercial single-chip microprocessor |
| 1978 | 8086 | Started the x86 architecture |
| 1993 | Pentium | Brought x86 into the mainstream PC boom |
| 2006+ | Core / Xeon | Modern PC and server processor families |

## Where it fits

The PC era was largely built on Intel x86 CPUs running Windows and Linux, and most
general-purpose servers — including the machines that host an SDR decoder pipeline or a
fleet of GopherTrunk capture nodes — still use x86 processors from Intel or its rival
[AMD](/reference/amd/). The architecture's longevity means software compiled for x86
decades ago often still runs on current hardware.

## Sources

[^home]: [Intel](https://www.intel.com/) — the company's official site, for current product lines.
[^wiki]: [Intel](https://en.wikipedia.org/wiki/Intel) — Wikipedia, for the company's history and significance.
