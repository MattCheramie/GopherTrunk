---
slug: application-specific-integrated-circuit
title: Application-specific integrated circuit (ASIC)
entry_type: hardware
category: hw-accelerators
description: An application-specific integrated circuit (ASIC) is a chip custom-designed for one fixed task, trading flexibility for the best possible speed, power, and cost at high volume.
keywords: ASIC, application-specific integrated circuit, custom chip, fixed-function, fabrication, mask, tape-out, hardware acceleration
aka: [ASIC]
autolink: true
infobox:
  - { label: Type, value: Fixed-function custom chip }
  - { label: Designed for, value: One specific application }
  - { label: Strength, value: Best speed / power / unit cost }
  - { label: Weakness, value: "Not reprogrammable; high NRE" }
  - { label: Built by, value: "Foundry (e.g. TSMC)" }
see_also: [field-programmable-gate-array, integrated-circuit, tensor-processing-unit, hardware-acceleration, soc-vs-discrete, semiconductor]
cite_urls:
  - https://en.wikipedia.org/wiki/Application-specific_integrated_circuit
---

An **application-specific integrated circuit** (**ASIC**) is an [integrated circuit](/reference/integrated-circuit/) custom-designed for a single, fixed task — trading away general-purpose flexibility for the best achievable speed, power efficiency, and per-unit cost.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="A spectrum from CPU to GPU to FPGA to ASIC. Moving right, flexibility and reprogrammability fall while efficiency, speed per watt, and cost-at-scale rise. A CPU runs any task; an ASIC does one fixed job." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="42" x2="360" y2="42" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#as_ar)"/>
  <text x="200" y="32" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">more flexible · reprogrammable</text>
  <line x1="60" y1="90" x2="60" y2="90" stroke="currentColor"/>
  <g text-anchor="middle" fill="currentColor" font-size="10">
    <circle cx="70" cy="90" r="7" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="70" y="112" font-weight="600" font-size="9.5">CPU</text><text x="70" y="124" font-size="7.5" fill-opacity="0.85">any task</text>
    <circle cx="180" cy="90" r="7" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="180" y="112" font-weight="600" font-size="9.5">GPU</text><text x="180" y="124" font-size="7.5" fill-opacity="0.85">parallel</text>
    <circle cx="290" cy="90" r="7" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="290" y="112" font-weight="600" font-size="9.5">FPGA</text><text x="290" y="124" font-size="7.5" fill-opacity="0.85">reconfigurable</text>
    <circle cx="400" cy="90" r="7" fill="currentColor" fill-opacity="0.35" stroke="currentColor" stroke-width="1.4"/><text x="400" y="112" font-weight="600" font-size="9.5">ASIC</text><text x="400" y="124" font-size="7.5" fill-opacity="0.85">one fixed task</text>
  </g>
  <line x1="63" y1="90" x2="177" y2="90" stroke="currentColor" stroke-width="1" stroke-opacity="0.4"/>
  <line x1="187" y1="90" x2="283" y2="90" stroke="currentColor" stroke-width="1" stroke-opacity="0.4"/>
  <line x1="297" y1="90" x2="393" y2="90" stroke="currentColor" stroke-width="1" stroke-opacity="0.4"/>
  <line x1="420" y1="138" x2="100" y2="138" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#as_ar)"/>
  <text x="260" y="152" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">more efficient · faster · cheaper at scale · higher up-front (NRE) cost</text>
  <defs><marker id="as_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The classic hardware spectrum: from a fully programmable CPU to a permanently etched ASIC, flexibility falls and efficiency rises at each step. An ASIC wins decisively on performance-per-watt and cost at volume but can't be changed and is slow to build — so an FPGA is the hedge when volumes are low or the design might still change.</figcaption>
</figure>

## Overview

Where a CPU or [FPGA](/reference/field-programmable-gate-array/) is built to run many possible workloads, an ASIC's circuit is etched permanently into silicon to do exactly one job — a network switch chip, a Bitcoin miner, a phone's modem, a [TPU](/reference/tensor-processing-unit/). The design is fabricated at a foundry such as [TSMC](/reference/tsmc/) from a set of photomasks; this *non-recurring engineering* cost is large and a finished ASIC cannot be changed, so the economics only work at high volume or where nothing else meets the performance target.

## Trade-offs

The classic spectrum runs CPU → GPU → FPGA → ASIC, with flexibility falling and efficiency rising at each step. An ASIC wins decisively on performance-per-watt and cost-at-scale but is inflexible and slow to bring to market; an FPGA is the hedge when volumes are low or the design may still change. Many specialized [hardware accelerators](/reference/hardware-acceleration/) are ASICs. For a project like GopherTrunk, ASICs already appear *inside* the radio — the [RTL-SDR](/reference/rtl-sdr/)'s tuner and demodulator are fixed-function chips — even though the decoding itself is done in software.

## Sources

[^wiki]: [Application-specific integrated circuit](https://en.wikipedia.org/wiki/Application-specific_integrated_circuit) — Wikipedia, on custom fixed-function chips and their trade-offs.
