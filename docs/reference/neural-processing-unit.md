---
slug: neural-processing-unit
title: Neural processing unit (NPU)
entry_type: hardware
category: hw-accelerators
description: A neural processing unit (NPU) is an on-device accelerator for machine-learning inference, integrated into phones, laptops, and edge devices to run neural networks efficiently at low power.
keywords: NPU, neural processing unit, AI accelerator, on-device AI, edge inference, machine learning chip, SoC
aka: [NPU]
autolink: true
infobox:
  - { label: Type, value: AI accelerator (on-device) }
  - { label: Found in, value: Phones, laptops, edge SoCs }
  - { label: Job, value: Neural-network inference }
  - { label: Optimized for, value: Low-power, low-latency AI }
  - { label: Measured in, value: TOPS (tera-ops/sec) }
see_also: [ai-accelerator, tensor-processing-unit, system-on-a-chip, edge-ai, graphics-processing-unit, hardware-acceleration]
cite_urls:
  - https://en.wikipedia.org/wiki/AI_accelerator
---

A **neural processing unit** (**NPU**) is an on-device accelerator built to run neural-network inference efficiently — typically integrated into a phone, laptop, or edge device's main chip so AI features work without the cloud.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 214" role="img" aria-label="Inside a system-on-a-chip sit a CPU, a GPU, and an NPU. The NPU is a block of parallel multiply-accumulate lanes that runs a trained neural network right on the device. A dashed link to a crossed-out cloud shows the inference stays local, for low power and low latency, with no round-trip to a server." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="34" width="286" height="152" rx="6" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 3"/>
  <text x="28" y="48" text-anchor="start" font-size="8" fill="currentColor" fill-opacity="0.85">system-on-a-chip (phone / SBC)</text>
  <rect x="30" y="58" width="52" height="30" rx="3" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="56" y="77" text-anchor="middle" font-size="8.5" fill="currentColor">CPU</text>
  <rect x="30" y="96" width="52" height="30" rx="3" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1"/><text x="56" y="115" text-anchor="middle" font-size="8.5" fill="currentColor">GPU</text>
  <rect x="98" y="56" width="196" height="120" rx="4" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/>
  <text x="196" y="70" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">NPU — parallel MAC lanes</text>
  <g fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="0.8">
    <rect x="112" y="80" width="24" height="16" rx="2"/><rect x="140" y="80" width="24" height="16" rx="2"/><rect x="168" y="80" width="24" height="16" rx="2"/><rect x="196" y="80" width="24" height="16" rx="2"/><rect x="224" y="80" width="24" height="16" rx="2"/>
    <rect x="112" y="102" width="24" height="16" rx="2"/><rect x="140" y="102" width="24" height="16" rx="2"/><rect x="168" y="102" width="24" height="16" rx="2"/><rect x="196" y="102" width="24" height="16" rx="2"/><rect x="224" y="102" width="24" height="16" rx="2"/>
    <rect x="112" y="124" width="24" height="16" rx="2"/><rect x="140" y="124" width="24" height="16" rx="2"/><rect x="168" y="124" width="24" height="16" rx="2"/><rect x="196" y="124" width="24" height="16" rx="2"/><rect x="224" y="124" width="24" height="16" rx="2"/>
  </g>
  <g fill="currentColor" text-anchor="middle" font-size="7">
    <text x="124" y="92">×+</text><text x="152" y="92">×+</text><text x="180" y="92">×+</text><text x="208" y="92">×+</text><text x="236" y="92">×+</text>
    <text x="124" y="114">×+</text><text x="152" y="114">×+</text><text x="180" y="114">×+</text><text x="208" y="114">×+</text><text x="236" y="114">×+</text>
    <text x="124" y="136">×+</text><text x="152" y="136">×+</text><text x="180" y="136">×+</text><text x="208" y="136">×+</text><text x="236" y="136">×+</text>
  </g>
  <text x="196" y="158" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">trained model runs here · measured in TOPS</text>
  <line x1="304" y1="90" x2="336" y2="90" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 2" marker-end="url(#npu_ar)"/>
  <rect x="338" y="70" width="72" height="40" rx="8" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1"/>
  <text x="374" y="94" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.7">cloud</text>
  <line x1="346" y1="72" x2="402" y2="108" stroke="currentColor" stroke-width="1.4"/>
  <text x="374" y="128" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">no round-trip</text>
  <text x="196" y="200" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">many multiply-accumulate lanes run inference on-device, at low power</text>
  <defs><marker id="npu_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An NPU is one block inside a system-on-a-chip, alongside the CPU and GPU: a wide array of multiply-accumulate lanes that runs an already-trained neural network right on the device. Keeping inference local — no round-trip to the cloud — is what makes on-device AI fast and low-power, and its headline figure is TOPS, trillions of operations per second.</figcaption>
</figure>

## Overview

An NPU is a class of [AI accelerator](/reference/ai-accelerator/) optimized for *inference at the edge*: running an already-trained model with low latency and minimal power, rather than the large-scale training that data-center [TPUs](/reference/tensor-processing-unit/) and GPUs handle. NPUs are usually one block inside a larger [system-on-a-chip](/reference/system-on-a-chip/), sitting alongside the CPU and GPU and handling tasks such as camera processing, voice recognition, and background-blur. Their headline spec is TOPS (trillions of operations per second) at a given power budget.

## Where it fits

The NPU is what makes [edge AI](/reference/edge-ai/) practical: instead of streaming data to a server, a device runs the model locally. For a distributed scanner like GopherTrunk, an SBC-class NPU could in principle do on-the-edge classification of decoded audio or signals near the antenna, keeping bandwidth and latency low — though GopherTrunk's core DSP is conventional [hardware acceleration](/reference/hardware-acceleration/) territory, not neural-network work.

## Sources

[^wiki]: [AI accelerator](https://en.wikipedia.org/wiki/AI_accelerator) — Wikipedia, on NPUs and related on-device machine-learning hardware.
