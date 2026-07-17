---
slug: teensy
title: Teensy
entry_type: hardware
category: hw-microcontrollers
description: Teensy is a line of compact, high-performance ARM Cortex-M microcontroller boards from PJRC, Arduino-compatible and popular for audio, USB, and demanding real-time projects, headlined by the 600 MHz Cortex-M7 Teensy 4.x.
keywords: Teensy, PJRC, Teensy 4.0, Teensy 4.1, Cortex-M7, Arduino compatible, Teensyduino, audio library, 600 MHz
aka: [Teensy]
infobox:
  - { label: Type, value: Microcontroller board }
  - { label: Core, value: ARM Cortex-M7 (Teensy 4.x) }
  - { label: Vendor, value: PJRC }
  - { label: Compatible, value: Arduino (via Teensyduino) }
  - { label: Strengths, value: USB, audio, real-time I/O }
see_also: [arm-cortex-m, microcontroller, arduino, stm32, rp2040, sensor]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Teensy
---

**Teensy** is a line of compact, high-performance [microcontroller](/reference/microcontroller/) boards made by PJRC, built on [ARM Cortex-M](/reference/arm-cortex-m/) cores.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 156" role="img" aria-label="A bar chart comparing microcontroller clock speeds. An 8-bit Arduino Uno runs at 16 megahertz, shown as a very short bar. A 133 megahertz RP2040 is a short bar. An STM32F4 at 168 megahertz is a little longer. The 600 megahertz Cortex-M7 Teensy 4.0 towers over them all with by far the longest bar." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <line x1="118" y1="24" x2="118" y2="132"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.16" stroke-width="1.1">
    <rect x="118" y="30" width="9" height="16"/>
    <rect x="118" y="58" width="72" height="16"/>
    <rect x="118" y="86" width="91" height="16"/>
    <rect x="118" y="114" width="324" height="16"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="112" y="42" text-anchor="end">Uno (AVR)</text>
    <text x="112" y="70" text-anchor="end">RP2040</text>
    <text x="112" y="98" text-anchor="end">STM32F4</text>
    <text x="112" y="126" text-anchor="end" font-weight="600">Teensy 4.0</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" fill-opacity="0.9">
    <text x="133" y="42" text-anchor="start">16 MHz</text>
    <text x="196" y="70" text-anchor="start">133 MHz</text>
    <text x="215" y="98" text-anchor="start">168 MHz</text>
    <text x="436" y="126" text-anchor="end">600 MHz &#183; Cortex-M7</text>
    <text x="118" y="148" text-anchor="start" fill-opacity="0.85">clock speed (bars roughly to scale)</text>
  </g>
</svg>
<figcaption>Teensy's calling card is horsepower: the Teensy 4.0's 600 MHz Cortex-M7 dwarfs a classic 16 MHz Arduino Uno and comfortably outruns typical hobby MCUs — all in a board the size of a postage stamp, still driven from the familiar Arduino IDE.</figcaption>
</figure>

## Overview

The flagship Teensy 4.0 and 4.1 run a 600 MHz Cortex-M7 — far faster than a typical hobby MCU — in a board barely larger than a postage stamp. Teensy boards are [Arduino](/reference/arduino/)-compatible through the Teensyduino add-on, so they use familiar libraries and the Arduino IDE, but they also expose deep hardware features: many [GPIO](/reference/gpio/) pins, several [UART](/reference/uart/)/[SPI](/reference/spi/)/[I²C](/reference/i2c/) buses, [PWM](/reference/pulse-width-modulation/), and strong USB support.

PJRC, the small Oregon company behind Teensy, is also known for a polished audio library that turns the boards into capable real-time signal processors — a large part of their following.

## Teensy 4.x versus a classic board

The gap in raw capability is stark:

| Spec | Arduino Uno | Teensy 4.0 |
|------|-------------|------------|
| Core | 8-bit AVR | 32-bit Cortex-M7 |
| Clock | 16 MHz | 600 MHz |
| Flash | 32 KB | 2 MB |
| RAM | 2 KB | 1 MB |
| FPU / DSP | No | Yes |

## Where it fits

Teensy is the go-to when an Arduino-style workflow needs serious horsepower or precise timing. Its well-known audio library makes it a favorite for synthesizers and effects, and its fast cores suit MIDI controllers, data loggers, and other real-time work. Where a project wants the same class of compute with vendor tooling instead of the Arduino layer, an [STM32](/reference/stm32/) is the usual alternative. Its DSP-capable core and audio pipeline also make it a plausible front-end for light real-time signal work near an SDR setup — though full trunking decode still belongs on a general-purpose host.

## Sources

[^wiki]: [Teensy](https://en.wikipedia.org/wiki/Teensy) — Wikipedia, on the Teensy boards from PJRC.
