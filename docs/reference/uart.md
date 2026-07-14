---
slug: uart
title: UART
entry_type: concept
category: hw-microcontrollers
description: A UART is a hardware block that sends and receives asynchronous serial data over two wires without a shared clock, the classic way to give a microcontroller a simple point-to-point link or console.
keywords: UART, serial, asynchronous, baud rate, TX, RX, RS-232, serial console, USB-serial
aka: [UART, serial port]
infobox:
  - { label: Type, value: Serial interface }
  - { label: Wires, value: 2 (TX, RX) }
  - { label: Clocking, value: Asynchronous (no shared clock) }
  - { label: Set by, value: Baud rate }
see_also: [spi, i2c, microcontroller, bootloader, firmware, embedded-system]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Universal_asynchronous_receiver-transmitter
---

**A UART** (Universal Asynchronous Receiver/Transmitter) is a hardware block that sends and receives serial data over two wires without a shared clock.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 468 156" role="img" aria-label="A UART frame on one line. The line idles high, drops for one start bit, sends eight data bits least-significant first, then returns high for a stop bit. There is no clock line; both ends agree on a baud rate and the receiver recovers timing from the falling edge of the start bit." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="0.7" stroke-opacity="0.25">
    <line x1="60" y1="34" x2="60" y2="104"/><line x1="96" y1="34" x2="96" y2="104"/><line x1="132" y1="34" x2="132" y2="104"/><line x1="168" y1="34" x2="168" y2="104"/><line x1="204" y1="34" x2="204" y2="104"/><line x1="240" y1="34" x2="240" y2="104"/><line x1="276" y1="34" x2="276" y2="104"/><line x1="312" y1="34" x2="312" y2="104"/><line x1="348" y1="34" x2="348" y2="104"/><line x1="384" y1="34" x2="384" y2="104"/><line x1="420" y1="34" x2="420" y2="104"/>
  </g>
  <polyline points="24,44 60,44 60,90 96,90 96,44 132,44 132,90 168,90 168,44 204,44 240,44 240,90 276,90 276,44 312,44 312,90 348,90 348,44 384,44 420,44 452,44" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.9">
    <text x="42" y="30">idle</text>
    <text x="78" y="30">start</text>
    <text x="240" y="26">8 data bits (LSB first)</text>
    <text x="402" y="30">stop</text>
  </g>
  <line x1="96" y1="20" x2="384" y2="20" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
  <line x1="96" y1="20" x2="96" y2="26" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
  <line x1="384" y1="20" x2="384" y2="26" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
  <text x="18" y="47" text-anchor="end" font-size="8" fill="currentColor">TX</text>
  <text x="234" y="128" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">no clock line — both ends agree on a baud rate; the start bit re-syncs the receiver each byte</text>
</svg>
<figcaption>A UART has no clock wire. The line idles high; a falling edge marks the start bit, eight data bits follow (least-significant first), and a stop bit returns the line high. Because both ends are set to the same baud rate, the receiver recovers timing from that start edge — which is what makes it the simplest point-to-point serial link.</figcaption>
</figure>

## Overview

Instead of a clock line, both ends agree on a **baud rate** and frame each byte with start and stop bits, so the receiver can recover timing from the data itself. The link uses just transmit (TX) and receive (RX), making it the simplest of the common serial interfaces. A [microcontroller](/reference/microcontroller/) usually has several UARTs; one is often wired to a USB-serial adapter to provide a console. Classic RS-232 ports are UARTs with higher voltage line drivers.

## Where it fits

UART is the workhorse point-to-point link: a debug/console port, a connection to a GPS or radio module, or a board-to-board serial line. Unlike the multi-drop [I²C](/reference/i2c/) and [SPI](/reference/spi/) buses, a UART connects exactly two endpoints, which keeps it dead simple. Many [bootloaders](/reference/bootloader/) accept new [firmware](/reference/firmware/) over UART, and it is a fixture of almost every [embedded system](/reference/embedded-system/).

## Sources

[^wiki]: [UART](https://en.wikipedia.org/wiki/Universal_asynchronous_receiver-transmitter) — Wikipedia, on asynchronous serial communication.
