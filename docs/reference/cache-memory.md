---
slug: cache-memory
title: Cache memory
entry_type: hardware
category: hw-foundations
description: Cache memory is a small, very fast store close to the CPU that holds recently used data and instructions, bridging the speed gap between the processor and main memory.
keywords: cache memory, CPU cache, L1, L2, L3, cache hit, cache miss, locality, SRAM
aka: [CPU cache]
infobox:
  - { label: Type, value: Fast on-chip memory (SRAM) }
  - { label: Levels, value: "L1, L2, L3" }
  - { label: Goal, value: Hide main-memory latency }
  - { label: Exploits, value: Locality of reference }
see_also: [central-processing-unit, random-access-memory, clock-speed, system-bus, integrated-circuit, von-neumann-architecture]
cite_urls:
  - https://en.wikipedia.org/wiki/CPU_cache
---

**Cache memory** is a small, very fast memory close to the [CPU](/reference/central-processing-unit/) that keeps recently and frequently used data and instructions on hand, hiding the latency of slower [main memory](/reference/random-access-memory/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 168" role="img" aria-label="A row from CPU outward: CPU, then on-chip caches L1, L2, and L3 growing larger and slower, then main memory RAM. A cache hit returns immediately; a miss travels one level further out toward RAM." xmlns="http://www.w3.org/2000/svg">
  <line x1="96" y1="44" x2="332" y2="44" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <line x1="96" y1="44" x2="96" y2="52" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <line x1="332" y1="44" x2="332" y2="52" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <text x="214" y="38" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85">on-chip cache · SRAM</text>
  <g text-anchor="middle" fill="currentColor" font-size="10">
    <rect x="18" y="60" width="58" height="46" rx="5" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="47" y="87">CPU</text>
    <rect x="96" y="60" width="52" height="46" rx="5" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.2"/><text x="122" y="82">L1</text><text x="122" y="96" font-size="7.5">tiny</text>
    <rect x="168" y="60" width="62" height="46" rx="5" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="199" y="82">L2</text><text x="199" y="96" font-size="7.5">bigger</text>
    <rect x="250" y="60" width="82" height="46" rx="5" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.2"/><text x="291" y="82">L3</text><text x="291" y="96" font-size="7.5">shared</text>
    <rect x="360" y="60" width="102" height="46" rx="5" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.2"/><text x="411" y="82">RAM</text><text x="411" y="96" font-size="7.5">large · slow</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75">
    <line x1="76" y1="83" x2="96" y2="83" marker-end="url(#ch_ar)"/>
    <line x1="148" y1="83" x2="168" y2="83" marker-end="url(#ch_ar)"/>
    <line x1="230" y1="83" x2="250" y2="83" marker-end="url(#ch_ar)"/>
    <line x1="332" y1="83" x2="360" y2="83" marker-end="url(#ch_ar)"/>
  </g>
  <text x="122" y="126" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">hit → return</text>
  <text x="278" y="126" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">miss → try the next level out</text>
  <text x="240" y="150" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85">fast &amp; small near the core · large &amp; slow further out</text>
  <defs><marker id="ch_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Cache bridges the speed gap between the fast CPU and slow RAM. A hit in L1 returns almost instantly; a miss walks outward — L2, L3, then RAM — each level larger but slower. Locality of reference is what keeps most accesses in the small, fast levels.</figcaption>
</figure>

## Overview

Caches are usually built from fast SRAM on the processor die and arranged in *levels*: a tiny, fastest L1 per core, a larger L2, and a big shared L3. When the CPU needs data, a *cache hit* serves it immediately; a *cache miss* forces a slow trip to RAM over the [system bus](/reference/system-bus/). Caches work because programs show *locality of reference* — they tend to reuse the same data and access nearby addresses — so a small store captures most accesses.

## Where it fits

Cache exists because main memory cannot keep up with [clock speed](/reference/clock-speed/): without it, a fast CPU would stall waiting on RAM. It sits between the registers and main memory in the broader [memory hierarchy](/reference/memory-hierarchy/). For throughput-heavy code like GopherTrunk's streaming DSP, keeping the hot working set — filter taps, buffers — inside cache is what lets the processor sustain its rate instead of stalling on memory.

## Sources

[^wiki]: [CPU cache](https://en.wikipedia.org/wiki/CPU_cache) — Wikipedia, on cache levels, hits and misses, and locality.
