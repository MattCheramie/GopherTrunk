---
slug: clock-speed
title: Clock speed
entry_type: concept
category: hw-foundations
description: Clock speed is the rate at which a processor's internal clock ticks, measured in hertz, setting the pace at which it steps through instructions — though it is only one factor in real performance.
keywords: clock speed, clock rate, gigahertz, GHz, frequency, throttling, overclocking, IPC
aka: [Clock rate]
infobox:
  - { label: Type, value: Processor timing metric }
  - { label: Unit, value: Hertz (MHz, GHz) }
  - { label: Sets, value: Instruction pacing }
  - { label: Limited by, value: Heat & power }
see_also: [central-processing-unit, cooling-and-thermals, cache-memory, moores-law, transistor, instruction-set-architecture]
cite_urls:
  - https://en.wikipedia.org/wiki/Clock_rate
---

**Clock speed** (or clock rate) is the frequency at which a processor's internal clock ticks, measured in hertz, setting the basic pace at which the chip steps through its work.[^wiki]

## Overview

A digital chip advances in time with a clock signal; each tick, or *cycle*, lets the logic settle and move to its next state. A 3 GHz [CPU](/reference/central-processing-unit/) ticks three billion times a second. But clock speed alone does not equal performance — *how much work per cycle* (instructions per cycle, dependent on the [microarchitecture](/reference/instruction-set-architecture/), [cache](/reference/cache-memory/) behavior, and parallelism) matters just as much. Comparing clock speeds across different designs is therefore unreliable.

## Where it fits

For years rising clock speeds delivered most of computing's gains, but heat and power put a hard ceiling on how fast a chip can tick — push too far and it overheats, so [cooling](/reference/cooling-and-thermals/) and *throttling* govern the real sustained rate. With the end of easy clock scaling (part of the Dennard-scaling breakdown noted under [Moore's law](/reference/moores-law/)), designers turned to more cores instead. For steady SDR decoding, sustained clock under thermal load matters more than a high peak boost a node cannot hold.

## Sources

[^wiki]: [Clock rate](https://en.wikipedia.org/wiki/Clock_rate) — Wikipedia, on processor clock frequency and its limits.
