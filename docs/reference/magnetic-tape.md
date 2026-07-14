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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 192" role="img" aria-label="A tape drive threads a ribbon from a supply reel across guide rollers and over a read/write head to a take-up reel. Because the data is laid end to end along the ribbon, reaching a given block means spooling through every block before it — sequential access, unlike a disk that seeks directly." xmlns="http://www.w3.org/2000/svg">
  <text x="220" y="18" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Tape runs reel-to-reel past one head</text>
  <g fill="none" stroke="currentColor" stroke-width="1.1">
    <circle cx="88" cy="92" r="40" stroke-opacity="0.9"/><circle cx="88" cy="92" r="30" stroke-opacity="0.4"/><circle cx="88" cy="92" r="20" stroke-opacity="0.3"/>
    <circle cx="352" cy="92" r="40" stroke-opacity="0.9"/><circle cx="352" cy="92" r="30" stroke-opacity="0.4"/><circle cx="352" cy="92" r="20" stroke-opacity="0.3"/>
  </g>
  <circle cx="88" cy="92" r="9" fill="currentColor"/><circle cx="352" cy="92" r="9" fill="currentColor"/>
  <line x1="88" y1="52" x2="352" y2="52" stroke="currentColor" stroke-width="1.4"/>
  <circle cx="150" cy="52" r="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/>
  <circle cx="290" cy="52" r="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/>
  <rect x="198" y="52" width="44" height="22" rx="2" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="220" y="46" text-anchor="middle" font-size="8" fill="currentColor">read/write head</text>
  <line x1="120" y1="52" x2="138" y2="52" stroke="currentColor" stroke-width="1.1" marker-end="url(#mt_ar)"/>
  <line x1="304" y1="52" x2="322" y2="52" stroke="currentColor" stroke-width="1.1" marker-end="url(#mt_ar)"/>
  <text x="88" y="150" text-anchor="middle" font-size="8.5" fill="currentColor">supply reel</text>
  <text x="352" y="150" text-anchor="middle" font-size="8.5" fill="currentColor">take-up reel</text>
  <line x1="50" y1="168" x2="386" y2="168" stroke="currentColor" stroke-width="1.1" stroke-dasharray="5 3" stroke-opacity="0.8" marker-end="url(#mt_ar)"/>
  <text x="218" y="184" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">sequential: to reach a block you must spool through every block before it</text>
  <defs><marker id="mt_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A drive pulls the ribbon from the supply reel, across guide rollers and over a single read/write head, onto the take-up reel. The data is written end to end, so unlike a disk that seeks straight to a spot, tape must be spooled through everything ahead of the block you want — sequential access, which is why it is slow but very cheap per terabyte.</figcaption>
</figure>

## Overview

A drive pulls the tape past a head that magnetises regions to record bits, much like a [hard disk drive](/reference/hard-disk-drive/) but on flexible media that must be streamed end to end rather than seeked. The dominant modern format is LTO (Linear Tape-Open), whose cartridges now hold many terabytes each and improve with every generation. Because there are no fast random-access mechanics, the medium itself is extremely cheap, and a tape sitting on a shelf consumes no power and keeps data for decades.

## Where it fits

Tape lives at the coldest, deepest end of the [memory hierarchy](/reference/memory-hierarchy/): the slowest access but the lowest cost per terabyte and the longest shelf life, which is exactly what large-scale backup and archival want. Data centres still move petabytes onto LTO for "cold storage." Its sequential nature suits write-once-read-rarely archives — a fitting place to retire years of GopherTrunk IQ captures that you want to keep but rarely touch, while live decoding stays on [SSD](/reference/solid-state-drive/) or disk.

## Sources

[^wiki]: [Magnetic tape data storage](https://en.wikipedia.org/wiki/Magnetic_tape_data_storage) — Wikipedia, on tape storage, LTO, and its role in archival.
