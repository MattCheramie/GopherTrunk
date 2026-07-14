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

<figure class="figure" markdown="0">
<svg viewBox="0 0 420 240" role="img" aria-label="A systolic array: a four-by-four grid of multiply-accumulate cells. Weights stream down from the top and activations stream in from the left; each cell multiplies and accumulates as data passes through, and partial sums flow out of the bottom, computing a matrix product with high throughput." xmlns="http://www.w3.org/2000/svg">
  <text x="210" y="16" text-anchor="middle" font-size="10.5" fill="currentColor" font-weight="600">Systolic array: one grid, data streams through</text>
  <text x="210" y="33" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85">weights ↓</text>
  <text x="26" y="128" font-size="8.5" fill="currentColor" fill-opacity="0.85" transform="rotate(-90 26 128)">activations →</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="100" y="52" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="122" y="72">×+</text>
    <rect x="148" y="52" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="170" y="72">×+</text>
    <rect x="196" y="52" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="218" y="72">×+</text>
    <rect x="244" y="52" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="266" y="72">×+</text>
    <rect x="100" y="90" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="122" y="110">×+</text>
    <rect x="148" y="90" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="170" y="110">×+</text>
    <rect x="196" y="90" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="218" y="110">×+</text>
    <rect x="244" y="90" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="266" y="110">×+</text>
    <rect x="100" y="128" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="122" y="148">×+</text>
    <rect x="148" y="128" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="170" y="148">×+</text>
    <rect x="196" y="128" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="218" y="148">×+</text>
    <rect x="244" y="128" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="266" y="148">×+</text>
    <rect x="100" y="166" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="122" y="186">×+</text>
    <rect x="148" y="166" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="170" y="186">×+</text>
    <rect x="196" y="166" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="218" y="186">×+</text>
    <rect x="244" y="166" width="44" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/><text x="266" y="186">×+</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7">
    <line x1="122" y1="40" x2="122" y2="52" marker-end="url(#tpu_ar)"/>
    <line x1="170" y1="40" x2="170" y2="52" marker-end="url(#tpu_ar)"/>
    <line x1="218" y1="40" x2="218" y2="52" marker-end="url(#tpu_ar)"/>
    <line x1="266" y1="40" x2="266" y2="52" marker-end="url(#tpu_ar)"/>
    <line x1="62" y1="69" x2="100" y2="69" marker-end="url(#tpu_ar)"/>
    <line x1="62" y1="107" x2="100" y2="107" marker-end="url(#tpu_ar)"/>
    <line x1="62" y1="145" x2="100" y2="145" marker-end="url(#tpu_ar)"/>
    <line x1="62" y1="183" x2="100" y2="183" marker-end="url(#tpu_ar)"/>
    <line x1="122" y1="200" x2="122" y2="214" marker-end="url(#tpu_ar)"/>
    <line x1="170" y1="200" x2="170" y2="214" marker-end="url(#tpu_ar)"/>
    <line x1="218" y1="200" x2="218" y2="214" marker-end="url(#tpu_ar)"/>
    <line x1="266" y1="200" x2="266" y2="214" marker-end="url(#tpu_ar)"/>
  </g>
  <text x="210" y="232" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.85">partial sums out</text>
  <defs><marker id="tpu_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A TPU dedicates almost all its silicon to a systolic array: a grid of multiply-accumulate cells. Weights and activations stream in from two edges and ripple through the cells, so one matrix multiply flows through the array at high throughput per watt — very fast at tensor math, and little else.</figcaption>
</figure>

## Overview

A TPU is an [application-specific integrated circuit](/reference/application-specific-integrated-circuit/) (ASIC): rather than the general flexibility of a CPU or GPU, it dedicates almost all its silicon to a *systolic array* — a grid of multiply-accumulate units that streams data through to compute matrix products with very high throughput per watt. The trade-off is narrow specialization: a TPU runs tensor math (typically in reduced precision such as bfloat16) extremely well and little else. Google introduced TPUs in 2016 to power its own services and now offers them through its cloud, with the small Coral Edge TPU bringing the design to embedded devices.[^cloud]

## Where it fits

The TPU is the canonical example of a purpose-built [AI accelerator](/reference/ai-accelerator/), competing with the [GPU](/reference/graphics-processing-unit/) for training and inference and with the on-device [NPU](/reference/neural-processing-unit/) at the edge. Its strength is data-center scale neural-network training and serving; it is not a general DSP engine, so it has no direct role in GopherTrunk's signal chain, though the same edge-AI parts (see [Google Coral](/reference/google-coral/)) could classify decoded traffic.

## Sources

[^wiki]: [Tensor Processing Unit](https://en.wikipedia.org/wiki/Tensor_Processing_Unit) — Wikipedia, on Google's machine-learning accelerator.
[^cloud]: [Cloud TPU](https://cloud.google.com/tpu) — Google's documentation for the TPU and its architecture.
