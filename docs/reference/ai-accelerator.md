---
slug: ai-accelerator
title: AI accelerator
entry_type: hardware
category: hw-accelerators
description: An AI accelerator is specialized hardware designed to speed up machine-learning workloads, especially the matrix math of neural networks, far more efficiently than a general-purpose CPU.
keywords: AI accelerator, machine learning hardware, neural network chip, TPU, NPU, GPU, inference, training, deep learning
infobox:
  - { label: Type, value: Specialized compute hardware }
  - { label: Job, value: Neural-network training / inference }
  - { label: Forms, value: "GPU, TPU, NPU, FPGA, ASIC" }
  - { label: Optimized for, value: Parallel matrix math }
  - { label: Metric, value: Throughput per watt }
see_also: [tensor-processing-unit, neural-processing-unit, graphics-processing-unit, hardware-acceleration, application-specific-integrated-circuit, edge-ai]
cite_urls:
  - https://en.wikipedia.org/wiki/AI_accelerator
---

An **AI accelerator** is specialized hardware designed to run machine-learning workloads — above all the dense matrix multiplications of neural networks — far faster and more efficiently than a general-purpose [CPU](/reference/central-processing-unit/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 196" role="img" aria-label="A spectrum of four ways to run neural-network math, from general to specialized: a CPU with a couple of cores, a GPU with a wide grid, a TPU or NPU with a denser array, and a fixed-function ASIC. Moving right adds parallel multiply-accumulate lanes and throughput per watt but gives up general-purpose flexibility." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">One workload, general to specialized</text>
  <g text-anchor="middle" font-size="9" fill="currentColor">
    <rect x="30" y="30" width="76" height="34" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/><text x="68" y="46">CPU</text><text x="68" y="58" font-size="7" fill-opacity="0.85">general</text>
    <rect x="144" y="30" width="76" height="34" rx="4" fill="currentColor" fill-opacity="0.11" stroke="currentColor" stroke-width="1.1"/><text x="182" y="46">GPU</text><text x="182" y="58" font-size="7" fill-opacity="0.85">wide SIMD</text>
    <rect x="258" y="30" width="76" height="34" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="296" y="46">TPU · NPU</text><text x="296" y="58" font-size="7" fill-opacity="0.85">matrix array</text>
    <rect x="372" y="30" width="76" height="34" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.1"/><text x="410" y="46">ASIC</text><text x="410" y="58" font-size="7" fill-opacity="0.85">fixed-function</text>
  </g>
  <g fill="currentColor" fill-opacity="0.3" stroke="none">
    <rect x="59" y="76" width="8" height="8" rx="1"/><rect x="69" y="76" width="8" height="8" rx="1"/>
    <rect x="168" y="76" width="7" height="7" rx="1"/><rect x="177" y="76" width="7" height="7" rx="1"/><rect x="186" y="76" width="7" height="7" rx="1"/><rect x="168" y="85" width="7" height="7" rx="1"/><rect x="177" y="85" width="7" height="7" rx="1"/><rect x="186" y="85" width="7" height="7" rx="1"/>
    <rect x="278" y="76" width="7" height="7" rx="1"/><rect x="287" y="76" width="7" height="7" rx="1"/><rect x="296" y="76" width="7" height="7" rx="1"/><rect x="305" y="76" width="7" height="7" rx="1"/><rect x="278" y="85" width="7" height="7" rx="1"/><rect x="287" y="85" width="7" height="7" rx="1"/><rect x="296" y="85" width="7" height="7" rx="1"/><rect x="305" y="85" width="7" height="7" rx="1"/>
    <rect x="388" y="76" width="7" height="7" rx="1"/><rect x="397" y="76" width="7" height="7" rx="1"/><rect x="406" y="76" width="7" height="7" rx="1"/><rect x="415" y="76" width="7" height="7" rx="1"/><rect x="424" y="76" width="7" height="7" rx="1"/><rect x="388" y="85" width="7" height="7" rx="1"/><rect x="397" y="85" width="7" height="7" rx="1"/><rect x="406" y="85" width="7" height="7" rx="1"/><rect x="415" y="85" width="7" height="7" rx="1"/><rect x="424" y="85" width="7" height="7" rx="1"/>
  </g>
  <text x="230" y="108" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">more parallel multiply-accumulate lanes →</text>
  <line x1="30" y1="132" x2="446" y2="132" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.8" marker-end="url(#aia_ar)"/>
  <text x="230" y="127" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">throughput per watt (efficiency)</text>
  <line x1="446" y1="152" x2="30" y2="152" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.6" stroke-dasharray="5 3" marker-end="url(#aia_ar)"/>
  <text x="230" y="166" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">general-purpose flexibility</text>
  <defs><marker id="aia_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>"AI accelerator" is an umbrella, not one chip. As you move from a general CPU through the GPU and TPU/NPU to a fixed-function ASIC, you add parallel multiply-accumulate lanes and win throughput per watt on tensor math — but give up flexibility. The right tool is a point on that trade-off, not the far end.</figcaption>
</figure>

## Overview

"AI accelerator" is an umbrella term, not a single chip. It covers the [GPU](/reference/graphics-processing-unit/) (the workhorse of deep-learning training), Google's [TPU](/reference/tensor-processing-unit/), the on-device [NPU](/reference/neural-processing-unit/) found in phones and laptops, and custom [ASIC](/reference/application-specific-integrated-circuit/) and [FPGA](/reference/field-programmable-gate-array/) designs. What they share is an architecture built for *throughput on parallel, reduced-precision arithmetic* — many multiply-accumulate units fed by high-bandwidth memory — rather than the low-latency, branch-heavy execution a CPU optimizes for.

## Where it fits

Accelerators split roughly by role: large, power-hungry parts (GPUs, TPUs) train and serve big models in the data center, while compact NPUs run inference on the [edge](/reference/edge-ai/) at low power. The performance metric that matters is throughput per watt. For a scanner like GopherTrunk the relevant case is the edge: an accelerator near the antenna could classify or transcribe decoded traffic locally, though the radio's own DSP is conventional [hardware acceleration](/reference/hardware-acceleration/), not neural-network work.

## Sources

[^wiki]: [AI accelerator](https://en.wikipedia.org/wiki/AI_accelerator) — Wikipedia, on the class of hardware built to speed up machine learning.
