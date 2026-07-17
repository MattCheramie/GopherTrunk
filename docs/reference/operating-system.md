---
slug: operating-system
title: Operating system
entry_type: concept
category: hw-foundations
description: An operating system is the software that manages a device's hardware and resources and gives other programs a consistent, shared way to run.
keywords: operating system, OS, kernel, Linux, Windows, macOS, Android, iOS, bare metal, scheduler, resource management
aka: [operating system, OS]
infobox:
  - { label: Type, value: System software }
  - { label: Core, value: The kernel }
  - { label: Manages, value: CPU, memory, I/O, files }
  - { label: Examples, value: Linux, Windows, macOS, Android, iOS }
  - { label: Small devices, value: Often none (bare metal) }
see_also: [computer-hardware, firmware, microcontroller, single-board-computer, input-output]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Operating_system
---

**An operating system (OS)** is the software that manages a device's hardware and resources and gives other programs a consistent, shared way to run.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A layered stack. At the bottom is computer hardware; above it the operating system kernel manages the CPU, memory, and input-output; on top run the applications, which reach the hardware only by asking the kernel through system calls." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="40" y="18" width="380" height="30" rx="4"/>
    <rect x="40" y="62" width="380" height="46" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <rect x="40" y="122" width="380" height="28" rx="4"/>
    <line x1="150" y1="62" x2="150" y2="108" stroke-opacity="0.6"/>
    <line x1="270" y1="62" x2="270" y2="108" stroke-opacity="0.6"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none" stroke-opacity="0.75">
    <path d="M120 48 v14 m-3 -6 l3 6 l3 -6"/>
    <path d="M230 48 v14 m-3 -6 l3 6 l3 -6"/>
    <path d="M340 48 v14 m-3 -6 l3 6 l3 -6"/>
    <path d="M230 108 v14 m-3 -6 l3 6 l3 -6"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="230" y="37" font-size="9">Applications</text>
    <text x="230" y="80" font-size="9" font-weight="600">Operating system (kernel)</text>
    <text x="95" y="98" font-size="7.5" fill-opacity="0.85">schedule CPU</text>
    <text x="210" y="98" font-size="7.5" fill-opacity="0.85">manage memory</text>
    <text x="345" y="98" font-size="7.5" fill-opacity="0.85">handle I/O</text>
    <text x="230" y="140" font-size="9">Computer hardware</text>
    <text x="365" y="58" font-size="7" fill-opacity="0.8">system calls</text>
  </g>
</svg>
<figcaption>Applications sit on top, hardware at the bottom, and the OS kernel in the middle — programs reach the CPU, memory, and devices only by asking the kernel, which shares those resources fairly among everything running.</figcaption>
</figure>

## Overview

The OS sits between [computer hardware](/reference/computer-hardware/) and the applications on top of it. Its core is the *kernel*, the always-resident code that decides which program gets the [CPU](/reference/central-processing-unit/) and for how long (scheduling), how much memory each one holds and how they are kept from stepping on each other, and how requests to devices are handled through [input/output](/reference/input-output/).

Just as important, the OS presents a *uniform interface* — a file system, network sockets, a way to start and stop programs — so an application need not know the exact model of disk or network card underneath. Programs ask for services through *system calls* rather than touching hardware directly, which is what lets the same app run on very different machines. Common examples are Linux, Windows, and macOS on larger computers, and Android and iOS on phones.

## What an OS manages

The kernel's job is really the disciplined sharing of a handful of scarce hardware resources:

| Resource | What the OS does | Without it |
|----------|------------------|-----------|
| CPU time | Schedules which program runs when | One program could hog the machine |
| Memory | Allocates and isolates each program's RAM | Programs corrupt each other |
| Storage | Presents files and directories | Raw block numbers, no structure |
| Devices | Drivers + a common I/O interface | Every app needs its own driver |

By owning these, the OS lets many programs share one computer safely and lets developers write against a stable interface instead of bare metal.

## Where it fits

Not every device has an OS. A [microcontroller](/reference/microcontroller/) often runs **bare metal** — a single program written straight to the chip as [firmware](/reference/firmware/), with no operating system in between. The larger and more general-purpose a device is, the more it leans on an OS to juggle many programs at once. A GopherTrunk decode node typically runs Linux, whether on an x86 box or a [single-board computer](/reference/single-board-computer/) by the antenna, and relies on the OS to schedule the daemon and stream data off the SDR. (This entry is the hardware-tier view; the software-development guide covers operating systems from the programming side.)

## Sources

[^wiki]: [Operating system](https://en.wikipedia.org/wiki/Operating_system) — Wikipedia, on OS resource management, the kernel, scheduling, and examples.
