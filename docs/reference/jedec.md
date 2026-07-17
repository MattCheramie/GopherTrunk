---
slug: jedec
title: JEDEC
entry_type: organization
category: hw-organizations
description: JEDEC is the semiconductor industry's standards body, best known for defining the memory standards such as the DDR and LPDDR specifications used in computer RAM.
keywords: JEDEC, memory standards, DDR, LPDDR, DRAM, semiconductor standards, RAM, flash
aka: [JEDEC Solid State Technology Association]
autolink: true
infobox:
  - { label: Type, value: Industry standards organization }
  - { label: Founded, value: "1958" }
  - { label: HQ, value: Arlington, Virginia, USA }
  - { label: Makes, value: Semiconductor and memory standards }
see_also: [random-access-memory, semiconductor, ieee, flash-memory]
cite_urls:
  - https://www.jedec.org/
  - https://en.wikipedia.org/wiki/JEDEC
---

**JEDEC** (the JEDEC Solid State Technology Association) is the semiconductor industry's
standards body, best known for defining the memory standards used in computer
[RAM](/reference/random-access-memory/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A timeline of JEDEC's DDR memory generations. Starting from the first DDR standard around 2000, each successive generation — DDR2, DDR3, DDR4, and DDR5 around 2020 — roughly doubles the peak data rate while keeping a common, vendor-neutral interface so parts stay interchangeable." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="60" x2="440" y2="60" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.1">
    <circle cx="55" cy="60" r="4" fill-opacity="0.15"/>
    <circle cx="150" cy="60" r="4" fill-opacity="0.15"/>
    <circle cx="245" cy="60" r="4" fill-opacity="0.15"/>
    <circle cx="340" cy="60" r="5" fill-opacity="0.4"/>
    <circle cx="415" cy="60" r="6" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="55" y="45" font-size="9" font-weight="600">DDR</text>
    <text x="55" y="78" font-size="8">~2000</text>
    <text x="150" y="45" font-size="9" font-weight="600">DDR2</text>
    <text x="245" y="45" font-size="9" font-weight="600">DDR3</text>
    <text x="340" y="45" font-size="9" font-weight="600">DDR4</text>
    <text x="415" y="45" font-size="9" font-weight="600">DDR5</text>
    <text x="415" y="78" font-size="8">~2020</text>
    <text x="235" y="102" font-size="8" fill-opacity="0.9">each generation roughly doubles peak data rate — same open, vendor-neutral interface</text>
  </g>
</svg>
<figcaption>JEDEC's DDR standards advance in lockstep generations — DDR through DDR5 — each roughly doubling bandwidth while holding to one open interface, so memory from any member maker stays interchangeable.</figcaption>
</figure>

## Overview

JEDEC began in 1958 as a joint standards effort within the electronics industry and is now
an independent member association of [semiconductor](/reference/semiconductor/) companies. It
develops open standards through committees of competing manufacturers, ensuring that parts
from different vendors are compatible.[^home]

Its most visible work is the DDR (Double Data Rate) and LPDDR memory specifications that
define the DRAM modules in PCs, servers, and phones. JEDEC also standardizes
[flash memory](/reference/flash-memory/) interfaces, packaging, and reliability test methods
used across the industry.

## What JEDEC standardizes

The association's committees cover far more than desktop RAM:

| Standard family | Where it is used |
|-----------------|------------------|
| DDR | Main memory in PCs and servers |
| LPDDR | Low-power memory in phones and tablets |
| eMMC / UFS | Managed flash storage in mobile devices |
| Packaging / JESD tests | Physical formats and reliability methods |

## Where it fits

Because JEDEC sets the memory standards, a DDR module from one maker works in a board
designed for that standard, regardless of brand. Every computer that runs a GopherTrunk
decoder — from a server holding hours of IQ data in RAM to a single-board capture node —
depends on JEDEC-standardized memory to do it, and the sample buffers streaming through the
DSP live in exactly that DRAM.

## Sources

[^home]: [JEDEC](https://www.jedec.org/) — the association's official site, for its standards.
[^wiki]: [JEDEC](https://en.wikipedia.org/wiki/JEDEC) — Wikipedia, for the organization's history and memory standards.
