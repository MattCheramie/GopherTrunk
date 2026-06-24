---
slug: firmware
title: Firmware
entry_type: concept
category: hw-microcontrollers
description: Firmware is low-level software loaded onto a device's chip to control its hardware directly, sitting between pure hardware and ordinary application programs.
keywords: firmware, bare metal, flash memory, reflashing, embedded software, non-volatile
aka: [firmware]
infobox:
  - { label: Type, value: Low-level embedded software }
  - { label: Stored in, value: Non-volatile flash }
  - { label: Updated by, value: Reflashing }
  - { label: On an MCU, value: The whole program (no OS) }
see_also: [microcontroller, esp32, arduino, operating-system]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Firmware
---

**Firmware** is low-level software loaded onto a device's chip to control its hardware directly, sitting between the pure hardware and ordinary application programs.[^wiki]

## Overview

On a [microcontroller](/reference/microcontroller/) there is no operating system, so the firmware *is* the whole program: it runs bare metal, owning the chip from the moment it powers on. Firmware is stored in non-volatile flash memory so it survives a power cycle, and it is changed by reflashing the chip rather than by installing software the usual way.

## Where it fits

Almost every embedded device runs firmware of some kind — from a thermostat to a Wi-Fi radio. Updating it means writing a new image to flash, which is why "firmware update" and "reflashing" describe the same act. The tiny radios whose signals fill the airwaves GopherTrunk listens to are all driven by firmware of this sort.

## Sources

[^wiki]: [Firmware](https://en.wikipedia.org/wiki/Firmware) — Wikipedia, on firmware's role between hardware and software and its storage in flash.
