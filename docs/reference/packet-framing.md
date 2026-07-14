---
slug: packet-framing
title: Packet framing
entry_type: concept
category: sdr-app-building
description: Packet framing is the layout that wraps payload data with a preamble, sync word, header, and CRC so a receiver can find, delimit, and validate each unit of transmitted data.
keywords: packet framing, frame format, preamble sync header payload crc, frame structure, packet structure, data link framing, delimiter, framing layout
aka: [frame format, frame structure, packet structure]
autolink: true
infobox:
  - { label: Type, value: Data-link layout }
  - { label: Fields, value: "Preamble · sync · header · payload · CRC" }
  - { label: Recovered by, value: Deframing }
see_also: [deframing, correlate-access-code, hdlc, cyclic-redundancy-check, frame-synchronization, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Frame_(networking)
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
---

**Packet framing** is the layout that wraps payload data with the fixed structural fields a
receiver needs to find and validate it — typically a **preamble, a sync word, a header, the
payload, and a CRC**.[^frame] Without this envelope a stream of bits is unrecoverable: the
receiver would have no way to know where a message starts, how long it is, or whether it
arrived intact. The frame format is a contract between transmitter and receiver, and
[deframing](/reference/deframing/) is the act of reading it back.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 90" role="img" aria-label="A frame drawn as five labelled boxes in a row: preamble, sync word, header, payload, and CRC, each field serving one purpose for the receiver." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="14" y="30" width="70" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="49" y="49">preamble</text>
    <rect x="88" y="30" width="60" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="118" y="49">sync</text>
    <rect x="152" y="30" width="66" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="185" y="49">header</text>
    <rect x="222" y="30" width="150" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="297" y="49">payload</text>
    <rect x="376" y="30" width="60" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="406" y="49">CRC</text>
  </g>
  <g font-size="6" fill="currentColor" text-anchor="middle">
    <text x="49" y="74">clock/AGC</text>
    <text x="118" y="74">find start</text>
    <text x="185" y="74">len/type</text>
    <text x="297" y="74">data</text>
    <text x="406" y="74">integrity</text>
  </g>
</svg>
<figcaption>A typical packet frame: each field does one job — settle the receiver, mark the start, describe, carry, and check.</figcaption>
</figure>

## How it works

Reading a frame left to right, each field earns its place:

- **Preamble.** A run of alternating symbols (…101010…) at the front gives the receiver's
  [AGC](/reference/automatic-gain-control/) and [clock recovery](/reference/clock-recovery/)
  time to settle and lock before any information-bearing bits arrive.
- **Sync word.** A fixed, distinctive pattern the receiver
  [correlates for](/reference/correlate-access-code/) to pin the exact frame start and resolve
  bit/byte phase. It is chosen for sharp autocorrelation so false alignments are unlikely.
- **Header.** Fields describing the frame — length, type, addressing, sequence number — so the
  receiver knows how to interpret and how far to read.
- **Payload.** The actual data, often itself protected by
  [forward-error-correction](/reference/forward-error-correction/) and interleaving.
- **CRC / FCS.** A [cyclic redundancy check](/reference/cyclic-redundancy-check/) computed over
  the frame lets the receiver reject a corrupted frame instead of acting on bad data.

## Variants

Framing styles differ in how they mark boundaries. **Length-prefixed** frames put a byte count
in the header and read exactly that many bytes. **Delimiter/flag-based** frames — like
[HDLC](/reference/hdlc/) — bracket the payload with a reserved flag byte and use bit-stuffing so
the flag can never appear inside the data. **Fixed-slot** frames (TDMA voice like P25 or DMR)
have a constant geometry, so the sync word alone locates every field. Many real systems combine
these: a correlated sync word plus a length or type header plus a trailing CRC.

## Relevance to SDR

Every framed radio protocol an SDR decodes defines a packet frame, and knowing that layout is
what lets a decoder be written at all: P25 wraps a Frame Sync and Network ID around its
information; DMR uses SYNC patterns and a defined burst structure; AX.25/APRS uses HDLC flags
and a 16-bit FCS; ADS-B uses a preamble and a fixed 112-bit message with a parity/CRC field.

**GopherTrunk** encodes each protocol's frame format directly in its per-radio packages under
`internal/radio`. Its AX.25 path, for example, includes an `hdlc` framer that finds the `0x7E`
flag bytes, removes bit-stuffing, and hands a delimited frame body to the parser, which then
checks the trailing FCS — a textbook delimiter-plus-CRC frame. The other radios encode their
own preamble/sync/header/payload/CRC geometries so the [deframer](/reference/deframing/) knows
exactly where each field begins.

## Sources

[^frame]: [Frame (networking)](https://en.wikipedia.org/wiki/Frame_(networking)) — Wikipedia, on the delimited data unit and its structural fields.
[^crc]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, on the trailing check field that validates a received frame.
