---
slug: nxdn-frame-structure
title: NXDN Frame Structure
entry_type: term
category: trunked-radio
description: "The NXDN frame is a 192-dibit (384-bit) unit lasting 80 ms — a Frame Sync Word, a LICH, a SACCH, and a 144-dibit information field carrying control, voice, or data — with the same layout at both NXDN channel rates."
keywords: NXDN frame structure, 192 dibits, NXDN superframe, LICH SACCH information field, NXDN 80 ms frame, 4800 9600 baud, NXDN layout
aka: ["NXDN frame", "NXDN frame layout", "NXDN superframe"]
autolink: true
infobox:
  - { label: Frame size, value: 192 dibits (384 bits) }
  - { label: Duration, value: 80 ms }
  - { label: Fields, value: "FSW · LICH · SACCH · info" }
  - { label: Spec, value: NXDN TS 1-A §4 }
see_also: [nxdn, nxdn-fsw, nxdn-lich, nxdn-sacch, nxdn-cac, nxdn-facch, four-fsk, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
  - https://en.wikipedia.org/wiki/Frame_(networking)
---

The **NXDN frame** is the fixed 192-dibit (384-bit) unit that every [NXDN](/reference/nxdn/)
transmission is built from, lasting **80 ms** and carrying four fields in a fixed order: the
[Frame Sync Word](/reference/nxdn-fsw/) (FSW), the [LICH](/reference/nxdn-lich/), the
[SACCH](/reference/nxdn-sacch/), and a 144-dibit **information field**.[^wiki] The structural
layout is identical whether the channel runs the 4800-baud (BFSK) or 9600-baud (4FSK) air
interface; the two rates differ only in how symbols map to bits, not in how the frame is
divided.[^frame]

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 150" role="img" aria-label="An NXDN frame laid out left to right as four fields: an 8-dibit Frame Sync Word, an 8-dibit LICH, a 32-dibit SACCH, and a 144-dibit information field, summing to 192 dibits over 80 milliseconds, with the information field able to carry CAC, voice, data, or FACCH." xmlns="http://www.w3.org/2000/svg">
  <rect x="14" y="34" width="40" height="30" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1.1"/>
  <text x="34" y="53" text-anchor="middle" font-size="8" fill="currentColor">FSW</text>
  <text x="34" y="78" text-anchor="middle" font-size="7" fill="currentColor">8</text>
  <rect x="54" y="34" width="40" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="74" y="53" text-anchor="middle" font-size="8" fill="currentColor">LICH</text>
  <text x="74" y="78" text-anchor="middle" font-size="7" fill="currentColor">8</text>
  <rect x="94" y="34" width="86" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="137" y="53" text-anchor="middle" font-size="8" fill="currentColor">SACCH</text>
  <text x="137" y="78" text-anchor="middle" font-size="7" fill="currentColor">32</text>
  <rect x="180" y="34" width="286" height="30" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="323" y="52" text-anchor="middle" font-size="8" fill="currentColor">information field — CAC / voice / data / FACCH</text>
  <text x="323" y="78" text-anchor="middle" font-size="7" fill="currentColor">144 dibits</text>
  <text x="14" y="104" font-size="8" fill="currentColor">192 dibits = 384 bits · 80 ms · identical layout at 4800 and 9600 baud</text>
  <path d="M14 118 L466 118" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 2"/>
  <text x="240" y="134" text-anchor="middle" font-size="7.5" fill="currentColor">SACCH fragments reassemble across a group of frames (superframe)</text>
</svg>
<figcaption>Each NXDN frame packs an 8-dibit sync word, an 8-dibit LICH, a 32-dibit SACCH, and a 144-dibit information field into 192 dibits over 80 ms; the SACCH's short per-frame fragments accumulate across a superframe to reassemble a full signalling message.</figcaption>
</figure>

## The four fields

The frame divides at fixed dibit offsets, so once the FSW correlator locks the grid every
downstream field lives at a known position:

| Field | Dibits | Bits | Offset | Role |
| --- | --- | --- | --- | --- |
| FSW | 8 | 16 | 0 | frame boundary marker (per-direction pattern) |
| LICH | 8 | 16 | 8 | 8 info bits, bit-doubled — RF channel type, direction |
| SACCH | 32 | 64 | 16 | slow signalling, one fragment per frame |
| Information | 144 | 288 | 48 | CAC (control), VCH/UDCH (voice/data), or FACCH |

The FSW is a fixed per-direction pattern; the LICH carries a heavily-protected 8-bit steering
field; the SACCH carries a short slice of a longer control message; and the information field
is the frame's payload, whose meaning is selected by context. On a control channel that field
carries the [CAC](/reference/nxdn-cac/) (the RCCH signalling that grants calls); on a traffic
channel it carries voice (VCH) or user data (UDCH), and it can be stolen frame-by-frame for
in-band [FACCH](/reference/nxdn-facch/) signalling.

## Superframes and the SACCH

A single frame is only 80 ms, too short to carry a full control message inside its 64-bit
SACCH, so NXDN groups frames into a **superframe** and spreads one SACCH message across the
group. Each frame contributes one SACCH fragment; the receiver accumulates fragments until it
holds the whole message, then decodes and CRC-checks it. This is the same
"short-field-repeated-across-frames" strategy other digital land-mobile standards use to fit
control signalling alongside voice without stealing payload from every frame — it trades
latency (you wait a superframe for a complete message) for a steady low-rate signalling
channel that runs underneath live traffic.

## Relevance to SDR

`internal/radio/nxdn/frame.go` defines the frame geometry GopherTrunk decodes against: the
`FrameDibits` (192), the per-field dibit counts, and the `OffsetFSW`/`OffsetLICH`/
`OffsetSACCH`/`OffsetInfo` constants that slice a captured frame into its fields. The `Frame`
type exposes `FSW()`, `LICH()`, `SACCH()`, and `Info()` accessors that return exactly those
sub-slices. Because the layout is rate-invariant, the same deframing code serves both NXDN
channel rates — the demodulator absorbs the BFSK-versus-4FSK difference and hands the deframer
a uniform dibit stream. The source flags that the exact dibit lengths follow the most-cited
interpretation of the specification and should be confirmed against the published technical
document before unusual captures are trusted; the field *ordering* and the 80 ms frame period,
however, are stable across every NXDN reference.

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, on the NXDN standard and its frame organisation.
[^frame]: [Frame (networking)](https://en.wikipedia.org/wiki/Frame_(networking)) — Wikipedia, on the general notion of a fixed-layout transmission frame.
