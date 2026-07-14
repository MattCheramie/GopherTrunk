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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 180" role="img" aria-label="A square-wave clock signal alternating between low and high over time. One low-to-high-to-low span is marked as a single cycle; the number of cycles per second is the frequency in hertz, so a three-gigahertz processor completes three billion cycles each second. Heat and power cap how fast the wave can go." xmlns="http://www.w3.org/2000/svg">
  <line x1="34" y1="130" x2="34" y2="40" stroke="currentColor" stroke-width="1" stroke-opacity="0.6" marker-end="url(#cs_ar)"/>
  <text x="30" y="86" font-size="8" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(-90 30 86)">voltage</text>
  <line x1="34" y1="120" x2="410" y2="120" stroke="currentColor" stroke-width="1" stroke-opacity="0.6" marker-end="url(#cs_ar)"/>
  <text x="400" y="134" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">time</text>
  <path d="M44 118 V56 H76 V118 H108 V56 H140 V118 H172 V56 H204 V118 H236 V56 H268 V118 H300 V56 H332 V118 H364" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <line x1="44" y1="44" x2="108" y2="44" stroke="currentColor" stroke-width="1" stroke-opacity="0.8" marker-start="url(#cs_ar)" marker-end="url(#cs_ar)"/>
  <text x="76" y="40" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">1 cycle (tick)</text>
  <text x="220" y="152" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">frequency = cycles per second (Hz) — a 3 GHz CPU ticks 3 billion times a second</text>
  <text x="220" y="170" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">heat &amp; power cap how fast it can tick</text>
  <defs><marker id="cs_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto-start-reverse"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A clock is a square wave: each low-to-high tick, or cycle, lets the logic settle and advance one step. How many cycles fit in a second is the frequency — 3 GHz means three billion a second. But heat and power cap how fast it can tick, and how much work each cycle does matters just as much, so clock speed alone does not equal performance.</figcaption>
</figure>

## Overview

A digital chip advances in time with a clock signal; each tick, or *cycle*, lets the logic settle and move to its next state. A 3 GHz [CPU](/reference/central-processing-unit/) ticks three billion times a second. But clock speed alone does not equal performance — *how much work per cycle* (instructions per cycle, dependent on the [microarchitecture](/reference/instruction-set-architecture/), [cache](/reference/cache-memory/) behavior, and parallelism) matters just as much. Comparing clock speeds across different designs is therefore unreliable.

## Where it fits

For years rising clock speeds delivered most of computing's gains, but heat and power put a hard ceiling on how fast a chip can tick — push too far and it overheats, so [cooling](/reference/cooling-and-thermals/) and *throttling* govern the real sustained rate. With the end of easy clock scaling (part of the Dennard-scaling breakdown noted under [Moore's law](/reference/moores-law/)), designers turned to more cores instead. For steady SDR decoding, sustained clock under thermal load matters more than a high peak boost a node cannot hold.

## Sources

[^wiki]: [Clock rate](https://en.wikipedia.org/wiki/Clock_rate) — Wikipedia, on processor clock frequency and its limits.
