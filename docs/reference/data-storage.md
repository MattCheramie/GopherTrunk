---
slug: data-storage
title: Storage (SSD, HDD, flash)
entry_type: hardware
category: hw-foundations
description: Storage is where a computer keeps data long-term so it survives a power cycle, using technologies such as hard drives, solid-state drives, and flash memory.
keywords: storage, SSD, HDD, hard drive, flash memory, SD card, non-volatile, permanent storage
aka: [storage, data storage]
infobox:
  - { label: Type, value: Long-term storage }
  - { label: Survives power loss, value: Yes (non-volatile) }
  - { label: Forms, value: HDD, SSD, flash }
  - { label: Speed, value: Slower than RAM }
see_also: [computer-hardware, random-access-memory, central-processing-unit, single-board-computer, microcontroller]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_data_storage
---

**Storage** is where a computer keeps data long-term, so that files and programs survive a power cycle.[^wiki] It is the permanent counterpart to [RAM](/reference/random-access-memory/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 230" role="img" aria-label="A pyramid of storage tiers: RAM at the narrow fast top, then SSD, then hard disk, then magnetic tape widening toward the bottom. A dashed line between RAM and SSD marks the volatile split — everything above is lost at power-off, everything below is kept. An arrow up the left reads faster and smaller; an arrow down the right reads larger, cheaper, and slower." xmlns="http://www.w3.org/2000/svg">
  <g text-anchor="middle" fill="currentColor" font-size="10">
    <rect x="150" y="40" width="140" height="32" rx="3" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/><text x="220" y="60">RAM · working memory</text>
    <rect x="110" y="76" width="220" height="32" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.1"/><text x="220" y="96">SSD · flash — no moving parts</text>
    <rect x="70" y="112" width="300" height="32" rx="3" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.1"/><text x="220" y="132">HDD · spinning platters</text>
    <rect x="30" y="148" width="380" height="32" rx="3" fill="currentColor" fill-opacity="0.07" stroke="currentColor" stroke-width="1.1"/><text x="220" y="168">magnetic tape · archive</text>
  </g>
  <line x1="30" y1="74" x2="410" y2="74" stroke="currentColor" stroke-width="1.1" stroke-dasharray="5 3" stroke-opacity="0.7"/>
  <text x="352" y="58" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">volatile ↑</text>
  <text x="360" y="100" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">non-volatile ↓</text>
  <line x1="16" y1="176" x2="16" y2="44" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7" marker-end="url(#ds_ar)"/>
  <text x="14" y="112" font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(-90 14 112)">faster · smaller</text>
  <line x1="424" y1="44" x2="424" y2="176" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7" marker-end="url(#ds_ar)"/>
  <text x="422" y="112" font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(90 422 112)">larger · cheaper · slower</text>
  <text x="220" y="204" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">the CPU works in RAM; anything worth keeping sinks to the tiers below</text>
  <defs><marker id="ds_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Every storage device is a point on one speed-versus-cost curve: RAM at the top is fast but volatile, and each step down — SSD, hard disk, tape — is larger and cheaper per byte but slower. The dashed line is the volatile/non-volatile split: only RAM loses its contents at power-off, which is why the CPU works in memory and writes anything lasting to the tiers below.</figcaption>
</figure>

## Overview
Storage comes in a few common forms:

- **HDD** (hard disk drive): data on spinning magnetic platters — cheap and high-capacity, but slower and mechanically fragile.
- **SSD** (solid-state drive): flash memory with no moving parts — faster, quieter, and more rugged.
- **Flash memory**: the non-volatile technology inside SSDs, SD cards, and the on-board storage of small devices.

All of these are **non-volatile** — they hold their contents with the power off, unlike RAM. The trade-off is speed: even fast storage is much slower than RAM, which is why the [CPU](/reference/central-processing-unit/) does its active work in memory and reads from or writes to storage only when it must.

## Where it fits
Storage is one of the four building blocks of [computer hardware](/reference/computer-hardware/). On a [single-board computer](/reference/single-board-computer/) it is often a microSD card; on a [microcontroller](/reference/microcontroller/) it may be a small block of on-chip flash; on a [server](/reference/server/) it can be racks of SSDs.

## Sources
[^wiki]: [Computer data storage](https://en.wikipedia.org/wiki/Computer_data_storage) — Wikipedia, on non-volatile storage and HDD/SSD/flash technologies.
