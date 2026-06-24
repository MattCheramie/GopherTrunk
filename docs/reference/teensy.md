---
slug: teensy
title: Teensy
entry_type: hardware
category: hw-microcontrollers
description: Teensy is a line of compact, high-performance ARM Cortex-M microcontroller boards from PJRC, Arduino-compatible and popular for audio, USB, and demanding real-time projects.
keywords: Teensy, PJRC, Teensy 4.0, Teensy 4.1, Cortex-M7, Arduino compatible, Teensyduino, audio library
aka: [Teensy]
infobox:
  - { label: Type, value: Microcontroller board }
  - { label: Core, value: ARM Cortex-M7 (Teensy 4.x) }
  - { label: Vendor, value: PJRC }
  - { label: Compatible, value: Arduino (via Teensyduino) }
  - { label: Strengths, value: USB, audio, real-time I/O }
see_also: [arm-cortex-m, microcontroller, arduino, stm32, rp2040, firmware]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Teensy
---

**Teensy** is a line of compact, high-performance [microcontroller](/reference/microcontroller/) boards made by PJRC, built on [ARM Cortex-M](/reference/arm-cortex-m/) cores.[^wiki]

## Overview

The flagship Teensy 4.0 and 4.1 run a 600 MHz Cortex-M7 — far faster than a typical hobby MCU — in a board barely larger than a postage stamp. Teensy boards are [Arduino](/reference/arduino/)-compatible through the Teensyduino add-on, so they use familiar libraries and the Arduino IDE, but they also expose deep hardware features: many [GPIO](/reference/gpio/) pins, several [UART](/reference/uart/)/[SPI](/reference/spi/)/[I²C](/reference/i2c/) buses, [PWM](/reference/pulse-width-modulation/), and strong USB support.

## Where it fits

Teensy is the go-to when an Arduino-style workflow needs serious horsepower or precise timing. Its well-known audio library makes it a favorite for synthesizers and effects, and its fast cores suit MIDI controllers, data loggers, and other real-time work. Where a project wants the same class of compute with vendor tooling instead of the Arduino layer, an [STM32](/reference/stm32/) is the usual alternative.

## Sources

[^wiki]: [Teensy](https://en.wikipedia.org/wiki/Teensy) — Wikipedia, on the Teensy boards from PJRC.
