---
slug: moores-law
title: Moore's law
entry_type: concept
category: hw-foundations
description: Moore's law is the observation that the number of transistors on an integrated circuit roughly doubles about every two years, driving decades of exponential gains in computing.
keywords: Moore's law, transistor density, Gordon Moore, exponential, scaling, integrated circuit, semiconductor
aka: [Moore's Law]
autolink: true
infobox:
  - { label: Type, value: Empirical observation }
  - { label: Stated by, value: "Gordon Moore, 1965" }
  - { label: Claim, value: Transistor count doubles ~2 yrs }
  - { label: Status, value: Slowing }
see_also: [transistor, integrated-circuit, semiconductor, central-processing-unit, clock-speed, graphics-processing-unit]
cite_urls:
  - https://en.wikipedia.org/wiki/Moore%27s_law
---

**Moore's law** is the observation, made by Intel co-founder Gordon Moore in 1965, that the number of [transistors](/reference/transistor/) on an [integrated circuit](/reference/integrated-circuit/) roughly doubles about every two years.[^wiki]

## Overview

It is not a law of physics but an empirical trend that held for decades as [semiconductor](/reference/semiconductor/) makers steadily shrank feature sizes, packing ever more switches onto a chip at falling cost per transistor. That exponential scaling is what turned room-sized computers into pocket-sized ones and made each generation of [CPUs](/reference/central-processing-unit/) and [GPUs](/reference/graphics-processing-unit/) dramatically more capable. A related trend, Dennard scaling, once let [clock speeds](/reference/clock-speed/) rise as transistors shrank — but that broke down in the mid-2000s, pushing designers toward multiple cores.

## Where it fits

Moore's law explains why computing got cheap and abundant enough that a hobbyist can run real DSP — like GopherTrunk decoding multiple trunked channels — on a board costing a few tens of dollars. The doubling has slowed as features approach atomic scale, so progress now leans more on parallelism, specialized accelerators, and packaging than on raw shrinking.

## Sources

[^wiki]: [Moore's law](https://en.wikipedia.org/wiki/Moore%27s_law) — Wikipedia, on the transistor-doubling trend and its slowdown.
