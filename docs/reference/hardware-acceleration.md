---
slug: hardware-acceleration
title: Hardware acceleration
entry_type: concept
category: hw-accelerators
description: Hardware acceleration is offloading a task from the general-purpose CPU to specialized hardware built to do it faster or more efficiently, such as a GPU, FPGA, or fixed-function ASIC, trading flexibility for throughput and power efficiency.
keywords: hardware acceleration, offload, GPU, FPGA, ASIC, DSP, NPU, fixed-function, accelerator, throughput, efficiency, flexibility
infobox:
  - { label: Type, value: Computing technique }
  - { label: Idea, value: Offload work from the CPU }
  - { label: Done by, value: "GPU, FPGA, ASIC, DSP, NPU" }
  - { label: Wins, value: Speed and power efficiency }
  - { label: Cost, value: Less flexibility, transfer overhead }
see_also: [graphics-processing-unit, field-programmable-gate-array, application-specific-integrated-circuit, vector-processor, ai-accelerator, soc-vs-discrete]
cite_urls:
  - https://en.wikipedia.org/wiki/Hardware_acceleration
---

**Hardware acceleration** is the practice of offloading a task from the general-purpose [CPU](/reference/central-processing-unit/) onto specialized hardware built to do it faster, more efficiently, or both.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A spectrum of accelerators from the fully programmable CPU through the GPU and the reconfigurable FPGA to the fixed-function ASIC. Moving right, flexibility falls while speed and power efficiency rise." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <rect x="20" y="46" width="94" height="40" rx="4" fill-opacity="0.06" stroke-width="1.3"/>
    <text x="67" y="70" text-anchor="middle" font-size="9" stroke="none" font-weight="600">CPU</text>
    <rect x="130" y="46" width="94" height="40" rx="4" fill-opacity="0.11" stroke-width="1.3"/>
    <text x="177" y="70" text-anchor="middle" font-size="9" stroke="none" font-weight="600">GPU</text>
    <rect x="240" y="46" width="94" height="40" rx="4" fill-opacity="0.18" stroke-width="1.3"/>
    <text x="287" y="70" text-anchor="middle" font-size="9" stroke="none" font-weight="600">FPGA</text>
    <rect x="350" y="46" width="94" height="40" rx="4" fill-opacity="0.26" stroke-width="1.3"/>
    <text x="397" y="70" text-anchor="middle" font-size="9" stroke="none" font-weight="600">ASIC</text>
  </g>
  <g stroke="currentColor" fill="currentColor">
    <line x1="20" y1="106" x2="444" y2="106" stroke-width="1.2" fill="none"/>
    <path d="M444 106 l-8 -4 v8 z" stroke="none"/>
    <line x1="444" y1="126" x2="20" y2="126" stroke-width="1.2" fill="none"/>
    <path d="M20 126 l8 -4 v8 z" stroke="none"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="24" y="102" fill-opacity="0.9">more flexible / programmable</text>
    <text x="440" y="102" text-anchor="end" fill-opacity="0.9">faster &amp; more power-efficient</text>
    <text x="24" y="138" fill-opacity="0.9">runs any code</text>
    <text x="440" y="138" text-anchor="end" fill-opacity="0.9">does one job only</text>
  </g>
</svg>
<figcaption>Accelerators sit on a spectrum: the fully programmable CPU, the massively parallel GPU, the reconfigurable FPGA, and the fixed-function ASIC — moving right trades flexibility for speed and power efficiency.</figcaption>
</figure>

## Overview

A CPU is a generalist: it can run any code, but it pays for that flexibility in speed and power. When a task is performed constantly and has a regular structure — graphics, video encoding, neural-network math, signal processing — it is often worth building hardware that does *only* that. Accelerators span a spectrum: the programmable [GPU](/reference/graphics-processing-unit/) and [vector](/reference/vector-processor/) units, the reconfigurable [FPGA](/reference/field-programmable-gate-array/), and the fixed-function [ASIC](/reference/application-specific-integrated-circuit/), with flexibility falling and efficiency rising along the way.

The gains come from two sources: doing many operations at once (parallelism) and doing each with circuitry shaped exactly to the task rather than a general instruction pipeline. A dedicated video encoder or crypto block can beat a CPU by orders of magnitude in both speed and energy per operation — but it can do nothing else, and it cannot be changed once fabricated.

## How it works

The engineering choice is *how much to freeze in silicon*. The more a design commits to one task, the faster and leaner it runs it, and the less it can do anything else:

| Accelerator | Programmability | Best at | Cost |
|-------------|-----------------|---------|------|
| CPU | Fully general | Anything, sequentially | Slowest per watt |
| GPU | Software kernels | Data-parallel math | Transfer overhead |
| FPGA | Reconfigurable logic | Custom pipelines, low latency | Hard to develop |
| ASIC | Fixed at fabrication | One job, at scale | No changes, high NRE |

Because offloading adds data-transfer and coordination overhead, it pays only when the accelerated portion is both large and regular — Amdahl's law limits how much a fast accelerator can help if most of the runtime is still ordinary CPU code.

## Where it fits

The engineering question is always *what to offload*: moving work to an accelerator adds complexity and data-transfer overhead, so it pays only when the speedup is large or the CPU genuinely cannot keep up. This is a live trade-off in software-defined radio. GopherTrunk does its DSP and protocol decoding in software on the CPU, deliberately keeping the radio a simple front end and leaning on CPU [vector units](/reference/vector-processor/) for the per-sample math; an [FPGA](/reference/field-programmable-gate-array/) doing channelization in hardware would accelerate wideband, many-channel capture, at the cost of flexibility and a much harder development path.

## Sources

[^wiki]: [Hardware acceleration](https://en.wikipedia.org/wiki/Hardware_acceleration) — Wikipedia, on offloading tasks from the CPU to specialized hardware and the flexibility-versus-efficiency spectrum.
