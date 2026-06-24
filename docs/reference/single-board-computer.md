---
slug: single-board-computer
title: Single-board computer (SBC)
entry_type: hardware
category: hw-sbc
description: A single-board computer (SBC) is a complete computer built on one small circuit board, with CPU, memory, storage, and ports, capable of running a full operating system.
keywords: single-board computer, SBC, Raspberry Pi, embedded Linux, GPIO, edge device, low power computer
aka: [SBC]
autolink: true
infobox:
  - { label: Type, value: Complete computer on one board }
  - { label: CPU, value: Usually ARM (sometimes x86) }
  - { label: RAM, value: ~512 MB – 16 GB }
  - { label: Runs, value: Linux (full OS) }
  - { label: Typical price, value: ~$15 – $100 }
see_also: [raspberry-pi, nvidia-jetson, beaglebone, gpio, microcontroller, personal-computer]
related_lessons:
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Single-board_computer
---

**A single-board computer (SBC)** is a complete computer built on one small circuit board — [CPU](/reference/central-processing-unit/), [memory](/reference/random-access-memory/), [storage](/reference/data-storage/), and ports — capable of running a full [operating system](/reference/operating-system/), usually Linux.[^wiki]

## Overview

An SBC sits between a [personal computer](/reference/personal-computer/) and a [microcontroller](/reference/microcontroller/). Unlike a microcontroller, it runs a real OS and ordinary languages and tools; unlike a sealed PC or phone, it exposes [GPIO](/reference/gpio/) pins that let your code talk directly to electronics. Most are credit-card sized and draw only a few watts.

## Where it fits

The category is broad: the [Raspberry Pi](/reference/raspberry-pi/) is the best-known general-purpose board, the [NVIDIA Jetson](/reference/nvidia-jetson/) adds a GPU for edge AI, and the [BeagleBone](/reference/beaglebone/) emphasises real-time I/O. Their low power and small size make them well suited to always-on, embedded, and field roles — for example, a Raspberry Pi by the antenna can run GopherTrunk as a small, low-power SDR capture node.

## Sources

[^wiki]: [Single-board computer](https://en.wikipedia.org/wiki/Single-board_computer) — Wikipedia, on the definition and scope of SBCs.
