---
slug: mdc1200-frame
title: MDC1200 frame
entry_type: term
category: paging-data
description: The MDC1200 frame is Motorola's in-band data burst — a 40-bit sync word (0x07092A446F) followed by 112 interleaved payload bits carrying an opcode, argument, unit ID and a reflected CRC-16/CCITT, keyed at the head or tail of an analog FM transmission.
keywords: MDC1200, MDC-1200, 0x07092A446F, PTT ID, ANI, FFSK 1200 baud, Motorola signaling, 16x7 interleave, radio inhibit stun, MDC opcode
aka: [MDC1200, MDC-1200, "Motorola Data Communications"]
autolink: true
infobox:
  - { label: Modulation, value: "1200-baud FFSK (NRZ)" }
  - { label: Sync word, value: "0x07092A446F (40 bits)" }
  - { label: Payload, value: "112 bits, 16×7 interleave" }
  - { label: CRC, value: "CRC-16/CCITT, reflected" }
see_also: [mdc1200, ffsk, crc-16-ccitt, cyclic-redundancy-check, interleaving, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/MDC-1200
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

The **MDC1200 frame** is the on-air unit of Motorola Data Communications — the analog in-band
data burst two-way radios key at the start or end of a transmission to signal the radio's unit
ID (ANI), emergency, status, call-alert and radio-check events on an otherwise-plain FM voice
channel.[^mdc] It is a 40-bit synchronisation word followed by 112 interleaved payload bits,
carried as a [1200-baud FFSK](/reference/ffsk/) burst (mark 1200 Hz, space 1800 Hz) inside the
narrowband-FM voice channel.[^fsync]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="An MDC1200 burst shown as a 40-bit sync word followed by a 112-bit payload, with the de-interleaved payload broken into an opcode byte, an argument byte, a two-byte unit ID, a two-byte CRC and eight bytes of redundancy." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="22" width="110" height="24" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="75" y="38" text-anchor="middle" font-size="8" fill="currentColor">sync 07092A446F</text>
  <rect x="130" y="22" width="310" height="24" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="285" y="38" text-anchor="middle" font-size="8" fill="currentColor">112 payload bits (16×7 column-interleaved)</text>
  <text x="20" y="70" font-size="7.5" fill="currentColor">after de-interleave → 14 bytes:</text>
  <rect x="20" y="80" width="40" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
  <text x="40" y="97" text-anchor="middle" font-size="8" fill="currentColor">op</text>
  <rect x="60" y="80" width="40" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
  <text x="80" y="97" text-anchor="middle" font-size="8" fill="currentColor">arg</text>
  <rect x="100" y="80" width="80" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="140" y="97" text-anchor="middle" font-size="8" fill="currentColor">unit ID</text>
  <rect x="180" y="80" width="80" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="220" y="97" text-anchor="middle" font-size="8" fill="currentColor">CRC-16</text>
  <rect x="260" y="80" width="180" height="26" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="350" y="97" text-anchor="middle" font-size="8" fill="currentColor">8 bytes redundancy (FEC)</text>
  <text x="20" y="128" font-size="7.5" fill="currentColor">CRC covers op‖arg‖unit ID; unit ID big-endian, CRC little-endian on the wire</text>
</svg>
<figcaption>An MDC1200 burst is a fixed 40-bit sync word and a 112-bit column-interleaved payload; de-interleaved it yields an opcode, argument, unit ID, a two-byte CRC and eight redundancy bytes.</figcaption>
</figure>

## How it works

MDC1200 rides the same modulation class GopherTrunk already demodulates for MPT 1327 and
[APRS](/reference/aprs/), but unlike APRS the line code is plain **NRZ**, not
[NRZI](/reference/nrzi/). A receiver hunts the 40-bit sync word `0x07092A446F`
(most-significant bit first), then captures the 112 payload bits that follow. Those bits are
**column-interleaved** over a 16×7 grid: GopherTrunk's `deinterleave` reads them back in
`bits[j*16+i]` order for `i` in 0..15 and `j` in 0..6, then packs the result LSB-first into 14
bytes. The interleave spreads a fading burst across the codeword so the over-the-air redundancy
can recover it. Of the 14 recovered bytes, the header is `op, arg, unit-ID (big-endian), CRC
(little-endian on the wire)`, and the trailing eight bytes are the redundancy field the frame's
FEC uses.

## Opcode table

The `(op, arg)` pair names the event. GopherTrunk's `opLabel` resolves the widely-published
opcode conventions; the set is best-effort and intentionally non-exhaustive, since many
vendor-specific and extended opcodes exist:

| Op | Meaning |
| --- | --- |
| `0x01` | PTT ID / ANI (arg `0x00` = keyed at tail) |
| `0x00` | Emergency |
| `0x06` | Status request |
| `0x12` | Status *N* (arg is the status number) |
| `0x0A` | Call alert / page |
| `0x22` | Message |
| `0x2B` | Radio inhibit (stun) |
| `0x2C` | Radio enable (revive) |
| `0x35` | Voice selective call |
| `0x46` | Remote monitor |
| `0x63` | Radio check |

Opcodes `0x35` and `0x55` select an *extended* two-block message; GopherTrunk flags these as
double packets and preserves the second block's raw bytes for a follow-up parse.

## The CRC

MDC1200 protects its header with a **reflected CRC-16/CCITT** — polynomial `0x1021` used in its
reflected form `0x8408`, initial value `0x0000`, and a final XOR of `0xFFFF`. GopherTrunk
computes it over the four header bytes (`op`, `arg`, and the two unit-ID bytes) and compares
against the received value. A CRC failure does not discard the burst outright: `DecodeFrame`
still returns the best-effort decode with `CRCOK = false`, so an operator can choose to surface
marginal bursts. The same `0x1021` polynomial appears across GopherTrunk's other framers under
different init/reflect/xorout settings, catalogued under [CRC-16/CCITT](/reference/crc-16-ccitt/).
The whole MDC1200 package is a clean-room implementation from the public protocol description —
sync word, interleave geometry, CRC parameters and opcode semantics are protocol facts, with no
third-party decoder source incorporated.

## Sources

[^mdc]: [MDC-1200](https://en.wikipedia.org/wiki/MDC-1200) — Wikipedia, on Motorola's in-band signalling burst and its ANI / emergency / status functions.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on locating a burst by its fixed sync word.
