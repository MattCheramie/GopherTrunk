---
slug: dsc-message-format
title: DSC message format
entry_type: term
category: aviation-marine
description: The DSC message format is ITU-R M.493 Digital Selective Calling — a 1200-baud FFSK marine calling protocol whose sequences carry a format specifier, category, source and target MMSI and (for distress) a nature and position, each character protected by a detect-only BCH(10,7) code and sent twice for redundancy.
keywords: DSC, digital selective calling, ITU-R M.493, marine VHF channel 70, MMSI, distress alert, format specifier, nature of distress, BCH 10 7, DX RX redundancy
aka: [DSC, "digital selective calling", "ITU-R M.493"]
autolink: true
infobox:
  - { label: Modulation, value: "1200-baud FFSK (1300/2100 Hz)" }
  - { label: FEC, value: "BCH(10,7), d=2 (detect only)" }
  - { label: Address, value: 9-digit MMSI }
  - { label: Spec, value: "ITU-R M.493" }
see_also: [dsc, marine-vhf, cospas-sarsat, epirb-406, ffsk, cyclic-redundancy-check]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_selective_calling
  - https://en.wikipedia.org/wiki/Maritime_Mobile_Service_Identity
---

The **DSC message format** is [Digital Selective Calling](/reference/dsc/) as defined by
ITU-R M.493 — the SOLAS-mandated digital calling protocol on marine VHF channel 70
(156.525 MHz) and the MF/HF distress channels, used for distress, urgency and safety alerts,
ship-to-ship paging, and the routine call-up that precedes a voice working-frequency
hand-off.[^dsc] On the air it is a [1200-baud FFSK](/reference/ffsk/) stream (tones 1300 and
2100 Hz) of 7-bit data symbols, each wrapped in a short error-detecting code and sent twice for
redundancy.[^mmsi]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A DSC sequence shown as a phasing preamble, a format-specifier symbol, a five-symbol MMSI address, a category symbol, distress nature and position fields, and an end-of-sequence symbol, with a callout that each character is a 7-bit value plus three BCH check bits." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="24" width="60" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="48" y="40" text-anchor="middle" font-size="7.5" fill="currentColor">phasing</text>
  <rect x="78" y="24" width="52" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/>
  <text x="104" y="40" text-anchor="middle" font-size="7.5" fill="currentColor">format</text>
  <rect x="130" y="24" width="90" height="24" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
  <text x="175" y="40" text-anchor="middle" font-size="7.5" fill="currentColor">MMSI (5 sym)</text>
  <rect x="220" y="24" width="52" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="246" y="40" text-anchor="middle" font-size="7.5" fill="currentColor">category</text>
  <rect x="272" y="24" width="120" height="24" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="332" y="40" text-anchor="middle" font-size="7.5" fill="currentColor">nature · position · time</text>
  <rect x="392" y="24" width="60" height="24" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/>
  <text x="422" y="40" text-anchor="middle" font-size="7.5" fill="currentColor">EOS</text>
  <text x="18" y="80" font-size="8" fill="currentColor">each character = 10 bits:</text>
  <rect x="130" y="70" width="120" height="18" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/>
  <text x="190" y="83" text-anchor="middle" font-size="7.5" fill="currentColor">7 data bits</text>
  <rect x="250" y="70" width="70" height="18" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1"/>
  <text x="285" y="83" text-anchor="middle" font-size="7.5" fill="currentColor">3 check</text>
  <text x="18" y="112" font-size="7.5" fill="currentColor">BCH(10,7), g(x)=x³+x+1 (0x0B) — minimum distance 2: detects one error, cannot correct it</text>
  <text x="18" y="128" font-size="7.5" fill="currentColor">real correction comes from sending every character twice (DX / RX redundancy)</text>
</svg>
<figcaption>A DSC sequence is a chain of 7-bit symbols — format, MMSI address, category, and (for distress) nature/position/time — each carrying three BCH check bits and repeated for redundancy.</figcaption>
</figure>

## How it works

By the time symbols reach GopherTrunk's `dsc.Decode`, the bit-stream layer above has already
handled sync, the per-character BCH check, and the redundancy merge, so each symbol is one
clean 7-bit value 0..127. The decoder reads the first symbol as the **format specifier**, then
walks the fields that format prescribes. A distress alert carries the sender's own MMSI
immediately (no separate target), followed by a nature-of-distress symbol, a five-symbol
position, and an HH:MM time; a non-distress call carries a target MMSI, a category, then the
sender's MMSI. Malformed or short sequences never error — they return with `FormatUnknown`
and the raw symbols preserved, because a real receiver sees noisy half-frames constantly.

## Format, category and nature tables

The leading symbols classify the call. GopherTrunk encodes the ITU-R M.493 code points
directly:

| Format | Code | Category | Code | Nature (distress) | Code |
| --- | --- | --- | --- | --- | --- |
| Distress | 112 | Distress | 112 | Fire / explosion | 100 |
| All-ships | 116 | Urgency | 110 | Flooding | 101 |
| Group | 114 | Safety | 108 | Collision | 102 |
| Individual | 120 | Routine | 100 | Sinking | 105 |
| Geographic | 102 | | | Man overboard | 110 |
| Auto-individual | 123 | | | EPIRB emission | 112 |

The "all-ships" and "distress-relay" formats share the same 116 code point and are told apart
by the category. The nature table continues through grounding, listing, disabled-and-adrift,
abandoning ship and piracy.

## MMSI and position codec

Addresses and coordinates are packed as pairs of decimal digits per symbol. An MMSI is five
symbols: each of the first four contributes two of the nine digits, and the fifth symbol's high
digit is the ninth MMSI digit while its low digit is a format-extension byte. A position is
also five symbols — one quadrant digit (0 = NE, 1 = NW, 2 = SE, 3 = SW), four latitude digits
(DDMM) and five longitude digits (DDDMM); GopherTrunk collapses the spec's all-9s sentinel to
"position unknown" and applies the quadrant as sign (bit 0 = West, bit 1 = South).

## The detect-only BCH(10,7)

Despite the "BCH" label the spec uses, DSC's per-character code has a **minimum Hamming
distance of only 2**. Each 10-bit character is 7 data bits followed by 3 check bits computed as
a CRC-3 with generator g(x) = x³ + x + 1 (binary `0x0B`). Distance-2 means a single flipped bit
is reliably *detected* but cannot be reliably *corrected* — legal codewords sit only one bit
apart, so "correcting" a corrupted word could silently produce a different valid one.
GopherTrunk's `BCHCheck` therefore returns only a pass/fail syndrome; the real error correction
comes from **DX/RX redundancy**, the M.493 scheme of sending every character twice and merging
the two copies, which the bit-stream layer performs before symbols reach the decoder.

## Sources

[^dsc]: [Digital selective calling](https://en.wikipedia.org/wiki/Digital_selective_calling) — Wikipedia, on the ITU-R M.493 marine calling protocol and its distress functions.
[^mmsi]: [Maritime Mobile Service Identity](https://en.wikipedia.org/wiki/Maritime_Mobile_Service_Identity) — Wikipedia, on the 9-digit MMSI a DSC sequence addresses.
