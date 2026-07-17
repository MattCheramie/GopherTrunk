---
slug: edge-ai
title: Edge AI
entry_type: concept
category: hw-sbc
description: Edge AI is the practice of running machine-learning models directly on local devices such as single-board computers, cameras, and sensors, rather than sending data to the cloud for inference.
keywords: edge AI, on-device inference, edge inference, machine learning at the edge, AI accelerator, Edge TPU, low latency AI, privacy, quantized models
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two inference paths compared. In the cloud path a sensor sends raw data over the network to a distant server and waits for a round trip. In the edge path the sensor feeds a local accelerator on the same device, which runs the model and produces a result immediately, sending only the small result onward." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="24" y="20" width="52" height="30" rx="4" fill-opacity="0.08" fill="currentColor"/>
    <path d="M76 35 H150" stroke-dasharray="4 3"/>
    <rect x="150" y="18" width="60" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <path d="M210 35 H284" stroke-dasharray="4 3"/>
    <rect x="284" y="20" width="60" height="30" rx="4" fill-opacity="0.08" fill="currentColor"/>
    <rect x="24" y="98" width="52" height="30" rx="4" fill-opacity="0.08" fill="currentColor"/>
    <path d="M76 113 H150"/>
    <rect x="150" y="96" width="70" height="34" rx="4" fill-opacity="0.16" fill="currentColor"/>
    <path d="M220 113 H284"/>
    <rect x="284" y="98" width="60" height="30" rx="4" fill-opacity="0.08" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="50" y="39" font-size="7.5">sensor</text>
    <text x="180" y="33" font-size="7.5">cloud</text>
    <text x="180" y="45" font-size="7.5">server</text>
    <text x="314" y="39" font-size="7.5">result</text>
    <text x="113" y="30" font-size="7" fill-opacity="0.85">network</text>
    <text x="380" y="39" text-anchor="start" font-size="7.5" fill-opacity="0.9">round trip · latency</text>
    <text x="50" y="117" font-size="7.5">sensor</text>
    <text x="185" y="113" font-size="8" font-weight="600">accelerator</text>
    <text x="185" y="124" font-size="7">on-device</text>
    <text x="314" y="117" font-size="7.5">result</text>
    <text x="380" y="117" text-anchor="start" font-size="7.5" fill-opacity="0.9">local · instant</text>
    <text x="12" y="35" text-anchor="end" font-size="8" font-weight="600" transform="rotate(-90 12 35)">cloud</text>
    <text x="12" y="113" text-anchor="end" font-size="8" font-weight="600" transform="rotate(-90 12 113)">edge</text>
  </g>
</svg>
<figcaption>Cloud inference ships raw data to a distant server and waits for the round trip; edge AI runs the model on an accelerator right next to the sensor, returning a result immediately and sending only the small answer onward — the source of its latency, privacy, and offline advantages.</figcaption>
</figure>

## Overview

The appeal is concrete: inference next to the data is lower-latency, keeps private data on the device, and keeps working without a network. A doorbell that recognises a face locally answers in milliseconds and never uploads the video; a cloud version has to survive a round trip and trust a remote server with the footage. Bandwidth costs drop too, because only the small result travels rather than a continuous raw stream.

The cost is that small devices have limited compute, so edge AI leans on efficient, quantised models and on dedicated [AI accelerators](/reference/ai-accelerator/). Rather than run a full-precision network on a general CPU, edge deployments shrink the model — lower numeric precision, pruning, distillation — and hand the heavy math to purpose-built silicon like [Google Coral](/reference/google-coral/)'s Edge TPU or the GPU on an [NVIDIA Jetson](/reference/nvidia-jetson/). It is a specific case of the broader move toward [edge computing](/reference/edge-computing/).

## Edge vs cloud inference

| | Cloud inference | Edge AI |
|---|-----------------|---------|
| Latency | Network round trip | Immediate, local |
| Privacy | Data leaves the device | Data stays put |
| Offline | Fails without a link | Keeps working |
| Compute available | Effectively unlimited | Small, power-bound |
| Model size | Large, full precision | Quantised, compact |
| Ongoing cost | Bandwidth + server time | One-time hardware |

## Where it fits

Edge AI shows up wherever round-tripping to a server is too slow, too costly, or impossible: factory cameras, doorbells, robots, and [home automation](/reference/home-automation/) hubs. In a GopherTrunk-style deployment, edge AI could classify or flag activity in decoded data on the same node that does the radio work — spotting a signal of interest, sorting traffic, or triggering a recording — and send up only the results rather than every sample, keeping a remote capture node useful even on a thin link.

## Sources

[^wiki]: [Edge computing](https://en.wikipedia.org/wiki/Edge_computing) — Wikipedia, on processing data near its source, including on-device inference.
