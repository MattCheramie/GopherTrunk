---
slug: single-board-computer
title: Single-board computer (SBC)
entry_type: hardware
category: hw-sbc
description: A single-board computer (SBC) is a complete computer built on one small circuit board, with CPU, memory, storage, and ports, capable of running a full operating system.
keywords: single-board computer, SBC, Raspberry Pi, embedded Linux, GPIO, edge device, low power computer, capture node
aka: [SBC]
autolink: true
infobox:
  - { label: Type, value: Complete computer on one board }
  - { label: CPU, value: Usually ARM (sometimes x86) }
  - { label: RAM, value: ~512 MB – 16 GB }
  - { label: Runs, value: Linux (full OS) }
  - { label: Typical price, value: ~$15 – $100 }
see_also: [raspberry-pi, nvidia-jetson, beaglebone, gpio, microcontroller, personal-computer]
related_lessons:
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Single-board_computer
---

**A single-board computer (SBC)** is a complete computer built on one small circuit board — [CPU](/reference/central-processing-unit/), [memory](/reference/random-access-memory/), [storage](/reference/data-storage/), and ports — capable of running a full [operating system](/reference/operating-system/), usually Linux.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A spectrum placing the single-board computer between a microcontroller and a personal computer. The microcontroller runs bare firmware with pins but no operating system; the SBC in the middle runs a full Linux operating system yet still exposes GPIO pins; the personal computer runs a full OS but is sealed with no direct pin access." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <line x1="30" y1="46" x2="430" y2="46" stroke-width="1.2"/>
    <rect x="34" y="60" width="104" height="52" rx="5" fill-opacity="0.05" fill="currentColor"/>
    <rect x="178" y="56" width="104" height="60" rx="5" fill-opacity="0.16" fill="currentColor"/>
    <rect x="322" y="60" width="104" height="52" rx="5" fill-opacity="0.05" fill="currentColor"/>
    <circle cx="52" cy="46" r="4" fill-opacity="0.2" fill="currentColor"/>
    <circle cx="230" cy="46" r="5" fill="currentColor" fill-opacity="0.5"/>
    <circle cx="408" cy="46" r="4" fill-opacity="0.2" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="86" y="38" font-size="8.5" font-weight="600">microcontroller</text>
    <text x="86" y="82" font-size="7.5">firmware only</text>
    <text x="86" y="96" font-size="7.5">pins, no OS</text>
    <text x="230" y="34" font-size="9" font-weight="600">SBC</text>
    <text x="230" y="80" font-size="7.5">full Linux OS</text>
    <text x="230" y="94" font-size="7.5">and GPIO pins</text>
    <text x="230" y="108" font-size="7.5" fill-opacity="0.9">a few watts</text>
    <text x="374" y="38" font-size="8.5" font-weight="600">personal computer</text>
    <text x="374" y="82" font-size="7.5">full OS</text>
    <text x="374" y="96" font-size="7.5">sealed, no pins</text>
    <text x="230" y="138" font-size="7.5" fill-opacity="0.9">runs a real OS like a PC — but exposes pins like a microcontroller</text>
  </g>
</svg>
<figcaption>An SBC occupies the middle ground: it runs a full operating system the way a PC does, yet still breaks out GPIO pins to the physical world the way a microcontroller does — all on one small, few-watt board.</figcaption>
</figure>

## Overview

An SBC sits between a [personal computer](/reference/personal-computer/) and a [microcontroller](/reference/microcontroller/). Unlike a microcontroller, it runs a real OS and ordinary languages and tools; unlike a sealed PC or phone, it exposes [GPIO](/reference/gpio/) pins that let your code talk directly to electronics. That combination — a familiar Linux environment plus direct hardware access — is what makes SBCs so widely used for tinkering, prototyping, and embedding.

Most are credit-card sized and draw only a few watts, so they can run continuously without a fan or a meaningful power bill and fit inside small enclosures. Nearly all are built around an ARM [system-on-a-chip](/reference/system-on-a-chip/) that packs the CPU, GPU, and I/O onto one die, with RAM, storage (a microSD card, eMMC, or increasingly NVMe), and the ports all soldered to the single board.

## SBC vs its neighbours

| | Microcontroller | Single-board computer | Personal computer |
|---|-----------------|------------------------|-------------------|
| Runs an OS | No, bare firmware | Yes, full Linux | Yes, full OS |
| GPIO pins | Yes | Yes | No (sealed) |
| Typical power | Milliwatts | A few watts | Tens–hundreds of watts |
| Multitasking | Very limited | Yes | Yes |
| Best at | Tight real-time control | Embedded, always-on jobs | Heavy general compute |

## Where it fits

The category is broad: the [Raspberry Pi](/reference/raspberry-pi/) is the best-known general-purpose board, the [NVIDIA Jetson](/reference/nvidia-jetson/) adds a GPU for edge AI, and the [BeagleBone](/reference/beaglebone/) emphasises real-time I/O. Their low power and small size make them well suited to always-on, embedded, and field roles. For GopherTrunk this is a genuine fit rather than a stretch: a Raspberry Pi or similar board by the antenna can run the software as a small, low-power headless SDR capture-and-decode node, and several such nodes can be spread across sites to feed one collector.

## Sources

[^wiki]: [Single-board computer](https://en.wikipedia.org/wiki/Single-board_computer) — Wikipedia, on the definition and scope of SBCs.
