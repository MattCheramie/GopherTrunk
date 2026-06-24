---
slug: nvidia-jetson
title: NVIDIA Jetson
entry_type: hardware
category: hw-sbc
description: NVIDIA Jetson is a family of single-board computers with a powerful onboard GPU, aimed at on-device AI and computer vision such as edge inference and robotics.
keywords: NVIDIA Jetson, Jetson Nano, Jetson Orin, edge AI, GPU SBC, computer vision, edge inference, robotics
autolink: true
infobox:
  - { label: Type, value: Single-board computer (GPU) }
  - { label: CPU, value: ARM + NVIDIA GPU }
  - { label: RAM, value: ~4 GB – 64 GB }
  - { label: Runs, value: Linux (JetPack) }
  - { label: Typical price, value: ~$100 – $2000+ }
see_also: [single-board-computer, raspberry-pi, beaglebone, gpio, central-processing-unit, cloud-computing]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Nvidia_Jetson
---

**NVIDIA Jetson** is a family of [single-board computers](/reference/single-board-computer/) with a powerful onboard GPU, aimed at on-device AI and computer vision.[^wiki]

## Overview

Where a general-purpose board leans on its [CPU](/reference/central-processing-unit/), a Jetson pairs an ARM CPU with an NVIDIA GPU so that machine-learning inference can run locally — useful for robotics, cameras, and other edge devices that cannot rely on the cloud. Jetsons run Linux (NVIDIA's JetPack distribution).

## Where it fits

A Jetson is more expensive and more power-hungry than a [Raspberry Pi](/reference/raspberry-pi/), so it is the SBC you reach for specifically when you need GPU compute at the edge rather than a general-purpose board. For lighter, always-on roles a Pi is usually the better fit.

## Sources

[^wiki]: [Nvidia Jetson](https://en.wikipedia.org/wiki/Nvidia_Jetson) — Wikipedia, on the Jetson family and its edge-AI focus.
