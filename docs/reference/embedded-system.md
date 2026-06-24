---
slug: embedded-system
title: Embedded system
entry_type: concept
category: hw-microcontrollers
description: An embedded system is a computer built into a larger product to perform a dedicated function, usually around a microcontroller with fixed firmware rather than a general-purpose operating system.
keywords: embedded system, dedicated function, firmware, real-time, microcontroller, appliance, deeply embedded
infobox:
  - { label: Type, value: Dedicated-purpose computer }
  - { label: Built around, value: Microcontroller or SoC }
  - { label: Software, value: Firmware, often real-time }
  - { label: Found in, value: Appliances, vehicles, radios }
see_also: [microcontroller, firmware, real-time-operating-system, bare-metal-programming, internet-of-things, sensor]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Embedded_system
---

**An embedded system** is a computer built into a larger product to carry out a dedicated function, rather than a general-purpose machine you load arbitrary programs onto.[^wiki]

## Overview

Most embedded systems are built around a [microcontroller](/reference/microcontroller/) or system-on-chip running fixed [firmware](/reference/firmware/). They are everywhere — in appliances, cars, medical devices, industrial controllers, and radios — usually invisible to the user. Many have hard real-time constraints, met with [interrupts](/reference/interrupt/) and a [real-time operating system](/reference/real-time-operating-system/), while simpler ones run [bare metal](/reference/bare-metal-programming/). They read the physical world through [sensors](/reference/sensor/) and act on it through actuators.

## Where it fits

The defining trade-off is specialization: an embedded system does one job extremely well, cheaply, and reliably, at the cost of flexibility. When a device also needs network connectivity it becomes part of the [Internet of Things](/reference/internet-of-things/). In radio terms, the small transmitters that fill the airwaves GopherTrunk listens to are embedded systems — far too small to run GopherTrunk itself, which lives on a general-purpose computer with an SDR front end.

## Sources

[^wiki]: [Embedded system](https://en.wikipedia.org/wiki/Embedded_system) — Wikipedia, on dedicated-function computers.
