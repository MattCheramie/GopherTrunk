---
slug: rock-pi
title: Rock Pi (Radxa)
entry_type: hardware
category: hw-sbc
description: Rock Pi is a family of single-board computers from Radxa of China, built mainly on Rockchip processors and often offering NVMe, faster networking, or more RAM than a comparably priced Raspberry Pi.
keywords: Rock Pi, Radxa, Rockchip, RK3588, ROCK 5, NVMe SBC, Raspberry Pi alternative, ARM single-board computer, fast storage
aka: [Radxa Rock]
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Radxa (China) }
  - { label: CPU, value: ARM (Rockchip, e.g. RK3588) }
  - { label: Runs, value: Linux / Android }
  - { label: Known for, value: NVMe, fast I/O, high RAM }
see_also: [raspberry-pi, single-board-computer, odroid, orange-pi, banana-pi, nvme]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radxa
---

**Rock Pi** is a family of [single-board computers](/reference/single-board-computer/) from Radxa of China, built mainly on Rockchip chips and often offering faster I/O than a similarly priced [Raspberry Pi](/reference/raspberry-pi/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A storage comparison. A Raspberry Pi routes its filesystem over a slow microSD card path, shown as a thin pipe. A Rock Pi adds an NVMe solid-state drive on a PCIe lane, shown as a much wider pipe, so recording raw IQ or keeping a large call archive runs far faster." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="28" width="70" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="30" y="92" width="70" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <path d="M100 45 H320" stroke-width="1.6"/>
    <path d="M100 103 L320 103 L320 121 L100 121 Z" fill-opacity="0.14" fill="currentColor"/>
    <rect x="340" y="30" width="86" height="30" rx="3" fill-opacity="0.1" fill="currentColor"/>
    <rect x="340" y="92" width="86" height="34" rx="3" fill-opacity="0.2" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="65" y="42" font-size="8" font-weight="600">Rock Pi</text>
    <text x="65" y="54" font-size="6.5">RK3588</text>
    <text x="65" y="106" font-size="8" font-weight="600">Rasp. Pi</text>
    <text x="65" y="118" font-size="6.5">Pi 4</text>
    <text x="210" y="40" font-size="7">PCIe lane — wide, fast</text>
    <text x="210" y="115" font-size="7">microSD — narrow, slow</text>
    <text x="383" y="49" font-size="8" font-weight="600">NVMe SSD</text>
    <text x="383" y="106" font-size="7.5">microSD</text>
    <text x="383" y="118" font-size="7.5">card</text>
  </g>
</svg>
<figcaption>The Rock Pi's edge is I/O: an NVMe SSD on a PCIe lane moves data far faster than the microSD card a stock Pi of the same price relies on — the difference that matters when a node records raw IQ or keeps a large archive.</figcaption>
</figure>

## Overview

The line is built around Rockchip SoCs — the powerful RK3588 powers the higher-end ROCK 5 boards — and frequently adds features the Pi lacks at the same price, such as an [NVMe](/reference/nvme/) slot, faster Ethernet, or more RAM. The NVMe slot is the headline: instead of running the whole system off a microSD card, a Rock Pi can boot and store data on a proper solid-state drive over PCIe, which transforms any workload that reads or writes a lot of data.

Boards run Linux or Android and keep a Pi-style [GPIO](/reference/gpio/) header, so much of the Pi accessory ecosystem carries over mechanically. As with most Pi alternatives, software polish trails the Raspberry Pi — vendor images and community depth are not quite at Pi levels — but Radxa's boards are generally among the better-supported of the alternatives.

## Rock Pi vs a stock Pi

| | [Raspberry Pi](/reference/raspberry-pi/) 4 | Rock Pi / ROCK 5 |
|---|-------------|------------------|
| SoC | Broadcom quad-core | Rockchip (RK3588, 8-core) |
| Fast storage | microSD (optional USB SSD) | Native NVMe over PCIe |
| RAM ceiling | Up to 8 GB | Up to 16–32 GB |
| Networking | Gigabit | Gigabit / 2.5 GbE on some |
| Software polish | Excellent | Good, trails the Pi |

## Where it fits

Rock Pi competes with [ODROID](/reference/odroid/), [Orange Pi](/reference/orange-pi/), and [Banana Pi](/reference/banana-pi/), and is a natural pick when fast local storage matters. A GopherTrunk node that records raw IQ or keeps days of decoded calls benefits from NVMe rather than an SD card, which is where a Rock Pi can clearly edge out a stock Pi: SD cards are both slow and prone to wear out under continuous writes, while an NVMe drive sustains the throughput a busy capture-and-archive node generates. The extra RAM and cores also give more headroom for decoding several channels at once.

## Sources

[^wiki]: [Radxa](https://en.wikipedia.org/wiki/Radxa) — Wikipedia, on Radxa and its Rock Pi single-board computers.
