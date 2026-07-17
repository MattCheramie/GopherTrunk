---
slug: real-time-operating-system
title: Real-time operating system (RTOS)
entry_type: concept
category: hw-microcontrollers
description: A real-time operating system is a small OS that schedules tasks with guaranteed, bounded timing, used on microcontrollers where responses must happen within strict deadlines; its priority-based preemptive scheduler makes worst-case latency predictable.
keywords: RTOS, real-time operating system, FreeRTOS, Zephyr, ThreadX, task scheduling, deadline, preemptive, deterministic, priority
aka: [RTOS]
infobox:
  - { label: Type, value: Operating system class }
  - { label: Goal, value: Bounded, predictable timing }
  - { label: Scheduling, value: Priority-based, preemptive }
  - { label: Examples, value: FreeRTOS, Zephyr, ThreadX }
  - { label: Footprint, value: Kilobytes }
see_also: [microcontroller, bare-metal-programming, interrupt, embedded-system, arm-cortex-m, firmware]
related_lessons:
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Real-time_operating_system
---

**A real-time operating system (RTOS)** is a small operating system that schedules tasks with guaranteed, bounded timing, so a response is delivered within a known deadline.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="A preemptive scheduling timeline with three task priority lanes over time. A low-priority task runs until a medium-priority task becomes ready and preempts it; partway through, a high-priority task becomes ready, preempts the medium task, runs to completion, and then control falls back down to the medium task and finally the low task, which resumes where it left off." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none" stroke-opacity="0.5">
    <line x1="70" y1="44" x2="440" y2="44"/>
    <line x1="70" y1="80" x2="440" y2="80"/>
    <line x1="70" y1="116" x2="440" y2="116"/>
    <line x1="70" y1="132" x2="440" y2="132"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.16" stroke-width="1.1">
    <rect x="200" y="32" width="70" height="18"/>
    <rect x="130" y="68" width="70" height="18"/>
    <rect x="270" y="68" width="60" height="18"/>
    <rect x="70" y="104" width="60" height="18"/>
    <rect x="330" y="104" width="110" height="18"/>
  </g>
  <g stroke="currentColor" stroke-width="0.9" stroke-dasharray="2 2" stroke-opacity="0.6">
    <line x1="130" y1="122" x2="130" y2="68"/>
    <line x1="200" y1="86" x2="200" y2="32"/>
    <line x1="270" y1="50" x2="270" y2="86"/>
    <line x1="330" y1="86" x2="330" y2="104"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="10" y="45" text-anchor="start">High</text>
    <text x="10" y="81" text-anchor="start">Medium</text>
    <text x="10" y="117" text-anchor="start">Low</text>
    <text x="235" y="45" text-anchor="middle" fill-opacity="0.9">runs</text>
    <text x="255" y="150" text-anchor="middle" font-size="7.5" fill-opacity="0.85">time &#8594; &#183; higher priority always preempts lower</text>
  </g>
</svg>
<figcaption>Preemptive priority scheduling: the low task runs until a medium task becomes ready and preempts it, then a high-priority task preempts in turn, finishes, and control falls back down — the low task resuming only once nothing higher is waiting. Because the highest-priority ready task always runs, worst-case response time is bounded and predictable.</figcaption>
</figure>

## Overview

What makes an OS "real-time" is not raw speed but *determinism*: the worst-case time to react to an event is predictable and small. An RTOS provides a priority-based, usually preemptive scheduler, plus primitives like tasks, queues, semaphores, and timers, all in a footprint of a few kilobytes.

On a [microcontroller](/reference/microcontroller/) it sits between your application and the hardware, multiplexing several jobs onto one core while honoring deadlines. The scheduler guarantees that the highest-priority ready task is the one running, so a time-critical job is never left waiting behind less important work. Popular examples include FreeRTOS, Zephyr, and ThreadX.

## RTOS versus bare metal

The same chip can run either way; the trade is structure and guarantees against overhead:

| Aspect | Bare metal | RTOS |
|--------|-----------|------|
| Concurrency | Hand-coded super loop | Scheduled tasks |
| Timing guarantee | Ad hoc | Bounded worst case |
| Overhead | None | A few KB code + RAM |
| Primitives | You build them | Tasks, queues, semaphores |
| Sweet spot | 1–2 jobs | Many competing deadlines |

## Where it fits

Simple firmware often runs [bare metal](/reference/bare-metal-programming/) — a single loop plus [interrupts](/reference/interrupt/) — which is enough when there are only a couple of jobs. An RTOS earns its keep once an [embedded system](/reference/embedded-system/) juggles many concurrent tasks (reading [sensors](/reference/sensor/), driving a radio, servicing a network stack) with competing deadlines. It ports easily across [ARM Cortex-M](/reference/arm-cortex-m/) parts, so the same code runs on many chips. The same determinism argument scales up: a busy SDR decode host leans on its general-purpose OS scheduler to keep real-time sample streams flowing without dropouts.

## Sources

[^wiki]: [Real-time operating system](https://en.wikipedia.org/wiki/Real-time_operating_system) — Wikipedia, on RTOS design and determinism.
