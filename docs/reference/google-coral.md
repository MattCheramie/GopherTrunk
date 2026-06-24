---
slug: google-coral
title: Google Coral
entry_type: hardware
category: hw-sbc
description: Google Coral is a hardware platform built around the Edge TPU, an AI accelerator that runs TensorFlow Lite models locally, sold as a single-board computer, a plug-in module, and a USB stick.
keywords: Google Coral, Edge TPU, Coral Dev Board, Coral USB Accelerator, TensorFlow Lite, edge AI, machine learning accelerator
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

## Overview

The Edge TPU is a cut-down [tensor processing unit](/reference/tensor-processing-unit/) that executes TensorFlow Lite models very efficiently at low power. Coral ships in several forms: a full [single-board computer](/reference/single-board-computer/) (the Coral Dev Board), a solder-down module, and a USB Accelerator stick that adds the TPU to an existing host such as a [Raspberry Pi](/reference/raspberry-pi/).[^coral]

## What it's for

Coral targets [edge AI](/reference/edge-ai/): on-device vision, audio, and sensor inference where sending data to a server is too slow, too costly, or impossible. It is a more specialised choice than an [NVIDIA Jetson](/reference/nvidia-jetson/) — the Edge TPU runs supported quantised models fast and cheap, but it is not a general GPU. In a signal-processing project, a Coral could classify or flag patterns in decoded data at the edge while a Pi handles the radio.

## Sources

[^wiki]: [Edge TPU](https://en.wikipedia.org/wiki/Tensor_Processing_Unit#Edge_TPU) — Wikipedia, on the Edge TPU at the heart of Coral.
[^coral]: [Coral](https://coral.ai/) — Google's Coral product site, on the Dev Board, modules, and USB Accelerator.
