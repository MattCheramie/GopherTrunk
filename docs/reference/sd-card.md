---
slug: sd-card
title: SD card
entry_type: hardware
category: hw-storage
description: An SD card is a small removable flash-memory card used for storage in cameras, phones, and single-board computers, available in SD, microSD, and high-capacity variants.
keywords: SD card, microSD, SDHC, SDXC, flash card, memory card, speed class, removable storage
aka: [Secure Digital, microSD]
infobox:
  - { label: Type, value: Removable flash storage }
  - { label: Medium, value: NAND flash memory }
  - { label: Sizes, value: SD, miniSD, microSD }
  - { label: Capacity tiers, value: SD, SDHC, SDXC, SDUC }
  - { label: Common use, value: Cameras, phones, SBCs }
see_also: [flash-memory, emmc, solid-state-drive, data-storage, raspberry-pi, file-system]
cite_urls:
  - https://en.wikipedia.org/wiki/SD_card
---

An **SD card** (Secure Digital) is a small removable storage card built around [flash memory](/reference/flash-memory/), widely used in cameras, phones, and small computers.[^wiki]

## Overview

The standard defines several physical sizes — full-size SD, miniSD, and the tiny microSD — and capacity tiers that grew over time: SDHC, SDXC, and SDUC. A small controller inside the card handles wear leveling and presents a simple block device, so the host just sees storage formatted with a [file system](/reference/file-system/). Speed classes (Class 10, UHS, V30, and so on) rate sustained write performance, which matters for high-bitrate recording.

## Where it fits

The SD card is the default boot and storage medium for many single-board computers, including the [Raspberry Pi](/reference/raspberry-pi/), which boots from a microSD card by default. It is cheap and removable, but cards vary in quality and endurance, and a low-end card can wear out under constant logging. A GopherTrunk capture node can run from an SD card, though for heavy continuous writes an [eMMC](/reference/emmc/) module or an [SSD](/reference/solid-state-drive/) lasts longer.

## Sources

[^wiki]: [SD card](https://en.wikipedia.org/wiki/SD_card) — Wikipedia, on Secure Digital cards, form factors, and capacity tiers.
