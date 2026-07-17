---
slug: in-system-programming
title: In-system programming
entry_type: concept
category: hw-microcontrollers
description: In-system programming (ISP) flashes a microcontroller's firmware while the chip stays soldered in its circuit, using a debug interface such as SWD, JTAG, or SPI and a small programmer instead of removing the chip.
keywords: in-system programming, ISP, ICSP, SWD, JTAG, debug interface, flash, programmer, ST-LINK, PICkit, J-Link
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 156" role="img" aria-label="Wiring of an in-system programming session. A host PC connects by USB to a small programmer such as an ST-LINK. The programmer drives a few debug pins on a header on the target board, and those pins run to the microcontroller, which stays soldered in place. Labelled lines carry clock, data, reset, and ground." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="20" y="54" width="72" height="44" rx="4" fill="currentColor" fill-opacity="0.06"/>
    <rect x="140" y="54" width="80" height="44" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="300" y="34" width="150" height="94" rx="5" fill="currentColor" fill-opacity="0.04"/>
    <rect x="360" y="66" width="70" height="42" rx="3" fill="currentColor" fill-opacity="0.16"/>
    <rect x="306" y="54" width="16" height="60" rx="2" fill="currentColor" fill-opacity="0.10"/>
    <line x1="92" y1="76" x2="140" y2="76"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="220" y1="62" x2="306" y2="62"/>
    <line x1="220" y1="72" x2="306" y2="72"/>
    <line x1="220" y1="82" x2="306" y2="82"/>
    <line x1="220" y1="92" x2="306" y2="92"/>
    <line x1="322" y1="76" x2="360" y2="82"/>
    <line x1="322" y1="92" x2="360" y2="92"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="56" y="79" font-size="8">Host PC</text>
    <text x="56" y="90" font-size="7.5" fill-opacity="0.85">USB</text>
    <text x="180" y="73" font-size="8" font-weight="600">Programmer</text>
    <text x="180" y="85" font-size="7.5" fill-opacity="0.85">ST-LINK</text>
    <text x="314" y="128" font-size="7.5">header</text>
    <text x="395" y="90" font-size="8" font-weight="600">MCU</text>
    <text x="375" y="46" font-size="7.5" fill-opacity="0.85">target board (soldered)</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="7" text-anchor="start">
    <text x="236" y="60">CLK</text>
    <text x="236" y="70">DAT</text>
    <text x="236" y="80">RST</text>
    <text x="236" y="90">GND</text>
  </g>
</svg>
<figcaption>An ISP session: the host PC talks over USB to a small programmer (here an ST-LINK), which drives a handful of debug pins — clock, data, reset, ground — on a header wired straight to the microcontroller. The chip never leaves the board, and the same link usually allows step-debugging and memory readback.</figcaption>
</figure>

## Overview

Before ISP, a chip had to be removed and placed in a dedicated programmer. ISP instead uses a small debug interface on the board — SWD or JTAG on [ARM Cortex-M](/reference/arm-cortex-m/) parts, an [SPI](/reference/spi/)-based ICSP header on [AVR/ATmega](/reference/avr-atmega/), or vendor schemes on [PIC](/reference/pic-microcontroller/) — so a programmer like ST-LINK, J-Link, or PICkit can flash and debug the device in place.

Only a few wires are needed: a clock, one or two data lines, reset, and ground. The same interface usually allows single-step debugging and reading memory back, which is why the ISP header is as much a debugging port as a programming one.

## Common ISP interfaces

Which pins you break out depends on the chip family:

| Interface | Typical parts | Pins | Programmer |
|-----------|---------------|------|-----------|
| SWD | [ARM Cortex-M](/reference/arm-cortex-m/), [STM32](/reference/stm32/) | 2 (+ reset) | ST-LINK, J-Link |
| JTAG | Larger ARM, FPGAs | 4–5 | J-Link, generic |
| SPI/ICSP | [AVR/ATmega](/reference/avr-atmega/) | 3 (+ reset) | USBasp, Arduino as ISP |
| ICSP (PIC) | [PIC](/reference/pic-microcontroller/) | 2 (+ reset) | PICkit, ICD |

## Where it fits

ISP is the lowest-level way to get code onto a chip and is how a brand-new, blank MCU is first programmed — including installing a [bootloader](/reference/bootloader/), after which routine updates can happen over USB instead. On an [STM32](/reference/stm32/) board, the SWD pins broken out next to the MCU are the ISP interface. It is the standard production and bring-up workflow for [embedded systems](/reference/embedded-system/), including the small controllers and SDR front-end boards that surround a capture setup.

## Sources

[^wiki]: [In-system programming](https://en.wikipedia.org/wiki/In-system_programming) — Wikipedia, on programming chips in place.
