---
slug: arduino
title: Arduino
entry_type: hardware
category: hw-microcontrollers
description: Arduino is a widely used family of beginner-friendly microcontroller boards and the open-source ecosystem of tools and code built around them.
keywords: Arduino, sketch, Arduino IDE, shield, 8-bit microcontroller, open source hardware, AVR
aka: [Arduino]
autolink: true
infobox:
  - { label: Type, value: Microcontroller boards }
  - { label: Core, value: Classic boards 8-bit AVR }
  - { label: Memory, value: Kilobytes of RAM and flash }
  - { label: Software, value: Arduino IDE (sketches) }
  - { label: License, value: Open source }
see_also: [microcontroller, esp32, gpio, single-board-computer, firmware]
related_lessons:
  - { title: "Arduino", url: /learn/intro-hardware/arduino/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Arduino
---

**Arduino** is a widely used family of beginner-friendly [microcontroller](/reference/microcontroller/) boards and the open-source ecosystem of tools and code built around them.[^wiki]

## Overview

An Arduino program is called a *sketch*. The simple Arduino IDE, plus a huge library and "shield" ecosystem of plug-in add-on boards, is what made microcontrollers approachable to hobbyists and students. Classic Arduino boards are 8-bit and modest in speed and memory, but that is enough for sensors, motors, and small projects.

## Where it fits

Arduino sits at the gentle end of embedded computing: you wire components to its [GPIO](/reference/gpio/) pins, write a sketch, and flash it onto the board. The ecosystem's tooling has spread well beyond the original boards — many other microcontrollers, including the [ESP32](/reference/esp32/), can be programmed with the Arduino IDE too. These tiny boards drive the kinds of small radios that fill the airwaves GopherTrunk listens to.

## Sources

[^wiki]: [Arduino](https://en.wikipedia.org/wiki/Arduino) — Wikipedia, on the boards, sketches, and open-source ecosystem.
