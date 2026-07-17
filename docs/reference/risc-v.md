---
slug: risc-v
title: RISC-V
entry_type: concept
category: hw-foundations
description: RISC-V is an open, royalty-free RISC instruction set architecture that anyone can implement freely, built as a small base plus optional extensions, used in research, embedded systems, and a growing range of processors.
keywords: RISC-V, open ISA, RISC, royalty-free, open standard, extensions, base integer, embedded, RV32, RV64
aka: [RISC-V, RISCV]
autolink: true
infobox:
  - { label: Type, value: ISA (RISC) }
  - { label: Licensing, value: Open, royalty-free }
  - { label: Steward, value: RISC-V International }
  - { label: Design, value: Small base + extensions }
  - { label: Base widths, value: "RV32, RV64, RV128" }
see_also: [instruction-set-architecture, arm-architecture, x86, central-processing-unit, microcontroller, semiconductor]
cite_urls:
  - https://en.wikipedia.org/wiki/RISC-V
---

**RISC-V** is an open, royalty-free RISC [instruction set architecture](/reference/instruction-set-architecture/) that anyone can implement without a license fee, built as a small mandatory base plus optional extensions.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="RISC-V is built as a small mandatory base integer instruction set at the center, surrounded by optional extensions a designer can add: M for multiply and divide, A for atomics, F and D for floating point, C for compressed instructions, and V for vectors." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <circle cx="140" cy="80" r="42" fill="currentColor" fill-opacity="0.14"/>
    <rect x="270" y="18" width="60" height="24" rx="4"/>
    <rect x="340" y="18" width="60" height="24" rx="4"/>
    <rect x="270" y="54" width="60" height="24" rx="4"/>
    <rect x="340" y="54" width="60" height="24" rx="4"/>
    <rect x="270" y="90" width="60" height="24" rx="4"/>
    <rect x="340" y="90" width="60" height="24" rx="4"/>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none" stroke-opacity="0.6">
    <line x1="182" y1="70" x2="270" y2="30"/>
    <line x1="182" y1="72" x2="340" y2="30"/>
    <line x1="182" y1="80" x2="270" y2="66"/>
    <line x1="182" y1="80" x2="340" y2="66"/>
    <line x1="182" y1="90" x2="270" y2="102"/>
    <line x1="182" y1="92" x2="340" y2="102"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="140" y="76" font-size="9" font-weight="600">Base</text>
    <text x="140" y="88" font-size="8">integer (I)</text>
    <text x="140" y="99" font-size="7" fill-opacity="0.85">mandatory</text>
    <text x="300" y="34" font-size="8">M · mul/div</text>
    <text x="370" y="34" font-size="8">A · atomics</text>
    <text x="300" y="70" font-size="8">F · float</text>
    <text x="370" y="70" font-size="8">D · double</text>
    <text x="300" y="106" font-size="8">C · compressed</text>
    <text x="370" y="106" font-size="8">V · vectors</text>
    <text x="230" y="145" font-size="8" fill-opacity="0.9">pick only the extensions a chip needs</text>
  </g>
</svg>
<figcaption>Every RISC-V chip starts from the same small mandatory base integer instruction set, then adds only the optional extensions it needs — multiply, atomics, floating point, compressed encodings, vectors — so a tiny microcontroller and a big application core share one open foundation.</figcaption>
</figure>

## Overview

Where [x86](/reference/x86/) and [ARM](/reference/arm-architecture/) are owned and licensed, RISC-V is a free and open standard stewarded by the non-profit RISC-V International. Anyone — a student, a startup, or a large vendor — can implement it, extend it, and ship silicon without paying royalties or signing an architecture licence.

Its defining feature is *modularity*. A small mandatory **base** integer instruction set (in 32-, 64-, or 128-bit register widths, named RV32/RV64/RV128) does the essentials; everything else lives in optional **extensions** that a designer bolts on only when needed. A minimal embedded core might implement just the base, while an application processor adds multiply, floating point, and vector extensions.

## The base-plus-extensions model

Standard extension letters name the optional pieces, and a chip's supported set is written as a string like "RV64GC":

| Letter | Adds | Typical need |
|--------|------|--------------|
| I | Base integer ops | Always (mandatory) |
| M | Integer multiply/divide | General compute |
| A | Atomic operations | Multi-core, OS support |
| F / D | Single-/double-precision float | Math-heavy workloads |
| C | Compressed 16-bit encodings | Smaller code, embedded |
| V | Vector operations | DSP, ML, signal work |

The common shorthand "G" bundles the I, M, A, F, and D set as a general-purpose baseline. This pick-what-you-need design keeps the smallest [microcontrollers](/reference/microcontroller/) lean while letting the same ISA scale up to full application cores.

## Where it fits

RISC-V's appeal is freedom from licensing and the ability to customize the ISA, which has made it popular in research, education, and embedded designs, with broader commercial adoption growing steadily. It competes with ARM in efficiency-focused niches and offers an open alternative to both incumbents. For a toolchain like GopherTrunk's, supporting a new ISA is mostly a matter of the [compiler](/reference/compiler/) gaining a target — and the open, extensible nature of RISC-V, including its vector extension for signal work, lowers the barrier to that happening.

## Sources

[^wiki]: [RISC-V](https://en.wikipedia.org/wiki/RISC-V) — Wikipedia, on the open RISC-V ISA, its base-plus-extensions model, and RISC-V International.
