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

## Overview

Inside the sealed enclosure, one or more rigid platters spin on a spindle at a fixed rate — commonly 5400 or 7200 RPM. A read/write head floats nanometres above each surface, magnetising tiny regions to store bits. Because the head must physically seek to the right track and wait for the platter to rotate into position, access has mechanical latency that an [SSD](/reference/solid-state-drive/) avoids. HDDs are still the cheapest way to hold large amounts of data per terabyte, which keeps them in servers, archives, and bulk storage.

## Where it fits

HDDs sit near the slow, high-capacity end of the [memory hierarchy](/reference/memory-hierarchy/), below RAM and below flash-based storage. They are a common form of [data storage](/reference/data-storage/), often combined in a [RAID](/reference/raid/) array or a [network-attached storage](/reference/network-attached-storage/) box for capacity and redundancy. A GopherTrunk logging server can keep months of decoded calls and raw IQ captures cheaply on spinning disks, reserving faster [SSD](/reference/solid-state-drive/) storage for the active working set. The drive's contents are organised by a [file system](/reference/file-system/).

## Sources

[^wiki]: [Hard disk drive](https://en.wikipedia.org/wiki/Hard_disk_drive) — Wikipedia, on the construction and operation of magnetic hard drives.
