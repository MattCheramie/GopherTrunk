---
slug: field-programmable-gate-array
title: Field-programmable gate array (FPGA)
entry_type: hardware
category: hw-accelerators
description: A field-programmable gate array (FPGA) is a chip whose digital logic can be reconfigured after manufacture, letting designers build custom hardware circuits in software for DSP, prototyping, and acceleration.
keywords: FPGA, field-programmable gate array, reconfigurable logic, HDL, VHDL, Verilog, lookup table, DSP, hardware acceleration
aka: [FPGA]
autolink: true
infobox:
  - { label: Type, value: Reconfigurable logic chip }
  - { label: Configured with, value: "HDL (VHDL / Verilog)" }
  - { label: Built from, value: "LUTs, flip-flops, DSP & RAM blocks" }
  - { label: Strength, value: Custom parallel digital hardware }
  - { label: Vendors, value: "AMD (Xilinx), Intel (Altera), Lattice" }
see_also: [application-specific-integrated-circuit, hardware-acceleration, software-defined-radio, integrated-circuit, digital-filter, fast-fourier-transform]
cite_urls:
  - https://en.wikipedia.org/wiki/Field-programmable_gate_array
---

A **field-programmable gate array** (**FPGA**) is an [integrated circuit](/reference/integrated-circuit/) whose internal digital logic can be reconfigured *after* manufacture — letting an engineer describe a custom hardware circuit in software and load it onto the chip.[^wiki]

## Overview

An FPGA is a fabric of programmable logic blocks — lookup tables (LUTs) and flip-flops — wired together by a configurable interconnect, plus dedicated blocks for arithmetic (DSP slices) and memory. A designer writes the circuit in a hardware description language (VHDL or Verilog); a toolchain *synthesizes* and *places-and-routes* it into a bitstream that configures the chip. Unlike a CPU running instructions, an FPGA *becomes* the circuit, executing many operations truly in parallel and on every clock cycle.

## Where it fits

FPGAs sit between general-purpose processors and fixed-function [ASICs](/reference/application-specific-integrated-circuit/): more flexible than an ASIC (you can re-flash the logic) but lower performance-per-watt and higher unit cost, making them ideal for prototyping, low-to-medium volume products, and latency-critical [hardware acceleration](/reference/hardware-acceleration/). They are a natural fit for [software-defined radio](/reference/software-defined-radio/): an FPGA in the radio front end can do the high-rate DSP — [digital filtering](/reference/digital-filter/), decimation, channelization, and [FFTs](/reference/fast-fourier-transform/) — at sample rates a CPU could never keep up with, handing GopherTrunk a stream that is already down-converted to manageable channels.

## Sources

[^wiki]: [Field-programmable gate array](https://en.wikipedia.org/wiki/Field-programmable_gate_array) — Wikipedia, on reconfigurable logic chips and their uses.
