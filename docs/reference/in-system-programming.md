---
slug: in-system-programming
title: In-system programming
entry_type: concept
category: hw-microcontrollers
description: In-system programming (ISP) flashes a microcontroller's firmware while the chip stays soldered in its circuit, using a debug interface such as SWD, JTAG, or SPI instead of a chip programmer.
keywords: in-system programming, ISP, ICSP, SWD, JTAG, debug interface, flash, programmer, ST-LINK
aka: [ISP, ICSP]
infobox:
  - { label: Type, value: Programming method }
  - { label: Means, value: Flash chip in-place }
  - { label: Interfaces, value: SWD, JTAG, SPI }
  - { label: Tools, value: ST-LINK, PICkit, J-Link }
see_also: [bootloader, firmware, microcontroller, spi, avr-atmega, stm32]
related_lessons:
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/In-system_programming
---

**In-system programming (ISP)** is the practice of writing a [microcontroller's](/reference/microcontroller/) [firmware](/reference/firmware/) while the chip remains soldered into its circuit board.[^wiki]

## Overview

Before ISP, a chip had to be removed and placed in a dedicated programmer. ISP instead uses a small debug interface on the board — SWD or JTAG on [ARM Cortex-M](/reference/arm-cortex-m/) parts, an [SPI](/reference/spi/)-based ICSP header on [AVR/ATmega](/reference/avr-atmega/), or vendor schemes on [PIC](/reference/pic-microcontroller/) — so a programmer like ST-LINK, J-Link, or PICkit can flash and debug the device in place. The same interface usually allows single-step debugging and reading back memory.

## Where it fits

ISP is the lowest-level way to get code onto a chip and is how a brand-new, blank MCU is first programmed — including installing a [bootloader](/reference/bootloader/), after which routine updates can happen over USB instead. On an [STM32](/reference/stm32/) board, the SWD pins broken out next to the MCU are the ISP interface. It is the standard production and bring-up workflow for [embedded systems](/reference/embedded-system/).

## Sources

[^wiki]: [In-system programming](https://en.wikipedia.org/wiki/In-system_programming) — Wikipedia, on programming chips in place.
