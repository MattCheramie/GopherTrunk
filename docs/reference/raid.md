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

## Overview

RAID uses three basic techniques, mixed in different *levels*. *Striping* spreads data across drives for speed (RAID 0, no redundancy). *Mirroring* keeps identical copies so either drive can fail (RAID 1). *Parity* stores recovery information that lets the array rebuild a lost drive (RAID 5 tolerates one failure, RAID 6 tolerates two). Combined levels like RAID 10 mirror and stripe together. A key caveat: RAID protects against *drive* failure, not against deletion, corruption, or disaster — it is not a backup.

## Where it fits

RAID is the foundation of reliable [data storage](/reference/data-storage/) in [network-attached storage](/reference/network-attached-storage/), servers, and the [data center](/reference/data-center/), and it underpins [high availability](/reference/high-availability/) at the disk level. Built from ordinary [hard disk drives](/reference/hard-disk-drive/) or SSDs, a small mirror is a cheap way to keep a GopherTrunk archive of decoded calls alive through a single disk failure.

## Sources

[^wiki]: [RAID](https://en.wikipedia.org/wiki/RAID) — Wikipedia, on RAID levels and techniques.
