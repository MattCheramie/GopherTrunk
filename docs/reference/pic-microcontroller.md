---
slug: pic-microcontroller
title: PIC microcontroller
entry_type: hardware
category: hw-microcontrollers
description: PIC is a long-running family of low-cost microcontrollers from Microchip Technology, spanning 8-, 16-, and 32-bit cores and prized for simplicity, ruggedness, and very long supply lifetimes.
keywords: PIC, PIC microcontroller, Microchip, PIC16, PIC18, PICkit, MPLAB, 8-bit MCU
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

## Overview

The range stretches from simple 8-bit PIC10/PIC16/PIC18 chips through 16-bit PIC24/dsPIC up to 32-bit PIC32 parts. PICs are known for a small, regular instruction set, robust [GPIO](/reference/gpio/), and decades-long availability that makes them a staple of industrial and consumer products. They are programmed in [C](/reference/c-language/) or assembly with Microchip's MPLAB tools and flashed via [in-system programming](/reference/in-system-programming/) using a PICkit or ICD programmer.

## Trade-offs

PICs compete head-to-head with [AVR/ATmega](/reference/avr-atmega/) in the 8-bit space; the choice often comes down to ecosystem familiarity rather than capability. For more compute or richer peripherals, designers reach for an ARM part such as the [STM32](/reference/stm32/). Like all small [MCUs](/reference/microcontroller/), a PIC runs your code as [firmware](/reference/firmware/) with no operating system, so it boots instantly and sips power.

## Sources

[^wiki]: [PIC microcontrollers](https://en.wikipedia.org/wiki/PIC_microcontrollers) — Wikipedia, on the PIC families and tooling.
