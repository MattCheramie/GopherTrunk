---
slug: magnetic-tape
title: Magnetic tape
entry_type: hardware
category: hw-storage
description: Magnetic tape stores data sequentially on a long ribbon of magnetic film, offering very low cost per terabyte and long shelf life, making it the workhorse of large-scale archival.
keywords: magnetic tape, LTO, tape drive, sequential storage, backup, archival, cold storage, data tape
aka: [tape, LTO]
infobox:
  - { label: Type, value: Magnetic sequential storage }
  - { label: Medium, value: Coated tape ribbon }
  - { label: Common format, value: LTO (Linear Tape-Open) }
  - { label: Access, value: Sequential (no random seek) }
  - { label: Strength, value: Cheapest per terabyte, durable }
see_also: [hard-disk-drive, optical-disc, data-storage, memory-hierarchy, solid-state-drive, file-system]
cite_urls:
  - https://en.wikipedia.org/wiki/Magnetic_tape_data_storage
---

**Magnetic tape** stores data on a long, thin ribbon of magnetic film wound on reels, written and read sequentially by a tape drive.[^wiki]

## Overview

A drive pulls the tape past a head that magnetises regions to record bits, much like a [hard disk drive](/reference/hard-disk-drive/) but on flexible media that must be streamed end to end rather than seeked. The dominant modern format is LTO (Linear Tape-Open), whose cartridges now hold many terabytes each and improve with every generation. Because there are no fast random-access mechanics, the medium itself is extremely cheap, and a tape sitting on a shelf consumes no power and keeps data for decades.

## Where it fits

Tape lives at the coldest, deepest end of the [memory hierarchy](/reference/memory-hierarchy/): the slowest access but the lowest cost per terabyte and the longest shelf life, which is exactly what large-scale backup and archival want. Data centres still move petabytes onto LTO for "cold storage." Its sequential nature suits write-once-read-rarely archives — a fitting place to retire years of GopherTrunk IQ captures that you want to keep but rarely touch, while live decoding stays on [SSD](/reference/solid-state-drive/) or disk.

## Sources

[^wiki]: [Magnetic tape data storage](https://en.wikipedia.org/wiki/Magnetic_tape_data_storage) — Wikipedia, on tape storage, LTO, and its role in archival.
