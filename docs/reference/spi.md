---
slug: spi
title: SPI
entry_type: concept
category: hw-microcontrollers
description: SPI is a fast, full-duplex serial bus that connects a controller to peripheral chips using separate clock, data-out, data-in, and chip-select lines, common for displays, flash, and ADCs.
keywords: SPI, serial peripheral interface, MOSI, MISO, SCLK, chip select, full duplex, flash, display
aka: [SPI]
infobox:
  - { label: Type, value: Serial bus }
  - { label: Wires, value: 4 (SCLK, MOSI, MISO, CS) }
  - { label: Mode, value: Full-duplex }
  - { label: Speed, value: Tens of MHz }
see_also: [i2c, uart, microcontroller, sensor, in-system-programming, flash-memory]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Serial_Peripheral_Interface
---

**SPI** (Serial Peripheral Interface) is a fast, full-duplex serial bus that connects a controller to one or more peripheral chips.[^wiki]

## Overview

SPI uses four lines: a clock (SCLK), data out (MOSI), data in (MISO), and a chip-select (CS) per peripheral. Because data flows both directions on every clock edge, it is full-duplex and can run at tens of MHz — much faster than [I²C](/reference/i2c/). The cost is wiring: each additional peripheral needs its own chip-select line rather than an address. A [microcontroller's](/reference/microcontroller/) SPI peripheral shifts the bits in hardware.

## Where it fits

SPI is the bus of choice when speed matters: driving displays, reading high-rate [ADCs](/reference/analog-to-digital-converter/), and talking to [flash memory](/reference/flash-memory/) and SD cards. Many [sensors](/reference/sensor/) offer SPI as a faster alternative to I²C. It is also the interface behind AVR-style [in-system programming](/reference/in-system-programming/). Choosing between SPI, I²C, and [UART](/reference/uart/) is a routine design decision in any [embedded system](/reference/embedded-system/).

## Sources

[^wiki]: [Serial Peripheral Interface](https://en.wikipedia.org/wiki/Serial_Peripheral_Interface) — Wikipedia, on the SPI bus and its lines.
