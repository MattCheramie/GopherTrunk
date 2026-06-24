---
slug: tensor-processing-unit
title: Tensor processing unit (TPU)
entry_type: hardware
category: hw-accelerators
description: A tensor processing unit (TPU) is a custom chip designed by Google to accelerate machine-learning workloads, built around large matrix-multiply arrays optimized for neural-network math.
keywords: TPU, tensor processing unit, Google, AI accelerator, machine learning chip, systolic array, matrix multiply
aka: [TPU]
autolink: true
infobox:
  - { label: Type, value: AI accelerator (ASIC) }
  - { label: Vendor, value: Google }
  - { label: Introduced, value: "2016" }
  - { label: Specialty, value: Neural-network matrix math }
  - { label: Core, value: Systolic matrix-multiply array }
see_also: [ai-accelerator, neural-processing-unit, application-specific-integrated-circuit, graphics-processing-unit, google-coral, hardware-acceleration]
cite_urls:
  - https://en.wikipedia.org/wiki/Tensor_Processing_Unit
  - https://cloud.google.com/tpu
---

A **tensor processing unit** (**TPU**) is a custom chip designed by Google to accelerate machine-learning workloads, built around large arrays that perform the matrix multiplications at the heart of neural networks.[^wiki]

## Overview

A TPU is an [application-specific integrated circuit](/reference/application-specific-integrated-circuit/) (ASIC): rather than the general flexibility of a CPU or GPU, it dedicates almost all its silicon to a *systolic array* — a grid of multiply-accumulate units that streams data through to compute matrix products with very high throughput per watt. The trade-off is narrow specialization: a TPU runs tensor math (typically in reduced precision such as bfloat16) extremely well and little else. Google introduced TPUs in 2016 to power its own services and now offers them through its cloud, with the small Coral Edge TPU bringing the design to embedded devices.[^cloud]

## Where it fits

The TPU is the canonical example of a purpose-built [AI accelerator](/reference/ai-accelerator/), competing with the [GPU](/reference/graphics-processing-unit/) for training and inference and with the on-device [NPU](/reference/neural-processing-unit/) at the edge. Its strength is data-center scale neural-network training and serving; it is not a general DSP engine, so it has no direct role in GopherTrunk's signal chain, though the same edge-AI parts (see [Google Coral](/reference/google-coral/)) could classify decoded traffic.

## Sources

[^wiki]: [Tensor Processing Unit](https://en.wikipedia.org/wiki/Tensor_Processing_Unit) — Wikipedia, on Google's machine-learning accelerator.
[^cloud]: [Cloud TPU](https://cloud.google.com/tpu) — Google's documentation for the TPU and its architecture.
