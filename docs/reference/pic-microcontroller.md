---
slug: pic-microcontroller
title: PIC microcontroller
entry_type: hardware
category: hw-microcontrollers
description: PIC is a long-running family of low-cost microcontrollers from Microchip Technology, spanning 8-, 16-, and 32-bit cores and prized for simplicity, ruggedness, and very long supply lifetimes.
keywords: PIC, PIC microcontroller, Microchip, PIC16, PIC18, PIC24, dsPIC, PIC32, PICkit, MPLAB, 8-bit MCU
aka: [PIC]
infobox:
  - { label: Type, value: Microcontroller family }
  - { label: Cores, value: 8/16/32-bit }
  - { label: Vendor, value: Microchip Technology }
  - { label: Tooling, value: MPLAB X, PICkit programmer }
  - { label: Language, value: C, assembly }
see_also: [microcontroller, avr-atmega, stm32, firmware, in-system-programming, gpio]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/PIC_microcontrollers
---

**PIC** is a long-running family of low-cost [microcontrollers](/reference/microcontroller/) from Microchip Technology, originally introduced as "Peripheral Interface Controller" parts.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="A tier chart of the PIC microcontroller family arranged as three rising steps. The lowest and widest step is the 8-bit PIC10, 12, 16, and 18 lines. The middle step is the 16-bit PIC24 and dsPIC digital-signal parts. The tallest step is the 32-bit PIC32 line. An arrow along the bottom marks increasing performance from left to right." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="94" width="130" height="40" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="168" y="66" width="130" height="68" rx="4" fill="currentColor" fill-opacity="0.11"/>
    <rect x="306" y="38" width="130" height="96" rx="4" fill="currentColor" fill-opacity="0.15"/>
    <line x1="30" y1="150" x2="430" y2="150" stroke-width="1.2"/>
    <path d="M430 150 L420 145 M430 150 L420 155" stroke-width="1.2"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="95" y="112" font-size="8.5" font-weight="600">8-bit</text>
    <text x="95" y="125" font-size="7.5" fill-opacity="0.85">PIC10/12/16/18</text>
    <text x="233" y="88" font-size="8.5" font-weight="600">16-bit</text>
    <text x="233" y="101" font-size="7.5" fill-opacity="0.85">PIC24 &#183; dsPIC</text>
    <text x="371" y="60" font-size="8.5" font-weight="600">32-bit</text>
    <text x="371" y="73" font-size="7.5" fill-opacity="0.85">PIC32</text>
    <text x="230" y="164" font-size="7.5" fill-opacity="0.9">increasing performance &#8594;</text>
  </g>
</svg>
<figcaption>The PIC family climbs in steps: cheap, rugged 8-bit PIC10/12/16/18 parts at the base, 16-bit PIC24 and DSP-capable dsPIC in the middle, and 32-bit PIC32 at the top. One vendor, one tool suite (MPLAB), one programming workflow spanning the whole range.</figcaption>
</figure>

## Overview

The range stretches from simple 8-bit PIC10/PIC16/PIC18 chips through 16-bit PIC24/dsPIC up to 32-bit PIC32 parts. PICs are known for a small, regular instruction set, robust [GPIO](/reference/gpio/), and decades-long availability that makes them a staple of industrial and consumer products.

That longevity is a genuine selling point: a design built around a PIC can often buy the same part years later, which matters for equipment with long service lives. They are programmed in [C](/reference/c-language/) or assembly with Microchip's MPLAB tools and flashed via [in-system programming](/reference/in-system-programming/) using a PICkit or ICD programmer.

## The PIC tiers

One family, three levels of capability:

| Tier | Lines | Bits | Typical role |
|------|-------|------|--------------|
| Entry | PIC10 / PIC12 / PIC16 | 8 | Cheapest, tiny control tasks |
| Enhanced 8-bit | PIC18 | 8 | More memory and peripherals |
| Digital signal | PIC24 / dsPIC | 16 | DSP maths, motor control |
| High-end | PIC32 | 32 | MIPS core, heavier compute |

## Trade-offs

PICs compete head-to-head with [AVR/ATmega](/reference/avr-atmega/) in the 8-bit space; the choice often comes down to ecosystem familiarity rather than capability. For more compute or richer peripherals, designers reach for an ARM part such as the [STM32](/reference/stm32/). Like all small [MCUs](/reference/microcontroller/), a PIC runs your code as [firmware](/reference/firmware/) with no operating system, so it boots instantly and sips power — the profile of the small controllers and [sensors](/reference/sensor/) that surround an SDR capture setup rather than the host that runs GopherTrunk.

## Sources

[^wiki]: [PIC microcontrollers](https://en.wikipedia.org/wiki/PIC_microcontrollers) — Wikipedia, on the PIC families and tooling.
