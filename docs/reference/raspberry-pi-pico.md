---
slug: raspberry-pi-pico
title: Raspberry Pi Pico
entry_type: hardware
category: hw-microcontrollers
description: The Raspberry Pi Pico is a tiny, low-cost microcontroller board built on the RP2040 chip; the Pico W variant adds Wi-Fi, making it a popular entry point for embedded and IoT projects.
keywords: Raspberry Pi Pico, Pico W, RP2040, RP2350, microcontroller board, MicroPython, Pico SDK, Wi-Fi, castellated pins
aka: [Pico, Pico W]
autolink: true
infobox:
  - { label: Type, value: Microcontroller board }
  - { label: Chip, value: RP2040 (dual Cortex-M0+) }
  - { label: Wireless, value: Wi-Fi on Pico W / 2 W }
  - { label: Typical price, value: ~$4–6 }
  - { label: Language, value: C/C++, MicroPython }
see_also: [rp2040, raspberry-pi, microcontroller, esp32, gpio, sensor]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi_Pico
---

**The Raspberry Pi Pico** is a tiny, low-cost [microcontroller](/reference/microcontroller/) board built around the [RP2040](/reference/rp2040/) chip.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="Top view of a Raspberry Pi Pico board. A micro-USB port sits at the top edge, the square RP2040 chip is in the centre, and rows of castellated pads line both long edges, most of them general-purpose I/O pins that also carry serial, PWM, and analog-to-digital functions." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="140" y="20" width="180" height="140" rx="10" stroke-opacity="0.8"/>
    <rect x="206" y="14" width="48" height="16" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="196" y="70" width="68" height="46" rx="3" fill="currentColor" fill-opacity="0.16"/>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="currentColor" fill-opacity="0.18">
    <rect x="130" y="34" width="12" height="9"/><rect x="130" y="50" width="12" height="9"/><rect x="130" y="66" width="12" height="9"/><rect x="130" y="82" width="12" height="9"/><rect x="130" y="98" width="12" height="9"/><rect x="130" y="114" width="12" height="9"/><rect x="130" y="130" width="12" height="9"/>
    <rect x="318" y="34" width="12" height="9"/><rect x="318" y="50" width="12" height="9"/><rect x="318" y="66" width="12" height="9"/><rect x="318" y="82" width="12" height="9"/><rect x="318" y="98" width="12" height="9"/><rect x="318" y="114" width="12" height="9"/><rect x="318" y="130" width="12" height="9"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="230" y="10" font-size="8">micro-USB (power + flashing)</text>
    <text x="230" y="90" font-size="8.5" font-weight="600">RP2040</text>
    <text x="230" y="102" font-size="7.5" fill-opacity="0.85">dual Cortex-M0+</text>
    <text x="86" y="88" font-size="7.5" text-anchor="middle">GPIO</text>
    <text x="86" y="99" font-size="7.5" text-anchor="middle">edge</text>
    <text x="374" y="88" font-size="7.5" text-anchor="middle">GPIO</text>
    <text x="374" y="99" font-size="7.5" text-anchor="middle">edge</text>
    <text x="230" y="172" font-size="7.5" fill-opacity="0.9">castellated pads: solder as a header or reflow like a module</text>
  </g>
</svg>
<figcaption>The Pico's layout: a micro-USB port for power and drag-and-drop programming, the RP2040 in the middle, and castellated pads down both edges — most of them GPIO that double as serial, PWM, and ADC lines. The castellations let you solder headers or reflow the whole board onto another PCB like a module.</figcaption>
</figure>

## Overview

Unlike the Linux-running [Raspberry Pi](/reference/raspberry-pi/) single-board computers, the Pico is a bare-metal MCU board: castellated edges for soldering, a row of [GPIO](/reference/gpio/) pins, and a USB port that doubles as power and a drag-and-drop programming interface. It is written in C/C++ with the Pico SDK or in MicroPython.

The **Pico W** adds Wi-Fi (and Bluetooth), turning it into a cheap connected node for the [Internet of Things](/reference/internet-of-things/); the Pico 2 and 2 W move to the newer RP2350 chip with faster cores and more memory.

## The Pico lineup

Several variants share the same board footprint:

| Board | Chip | Wireless | Note |
|-------|------|----------|------|
| Pico | RP2040 | — | The original |
| Pico W | RP2040 | Wi-Fi + BLE | Connected projects |
| Pico 2 | RP2350 | — | Faster, more RAM |
| Pico 2 W | RP2350 | Wi-Fi + BLE | Newest connected board |

## Where it fits

The Pico is a common first board for learning embedded programming, competing with [Arduino](/reference/arduino/) and [ESP32](/reference/esp32/) boards. It is well suited to reading [sensors](/reference/sensor/), driving displays, and bit-banging protocols via the RP2040's programmable I/O. Like any MCU it is far too small to run GopherTrunk, but it is exactly the sort of device that crowds the airwaves GopherTrunk listens to — and its PIO blocks make it a capable GPIO/timing helper alongside a capture node.

## Sources

[^wiki]: [Raspberry Pi Pico](https://en.wikipedia.org/wiki/Raspberry_Pi_Pico) — Wikipedia, on the Pico board and its variants.
