---
slug: bare-metal-programming
title: Bare-metal programming
entry_type: concept
category: hw-microcontrollers
description: Bare-metal programming means writing software that runs directly on hardware with no operating system underneath, giving full control of the chip's registers, memory, and timing.
keywords: bare metal, bare-metal programming, no OS, registers, super loop, embedded, firmware, direct hardware
infobox:
  - { label: Type, value: Programming approach }
  - { label: OS, value: None }
  - { label: Structure, value: Main loop + interrupts }
  - { label: Control, value: Direct register access }
see_also: [firmware, microcontroller, interrupt, real-time-operating-system, embedded-system, in-system-programming]
related_lessons:
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bare_machine
---

**Bare-metal programming** means writing software that runs directly on the hardware with no operating system underneath it.[^wiki]

## Overview

On a [microcontroller](/reference/microcontroller/), bare-metal code is the [firmware](/reference/firmware/) itself: it configures the chip's peripheral registers by hand, manages its own memory, and typically runs as a "super loop" — an endless main loop that does work, with [interrupts](/reference/interrupt/) handling time-critical events. There is no scheduler, no driver model, and no abstraction between your code and the silicon, which means total control and minimal overhead, but also total responsibility for timing and correctness.

## Trade-offs

Bare metal is the right choice for the simplest, smallest, or most timing-sensitive [embedded systems](/reference/embedded-system/), where an OS would only add size and unpredictability. It boots instantly and wastes no resources. The cost shows up as a project grows: juggling many concurrent jobs by hand becomes error-prone, which is the point at which developers adopt a [real-time operating system](/reference/real-time-operating-system/). New firmware is loaded by [in-system programming](/reference/in-system-programming/) or a bootloader.

## Sources

[^wiki]: [Bare machine](https://en.wikipedia.org/wiki/Bare_machine) — Wikipedia, on running software directly on hardware.
