---
slug: stm32
title: STM32
entry_type: hardware
category: hw-microcontrollers
description: STM32 is STMicroelectronics' broad family of 32-bit ARM Cortex-M microcontrollers, spanning ultra-low-power to high-performance parts widely used in industrial, consumer, and hobby embedded designs.
keywords: STM32, STMicroelectronics, ARM Cortex-M, STM32F4, STM32 Nucleo, Blue Pill, HAL, STM32CubeIDE
aka: [STM32]
autolink: true
infobox:
  - { label: Type, value: 32-bit microcontroller family }
  - { label: Core, value: ARM Cortex-M (M0 to M7/M33) }
  - { label: Vendor, value: STMicroelectronics }
  - { label: Tooling, value: STM32Cube, HAL, ST-LINK }
  - { label: Language, value: C, C++, Rust }
see_also: [arm-cortex-m, microcontroller, rp2040, avr-atmega, bare-metal-programming, firmware]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/STM32
---

**STM32** is STMicroelectronics' broad family of 32-bit [microcontrollers](/reference/microcontroller/) built around [ARM Cortex-M](/reference/arm-cortex-m/) cores.[^wiki]

## Overview

The line spans dozens of series, from the tiny low-power STM32L0 and value-line STM32F0 up to the high-performance STM32F7 and STM32H7. All share a common peripheral set — [UART](/reference/uart/), [SPI](/reference/spi/), [I²C](/reference/i2c/), timers with [PWM](/reference/pulse-width-modulation/), and on-chip [ADCs](/reference/analog-to-digital-converter/) — and a shared HAL (hardware abstraction layer) so code ports across the family. Parts are programmed in [C](/reference/c-language/), [C++](/reference/cpp-language/), or [Rust](/reference/rust-language/) and flashed over the ST-LINK debugger using [in-system programming](/reference/in-system-programming/).

## Where it fits

STM32 is the default choice when a project outgrows an 8-bit [AVR/ATmega](/reference/avr-atmega/) but does not need a full [single-board computer](/reference/single-board-computer/). The cheap "Blue Pill" board and the official Nucleo and Discovery kits make it popular with hobbyists, while its rich peripherals and long supply lifetimes keep it common in industrial and automotive products. Many designs run [bare metal](/reference/bare-metal-programming/) or under a small [RTOS](/reference/real-time-operating-system/).

## Sources

[^wiki]: [STM32](https://en.wikipedia.org/wiki/STM32) — Wikipedia, on the STM32 family, cores, and tooling.
