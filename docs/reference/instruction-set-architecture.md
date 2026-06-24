---
slug: instruction-set-architecture
title: Instruction set architecture (ISA)
entry_type: concept
category: hw-foundations
description: An instruction set architecture is the contract between hardware and software, defining the instructions, registers, and data types a processor understands, independent of how it is built.
keywords: instruction set architecture, ISA, machine code, registers, RISC, CISC, opcode, microarchitecture
aka: [ISA]
infobox:
  - { label: Type, value: Hardware/software contract }
  - { label: Defines, value: Instructions, registers }
  - { label: Styles, value: "RISC, CISC" }
  - { label: Examples, value: "x86, ARM, RISC-V" }
see_also: [central-processing-unit, x86, arm-architecture, risc-v, von-neumann-architecture, compiler]
cite_urls:
  - https://en.wikipedia.org/wiki/Instruction_set_architecture
---

An **instruction set architecture** (**ISA**) is the contract between hardware and software: the set of instructions, registers, and data types a [processor](/reference/central-processing-unit/) understands, separate from how any particular chip implements it.[^wiki]

## Overview

The ISA defines what a program *can ask the CPU to do* — the opcodes, the registers, how memory is addressed — so that machine code written for an ISA runs on any chip that implements it. The *microarchitecture* is the separate question of how a given chip carries those instructions out (pipelines, caches, execution units). ISAs split loosely into **RISC** (few, simple, fixed-length instructions) and **CISC** (more, richer instructions); the major families today are [x86](/reference/x86/), [ARM](/reference/arm-architecture/), and the open [RISC-V](/reference/risc-v/).

## Where it fits

The ISA is why a [compiler](/reference/compiler/) must *target* a specific architecture, and why a binary built for x86 will not run on an ARM chip without recompilation or emulation. It builds on the [von Neumann](/reference/von-neumann-architecture/) idea of a stored program the processor fetches and executes. For GopherTrunk this is a daily reality: Go cross-compiles the same source to x86 servers and to the ARM CPU in a [Raspberry Pi](/reference/raspberry-pi/) capture node.

## Sources

[^wiki]: [Instruction set architecture](https://en.wikipedia.org/wiki/Instruction_set_architecture) — Wikipedia, on the ISA as the hardware/software interface.
