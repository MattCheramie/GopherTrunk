---
slug: arm-cortex-m
title: ARM Cortex-M
entry_type: concept
category: hw-microcontrollers
description: ARM Cortex-M is a family of 32-bit processor cores designed for microcontrollers, prioritizing low power, deterministic interrupt handling, and small silicon area over raw throughput.
keywords: ARM Cortex-M, Cortex-M0, Cortex-M4, Cortex-M7, Thumb, NVIC, microcontroller core, embedded processor
aka: [Cortex-M]
autolink: true
infobox:
  - { label: Type, value: Processor core family }
  - { label: Width, value: 32-bit }
  - { label: Designer, value: Arm Holdings }
  - { label: Members, value: M0, M0+, M3, M4, M7, M23, M33 }
  - { label: For, value: Microcontrollers }
see_also: [microcontroller, stm32, rp2040, teensy, real-time-operating-system, interrupt]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/ARM_Cortex-M
---

**ARM Cortex-M** is a family of 32-bit processor cores from Arm Holdings designed specifically for [microcontrollers](/reference/microcontroller/).[^wiki]

## Overview

Rather than chasing raw throughput, Cortex-M cores prioritize low power, small silicon area, and predictable real-time behavior. They execute the compact Thumb instruction set and include a built-in nested vectored [interrupt](/reference/interrupt/) controller (NVIC) that makes response latency deterministic. The lineup ranges from the minimal Cortex-M0/M0+ through the DSP-capable M4 and high-performance M7, plus the security-focused M23/M33. Arm licenses the core; chip vendors wrap it in their own memory and peripherals.

## Where it fits

Because the core is licensed rather than sold as a chip, the same Cortex-M architecture underlies products from many vendors: [STM32](/reference/stm32/) from STMicroelectronics, the [RP2040](/reference/rp2040/) from Raspberry Pi, [Teensy](/reference/teensy/) boards, and NXP, Nordic, and Microchip parts. This shared architecture means tooling, debuggers, and [RTOS](/reference/real-time-operating-system/) ports are broadly portable across the ecosystem — a major reason Cortex-M dominates 32-bit embedded design.

## Sources

[^wiki]: [ARM Cortex-M](https://en.wikipedia.org/wiki/ARM_Cortex-M) — Wikipedia, on the Cortex-M core family.
