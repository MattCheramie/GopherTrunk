---
slug: stm32
title: STM32
entry_type: hardware
category: hw-microcontrollers
description: STM32 is STMicroelectronics' broad family of 32-bit ARM Cortex-M microcontrollers, spanning ultra-low-power to high-performance parts widely used in industrial, consumer, and hobby embedded designs, all sharing one HAL and toolchain.
keywords: STM32, STMicroelectronics, ARM Cortex-M, STM32F4, STM32H7, STM32L0, STM32 Nucleo, Blue Pill, HAL, STM32CubeIDE
aka: [STM32]
autolink: true
infobox:
  - { label: Type, value: 32-bit microcontroller family }
  - { label: Core, value: ARM Cortex-M (M0 to M7/M33) }
  - { label: Vendor, value: STMicroelectronics }
  - { label: Tooling, value: STM32Cube, HAL, ST-LINK }
  - { label: Language, value: C, C++, Rust }
see_also: [arm-cortex-m, microcontroller, rp2040, avr-atmega, in-system-programming, real-time-operating-system]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/STM32
---

**STM32** is STMicroelectronics' broad family of 32-bit [microcontrollers](/reference/microcontroller/) built around [ARM Cortex-M](/reference/arm-cortex-m/) cores.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 172" role="img" aria-label="A positioning chart of STM32 series on two axes. The horizontal axis is increasing performance and the vertical axis is increasing low-power focus. The low-power STM32L0 sits high on the power axis but low on performance; the value-line F0 is low on both; the mainstream F1 and F4 sit in the middle; and the high-performance F7 and H7 sit far right for performance. All share one Cortex-M core family." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="50" y1="26" x2="50" y2="140"/>
    <line x1="50" y1="140" x2="440" y2="140"/>
    <path d="M50 26 L46 34 M50 26 L54 34" stroke-width="1.1"/>
    <path d="M440 140 L432 136 M440 140 L432 144" stroke-width="1.1"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.18" stroke-width="1.1">
    <circle cx="96" cy="46" r="6"/>
    <circle cx="104" cy="118" r="6"/>
    <circle cx="196" cy="96" r="6"/>
    <circle cx="270" cy="72" r="7"/>
    <circle cx="356" cy="58" r="7"/>
    <circle cx="412" cy="44" r="8"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="96" y="36" font-weight="600">L0</text>
    <text x="104" y="134" font-weight="600">F0</text>
    <text x="196" y="86" font-weight="600">F1</text>
    <text x="270" y="62" font-weight="600">F4</text>
    <text x="356" y="48" font-weight="600">F7</text>
    <text x="412" y="32" font-weight="600">H7</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" fill-opacity="0.9">
    <text x="60" y="22" text-anchor="start">low-power focus &#8593;</text>
    <text x="250" y="158" text-anchor="middle">performance &#8594;</text>
  </g>
</svg>
<figcaption>One family, many trade-offs: the ultra-low-power STM32L0 favours battery life, the value-line F0 minimises cost, the mainstream F1/F4 balance both, and the F7/H7 chase raw performance. Because they all share the same Cortex-M lineage, HAL, and tools, code moves across the map with little friction.</figcaption>
</figure>

## Overview

The line spans dozens of series, from the tiny low-power STM32L0 and value-line STM32F0 up to the high-performance STM32F7 and STM32H7. All share a common peripheral set — [UART](/reference/uart/), [SPI](/reference/spi/), [I²C](/reference/i2c/), timers with [PWM](/reference/pulse-width-modulation/), and on-chip [ADCs](/reference/analog-to-digital-converter/) — and a shared HAL (hardware abstraction layer) so code ports across the family.

That breadth-plus-consistency is the family's whole appeal: pick the series that fits a project's power and performance budget, and reuse most of the same code and tooling if requirements change. Parts are programmed in [C](/reference/c-language/), [C++](/reference/cpp-language/), or [Rust](/reference/rust-language/) and flashed over the ST-LINK debugger using [in-system programming](/reference/in-system-programming/).

## Series at a glance

A slice across the range shows the spread of cores and priorities:

| Series | Core | Focus | Typical part |
|--------|------|-------|--------------|
| STM32L0 | Cortex-M0+ | Ultra-low power | Battery sensors |
| STM32F0 | Cortex-M0 | Low cost | Value designs |
| STM32F1 | Cortex-M3 | Mainstream | "Blue Pill" boards |
| STM32F4 | Cortex-M4 | Balanced + DSP | Popular all-rounder |
| STM32H7 | Cortex-M7 | High performance | Demanding compute |

## Where it fits

STM32 is the default choice when a project outgrows an 8-bit [AVR/ATmega](/reference/avr-atmega/) but does not need a full [single-board computer](/reference/single-board-computer/). The cheap "Blue Pill" board and the official Nucleo and Discovery kits make it popular with hobbyists, while its rich peripherals and long supply lifetimes keep it common in industrial and automotive products. Many designs run [bare metal](/reference/bare-metal-programming/) or under a small [RTOS](/reference/real-time-operating-system/). In SDR gear an STM32 often handles front-end control — tuning, gain, and GPIO housekeeping — beside the host that does the actual decoding.

## Sources

[^wiki]: [STM32](https://en.wikipedia.org/wiki/STM32) — Wikipedia, on the STM32 family, cores, and tooling.
