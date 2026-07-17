---
slug: bits-and-bytes
title: Bits & bytes
entry_type: concept
category: hw-foundations
description: A bit is the smallest unit of digital information, a single 0 or 1; a byte is a group of eight bits. Together they are the units in which computers store, address, and move all data.
keywords: bit, byte, binary, kilobyte, megabyte, gigabyte, base-2, data units, octet, most significant bit, place value
aka: [Bit, Byte]
infobox:
  - { label: Bit, value: One binary digit (0 or 1) }
  - { label: Byte, value: 8 bits }
  - { label: Byte range, value: 0–255 (256 values) }
  - { label: Bit rate unit, value: bits per second }
  - { label: Larger units, value: "KB, MB, GB, TB" }
see_also: [logic-gate, random-access-memory, data-storage, bandwidth, central-processing-unit, transistor]
cite_urls:
  - https://en.wikipedia.org/wiki/Bit
  - https://en.wikipedia.org/wiki/Byte
---

A **bit** is the smallest unit of digital information — a single binary digit, either 0 or 1 — and a **byte** is a group of eight bits.[^bit][^byte]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A single byte shown as eight bit cells, from the most significant bit on the left with place value 128 down to the least significant bit on the right with place value 1. The example pattern 0100 0001 sums to 65, the ASCII code for the letter A." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="30" y="40" width="50" height="40"/>
    <rect x="80" y="40" width="50" height="40" fill="currentColor" fill-opacity="0.14"/>
    <rect x="130" y="40" width="50" height="40"/>
    <rect x="180" y="40" width="50" height="40"/>
    <rect x="230" y="40" width="50" height="40"/>
    <rect x="280" y="40" width="50" height="40"/>
    <rect x="330" y="40" width="50" height="40"/>
    <rect x="380" y="40" width="50" height="40" fill="currentColor" fill-opacity="0.14"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-family="ui-monospace, monospace">
    <text x="55" y="66" font-size="13">0</text>
    <text x="105" y="66" font-size="13">1</text>
    <text x="155" y="66" font-size="13">0</text>
    <text x="205" y="66" font-size="13">0</text>
    <text x="255" y="66" font-size="13">0</text>
    <text x="305" y="66" font-size="13">0</text>
    <text x="355" y="66" font-size="13">0</text>
    <text x="405" y="66" font-size="13">1</text>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8" fill-opacity="0.85">
    <text x="55" y="94">128</text>
    <text x="105" y="94">64</text>
    <text x="155" y="94">32</text>
    <text x="205" y="94">16</text>
    <text x="255" y="94">8</text>
    <text x="305" y="94">4</text>
    <text x="355" y="94">2</text>
    <text x="405" y="94">1</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="30" y="30">MSB</text>
    <text x="410" y="30">LSB</text>
  </g>
  <text x="230" y="120" text-anchor="middle" font-size="9" fill="currentColor">64 + 1 = 65 = ASCII 'A'</text>
  <text x="230" y="138" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">one byte holds 256 distinct values (0–255)</text>
</svg>
<figcaption>A byte is eight bits, each holding a place value that doubles from right to left; adding up the "on" bits gives a number from 0 to 255 — here 0100 0001 equals 65, the code for the character 'A'.</figcaption>
</figure>

## Overview

Everything a computer stores or moves is encoded in bits, physically realized as a [transistor](/reference/transistor/) switched on or off, a charge present or absent, a voltage high or low. Grouping eight bits into a byte gives a convenient chunk that can represent 256 distinct values (0–255) — enough for one character of basic text, one level of a colour channel, or one small integer.

Within a byte each bit carries a *place value* that doubles from the least significant bit (worth 1) on the right up to the most significant bit (worth 128) on the left, exactly like decimal place value but in base two. Larger quantities are counted in kilobytes, megabytes, gigabytes, and terabytes.

A persistent source of confusion is that data *rates* are usually quoted in **bits** per second (the unit of [bandwidth](/reference/bandwidth/)) while storage *sizes* are usually quoted in **bytes** — so a "100 Mbps" link moves only about 12.5 MB each second.

## Counting in bigger units

Because computers address memory in powers of two, the "kilo/mega/giga" prefixes are often used to mean powers of 1024 rather than 1000; standards bodies coined the binary prefixes (KiB, MiB) to make the distinction explicit.

| Unit | Bits | Bytes | Rough scale |
|------|------|-------|-------------|
| Bit | 1 | ⅛ | one 0 or 1 |
| Byte | 8 | 1 | one text character |
| Kilobyte (KB) | 8 × 10³ | 10³ | a short email |
| Megabyte (MB) | 8 × 10⁶ | 10⁶ | a minute of music |
| Gigabyte (GB) | 8 × 10⁹ | 10⁹ | a movie |
| Terabyte (TB) | 8 × 10¹² | 10¹² | a large drive |

## Where it fits

Bits and bytes are the common currency of every other hardware idea: [memory](/reference/random-access-memory/) and [storage](/reference/data-storage/) are sized in bytes, [logic gates](/reference/logic-gate/) operate on bits, and a [CPU's](/reference/central-processing-unit/) word width is a bit count. In SDR terms, the IQ samples GopherTrunk ingests are streams of bytes, and a decoded voice frame is ultimately a precise pattern of bits pulled out of the noise.

## Sources

[^bit]: [Bit](https://en.wikipedia.org/wiki/Bit) — Wikipedia, on the bit as the basic unit of information.
[^byte]: [Byte](https://en.wikipedia.org/wiki/Byte) — Wikipedia, on the byte as a group of (usually eight) bits and the binary multiples.
