---
slug: central-processing-unit
title: Central processing unit (CPU)
entry_type: hardware
category: hw-foundations
description: A central processing unit (CPU) is the chip that carries out a program's instructions — the part of a computer most often called its brain.
keywords: CPU, central processing unit, processor, cores, multi-core, clock speed, GHz
aka: [CPU, central processing unit, processor]
infobox:
  - { label: Type, value: Processor chip }
  - { label: Role, value: Executes instructions }
  - { label: Measured by, value: Cores, clock speed (GHz) }
  - { label: Range, value: One tiny core to dozens }
see_also: [computer-hardware, random-access-memory, data-storage, microcontroller, server]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Central_processing_unit
---

**The central processing unit (CPU)** is the chip that carries out a program's instructions — the part of [computer hardware](/reference/computer-hardware/) most often called the "brain."[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 190" role="img" aria-label="On the left, one CPU core repeats a three-step instruction cycle — fetch, decode, execute — looping back to fetch. On the right, a CPU package holds four such cores sharing a cache, exchanging data with separate RAM." xmlns="http://www.w3.org/2000/svg">
  <text x="130" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">One core: the instruction cycle</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="66" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="53" y="71">Fetch</text>
    <rect x="97" y="52" width="66" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="130" y="71">Decode</text>
    <rect x="174" y="52" width="66" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="207" y="71">Execute</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="86" y1="67" x2="97" y2="67" marker-end="url(#cpu_ar)"/>
    <line x1="163" y1="67" x2="174" y2="67" marker-end="url(#cpu_ar)"/>
    <path d="M207 82 V 108 H 53 V 82" marker-end="url(#cpu_ar)"/>
  </g>
  <text x="130" y="126" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.8">cycles per second = clock speed (GHz)</text>
  <text x="375" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">One package: many cores</text>
  <rect x="288" y="30" width="122" height="96" rx="7" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="298" y="40" width="46" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="321" y="58">core</text>
    <rect x="354" y="40" width="46" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="377" y="58">core</text>
    <rect x="298" y="74" width="46" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="321" y="92">core</text>
    <rect x="354" y="74" width="46" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="377" y="92">core</text>
    <rect x="298" y="108" width="102" height="12" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.8"/><text x="349" y="117">shared cache</text>
  </g>
  <rect x="288" y="146" width="122" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="349" y="163" text-anchor="middle" font-size="9" fill="currentColor">RAM</text>
  <line x1="349" y1="126" x2="349" y2="146" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2" marker-end="url(#cpu_ar)"/>
  <text x="428" y="139" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.8">data</text>
  <defs><marker id="cpu_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A single core runs the same loop — fetch, decode, execute — billions of times a second; that rate is the clock speed. A modern CPU packs several cores that share a cache and pass data to and from separate RAM, so it can genuinely work on many tasks at once.</figcaption>
</figure>

## Overview
A CPU fetches instructions, decodes them, and executes them one after another, very fast. Modern CPUs contain several **cores**: each core is a self-contained processing unit, so a multi-core CPU can genuinely work on several tasks at the same time. This is what makes [concurrency](/reference/concurrency/) and parallel work practical.

**Clock speed**, measured in gigahertz (GHz), counts how many cycles a core runs per second. It is a rough guide to speed, not an absolute one — a newer 3 GHz core can outpace an older 4 GHz core because it does more useful work per cycle.

## Where it fits
The CPU scales across the whole [hardware spectrum](/reference/hardware-spectrum/). A [microcontroller](/reference/microcontroller/) has a single tiny, low-power core; a desktop has a handful; a [server](/reference/server/) may carry dozens. The CPU works hand in hand with [RAM](/reference/random-access-memory/), which holds the data and instructions it is actively using.

## Sources
[^wiki]: [Central processing unit](https://en.wikipedia.org/wiki/Central_processing_unit) — Wikipedia, on CPU function, cores, and clock speed.
