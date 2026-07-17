---
slug: gpio
title: GPIO
entry_type: hardware
category: hw-sbc
description: GPIO (general-purpose input/output) are pins on an SBC or microcontroller that code can read or switch on and off to talk to sensors, lights, motors, and other electronics.
keywords: GPIO, general-purpose input output, pins, sensors, microcontroller, Raspberry Pi GPIO, hardware interface, 40-pin header, I2C, SPI
aka: [GPIO]
autolink: true
infobox:
  - { label: Type, value: Digital I/O pins }
  - { label: Found on, value: SBCs, microcontrollers }
  - { label: Controls, value: Sensors, lights, motors }
  - { label: Driven from, value: Code (Python, C, Go) }
  - { label: Common form, value: 40-pin header }
see_also: [single-board-computer, microcontroller, input-output, raspberry-pi, beaglebone, arduino]
related_lessons:
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
  - { title: "Programming an SBC", url: /learn/intro-hardware/programming-an-sbc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/General-purpose_input/output
---

**GPIO** (general-purpose input/output) are pins on a [single-board computer](/reference/single-board-computer/) or [microcontroller](/reference/microcontroller/) that your code can read or switch on and off to talk to sensors, lights, motors, and other electronics.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A slice of a 40-pin GPIO header showing mixed pin roles. Some pins supply power at fixed voltages, some are ground, and the rest are general-purpose I/O — several of which double as dedicated bus pins for I2C and SPI. One I/O pin is shown wired out to an LED, another to a sensor." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="40" y="46" width="300" height="52" rx="4" fill-opacity="0.05" fill="currentColor"/>
    <g>
      <circle cx="66" cy="64" r="6" fill-opacity="0.3" fill="currentColor"/>
      <circle cx="94" cy="64" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="122" cy="64" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="150" cy="64" r="6" fill-opacity="0.5" fill="currentColor"/>
      <circle cx="178" cy="64" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="206" cy="64" r="6" fill-opacity="0.5" fill="currentColor"/>
      <circle cx="234" cy="64" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="262" cy="64" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="290" cy="64" r="6" fill-opacity="0.3" fill="currentColor"/>
      <circle cx="318" cy="64" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="66" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="94" cy="82" r="6" fill-opacity="0.3" fill="currentColor"/>
      <circle cx="122" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="150" cy="82" r="6" fill-opacity="0.5" fill="currentColor"/>
      <circle cx="178" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="206" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="234" cy="82" r="6" fill-opacity="0.3" fill="currentColor"/>
      <circle cx="262" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="290" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
      <circle cx="318" cy="82" r="6" fill-opacity="0.12" fill="currentColor"/>
    </g>
    <path d="M178 76 V120 H210" stroke-width="1.1"/>
    <circle cx="222" cy="120" r="7" fill-opacity="0.14" fill="currentColor"/>
    <path d="M262 76 V120 H300" stroke-width="1.1"/>
    <rect x="300" y="112" width="26" height="16" rx="2" fill-opacity="0.1" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="66" y="36" font-size="7">3V3</text>
    <text x="150" y="36" font-size="7">SDA</text>
    <text x="206" y="36" font-size="7">SCL</text>
    <text x="290" y="36" font-size="7">GND</text>
    <text x="222" y="140" font-size="7.5">LED (out)</text>
    <text x="313" y="140" font-size="7.5">sensor (in)</text>
    <text x="380" y="60" text-anchor="start" font-size="7.5" fill-opacity="0.9">power / ground</text>
    <text x="380" y="76" text-anchor="start" font-size="7.5" fill-opacity="0.9">I2C · SPI</text>
    <text x="380" y="92" text-anchor="start" font-size="7.5" fill-opacity="0.9">general I/O</text>
  </g>
</svg>
<figcaption>A GPIO header mixes fixed power and ground pins with general-purpose I/O; some I/O pins double as dedicated bus lines (I2C's SDA/SCL, SPI), while any plain pin can be driven as an output to an LED or read as an input from a sensor.</figcaption>
</figure>

## Overview

A GPIO pin can be configured as an input (read a button or sensor) or an output (drive an LED, relay, or motor). It is the most basic form of [input/output](/reference/input-output/) a board offers: at heart each pin is just a voltage the processor can either sense or force high or low. Code reads and writes pins through simple library calls, so blinking an LED or polling a switch is a few lines in [Python](/reference/python-language/), C, or [Go](/reference/go-language/).

Richer interfaces are often layered on top of the same header. Certain pins are wired to hardware for standard buses — [I2C](/reference/i2c/), [SPI](/reference/spi/), UART — so instead of bit-banging a protocol by hand you can hand a whole conversation to dedicated silicon. That is why a header's pinout matters: a given pin may be plain I/O, a bus line, or a fixed power or ground rail, and only some combinations are available at once.

## Pin roles

A typical header pin falls into one of a few roles:

| Pin role | Purpose | Example |
|----------|---------|---------|
| Power | Supply a fixed voltage | 3.3 V, 5 V rails |
| Ground | Return path | GND pins |
| General I/O | Read or drive a signal | Button in, LED out |
| Bus (I2C / SPI / UART) | Talk to peripherals over a protocol | SDA/SCL, MOSI/MISO |
| PWM | Timed pulses for dimming / motors | LED brightness, servo |

## Where it fits

GPIO is the bridge between software and the physical world — it is what sets an SBC or MCU apart from a sealed PC or phone. The [Raspberry Pi](/reference/raspberry-pi/) header and the extensive pinout of the [BeagleBone](/reference/beaglebone/) are common examples; on the microcontroller side, boards like the [Arduino](/reference/arduino/) expose the same idea. In a GopherTrunk capture node, GPIO is how the host reaches beyond the radio itself: switching an antenna relay, reading a temperature sensor in the enclosure, or wiring in a GPS pulse-per-second line for accurate timing.

## Sources

[^wiki]: [General-purpose input/output](https://en.wikipedia.org/wiki/General-purpose_input/output) — Wikipedia, on what GPIO pins are and how they are used.
