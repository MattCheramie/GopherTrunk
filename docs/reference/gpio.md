---
slug: gpio
title: GPIO
entry_type: hardware
category: hw-sbc
description: GPIO (general-purpose input/output) are pins on an SBC or microcontroller that code can read or switch on and off to talk to sensors, lights, motors, and other electronics.
keywords: GPIO, general-purpose input output, pins, sensors, microcontroller, Raspberry Pi GPIO, hardware interface
aka: [GPIO]
autolink: true
infobox:
  - { label: Type, value: Digital I/O pins }
  - { label: Found on, value: SBCs, microcontrollers }
  - { label: Controls, value: Sensors, lights, motors }
  - { label: Driven from, value: Code (Python, C, Go) }
see_also: [single-board-computer, microcontroller, input-output, raspberry-pi, beaglebone, arduino]
related_lessons:
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
  - { title: "Programming an SBC", url: /learn/intro-hardware/programming-an-sbc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/General-purpose_input/output
---

**GPIO** (general-purpose input/output) are pins on a [single-board computer](/reference/single-board-computer/) or [microcontroller](/reference/microcontroller/) that your code can read or switch on and off to talk to sensors, lights, motors, and other electronics.[^wiki]

## Overview

A GPIO pin can be configured as an input (read a button or sensor) or an output (drive an LED, relay, or motor). It is the most basic form of [input/output](/reference/input-output/) a board offers, and richer interfaces are often layered on top of the same header. Code reads and writes pins through simple library calls.

## Where it fits

GPIO is the bridge between software and the physical world — it is what sets an SBC or MCU apart from a sealed PC or phone. The [Raspberry Pi](/reference/raspberry-pi/) header and the extensive pinout of the [BeagleBone](/reference/beaglebone/) are common examples; on the microcontroller side, boards like the [Arduino](/reference/arduino/) expose the same idea.

## Sources

[^wiki]: [General-purpose input/output](https://en.wikipedia.org/wiki/General-purpose_input/output) — Wikipedia, on what GPIO pins are and how they are used.
