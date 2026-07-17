---
slug: cuda
title: CUDA
entry_type: concept
category: hw-accelerators
description: CUDA is NVIDIA's parallel computing platform and programming model that lets general-purpose code run on the GPU, launching kernels across thousands of lightweight threads through C, C++, and other languages.
keywords: CUDA, NVIDIA, GPGPU, GPU computing, parallel computing, kernels, threads, blocks, grid, cuBLAS, cuDNN, GPU programming
aka: [Compute Unified Device Architecture]
autolink: true
infobox:
  - { label: Type, value: Parallel computing platform / API }
  - { label: Vendor, value: NVIDIA }
  - { label: Introduced, value: "2007" }
  - { label: Runs on, value: NVIDIA GPUs }
  - { label: Model, value: "Kernels over a grid of threads" }
  - { label: Languages, value: "C, C++, Fortran, Python bindings" }
see_also: [gpgpu, graphics-processing-unit, nvidia, hardware-acceleration, vector-processor, ai-accelerator]
cite_urls:
  - https://en.wikipedia.org/wiki/CUDA
  - https://developer.nvidia.com/cuda-zone
---

**CUDA** is [NVIDIA](/reference/nvidia/)'s parallel computing platform and programming model that lets ordinary, general-purpose code run on the [GPU](/reference/graphics-processing-unit/) instead of only the CPU.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A CUDA offload pipeline. The host CPU launches a kernel, which fans out across a grid of thread blocks on the GPU device, each block holding many lightweight threads that run in parallel, before the result is copied back to the host." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <rect x="18" y="60" width="70" height="46" rx="4" fill-opacity="0.10" stroke-width="1.4"/>
    <text x="53" y="80" text-anchor="middle" font-size="9" stroke="none" font-weight="600">Host</text>
    <text x="53" y="93" text-anchor="middle" font-size="8" stroke="none">CPU</text>
    <line x1="88" y1="83" x2="138" y2="83" stroke-width="1.3" fill="none"/>
    <path d="M138 83 l-8 -4 v8 z" stroke="none"/>
    <text x="113" y="76" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.9">launch kernel</text>
    <rect x="140" y="26" width="220" height="118" rx="5" fill-opacity="0.05" stroke-width="1.4"/>
    <text x="250" y="40" text-anchor="middle" font-size="8.5" stroke="none" font-weight="600">GPU device &#183; grid of blocks</text>
  </g>
  <g stroke="currentColor" fill="currentColor">
    <g transform="translate(152,50)">
      <rect x="0" y="0" width="94" height="42" rx="3" fill-opacity="0.08" stroke-width="1.1"/>
      <text x="47" y="-4" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.9">block</text>
    </g>
    <g transform="translate(254,50)">
      <rect x="0" y="0" width="94" height="42" rx="3" fill-opacity="0.08" stroke-width="1.1"/>
      <text x="47" y="-4" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.9">block</text>
    </g>
    <g transform="translate(152,100)">
      <rect x="0" y="0" width="94" height="42" rx="3" fill-opacity="0.08" stroke-width="1.1"/>
    </g>
    <g transform="translate(254,100)">
      <rect x="0" y="0" width="94" height="42" rx="3" fill-opacity="0.08" stroke-width="1.1"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none">
    <g font-size="0">
      <rect x="156" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
      <rect x="168" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
      <rect x="180" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
      <rect x="192" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
      <rect x="204" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
      <rect x="216" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
      <rect x="228" y="54" width="8" height="34" fill="currentColor" fill-opacity="0.35"/>
    </g>
    <text x="250" y="160" text-anchor="middle" font-size="7.5" fill-opacity="0.9">each bar = one thread, all running the same kernel on different data</text>
  </g>
</svg>
<figcaption>The host CPU launches a kernel that fans out across a grid of thread blocks on the GPU; every thread runs the same code on its own slice of data, then the result is copied back to the host.</figcaption>
</figure>

## Overview

A CUDA program splits work into a *kernel* — a small function executed in parallel by thousands of lightweight threads, each handling one element of the data. Threads are organised into *blocks*, and blocks into a *grid*, so the same code scales from a small GPU to a large one without being rewritten. The platform exposes the GPU through extensions to C and C++ (and bindings for [Python](/reference/python-language/), Fortran, and others), plus tuned libraries such as cuBLAS for linear algebra and cuDNN for neural networks.

Because it is proprietary to NVIDIA hardware, CUDA competes with the cross-vendor OpenCL and with newer portable frameworks, but its mature tooling, profilers, and library ecosystem made it the de facto standard for GPU computing.[^cuda] The programming model is the practical face of [GPGPU](/reference/gpgpu/): it hides the graphics pipeline entirely and presents the GPU as a general, massively parallel co-processor.

## How it works

CUDA divides the machine into a *host* (the CPU and its memory) and a *device* (the GPU and its memory). Work moves across that boundary in a fixed rhythm, and the thread hierarchy maps onto the hardware's execution units:

| Concept | Meaning | Maps to |
|---------|---------|---------|
| Kernel | Function launched to run in parallel | GPU program |
| Thread | One instance of the kernel | Single lane |
| Block | Group of threads sharing fast memory | Streaming multiprocessor |
| Grid | All blocks of one launch | Whole GPU |
| Host &#8596; device copy | Moving data over PCI Express | The main overhead |

The host-to-device copy is the cost that decides whether offloading pays off: if the compute per byte is low, the transfer dominates and the CPU would have been faster.

## Where it fits

CUDA is the bridge that turned the GPU from a graphics device into a general accelerator, and it underpins most modern [AI accelerator](/reference/ai-accelerator/) workloads on NVIDIA hardware. For a signal-processing pipeline like GopherTrunk, a CUDA kernel can run massively parallel work — large FFTs across many channels, batched [FIR filtering](/reference/digital-filter/), or a polyphase channelizer splitting one wide capture into hundreds of channels — far faster than a CPU. The catch is the host-device transfer: for a handful of narrowband channels the cost of shipping samples to the GPU often outweighs the gain, so CUDA earns its keep only when the channel count and sample rate are high.

## Sources

[^wiki]: [CUDA](https://en.wikipedia.org/wiki/CUDA) — Wikipedia, on NVIDIA's parallel computing platform, kernels, and the thread/block/grid model.
[^cuda]: [CUDA Zone](https://developer.nvidia.com/cuda-zone) — NVIDIA's developer site for the CUDA toolkit and libraries.
