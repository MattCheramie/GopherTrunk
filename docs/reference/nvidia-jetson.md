---
slug: nvidia-jetson
title: NVIDIA Jetson
entry_type: hardware
category: hw-sbc
description: NVIDIA Jetson is a family of single-board computers with a powerful onboard GPU, aimed at on-device AI and computer vision such as edge inference and robotics.
keywords: NVIDIA Jetson, Jetson Nano, Jetson Orin, edge AI, GPU SBC, computer vision, edge inference, robotics, CUDA, JetPack
autolink: true
infobox:
  - { label: Type, value: Single-board computer (GPU) }
  - { label: CPU, value: ARM + NVIDIA GPU }
  - { label: RAM, value: ~4 GB – 64 GB }
  - { label: Runs, value: Linux (JetPack) }
  - { label: Typical price, value: ~$100 – $2000+ }
see_also: [single-board-computer, raspberry-pi, beaglebone, gpio, central-processing-unit, edge-ai]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Nvidia_Jetson
---

**NVIDIA Jetson** is a family of [single-board computers](/reference/single-board-computer/) with a powerful onboard GPU, aimed at on-device AI and computer vision.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 156" role="img" aria-label="An edge inference pipeline on a Jetson. A camera feeds frames into the module, where an ARM CPU handles the operating system and a large array of GPU cores runs the neural-network math in parallel; the result — a detection or classification — comes straight back out on-device, with no cloud round trip." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="26" y="56" width="46" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <path d="M72 73 H104"/>
    <rect x="104" y="30" width="220" height="90" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="120" y="48" width="60" height="54" rx="4" fill-opacity="0.12" fill="currentColor"/>
    <g stroke-width="0.9">
      <rect x="196" y="48" width="112" height="54" rx="4" fill-opacity="0.05" fill="currentColor"/>
      <rect x="204" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="220" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="236" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="252" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="268" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="284" y="56" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="204" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="220" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="236" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="252" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="268" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="284" y="72" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="204" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="220" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="236" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="252" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="268" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
      <rect x="284" y="88" width="10" height="10" rx="1.5" fill-opacity="0.35" fill="currentColor"/>
    </g>
    <path d="M324 73 H356"/>
    <rect x="356" y="56" width="80" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="49" y="77" font-size="7.5">camera</text>
    <text x="150" y="79" font-size="8" font-weight="600">ARM</text>
    <text x="150" y="90" font-size="6.5">OS</text>
    <text x="252" y="40" font-size="8" font-weight="600">GPU cores</text>
    <text x="396" y="72" font-size="7.5">detection</text>
    <text x="396" y="84" font-size="7">on-device</text>
    <text x="230" y="140" font-size="7.5" fill-opacity="0.9">CPU + many parallel GPU cores run inference locally — no cloud</text>
  </g>
</svg>
<figcaption>A Jetson pairs a conventional ARM CPU with a large array of GPU cores; frames from a camera are classified by the neural network running across those cores and the result comes straight back on-device, without a cloud round trip.</figcaption>
</figure>

## Overview

Where a general-purpose board leans on its [CPU](/reference/central-processing-unit/), a Jetson pairs an ARM CPU with an NVIDIA GPU so that machine-learning inference can run locally. Neural-network math is overwhelmingly parallel — the same operation across thousands of values — which is precisely what a GPU's many cores are built for, so a model that would crawl on a CPU runs in real time on the Jetson's silicon. This makes it a fit for robotics, cameras, and other edge devices that cannot rely on the cloud.

Jetsons run Linux through NVIDIA's JetPack distribution, which bundles the CUDA and TensorRT libraries that let frameworks reach the GPU. The family spans a wide range, from the modest Jetson Nano to the far more capable Orin modules, so "a Jetson" can mean anything from a hobby board to a several-thousand-dollar module driving an autonomous machine.

## How it compares

| | [Raspberry Pi](/reference/raspberry-pi/) | NVIDIA Jetson | [Google Coral](/reference/google-coral/) |
|---|-------------|---------------|--------|
| Accelerator | None (CPU only) | CUDA GPU | Edge TPU |
| ML flexibility | Light CPU inference | Broad, general GPU | Quantised TF Lite only |
| Power draw | ~2–8 W | ~5–60 W | ~2 W |
| Price | ~$15–80 | ~$100–2000+ | ~$60–150 |
| Best role | General-purpose node | Heavy edge AI / vision | Cheap fixed inference |

## Where it fits

A Jetson is more expensive and more power-hungry than a [Raspberry Pi](/reference/raspberry-pi/), so it is the SBC you reach for specifically when you need GPU compute at the edge — [edge AI](/reference/edge-ai/), computer vision, robotics — rather than a general-purpose board. For lighter, always-on roles a Pi is usually the better fit; when real-time I/O matters more than compute, look at the [BeagleBone](/reference/beaglebone/). In a GopherTrunk context a Jetson is overkill for plain decoding, but its GPU could accelerate signal-classification or machine-learning analysis of decoded traffic on the same node.

## Sources

[^wiki]: [Nvidia Jetson](https://en.wikipedia.org/wiki/Nvidia_Jetson) — Wikipedia, on the Jetson family and its edge-AI focus.
