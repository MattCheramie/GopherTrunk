---
slug: sd-card
title: SD card
entry_type: hardware
category: hw-storage
description: An SD card is a small removable flash-memory card used for storage in cameras, phones, and single-board computers, available in SD, microSD, and high-capacity variants.
keywords: SD card, microSD, SDHC, SDXC, flash card, memory card, speed class, removable storage, UHS, V30
aka: [Secure Digital, microSD]
infobox:
  - { label: Type, value: Removable flash storage }
  - { label: Medium, value: NAND flash memory }
  - { label: Sizes, value: SD, miniSD, microSD }
  - { label: Capacity tiers, value: SD, SDHC, SDXC, SDUC }
  - { label: Common use, value: Cameras, phones, SBCs }
see_also: [flash-memory, emmc, solid-state-drive, data-storage, raspberry-pi, file-system, single-board-computer]
cite_urls:
  - https://en.wikipedia.org/wiki/SD_card
---

An **SD card** (Secure Digital) is a small removable storage card built around [flash memory](/reference/flash-memory/), widely used in cameras, phones, and small computers.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A stylised SD card showing its notched corner, a row of gold contact pads along the bottom edge, and a speed-class rating printed on the face. A callout explains that the class number, such as V30, states the minimum sustained write speed the card guarantees for recording." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <path d="M120 24 H300 V126 H120 V44 L140 24 Z" fill="currentColor" fill-opacity="0.05"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.3" stroke-width="0.6">
    <rect x="132" y="104" width="12" height="16" rx="1"/>
    <rect x="150" y="104" width="12" height="16" rx="1"/>
    <rect x="168" y="104" width="12" height="16" rx="1"/>
    <rect x="186" y="104" width="12" height="16" rx="1"/>
    <rect x="204" y="104" width="12" height="16" rx="1"/>
    <rect x="222" y="104" width="12" height="16" rx="1"/>
    <rect x="240" y="104" width="12" height="16" rx="1"/>
    <rect x="258" y="104" width="12" height="16" rx="1"/>
  </g>
  <g fill="currentColor" stroke="none">
    <text x="210" y="58" font-size="10" text-anchor="middle" font-weight="600">32 GB</text>
    <text x="210" y="82" font-size="9" text-anchor="middle">V30 &#183; UHS-I</text>
    <text x="195" y="138" font-size="7.5" text-anchor="middle" fill-opacity="0.9">gold contact pads</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1">
    <path d="M300 70 H340" stroke-dasharray="2 3" stroke-opacity="0.6"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="346" y="60">speed class</text>
    <text x="346" y="72">(here V30):</text>
    <text x="346" y="84">min sustained</text>
    <text x="346" y="96">write &#8805; 30 MB/s</text>
  </g>
  <text x="141" y="20" font-size="7.5" fill="currentColor" fill-opacity="0.85">notched corner</text>
</svg>
<figcaption>A row of gold pads carries the SD bus, the notched corner keys the card into its slot, and the printed speed class (here V30) promises a minimum sustained write rate — the number that matters for continuous recording.</figcaption>
</figure>

## Overview

The standard defines several physical sizes — full-size SD, miniSD, and the tiny microSD — and capacity tiers that grew over time: SDHC, SDXC, and SDUC. A small controller inside the card handles wear leveling, bad-block remapping, and error correction, then presents a simple block device, so the host just sees storage formatted with a [file system](/reference/file-system/) (usually FAT or exFAT).

Ratings on the card describe two different things. The *capacity tier* (SDHC, SDXC) sets how large the card can be and which addressing the host must support. The *speed class* (Class 10, UHS Speed Class, and the video V-classes) rates the minimum sustained write performance — the guaranteed floor under continuous recording, which matters far more than the flashy peak read numbers on the label.

## Speed classes

The class marking states the slowest write the card is allowed to sustain, so a recorder never outruns it:

| Class mark | Min sustained write | Typical use |
|------------|---------------------|-------------|
| Class 10 / U1 | 10 MB/s | HD video, general |
| U3 / V30 | 30 MB/s | 4K video capture |
| V60 | 60 MB/s | High-bitrate 4K/8K |
| V90 | 90 MB/s | Professional 8K |

## Where it fits

The SD card is the default boot and storage medium for many [single-board computers](/reference/single-board-computer/), including the [Raspberry Pi](/reference/raspberry-pi/), which boots from a microSD card by default. It is cheap and removable, but cards vary widely in quality and endurance, and a low-end card can wear out under constant logging. A GopherTrunk capture node can run from an SD card — a well-rated card easily keeps up with decoded-call logs — though for heavy continuous IQ writes an [eMMC](/reference/emmc/) module or an [SSD](/reference/solid-state-drive/) lasts longer.

## Sources

[^wiki]: [SD card](https://en.wikipedia.org/wiki/SD_card) — Wikipedia, on Secure Digital cards, form factors, and capacity tiers.
