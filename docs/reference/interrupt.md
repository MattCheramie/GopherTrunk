---
slug: interrupt
title: Interrupt
entry_type: concept
category: hw-microcontrollers
description: An interrupt is a hardware signal that pauses a processor's normal flow to run a short handler in response to an event, letting a microcontroller react instantly without constantly polling.
keywords: interrupt, ISR, interrupt service routine, IRQ, NVIC, polling, event, latency, vector table
infobox:
  - { label: Type, value: Hardware event mechanism }
  - { label: Triggers, value: Timers, I/O, pins, peripherals }
  - { label: Runs, value: An interrupt service routine (ISR) }
  - { label: Benefit, value: React without polling }
see_also: [microcontroller, real-time-operating-system, arm-cortex-m, bare-metal-programming, gpio, watchdog-timer]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Interrupt
---

**An interrupt** is a hardware signal that pauses a processor's normal program flow to run a short handler in response to an event, then resumes where it left off.[^wiki]

## Overview

When a peripheral needs attention — a timer expires, a byte arrives on a [UART](/reference/uart/), a [GPIO](/reference/gpio/) pin changes — it raises an interrupt. The processor jumps to the matching **interrupt service routine (ISR)** via a vector table, runs it, and returns. On [ARM Cortex-M](/reference/arm-cortex-m/) parts a nested vectored interrupt controller (NVIC) prioritizes and arbitrates these requests with deterministic latency. The alternative, polling, wastes cycles repeatedly checking; interrupts let a [microcontroller](/reference/microcontroller/) sleep until something actually happens.

## Where it fits

Interrupts are the foundation of responsive, low-power embedded code: an MCU can idle in a low-power mode and wake only on an event, which is how battery devices run for years. ISRs are kept short, often just setting a flag for the main loop. In [bare-metal](/reference/bare-metal-programming/) firmware interrupts plus a main loop are the whole architecture; an [RTOS](/reference/real-time-operating-system/) builds its scheduler on top of them.

## Sources

[^wiki]: [Interrupt](https://en.wikipedia.org/wiki/Interrupt) — Wikipedia, on interrupts and service routines.
