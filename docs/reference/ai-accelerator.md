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

## Overview

"AI accelerator" is an umbrella term, not a single chip. It covers the [GPU](/reference/graphics-processing-unit/) (the workhorse of deep-learning training), Google's [TPU](/reference/tensor-processing-unit/), the on-device [NPU](/reference/neural-processing-unit/) found in phones and laptops, and custom [ASIC](/reference/application-specific-integrated-circuit/) and [FPGA](/reference/field-programmable-gate-array/) designs. What they share is an architecture built for *throughput on parallel, reduced-precision arithmetic* — many multiply-accumulate units fed by high-bandwidth memory — rather than the low-latency, branch-heavy execution a CPU optimizes for.

## Where it fits

Accelerators split roughly by role: large, power-hungry parts (GPUs, TPUs) train and serve big models in the data center, while compact NPUs run inference on the [edge](/reference/edge-ai/) at low power. The performance metric that matters is throughput per watt. For a scanner like GopherTrunk the relevant case is the edge: an accelerator near the antenna could classify or transcribe decoded traffic locally, though the radio's own DSP is conventional [hardware acceleration](/reference/hardware-acceleration/), not neural-network work.

## Sources

[^wiki]: [AI accelerator](https://en.wikipedia.org/wiki/AI_accelerator) — Wikipedia, on the class of hardware built to speed up machine learning.
