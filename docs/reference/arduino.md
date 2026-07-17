---
slug: arduino
title: Arduino
entry_type: hardware
category: hw-microcontrollers
description: Arduino is a widely used family of beginner-friendly microcontroller boards and the open-source ecosystem of tools, libraries, and code built around them, centred on the simple "sketch" programming model.
keywords: Arduino, sketch, Arduino IDE, shield, 8-bit microcontroller, open source hardware, AVR, ATmega328P, Arduino Uno, GPIO
aka: [Arduino]
autolink: true
infobox:
  - { label: Type, value: Microcontroller boards }
  - { label: Core, value: Classic boards 8-bit AVR }
  - { label: Memory, value: Kilobytes of RAM and flash }
  - { label: Software, value: Arduino IDE (sketches) }
  - { label: License, value: Open source }
see_also: [microcontroller, avr-atmega, esp32, gpio, single-board-computer, firmware]
related_lessons:
  - { title: "Arduino", url: /learn/intro-hardware/arduino/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Arduino
---

**Arduino** is a widely used family of beginner-friendly [microcontroller](/reference/microcontroller/) boards and the open-source ecosystem of tools and code built around them.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Top view of a classic Arduino Uno board. Along the top edge is a header of digital input and output pins; along the bottom edge are power and analog input pins. A USB port and a barrel power jack sit on the left, and the 8-bit ATmega328P microcontroller is in the centre, wired to every header pin." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="40" y="34" width="380" height="112" rx="6" fill-opacity="0" stroke-opacity="0.8"/>
    <rect x="20" y="60" width="26" height="26" rx="2" fill="currentColor" fill-opacity="0.12"/>
    <rect x="20" y="98" width="24" height="20" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="196" y="74" width="70" height="34" rx="2" fill="currentColor" fill-opacity="0.12"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="currentColor" fill-opacity="0.18">
    <rect x="70" y="40" width="10" height="8"/><rect x="86" y="40" width="10" height="8"/><rect x="102" y="40" width="10" height="8"/><rect x="118" y="40" width="10" height="8"/><rect x="134" y="40" width="10" height="8"/><rect x="150" y="40" width="10" height="8"/><rect x="166" y="40" width="10" height="8"/><rect x="182" y="40" width="10" height="8"/><rect x="198" y="40" width="10" height="8"/><rect x="214" y="40" width="10" height="8"/>
    <rect x="250" y="132" width="10" height="8"/><rect x="266" y="132" width="10" height="8"/><rect x="282" y="132" width="10" height="8"/><rect x="298" y="132" width="10" height="8"/><rect x="314" y="132" width="10" height="8"/><rect x="330" y="132" width="10" height="8"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="150" y="26" font-size="8.5">DIGITAL 0&#8211;13 (GPIO / PWM)</text>
    <text x="231" y="94" font-size="8.5" font-weight="600">ATmega328P</text>
    <text x="231" y="105" font-size="7.5" fill-opacity="0.85">8-bit AVR core</text>
    <text x="33" y="55" font-size="7.5">USB</text>
    <text x="32" y="128" font-size="7.5">PWR</text>
    <text x="290" y="152" font-size="8.5">POWER &amp; ANALOG A0&#8211;A5</text>
  </g>
</svg>
<figcaption>A classic Arduino Uno: an 8-bit ATmega328P microcontroller at the centre, its pins broken out to labelled header rows. You wire sensors and actuators to those pins, write a sketch, and flash it over USB — with "shield" boards stacking onto the same headers to add features.</figcaption>
</figure>

## Overview

An Arduino program is called a *sketch*. The simple Arduino IDE — with its two-function skeleton of `setup()` and `loop()` — plus a huge library and "shield" ecosystem of plug-in add-on boards, is what made microcontrollers approachable to hobbyists and students. Classic boards are 8-bit and modest in speed and memory, but that is enough for sensors, motors, and small projects.

The project began in Italy in 2005 as open-source hardware: the board schematics, the IDE, and the core libraries are all freely available, which spawned countless clones and derivatives. That openness is central to Arduino's identity — the name refers as much to the software and community as to any one board.

The classic lineup runs on Microchip's [AVR/ATmega](/reference/avr-atmega/) chips (the Uno's ATmega328P is the icon), while newer official boards move to 32-bit [ARM Cortex-M](/reference/arm-cortex-m/) parts and other cores for more speed and memory.

## Board line-up

Arduino spans a range of form factors and cores, but they share the same sketch model and IDE:

| Board | Core | Clock | Flash | Notable |
|-------|------|-------|-------|---------|
| Uno / Nano | 8-bit AVR (ATmega328P) | 16 MHz | 32 KB | The classic beginner board |
| Mega 2560 | 8-bit AVR (ATmega2560) | 16 MHz | 256 KB | Many I/O pins |
| Micro / Leonardo | 8-bit AVR (ATmega32U4) | 16 MHz | 32 KB | Native USB |
| Nano 33 / Portenta | 32-bit ARM Cortex-M | 64+ MHz | 256 KB+ | BLE, more RAM |

## Where it fits

Arduino sits at the gentle end of embedded computing: you wire components to its [GPIO](/reference/gpio/) pins, write a sketch, and flash it onto the board. The ecosystem's tooling has spread well beyond the original boards — many other microcontrollers, including the [ESP32](/reference/esp32/), can be programmed with the Arduino IDE too. In an SDR context an Arduino makes a handy GPIO helper — switching an antenna relay, reading a temperature [sensor](/reference/sensor/), or blinking a status light beside a capture node — while GopherTrunk itself runs on a full computer. These tiny boards also drive the kinds of small radios that fill the airwaves GopherTrunk listens to.

## Sources

[^wiki]: [Arduino](https://en.wikipedia.org/wiki/Arduino) — Wikipedia, on the boards, sketches, and open-source ecosystem.
