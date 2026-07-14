---
slug: raid
title: RAID
entry_type: concept
category: hw-servers
description: RAID combines multiple physical drives into one logical unit to gain redundancy, performance, or both, so data can survive disk failures or be read and written faster.
keywords: RAID, redundant array, mirroring, striping, parity, RAID 0, RAID 1, RAID 5, RAID 6, RAID 10
aka: [Redundant Array of Independent Disks]
infobox:
  - { label: Type, value: Storage redundancy scheme }
  - { label: Combines, value: Multiple drives }
  - { label: Techniques, value: Striping, mirroring, parity }
  - { label: Common levels, value: 0, 1, 5, 6, 10 }
see_also: [data-storage, network-attached-storage, hard-disk-drive, high-availability, server, data-center]
cite_urls:
  - https://en.wikipedia.org/wiki/RAID
---

**RAID** (Redundant Array of Independent Disks) combines multiple physical drives into one logical unit to gain redundancy, performance, or both — so data can survive a disk failure or be read and written faster.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 214" role="img" aria-label="Three RAID schemes side by side. RAID 0 stripes blocks A1 to A6 across two disks for speed with no redundancy. RAID 1 mirrors identical copies on two disks. RAID 5 spreads data blocks and rotating parity blocks across three disks so the array can rebuild any one failed disk." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <text x="86" y="20" font-size="9.5" font-weight="600">RAID 0 · stripe</text>
    <rect x="40" y="40" width="40" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="43" y="44" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="60" y="65">A1</text>
    <rect x="43" y="80" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="60" y="101">A3</text>
    <rect x="43" y="116" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="60" y="137">A5</text>
    <rect x="92" y="40" width="40" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="95" y="44" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="112" y="65">A2</text>
    <rect x="95" y="80" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="112" y="101">A4</text>
    <rect x="95" y="116" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="112" y="137">A6</text>
    <text x="86" y="176" font-size="8" fill-opacity="0.85">fast · no redundancy</text>

    <text x="236" y="20" font-size="9.5" font-weight="600">RAID 1 · mirror</text>
    <rect x="190" y="40" width="40" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="193" y="44" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="210" y="65">A1</text>
    <rect x="193" y="80" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="210" y="101">A2</text>
    <rect x="193" y="116" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="210" y="137">A3</text>
    <rect x="242" y="40" width="40" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="245" y="44" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="262" y="65">A1</text>
    <rect x="245" y="80" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="262" y="101">A2</text>
    <rect x="245" y="116" width="34" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="262" y="137">A3</text>
    <text x="236" y="176" font-size="8" fill-opacity="0.85">identical copy</text>

    <text x="380" y="20" font-size="9.5" font-weight="600">RAID 5 · parity</text>
    <rect x="312" y="40" width="38" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="315" y="44" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="331" y="65">A1</text>
    <rect x="315" y="80" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="331" y="101">B1</text>
    <rect x="315" y="116" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 2"/><text x="331" y="137">P</text>
    <rect x="356" y="40" width="38" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="359" y="44" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="375" y="65">A2</text>
    <rect x="359" y="80" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 2"/><text x="375" y="101">P</text>
    <rect x="359" y="116" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="375" y="137">C1</text>
    <rect x="400" y="40" width="38" height="120" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="403" y="44" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 2"/><text x="419" y="65">P</text>
    <rect x="403" y="80" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="419" y="101">B2</text>
    <rect x="403" y="116" width="32" height="34" rx="2" fill="currentColor" fill-opacity="0.15"/><text x="419" y="137">C2</text>
    <text x="375" y="176" font-size="8" fill-opacity="0.85">rebuild any one disk</text>
  </g>
  <text x="230" y="200" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">RAID guards against a <tspan font-style="italic">drive</tspan> failure — it is not a backup</text>
</svg>
<figcaption>Three building blocks, mixed into the numbered levels. RAID 0 stripes for speed but loses everything if a disk dies; RAID 1 keeps a full mirror; RAID 5 rotates parity so any single disk can be rebuilt. None of them replaces a backup — they don't protect against deletion, corruption, or disaster.</figcaption>
</figure>

## Overview

RAID uses three basic techniques, mixed in different *levels*. *Striping* spreads data across drives for speed (RAID 0, no redundancy). *Mirroring* keeps identical copies so either drive can fail (RAID 1). *Parity* stores recovery information that lets the array rebuild a lost drive (RAID 5 tolerates one failure, RAID 6 tolerates two). Combined levels like RAID 10 mirror and stripe together. A key caveat: RAID protects against *drive* failure, not against deletion, corruption, or disaster — it is not a backup.

## Where it fits

RAID is the foundation of reliable [data storage](/reference/data-storage/) in [network-attached storage](/reference/network-attached-storage/), servers, and the [data center](/reference/data-center/), and it underpins [high availability](/reference/high-availability/) at the disk level. Built from ordinary [hard disk drives](/reference/hard-disk-drive/) or SSDs, a small mirror is a cheap way to keep a GopherTrunk archive of decoded calls alive through a single disk failure.

## Sources

[^wiki]: [RAID](https://en.wikipedia.org/wiki/RAID) — Wikipedia, on RAID levels and techniques.
