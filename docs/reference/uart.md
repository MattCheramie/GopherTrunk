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

## Overview

Instead of a clock line, both ends agree on a **baud rate** and frame each byte with start and stop bits, so the receiver can recover timing from the data itself. The link uses just transmit (TX) and receive (RX), making it the simplest of the common serial interfaces. A [microcontroller](/reference/microcontroller/) usually has several UARTs; one is often wired to a USB-serial adapter to provide a console. Classic RS-232 ports are UARTs with higher voltage line drivers.

## Where it fits

UART is the workhorse point-to-point link: a debug/console port, a connection to a GPS or radio module, or a board-to-board serial line. Unlike the multi-drop [I²C](/reference/i2c/) and [SPI](/reference/spi/) buses, a UART connects exactly two endpoints, which keeps it dead simple. Many [bootloaders](/reference/bootloader/) accept new [firmware](/reference/firmware/) over UART, and it is a fixture of almost every [embedded system](/reference/embedded-system/).

## Sources

[^wiki]: [UART](https://en.wikipedia.org/wiki/Universal_asynchronous_receiver-transmitter) — Wikipedia, on asynchronous serial communication.
