---
slug: beaglebone
title: BeagleBone
entry_type: hardware
category: hw-sbc
description: BeagleBone is an open-source single-board computer known for strong real-time I/O, with many GPIO pins and onboard programmable real-time units, favored for industrial control.
keywords: BeagleBone, BeagleBone Black, PRU, programmable real-time unit, real-time I/O, industrial control, open-source SBC, GPIO
autolink: true
infobox:
  - { label: Type, value: Single-board computer }
  - { label: CPU, value: ARM (TI SoC) + PRUs }
  - { label: RAM, value: ~512 MB – 4 GB }
  - { label: Runs, value: Linux }
  - { label: Typical price, value: ~$50 – $150 }
see_also: [single-board-computer, gpio, raspberry-pi, nvidia-jetson, input-output, microcontroller]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/BeagleBoard#BeagleBone
---

**BeagleBone** is an open-source [single-board computer](/reference/single-board-computer/) known for strong real-time I/O — many [GPIO](/reference/gpio/) pins and onboard programmable real-time units (PRUs).[^wiki]

## Overview

The PRUs are small, deterministic processors alongside the main ARM CPU, which lets a BeagleBone handle precise, timing-critical signalling that a general-purpose Linux board struggles with. Combined with its generous pin count, this makes it a favourite for industrial control and electronics-heavy projects. It runs Linux like other SBCs.

## Where it fits

The BeagleBone is the SBC alternative to the [Raspberry Pi](/reference/raspberry-pi/) when [I/O](/reference/input-output/) and determinism matter more than raw cost or community size. For general-purpose use a Pi is simpler; for GPU work at the edge see the [NVIDIA Jetson](/reference/nvidia-jetson/).

## Sources

[^wiki]: [BeagleBone](https://en.wikipedia.org/wiki/BeagleBoard#BeagleBone) — Wikipedia, on the BeagleBoard family and its real-time I/O.
