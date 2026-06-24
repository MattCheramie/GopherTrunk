---
slug: cuda
title: CUDA
entry_type: concept
category: hw-accelerators
description: CUDA is NVIDIA's parallel computing platform and programming model that lets general-purpose code run on the GPU, exposing thousands of cores through C, C++, and other languages.
keywords: CUDA, NVIDIA, GPGPU, GPU computing, parallel computing, kernels, cuDNN, GPU programming
aka: [Compute Unified Device Architecture]
autolink: true
infobox:
  - { label: Type, value: Parallel computing platform / API }
  - { label: Vendor, value: NVIDIA }
  - { label: Introduced, value: "2007" }
  - { label: Runs on, value: NVIDIA GPUs }
  - { label: Languages, value: "C, C++, Fortran, Python bindings" }
see_also: [gpgpu, graphics-processing-unit, nvidia, hardware-acceleration, vector-processor, ai-accelerator]
cite_urls:
  - https://en.wikipedia.org/wiki/CUDA
  - https://developer.nvidia.com/cuda-zone
---

**CUDA** is [NVIDIA](/reference/nvidia/)'s parallel computing platform and programming model that lets ordinary, general-purpose code run on the [GPU](/reference/graphics-processing-unit/) instead of only the CPU.[^wiki]

## Overview

A CUDA program splits work into a *kernel* — a small function executed in parallel by thousands of lightweight threads, each handling one element of the data. The platform exposes the GPU through extensions to C and C++ (and bindings for [Python](/reference/python-language/), Fortran, and others), plus tuned libraries such as cuBLAS for linear algebra and cuDNN for neural networks. Because it is proprietary to NVIDIA hardware, CUDA competes with the cross-vendor OpenCL and with newer portable frameworks, but its mature tooling made it the de facto standard for GPU computing.[^cuda]

## Where it fits

CUDA is the bridge that turned the GPU from a graphics device into a general accelerator (see [GPGPU](/reference/gpgpu/)), and it underpins most modern [AI accelerator](/reference/ai-accelerator/) workloads on NVIDIA hardware. For a signal-processing pipeline like GopherTrunk, a CUDA kernel can run massively parallel work — large FFTs across many channels, or batched filtering — far faster than a CPU, though for a handful of narrowband channels the data-transfer overhead to the GPU often outweighs the gain.

## Sources

[^wiki]: [CUDA](https://en.wikipedia.org/wiki/CUDA) — Wikipedia, on NVIDIA's parallel computing platform and programming model.
[^cuda]: [CUDA Zone](https://developer.nvidia.com/cuda-zone) — NVIDIA's developer site for the CUDA toolkit and libraries.
