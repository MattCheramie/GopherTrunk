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

## Overview

Classic *mask ROM* has its data baked in during manufacture and can never be changed. Later variants made ROM progressively more editable: PROM (programmable once), EPROM (erasable with ultraviolet light), and EEPROM (electrically erasable). [Flash memory](/reference/flash-memory/) is the modern, block-erasable descendant of EEPROM and now fills most roles once handled by ROM chips. Despite the name, today's "ROM" is usually rewritable — but only deliberately, which is the point: it survives power loss and is not casually overwritten like [RAM](/reference/random-access-memory/).

## Where it fits

ROM's job is to store the [firmware](/reference/firmware/) and boot code that bring hardware to life, such as a [BIOS/UEFI](/reference/bios-uefi/) on a PC or the bootloader on a microcontroller. It is the [non-volatile](/reference/volatile-memory/) counterpart to volatile RAM: RAM holds the running program and is wiped at power-off, while ROM holds the unchanging instructions that start the machine. In a GopherTrunk capture node, the SDR dongle and the host both rely on firmware held in this kind of memory.

## Sources

[^wiki]: [Read-only memory](https://en.wikipedia.org/wiki/Read-only_memory) — Wikipedia, on ROM, its PROM/EPROM/EEPROM variants, and firmware storage.
