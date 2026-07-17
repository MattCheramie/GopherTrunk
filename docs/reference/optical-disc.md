---
slug: optical-disc
title: Optical disc
entry_type: hardware
category: hw-storage
description: An optical disc stores data as pits read by a laser, in formats such as CD, DVD, and Blu-ray, once dominant for software and media distribution and still useful for archival.
keywords: optical disc, CD, DVD, Blu-ray, laser, pits and lands, optical storage, archival, wavelength
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

An **optical disc** stores data as microscopic pits and flat lands on a reflective spiral track, read by a focused laser.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="How a laser reads an optical disc. A laser beam focuses up through the disc onto a spiral track of pits and flat lands; light reflecting from a land returns strongly while light from a pit is scattered and dimmed, and a photodetector turns that changing brightness into a bit stream." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="40" y1="52" x2="440" y2="52"/>
    <line x1="40" y1="66" x2="440" y2="66"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.22" stroke-width="1">
    <rect x="70" y="52" width="34" height="14"/>
    <rect x="150" y="52" width="20" height="14"/>
    <rect x="230" y="52" width="46" height="14"/>
    <rect x="322" y="52" width="24" height="14"/>
    <rect x="392" y="52" width="30" height="14"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="40" y="46" fill-opacity="0.9">reflective spiral track (unrolled): raised pits and flat lands</text>
    <text x="126" y="46" text-anchor="middle" font-size="7.5">land</text>
    <text x="253" y="79" text-anchor="middle" font-size="7.5">pit</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <path d="M180 128 L200 66"/>
    <path d="M220 128 L200 66"/>
    <path d="M200 66 L200 120" stroke-dasharray="2 3" stroke-opacity="0.6"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="200" y="140" text-anchor="middle" font-weight="600">laser + lens</text>
    <text x="200" y="150" text-anchor="middle" font-size="7.5" fill-opacity="0.9">reads brightness &#8594; 1s and 0s</text>
  </g>
  <text x="345" y="140" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">shorter wavelength &#8594; smaller pits &#8594; more data</text>
</svg>
<figcaption>Data rides a single spiral of pits and lands; a focused laser reflects brightly off a land and weakly off a pit, and each generation shrank the laser's wavelength to pack smaller pits and hold more.</figcaption>
</figure>

## Overview

A laser shines on the spinning disc and a photodetector senses how the reflected light changes between pits and lands, recovering the bit stream from one long spiral track. Successive generations shortened the laser's wavelength to pack data more tightly: the infrared CD, the red-laser DVD, and the blue-violet Blu-ray, whose shorter beam focuses to a smaller spot and reads finer pits. Discs come as read-only pressed media, write-once recordable, and rewritable types, and data is laid out under standard file systems so any compatible drive can read it.

Optical storage is cheap to press in volume, physically robust, and needs no power to hold data, but it is slow to read and write compared with flash, and mechanical drives to spin it are increasingly rare on new machines.

## DVD vs Blu-ray

The jump from red to blue-violet lasers is what multiplied capacity across generations:

| Format | Laser | Wavelength | Capacity (single layer) |
|--------|-------|-----------|-------------------------|
| CD | Infrared | 780 nm | ~700 MB |
| DVD | Red | 650 nm | 4.7 GB |
| Blu-ray | Blue-violet | 405 nm | 25 GB |

Shorter wavelengths focus to a smaller spot, so pits shrink and the spiral packs more turns into the same disc.

## Where it fits

Optical discs were the main way software, music, and films were distributed before broadband and [flash](/reference/flash-memory/) storage took over, and they remain useful for cheap, long-lived archival because a written disc needs no power and resists data rot if stored well. In the [memory hierarchy](/reference/memory-hierarchy/) they sit alongside [magnetic tape](/reference/magnetic-tape/) at the slow, archival end — fine for keeping a frozen snapshot of GopherTrunk logs offline, but far too slow for live capture, where an [SSD](/reference/solid-state-drive/) or [HDD](/reference/hard-disk-drive/) is required.

## Sources

[^wiki]: [Optical disc](https://en.wikipedia.org/wiki/Optical_disc) — Wikipedia, on CD/DVD/Blu-ray optical storage and how lasers read pits and lands.
