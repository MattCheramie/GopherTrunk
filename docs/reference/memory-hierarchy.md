---
slug: memory-hierarchy
title: Memory hierarchy
entry_type: concept
category: hw-storage
description: The memory hierarchy arranges a computer's storage in layers trading speed against capacity and cost, from fast registers and cache down to RAM, flash, disk, and tape.
keywords: memory hierarchy, cache, RAM, registers, storage tiers, latency, locality, speed vs capacity
infobox:
  - { label: Type, value: Storage organisation principle }
  - { label: Top (fast/small), value: Registers, cache }
  - { label: Middle, value: RAM, SSD }
  - { label: Bottom (slow/large), value: HDD, tape }
  - { label: Trade-off, value: Speed vs capacity vs cost }
see_also: [cache-memory, random-access-memory, solid-state-drive, hard-disk-drive, volatile-memory, data-storage]
related_lessons:
  - { title: "Building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Memory_hierarchy
---

The **memory hierarchy** is the layered arrangement of a computer's storage, ordered so that each level trades speed for capacity and cost against the one below it.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 250" role="img" aria-label="A pyramid of storage levels, narrow and fast at the top widening to large and slow at the bottom: CPU registers, then cache, then RAM, then SSD, then hard disk, then magnetic tape. An arrow up the left marks faster, smaller, costlier per byte; an arrow down the right marks larger, cheaper, slower." xmlns="http://www.w3.org/2000/svg">
  <g text-anchor="middle" fill="currentColor" font-size="10">
    <rect x="175" y="28" width="90" height="30" rx="3" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.1"/><text x="220" y="47">registers</text>
    <rect x="145" y="60" width="150" height="30" rx="3" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1.1"/><text x="220" y="79">CPU cache · L1/L2/L3</text>
    <rect x="115" y="92" width="210" height="30" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="220" y="111">RAM · main memory</text>
    <rect x="85" y="124" width="270" height="30" rx="3" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.1"/><text x="220" y="143">SSD · flash</text>
    <rect x="55" y="156" width="330" height="30" rx="3" fill="currentColor" fill-opacity="0.09" stroke="currentColor" stroke-width="1.1"/><text x="220" y="175">hard disk</text>
    <rect x="25" y="188" width="390" height="30" rx="3" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.1"/><text x="220" y="207">magnetic tape · archive</text>
  </g>
  <line x1="14" y1="216" x2="14" y2="32" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7" marker-end="url(#mh_ar)"/>
  <text x="12" y="126" font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(-90 12 126)">faster · smaller · costlier/byte</text>
  <line x1="428" y1="32" x2="428" y2="216" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7" marker-end="url(#mh_ar)"/>
  <text x="426" y="124" font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(90 426 124)">larger · cheaper · slower</text>
  <text x="220" y="238" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85">locality keeps hot data in the fast upper levels</text>
  <defs><marker id="mh_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Every storage device is really a point on one curve. The top levels are tiny, fast, and expensive per byte; each step down is larger and cheaper but slower to reach. It works because programs reuse the same and nearby data (locality), so a small fast top captures most accesses.</figcaption>
</figure>

## Overview

At the top sit the [CPU](/reference/central-processing-unit/) registers and [cache memory](/reference/cache-memory/): tiny, blisteringly fast, and expensive per byte. Below them is main memory, [RAM](/reference/random-access-memory/), which is larger but slower. Further down comes persistent storage — a [solid-state drive](/reference/solid-state-drive/), then a [hard disk drive](/reference/hard-disk-drive/), and finally archival [magnetic tape](/reference/magnetic-tape/) — each step larger and cheaper but slower to reach. The hierarchy works because programs exhibit *locality*: they tend to reuse the same data and nearby data, so keeping hot items in the fast upper levels gives most of the speed of fast memory at most of the cost of slow storage.

## Where it fits

The hierarchy is the unifying framework for the whole storage-and-memory category: every device here is really a point on this speed-versus-capacity curve, with the [volatile/non-volatile](/reference/volatile-memory/) split cutting across it. The [operating system](/reference/operating-system/) and hardware constantly shuttle data between levels — caching disk blocks in RAM, paging RAM to disk. GopherTrunk benefits the same way: hot decode state stays in RAM and cache, the working call database lives on fast [SSD](/reference/solid-state-drive/), and bulk captures sink to cheap disk or tape.

## Sources

[^wiki]: [Memory hierarchy](https://en.wikipedia.org/wiki/Memory_hierarchy) — Wikipedia, on the layered organisation of computer storage by speed and cost.
