---
slug: microcontroller
title: Microcontroller (MCU)
entry_type: hardware
category: hw-microcontrollers
description: A microcontroller is a tiny, low-power computer on a single chip that combines a processor, memory, and I/O to control one device or task without an operating system.
keywords: microcontroller, MCU, embedded, bare metal, flash memory, single chip computer, low power
aka: [MCU, microcontroller]
autolink: true
infobox:
  - { label: Type, value: Single-chip computer }
  - { label: Core, value: 8/16/32-bit, often single-core }
  - { label: Memory, value: Kilobytes of RAM, on-chip flash }
  - { label: OS, value: None (bare metal) }
  - { label: Language, value: C, C++, Rust, MicroPython }
see_also: [arduino, esp32, gpio, firmware, single-board-computer, internet-of-things]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Microcontroller
---

**A microcontroller (MCU)** is a tiny, low-power computer on a single chip that combines a processor, memory, and input/output, dedicated to controlling one device or task.[^wiki]

## Overview

Unlike a full computer, a microcontroller has no operating system: your code runs bare metal as the device's [firmware](/reference/firmware/). It boots in milliseconds and can run for years on a small battery, working with only kilobytes of RAM and a modest amount of on-chip flash. That frugality is the point — an MCU does one job reliably and cheaply.

## How it differs from a single-board computer

A [single-board computer](/reference/single-board-computer/) like a Raspberry Pi runs a real OS and behaves like a small PC; a microcontroller does not. MCUs are programmed in [C](/reference/c-language/), [C++](/reference/cpp-language/), and sometimes [Rust](/reference/rust-language/) or MicroPython, then flashed onto the chip. Popular families include [Arduino](/reference/arduino/) boards and the Wi-Fi-equipped [ESP32](/reference/esp32/), and most expose their pins as [GPIO](/reference/gpio/) for wiring up sensors and radios. These are the small radios that populate the airwaves GopherTrunk listens to, even though they are far too small to run GopherTrunk itself.

## Sources

[^wiki]: [Microcontroller](https://en.wikipedia.org/wiki/Microcontroller) — Wikipedia, on the single-chip design and embedded role of MCUs.
