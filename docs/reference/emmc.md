---
slug: emmc
title: eMMC
entry_type: hardware
category: hw-storage
description: eMMC is flash storage with a controller in a single chip soldered onto a board, giving embedded devices and cheaper computers built-in storage without a removable card.
keywords: eMMC, embedded MultiMediaCard, embedded flash, soldered storage, NAND, MMC, JEDEC
aka: [embedded MultiMediaCard]
autolink: true
infobox:
  - { label: Type, value: Embedded flash storage }
  - { label: Medium, value: NAND flash + controller }
  - { label: Mounting, value: Soldered to board (BGA) }
  - { label: Standard, value: JEDEC eMMC }
  - { label: Common use, value: Phones, tablets, SBCs }
see_also: [flash-memory, sd-card, solid-state-drive, jedec, nvme, data-storage]
cite_urls:
  - https://en.wikipedia.org/wiki/MultiMediaCard#eMMC
---

**eMMC (embedded MultiMediaCard)** is a single package that combines [flash memory](/reference/flash-memory/) and its controller, soldered directly onto a device's board as built-in storage.[^wiki]

## Overview

Like an [SD card](/reference/sd-card/), eMMC bundles NAND flash with a controller that handles wear leveling and presents a simple block device — but the package is permanently mounted rather than removable. The interface and command set are standardised by [JEDEC](/reference/jedec/). Because it is fixed in place, eMMC is reliable and compact, but capacities are modest and throughput sits below a modern [NVMe](/reference/nvme/) [SSD](/reference/solid-state-drive/). It is common in phones, tablets, and the cheaper tiers of single-board computers.

## Where it fits

eMMC is the middle ground between a removable SD card and a full SSD: faster and more durable than a typical card, cheaper and smaller than a discrete drive. Some single-board computers offer an eMMC module as a sturdier alternative to booting from microSD. For a GopherTrunk node that runs unattended near an antenna, soldered or socketed eMMC avoids the wear and connection issues that plague constantly written SD cards.

## Sources

[^wiki]: [MultiMediaCard — eMMC](https://en.wikipedia.org/wiki/MultiMediaCard#eMMC) — Wikipedia, on embedded MMC flash storage and its JEDEC standardisation.
