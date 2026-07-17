---
slug: instruction-set-architecture
title: Instruction set architecture (ISA)
entry_type: concept
category: hw-foundations
description: An instruction set architecture is the contract between hardware and software, defining the instructions, registers, and data types a processor understands, independent of how it is built.
keywords: instruction set architecture, ISA, machine code, registers, RISC, CISC, opcode, microarchitecture, fetch decode execute
aka: [ISA]
infobox:
  - { label: Type, value: Hardware/software contract }
  - { label: Defines, value: Instructions, registers, data types }
  - { label: Separate from, value: Microarchitecture }
  - { label: Styles, value: "RISC, CISC" }
  - { label: Examples, value: "x86, ARM, RISC-V" }
see_also: [central-processing-unit, x86, arm-architecture, risc-v, von-neumann-architecture, compiler]
cite_urls:
  - https://en.wikipedia.org/wiki/Instruction_set_architecture
---

An **instruction set architecture** (**ISA**) is the contract between hardware and software: the set of instructions, registers, and data types a [processor](/reference/central-processing-unit/) understands, separate from how any particular chip implements it.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="The fetch-decode-execute cycle. The processor fetches the next instruction from memory, decodes it into a defined operation, executes it using registers and the arithmetic unit, then writes back the result and repeats. The ISA defines the instructions and registers this cycle uses." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="16" y="52" width="88" height="44" rx="4"/>
    <rect x="132" y="52" width="88" height="44" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="248" y="52" width="88" height="44" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="364" y="52" width="88" height="44" rx="4"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <path d="M104 74 h28 m-6 -4 l6 4 l-6 4"/>
    <path d="M220 74 h28 m-6 -4 l6 4 l-6 4"/>
    <path d="M336 74 h28 m-6 -4 l6 4 l-6 4"/>
    <path d="M408 96 v18 h-350 v-18 m-4 6 l4 -6 l4 6"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="60" y="72" font-size="9" font-weight="600">Fetch</text>
    <text x="60" y="85" font-size="7.5" fill-opacity="0.85">read instruction</text>
    <text x="176" y="72" font-size="9" font-weight="600">Decode</text>
    <text x="176" y="85" font-size="7.5" fill-opacity="0.85">read opcode</text>
    <text x="292" y="72" font-size="9" font-weight="600">Execute</text>
    <text x="292" y="85" font-size="7.5" fill-opacity="0.85">registers + ALU</text>
    <text x="408" y="72" font-size="9" font-weight="600">Write back</text>
    <text x="408" y="85" font-size="7.5" fill-opacity="0.85">store result</text>
    <text x="230" y="132" font-size="8" fill-opacity="0.9">ISA defines the opcodes and registers · microarchitecture decides how the cycle is built</text>
  </g>
</svg>
<figcaption>The processor repeats a fetch-decode-execute-write-back cycle; the ISA is the fixed vocabulary of opcodes and registers that this cycle operates on, while the microarchitecture is the separate engineering of how fast and how cleverly the cycle runs.</figcaption>
</figure>

## Overview

The ISA defines what a program *can ask the CPU to do* — the opcodes, the registers, how memory is addressed, what data types exist — so that machine code written for an ISA runs on any chip that implements it. It is a stable interface: a binary compiled for x86-64 in 2010 still runs on an x86-64 chip made today, because both honour the same contract.

The *microarchitecture* is the separate question of how a given chip carries those instructions out — its pipeline depth, caches, branch predictors, and execution units. Two chips can share an ISA yet differ wildly in speed and power because their microarchitectures differ. This split is what lets Intel and AMD both build x86 chips, or dozens of vendors all build ARM chips, that run the same software.

## RISC versus CISC

ISAs split loosely into two philosophies, though modern designs blur the line internally:

| Trait | RISC | CISC |
|-------|------|------|
| Instruction count | Few, simple | Many, complex |
| Instruction length | Fixed | Variable |
| Work per instruction | Small, uniform | Can be large |
| Memory access | Load/store only | Many ops touch memory |
| Examples | ARM, RISC-V | x86 |

The major families today are the CISC [x86](/reference/x86/) and the RISC [ARM](/reference/arm-architecture/) and open [RISC-V](/reference/risc-v/). In practice even x86 chips decode their complex instructions into simple RISC-like internal operations, so the distinction is now more about the external contract than the silicon.

## Where it fits

The ISA is why a [compiler](/reference/compiler/) must *target* a specific architecture, and why a binary built for x86 will not run on an ARM chip without recompilation or emulation. It builds on the [von Neumann](/reference/von-neumann-architecture/) idea of a stored program the processor fetches and executes. For GopherTrunk this is a daily reality: Go cross-compiles the same source to x86 servers and to the ARM CPU in a [Raspberry Pi](/reference/raspberry-pi/) capture node, each producing native machine code for that ISA.

## Sources

[^wiki]: [Instruction set architecture](https://en.wikipedia.org/wiki/Instruction_set_architecture) — Wikipedia, on the ISA as the hardware/software interface and its separation from microarchitecture.
