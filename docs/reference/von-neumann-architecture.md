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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A CPU containing a control unit and an ALU on the left connects over a single shared bus to one main memory holding both program and data, and to input/output. Because instructions and data share that one path, it is the von Neumann bottleneck." xmlns="http://www.w3.org/2000/svg">
  <rect x="24" y="46" width="132" height="104" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.3"/>
  <text x="90" y="62" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">CPU</text>
  <rect x="34" y="72" width="112" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="90" y="91" text-anchor="middle" font-size="8.5" fill="currentColor">control unit</text>
  <rect x="34" y="110" width="112" height="30" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="90" y="129" text-anchor="middle" font-size="8.5" fill="currentColor">ALU</text>
  <rect x="300" y="56" width="134" height="48" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="367" y="76" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">Main memory</text><text x="367" y="91" text-anchor="middle" font-size="8" fill="currentColor">program + data</text>
  <rect x="300" y="120" width="134" height="42" rx="5" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.2"/><text x="367" y="145" text-anchor="middle" font-size="9" fill="currentColor">Input / Output</text>
  <line x1="248" y1="80" x2="248" y2="140" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" stroke-width="2" fill="none">
    <line x1="200" y1="100" x2="156" y2="100" marker-end="url(#vn_ar)"/>
    <line x1="200" y1="100" x2="248" y2="100"/>
  </g>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <line x1="248" y1="80" x2="300" y2="80" marker-end="url(#vn_ar)"/>
    <line x1="248" y1="140" x2="300" y2="140" marker-end="url(#vn_ar)"/>
  </g>
  <text x="202" y="90" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">shared bus</text>
  <path d="M158 158 V164 H246 V158" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <text x="202" y="180" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">instructions + data on one path = the bottleneck</text>
  <defs><marker id="vn_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Both the program and its data live in one memory, reached by the CPU over a single shared bus. Storing the program alongside the data — rather than wiring it as fixed hardware — is what makes the machine general-purpose. The cost is the von Neumann bottleneck: that one path caps throughput, which is exactly why caches and wide buses exist.</figcaption>
</figure>

## Overview

In this model a computer has a [CPU](/reference/central-processing-unit/) (with a control unit and an arithmetic/logic unit), a single main memory holding both program and data, and I/O — and the processor works by repeatedly *fetching* an instruction from memory, *decoding* it, and *executing* it. Storing the program in the same memory as the data, rather than wiring it as fixed hardware, is what makes a computer general-purpose: change the program, change what the machine does.

## Where it fits

Almost every mainstream computer follows this design, which is why concepts like the [instruction set architecture](/reference/instruction-set-architecture/) and the fetch-execute cycle are universal. Its famous weakness is the *von Neumann bottleneck*: the shared path between CPU and memory limits throughput, which is exactly why [cache memory](/reference/cache-memory/) and wide modern buses exist. The streaming decode loop in GopherTrunk is, at heart, a von Neumann machine pulling samples and instructions from memory in lockstep.

## Sources

[^wiki]: [Von Neumann architecture](https://en.wikipedia.org/wiki/Von_Neumann_architecture) — Wikipedia, on the stored-program model and the von Neumann bottleneck.
