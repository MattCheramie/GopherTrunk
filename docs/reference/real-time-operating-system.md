---
slug: real-time-operating-system
title: Real-time operating system (RTOS)
entry_type: concept
category: hw-microcontrollers
description: A real-time operating system is a small OS that schedules tasks with guaranteed, bounded timing, used on microcontrollers where responses must happen within strict deadlines.
keywords: RTOS, real-time operating system, FreeRTOS, Zephyr, task scheduling, deadline, preemptive, deterministic
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

## Overview

What makes an OS "real-time" is not raw speed but *determinism*: the worst-case time to react to an event is predictable and small. An RTOS provides a priority-based, usually preemptive scheduler, plus primitives like tasks, queues, semaphores, and timers, all in a footprint of a few kilobytes. On a [microcontroller](/reference/microcontroller/) it sits between your application and the hardware, multiplexing several jobs onto one core while honoring deadlines. Popular examples include FreeRTOS, Zephyr, and ThreadX.

## Where it fits

Simple firmware often runs [bare metal](/reference/bare-metal-programming/) — a single loop plus [interrupts](/reference/interrupt/) — which is enough when there are only a couple of jobs. An RTOS earns its keep once an [embedded system](/reference/embedded-system/) juggles many concurrent tasks (reading [sensors](/reference/sensor/), driving a radio, servicing a network stack) with competing deadlines. It ports easily across [ARM Cortex-M](/reference/arm-cortex-m/) parts, so the same code runs on many chips.

## Sources

[^wiki]: [Real-time operating system](https://en.wikipedia.org/wiki/Real-time_operating_system) — Wikipedia, on RTOS design and determinism.
