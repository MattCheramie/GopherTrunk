---
slug: watchdog-timer
title: Watchdog timer
entry_type: concept
category: hw-microcontrollers
description: A watchdog timer is a hardware counter that resets a system if software fails to periodically signal it, automatically recovering an embedded device that has hung or crashed.
keywords: watchdog timer, WDT, kick, pet the dog, reset, hang recovery, fail-safe, embedded reliability
aka: [watchdog, WDT]
infobox:
  - { label: Type, value: Hardware timer }
  - { label: Job, value: Reset on software hang }
  - { label: Needs, value: Periodic "kick" from firmware }
  - { label: Benefit, value: Automatic recovery }
see_also: [microcontroller, firmware, embedded-system, interrupt, bare-metal-programming, real-time-operating-system]
related_lessons:
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Watchdog_timer
---

**A watchdog timer (WDT)** is a hardware counter that resets a system if the software fails to signal it within a set interval.[^wiki]

## Overview

The watchdog counts down continuously; healthy [firmware](/reference/firmware/) must periodically "kick" (reset) it before it reaches zero. If the code hangs in an infinite loop, deadlocks, or crashes, it stops kicking, the counter expires, and the watchdog forces a hardware reset — bringing the device back to a known state with no human intervention. Most [microcontrollers](/reference/microcontroller/) include one as a dedicated peripheral, often clocked independently so it survives even if the main clock fails.

## Where it fits

A watchdog is a core reliability tool for any unattended [embedded system](/reference/embedded-system/): an industrial controller, a remote [sensor](/reference/sensor/) node, or a [bootloader](/reference/bootloader/) guarding a firmware update. The trick is kicking it only when the system is genuinely making progress — kicking it blindly from an [interrupt](/reference/interrupt/) defeats the purpose. It is the safety net under both [bare-metal](/reference/bare-metal-programming/) and [RTOS](/reference/real-time-operating-system/)-based designs.

## Sources

[^wiki]: [Watchdog timer](https://en.wikipedia.org/wiki/Watchdog_timer) — Wikipedia, on watchdog timers and recovery.
