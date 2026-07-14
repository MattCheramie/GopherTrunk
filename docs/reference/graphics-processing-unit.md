---
slug: graphics-processing-unit
title: Graphics processing unit (GPU)
entry_type: hardware
category: hw-foundations
description: A GPU is a processor with thousands of small cores optimized for the parallel math behind graphics, and now widely used for general-purpose compute such as AI and DSP.
keywords: GPU, graphics processing unit, parallel computing, CUDA, shaders, GPGPU, AI accelerator
aka: [GPU, Graphics card]
infobox:
  - { label: Type, value: Parallel processor }
  - { label: Cores, value: Hundreds to thousands }
  - { label: Good at, value: Parallel math (SIMD) }
  - { label: Uses, value: Graphics, AI, DSP, simulation }
see_also: [central-processing-unit, integrated-circuit, pci-express, instruction-set-architecture, moores-law, system-bus]
cite_urls:
  - https://en.wikipedia.org/wiki/Graphics_processing_unit
---

A **graphics processing unit** (**GPU**) is a processor built from many small cores that run the same operation across large blocks of data in parallel — originally for rendering graphics, now also for general computation.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="A comparison of two processors. On the left, a CPU has four large cores tuned for sequential work. On the right, a GPU has a dense grid of many small cores that apply the same operation across a wide block of data at once." xmlns="http://www.w3.org/2000/svg">
  <text x="110" y="16" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">CPU: a few powerful cores</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="22" y="30" width="85" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="64" y="57">core</text>
    <rect x="115" y="30" width="85" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="157" y="57">core</text>
    <rect x="22" y="82" width="85" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="64" y="109">core</text>
    <rect x="115" y="82" width="85" height="46" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="157" y="109">core</text>
  </g>
  <text x="111" y="150" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.8">sequential, branchy work</text>
  <line x1="230" y1="24" x2="230" y2="134" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.35" stroke-dasharray="3 3"/>
  <text x="350" y="16" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">GPU: hundreds of small cores</text>
  <g fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="0.8">
    <rect x="252" y="30" width="22" height="18" rx="2"/><rect x="278" y="30" width="22" height="18" rx="2"/><rect x="304" y="30" width="22" height="18" rx="2"/><rect x="330" y="30" width="22" height="18" rx="2"/><rect x="356" y="30" width="22" height="18" rx="2"/><rect x="382" y="30" width="22" height="18" rx="2"/><rect x="408" y="30" width="22" height="18" rx="2"/>
    <rect x="252" y="52" width="22" height="18" rx="2"/><rect x="278" y="52" width="22" height="18" rx="2"/><rect x="304" y="52" width="22" height="18" rx="2"/><rect x="330" y="52" width="22" height="18" rx="2"/><rect x="356" y="52" width="22" height="18" rx="2"/><rect x="382" y="52" width="22" height="18" rx="2"/><rect x="408" y="52" width="22" height="18" rx="2"/>
    <rect x="252" y="74" width="22" height="18" rx="2"/><rect x="278" y="74" width="22" height="18" rx="2"/><rect x="304" y="74" width="22" height="18" rx="2"/><rect x="330" y="74" width="22" height="18" rx="2"/><rect x="356" y="74" width="22" height="18" rx="2"/><rect x="382" y="74" width="22" height="18" rx="2"/><rect x="408" y="74" width="22" height="18" rx="2"/>
    <rect x="252" y="96" width="22" height="18" rx="2"/><rect x="278" y="96" width="22" height="18" rx="2"/><rect x="304" y="96" width="22" height="18" rx="2"/><rect x="330" y="96" width="22" height="18" rx="2"/><rect x="356" y="96" width="22" height="18" rx="2"/><rect x="382" y="96" width="22" height="18" rx="2"/><rect x="408" y="96" width="22" height="18" rx="2"/>
  </g>
  <text x="341" y="150" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.8">same op across all lanes (SIMD)</text>
</svg>
<figcaption>A CPU spends its silicon on a few powerful cores for sequential, branchy work; a GPU spends it on hundreds of simple cores that apply one operation across a wide block of data at once. That data-parallel shape is what suits graphics, AI, and batched DSP.</figcaption>
</figure>

## Overview

Where a [CPU](/reference/central-processing-unit/) has a few powerful cores tuned for sequential work, a GPU has hundreds or thousands of simpler cores that excel at *data-parallel* math: the same calculation applied to many pixels, vertices, or array elements at once. A discrete GPU is a card that plugs into a [PCIe](/reference/pci-express/) slot with its own high-bandwidth memory; integrated GPUs share the CPU's memory. Modern GPUs are large [integrated circuits](/reference/integrated-circuit/) whose transistor counts have ridden [Moore's law](/reference/moores-law/) upward for decades.

## What it's for

Beyond 3D graphics, GPUs power AI training and inference, scientific simulation, and signal processing — any workload that maps onto wide parallel arithmetic. In SDR work a GPU can accelerate FFTs and filtering across many channels at once, complementing the streaming DSP GopherTrunk runs on the CPU. The trade-off is latency and complexity: GPUs shine on big batched workloads, less so on small, branchy, latency-sensitive tasks where the CPU wins.

## Sources

[^wiki]: [Graphics processing unit](https://en.wikipedia.org/wiki/Graphics_processing_unit) — Wikipedia, on GPUs and parallel computation.
