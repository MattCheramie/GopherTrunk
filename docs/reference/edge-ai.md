---
slug: edge-ai
title: Edge AI
entry_type: concept
category: hw-sbc
description: Edge AI is the practice of running machine-learning models directly on local devices such as single-board computers, cameras, and sensors, rather than sending data to the cloud for inference.
keywords: edge AI, on-device inference, edge inference, machine learning at the edge, AI accelerator, Edge TPU, low latency AI, privacy
aka: [on-device AI, edge inference]
infobox:
  - { label: Type, value: Computing approach }
  - { label: Where, value: On local devices }
  - { label: Versus, value: Cloud inference }
  - { label: Wins, value: Latency, privacy, offline use }
  - { label: Needs, value: Efficient models / accelerators }
see_also: [google-coral, nvidia-jetson, single-board-computer, ai-accelerator, edge-computing, home-automation]
related_lessons:
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Edge_computing
---

**Edge AI** is the practice of running machine-learning models directly on local devices — [single-board computers](/reference/single-board-computer/), cameras, sensors — instead of sending data to the cloud for inference.[^wiki]

## Overview

The appeal is concrete: inference next to the data is lower-latency, keeps private data on the device, and keeps working without a network. The cost is that small devices have limited compute, so edge AI leans on efficient, quantised models and on dedicated [AI accelerators](/reference/ai-accelerator/) such as [Google Coral](/reference/google-coral/)'s Edge TPU or the GPU on an [NVIDIA Jetson](/reference/nvidia-jetson/). It is a specific case of the broader move toward [edge computing](/reference/edge-computing/).

## Where it fits

Edge AI shows up wherever round-tripping to a server is too slow, too costly, or impossible: factory cameras, doorbells, robots, and [home automation](/reference/home-automation/). In a GopherTrunk-style deployment, edge AI could classify or flag activity in decoded data on the same node that does the radio work, sending up only the results rather than every sample.

## Sources

[^wiki]: [Edge computing](https://en.wikipedia.org/wiki/Edge_computing) — Wikipedia, on processing data near its source, including on-device inference.
