---
slug: odroid
title: ODROID
entry_type: hardware
category: hw-sbc
description: ODROID is a line of single-board computers from Hardkernel of South Korea, known for higher performance than the Raspberry Pi and unusual models such as big.LITTLE and x86 boards.
keywords: ODROID, Hardkernel, ODROID-N2, ODROID-XU4, Amlogic, big.LITTLE, high-performance SBC, Raspberry Pi alternative, heterogeneous cores
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Hardkernel (South Korea) }
  - { label: CPU, value: ARM (Amlogic) or x86 }
  - { label: Runs, value: Linux / Android }
  - { label: Known for, value: Performance per board }
see_also: [raspberry-pi, single-board-computer, rock-pi, orange-pi, banana-pi, gpio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/ODROID
---

**ODROID** is a line of [single-board computers](/reference/single-board-computer/) from Hardkernel of South Korea, known for offering more performance than the [Raspberry Pi](/reference/raspberry-pi/) and for some unusual designs.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 158" role="img" aria-label="A big.LITTLE core layout as used on boards like the ODROID-XU4. Four large high-performance cores sit in one cluster and four small energy-efficient cores in another, sharing memory. Light background tasks run on the little cores to save power, and heavy work spills onto the big cores for speed." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="34" width="180" height="70" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="44" y="50" width="34" height="34" rx="3" fill-opacity="0.22" fill="currentColor"/>
    <rect x="84" y="50" width="34" height="34" rx="3" fill-opacity="0.22" fill="currentColor"/>
    <rect x="124" y="50" width="34" height="34" rx="3" fill-opacity="0.22" fill="currentColor"/>
    <rect x="164" y="50" width="34" height="34" rx="3" fill-opacity="0.22" fill="currentColor"/>
    <rect x="250" y="34" width="180" height="70" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="264" y="58" width="20" height="20" rx="2" fill-opacity="0.1" fill="currentColor"/>
    <rect x="292" y="58" width="20" height="20" rx="2" fill-opacity="0.1" fill="currentColor"/>
    <rect x="320" y="58" width="20" height="20" rx="2" fill-opacity="0.1" fill="currentColor"/>
    <rect x="348" y="58" width="20" height="20" rx="2" fill-opacity="0.1" fill="currentColor"/>
    <path d="M210 69 H250" stroke-dasharray="4 3"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="120" y="26" font-size="8.5" font-weight="600">big cluster</text>
    <text x="61" y="71" font-size="7.5">A15</text>
    <text x="101" y="71" font-size="7.5">A15</text>
    <text x="141" y="71" font-size="7.5">A15</text>
    <text x="181" y="71" font-size="7.5">A15</text>
    <text x="120" y="98" font-size="7" fill-opacity="0.9">heavy work — fast</text>
    <text x="340" y="26" font-size="8.5" font-weight="600">LITTLE cluster</text>
    <text x="274" y="72" font-size="6.5">A7</text>
    <text x="302" y="72" font-size="6.5">A7</text>
    <text x="330" y="72" font-size="6.5">A7</text>
    <text x="358" y="72" font-size="6.5">A7</text>
    <text x="340" y="98" font-size="7" fill-opacity="0.9">idle tasks — efficient</text>
    <text x="230" y="132" font-size="7.5" fill-opacity="0.9">shared memory · scheduler moves work between clusters</text>
  </g>
</svg>
<figcaption>Boards like the ODROID-XU4 use a big.LITTLE layout: a cluster of large high-performance cores paired with a cluster of small efficient ones, letting the scheduler run background work cheaply and burst onto the big cores when load demands.</figcaption>
</figure>

## Overview

ODROID boards mostly use Amlogic ARM chips, and over the years the line has included big.LITTLE designs (the ODROID-XU4), strong general-purpose boards (the ODROID-N2+), and even x86 models. The big.LITTLE arrangement pairs a cluster of fast cores with a cluster of efficient ones so the scheduler can keep light background work on the small cores and only spin up the power-hungry cores when a burst of load arrives — more sustained throughput without a proportional jump in power.

Most run Linux or Android and expose [GPIO](/reference/gpio/) headers, though pinouts and accessories differ from the Pi's, so cases and add-ons are less universally compatible. Hardkernel's reputation is for squeezing genuinely higher performance out of a board at a modest price premium, which is the main reason to choose one over a stock Pi.

## The family

A representative slice of the ODROID line shows the range of designs:

| Board | CPU | Notable trait |
|-------|-----|---------------|
| ODROID-XU4 | 8-core big.LITTLE (A15 + A7) | Heterogeneous cores, high throughput |
| ODROID-N2+ | 6-core Amlogic (A73 + A53) | Strong general-purpose performance |
| ODROID-H series | x86 (Intel) | Runs standard PC software |
| ODROID-C | Amlogic quad-core | Lower-cost Pi-class board |

## Where it fits

ODROID is the alternative you reach for when a [Raspberry Pi](/reference/raspberry-pi/) is not fast enough but a [Jetson](/reference/nvidia-jetson/) is overkill — more CPU and memory bandwidth without a price jump into GPU territory. It competes with [Rock Pi](/reference/rock-pi/) and [Orange Pi](/reference/orange-pi/). For a GopherTrunk node decoding several busy trunked systems at once, an ODROID's extra cores can keep the demodulators fed where a smaller board would fall behind, and the big.LITTLE scheduling means the board still idles efficiently between bursts of traffic.

## Sources

[^wiki]: [ODROID](https://en.wikipedia.org/wiki/ODROID) — Wikipedia, on Hardkernel's ODROID single-board computers.
