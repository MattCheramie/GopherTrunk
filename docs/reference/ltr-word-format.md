---
slug: ltr-word-format
title: LTR word format
entry_type: term
category: trunked-radio
description: The LTR (Logic Trunked Radio) word is the 41-bit status word an EF Johnson LTR repeater transmits continuously at 300 baud beneath the voice, carrying the area, home repeater, group ID, and free-channel hint behind a CRC frame check.
keywords: LTR word format, Logic Trunked Radio, LTR status word, EF Johnson LTR, sub-audible signalling, CRC-7 FCS, home repeater, free channel, LTR area code
aka: ["LTR status word", "LTR data word", "Logic Trunked Radio word"]
autolink: true
infobox:
  - { label: Length, value: 41 bits at 300 baud }
  - { label: Fields, value: "Sync · Area · Group · Chan · Home · Group ID · Free · FCS" }
  - { label: FCS, value: "CRC-7, poly 0xFD, init 0x00" }
  - { label: Layer, value: Sub-audible under NBFM voice }
see_also: [ltr, trunked-radio, control-channel, conventional-radio, cyclic-redundancy-check, talkgroup, edacs-control-channel-word, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/Logic_Trunked_Radio
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
---

The **LTR word** is the status word an [LTR](/reference/ltr/) (Logic Trunked Radio) repeater
transmits continuously at 300 baud, sub-audibly beneath the voice on every channel.[^ltr] LTR
— the EF Johnson trunking scheme — has no dedicated [control channel](/reference/control-channel/):
each repeater broadcasts its own status word inline, and a radio pieces the system together by
reading the words from whichever channels it can hear. The word names the area, the active
group's home repeater, the group ID, and a hint at which repeater is currently free, all
behind a CRC frame check so a receiver can reject noise that happens to look like a frame.[^crc]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="The 41-bit LTR status word laid out as fields: a 1-bit sync marker, 5-bit area, 1-bit group flag, 4-bit channel, 5-bit home repeater, 8-bit group ID, 5-bit free-channel hint, and a 12-bit frame check sequence." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7" fill="currentColor" text-anchor="middle">
  <rect x="14" y="34" width="20" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/><text x="24" y="50">S</text>
  <rect x="34" y="34" width="46" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="57" y="50">Area 5</text>
  <rect x="80" y="34" width="22" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/><text x="91" y="50">G</text>
  <rect x="102" y="34" width="40" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="122" y="50">Chan 4</text>
  <rect x="142" y="34" width="48" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="166" y="50">Home 5</text>
  <rect x="190" y="34" width="70" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/><text x="225" y="50">GroupID 8</text>
  <rect x="260" y="34" width="48" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="284" y="50">Free 5</text>
  <rect x="308" y="34" width="90" height="26" fill="none" stroke="currentColor" stroke-width="1"/><text x="353" y="50">FCS 12</text>
  </g>
  <text x="200" y="82" text-anchor="middle" font-size="8" fill="currentColor">41 bits, MSB-first · sync bit always 1 · transmitted at 300 baud</text>
</svg>
<figcaption>The LTR status word packs eight fields across 41 bits and rides continuously under the voice; a decoder reads the home-repeater and group ID from the words it hears to follow a call.</figcaption>
</figure>

## Field layout

GopherTrunk models the 41-bit status word MSB-first:

| Bits | Width | Field | Meaning |
|---|---|---|---|
| 40 | 1 | Sync | Frame-start marker (always 1) |
| 39..35 | 5 | Area | Area code (0..31); disambiguates co-channel LTR systems |
| 34 | 1 | Group | The "F-bit": 1 = call active on this repeater, 0 = idle |
| 33..30 | 4 | Channel | Physical channel number (1..20) this word references |
| 29..25 | 5 | Home | Home-repeater number (1..20) for the active group |
| 24..17 | 8 | Group ID | Group / [talkgroup](/reference/talkgroup/) ID (1..250) |
| 16..12 | 5 | Free | Free-repeater hint (which repeater is unallocated) |
| 11..0 | 12 | FCS | Frame check / parity |

A word is "active" when the group F-bit is set and the group ID is non-zero. Because LTR
channels are numbered 1..20 and home repeaters 1..20, a well-formed check rejects any word
whose channel or home field is zero or out of range — a strong sign the frame is bit-garbage
that slipped past the sync bit.

## Frame check

GopherTrunk's CRC primitive follows sdrtrunk's `CRCLTR`: a **CRC-7** computed over a 24-bit
message by XORing precomputed syndrome contributions from a 24-entry lookup table.

- Polynomial **0xFD** = x^7 + x^6 + x^5 + x^4 + x^3 + x^2 + 1 (with the explicit x^7 term)
- Initial fill **0x00**

The 24 message bits the CRC covers are the Area, Channel, Home, Group, and Free fields. A
direction gotcha matters: the **OSW** (outbound-from-base) variant requires `calculated ==
transmitted`, but the **ISW** (inbound-from-subscriber) variant requires
`(calculated ^ 0x7F) == transmitted` — the inbound check inverts the low 7 bits. Most decoders
consume only outbound frames, so the default path expects the direct match.

The 41-bit `Status` struct and the CRC-7 primitive model the word under *different*
field-width conventions: the struct carries a 12-bit FCS and reads Area as 5 bits / Channel as
4 bits, while sdrtrunk's CRC-7 message reads Area as 1 bit / Channel as 5 bits. Both
conventions are in circulation for LTR Standard; reconciling the two layouts before wiring the
CRC primitive into the live LTR adapter is a documented follow-up — a caveat worth respecting
before trusting a marginal decode against a fresh capture.

## In practice

LTR's signalling is a slow sub-audible layer rather than a fast digital control channel, so a
scanner following LTR watches the status words that surface on active channels and uses the
home / free fields to predict where the next transmission for a group will land. The field
positions follow the most-cited public reference; some LTR-Net variants pack the fields
slightly differently, so cross-check before trusting live captures.

## Relevance to SDR

`internal/radio/ltr/status.go` holds the 41-bit status-word packing (`ParseStatus`,
`StatusFromBits`, the `IsActive` / `IsWellFormed` sanity checks), and
`internal/radio/framing/crc_ltr.go` implements the CRC-7 frame check with its OSW/ISW
direction logic. Together they let GopherTrunk read LTR status words and follow group activity
across an EF Johnson LTR system without a separate control channel to lock onto.

## Sources

[^ltr]: [Logic Trunked Radio](https://en.wikipedia.org/wiki/Logic_Trunked_Radio) — Wikipedia, on the EF Johnson LTR trunking scheme and its sub-audible signalling.
[^crc]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, on the CRC frame-check family the LTR FCS belongs to.
