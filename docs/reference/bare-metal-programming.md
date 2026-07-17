---
slug: bare-metal-programming
title: Bare-metal programming
entry_type: concept
category: hw-microcontrollers
description: Bare-metal programming means writing software that runs directly on hardware with no operating system underneath, giving full control of the chip's registers, memory, and timing through a main "super loop" plus interrupts.
keywords: bare metal, bare-metal programming, no OS, registers, super loop, embedded, firmware, direct hardware, interrupts, polling
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="Diagram of a bare-metal super loop. A boot arrow leads into a circular main loop that repeatedly reads inputs, does work, and updates outputs. Separately, a hardware interrupt arrives on its own line, momentarily diverts the processor to a short interrupt handler, and then returns control to the loop where it left off." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <path d="M150 92 A70 44 0 1 1 149 92" marker-mid="none"/>
    <path d="M212 50 L206 44 M212 50 L206 56" stroke-width="1.2"/>
    <line x1="30" y1="118" x2="80" y2="100"/>
    <rect x="330" y="44" width="104" height="30" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <line x1="150" y1="70" x2="150" y2="24"/>
    <path d="M150 24 L330 24 L330 44" />
    <path d="M330 74 L330 92 L162 92" stroke-dasharray="3 3"/>
    <path d="M170 92 L164 86 M170 92 L164 98" stroke-width="1.2"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="150" y="70" font-size="8.5" font-weight="600">read</text>
    <text x="208" y="96" font-size="8.5" font-weight="600">do work</text>
    <text x="150" y="126" font-size="8.5" font-weight="600">update out</text>
    <text x="150" y="150" font-size="8" fill-opacity="0.9">super loop (forever)</text>
    <text x="30" y="112" font-size="8" text-anchor="start">boot</text>
    <text x="382" y="62" font-size="8" font-weight="600">interrupt</text>
    <text x="382" y="88" font-size="7.5" fill-opacity="0.85">handler, then return</text>
  </g>
</svg>
<figcaption>Bare-metal firmware is a "super loop": after boot the processor circles forever — read inputs, do work, update outputs. Time-critical events don't wait their turn; a hardware interrupt briefly diverts the core to a short handler and then hands control back to the loop. No scheduler, no OS — just your code and the silicon.</figcaption>
</figure>

## Overview

On a [microcontroller](/reference/microcontroller/), bare-metal code is the [firmware](/reference/firmware/) itself: it configures the chip's peripheral registers by hand, manages its own memory, and typically runs as a "super loop" — an endless main loop that does work, with [interrupts](/reference/interrupt/) handling time-critical events. There is no scheduler, no driver model, and no abstraction between your code and the silicon, which means total control and minimal overhead, but also total responsibility for timing and correctness.

The two structural tools are *polling* — the loop repeatedly checking a flag or input — and *interrupts*, which let hardware pull the processor aside the instant something needs attention. Getting the balance right between them is much of the craft of bare-metal work.

## Bare metal versus an RTOS

The same chip can run either way; the choice is about how many concurrent jobs the firmware must juggle:

| Aspect | Bare metal | With an [RTOS](/reference/real-time-operating-system/) |
|--------|-----------|------|
| Structure | One super loop + interrupts | Multiple scheduled tasks |
| Overhead | Essentially none | A few KB of code and RAM |
| Concurrency | Hand-managed | Scheduler handles it |
| Boot time | Instant | Near-instant |
| Best for | Simple, tiny, timing-critical | Many competing deadlines |

## Where it fits

Bare metal is the right choice for the simplest, smallest, or most timing-sensitive [embedded systems](/reference/embedded-system/), where an OS would only add size and unpredictability. It boots instantly and wastes no resources. The cost shows up as a project grows: juggling many concurrent jobs by hand becomes error-prone, which is the point at which developers adopt a [real-time operating system](/reference/real-time-operating-system/). New firmware is loaded by [in-system programming](/reference/in-system-programming/) or a bootloader. The small radios and [sensors](/reference/sensor/) whose signals GopherTrunk decodes are very often bare-metal devices — a single loop is all their one job requires.

## Sources

[^wiki]: [Bare machine](https://en.wikipedia.org/wiki/Bare_machine) — Wikipedia, on running software directly on hardware.
