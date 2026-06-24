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

## Overview

An NPU is a class of [AI accelerator](/reference/ai-accelerator/) optimized for *inference at the edge*: running an already-trained model with low latency and minimal power, rather than the large-scale training that data-center [TPUs](/reference/tensor-processing-unit/) and GPUs handle. NPUs are usually one block inside a larger [system-on-a-chip](/reference/system-on-a-chip/), sitting alongside the CPU and GPU and handling tasks such as camera processing, voice recognition, and background-blur. Their headline spec is TOPS (trillions of operations per second) at a given power budget.

## Where it fits

The NPU is what makes [edge AI](/reference/edge-ai/) practical: instead of streaming data to a server, a device runs the model locally. For a distributed scanner like GopherTrunk, an SBC-class NPU could in principle do on-the-edge classification of decoded audio or signals near the antenna, keeping bandwidth and latency low — though GopherTrunk's core DSP is conventional [hardware acceleration](/reference/hardware-acceleration/) territory, not neural-network work.

## Sources

[^wiki]: [AI accelerator](https://en.wikipedia.org/wiki/AI_accelerator) — Wikipedia, on NPUs and related on-device machine-learning hardware.
