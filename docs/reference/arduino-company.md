---
slug: arduino-company
title: Arduino (company)
entry_type: organization
category: hw-organizations
description: Arduino is an open-source hardware company that makes the Arduino microcontroller boards and development environment, popularising electronics for makers and beginners.
keywords: Arduino, open-source hardware, microcontroller board, maker, Massimo Banzi, IDE, ATmega, shield
aka: [Arduino LLC, Arduino SA]
autolink: false
infobox:
  - { label: Type, value: Open-source hardware company }
  - { label: Founded, value: "2005" }
  - { label: HQ, value: Italy }
  - { label: Makes, value: Arduino microcontroller boards and IDE }
see_also: [arduino, massimo-banzi, microcontroller, avr-atmega, internet-of-things]
related_lessons:
  - { title: "Arduino", url: /learn/intro-hardware/arduino/ }
cite_urls:
  - https://www.arduino.cc/
  - https://en.wikipedia.org/wiki/Arduino
---

**Arduino** is an open-source hardware company that makes the
[Arduino](/reference/arduino/) [microcontroller](/reference/microcontroller/) boards and the
accompanying development environment.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A map of what Arduino makes. A central Arduino node connects to three things it releases as open source: the physical boards built around an ATmega microcontroller, the friendly Arduino IDE and libraries, and the open board designs, which together feed a large maker ecosystem of clones, shields, and tutorials." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="90" y1="75" x2="220" y2="40"/>
    <line x1="90" y1="75" x2="220" y2="75"/>
    <line x1="90" y1="75" x2="220" y2="110"/>
    <line x1="300" y1="40" x2="360" y2="75"/>
    <line x1="300" y1="75" x2="360" y2="75"/>
    <line x1="300" y1="110" x2="360" y2="75"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.12" stroke-width="1.2">
    <rect x="20" y="60" width="72" height="30" rx="4"/>
    <rect x="222" y="26" width="80" height="28" rx="4"/>
    <rect x="222" y="61" width="80" height="28" rx="4"/>
    <rect x="222" y="96" width="80" height="28" rx="4"/>
    <rect x="360" y="60" width="86" height="30" rx="4"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="9">
    <text x="56" y="79" font-weight="600">Arduino</text>
    <text x="262" y="44">board (ATmega)</text>
    <text x="262" y="79">IDE &amp; libraries</text>
    <text x="262" y="114">open designs</text>
    <text x="403" y="79">maker ecosystem</text>
  </g>
  <text x="230" y="144" font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.9">everything released open source — anyone may study, copy, and extend it</text>
</svg>
<figcaption>Arduino pairs a simple microcontroller board with a friendly IDE and releases both as open source, so a broad ecosystem of compatible clones, add-on shields, and tutorials could grow around it.</figcaption>
</figure>

## Overview

Arduino began around 2005 at a design school in Ivrea, Italy, from a project led by
[Massimo Banzi](/reference/massimo-banzi/) and collaborators who wanted an easy, cheap way
for artists and students to build interactive electronics. The result paired a simple
microcontroller board — early boards built around Atmel's
[AVR ATmega](/reference/avr-atmega/) chips — with a friendly programming environment, and
released both the board designs and software as open source.[^home]

That openness let others freely study, copy, and extend the platform, fueling a huge maker
ecosystem of compatible boards, shields, and tutorials. The company designs and sells
official boards and maintains the Arduino IDE and libraries.

## What they make

The Arduino line spans a few board families plus the software that ties them together:

| Product | What it is |
|---------|------------|
| Uno / Nano | Classic 8-bit ATmega boards for learning and small projects |
| Mega | Larger board with many I/O pins |
| MKR / Nano 33 | 32-bit boards with wireless for IoT |
| Arduino IDE | The editor, compiler, and library manager |

## Where it fits

Arduino made microcontroller programming approachable for people without an electronics
background, and its boards are a common starting point for sensors, automation, and small
embedded projects. In an SDR context, an Arduino is a natural choice for the simple control
glue around a receiver — switching an antenna relay, reading a sensor, or driving status
LEDs — while heavier signal processing happens on a host computer nearby.

## Sources

[^home]: [Arduino](https://www.arduino.cc/) — the company's official site.
[^wiki]: [Arduino](https://en.wikipedia.org/wiki/Arduino) — Wikipedia, for the project's origin and ecosystem.
