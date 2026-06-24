---
slug: odroid
title: ODROID
entry_type: hardware
category: hw-sbc
description: ODROID is a line of single-board computers from Hardkernel of South Korea, known for higher performance than the Raspberry Pi and unusual models such as big.LITTLE and x86 boards.
keywords: ODROID, Hardkernel, ODROID-N2, ODROID-XU4, Amlogic, big.LITTLE, high-performance SBC, Raspberry Pi alternative
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Hardkernel (South Korea) }
  - { label: CPU, value: ARM (Amlogic) or x86 }
  - { label: Runs, value: Linux / Android }
  - { label: Known for, value: Performance per board }
see_also: [raspberry-pi, single-board-computer, rock-pi, orange-pi, banana-pi, gpio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/ODROID
---

**ODROID** is a line of [single-board computers](/reference/single-board-computer/) from Hardkernel of South Korea, known for offering more performance than the [Raspberry Pi](/reference/raspberry-pi/) and for some unusual designs.[^wiki]

## Overview

ODROID boards mostly use Amlogic ARM chips, and over the years the line has included big.LITTLE designs (the ODROID-XU4), strong general-purpose boards (the ODROID-N2+), and even x86 models. Most run Linux or Android and expose [GPIO](/reference/gpio/) headers, though pinouts and accessories differ from the Pi's, so cases and add-ons are less universally compatible.

## Where it fits

ODROID is the alternative you reach for when a [Raspberry Pi](/reference/raspberry-pi/) is not fast enough but a [Jetson](/reference/nvidia-jetson/) is overkill — more CPU and memory bandwidth without a price jump into GPU territory. It competes with [Rock Pi](/reference/rock-pi/) and [Orange Pi](/reference/orange-pi/). For a GopherTrunk node decoding several busy trunked systems at once, an ODROID's extra cores can keep the demodulators fed where a smaller board would fall behind.

## Sources

[^wiki]: [ODROID](https://en.wikipedia.org/wiki/ODROID) — Wikipedia, on Hardkernel's ODROID single-board computers.
