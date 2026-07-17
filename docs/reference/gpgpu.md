---
slug: gpgpu
title: GPGPU
entry_type: concept
category: hw-accelerators
description: GPGPU is the practice of using a graphics processing unit for general-purpose computation, exploiting its thousands of parallel cores for non-graphics workloads such as math, simulation, signal processing, and machine learning.
keywords: GPGPU, general-purpose GPU, GPU computing, parallel computing, CUDA, OpenCL, compute shader, data parallelism, SIMD
aka: [General-purpose computing on graphics processing units]
autolink: true
infobox:
  - { label: Type, value: Computing technique }
  - { label: Hardware, value: Graphics processing unit }
  - { label: Strength, value: Massively parallel data }
  - { label: Weakness, value: Branch-heavy sequential work }
  - { label: APIs, value: "CUDA, OpenCL, compute shaders" }
see_also: [graphics-processing-unit, cuda, hardware-acceleration, vector-processor, central-processing-unit, ai-accelerator]
cite_urls:
  - https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units
---

**GPGPU** (general-purpose computing on graphics processing units) is the practice of using a [GPU](/reference/graphics-processing-unit/) to run ordinary computation rather than only rendering images.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A CPU with a few large cores beside a GPU with a dense grid of many small cores. The CPU suits sequential, branch-heavy work while the GPU applies the same operation across a large grid of data elements in parallel." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <rect x="20" y="26" width="170" height="98" rx="5" fill-opacity="0.05" stroke-width="1.4"/>
    <text x="105" y="20" text-anchor="middle" font-size="9" stroke="none" font-weight="600">CPU</text>
    <rect x="34" y="40" width="66" height="34" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <rect x="110" y="40" width="66" height="34" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <rect x="34" y="82" width="66" height="34" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <rect x="110" y="82" width="66" height="34" rx="3" fill-opacity="0.14" stroke-width="1.2"/>
    <text x="105" y="138" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.9">a few large cores &#183; sequential, branchy</text>
  </g>
  <g stroke="currentColor" fill="currentColor">
    <rect x="270" y="26" width="170" height="98" rx="5" fill-opacity="0.05" stroke-width="1.4"/>
    <text x="355" y="20" text-anchor="middle" font-size="9" stroke="none" font-weight="600">GPU</text>
    <g stroke-width="0.8">
      <rect x="282" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="300" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="318" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="336" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="354" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="372" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="390" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="408" y="38" width="14" height="14" fill-opacity="0.30"/>
      <rect x="282" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="300" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="318" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="336" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="354" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="372" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="390" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="408" y="56" width="14" height="14" fill-opacity="0.30"/>
      <rect x="282" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="300" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="318" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="336" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="354" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="372" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="390" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="408" y="74" width="14" height="14" fill-opacity="0.30"/>
      <rect x="282" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="300" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="318" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="336" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="354" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="372" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="390" y="92" width="14" height="14" fill-opacity="0.30"/>
      <rect x="408" y="92" width="14" height="14" fill-opacity="0.30"/>
    </g>
    <text x="355" y="138" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.9">thousands of small cores &#183; data-parallel</text>
  </g>
</svg>
<figcaption>A CPU spends its silicon on a few large cores tuned for sequential, branch-heavy code; a GPU spends its silicon on a dense grid of small cores that apply the same operation to many data elements at once — the pattern GPGPU exploits.</figcaption>
</figure>

## Overview

A GPU contains thousands of small cores built to shade pixels in parallel. When a problem can be expressed as the *same operation applied to many data elements at once* — the same SIMD/data-parallel pattern a [vector processor](/reference/vector-processor/) exploits — those cores can be redirected at it. Early GPGPU work smuggled math through the graphics pipeline, dressing numbers up as textures and pixels; dedicated APIs like [CUDA](/reference/cuda/) and the cross-vendor OpenCL later exposed the hardware directly.

That direct access made GPGPU mainstream for scientific computing, cryptography, and machine learning, where a single GPU can replace a rack of CPUs on the right workload. The trade-off is architectural: a GPU commits its transistors to arithmetic throughput rather than to the branch prediction and large caches that make a [CPU](/reference/central-processing-unit/) fast at irregular, sequential code.

## What it's for

The dividing line is the shape of the work, not the difficulty of the math. A workload that does the same thing to every element of a long array is ideal; one full of data-dependent branches stalls the GPU's lock-step lanes.

| Suits the GPU | Suits the CPU |
|---------------|---------------|
| Same op over large arrays | Branchy, data-dependent logic |
| High arithmetic per byte | Latency-sensitive single tasks |
| Regular, predictable memory | Irregular pointer chasing |
| FFTs, matrix math, filtering | Control flow, I/O, coordination |

GPGPU is the foundation of modern deep learning, where it long preceded purpose-built [AI accelerators](/reference/ai-accelerator/).

## Where it fits

In a software-defined radio context, GPGPU can accelerate wideband DSP — large [FFTs](/reference/fast-fourier-transform/), polyphase [channelizers](/reference/channelizer/) splitting one capture into many channels, and batched [filtering](/reference/digital-filter/) across those channels. These are exactly the same-op-over-many-samples workloads the hardware wants. As with any [hardware acceleration](/reference/hardware-acceleration/), the cost of moving samples to and from the GPU only pays off when the channel count is high; GopherTrunk's real-time, often narrowband decoding runs comfortably on CPU vector units for the common case.

## Sources

[^wiki]: [General-purpose computing on graphics processing units](https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units) — Wikipedia, on using GPUs for non-graphics computation and the data-parallel model.
