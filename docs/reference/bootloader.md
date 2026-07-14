---
slug: bootloader
title: Bootloader
entry_type: concept
category: hw-microcontrollers
description: A bootloader is a small program that runs first at power-on and loads the main firmware or operating system; on microcontrollers it also enables reflashing without external hardware.
keywords: bootloader, boot, DFU, firmware update, reflash, USB bootloader, second stage, OTA
infobox:
  - { label: Type, value: Startup program }
  - { label: Runs, value: First, at power-on/reset }
  - { label: Job, value: Load and start firmware }
  - { label: Bonus, value: Field reprogramming }
see_also: [firmware, in-system-programming, microcontroller, embedded-system, bare-metal-programming, watchdog-timer]
related_lessons:
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bootloader
---

**A bootloader** is a small program that runs first when a device powers on or resets, then loads and starts the main [firmware](/reference/firmware/) or operating system.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="A boot handoff chain reading left to right: power-on or reset starts built-in ROM firmware, which hands control to the bootloader, which loads and verifies the operating-system kernel or application, which then takes over. Each stage passes control to the next in boot order." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Each stage hands control to the next</text>
  <g font-size="8.5" text-anchor="middle" fill="currentColor">
    <rect x="20" y="46" width="84" height="42" rx="5" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.1"/><text x="62" y="64">power-on</text><text x="62" y="76">/ reset</text>
    <rect x="128" y="46" width="84" height="42" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="170" y="64">ROM</text><text x="170" y="76">firmware</text>
    <rect x="236" y="46" width="84" height="42" rx="5" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/><text x="278" y="70" font-weight="600">bootloader</text>
    <rect x="344" y="46" width="96" height="42" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="392" y="64">OS kernel</text><text x="392" y="76">/ app</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3">
    <line x1="106" y1="67" x2="126" y2="67" marker-end="url(#bl_ar)"/>
    <line x1="214" y1="67" x2="234" y2="67" marker-end="url(#bl_ar)"/>
    <line x1="322" y1="67" x2="342" y2="67" marker-end="url(#bl_ar)"/>
  </g>
  <g font-size="7" text-anchor="middle" fill="currentColor" fill-opacity="0.85">
    <text x="62" y="104">runs first</text>
    <text x="170" y="104">built-in</text>
    <text x="278" y="102">loads &amp; verifies</text>
    <text x="278" y="112">image · can reflash</text>
    <text x="392" y="104">your program</text>
  </g>
  <line x1="20" y1="134" x2="440" y2="134" stroke="currentColor" stroke-width="1" stroke-opacity="0.6" marker-end="url(#bl_ar)"/>
  <text x="230" y="150" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">boot order (time) →</text>
  <defs><marker id="bl_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Booting is a relay of control: power-on starts built-in ROM firmware, which hands off to the bootloader, which loads and verifies the OS or application and jumps to it. The bootloader is the small program in the middle — and because it can also reflash the image, it is how a board updates itself over USB or the air.</figcaption>
</figure>

## Overview

On a desktop the bootloader chain hands off to an OS; on a [microcontroller](/reference/microcontroller/) the bootloader is often the only thing between reset and your application. Many MCUs ship with a factory bootloader in ROM that speaks USB DFU, [UART](/reference/uart/), or [SPI](/reference/spi/)/[I²C](/reference/i2c/), so new code can be loaded without a dedicated programmer. Custom bootloaders add over-the-air (OTA) updates, integrity checks, and fallback to a known-good image if an update fails.

## Where it fits

A bootloader is what lets you reflash a board over USB instead of always using [in-system programming](/reference/in-system-programming/) with an external debugger — the Arduino "press reset, upload sketch" flow is exactly this. In a fielded [embedded system](/reference/embedded-system/) it is the safe path for firmware updates, often paired with a [watchdog timer](/reference/watchdog-timer/) so a hung update cannot brick the device.

## Sources

[^wiki]: [Bootloader](https://en.wikipedia.org/wiki/Bootloader) — Wikipedia, on bootloaders and their role at startup.
