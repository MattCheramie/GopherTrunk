---
slug: google-coral
title: Google Coral
entry_type: hardware
category: hw-sbc
description: Google Coral is a hardware platform built around the Edge TPU, an AI accelerator that runs TensorFlow Lite models locally, sold as a single-board computer, a plug-in module, and a USB stick.
keywords: Google Coral, Edge TPU, Coral Dev Board, Coral USB Accelerator, TensorFlow Lite, edge AI, machine learning accelerator, quantized model
aka: [Coral, Edge TPU]
autolink: true
infobox:
  - { label: Type, value: Edge-AI platform (SBC + accelerators) }
  - { label: Maker, value: Google }
  - { label: Core, value: Edge TPU }
  - { label: Runs, value: TensorFlow Lite models }
  - { label: Forms, value: Dev Board, module, USB stick }
see_also: [edge-ai, single-board-computer, ai-accelerator, tensor-processing-unit, raspberry-pi, neural-processing-unit]
related_lessons:
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Tensor_Processing_Unit#Edge_TPU
  - https://coral.ai/
---

**Google Coral** is a hardware platform built around the *Edge TPU*, a small [AI accelerator](/reference/ai-accelerator/) that runs machine-learning models locally rather than in the cloud.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 158" role="img" aria-label="The three forms Google Coral ships in, each containing the same Edge TPU. On the left a full Dev Board single-board computer. In the middle a small solder-down module. On the right a USB Accelerator stick that plugs into a host computer such as a Raspberry Pi. All three run the same TensorFlow Lite models." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="26" y="34" width="118" height="80" rx="5" fill-opacity="0.05" fill="currentColor"/>
    <rect x="60" y="60" width="40" height="28" rx="3" fill-opacity="0.2" fill="currentColor"/>
    <rect x="34" y="102" width="102" height="7" rx="1.5" fill-opacity="0.14" fill="currentColor"/>
    <rect x="176" y="52" width="94" height="52" rx="4" fill-opacity="0.05" fill="currentColor"/>
    <rect x="200" y="66" width="40" height="24" rx="3" fill-opacity="0.2" fill="currentColor"/>
    <rect x="184" y="96" width="78" height="5" rx="1" fill-opacity="0.14" fill="currentColor"/>
    <rect x="322" y="58" width="86" height="34" rx="4" fill-opacity="0.05" fill="currentColor"/>
    <rect x="336" y="66" width="34" height="18" rx="3" fill-opacity="0.2" fill="currentColor"/>
    <rect x="408" y="68" width="20" height="14" rx="2" fill-opacity="0.14" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="85" y="27" font-size="8.5" font-weight="600">Dev Board</text>
    <text x="80" y="78" font-size="7">TPU</text>
    <text x="85" y="126" font-size="7.5" fill-opacity="0.9">full SBC</text>
    <text x="223" y="45" font-size="8.5" font-weight="600">Module</text>
    <text x="220" y="82" font-size="7">TPU</text>
    <text x="223" y="118" font-size="7.5" fill-opacity="0.9">solder-down</text>
    <text x="365" y="51" font-size="8.5" font-weight="600">USB stick</text>
    <text x="353" y="79" font-size="7">TPU</text>
    <text x="365" y="106" font-size="7.5" fill-opacity="0.9">adds to a host</text>
    <text x="230" y="146" font-size="8" fill-opacity="0.9">one Edge TPU, three packages — all run TensorFlow Lite</text>
  </g>
</svg>
<figcaption>The same Edge TPU ships three ways: a full Dev Board, a solder-down module for products, and a USB Accelerator that clips the TPU onto an existing host — so you can add local inference at whatever integration level a project needs.</figcaption>
</figure>

## Overview

The Edge TPU is a cut-down [tensor processing unit](/reference/tensor-processing-unit/) that executes TensorFlow Lite models very efficiently at low power. It is not a general processor: it accelerates the tensor math at the core of neural-network inference and little else, which is exactly why it can hit high throughput on a couple of watts. Models must be quantised to 8-bit integers and compiled for the TPU before they will run on it.

Coral ships in several forms so the same accelerator suits different projects: a full [single-board computer](/reference/single-board-computer/) (the Coral Dev Board), a solder-down module for embedding in a product, and a USB Accelerator stick that adds the TPU to an existing host such as a [Raspberry Pi](/reference/raspberry-pi/).[^coral] The USB stick is the most common entry point, because it upgrades a board you already have rather than replacing it.

## Coral vs Jetson

| | Google Coral (Edge TPU) | [NVIDIA Jetson](/reference/nvidia-jetson/) |
|---|-------------------------|--------|
| Core | Edge TPU (fixed-function) | ARM CPU + CUDA GPU |
| Runs | Quantised TensorFlow Lite | Broad frameworks, general GPU |
| Power | Very low (~2 W) | Higher (5–60 W) |
| Flexibility | Narrow, supported models only | General-purpose accelerator |
| Best for | Cheap, fixed inference tasks | Heavier or varied ML workloads |

## Where it fits

Coral targets [edge AI](/reference/edge-ai/): on-device vision, audio, and sensor inference where sending data to a server is too slow, too costly, or impossible. It is a more specialised choice than an [NVIDIA Jetson](/reference/nvidia-jetson/) — the Edge TPU runs supported quantised models fast and cheap, but it is not a general GPU, so anything outside that lane belongs elsewhere. In a signal-processing project, a Coral USB Accelerator on a Pi could classify or flag patterns in decoded data at the edge while the Pi itself handles the radio, keeping inference off the CPU that is busy demodulating.

## Sources

[^wiki]: [Edge TPU](https://en.wikipedia.org/wiki/Tensor_Processing_Unit#Edge_TPU) — Wikipedia, on the Edge TPU at the heart of Coral.
[^coral]: [Coral](https://coral.ai/) — Google's Coral product site, on the Dev Board, modules, and USB Accelerator.
