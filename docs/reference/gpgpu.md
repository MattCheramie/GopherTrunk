---
slug: gpgpu
title: GPGPU
entry_type: concept
category: hw-accelerators
description: GPGPU is the practice of using a graphics processing unit for general-purpose computation, exploiting its thousands of parallel cores for non-graphics workloads such as math, simulation, and machine learning.
keywords: GPGPU, general-purpose GPU, GPU computing, parallel computing, CUDA, OpenCL, compute shader
aka: [General-purpose computing on graphics processing units]
autolink: true
infobox:
  - { label: Type, value: Computing technique }
  - { label: Hardware, value: Graphics processing unit }
  - { label: Strength, value: Massively parallel data }
  - { label: APIs, value: "CUDA, OpenCL, compute shaders" }
see_also: [graphics-processing-unit, cuda, hardware-acceleration, vector-processor, central-processing-unit, ai-accelerator]
cite_urls:
  - https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units
---

**GPGPU** (general-purpose computing on graphics processing units) is the practice of using a [GPU](/reference/graphics-processing-unit/) to run ordinary computation rather than only rendering images.[^wiki]

## Overview

A GPU contains thousands of small cores built to shade pixels in parallel. When a problem can be expressed as the *same operation applied to many data elements at once* — the same SIMD/data-parallel pattern a [vector processor](/reference/vector-processor/) exploits — those cores can be redirected at it. Early GPGPU work smuggled math through the graphics pipeline; dedicated APIs like [CUDA](/reference/cuda/) and the cross-vendor OpenCL later exposed the hardware directly, making GPGPU mainstream for scientific computing, cryptography, and machine learning.

## What it's for

GPGPU shines on large, regular, parallel workloads and is the foundation of modern deep learning, where it long preceded purpose-built [AI accelerators](/reference/ai-accelerator/). It is poor at branch-heavy, sequential work, which still belongs on the [CPU](/reference/central-processing-unit/). In a software-defined radio context, GPGPU can accelerate wideband DSP — large FFTs, polyphase channelizers splitting one capture into many channels — but the cost of moving samples to and from the GPU only pays off when the channel count is high.

## Sources

[^wiki]: [General-purpose computing on graphics processing units](https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units) — Wikipedia, on using GPUs for non-graphics computation.
