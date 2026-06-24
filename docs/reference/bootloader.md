---
slug: bootloader
title: Bootloader
entry_type: concept
category: hw-microcontrollers
description: A bootloader is a small program that runs first at power-on and loads the main firmware or operating system; on microcontrollers it also enables reflashing without external hardware.
keywords: bootloader, boot, DFU, firmware update, reflash, USB bootloader, second stage, OTA
infobox:
  - { label: Type, value: Startup program }
  - { label: Runs, value: First, at power-on/reset }
  - { label: Job, value: Load and start firmware }
  - { label: Bonus, value: Field reprogramming }
see_also: [firmware, in-system-programming, microcontroller, embedded-system, bare-metal-programming, watchdog-timer]
related_lessons:
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bootloader
---

**A bootloader** is a small program that runs first when a device powers on or resets, then loads and starts the main [firmware](/reference/firmware/) or operating system.[^wiki]

## Overview

On a desktop the bootloader chain hands off to an OS; on a [microcontroller](/reference/microcontroller/) the bootloader is often the only thing between reset and your application. Many MCUs ship with a factory bootloader in ROM that speaks USB DFU, [UART](/reference/uart/), or [SPI](/reference/spi/)/[I²C](/reference/i2c/), so new code can be loaded without a dedicated programmer. Custom bootloaders add over-the-air (OTA) updates, integrity checks, and fallback to a known-good image if an update fails.

## Where it fits

A bootloader is what lets you reflash a board over USB instead of always using [in-system programming](/reference/in-system-programming/) with an external debugger — the Arduino "press reset, upload sketch" flow is exactly this. In a fielded [embedded system](/reference/embedded-system/) it is the safe path for firmware updates, often paired with a [watchdog timer](/reference/watchdog-timer/) so a hung update cannot brick the device.

## Sources

[^wiki]: [Bootloader](https://en.wikipedia.org/wiki/Bootloader) — Wikipedia, on bootloaders and their role at startup.
