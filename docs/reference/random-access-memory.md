---
slug: random-access-memory
title: Random-access memory (RAM)
entry_type: hardware
category: hw-foundations
description: Random-access memory (RAM) is a computer's fast, temporary working storage, holding the data and programs in active use and clearing when power is lost.
keywords: RAM, random access memory, working memory, volatile memory, memory vs storage
aka: [RAM, random-access memory, memory]
infobox:
  - { label: Type, value: Working memory }
  - { label: Speed, value: Very fast }
  - { label: Volatile, value: Yes (cleared on power loss) }
  - { label: Range, value: Kilobytes to terabytes }
see_also: [computer-hardware, central-processing-unit, data-storage, input-output, server]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Random-access_memory
---

**Random-access memory (RAM)** is a computer's fast, temporary working storage — it holds the data and programs in active use so the [CPU](/reference/central-processing-unit/) can reach them quickly.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 236" role="img" aria-label="Top: one DRAM cell — a single access transistor gated by a word line and switched onto a bit line, storing one bit as charge on a tiny capacitor. Bottom: those cells packed into a grid addressed by rows and columns; the charge leaks away so each row must be refreshed constantly, and every bit vanishes when power is removed." xmlns="http://www.w3.org/2000/svg">
  <text x="150" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">One DRAM cell: 1 transistor + 1 capacitor</text>
  <g fill="currentColor" font-size="8">
    <line x1="208" y1="30" x2="208" y2="52" stroke="currentColor" stroke-width="1.1"/>
    <text x="213" y="37" text-anchor="start">bit line</text>
    <rect x="190" y="52" width="36" height="22" rx="2" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.1"/>
    <text x="208" y="66" text-anchor="middle">T</text>
    <line x1="120" y1="63" x2="190" y2="63" stroke="currentColor" stroke-width="1.1"/>
    <text x="117" y="60" text-anchor="end">word line (select)</text>
    <line x1="208" y1="74" x2="208" y2="82" stroke="currentColor" stroke-width="1.1"/>
    <line x1="188" y1="82" x2="228" y2="82" stroke="currentColor" stroke-width="1.6"/>
    <line x1="188" y1="90" x2="228" y2="90" stroke="currentColor" stroke-width="1.6"/>
    <circle cx="198" cy="86" r="1.6" fill="currentColor"/><circle cx="208" cy="86" r="1.6" fill="currentColor"/><circle cx="218" cy="86" r="1.6" fill="currentColor"/>
    <line x1="208" y1="90" x2="208" y2="99" stroke="currentColor" stroke-width="1.1"/>
    <line x1="200" y1="99" x2="216" y2="99" stroke="currentColor" stroke-width="1.1"/>
    <text x="234" y="84" text-anchor="start">capacitor holds</text>
    <text x="234" y="94" text-anchor="start" fill-opacity="0.85">charge = the bit</text>
  </g>
  <text x="220" y="128" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Cells in a grid — addressed by row × column</text>
  <g fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="0.8">
    <rect x="104" y="150" width="26" height="14" rx="2"/><rect x="132" y="150" width="26" height="14" rx="2"/><rect x="160" y="150" width="26" height="14" rx="2"/><rect x="188" y="150" width="26" height="14" rx="2"/><rect x="216" y="150" width="26" height="14" rx="2"/><rect x="244" y="150" width="26" height="14" rx="2"/><rect x="272" y="150" width="26" height="14" rx="2"/><rect x="300" y="150" width="26" height="14" rx="2"/><rect x="328" y="150" width="26" height="14" rx="2"/>
    <rect x="104" y="168" width="26" height="14" rx="2"/><rect x="132" y="168" width="26" height="14" rx="2"/><rect x="160" y="168" width="26" height="14" rx="2"/><rect x="188" y="168" width="26" height="14" rx="2"/><rect x="216" y="168" width="26" height="14" rx="2"/><rect x="244" y="168" width="26" height="14" rx="2"/><rect x="272" y="168" width="26" height="14" rx="2"/><rect x="300" y="168" width="26" height="14" rx="2"/><rect x="328" y="168" width="26" height="14" rx="2"/>
    <rect x="104" y="186" width="26" height="14" rx="2"/><rect x="132" y="186" width="26" height="14" rx="2"/><rect x="160" y="186" width="26" height="14" rx="2"/><rect x="188" y="186" width="26" height="14" rx="2"/><rect x="216" y="186" width="26" height="14" rx="2"/><rect x="244" y="186" width="26" height="14" rx="2"/><rect x="272" y="186" width="26" height="14" rx="2"/><rect x="300" y="186" width="26" height="14" rx="2"/><rect x="328" y="186" width="26" height="14" rx="2"/>
  </g>
  <rect x="100" y="146" width="258" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
  <text x="368" y="160" text-anchor="start" font-size="8" fill="currentColor" fill-opacity="0.9">one row</text>
  <text x="220" y="220" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">charge leaks → rows refreshed constantly · all lost at power-off (volatile)</text>
</svg>
<figcaption>Each DRAM cell is just one transistor guarding one tiny capacitor, and the stored bit is the charge on that capacitor. Cells pack into a row-and-column grid, read by selecting a word line onto a bit line. Because the charge leaks, every row must be refreshed thousands of times a second — and it all disappears the instant power is removed, which is what makes RAM volatile.</figcaption>
</figure>

## Overview
RAM is **volatile**: its contents are lost the moment power is removed. That is the key contrast with [storage](/reference/data-storage/), which is permanent. When you open a program, the device copies what it needs from storage into RAM, works there at high speed, then saves anything worth keeping back to storage before shutting down.

More RAM lets a device hold more programs and larger data sets in active use at once. Run out, and the system slows to a crawl as it shuffles data back and forth with slower storage.

## Where it fits
RAM is one of the four building blocks of [computer hardware](/reference/computer-hardware/), and its size tracks the [hardware spectrum](/reference/hardware-spectrum/): a [microcontroller](/reference/microcontroller/) may have only a few kilobytes, a phone several gigabytes, and a large [server](/reference/server/) terabytes.

## Sources
[^wiki]: [Random-access memory](https://en.wikipedia.org/wiki/Random-access_memory) — Wikipedia, on RAM as fast volatile working memory.
