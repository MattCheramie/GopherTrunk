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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 208" role="img" aria-label="An SPI link between a controller and a peripheral over four lines: SCLK the clock, MOSI carrying data from controller to peripheral, MISO carrying data back at the same time (full-duplex), and CS the chip-select. Each additional peripheral needs its own chip-select line." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="42" width="104" height="104" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="72" y="90" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">SPI</text>
  <text x="72" y="104" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">controller</text>
  <rect x="336" y="42" width="104" height="104" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <text x="388" y="97" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">peripheral</text>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="124" y1="60" x2="336" y2="60" marker-end="url(#spi_ar)"/>
    <line x1="124" y1="88" x2="336" y2="88" marker-end="url(#spi_ar)"/>
    <line x1="336" y1="116" x2="124" y2="116" marker-end="url(#spi_ar)"/>
    <line x1="124" y1="140" x2="336" y2="140" marker-end="url(#spi_ar)"/>
  </g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="230" y="54">SCLK — clock</text>
    <text x="230" y="82">MOSI — controller → peripheral</text>
    <text x="230" y="110">MISO — peripheral → controller</text>
    <text x="230" y="134">CS — chip-select</text>
  </g>
  <path d="M132 70 h-8 v40 h8" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.55"/>
  <text x="108" y="93" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85" transform="rotate(-90 108 93)">full-duplex</text>
  <text x="230" y="182" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">fast &amp; full-duplex · but each extra chip needs its own CS line</text>
  <defs><marker id="spi_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SPI trades wires for speed: a shared clock (SCLK) plus two data lines that move bits both ways on every clock edge — MOSI out and MISO back at the same time, so it's full-duplex and can run at tens of MHz. The catch is the chip-select: every additional peripheral needs its own CS line rather than a shared address.</figcaption>
</figure>

## Overview

SPI uses four lines: a clock (SCLK), data out (MOSI), data in (MISO), and a chip-select (CS) per peripheral. Because data flows both directions on every clock edge, it is full-duplex and can run at tens of MHz — much faster than [I²C](/reference/i2c/). The cost is wiring: each additional peripheral needs its own chip-select line rather than an address. A [microcontroller's](/reference/microcontroller/) SPI peripheral shifts the bits in hardware.

## Where it fits

SPI is the bus of choice when speed matters: driving displays, reading high-rate [ADCs](/reference/analog-to-digital-converter/), and talking to [flash memory](/reference/flash-memory/) and SD cards. Many [sensors](/reference/sensor/) offer SPI as a faster alternative to I²C. It is also the interface behind AVR-style [in-system programming](/reference/in-system-programming/). Choosing between SPI, I²C, and [UART](/reference/uart/) is a routine design decision in any [embedded system](/reference/embedded-system/).

## Sources

[^wiki]: [Serial Peripheral Interface](https://en.wikipedia.org/wiki/Serial_Peripheral_Interface) — Wikipedia, on the SPI bus and its lines.
