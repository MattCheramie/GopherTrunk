---
slug: amd
title: AMD
entry_type: organization
category: hw-organizations
description: AMD (Advanced Micro Devices) is an American semiconductor company that designs x86 CPUs and, through its Radeon line, GPUs, competing directly with Intel and NVIDIA.
keywords: AMD, Advanced Micro Devices, Ryzen, EPYC, Radeon, x86, CPU, GPU, fabless, ATI, Xilinx
aka: [Advanced Micro Devices]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor company }
  - { label: Founded, value: "1969" }
  - { label: HQ, value: Santa Clara, California, USA }
  - { label: Makes, value: x86 CPUs, GPUs }
see_also: [central-processing-unit, graphics-processing-unit, intel, x86, tsmc, semiconductor]
cite_urls:
  - https://www.amd.com/
  - https://en.wikipedia.org/wiki/AMD
---

**AMD** (Advanced Micro Devices) is an American semiconductor company founded in 1969 that
designs [x86](/reference/x86/) [CPUs](/reference/central-processing-unit/) and, through its
Radeon brand, [GPUs](/reference/graphics-processing-unit/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A timeline of AMD from its 1969 founding as an Intel-compatible second source, through the 2006 acquisition of the graphics company ATI, the 2017 launch of the Zen-based Ryzen and EPYC processors, to the 2022 acquisition of the FPGA maker Xilinx." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="64" x2="440" y2="64" stroke="currentColor" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.2">
    <circle cx="60" cy="64" r="5" fill-opacity="0.15"/>
    <circle cx="185" cy="64" r="5" fill-opacity="0.15"/>
    <circle cx="310" cy="64" r="6" fill="currentColor"/>
    <circle cx="410" cy="64" r="5" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="60" y="48" font-size="9" font-weight="600">1969</text>
    <text x="60" y="84" font-size="8">founded</text>
    <text x="60" y="95" font-size="8">Intel 2nd source</text>
    <text x="185" y="48" font-size="9" font-weight="600">2006</text>
    <text x="185" y="84" font-size="8">buys ATI</text>
    <text x="185" y="95" font-size="8">(Radeon GPUs)</text>
    <text x="310" y="48" font-size="9" font-weight="600">2017</text>
    <text x="310" y="84" font-size="8">Zen: Ryzen</text>
    <text x="310" y="95" font-size="8">&amp; EPYC</text>
    <text x="410" y="48" font-size="9" font-weight="600">2022</text>
    <text x="410" y="84" font-size="8">buys Xilinx</text>
    <text x="410" y="95" font-size="8">(FPGAs)</text>
  </g>
</svg>
<figcaption>AMD grew from a 1969 Intel-compatible second source into a full CPU and GPU competitor, absorbing ATI's graphics in 2006, retaking the performance lead with the Zen-based Ryzen and EPYC in 2017, and adding Xilinx's programmable logic in 2022.</figcaption>
</figure>

## Overview

AMD began as a second source for chips compatible with Intel's designs and grew into a
full competitor. Its modern Ryzen processors target desktops and laptops, while EPYC
serves data centers and servers; its Radeon line competes in graphics. AMD acquired the
graphics company ATI in 2006 and the FPGA maker Xilinx in 2022, broadening its reach into
accelerators and programmable logic.[^home]

Unlike a traditional integrated manufacturer, AMD is "fabless" — it designs chips but
contracts their fabrication to foundries such as [TSMC](/reference/tsmc/), letting it focus
investment on design rather than building plants.

## Product lines

AMD's catalogue splits cleanly into a few families, each aimed at a different market:

| Product line | What it is |
|--------------|------------|
| Ryzen | x86 CPUs for desktops and laptops |
| EPYC | Many-core x86 CPUs for servers and data centers |
| Radeon | GPUs for graphics and compute |
| Instinct | Data-center GPU accelerators for AI/HPC |
| Versal / Xilinx | FPGAs and adaptive SoCs (from the 2022 acquisition) |

## Where it fits

AMD keeps the x86 market competitive with [Intel](/reference/intel/), and its EPYC server
chips and Radeon GPUs are common in workstations and data centers. For compute-heavy SDR
work — running many decode channels or DSP on a single host — a high-core-count AMD CPU is
a frequent choice, and the Xilinx FPGAs it now owns are the same class of programmable logic
found inside many software-defined radios' front ends.

## Sources

[^home]: [AMD](https://www.amd.com/) — the company's official site, for product lines.
[^wiki]: [AMD](https://en.wikipedia.org/wiki/AMD) — Wikipedia, for the company's history and role.
