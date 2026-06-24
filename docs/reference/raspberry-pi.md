---
slug: raspberry-pi
title: Raspberry Pi
entry_type: hardware
category: hw-sbc
description: Raspberry Pi is a popular, low-cost single-board computer used for learning, hobby projects, home servers, and edge devices, running Linux and programmed in Python, C, or Go.
keywords: Raspberry Pi, Pi Zero 2 W, Pi 4, Pi 5, Compute Module, HAT, Raspberry Pi OS, single-board computer
autolink: true
infobox:
  - { label: Type, value: Single-board computer }
  - { label: CPU, value: ARM (Broadcom SoC) }
  - { label: RAM, value: ~512 MB – 16 GB }
  - { label: Runs, value: Raspberry Pi OS / Linux }
  - { label: Typical price, value: ~$15 – $80 }
see_also: [single-board-computer, gpio, nvidia-jetson, beaglebone, home-server, software-defined-radio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi
---

**Raspberry Pi** is the popular, low-cost [single-board computer](/reference/single-board-computer/) that defined the category — used for learning, hobby projects, home servers, and edge devices.[^wiki]

## Overview

The range runs from the tiny Pi Zero 2 W through the Pi 4 and Pi 5 to the Compute Module for embedding in custom hardware. A Raspberry Pi runs Raspberry Pi OS (a Linux distribution) and is programmed in ordinary languages such as [Python](/reference/python-language/), [C](/reference/c-language/), and [Go](/reference/go-language/). A *HAT* is an add-on board that stacks onto its [GPIO](/reference/gpio/) header to add hardware.

## Where it fits

For most projects the Pi is the default choice: cheap, well documented, and broadly supported. A Raspberry Pi by the antenna can run GopherTrunk as a small, low-power SDR capture node. When you need GPU compute at the edge, the [NVIDIA Jetson](/reference/nvidia-jetson/) is an alternative; when you need stronger real-time I/O, look at the [BeagleBone](/reference/beaglebone/).

## Sources

[^wiki]: [Raspberry Pi](https://en.wikipedia.org/wiki/Raspberry_Pi) — Wikipedia, on the models and uses of the Raspberry Pi.
