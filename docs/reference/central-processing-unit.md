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

## Overview
A CPU fetches instructions, decodes them, and executes them one after another, very fast. Modern CPUs contain several **cores**: each core is a self-contained processing unit, so a multi-core CPU can genuinely work on several tasks at the same time. This is what makes [concurrency](/reference/concurrency/) and parallel work practical.

**Clock speed**, measured in gigahertz (GHz), counts how many cycles a core runs per second. It is a rough guide to speed, not an absolute one — a newer 3 GHz core can outpace an older 4 GHz core because it does more useful work per cycle.

## Where it fits
The CPU scales across the whole [hardware spectrum](/reference/hardware-spectrum/). A [microcontroller](/reference/microcontroller/) has a single tiny, low-power core; a desktop has a handful; a [server](/reference/server/) may carry dozens. The CPU works hand in hand with [RAM](/reference/random-access-memory/), which holds the data and instructions it is actively using.

## Sources
[^wiki]: [Central processing unit](https://en.wikipedia.org/wiki/Central_processing_unit) — Wikipedia, on CPU function, cores, and clock speed.
