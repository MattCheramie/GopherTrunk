---
slug: i2c
title: I²C
entry_type: concept
category: hw-microcontrollers
description: I²C is a two-wire serial bus that lets a controller talk to many peripheral chips over a shared clock and data line, each addressed by number, widely used to connect sensors to microcontrollers.
keywords: I2C, I²C, IIC, two-wire, SDA, SCL, serial bus, sensor bus, TWI, address
aka: [I2C, IIC, TWI]
infobox:
  - { label: Type, value: Serial bus }
  - { label: Wires, value: Two (SDA, SCL) }
  - { label: Topology, value: Multi-drop, addressed }
  - { label: Speed, value: ~100 kHz – 3.4 MHz }
see_also: [spi, uart, microcontroller, sensor, gpio, embedded-system]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/I%C2%B2C
---

**I²C** (Inter-Integrated Circuit) is a two-wire serial bus that lets one controller communicate with many peripheral chips over a shared clock and data line.[^wiki]

## Overview

I²C needs just two signals: a clock (SCL) and a bidirectional data line (SDA), both open-drain with pull-up resistors. Every peripheral has a numeric address, so a single pair of wires can fan out to dozens of devices on the same bus. It is slower than [SPI](/reference/spi/) — typically 100 kHz to a few MHz — but its low pin count and addressing make it ideal for connecting many small chips. A [microcontroller's](/reference/microcontroller/) I²C peripheral handles the bit-level timing in hardware.

## Where it fits

I²C is the standard way to wire up [sensors](/reference/sensor/) — temperature, accelerometers, real-time clocks, small displays — to an MCU, because each adds no extra pins beyond the shared bus. It trades speed for simplicity versus SPI, and addressing versus a point-to-point link like [UART](/reference/uart/). It is one of the core peripheral buses in nearly every [embedded system](/reference/embedded-system/), exposed alongside the chip's [GPIO](/reference/gpio/).

## Sources

[^wiki]: [I²C](https://en.wikipedia.org/wiki/I%C2%B2C) — Wikipedia, on the two-wire bus and its addressing.
