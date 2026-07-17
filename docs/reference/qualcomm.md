---
slug: qualcomm
title: Qualcomm
entry_type: organization
category: hw-organizations
description: Qualcomm is an American company known for cellular modem technology and its Snapdragon system-on-chip processors that power a large share of smartphones.
keywords: Qualcomm, Snapdragon, modem, cellular, SoC, baseband, wireless, CDMA
aka: [Qualcomm Incorporated]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor company }
  - { label: Founded, value: "1985" }
  - { label: HQ, value: San Diego, California, USA }
  - { label: Makes, value: Mobile SoCs, cellular modems, wireless chips }
see_also: [system-on-a-chip, arm-holdings, arm-architecture, cellular-modem, tsmc]
cite_urls:
  - https://www.qualcomm.com/
  - https://en.wikipedia.org/wiki/Qualcomm
---

**Qualcomm** is an American company founded in 1985, best known for cellular modem
technology and its Snapdragon [systems-on-chip](/reference/system-on-a-chip/) that power a
large share of the world's smartphones.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A diagram of a Qualcomm Snapdragon system-on-chip. A single package outline contains four integrated blocks: Arm-based CPU cores, a GPU, an AI accelerator, and an integrated cellular modem, showing how a whole phone's processing and radio front end are packed onto one power-efficient chip." xmlns="http://www.w3.org/2000/svg">
  <rect x="70" y="20" width="320" height="72" rx="6" stroke="currentColor" fill="currentColor" fill-opacity="0.06" stroke-width="1.4"/>
  <text x="230" y="14" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">Snapdragon SoC</text>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.14" stroke-width="1.1">
    <rect x="82" y="34" width="70" height="44" rx="3"/>
    <rect x="158" y="34" width="70" height="44" rx="3"/>
    <rect x="234" y="34" width="70" height="44" rx="3"/>
    <rect x="310" y="34" width="70" height="44" rx="3"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8.5">
    <text x="117" y="52">Arm</text>
    <text x="117" y="64">CPU cores</text>
    <text x="193" y="58">GPU</text>
    <text x="269" y="52">AI</text>
    <text x="269" y="64">accel.</text>
    <text x="345" y="52">cellular</text>
    <text x="345" y="64">modem</text>
  </g>
  <text x="230" y="110" font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.9">compute and the radio front end on one power-efficient chip</text>
</svg>
<figcaption>A Snapdragon system-on-chip fuses Arm CPU cores, a GPU, an AI accelerator, and a cellular modem onto one die — the same radio-plus-processor integration that lets a whole phone fit in a pocket.</figcaption>
</figure>

## Overview

Qualcomm built its early business on CDMA cellular technology and the patents around it,
becoming a major licensor of mobile wireless intellectual property. Its Snapdragon
SoCs combine [Arm](/reference/arm-holdings/)-based CPU cores, a GPU, an AI accelerator, and
an integrated [cellular modem](/reference/cellular-modem/) on one chip, and are widely used
in Android phones, tablets, and increasingly in laptops.[^home]

The company is largely fabless, designing its chips and outsourcing manufacturing to
foundries such as [TSMC](/reference/tsmc/). It also supplies Wi-Fi, Bluetooth, and GPS
components used across the mobile and embedded industry.

## What Qualcomm sells

Qualcomm earns money from both silicon and the patent portfolio behind it:

| Business | What it is |
|----------|------------|
| Snapdragon SoCs | Integrated CPU + GPU + modem chips for phones |
| Standalone modems | Cellular baseband chips for other makers' devices |
| Connectivity | Wi-Fi, Bluetooth, and GPS components |
| Licensing | Patents on CDMA and later cellular standards |

## Where it fits

Qualcomm's modems and SoCs sit inside a huge fraction of the mobile devices in use, making
it central to how phones connect to networks. Its baseband and RF expertise illustrates how
a complete radio front end and digital processor can be packed into a single power-efficient
chip — the same integration trend that puts capable SDR-adjacent radios into small devices,
and that a GopherTrunk decoder ultimately mirrors in software.

## Sources

[^home]: [Qualcomm](https://www.qualcomm.com/) — the company's official site, for products and technology.
[^wiki]: [Qualcomm](https://en.wikipedia.org/wiki/Qualcomm) — Wikipedia, for the company's history and role.
