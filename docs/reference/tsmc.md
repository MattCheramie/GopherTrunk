---
slug: tsmc
title: TSMC
entry_type: organization
category: hw-organizations
description: TSMC (Taiwan Semiconductor Manufacturing Company) is the world's largest dedicated chip foundry, manufacturing integrated circuits designed by other companies on contract.
keywords: TSMC, foundry, semiconductor manufacturing, fab, integrated circuit, fabless, process node
aka: [Taiwan Semiconductor Manufacturing Company]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor foundry }
  - { label: Founded, value: "1987" }
  - { label: HQ, value: Hsinchu, Taiwan }
  - { label: Makes, value: Integrated circuits (contract manufacturing) }
see_also: [semiconductor, integrated-circuit, amd, nvidia, qualcomm, moores-law]
cite_urls:
  - https://www.tsmc.com/
  - https://en.wikipedia.org/wiki/TSMC
---

**TSMC** (Taiwan Semiconductor Manufacturing Company) is the world's largest dedicated chip
foundry, manufacturing [integrated circuits](/reference/integrated-circuit/) designed by
other companies on contract.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A diagram of the pure-play foundry model. Several fabless design houses — AMD, NVIDIA, Apple, and Qualcomm — send their chip designs to TSMC, which owns the fabrication plants and manufactures the physical silicon that is returned to each customer as finished chips. TSMC designs no branded chips of its own." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.12" stroke-width="1.1">
    <rect x="16" y="14" width="96" height="22" rx="3"/>
    <rect x="16" y="44" width="96" height="22" rx="3"/>
    <rect x="16" y="74" width="96" height="22" rx="3"/>
    <rect x="16" y="104" width="96" height="22" rx="3"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5" text-anchor="middle">
    <text x="64" y="29">AMD</text>
    <text x="64" y="59">NVIDIA</text>
    <text x="64" y="89">Apple</text>
    <text x="64" y="119">Qualcomm</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <line x1="112" y1="25" x2="196" y2="60"/>
    <line x1="112" y1="55" x2="196" y2="66"/>
    <line x1="112" y1="85" x2="196" y2="74"/>
    <line x1="112" y1="115" x2="196" y2="80"/>
    <line x1="304" y1="70" x2="356" y2="70"/>
    <polygon points="356,70 348,66 348,74" fill="currentColor"/>
  </g>
  <rect x="196" y="46" width="108" height="48" rx="5" stroke="currentColor" fill="currentColor" fill-opacity="0.12" stroke-width="1.4"/>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="250" y="66" font-size="9" font-weight="600">TSMC</text>
    <text x="250" y="80" font-size="8">owns the fabs</text>
    <rect x="356" y="56" width="88" height="28" rx="4" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
    <text x="400" y="74" font-size="8.5">finished chips</text>
  </g>
  <text x="240" y="112" font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.9">fabless designers bring the design; TSMC makes the silicon</text>
</svg>
<figcaption>The pure-play foundry model in one picture: fabless firms like AMD, NVIDIA, Apple, and Qualcomm send designs to TSMC, which owns the fabrication plants and returns finished silicon — TSMC sells no branded chips of its own.</figcaption>
</figure>

## Overview

Founded in 1987, TSMC pioneered the "pure-play foundry" model: it does not design or sell
its own branded chips, but instead fabricates designs supplied by customers. That model
enabled the rise of "fabless" companies — firms like [AMD](/reference/amd/),
[NVIDIA](/reference/nvidia/), Apple, and [Qualcomm](/reference/qualcomm/) that design chips
but own no factories.[^home]

TSMC operates some of the most advanced [semiconductor](/reference/semiconductor/)
fabrication plants in the world, repeatedly leading the industry to smaller process nodes —
the steady shrink that keeps [Moore's law](/reference/moores-law/) alive. Because so many top
chip designs are made there, its plants are a critical link in the global electronics supply
chain.

## Why the model won

Splitting design from manufacturing gave each side an advantage that reshaped the industry:

| Party | What they gain |
|-------|----------------|
| Fabless designer | No multi-billion-dollar fab to build or fill |
| TSMC (foundry) | Volume from many customers funds leading-edge fabs |
| The industry | Faster process advances, more chip-design startups |

## Where it fits

Almost any modern device — a phone, a GPU, a server CPU, the SoC in a single-board computer
running a GopherTrunk node — is likely built on silicon fabricated by TSMC. By separating
chip design from manufacturing, TSMC reshaped the industry and concentrated a large share of
advanced production capacity in one company, so the RTL-SDR dongle, the Pi, and the decode
server on a GopherTrunk bench very probably all contain TSMC-made chips.

## Sources

[^home]: [TSMC](https://www.tsmc.com/) — the company's official site, for its foundry services.
[^wiki]: [TSMC](https://en.wikipedia.org/wiki/TSMC) — Wikipedia, for the company's history and the foundry model.
