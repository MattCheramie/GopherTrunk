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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 236" role="img" aria-label="Left: an FPGA fabric — a grid of logic blocks threaded by horizontal and vertical routing channels that meet at switch boxes, with one bold configured path snaking between blocks. Right: the inside of one logic block, a lookup table feeding a flip-flop. Loading a bitstream sets every lookup table and switch, so the chip becomes the circuit the designer described." xmlns="http://www.w3.org/2000/svg">
  <text x="118" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Reconfigurable fabric</text>
  <g stroke="currentColor" stroke-width="0.8" stroke-opacity="0.28">
    <line x1="24" y1="34" x2="24" y2="210"/><line x1="70" y1="34" x2="70" y2="210"/><line x1="116" y1="34" x2="116" y2="210"/><line x1="162" y1="34" x2="162" y2="210"/><line x1="208" y1="34" x2="208" y2="210"/>
    <line x1="24" y1="34" x2="208" y2="34"/><line x1="24" y1="80" x2="208" y2="80"/><line x1="24" y1="126" x2="208" y2="126"/><line x1="24" y1="172" x2="208" y2="172"/><line x1="24" y1="210" x2="208" y2="210"/>
  </g>
  <g fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1">
    <rect x="35" y="45" width="24" height="24" rx="2"/><rect x="81" y="45" width="24" height="24" rx="2"/><rect x="127" y="45" width="24" height="24" rx="2"/><rect x="173" y="45" width="24" height="24" rx="2"/>
    <rect x="35" y="91" width="24" height="24" rx="2"/><rect x="81" y="91" width="24" height="24" rx="2"/><rect x="127" y="91" width="24" height="24" rx="2"/><rect x="173" y="91" width="24" height="24" rx="2"/>
    <rect x="35" y="137" width="24" height="24" rx="2"/><rect x="81" y="137" width="24" height="24" rx="2"/><rect x="127" y="137" width="24" height="24" rx="2"/><rect x="173" y="137" width="24" height="24" rx="2"/>
  </g>
  <g fill="currentColor" fill-opacity="0.55" stroke="currentColor" stroke-width="0.6">
    <rect x="66" y="76" width="8" height="8"/><rect x="158" y="76" width="8" height="8"/><rect x="112" y="122" width="8" height="8"/>
  </g>
  <path d="M47 69 L70 80 L116 80 L116 126 L162 126 L173 149" fill="none" stroke="currentColor" stroke-width="2" stroke-opacity="0.9"/>
  <text x="116" y="228" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">logic blocks + routing channels + switch boxes</text>
  <rect x="256" y="66" width="172" height="96" rx="5" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/>
  <text x="342" y="60" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">one logic block</text>
  <line x1="230" y1="112" x2="252" y2="112" stroke="currentColor" stroke-width="1" stroke-opacity="0.7" marker-end="url(#fpga_ar)"/>
  <rect x="270" y="92" width="60" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="300" y="110" text-anchor="middle" font-size="8.5" fill="currentColor">LUT</text>
  <text x="300" y="122" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">truth table</text>
  <rect x="356" y="92" width="60" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="386" y="110" text-anchor="middle" font-size="8" fill="currentColor">flip-flop</text>
  <text x="386" y="122" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">register</text>
  <line x1="330" y1="112" x2="356" y2="112" stroke="currentColor" stroke-width="1.1" marker-end="url(#fpga_ar)"/>
  <text x="342" y="150" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">any Boolean function, then clocked</text>
  <defs><marker id="fpga_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An FPGA is a grid of small logic blocks — each a lookup table plus a flip-flop — linked by routing channels that meet at configurable switch boxes. A bitstream programs every lookup table and every switch, wiring a bold path like the one shown, so instead of running instructions the chip physically becomes the circuit you designed.</figcaption>
</figure>

## Overview

An FPGA is a fabric of programmable logic blocks — lookup tables (LUTs) and flip-flops — wired together by a configurable interconnect, plus dedicated blocks for arithmetic (DSP slices) and memory. A designer writes the circuit in a hardware description language (VHDL or Verilog); a toolchain *synthesizes* and *places-and-routes* it into a bitstream that configures the chip. Unlike a CPU running instructions, an FPGA *becomes* the circuit, executing many operations truly in parallel and on every clock cycle.

## Where it fits

FPGAs sit between general-purpose processors and fixed-function [ASICs](/reference/application-specific-integrated-circuit/): more flexible than an ASIC (you can re-flash the logic) but lower performance-per-watt and higher unit cost, making them ideal for prototyping, low-to-medium volume products, and latency-critical [hardware acceleration](/reference/hardware-acceleration/). They are a natural fit for [software-defined radio](/reference/software-defined-radio/): an FPGA in the radio front end can do the high-rate DSP — [digital filtering](/reference/digital-filter/), decimation, channelization, and [FFTs](/reference/fast-fourier-transform/) — at sample rates a CPU could never keep up with, handing GopherTrunk a stream that is already down-converted to manageable channels.

## Sources

[^wiki]: [Field-programmable gate array](https://en.wikipedia.org/wiki/Field-programmable_gate_array) — Wikipedia, on reconfigurable logic chips and their uses.
