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

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 176" role="img" aria-label="A timeline of the main program running. An event raises an interrupt; the processor jumps via the vector table down to an interrupt service routine, runs it, then returns and resumes the main program where it left off." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="60" x2="150" y2="60" stroke="currentColor" stroke-width="1.8"/>
  <line x1="150" y1="60" x2="352" y2="60" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" stroke-opacity="0.5"/>
  <line x1="352" y1="60" x2="440" y2="60" stroke="currentColor" stroke-width="1.8"/>
  <text x="88" y="50" text-anchor="middle" font-size="8.5" fill="currentColor">main program</text>
  <text x="398" y="50" text-anchor="middle" font-size="8.5" fill="currentColor">resumes</text>
  <line x1="150" y1="60" x2="150" y2="30" stroke="currentColor" stroke-width="1.2" marker-end="url(#ir_ar)"/>
  <text x="150" y="22" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">interrupt raised</text>
  <line x1="150" y1="60" x2="196" y2="102" stroke="currentColor" stroke-width="1.3" marker-end="url(#ir_ar)"/>
  <text x="150" y="92" text-anchor="end" font-size="7.5" fill="currentColor" fill-opacity="0.85">vector → jump</text>
  <rect x="196" y="104" width="120" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
  <text x="256" y="121" text-anchor="middle" font-size="8.5" fill="currentColor" font-weight="600">ISR — handler</text>
  <text x="256" y="132" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">short: set a flag</text>
  <line x1="316" y1="112" x2="352" y2="66" stroke="currentColor" stroke-width="1.3" marker-end="url(#ir_ar)"/>
  <text x="360" y="98" text-anchor="start" font-size="7.5" fill="currentColor" fill-opacity="0.85">return</text>
  <text x="235" y="164" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">the CPU can sleep until an event wakes it — no wasteful polling</text>
  <defs><marker id="ir_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Instead of polling, the processor runs the main program until a peripheral raises an interrupt — a timer, a UART byte, a GPIO change. It jumps through the vector table to the matching service routine, runs it, and returns to where it left off. Between events the CPU can idle in low power, which is how battery devices last for years.</figcaption>
</figure>

## Overview

When a peripheral needs attention — a timer expires, a byte arrives on a [UART](/reference/uart/), a [GPIO](/reference/gpio/) pin changes — it raises an interrupt. The processor jumps to the matching **interrupt service routine (ISR)** via a vector table, runs it, and returns. On [ARM Cortex-M](/reference/arm-cortex-m/) parts a nested vectored interrupt controller (NVIC) prioritizes and arbitrates these requests with deterministic latency. The alternative, polling, wastes cycles repeatedly checking; interrupts let a [microcontroller](/reference/microcontroller/) sleep until something actually happens.

## Where it fits

Interrupts are the foundation of responsive, low-power embedded code: an MCU can idle in a low-power mode and wake only on an event, which is how battery devices run for years. ISRs are kept short, often just setting a flag for the main loop. In [bare-metal](/reference/bare-metal-programming/) firmware interrupts plus a main loop are the whole architecture; an [RTOS](/reference/real-time-operating-system/) builds its scheduler on top of them.

## Sources

[^wiki]: [Interrupt](https://en.wikipedia.org/wiki/Interrupt) — Wikipedia, on interrupts and service routines.
