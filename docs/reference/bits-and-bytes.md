---
slug: bits-and-bytes
title: Bits & bytes
entry_type: concept
category: hw-foundations
description: A bit is the smallest unit of digital information, a single 0 or 1; a byte is a group of eight bits. Together they are the units in which computers store and move all data.
keywords: bit, byte, binary, kilobyte, megabyte, gigabyte, base-2, data units, octet
aka: [Bit, Byte]
infobox:
  - { label: Bit, value: One binary digit (0 or 1) }
  - { label: Byte, value: 8 bits }
  - { label: Byte range, value: 0–255 (256 values) }
  - { label: Larger units, value: "KB, MB, GB, TB" }
see_also: [logic-gate, random-access-memory, data-storage, bandwidth, central-processing-unit, transistor]
cite_urls:
  - https://en.wikipedia.org/wiki/Bit
  - https://en.wikipedia.org/wiki/Byte
---

A **bit** is the smallest unit of digital information — a single binary digit, either 0 or 1 — and a **byte** is a group of eight bits.[^bit][^byte]

## Overview

Everything a computer stores or moves is encoded in bits, physically realized as a [transistor](/reference/transistor/) switched on or off, a charge present or absent, a voltage high or low. Eight bits make a byte, which can represent 256 distinct values (0–255) — enough for one character of basic text. Larger quantities are counted in kilobytes, megabytes, gigabytes, and terabytes. Data *rates* are usually measured in **bits** per second (the unit of [bandwidth](/reference/bandwidth/)), while storage sizes are usually in **bytes** — a common source of confusion.

## Where it fits

Bits and bytes are the common currency of every other hardware idea: [memory](/reference/random-access-memory/) and [storage](/reference/data-storage/) are sized in bytes, [logic gates](/reference/logic-gate/) operate on bits, and a [CPU's](/reference/central-processing-unit/) word width is a bit count. In SDR terms, the IQ samples GopherTrunk ingests are streams of bytes, and a decoded voice frame is ultimately a precise pattern of bits pulled out of the noise.

## Sources

[^bit]: [Bit](https://en.wikipedia.org/wiki/Bit) — Wikipedia, on the bit as the basic unit of information.
[^byte]: [Byte](https://en.wikipedia.org/wiki/Byte) — Wikipedia, on the byte as a group of (usually eight) bits.
