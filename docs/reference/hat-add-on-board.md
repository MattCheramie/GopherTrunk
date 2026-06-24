---
slug: hat-add-on-board
title: HAT add-on board
entry_type: hardware
category: hw-sbc
description: A HAT is an add-on board that stacks onto a Raspberry Pi's GPIO header to add hardware such as displays, sensors, real-time clocks, or radios, following a standard mechanical and electrical specification.
keywords: HAT, Hardware Attached on Top, Raspberry Pi HAT, GPIO add-on, daughterboard, expansion board, EEPROM ID
aka: [HAT, Hardware Attached on Top]
infobox:
  - { label: Type, value: SBC add-on board }
  - { label: Connects via, value: 40-pin GPIO header }
  - { label: Standard, value: Raspberry Pi HAT spec }
  - { label: Adds, value: Displays, sensors, radios, I/O }
  - { label: Stands for, value: Hardware Attached on Top }
see_also: [raspberry-pi, gpio, single-board-computer, compute-module, sensor, i2c]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi#HATs
---

**A HAT** (Hardware Attached on Top) is an add-on board that stacks onto a [Raspberry Pi](/reference/raspberry-pi/)'s [GPIO](/reference/gpio/) header to add hardware.[^wiki]

## Overview

The HAT specification fixes the board size, mounting holes, and 40-pin connector so add-ons fit predictably, and a small EEPROM on the HAT can identify itself so the Pi auto-configures the right pins. Typical HATs add displays, [sensors](/reference/sensor/), real-time clocks, motor drivers, power management, or radios — often talking to the Pi over [I2C](/reference/i2c/) or SPI. Other SBCs use similar stackable add-ons under different names.

## Where it fits

A HAT is the quickest way to extend a board without designing your own electronics — a middle ground before committing to a custom carrier for a [Compute Module](/reference/compute-module/). For GopherTrunk, a HAT can supply the parts a bare Pi lacks at the antenna: a real-time clock for accurate timestamps, a fan and power controller, or a GPS HAT for precise timing on a capture node.

## Sources

[^wiki]: [Raspberry Pi HATs](https://en.wikipedia.org/wiki/Raspberry_Pi#HATs) — Wikipedia, on the HAT add-on board standard.
