---
slug: von-neumann-architecture
title: Von Neumann architecture
entry_type: concept
category: hw-foundations
description: The von Neumann architecture is the stored-program computer design in which instructions and data share one memory accessed over a common bus, the basis of nearly every modern computer.
keywords: von Neumann architecture, stored program, von Neumann bottleneck, CPU, memory, control unit, ALU
aka: [Stored-program architecture]
autolink: true
infobox:
  - { label: Type, value: Computer architecture }
  - { label: Key idea, value: Stored program }
  - { label: Memory, value: Shared for code & data }
  - { label: Named for, value: John von Neumann }
see_also: [central-processing-unit, random-access-memory, system-bus, instruction-set-architecture, cache-memory, logic-gate]
cite_urls:
  - https://en.wikipedia.org/wiki/Von_Neumann_architecture
---

The **von Neumann architecture** is the *stored-program* computer design in which both instructions and data live in the same [memory](/reference/random-access-memory/), accessed by the processor over a common [bus](/reference/system-bus/).[^wiki]

## Overview

In this model a computer has a [CPU](/reference/central-processing-unit/) (with a control unit and an arithmetic/logic unit), a single main memory holding both program and data, and I/O — and the processor works by repeatedly *fetching* an instruction from memory, *decoding* it, and *executing* it. Storing the program in the same memory as the data, rather than wiring it as fixed hardware, is what makes a computer general-purpose: change the program, change what the machine does.

## Where it fits

Almost every mainstream computer follows this design, which is why concepts like the [instruction set architecture](/reference/instruction-set-architecture/) and the fetch-execute cycle are universal. Its famous weakness is the *von Neumann bottleneck*: the shared path between CPU and memory limits throughput, which is exactly why [cache memory](/reference/cache-memory/) and wide modern buses exist. The streaming decode loop in GopherTrunk is, at heart, a von Neumann machine pulling samples and instructions from memory in lockstep.

## Sources

[^wiki]: [Von Neumann architecture](https://en.wikipedia.org/wiki/Von_Neumann_architecture) — Wikipedia, on the stored-program model and the von Neumann bottleneck.
