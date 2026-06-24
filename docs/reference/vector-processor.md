---
slug: vector-processor
title: Vector processor
entry_type: concept
category: hw-accelerators
description: A vector processor is a design that applies one instruction to a whole array of data elements at once, exploiting data parallelism for high throughput in numeric and signal-processing workloads.
keywords: vector processor, SIMD, array processor, data parallelism, SSE, AVX, NEON, supercomputer, DSP
aka: [Array processor]
infobox:
  - { label: Type, value: Processor architecture }
  - { label: Model, value: "SIMD (single instruction, many data)" }
  - { label: Strength, value: Data-parallel numeric throughput }
  - { label: Modern form, value: "SIMD units (AVX, NEON), GPUs" }
see_also: [central-processing-unit, graphics-processing-unit, gpgpu, hardware-acceleration, digital-filter, fast-fourier-transform]
cite_urls:
  - https://en.wikipedia.org/wiki/Vector_processor
---

A **vector processor** is a processor designed to apply a single instruction to a whole array of data elements at once, rather than one element per instruction.[^wiki]

## Overview

The model is *SIMD* — single instruction, multiple data. Where a scalar processor adds two numbers per add instruction, a vector unit adds two arrays of numbers in one go, amortizing instruction overhead and keeping wide arithmetic pipelines full. The idea powered the early Cray supercomputers and survives today as the SIMD extensions built into ordinary CPUs (Intel's SSE/AVX, ARM's NEON) and, taken to an extreme, in the thousands-of-lanes design of the [GPU](/reference/graphics-processing-unit/).

## Where it fits

Vector processing is the foundation of [GPGPU](/reference/gpgpu/) and of most numeric [hardware acceleration](/reference/hardware-acceleration/): any workload that does the same math across long, regular arrays benefits. Digital signal processing is a prime example — a [FIR digital filter](/reference/digital-filter/) or an [FFT](/reference/fast-fourier-transform/) multiplies and sums across streams of samples, exactly the data-parallel shape SIMD exploits. GopherTrunk's per-sample DSP on the [CPU](/reference/central-processing-unit/) leans on these vector units to keep up with high sample rates.

## Sources

[^wiki]: [Vector processor](https://en.wikipedia.org/wiki/Vector_processor) — Wikipedia, on SIMD/array processor architectures.
