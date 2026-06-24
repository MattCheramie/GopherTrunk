---
slug: rock-pi
title: Rock Pi (Radxa)
entry_type: hardware
category: hw-sbc
description: Rock Pi is a family of single-board computers from Radxa of China, built mainly on Rockchip processors and often offering NVMe, faster networking, or more RAM than a comparably priced Raspberry Pi.
keywords: Rock Pi, Radxa, Rockchip, RK3588, ROCK 5, NVMe SBC, Raspberry Pi alternative, ARM single-board computer
aka: [Radxa Rock]
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Radxa (China) }
  - { label: CPU, value: ARM (Rockchip, e.g. RK3588) }
  - { label: Runs, value: Linux / Android }
  - { label: Known for, value: NVMe, fast I/O, high RAM }
see_also: [raspberry-pi, single-board-computer, odroid, orange-pi, banana-pi, nvme]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radxa
---

**Rock Pi** is a family of [single-board computers](/reference/single-board-computer/) from Radxa of China, built mainly on Rockchip chips and often offering faster I/O than a similarly priced [Raspberry Pi](/reference/raspberry-pi/).[^wiki]

## Overview

The line is built around Rockchip SoCs — the powerful RK3588 powers the higher-end ROCK 5 boards — and frequently adds features the Pi lacks at the same price, such as an [NVMe](/reference/nvme/) slot, faster Ethernet, or more RAM. Boards run Linux or Android and keep a Pi-style [GPIO](/reference/gpio/) header. As with most Pi alternatives, software polish trails the Raspberry Pi.

## Where it fits

Rock Pi competes with [ODROID](/reference/odroid/), [Orange Pi](/reference/orange-pi/), and [Banana Pi](/reference/banana-pi/), and is a natural pick when fast local storage matters. A GopherTrunk node that records raw IQ or keeps days of decoded calls benefits from NVMe rather than an SD card, which is where a Rock Pi can edge out a stock Pi.

## Sources

[^wiki]: [Radxa](https://en.wikipedia.org/wiki/Radxa) — Wikipedia, on Radxa and its Rock Pi single-board computers.
