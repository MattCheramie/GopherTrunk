---
slug: raspberry-pi-pico
title: Raspberry Pi Pico
entry_type: hardware
category: hw-microcontrollers
description: The Raspberry Pi Pico is a tiny, low-cost microcontroller board built on the RP2040 chip; the Pico W variant adds Wi-Fi, making it a popular entry point for embedded and IoT projects.
keywords: Raspberry Pi Pico, Pico W, RP2040, microcontroller board, MicroPython, Pico SDK, Wi-Fi
aka: [Pico, Pico W]
autolink: true
infobox:
  - { label: Type, value: Microcontroller board }
  - { label: Chip, value: RP2040 (dual Cortex-M0+) }
  - { label: Wireless, value: Wi-Fi on Pico W / 2 W }
  - { label: Typical price, value: ~$4–6 }
  - { label: Language, value: C/C++, MicroPython }
see_also: [rp2040, raspberry-pi, microcontroller, esp32, gpio, internet-of-things]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi_Pico
---

**The Raspberry Pi Pico** is a tiny, low-cost [microcontroller](/reference/microcontroller/) board built around the [RP2040](/reference/rp2040/) chip.[^wiki]

## Overview

Unlike the Linux-running [Raspberry Pi](/reference/raspberry-pi/) single-board computers, the Pico is a bare-metal MCU board: castellated edges for soldering, a row of [GPIO](/reference/gpio/) pins, and a USB port that doubles as power and a drag-and-drop programming interface. It is written in C/C++ with the Pico SDK or in MicroPython. The **Pico W** adds Wi-Fi (and Bluetooth), turning it into a cheap connected node for the [Internet of Things](/reference/internet-of-things/); the Pico 2 and 2 W move to the newer RP2350.

## Where it fits

The Pico is a common first board for learning embedded programming, competing with [Arduino](/reference/arduino/) and [ESP32](/reference/esp32/) boards. It is well suited to reading [sensors](/reference/sensor/), driving displays, and bit-banging protocols via the RP2040's programmable I/O. Like any MCU it is far too small to run GopherTrunk, but it is exactly the sort of device that crowds the airwaves GopherTrunk listens to.

## Sources

[^wiki]: [Raspberry Pi Pico](https://en.wikipedia.org/wiki/Raspberry_Pi_Pico) — Wikipedia, on the Pico board and its variants.
