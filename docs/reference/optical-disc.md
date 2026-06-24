---
slug: optical-disc
title: Optical disc
entry_type: hardware
category: hw-storage
description: An optical disc stores data as pits read by a laser, in formats such as CD, DVD, and Blu-ray, once dominant for software and media distribution and still useful for archival.
keywords: optical disc, CD, DVD, Blu-ray, laser, pits and lands, optical storage, archival
aka: [CD, DVD, Blu-ray]
infobox:
  - { label: Type, value: Optical non-volatile storage }
  - { label: Read by, value: Focused laser }
  - { label: Formats, value: CD, DVD, Blu-ray }
  - { label: Capacity, value: ~700 MB – 100+ GB }
  - { label: Status, value: Niche / archival }
see_also: [magnetic-tape, hard-disk-drive, data-storage, solid-state-drive, memory-hierarchy, file-system]
cite_urls:
  - https://en.wikipedia.org/wiki/Optical_disc
---

An **optical disc** stores data as microscopic pits and flat lands on a reflective surface, read by a focused laser.[^wiki]

## Overview

A laser shines on the spinning disc and a photodetector senses how the reflected light changes between pits and lands, recovering the bit stream. Successive generations shortened the laser's wavelength to pack data more tightly: the infrared CD (~700 MB), the red-laser DVD (4.7 GB and up), and the blue-violet Blu-ray (25 GB per layer and beyond). Discs come as read-only pressed media, write-once recordable, and rewritable types. Data is laid out under standard file systems so any drive can read it.

## Where it fits

Optical discs were the main way software, music, and films were distributed before broadband and [flash](/reference/flash-memory/) storage took over, and they remain useful for cheap, long-lived archival because a written disc needs no power and resists data rot if stored well. In the [memory hierarchy](/reference/memory-hierarchy/) they sit alongside [magnetic tape](/reference/magnetic-tape/) at the slow, archival end — fine for keeping a frozen snapshot of GopherTrunk logs offline, but far too slow for live capture, where an [SSD](/reference/solid-state-drive/) or [HDD](/reference/hard-disk-drive/) is required.

## Sources

[^wiki]: [Optical disc](https://en.wikipedia.org/wiki/Optical_disc) — Wikipedia, on CD/DVD/Blu-ray optical storage and how lasers read pits and lands.
