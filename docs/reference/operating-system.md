---
slug: operating-system
title: Operating system
entry_type: concept
category: hw-foundations
description: An operating system is the software that manages a device's hardware and resources and gives other programs a consistent way to run.
keywords: operating system, OS, Linux, Windows, macOS, Android, iOS, bare metal, kernel
aka: [operating system, OS]
infobox:
  - { label: Type, value: System software }
  - { label: Manages, value: Hardware and resources }
  - { label: Examples, value: Linux, Windows, macOS, Android, iOS }
  - { label: Small devices, value: Often none (bare metal) }
see_also: [computer-hardware, firmware, microcontroller, single-board-computer, input-output]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Operating_system
---

**An operating system (OS)** is the software that manages a device's hardware and resources and gives other programs a consistent, shared way to run.[^wiki]

## Overview
The OS sits between [computer hardware](/reference/computer-hardware/) and the applications on top of it. It decides which program gets the [CPU](/reference/central-processing-unit/) and how much memory each one holds, handles [input/output](/reference/input-output/), and presents a uniform interface so an app need not know the exact details of the chips underneath. Common examples are Linux, Windows, and macOS on larger machines, and Android and iOS on phones.

## Where it fits
Not every device has an OS. A [microcontroller](/reference/microcontroller/) often runs **bare metal** — a single program written straight to the chip as [firmware](/reference/firmware/), with no operating system in between. The larger and more general-purpose a device is, the more it leans on an OS to juggle many programs at once. (This entry is the hardware-tier view; the software-development guide covers operating systems from the programming side.)

## Sources
[^wiki]: [Operating system](https://en.wikipedia.org/wiki/Operating_system) — Wikipedia, on OS resource management and examples.
