---
slug: dmr-slot-type
title: DMR slot type
entry_type: term
category: trunked-radio
description: The DMR SLOT TYPE is the 20-bit field surrounding the sync of a data/control burst — a 4-bit Color Code plus a 4-bit Data Type protected by a (20,8,7) Hamming/Golay code — that tells the decoder which color code and which kind of block the burst carries.
keywords: DMR slot type, SLOT TYPE field, data type, color code, Hamming 20 8, Golay 20 8, CSBK MBC idle, ETSI TS 102 361-1 Table 9.6
aka: ["SLOT TYPE", "DMR slot type", "data type field"]
autolink: true
infobox:
  - { label: Length, value: 20 bits (10 + 10 around sync) }
  - { label: Carries, value: Color Code (4) + Data Type (4) }
  - { label: FEC, value: "(20,8,7) Hamming/Golay, t=3" }
  - { label: Spec, value: ETSI TS 102 361-1 §6.4.2 / Table 9.6 }
see_also: [dmr-burst, color-code, hamming-code, golay-code, csbk, dmr-mbc, dmr-full-link-control, dmr, dmr-sync-patterns]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Hamming_code
---

The **DMR slot type** (**SLOT TYPE**) is the 20-bit field that surrounds the
[sync word](/reference/dmr-sync-patterns/) of every data/control [burst](/reference/dmr-burst/):
10 bits immediately before the sync and 10 immediately after.[^wiki] It carries just two things a
decoder needs before it can interpret the payload — the 4-bit [Color Code](/reference/color-code/)
that separates co-channel systems, and the 4-bit **Data Type** that names what kind of block the
burst holds — but wraps them in a strong FEC so both survive a fading channel. Only 8 of the 20
bits are information; the other 12 are parity from a (20,8,7) [Hamming](/reference/hamming-code/) /
[Golay](/reference/golay-code/) code.[^ham]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The 20-bit DMR slot-type codeword: a 4-bit color code and 4-bit data type forming 8 information bits, followed by 12 parity bits of a distance-7 Hamming/Golay code that corrects up to three bit errors, split as ten bits before and ten bits after the sync." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1.1"/>
  <text x="55" y="52" text-anchor="middle" font-size="8" fill="currentColor">CC · 4</text>
  <rect x="90" y="34" width="80" height="28" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/>
  <text x="130" y="52" text-anchor="middle" font-size="8" fill="currentColor">Data Type · 4</text>
  <rect x="170" y="34" width="250" height="28" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="295" y="52" text-anchor="middle" font-size="8" fill="currentColor">(20,8,7) parity · 12 bits</text>
  <text x="20" y="86" font-size="8" fill="currentColor">8 info bits, 12 parity · d=7 → corrects 3 errors, detects 4</text>
  <text x="20" y="102" font-size="8" fill="currentColor">split 10 bits before sync + 10 bits after sync</text>
</svg>
<figcaption>Eight information bits — color code and data type — are protected by a distance-7 code and split into two 10-bit fields on either side of the sync, so a decoder recovers both even after up to three bit errors.</figcaption>
</figure>

## The Data Type enum

The Data Type names the block, per ETSI TS 102 361-1 §9.1.2 Table 9.6. GopherTrunk enumerates
the valid values in `internal/radio/dmr/slottype.go`:

| Value | Data Type | Meaning |
|---|---|---|
| `0x0` | PI Header | Privacy (encryption) header |
| `0x1` | Voice LC Header | [Full Link Control](/reference/dmr-full-link-control/) opening a voice call |
| `0x2` | Terminator with LC | Full LC closing a voice call |
| `0x3` | CSBK | [Control Signalling Block](/reference/csbk/) |
| `0x4` | MBC Header | [Multi-Block Control](/reference/dmr-mbc/) header |
| `0x5` | MBC Continuation | Multi-Block Control continuation |
| `0x6` | Data Header | Packet-data header |
| `0x7` | Rate ½ Data | Confirmed/unconfirmed rate-½ data block |
| `0x8` | Rate ¾ Data | Rate-¾ data block |
| `0x9` | Idle | Idle block |
| `0xA` | Rate 1 Data | Rate-1 data block |
| `0xB` | Reserved | (0xC–0xF also reserved) |

## The (20,8,7) code

The eight information bits are laid out as `[CC(4) | DataType(4)]` in the high byte, followed by
12 parity bits. The code has minimum distance 7, so it corrects up to 3 bit errors and detects 4.
GopherTrunk's `framing.HammingDecode20_8` (`internal/radio/framing/hamming20.go`) does a
minimum-distance search over all 256 valid codewords rather than a syndrome table — 256 entries
fit in cache and keep the decoder auditable against the spec's generator. `ParseSlotType` runs
that decode, returns the corrected bit count (0 on a clean codeword), and reports
`ErrSlotTypeUncorrectable` when the nearest codeword is more than 3 bits away. Some references
call this Hamming(20,8) and others Golay(20,8); the wire format is identical.

## Relevance to SDR

The slot type is the router of GopherTrunk's DMR control path. Because it decodes independently
of the 196-bit payload, a receiver can read the Color Code and Data Type even when the payload is
too corrupt to use, and the Data Type is what tells the deframer whether to hand the two payload
halves to the CSBK parser, the voice-LC / RS(12,9) path, the MBC assembler, or a data-block
decoder. It is also one of the arbiters that resolves the sync-word polarity ambiguity (issue
#264): a burst decoded at the wrong polarity yields an uncorrectable slot type and is dropped, so
getting the (20,8,7) code exactly right is part of what keeps a spectrum-inverted stream from
producing phantom blocks.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its burst framing.
[^ham]: [Hamming code](https://en.wikipedia.org/wiki/Hamming_code) — Wikipedia, on the linear block codes underlying the slot-type FEC.
