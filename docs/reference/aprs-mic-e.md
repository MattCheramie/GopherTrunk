---
slug: aprs-mic-e
title: APRS Mic-E
entry_type: term
category: wireless-data-iot
description: APRS Mic-E is the compressed position/status encoding used by mobile trackers — it packs six latitude digits, three message bits and the hemisphere flags into the AX.25 destination address, and longitude, speed, course, symbol and altitude into the information field.
keywords: APRS Mic-E, Mic-Encoder, compressed APRS position, AX.25 destination address, latitude encoding, speed course interleave, APRS message code, DTI, APRS Protocol Reference
aka: [Mic-E, "Mic-Encoded", "APRS Mic-E"]
autolink: true
infobox:
  - { label: Type, value: Compressed APRS position }
  - { label: Latitude in, value: AX.25 destination address }
  - { label: Info field, value: "longitude, speed/course, symbol" }
  - { label: Spec, value: "APRS Protocol Reference 1.0.1 §10" }
see_also: [aprs, ax25, afsk, nrzi, compact-position-reporting]
cite_urls:
  - http://www.aprs.org/doc/APRS101.PDF
  - https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System
---

**APRS Mic-E** (Mic-Encoded) is the compressed position/status format that most mobile
[APRS](/reference/aprs/) trackers transmit — a third the size of an uncompressed beacon, which
is why almost every vehicle tracker and handheld uses it.[^aprs] Its defining trick is that
half the payload lives in the [AX.25](/reference/ax25/) frame's *destination address* field:
the six characters that would normally name a station instead encode latitude, three message
bits and the hemisphere flags, while the information field carries longitude, speed, course,
symbol and an optional altitude.[^wp]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A Mic-E packet split into two halves: the six-character AX.25 destination address carrying six latitude digits plus message bits and hemisphere flags, and the information field carrying three longitude bytes, three speed and course bytes, two symbol bytes and an optional altitude and comment." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="22" font-size="8" fill="currentColor">AX.25 destination address (6 chars):</text>
  <rect x="20" y="30" width="230" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="135" y="47" text-anchor="middle" font-size="8" fill="currentColor">lat DDMMhh + 3 msg bits + N/S · lon-offset · E/W</text>
  <text x="20" y="82" font-size="8" fill="currentColor">information field:</text>
  <rect x="20" y="90" width="90" height="26" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/>
  <text x="65" y="107" text-anchor="middle" font-size="8" fill="currentColor">longitude 3B</text>
  <rect x="110" y="90" width="110" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="165" y="107" text-anchor="middle" font-size="8" fill="currentColor">speed + course 3B</text>
  <rect x="220" y="90" width="80" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="260" y="107" text-anchor="middle" font-size="8" fill="currentColor">symbol 2B</text>
  <rect x="300" y="90" width="140" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="370" y="107" text-anchor="middle" font-size="8" fill="currentColor">altitude + comment</text>
</svg>
<figcaption>Mic-E is a two-half codec: the AX.25 destination address holds latitude and the message/hemisphere bits, and the information field holds longitude, speed, course, the symbol and an optional altitude trailer.</figcaption>
</figure>

## How it works

Because half the payload is in the AX.25 envelope, GopherTrunk's entry point is
`DecodeWithDst(info, dst)`: the six-character destination callsign must be supplied alongside
the information field. If it is missing the packet still surfaces as a Mic-E type with its raw
bytes preserved. The information field is identified by its first byte, the DTI (`0x1C`, `0x1D`,
`` ` `` or `'`), and the remaining bytes follow a fixed layout. Like the rest of the APRS
decoder, Mic-E never errors — unknown or malformed payloads pass through with the raw bytes
kept, because APRS traffic is messy.

## Destination-address encoding

Each of the six destination characters decodes to one latitude digit plus one indicator bit,
using the APRS 101 §10.5 table. The character *range* also matters: characters `P`–`Z` are the
"standard" message set, `A`–`K` the "custom" set (with `K`/`L` acting as space carriers), and
`0`–`9` are plain digits. The indicator bit means something different depending on position:

| Dest char | Latitude role | Indicator bit means |
| --- | --- | --- |
| 1–3 | DD, first M | one of the 3 message bits |
| 4 | second M | N/S hemisphere |
| 5–6 | hh (hundredths) | 5: longitude +100° offset; 6: W/E hemisphere |

The three message bits form a code that maps to a status label — `M0 Off Duty`, `M3 Returning`,
and so on, with all-zeros reserved for `Emergency`. A packet mixing standard and custom
characters in the message positions is technically malformed; GopherTrunk reports whichever
range the characters fall in and prefers the standard label set.

## Information field

The longitude is three bytes, each offset by 28 to keep it printable ASCII, with the APRS 101
wraparound corrections applied and the `+100°` offset (from destination char 5) folded in.
Speed and course are **interleaved** across three bytes so that

- `speed = (byte4−28)·10 + (byte5−28)/10`
- `course = ((byte5−28) mod 10)·100 + (byte6−28)`

with 800/400 wrap corrections undoing the printable-ASCII bias. Byte 7 is the symbol code and
byte 8 the symbol table (together selecting the map icon — car, ambulance, and so on). Anything
after that is an optional altitude marker (three base-91 characters followed by `}`, valued as
`base91 − 10000` metres) and a free-form comment. GopherTrunk lifts the recovered position into
the standard APRS `Position` field so existing callers and the web panel pick it up without
special-casing the Mic-E shape.

## Sources

[^aprs]: [APRS Protocol Reference 1.0.1](http://www.aprs.org/doc/APRS101.PDF) — Bob Bruninga et al., Chapter 10, the authoritative Mic-E field layout, destination-address table and speed/course encoding.
[^wp]: [Automatic Packet Reporting System](https://en.wikipedia.org/wiki/Automatic_Packet_Reporting_System) — Wikipedia, on APRS and its position-report formats.
