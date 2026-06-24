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

## Overview

Caches are usually built from fast SRAM on the processor die and arranged in *levels*: a tiny, fastest L1 per core, a larger L2, and a big shared L3. When the CPU needs data, a *cache hit* serves it immediately; a *cache miss* forces a slow trip to RAM over the [system bus](/reference/system-bus/). Caches work because programs show *locality of reference* — they tend to reuse the same data and access nearby addresses — so a small store captures most accesses.

## Where it fits

Cache exists because main memory cannot keep up with [clock speed](/reference/clock-speed/): without it, a fast CPU would stall waiting on RAM. It sits between the registers and main memory in the broader [memory hierarchy](/reference/memory-hierarchy/). For throughput-heavy code like GopherTrunk's streaming DSP, keeping the hot working set — filter taps, buffers — inside cache is what lets the processor sustain its rate instead of stalling on memory.

## Sources

[^wiki]: [CPU cache](https://en.wikipedia.org/wiki/CPU_cache) — Wikipedia, on cache levels, hits and misses, and locality.
