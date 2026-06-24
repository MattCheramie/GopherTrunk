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

## Overview

Where a CPU or [FPGA](/reference/field-programmable-gate-array/) is built to run many possible workloads, an ASIC's circuit is etched permanently into silicon to do exactly one job — a network switch chip, a Bitcoin miner, a phone's modem, a [TPU](/reference/tensor-processing-unit/). The design is fabricated at a foundry such as [TSMC](/reference/tsmc/) from a set of photomasks; this *non-recurring engineering* cost is large and a finished ASIC cannot be changed, so the economics only work at high volume or where nothing else meets the performance target.

## Trade-offs

The classic spectrum runs CPU → GPU → FPGA → ASIC, with flexibility falling and efficiency rising at each step. An ASIC wins decisively on performance-per-watt and cost-at-scale but is inflexible and slow to bring to market; an FPGA is the hedge when volumes are low or the design may still change. Many specialized [hardware accelerators](/reference/hardware-acceleration/) are ASICs. For a project like GopherTrunk, ASICs already appear *inside* the radio — the [RTL-SDR](/reference/rtl-sdr/)'s tuner and demodulator are fixed-function chips — even though the decoding itself is done in software.

## Sources

[^wiki]: [Application-specific integrated circuit](https://en.wikipedia.org/wiki/Application-specific_integrated_circuit) — Wikipedia, on custom fixed-function chips and their trade-offs.
