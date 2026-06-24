---
slug: libre-computer
title: Libre Computer
entry_type: hardware
category: hw-sbc
description: Libre Computer is a maker of single-board computers that emphasise open, mainline software support and Raspberry Pi compatibility, with boards such as Le Potato and Renegade built on Amlogic and Rockchip chips.
keywords: Libre Computer, Le Potato, AML-S905X-CC, Renegade, open source SBC, mainline Linux, Raspberry Pi compatible, ARM single-board computer
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Emphasis, value: Open, mainline software }
  - { label: CPU, value: ARM (Amlogic / Rockchip) }
  - { label: Runs, value: Mainline Linux, Android }
  - { label: Boards, value: Le Potato, Renegade, others }
see_also: [raspberry-pi, single-board-computer, orange-pi, odroid, rock-pi, gpio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Libre_Computer_Project
  - https://libre.computer/
---

**Libre Computer** is a maker of [single-board computers](/reference/single-board-computer/) that emphasise open, mainline software support and broad [Raspberry Pi](/reference/raspberry-pi/) compatibility.[^wiki]

## Overview

Boards such as Le Potato (AML-S905X-CC) and the Renegade use Amlogic and Rockchip ARM chips and copy the Pi's footprint and 40-pin [GPIO](/reference/gpio/) header. The project's distinguishing goal is upstream support: getting drivers into the mainline Linux kernel and standard bootloaders so the hardware keeps working with ordinary, current software rather than a vendor's frozen image.[^libre]

## Where it fits

Among Pi alternatives like [Orange Pi](/reference/orange-pi/), [ODROID](/reference/odroid/), and [Rock Pi](/reference/rock-pi/), Libre Computer's pitch is longevity and trust in the software stack rather than raw specs. For an always-on GopherTrunk capture node you intend to leave in place for years, mainline support means OS updates are less likely to strand the board.

## Sources

[^wiki]: [Libre Computer Project](https://en.wikipedia.org/wiki/Libre_Computer_Project) — Wikipedia, on the project and its boards.
[^libre]: [Libre Computer](https://libre.computer/) — vendor site, on the boards and their open-software focus.
