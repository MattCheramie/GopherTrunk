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

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 190" role="img" aria-label="An I2C bus: one controller and three peripherals all share just two wires — SCL for the clock and SDA for data — with pull-up resistors to Vcc. Each peripheral has a unique address, so many chips fan out from the same pair of lines." xmlns="http://www.w3.org/2000/svg">
  <text x="150" y="14" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">Vcc · pull-ups</text>
  <line x1="132" y1="18" x2="168" y2="18" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <line x1="140" y1="18" x2="140" y2="40" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <line x1="160" y1="18" x2="160" y2="62" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <line x1="24" y1="40" x2="426" y2="40" stroke="currentColor" stroke-width="1.6"/>
  <line x1="24" y1="62" x2="426" y2="62" stroke="currentColor" stroke-width="1.6"/>
  <text x="432" y="43" font-size="9" fill="currentColor" font-weight="600">SCL</text>
  <text x="432" y="65" font-size="9" fill="currentColor" font-weight="600">SDA</text>
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <rect x="24" y="86" width="88" height="52" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <text x="68" y="108" font-weight="600">controller</text>
    <text x="68" y="123" font-size="8">(MCU)</text>
    <rect x="150" y="112" width="84" height="42" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="192" y="130">sensor</text><text x="192" y="144" font-size="8">0x1D</text>
    <rect x="252" y="112" width="84" height="42" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="294" y="130">RTC</text><text x="294" y="144" font-size="8">0x68</text>
    <rect x="354" y="112" width="72" height="42" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="390" y="130">display</text><text x="390" y="144" font-size="8">0x3C</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.6">
    <line x1="48" y1="86" x2="48" y2="40"/><line x1="88" y1="86" x2="88" y2="62"/>
    <line x1="174" y1="112" x2="174" y2="40"/><line x1="210" y1="112" x2="210" y2="62"/>
    <line x1="276" y1="112" x2="276" y2="40"/><line x1="312" y1="112" x2="312" y2="62"/>
    <line x1="378" y1="112" x2="378" y2="40"/><line x1="402" y1="112" x2="402" y2="62"/>
  </g>
  <g fill="currentColor"><circle cx="174" cy="40" r="2"/><circle cx="210" cy="62" r="2"/><circle cx="276" cy="40" r="2"/><circle cx="312" cy="62" r="2"/><circle cx="378" cy="40" r="2"/><circle cx="402" cy="62" r="2"/></g>
  <text x="235" y="178" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">two shared wires · every peripheral answers to its own address</text>
</svg>
<figcaption>I²C fans out from just two open-drain wires — a shared clock (SCL) and a bidirectional data line (SDA), both pulled up to Vcc. Because every chip has a numeric address, one pair of lines can reach dozens of devices, which is why it's the go-to bus for wiring many small sensors to an MCU.</figcaption>
</figure>

## Overview

I²C needs just two signals: a clock (SCL) and a bidirectional data line (SDA), both open-drain with pull-up resistors. Every peripheral has a numeric address, so a single pair of wires can fan out to dozens of devices on the same bus. It is slower than [SPI](/reference/spi/) — typically 100 kHz to a few MHz — but its low pin count and addressing make it ideal for connecting many small chips. A [microcontroller's](/reference/microcontroller/) I²C peripheral handles the bit-level timing in hardware.

## Where it fits

I²C is the standard way to wire up [sensors](/reference/sensor/) — temperature, accelerometers, real-time clocks, small displays — to an MCU, because each adds no extra pins beyond the shared bus. It trades speed for simplicity versus SPI, and addressing versus a point-to-point link like [UART](/reference/uart/). It is one of the core peripheral buses in nearly every [embedded system](/reference/embedded-system/), exposed alongside the chip's [GPIO](/reference/gpio/).

## Sources

[^wiki]: [I²C](https://en.wikipedia.org/wiki/I%C2%B2C) — Wikipedia, on the two-wire bus and its addressing.
