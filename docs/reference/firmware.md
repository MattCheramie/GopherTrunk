---
slug: firmware
title: Firmware
entry_type: concept
category: hw-microcontrollers
description: Firmware is low-level software loaded onto a device's chip to control its hardware directly, sitting between pure hardware and ordinary application programs and stored in non-volatile flash so it survives a power cycle.
keywords: firmware, bare metal, flash memory, reflashing, embedded software, non-volatile, bootloader, firmware update
aka: [firmware]
infobox:
  - { label: Type, value: Low-level embedded software }
  - { label: Stored in, value: Non-volatile flash }
  - { label: Updated by, value: Reflashing }
  - { label: On an MCU, value: The whole program (no OS) }
see_also: [microcontroller, bare-metal-programming, esp32, bootloader, in-system-programming, operating-system]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Firmware
---

**Firmware** is low-level software loaded onto a device's chip to control its hardware directly, sitting between the pure hardware and ordinary application programs.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="Two stacked software stacks compared. On a microcontroller the stack is just hardware at the bottom and firmware above it, and the firmware is the whole program. On a general-purpose computer the hardware sits under a thin firmware layer, then an operating system, then applications on top, so firmware is only a small foundation layer." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" text-anchor="middle">
    <rect x="40" y="112" width="150" height="34" rx="3" fill="currentColor" fill-opacity="0.06"/>
    <rect x="40" y="52" width="150" height="58" rx="3" fill="currentColor" fill-opacity="0.16"/>
    <rect x="270" y="112" width="150" height="34" rx="3" fill="currentColor" fill-opacity="0.06"/>
    <rect x="270" y="92" width="150" height="18" rx="3" fill="currentColor" fill-opacity="0.16"/>
    <rect x="270" y="66" width="150" height="24" rx="3" fill="currentColor" fill-opacity="0.10"/>
    <rect x="270" y="40" width="150" height="24" rx="3" fill="currentColor" fill-opacity="0.08"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="115" y="34" font-size="9" font-weight="600">Microcontroller</text>
    <text x="115" y="86" font-size="8.5" font-weight="600">Firmware</text>
    <text x="115" y="99" font-size="7.5" fill-opacity="0.85">= the whole program</text>
    <text x="115" y="133" font-size="8">Hardware</text>
    <text x="345" y="34" font-size="9" font-weight="600">General-purpose PC</text>
    <text x="345" y="56" font-size="8">Applications</text>
    <text x="345" y="82" font-size="8">Operating system</text>
    <text x="345" y="104" font-size="7.5" font-weight="600">Firmware (BIOS/UEFI)</text>
    <text x="345" y="133" font-size="8">Hardware</text>
  </g>
</svg>
<figcaption>Firmware always sits directly on the hardware. On a microcontroller there is nothing above it — the firmware is the entire program. On a full computer it is a thin foundation layer beneath the operating system and applications. Either way it lives in non-volatile flash and changes only by reflashing.</figcaption>
</figure>

## Overview

On a [microcontroller](/reference/microcontroller/) there is no operating system, so the firmware *is* the whole program: it runs [bare metal](/reference/bare-metal-programming/), owning the chip from the moment it powers on. On larger machines firmware is instead a thin layer — the BIOS/UEFI of a PC, the microcode in a disk drive — that boots the hardware and hands off to an [operating system](/reference/operating-system/) above it.

Firmware is stored in non-volatile flash memory so it survives a power cycle, and it is changed by *reflashing* the chip rather than by installing software the usual way. Because it runs so close to the metal, a bad firmware image can render a device unusable ("bricked") until it is reflashed through a recovery path.

## How it is loaded and updated

Getting firmware onto a chip, and replacing it later, follows a small set of paths:

| Method | When | Note |
|--------|------|------|
| [In-system programming](/reference/in-system-programming/) | First flash, bring-up | Via SWD/JTAG/SPI on a blank chip |
| [Bootloader](/reference/bootloader/) | Routine updates | A small resident loader accepts new images over USB/serial |
| Over-the-air | Deployed IoT devices | New image pushed over the network, then reflashed |

## Where it fits

Almost every embedded device runs firmware of some kind — from a thermostat to a Wi-Fi radio. Updating it means writing a new image to flash, which is why "firmware update" and "reflashing" describe the same act. The tiny radios whose signals fill the airwaves GopherTrunk listens to are all driven by firmware of this sort, and SDR dongles are no exception — the tuner and USB chips inside them run their own firmware, with the demodulation done in software on the host.

## Sources

[^wiki]: [Firmware](https://en.wikipedia.org/wiki/Firmware) — Wikipedia, on firmware's role between hardware and software and its storage in flash.
