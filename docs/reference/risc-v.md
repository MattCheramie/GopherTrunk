---
slug: risc-v
title: RISC-V
entry_type: concept
category: hw-foundations
description: RISC-V is an open, royalty-free RISC instruction set architecture that anyone can implement freely, used in research, embedded systems, and a growing range of processors.
keywords: RISC-V, open ISA, RISC, royalty-free, open standard, extensions, embedded
aka: [RISC-V, RISCV]
autolink: true
infobox:
  - { label: Type, value: ISA (RISC) }
  - { label: Licensing, value: Open, royalty-free }
  - { label: Steward, value: RISC-V International }
  - { label: Design, value: Small base + extensions }
see_also: [instruction-set-architecture, arm-architecture, x86, central-processing-unit, microcontroller, semiconductor]
cite_urls:
  - https://en.wikipedia.org/wiki/RISC-V
---

**RISC-V** is an open, royalty-free RISC [instruction set architecture](/reference/instruction-set-architecture/) that anyone can implement without a license fee.[^wiki]

## Overview

Where [x86](/reference/x86/) and [ARM](/reference/arm-architecture/) are owned and licensed, RISC-V is a free and open standard stewarded by RISC-V International. It is built as a small mandatory *base* integer instruction set plus optional *extensions* (for multiplication, floating point, vectors, and more), so a designer picks only what a chip needs. This modular, open model lets universities, startups, and large vendors build compatible processors from tiny [microcontrollers](/reference/microcontroller/) up to application cores without royalties.

## Where it fits

RISC-V's appeal is freedom from licensing and the ability to customize the ISA, which has made it popular in research, education, and embedded designs, with broader adoption growing. It competes with ARM in efficiency-focused niches and offers an open alternative to both incumbents. For a toolchain like GopherTrunk's, supporting a new ISA is mostly a matter of the [compiler](/reference/compiler/) gaining a target — the open ISA lowers the barrier to that happening.

## Sources

[^wiki]: [RISC-V](https://en.wikipedia.org/wiki/RISC-V) — Wikipedia, on the open RISC-V ISA and its extension model.
