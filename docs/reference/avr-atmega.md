---
slug: avr-atmega
title: AVR / ATmega
entry_type: hardware
category: hw-microcontrollers
description: AVR is an 8-bit RISC microcontroller architecture from Atmel (now Microchip) with a Harvard design; its ATmega line, especially the ATmega328P, powers the classic Arduino Uno and countless hobby projects.
keywords: AVR, ATmega, ATmega328P, ATtiny, Atmel, Microchip, Arduino Uno, 8-bit RISC, AVR-GCC, Harvard architecture
aka: [AVR, ATmega]
infobox:
  - { label: Type, value: 8-bit microcontroller architecture }
  - { label: Core, value: AVR RISC (Harvard) }
  - { label: Vendor, value: Atmel (now Microchip) }
  - { label: Famous part, value: ATmega328P (Arduino Uno) }
  - { label: Language, value: C, C++, assembly }
see_also: [arduino, microcontroller, pic-microcontroller, arm-cortex-m, firmware, in-system-programming]
related_lessons:
  - { title: "Arduino", url: /learn/intro-hardware/arduino/ }
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/AVR_microcontrollers
---

**AVR** is an 8-bit RISC [microcontroller](/reference/microcontroller/) architecture created by Atmel and now owned by Microchip; **ATmega** is its best-known product line.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="Simplified Harvard architecture of an AVR chip. The 8-bit RISC CPU with its 32 registers connects over one bus to program flash memory and over a separate bus to data SRAM and EEPROM. A peripheral bus links the CPU to GPIO ports, timers, an analog-to-digital converter, and serial interfaces." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="176" y="30" width="108" height="46" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="24" y="34" width="104" height="38" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="332" y="26" width="104" height="24" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="332" y="56" width="104" height="24" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="176" y="108" width="108" height="34" rx="4" fill="currentColor" fill-opacity="0.06"/>
    <rect x="24" y="112" width="104" height="26" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="332" y="108" width="104" height="34" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <line x1="128" y1="53" x2="176" y2="53"/>
    <line x1="284" y1="45" x2="332" y2="38"/>
    <line x1="284" y1="60" x2="332" y2="68"/>
    <line x1="230" y1="76" x2="230" y2="108"/>
    <line x1="176" y1="125" x2="128" y2="125"/>
    <line x1="284" y1="125" x2="332" y2="125"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="230" y="49" font-size="9" font-weight="600">8-bit AVR CPU</text>
    <text x="230" y="62" font-size="7.5" fill-opacity="0.85">32 registers, RISC</text>
    <text x="76" y="50" font-size="8" font-weight="600">Program flash</text>
    <text x="76" y="62" font-size="7.5" fill-opacity="0.85">instruction bus</text>
    <text x="384" y="41" font-size="8">SRAM</text>
    <text x="384" y="71" font-size="8">EEPROM</text>
    <text x="230" y="123" font-size="8" font-weight="600">Peripheral bus</text>
    <text x="230" y="135" font-size="7.5" fill-opacity="0.85">timers &#183; PWM</text>
    <text x="76" y="129" font-size="8">GPIO ports</text>
    <text x="384" y="123" font-size="8">ADC</text>
    <text x="384" y="135" font-size="7.5" fill-opacity="0.85">UART &#183; SPI &#183; I&#178;C</text>
  </g>
</svg>
<figcaption>AVR is a Harvard design: the 8-bit RISC core fetches instructions from program flash over one bus while reading and writing data in SRAM and EEPROM over another, so a fetch and a data access can happen at once. A peripheral bus hangs GPIO, timers, the ADC, and serial ports off the same core.</figcaption>
</figure>

## Overview

The AVR design pairs a simple, fast 8-bit core with on-chip flash, SRAM, and EEPROM, plus [GPIO](/reference/gpio/), timers with [PWM](/reference/pulse-width-modulation/), an [ADC](/reference/analog-to-digital-converter/), and [UART](/reference/uart/)/[SPI](/reference/spi/)/[I²C](/reference/i2c/) peripherals. It is a *Harvard* RISC architecture — separate buses for program and data — so most instructions run in a single clock cycle, giving surprisingly brisk throughput for an 8-bit part.

Sibling lines include the tiny ATtiny and the larger ATmega. The ATmega328P is iconic because it is the chip on the original [Arduino](/reference/arduino/) Uno, which made AVR the gateway architecture for a generation of makers.

## AVR versus the alternatives

Where AVR sits among common hobby and embedded cores:

| Family | Bits | Core style | Note |
|--------|------|-----------|------|
| AVR / ATmega | 8 | RISC, Harvard | Single-cycle, Arduino default |
| [PIC](/reference/pic-microcontroller/) | 8/16/32 | RISC | Long supply life, rival to AVR |
| [ARM Cortex-M](/reference/arm-cortex-m/) | 32 | RISC | More compute, richer peripherals |

## Where it fits

AVR shines where 8 bits are plenty: low cost, low power, and a friendly toolchain (AVR-GCC, the Arduino IDE). Code is written in [C](/reference/c-language/) or [C++](/reference/cpp-language/) and flashed via [in-system programming](/reference/in-system-programming/) over SPI. When a project needs 32-bit performance, more memory, or built-in radios, designers step up to an [STM32](/reference/stm32/) or [ESP32](/reference/esp32/); the rival 8-bit family is the [PIC](/reference/pic-microcontroller/). Around an SDR setup, an ATmega is a tidy way to toggle relays or read a [sensor](/reference/sensor/) beside a capture node — small housekeeping jobs a decode host should not be bothered with.

## Sources

[^wiki]: [AVR microcontrollers](https://en.wikipedia.org/wiki/AVR_microcontrollers) — Wikipedia, on the AVR architecture and ATmega line.
