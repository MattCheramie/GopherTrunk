---
slug: orange-pi
title: Orange Pi
entry_type: hardware
category: hw-sbc
description: Orange Pi is a family of low-cost single-board computers from China that mimic the Raspberry Pi form factor, often offering more cores or ports per dollar with less mature software support.
keywords: Orange Pi, Allwinner, Rockchip, Raspberry Pi alternative, low-cost SBC, ARM single-board computer
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Shenzhen Xunlong (China) }
  - { label: CPU, value: ARM (Allwinner / Rockchip) }
  - { label: Runs, value: Linux / Android }
  - { label: Typical price, value: ~$15 – $90 }
see_also: [raspberry-pi, single-board-computer, banana-pi, rock-pi, libre-computer, gpio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Orange_Pi
---

**Orange Pi** is a family of low-cost [single-board computers](/reference/single-board-computer/) made in China that copy the [Raspberry Pi](/reference/raspberry-pi/) form factor while often packing more cores or ports per dollar.[^wiki]

## Overview

Orange Pi boards are built around Allwinner and Rockchip ARM chips and run Linux or Android. Many keep a Pi-style layout and a compatible 40-pin [GPIO](/reference/gpio/) header, so existing add-ons and cases often fit. The catch is software: drivers and community support are usually less mature than the Raspberry Pi's, so the same hardware can be more work to get fully running.

## Where it fits

Orange Pi sits alongside [Banana Pi](/reference/banana-pi/), [Rock Pi](/reference/rock-pi/), and [Libre Computer](/reference/libre-computer/) as a Raspberry Pi alternative chosen on price or specs. For a GopherTrunk capture node where you control the OS image and just need a cheap Linux box near the antenna, an Orange Pi can be a good value — provided you accept the extra setup time over a Pi.

## Sources

[^wiki]: [Orange Pi](https://en.wikipedia.org/wiki/Orange_Pi) — Wikipedia, on the Orange Pi line of single-board computers.
