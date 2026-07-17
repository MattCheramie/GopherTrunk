---
slug: vector-processor
title: Vector processor
entry_type: concept
category: hw-accelerators
description: A vector processor applies one instruction to a whole array of data elements at once, exploiting data parallelism for high throughput in numeric and signal-processing workloads; the SIMD model survives in modern CPU vector units and GPUs.
keywords: vector processor, SIMD, array processor, data parallelism, lanes, SSE, AVX, NEON, supercomputer, Cray, DSP
aka: [Array processor]
infobox:
  - { label: Type, value: Processor architecture }
  - { label: Model, value: "SIMD (single instruction, many data)" }
  - { label: Strength, value: Data-parallel numeric throughput }
  - { label: Origin, value: "Cray supercomputers, 1970s" }
  - { label: Modern form, value: "SIMD units (AVX, NEON), GPUs" }
see_also: [central-processing-unit, graphics-processing-unit, gpgpu, hardware-acceleration, digital-filter, fast-fourier-transform]
cite_urls:
  - https://en.wikipedia.org/wiki/Vector_processor
---

A **vector processor** is a processor designed to apply a single instruction to a whole array of data elements at once, rather than one element per instruction.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A scalar add processes one pair of numbers per instruction, needing four instructions for a four-element array. A vector or SIMD add processes all four lanes with a single instruction." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <text x="20" y="20" font-size="9" stroke="none" font-weight="600">Scalar &#183; one add per instruction</text>
    <g font-family="ui-monospace, monospace" font-size="8.5" stroke="none">
      <rect x="20" y="30" width="26" height="18" rx="2" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
      <text x="33" y="43" text-anchor="middle">a0</text>
      <text x="54" y="43" text-anchor="middle" stroke="none">+</text>
      <rect x="62" y="30" width="26" height="18" rx="2" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
      <text x="75" y="43" text-anchor="middle">b0</text>
      <text x="96" y="43" text-anchor="middle" stroke="none">=</text>
      <rect x="104" y="30" width="26" height="18" rx="2" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
      <text x="117" y="43" text-anchor="middle">c0</text>
      <text x="150" y="43" stroke="none" fill-opacity="0.85">&#8592; instruction 1</text>
    </g>
    <text x="20" y="66" font-size="7.5" stroke="none" fill-opacity="0.85">... then a1+b1, a2+b2, a3+b3 &#8212; four separate instructions</text>
  </g>
  <g stroke="currentColor" fill="currentColor">
    <text x="20" y="94" font-size="9" stroke="none" font-weight="600">Vector / SIMD &#183; one add, four lanes</text>
    <g font-family="ui-monospace, monospace" font-size="8.5" stroke="none">
      <rect x="20" y="104" width="70" height="52" rx="3" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/>
      <text x="55" y="118" text-anchor="middle">a0</text>
      <text x="55" y="131" text-anchor="middle">a1</text>
      <text x="55" y="144" text-anchor="middle">a2</text>
      <text x="55" y="156" text-anchor="middle">a3</text>
      <text x="104" y="134" text-anchor="middle" stroke="none">+</text>
      <rect x="118" y="104" width="70" height="52" rx="3" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/>
      <text x="153" y="118" text-anchor="middle">b0</text>
      <text x="153" y="131" text-anchor="middle">b1</text>
      <text x="153" y="144" text-anchor="middle">b2</text>
      <text x="153" y="156" text-anchor="middle">b3</text>
      <text x="202" y="134" text-anchor="middle" stroke="none">=</text>
      <rect x="216" y="104" width="70" height="52" rx="3" fill-opacity="0.20" stroke="currentColor" stroke-width="1.1"/>
      <text x="251" y="118" text-anchor="middle">c0</text>
      <text x="251" y="131" text-anchor="middle">c1</text>
      <text x="251" y="144" text-anchor="middle">c2</text>
      <text x="251" y="156" text-anchor="middle">c3</text>
      <text x="304" y="134" stroke="none" fill-opacity="0.85">&#8592; one instruction</text>
    </g>
  </g>
</svg>
<figcaption>A scalar processor adds one pair of numbers per instruction, so a four-element array costs four instructions; a vector (SIMD) unit adds all four lanes with a single instruction, amortizing the overhead across the whole array.</figcaption>
</figure>

## Overview

The model is *SIMD* — single instruction, multiple data. Where a scalar processor adds two numbers per add instruction, a vector unit adds two arrays of numbers in one go, amortizing instruction overhead and keeping wide arithmetic pipelines full. The idea powered the early Cray supercomputers and survives today as the SIMD extensions built into ordinary CPUs (Intel's SSE/AVX, ARM's NEON) and, taken to an extreme, in the thousands-of-lanes design of the [GPU](/reference/graphics-processing-unit/).

The win is twofold: fewer instructions are fetched and decoded for the same work, and the hardware can lay out identical arithmetic units side by side as *lanes* that all fire together. The limit is that every lane must do the same operation in lock-step, so vector code favours long, regular arrays with no per-element branching — the further the data strays from that shape, the less the vector unit helps.

## Anatomy

A vector unit is defined by how wide it is and how it handles data that does not fill its lanes cleanly:

| Property | Scalar | Vector / SIMD |
|----------|--------|---------------|
| Elements per instruction | One | Many (a full register width) |
| Instruction overhead | Per element | Amortized over the lanes |
| Best-case shape | Any | Long, regular arrays |
| Branch handling | Free | Costly (predication or masks) |
| Examples | Plain add | AVX, NEON, GPU warps |

Because all lanes share one instruction stream, a vector unit spends far less silicon on control and far more on arithmetic than a scalar core — the same bargain a [GPU](/reference/graphics-processing-unit/) takes to its extreme.

## Where it fits

Vector processing is the foundation of [GPGPU](/reference/gpgpu/) and of most numeric [hardware acceleration](/reference/hardware-acceleration/): any workload that does the same math across long, regular arrays benefits. Digital signal processing is a prime example — a [FIR digital filter](/reference/digital-filter/) or an [FFT](/reference/fast-fourier-transform/) multiplies and sums across streams of samples, exactly the data-parallel shape SIMD exploits. GopherTrunk's per-sample DSP on the [CPU](/reference/central-processing-unit/) leans on these vector units to keep up with high sample rates, processing many IQ samples per instruction rather than one at a time.

## Sources

[^wiki]: [Vector processor](https://en.wikipedia.org/wiki/Vector_processor) — Wikipedia, on SIMD/array processor architectures, lanes, and the scalar-versus-vector distinction.
