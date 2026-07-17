---
slug: arm-cortex-m
title: ARM Cortex-M
entry_type: concept
category: hw-microcontrollers
description: ARM Cortex-M is a family of 32-bit processor cores designed for microcontrollers, prioritizing low power, deterministic interrupt handling, and small silicon area over raw throughput; chip vendors license the core and wrap it in their own memory and peripherals.
keywords: ARM Cortex-M, Cortex-M0, Cortex-M4, Cortex-M7, Thumb, NVIC, microcontroller core, embedded processor, licensed IP, ARM Holdings
aka: [Cortex-M]
autolink: true
infobox:
  - { label: Type, value: Processor core family }
  - { label: Width, value: 32-bit }
  - { label: Designer, value: Arm Holdings }
  - { label: Members, value: M0, M0+, M3, M4, M7, M23, M33 }
  - { label: For, value: Microcontrollers }
see_also: [microcontroller, arm-architecture, stm32, rp2040, teensy, real-time-operating-system]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/ARM_Cortex-M
---

**ARM Cortex-M** is a family of 32-bit processor cores from Arm Holdings designed specifically for [microcontrollers](/reference/microcontroller/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="Block diagram of a Cortex-M based chip. The 32-bit Cortex-M core and its nested vectored interrupt controller connect through a bus matrix to on-chip flash, SRAM, and a set of peripherals such as GPIO, timers, and analog-to-digital converters. Interrupt lines from the peripherals feed back into the interrupt controller." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="24" y="34" width="96" height="40" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="24" y="86" width="96" height="30" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="176" y="52" width="108" height="72" rx="4" fill="currentColor" fill-opacity="0.06"/>
    <rect x="336" y="30" width="100" height="26" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="336" y="66" width="100" height="26" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="336" y="102" width="100" height="30" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <line x1="120" y1="54" x2="176" y2="70"/>
    <line x1="120" y1="101" x2="176" y2="101"/>
    <line x1="284" y1="70" x2="336" y2="43"/>
    <line x1="284" y1="88" x2="336" y2="79"/>
    <line x1="284" y1="106" x2="336" y2="115"/>
  </g>
  <path d="M336 118 C300 150 150 150 100 118 L100 76" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3" fill="none" stroke-opacity="0.8"/>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="72" y="50" font-size="9" font-weight="600">Cortex-M</text>
    <text x="72" y="63" font-size="7.5" fill-opacity="0.85">32-bit core</text>
    <text x="72" y="104" font-size="8">NVIC</text>
    <text x="230" y="86" font-size="8.5" font-weight="600">Bus matrix</text>
    <text x="230" y="99" font-size="7.5" fill-opacity="0.85">AHB / APB</text>
    <text x="386" y="46" font-size="8">Flash</text>
    <text x="386" y="82" font-size="8">SRAM</text>
    <text x="386" y="115" font-size="8">GPIO &#183; timers</text>
    <text x="386" y="126" font-size="7.5" fill-opacity="0.85">ADC &#183; UART</text>
    <text x="205" y="164" font-size="7.5" fill-opacity="0.85">dashed: peripheral interrupt lines &#8594; NVIC</text>
  </g>
</svg>
<figcaption>A Cortex-M chip: the 32-bit core plus its nested vectored interrupt controller (NVIC) reach flash, SRAM, and peripherals over a bus matrix. Peripherals raise interrupts straight into the NVIC, giving the deterministic, low-latency response that defines the family. Arm designs the core; each vendor adds the memory and peripherals.</figcaption>
</figure>

## Overview

Rather than chasing raw throughput, Cortex-M cores prioritize low power, small silicon area, and predictable real-time behavior. They execute the compact Thumb instruction set and include a built-in nested vectored [interrupt](/reference/interrupt/) controller (NVIC) that makes response latency deterministic. The lineup ranges from the minimal Cortex-M0/M0+ through the DSP-capable M4 and high-performance M7, plus the security-focused M23/M33.

Crucially, Arm does not sell Cortex-M chips — it licenses the core as intellectual property (part of the broader [ARM architecture](/reference/arm-architecture/)) and chip vendors wrap it in their own flash, SRAM, and peripherals. That business model is why one instruction set shows up across parts from dozens of competing manufacturers.

## The lineup

The family trades performance for power and area in steps, all sharing the same tools and instruction set:

| Core | Tier | Highlights | Typical use |
|------|------|-----------|-------------|
| M0 / M0+ | Minimal | Smallest, lowest power | Cheap 32-bit MCUs, [RP2040](/reference/rp2040/) |
| M3 | Mainstream | Full Thumb-2, good all-rounder | General embedded |
| M4 | DSP | Single-precision FPU, DSP ops | Signal-processing MCUs |
| M7 | High-perf | Dual-issue, fast | [Teensy](/reference/teensy/) 4.x, audio |
| M23 / M33 | Secure | TrustZone security | Connected/secure devices |

## Where it fits

Because the core is licensed rather than sold as a chip, the same Cortex-M architecture underlies products from many vendors: [STM32](/reference/stm32/) from STMicroelectronics, the [RP2040](/reference/rp2040/) from Raspberry Pi, [Teensy](/reference/teensy/) boards, and NXP, Nordic, and Microchip parts. This shared architecture means tooling, debuggers, and [RTOS](/reference/real-time-operating-system/) ports are broadly portable across the ecosystem — a major reason Cortex-M dominates 32-bit embedded design. In SDR gear, Cortex-M parts commonly run the firmware inside dongles and front ends and serve as GPIO/[sensor](/reference/sensor/) helpers around a capture node, with the heavy DSP left to a larger host.

## Sources

[^wiki]: [ARM Cortex-M](https://en.wikipedia.org/wiki/ARM_Cortex-M) — Wikipedia, on the Cortex-M core family.
