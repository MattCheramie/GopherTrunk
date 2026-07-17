---
slug: broadcom
title: Broadcom
entry_type: organization
category: hw-organizations
description: Broadcom is an American semiconductor and software company that makes networking, broadband, and wireless chips, including the SoCs at the heart of the Raspberry Pi.
keywords: Broadcom, SoC, networking chips, wireless, broadband, semiconductor, Avago, BCM, Wi-Fi
aka: [Broadcom Inc, Broadcom Corporation]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor and software company }
  - { label: Founded, value: "1991" }
  - { label: HQ, value: Palo Alto, California, USA }
  - { label: Makes, value: Networking, wireless, and broadband chips; SoCs }
see_also: [raspberry-pi, system-on-a-chip, semiconductor, wi-fi, tsmc]
cite_urls:
  - https://www.broadcom.com/
  - https://en.wikipedia.org/wiki/Broadcom
---

**Broadcom** is an American semiconductor and infrastructure-software company that makes
networking, broadband, and wireless chips, including the systems-on-chip used in the
[Raspberry Pi](/reference/raspberry-pi/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A map of Broadcom's product areas. A central Broadcom node branches to Ethernet switch silicon, Wi-Fi and Bluetooth radios, broadband and set-top-box processors, storage controllers, and the BCM systems-on-chip that power the Raspberry Pi. A separate branch shows its expansion into enterprise software." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <line x1="82" y1="75" x2="200" y2="30"/>
    <line x1="82" y1="75" x2="200" y2="58"/>
    <line x1="82" y1="75" x2="200" y2="86"/>
    <line x1="82" y1="75" x2="200" y2="114"/>
    <line x1="82" y1="75" x2="200" y2="142"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.12" stroke-width="1.2">
    <rect x="16" y="60" width="66" height="30" rx="4"/>
    <rect x="200" y="18" width="150" height="24" rx="4"/>
    <rect x="200" y="46" width="150" height="24" rx="4"/>
    <rect x="200" y="74" width="150" height="24" rx="4"/>
    <rect x="200" y="102" width="150" height="24" rx="4"/>
    <rect x="200" y="130" width="150" height="24" rx="4"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="9">
    <text x="49" y="79" text-anchor="middle" font-weight="600">Broadcom</text>
    <text x="210" y="34">Ethernet switch silicon</text>
    <text x="210" y="62">Wi-Fi &amp; Bluetooth radios</text>
    <text x="210" y="90">BCM SoC &#8594; Raspberry Pi</text>
    <text x="210" y="118">broadband / set-top boxes</text>
    <text x="210" y="146">enterprise software</text>
  </g>
</svg>
<figcaption>Broadcom spans networking, wireless, and broadband silicon plus enterprise software; its most visible chip to makers is the BCM system-on-chip that powers Raspberry Pi boards.</figcaption>
</figure>

## Overview

The modern Broadcom traces back to a 1991 chip company and grew through a long series of
mergers — most notably with Avago Technologies in 2016, after which the combined firm took
the Broadcom name. Its products span Ethernet switch silicon, [Wi-Fi](/reference/wi-fi/) and
Bluetooth chips, set-top-box and broadband processors, and storage controllers.[^home]

Broadcom designs the BCM-series [SoCs](/reference/system-on-a-chip/) that power Raspberry Pi
boards, pairing an Arm CPU with a graphics core on one chip. In recent years the company has
also expanded heavily into enterprise software through large acquisitions.

## Growth by acquisition

Broadcom's scale is largely the product of serial mega-mergers, blending chips and software:

| Year | Deal | What it added |
|------|------|---------------|
| 2016 | Avago + Broadcom | The combined chip firm and the Broadcom name |
| 2018 | CA Technologies | Enterprise/mainframe software |
| 2019 | Symantec (enterprise) | Cybersecurity software |
| 2023 | VMware | Data-center virtualization software |

## Where it fits

Broadcom's networking and wireless chips are inside a vast amount of consumer and data-center
equipment, often unseen. For the maker world its most visible role is the Raspberry Pi SoC —
which means a GopherTrunk capture node running on a Pi is, at its core, running on Broadcom
silicon, fabricated (like most advanced chips) by a foundry such as [TSMC](/reference/tsmc/).

## Sources

[^home]: [Broadcom](https://www.broadcom.com/) — the company's official site, for its product portfolio.
[^wiki]: [Broadcom](https://en.wikipedia.org/wiki/Broadcom) — Wikipedia, for the company's history and acquisitions.
