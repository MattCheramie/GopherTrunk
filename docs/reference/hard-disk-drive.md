---
slug: hard-disk-drive
title: Hard disk drive (HDD)
entry_type: hardware
category: hw-storage
description: A hard disk drive stores data on spinning magnetic platters read by moving heads, offering large capacity at low cost per byte but slower access than solid-state storage.
keywords: hard disk drive, HDD, magnetic storage, platter, spindle, read/write head, RPM, mechanical drive
aka: [HDD, hard drive, hard disk]
infobox:
  - { label: Type, value: Magnetic non-volatile storage }
  - { label: Medium, value: Spinning platters }
  - { label: Capacity, value: ~1 – 30 TB }
  - { label: Speed, value: 5400 – 7200 RPM (typical) }
  - { label: Strength, value: Low cost per terabyte }
see_also: [solid-state-drive, data-storage, magnetic-tape, optical-disc, memory-hierarchy, file-system]
cite_urls:
  - https://en.wikipedia.org/wiki/Hard_disk_drive
---

A **hard disk drive (HDD)** is a non-volatile storage device that records data on rapidly spinning magnetic platters, read and written by heads on a moving arm.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 232" role="img" aria-label="Top-down view of a hard disk: a platter of concentric tracks spins on a central spindle while an actuator arm pivots from the corner, swinging a read/write head in an arc to the target track. Reaching data therefore takes seek time for the arm to move plus rotational delay for the platter to turn the data under the head." xmlns="http://www.w3.org/2000/svg">
  <circle cx="140" cy="120" r="92" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.2"/>
  <g fill="none" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4">
    <circle cx="140" cy="120" r="26"/><circle cx="140" cy="120" r="46"/><circle cx="140" cy="120" r="66"/><circle cx="140" cy="120" r="86"/>
  </g>
  <circle cx="140" cy="120" r="6" fill="currentColor"/>
  <line x1="250" y1="206" x2="120" y2="52" stroke="currentColor" stroke-width="5" stroke-opacity="0.4" stroke-linecap="round"/>
  <circle cx="250" cy="206" r="6" fill="currentColor" fill-opacity="0.5" stroke="currentColor" stroke-width="1"/>
  <rect x="112" y="46" width="18" height="13" rx="2" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1"/>
  <path d="M140 96 A 24 24 0 0 1 163 121" fill="none" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.8" marker-end="url(#hdd_ar)"/>
  <path d="M232 200 A 26 26 0 0 1 256 181" fill="none" stroke="currentColor" stroke-width="1.1" stroke-opacity="0.8" marker-end="url(#hdd_ar)"/>
  <g font-size="8" fill="currentColor" stroke="none">
    <line x1="298" y1="52" x2="132" y2="52" stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5"/>
    <text x="300" y="55" text-anchor="start">read/write head</text>
    <line x1="298" y1="92" x2="178" y2="116" stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5"/>
    <text x="300" y="95" text-anchor="start">actuator arm</text>
    <line x1="298" y1="124" x2="232" y2="122" stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5"/>
    <text x="300" y="127" text-anchor="start">platter</text>
    <line x1="298" y1="160" x2="196" y2="150" stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5"/>
    <text x="300" y="163" text-anchor="start">track</text>
    <line x1="298" y1="196" x2="150" y2="124" stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5"/>
    <text x="300" y="199" text-anchor="start">spindle</text>
  </g>
  <text x="300" y="176" text-anchor="start" font-size="7" fill="currentColor" fill-opacity="0.8">(one of several</text>
  <text x="300" y="186" text-anchor="start" font-size="7" fill="currentColor" fill-opacity="0.8">stacked platters)</text>
  <text x="150" y="226" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">access = seek (arm swings) + rotational delay (platter turns)</text>
  <defs><marker id="hdd_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Bits sit on concentric tracks of a platter spinning on the spindle. To read one, the actuator arm swings its head to the right track (seek time) and then waits for the platter to rotate that spot under the head (rotational delay). That mechanical two-step is the latency an SSD, with no moving parts, avoids.</figcaption>
</figure>

## Overview

Inside the sealed enclosure, one or more rigid platters spin on a spindle at a fixed rate — commonly 5400 or 7200 RPM. A read/write head floats nanometres above each surface, magnetising tiny regions to store bits. Because the head must physically seek to the right track and wait for the platter to rotate into position, access has mechanical latency that an [SSD](/reference/solid-state-drive/) avoids. HDDs are still the cheapest way to hold large amounts of data per terabyte, which keeps them in servers, archives, and bulk storage.

## Where it fits

HDDs sit near the slow, high-capacity end of the [memory hierarchy](/reference/memory-hierarchy/), below RAM and below flash-based storage. They are a common form of [data storage](/reference/data-storage/), often combined in a [RAID](/reference/raid/) array or a [network-attached storage](/reference/network-attached-storage/) box for capacity and redundancy. A GopherTrunk logging server can keep months of decoded calls and raw IQ captures cheaply on spinning disks, reserving faster [SSD](/reference/solid-state-drive/) storage for the active working set. The drive's contents are organised by a [file system](/reference/file-system/).

## Sources

[^wiki]: [Hard disk drive](https://en.wikipedia.org/wiki/Hard_disk_drive) — Wikipedia, on the construction and operation of magnetic hard drives.
