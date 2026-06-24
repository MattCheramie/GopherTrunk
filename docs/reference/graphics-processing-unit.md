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

## Overview

Where a [CPU](/reference/central-processing-unit/) has a few powerful cores tuned for sequential work, a GPU has hundreds or thousands of simpler cores that excel at *data-parallel* math: the same calculation applied to many pixels, vertices, or array elements at once. A discrete GPU is a card that plugs into a [PCIe](/reference/pci-express/) slot with its own high-bandwidth memory; integrated GPUs share the CPU's memory. Modern GPUs are large [integrated circuits](/reference/integrated-circuit/) whose transistor counts have ridden [Moore's law](/reference/moores-law/) upward for decades.

## What it's for

Beyond 3D graphics, GPUs power AI training and inference, scientific simulation, and signal processing — any workload that maps onto wide parallel arithmetic. In SDR work a GPU can accelerate FFTs and filtering across many channels at once, complementing the streaming DSP GopherTrunk runs on the CPU. The trade-off is latency and complexity: GPUs shine on big batched workloads, less so on small, branchy, latency-sensitive tasks where the CPU wins.

## Sources

[^wiki]: [Graphics processing unit](https://en.wikipedia.org/wiki/Graphics_processing_unit) — Wikipedia, on GPUs and parallel computation.
