---
slug: read-only-memory
title: Read-only memory (ROM)
entry_type: hardware
category: hw-storage
description: Read-only memory is non-volatile memory whose contents are fixed or rarely changed, used to hold firmware and boot code that must survive power loss.
keywords: read-only memory, ROM, PROM, EPROM, EEPROM, firmware, mask ROM, non-volatile, boot code
aka: [ROM]
infobox:
  - { label: Type, value: Non-volatile memory }
  - { label: Holds, value: Firmware, boot code }
  - { label: Variants, value: Mask ROM, PROM, EPROM, EEPROM }
  - { label: Retention, value: Keeps data without power }
  - { label: Contrast, value: vs volatile RAM }
see_also: [random-access-memory, flash-memory, volatile-memory, firmware, memory-hierarchy, bios-uefi]
cite_urls:
  - https://en.wikipedia.org/wiki/Read-only_memory
---

**Read-only memory (ROM)** is non-volatile memory whose contents are written once or rarely and retained without power, traditionally used to hold code a device needs the moment it switches on.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 214" role="img" aria-label="A mask-ROM array: a decoder drives one horizontal word line at a time across a grid of vertical bit lines, and a transistor is present or absent at each crossing. That fixed pattern of present-versus-absent connections is the stored data, etched into the wiring at manufacture and unchangeable." xmlns="http://www.w3.org/2000/svg">
  <text x="220" y="18" text-anchor="middle" font-size="10.5" fill="currentColor" font-weight="600">Mask ROM: the data is the wiring pattern</text>
  <rect x="24" y="52" width="50" height="120" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/>
  <text x="49" y="108" text-anchor="middle" font-size="8" fill="currentColor">row</text>
  <text x="49" y="119" text-anchor="middle" font-size="8" fill="currentColor">decoder</text>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.7">
    <line x1="74" y1="68" x2="360" y2="68"/>
    <line x1="74" y1="98" x2="360" y2="98"/>
    <line x1="74" y1="128" x2="360" y2="128"/>
    <line x1="74" y1="158" x2="360" y2="158"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" stroke-opacity="0.4">
    <line x1="110" y1="56" x2="110" y2="182"/>
    <line x1="150" y1="56" x2="150" y2="182"/>
    <line x1="190" y1="56" x2="190" y2="182"/>
    <line x1="230" y1="56" x2="230" y2="182"/>
    <line x1="270" y1="56" x2="270" y2="182"/>
    <line x1="310" y1="56" x2="310" y2="182"/>
  </g>
  <g fill="currentColor">
    <circle cx="150" cy="68" r="3.4"/><circle cx="230" cy="68" r="3.4"/><circle cx="270" cy="68" r="3.4"/>
    <circle cx="110" cy="98" r="3.4"/><circle cx="190" cy="98" r="3.4"/><circle cx="310" cy="98" r="3.4"/>
    <circle cx="110" cy="128" r="3.4"/><circle cx="150" cy="128" r="3.4"/><circle cx="230" cy="128" r="3.4"/><circle cx="270" cy="128" r="3.4"/>
    <circle cx="190" cy="158" r="3.4"/><circle cx="270" cy="158" r="3.4"/><circle cx="310" cy="158" r="3.4"/>
  </g>
  <text x="366" y="60" text-anchor="start" font-size="7.5" fill="currentColor" fill-opacity="0.85">bit lines</text>
  <text x="366" y="128" text-anchor="start" font-size="7.5" fill="currentColor" fill-opacity="0.9">● present = 1</text>
  <text x="366" y="140" text-anchor="start" font-size="7.5" fill="currentColor" fill-opacity="0.85">blank = 0</text>
  <text x="220" y="200" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">the 1s and 0s are fixed by the factory mask — non-volatile and read-only</text>
</svg>
<figcaption>Classic mask ROM stores each bit physically: at every crossing of a word line and a bit line there either is or isn't a transistor, and that pattern is decided by the photomask at manufacture. Reading just selects a row and senses the columns. Nothing rewrites it in normal use, and it keeps its contents with the power off.</figcaption>
</figure>

## Overview

Classic *mask ROM* has its data baked in during manufacture and can never be changed. Later variants made ROM progressively more editable: PROM (programmable once), EPROM (erasable with ultraviolet light), and EEPROM (electrically erasable). [Flash memory](/reference/flash-memory/) is the modern, block-erasable descendant of EEPROM and now fills most roles once handled by ROM chips. Despite the name, today's "ROM" is usually rewritable — but only deliberately, which is the point: it survives power loss and is not casually overwritten like [RAM](/reference/random-access-memory/).

## Where it fits

ROM's job is to store the [firmware](/reference/firmware/) and boot code that bring hardware to life, such as a [BIOS/UEFI](/reference/bios-uefi/) on a PC or the bootloader on a microcontroller. It is the [non-volatile](/reference/volatile-memory/) counterpart to volatile RAM: RAM holds the running program and is wiped at power-off, while ROM holds the unchanging instructions that start the machine. In a GopherTrunk capture node, the SDR dongle and the host both rely on firmware held in this kind of memory.

## Sources

[^wiki]: [Read-only memory](https://en.wikipedia.org/wiki/Read-only_memory) — Wikipedia, on ROM, its PROM/EPROM/EEPROM variants, and firmware storage.
