---
slug: m17-link-information
title: M17 Link Information
entry_type: term
category: amateur-digital
description: The M17 Link Setup Frame (LSF) carries the metadata of an M17 transmission — source and destination addresses, mode type, and a META block — while the LICH mechanism spreads that LSF across six stream frames as Golay(24,12) codewords so a late receiver still recovers it.
keywords: M17 link setup frame, LSF, LICH, link information channel, base-40 callsign, Golay 24 12, M17 CRC 0x5935, M17 sync words, 4FSK amateur
aka: [LSF, LICH, "link setup frame", "link information channel"]
autolink: true
infobox:
  - { label: LSF size, value: 240 bits (30 bytes) }
  - { label: Fields, value: "DST 48 · SRC 48 · TYPE 16 · META 112 · CRC 16" }
  - { label: LICH, value: "6 chunks × Golay(24,12) codewords" }
  - { label: CRC, value: "CRC-16, poly 0x5935" }
see_also: [m17, m17-project, golay-code, cyclic-redundancy-check, forward-error-correction, radio-id, four-fsk, frame-synchronization]
cite_urls:
  - https://spec.m17project.org/
  - https://en.wikipedia.org/wiki/Binary_Golay_code
---

**M17 Link Information** is the metadata layer of the [M17](/reference/m17/) digital
voice/data protocol — an open, Codec2-based amateur mode developed by the
[M17 Project](/reference/m17-project/).[^m17] It answers "who is transmitting to whom, in what
mode" through the **Link Setup Frame** (**LSF**): source and destination addresses, a TYPE
field, and a META block. The LSF travels two ways — as a dedicated frame at the head of a
transmission, and, crucially, as the **LICH** (Link Information CHannel) embedded in every
stream frame — so a receiver that tunes in mid-transmission still recovers the link metadata
within about a quarter-second.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="The M17 Link Setup Frame: a 48-bit destination address, 48-bit source address, 16-bit type field, 112-bit META block, and 16-bit CRC, totalling 240 bits; below it, the LICH mechanism splits the LSF into six 40-bit chunks carried one per stream frame as Golay codewords." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
  <rect x="14" y="20" width="70" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="49" y="37">DST 48</text>
  <rect x="84" y="20" width="70" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="119" y="37">SRC 48</text>
  <rect x="154" y="20" width="44" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="176" y="37">TYPE 16</text>
  <rect x="198" y="20" width="180" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/><text x="288" y="37">META 112</text>
  <rect x="378" y="20" width="44" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="400" y="37">CRC 16</text>
  <text x="218" y="64" font-size="7.5">240-bit LSF</text>
  </g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
  <text x="60" y="92">LICH: 6 frames × 40-bit LSF chunk</text>
  <rect x="150" y="82" width="46" height="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="173" y="94">chunk 0</text>
  <rect x="200" y="82" width="46" height="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="223" y="94">chunk 1</text>
  <rect x="250" y="82" width="46" height="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="273" y="94">…</text>
  <rect x="300" y="82" width="46" height="18" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/><text x="323" y="94">chunk 5</text>
  <text x="240" y="120" font-size="7.5">each chunk = four Golay(24,12) codewords in one stream frame</text>
  </g>
</svg>
<figcaption>The 240-bit LSF holds the addresses, TYPE, META, and CRC; the LICH mechanism splits it into six 40-bit chunks, one carried in each stream frame's Golay-coded Link Information CHannel.</figcaption>
</figure>

## Link Setup Frame

The LSF is 240 bits (30 bytes) packed MSB-first:

| Field | Bits | Meaning |
|---|---|---|
| DST | 48 | Destination address (base-40 callsign, BROADCAST, or reserved) |
| SRC | 48 | Source address |
| TYPE | 16 | Mode/type flags — stream vs packet, data mode, CAN |
| META | 112 | 14-byte META block (nonce / GNSS / extended callsign) |
| CRC | 16 | CRC-16 over DST…META |

The **TYPE** field is bit-packed: bit 0 selects stream vs packet mode; bits 1–2 give the data
mode (1 = data, 2 = voice, 3 = voice + data); and bits 7–10 hold the CAN (Channel Access
Number). The trailing **CRC-16** uses polynomial **0x5935**, initial value 0xFFFF, MSB-first,
no final XOR — a bad CRC leaves the fields populated for display but flags the decode as
unverified.

## Base-40 addresses

M17 encodes a callsign into a 48-bit integer using a **base-40** codec (there is no separate
alphabet page — the whole scheme is this: an index-0..39 alphabet
`" ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/."`, where index 0 is the blank/terminator and the
rest are the callsign characters). Decoding repeatedly takes `addr % 40` as the next character
and divides by 40, then reverses the result. Three address values are special: all-ones
(0xFFFFFFFFFFFF) decodes to `BROADCAST`; zero is the reserved empty address; and any value
above 40⁹ (0xEE6B28000000) is not a base-40 callsign and reads as `#RESERVED`. Because the
codec is reversible and compact, a 9-character callsign fits in the 48-bit address field with
room to spare.

## LICH mechanism

A receiver joining mid-transmission cannot wait for the next LSF frame, so M17 also carries
the LSF inside every stream frame's LICH. Each stream frame's 96-bit LICH is **four
[Golay(24,12)](/reference/golay-code/) codewords** — 96 coded bits protecting a 48-bit chunk.
That chunk's top 40 bits are one-sixth of the LSF, and a 3-bit counter names its position. Six
consecutive chunks (one per stream frame, spanning six frames) reassemble the full 240-bit
LSF, at which point its CRC is checked. Any LICH with an uncorrectable Golay codeword is
dropped, and the assembler simply waits for that chunk to repeat on the next pass — a form of
[forward error correction](/reference/forward-error-correction/) plus temporal redundancy that
lets link metadata survive a marginal 4-FSK channel.

## In practice

M17 frames open with a 16-bit sync word that also names the frame type: **0x55F7** for the
dedicated LSF frame, **0xFF5D** for a stream (voice/data) frame, and **0x75FF** for a packet
frame. GopherTrunk implements the LICH route (Golay only, no convolutional machinery), so it
recovers link metadata without decoding the Codec2 audio. The Golay generator and field parse
are validated end-to-end against a synthetic encoder; the exact Golay matrix of real M17
traffic is flagged as best-effort pending confirmation against a live capture.

## Relevance to SDR

`internal/radio/m17/m17.go` holds the LSF parse, the base-40 address codec, and the CRC-16
(`ParseLSF`, `DecodeAddress`, `CRC16`); `lich.go` holds the LICH reassembler that Golay-decodes
the four codewords per frame and stitches the six chunks into an LSF. Together they let
GopherTrunk surface an M17 transmission's source, destination, and mode on the bus as soon as
six stream frames have passed.

## Sources

[^m17]: [M17 Protocol Specification](https://spec.m17project.org/) — the M17 Project's data-link layer reference for the LSF, LICH, base-40 codec, and sync words.
[^golay]: [Binary Golay code](https://en.wikipedia.org/wiki/Binary_Golay_code) — Wikipedia, on the Golay(24,12) code that protects each LICH chunk.
