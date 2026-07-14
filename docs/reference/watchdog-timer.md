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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 196" role="img" aria-label="A sawtooth timeline of a watchdog counter: it falls from full toward zero, and each time the software kicks it the count jumps back to full. After the software hangs the kicks stop, the counter reaches zero, and the watchdog forces a hardware reset." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Kick it in time, or it resets you</text>
  <line x1="34" y1="50" x2="440" y2="50" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4" stroke-dasharray="4 3"/>
  <text x="30" y="53" text-anchor="end" font-size="7.5" fill="currentColor" fill-opacity="0.85">full</text>
  <line x1="34" y1="140" x2="440" y2="140" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4" stroke-dasharray="4 3"/>
  <text x="30" y="143" text-anchor="end" font-size="7.5" fill="currentColor" fill-opacity="0.85">0</text>
  <line x1="34" y1="150" x2="34" y2="40" stroke="currentColor" stroke-width="1" stroke-opacity="0.6" marker-end="url(#wdt_ar)"/>
  <text x="26" y="96" font-size="7.5" fill="currentColor" fill-opacity="0.85" text-anchor="middle" transform="rotate(-90 26 96)">count left</text>
  <path d="M46 50 L120 110 L120 50 L200 110 L200 50 L280 110 L280 50 L430 140" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g stroke="currentColor" stroke-width="1.2">
    <line x1="120" y1="30" x2="120" y2="48" marker-end="url(#wdt_ar)"/>
    <line x1="200" y1="30" x2="200" y2="48" marker-end="url(#wdt_ar)"/>
    <line x1="280" y1="30" x2="280" y2="48" marker-end="url(#wdt_ar)"/>
  </g>
  <g fill="currentColor" font-size="7.5" text-anchor="middle" fill-opacity="0.9">
    <text x="120" y="26">kick</text><text x="200" y="26">kick</text><text x="280" y="26">kick</text>
  </g>
  <text x="352" y="86" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">hang — kicks stop</text>
  <circle cx="430" cy="140" r="3.5" fill="currentColor"/>
  <rect x="356" y="150" width="96" height="24" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
  <text x="404" y="165" text-anchor="middle" font-size="8" fill="currentColor">hardware reset</text>
  <line x1="430" y1="143" x2="430" y2="150" stroke="currentColor" stroke-width="1.1" marker-end="url(#wdt_ar)"/>
  <text x="230" y="188" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">healthy code pets the dog before zero; a crashed one never does — so it reboots</text>
  <defs><marker id="wdt_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The watchdog counts down continuously, and healthy firmware must "kick" it back to full before it reaches zero. Each kick resets the count; but if the code hangs or crashes the kicks stop, the counter runs out, and the watchdog forces a hardware reset — recovering an unattended device with no human present.</figcaption>
</figure>

## Overview

The watchdog counts down continuously; healthy [firmware](/reference/firmware/) must periodically "kick" (reset) it before it reaches zero. If the code hangs in an infinite loop, deadlocks, or crashes, it stops kicking, the counter expires, and the watchdog forces a hardware reset — bringing the device back to a known state with no human intervention. Most [microcontrollers](/reference/microcontroller/) include one as a dedicated peripheral, often clocked independently so it survives even if the main clock fails.

## Where it fits

A watchdog is a core reliability tool for any unattended [embedded system](/reference/embedded-system/): an industrial controller, a remote [sensor](/reference/sensor/) node, or a [bootloader](/reference/bootloader/) guarding a firmware update. The trick is kicking it only when the system is genuinely making progress — kicking it blindly from an [interrupt](/reference/interrupt/) defeats the purpose. It is the safety net under both [bare-metal](/reference/bare-metal-programming/) and [RTOS](/reference/real-time-operating-system/)-based designs.

## Sources

[^wiki]: [Watchdog timer](https://en.wikipedia.org/wiki/Watchdog_timer) — Wikipedia, on watchdog timers and recovery.
