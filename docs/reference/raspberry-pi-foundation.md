---
slug: raspberry-pi-foundation
title: Raspberry Pi Foundation
entry_type: organization
category: hw-organizations
description: The Raspberry Pi Foundation is a UK charity that created the Raspberry Pi single-board computer to make computing education affordable and accessible.
keywords: Raspberry Pi Foundation, Raspberry Pi, single-board computer, education, charity, Eben Upton, Broadcom
aka: [Raspberry Pi]
autolink: false
infobox:
  - { label: Type, value: Educational charity (and trading company) }
  - { label: Founded, value: "2009" }
  - { label: HQ, value: Cambridge, England, UK }
  - { label: Makes, value: Raspberry Pi single-board computers }
see_also: [raspberry-pi, eben-upton, single-board-computer, broadcom, compute-module]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://www.raspberrypi.org/
  - https://en.wikipedia.org/wiki/Raspberry_Pi_Foundation
---

**The Raspberry Pi Foundation** is a UK-based charity, founded in 2009, that created the
[Raspberry Pi](/reference/raspberry-pi/) [single-board computer](/reference/single-board-computer/)
to make computing affordable and accessible for education.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 125" role="img" aria-label="A timeline of the Raspberry Pi Foundation. The charity is founded in 2009 to improve computing education, ships the first low-cost Raspberry Pi board in 2012 which sells far beyond the classroom, and later separates engineering and sales into a trading subsidiary, Raspberry Pi Ltd, while the charity continues educational work." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="62" x2="440" y2="62" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="70" cy="62" r="5" fill-opacity="0.15"/>
    <circle cx="220" cy="62" r="6" fill="currentColor"/>
    <circle cx="380" cy="62" r="5" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="70" y="46" font-size="9" font-weight="600">2009</text>
    <text x="70" y="82" font-size="8">charity founded</text>
    <text x="70" y="93" font-size="8">for CS education</text>
    <text x="220" y="46" font-size="9" font-weight="600">2012</text>
    <text x="220" y="82" font-size="8">first Pi ships</text>
    <text x="220" y="93" font-size="8">sells beyond class</text>
    <text x="380" y="46" font-size="9" font-weight="600">later</text>
    <text x="380" y="82" font-size="8">Raspberry Pi Ltd</text>
    <text x="380" y="93" font-size="8">trades; charity teaches</text>
  </g>
</svg>
<figcaption>The Foundation set out in 2009 to fix a decline in hands-on computing, shipped its first cheap board in 2012, and later split off Raspberry Pi Ltd for engineering and sales while the charity kept its educational mission.</figcaption>
</figure>

## Overview

The Foundation was started by a group including [Eben Upton](/reference/eben-upton/), who
were concerned that young people were arriving at university with less hands-on computing
experience than earlier generations. Their answer was a low-cost, credit-card-sized
computer that could be bought for tens of dollars and freely tinkered with.[^home]

The first Raspberry Pi shipped in 2012 and sold far beyond the classroom, becoming a staple
for hobbyists, makers, and industry. Engineering and sales now run through a trading
subsidiary (Raspberry Pi Ltd), while the charity continues educational programs and
curriculum work. The boards are built around [Broadcom](/reference/broadcom/) systems-on-chip.

## The Raspberry Pi range

From one educational board, the line has grown to cover several form factors:

| Model | What it is |
|-------|------------|
| Model B | The flagship full-size board with ports |
| Zero | A tiny, cheaper cut-down board |
| Compute Module | A board-to-embed version for products |
| Pico | An RP2040 microcontroller board (not a full computer) |

## Where it fits

The Raspberry Pi turned the cheap, capable single-board computer into a mainstream tool.
For a project like GopherTrunk, a Pi by the antenna makes a practical capture node — small,
low-power, and inexpensive enough to deploy several — and the Foundation's mission is why
that hardware exists at the price it does. A [Compute Module](/reference/compute-module/) can
carry the same idea into a purpose-built receiver enclosure.

## Sources

[^home]: [Raspberry Pi](https://www.raspberrypi.org/) — the Foundation's official site.
[^wiki]: [Raspberry Pi Foundation](https://en.wikipedia.org/wiki/Raspberry_Pi_Foundation) — Wikipedia, for the charity's history and mission.
