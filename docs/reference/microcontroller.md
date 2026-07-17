---
slug: microcontroller
title: Microcontroller (MCU)
entry_type: hardware
category: hw-microcontrollers
description: A microcontroller is a tiny, low-power computer on a single chip that combines a processor, memory, and I/O to control one device or task without an operating system, running its program as firmware straight from on-chip flash.
keywords: microcontroller, MCU, embedded, bare metal, flash memory, single chip computer, low power, GPIO, on-chip peripherals
aka: [MCU, microcontroller]
autolink: true
infobox:
  - { label: Type, value: Single-chip computer }
  - { label: Core, value: 8/16/32-bit, often single-core }
  - { label: Memory, value: Kilobytes of RAM, on-chip flash }
  - { label: OS, value: None (bare metal) }
  - { label: Language, value: C, C++, Rust, MicroPython }
see_also: [arduino, esp32, gpio, firmware, single-board-computer, sensor]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Microcontroller
---

**A microcontroller (MCU)** is a tiny, low-power computer on a single chip that combines a processor, memory, and input/output, dedicated to controlling one device or task.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="Block diagram inside a microcontroller chip. A central processor core connects over one internal bus to on-chip flash holding the program and to SRAM for working data. The same bus links a set of built-in peripherals: general-purpose I/O pins, timers with pulse-width modulation, an analog-to-digital converter, and serial interfaces. Everything is on one piece of silicon." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="30" width="400" height="120" rx="6" stroke-dasharray="4 3" stroke-opacity="0.7"/>
    <rect x="60" y="52" width="86" height="40" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="60" y="104" width="86" height="26" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="170" y="104" width="86" height="26" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="280" y="46" width="66" height="24" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="356" y="46" width="66" height="24" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="280" y="104" width="66" height="26" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="356" y="104" width="66" height="26" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <line x1="70" y1="98" x2="410" y2="98" stroke-width="1.6"/>
    <line x1="103" y1="92" x2="103" y2="98"/>
    <line x1="103" y1="98" x2="103" y2="104"/>
    <line x1="213" y1="98" x2="213" y2="104"/>
    <line x1="313" y1="70" x2="313" y2="98"/>
    <line x1="389" y1="70" x2="389" y2="98"/>
    <line x1="313" y1="98" x2="313" y2="104"/>
    <line x1="389" y1="98" x2="389" y2="104"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="103" y="76" font-size="9" font-weight="600">CPU core</text>
    <text x="103" y="121" font-size="7.5">Flash</text>
    <text x="213" y="121" font-size="7.5">SRAM</text>
    <text x="313" y="61" font-size="7.5">GPIO</text>
    <text x="389" y="61" font-size="7.5">Timers/PWM</text>
    <text x="313" y="121" font-size="7.5">ADC</text>
    <text x="389" y="121" font-size="7.5">UART/SPI/I&#178;C</text>
    <text x="245" y="92" font-size="7.5" fill-opacity="0.85">internal bus</text>
    <text x="230" y="166" font-size="8" fill-opacity="0.9">all on one chip &#183; no external memory or OS needed</text>
  </g>
</svg>
<figcaption>Inside a microcontroller: a CPU core, program flash, and SRAM, plus built-in peripherals — GPIO, timers, an ADC, and serial ports — all sharing one internal bus on a single chip. That integration is the whole point: a complete little computer that needs no external memory and no operating system.</figcaption>
</figure>

## Overview

Unlike a full computer, a microcontroller has no operating system: your code runs bare metal as the device's [firmware](/reference/firmware/). It boots in milliseconds and can run for years on a small battery, working with only kilobytes of RAM and a modest amount of on-chip flash. That frugality is the point — an MCU does one job reliably and cheaply.

The defining trait is *integration*: processor, memory, and I/O all live on one chip. Where a desktop CPU needs external RAM, storage, and support chips, a microcontroller packs enough of each to stand alone, exposing its pins as [GPIO](/reference/gpio/), an [ADC](/reference/analog-to-digital-converter/), timers, and serial buses for wiring up the outside world.

## How it differs from a single-board computer

A [single-board computer](/reference/single-board-computer/) like a Raspberry Pi runs a real OS and behaves like a small PC; a microcontroller does not:

| Aspect | Microcontroller | Single-board computer |
|--------|-----------------|-----------------------|
| Memory | Kilobytes of RAM | Gigabytes |
| Storage | On-chip flash | SD card / eMMC |
| Software | Firmware, bare metal | Full OS (Linux) |
| Boot | Milliseconds | Seconds |
| Power | Milliwatts | Watts |

MCUs are programmed in [C](/reference/c-language/), [C++](/reference/cpp-language/), and sometimes [Rust](/reference/rust-language/) or MicroPython, then flashed onto the chip. Popular families include [Arduino](/reference/arduino/) boards and the Wi-Fi-equipped [ESP32](/reference/esp32/).

## Where it fits

Microcontrollers are the small radios and controllers that populate the airwaves GopherTrunk listens to, even though they are far too small to run GopherTrunk itself. Around an SDR rig they earn their keep as helpers — reading a [sensor](/reference/sensor/), switching an antenna relay, or driving a status display over GPIO — leaving the demodulation and decoding to a general-purpose host.

## Sources

[^wiki]: [Microcontroller](https://en.wikipedia.org/wiki/Microcontroller) — Wikipedia, on the single-chip design and embedded role of MCUs.
