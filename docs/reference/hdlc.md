---
slug: hdlc
title: High-level Data Link Control (HDLC)
entry_type: technology
category: sdr-app-building
description: HDLC is a bit-oriented data-link framing protocol using a 0x7E flag byte, bit-stuffing, and a frame check sequence; it is the framing layer beneath AX.25 and APRS.
keywords: HDLC, high-level data link control, bit stuffing, flag byte 0x7E, frame check sequence, FCS, AX.25 framing, APRS framing, bit-oriented protocol, ISO 13239
aka: [HDLC, High-level Data Link Control]
autolink: true
infobox:
  - { label: Type, value: Data-link framing }
  - { label: Delimiter, value: "0x7E flag (01111110)" }
  - { label: Used by, value: "AX.25, APRS, PPP, X.25 LAPB" }
see_also: [ax25, packet-framing, cyclic-redundancy-check, deframing, aprs, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/High-Level_Data_Link_Control
  - https://en.wikipedia.org/wiki/AX.25
---

**HDLC** (High-level Data Link Control) is a bit-oriented data-link framing protocol that
delimits frames with a reserved flag byte, guarantees that flag can never appear inside the
data by *bit-stuffing*, and protects each frame with a frame check sequence.[^wiki] It is the
framing layer that sits beneath [AX.25](/reference/ax25/) — and therefore
[APRS](/reference/aprs/) and packet radio — turning a raw stream of demodulated bits into
cleanly delimited, integrity-checked frames.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="An HDLC frame drawn as a flag byte, the address/control/information payload, a frame check sequence, and a closing flag; a callout shows a zero bit stuffed after five consecutive one bits." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="14" y="26" width="56" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="42" y="45">flag 7E</text>
    <rect x="74" y="26" width="200" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="174" y="45">addr · control · info</text>
    <rect x="278" y="26" width="64" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="310" y="45">FCS</text>
    <rect x="346" y="26" width="56" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="374" y="45">flag 7E</text>
  </g>
  <g font-family="monospace" font-size="8" fill="currentColor"><text x="74" y="82">…11111 [0] …  stuffed 0 after five 1s</text></g>
  <line x1="174" y1="70" x2="174" y2="57" stroke="currentColor" stroke-width="0.7"/>
</svg>
<figcaption>An HDLC frame: flag-delimited, bit-stuffed so 0x7E never occurs in data, and checked by a trailing FCS.</figcaption>
</figure>

## How it works

HDLC solves the framing problem — where does a frame start and end? — with three
interlocking mechanisms:

- **Flag delimiter.** The byte `0x7E` (binary `01111110`) marks the start and end of every
  frame. Back-to-back frames can share one flag as both closing and opening.
- **Bit-stuffing.** So the flag's six-consecutive-1s pattern can never occur inside the
  payload, the transmitter inserts a `0` after any run of five `1` bits in the data; the
  receiver removes it. This makes the flag genuinely unique and keeps the frame
  *transparent* — any byte value can be carried.
- **Frame check sequence.** A trailing 16-bit [CRC](/reference/cyclic-redundancy-check/)
  (CRC-CCITT) computed over the frame lets the receiver discard corrupted frames rather than
  pass bad data upward.

Bit ordering matters: AX.25 sends each byte least-significant-bit first on the wire, so an HDLC
framer for it packs and unpacks bits LSB-first. HDLC is *bit-oriented* — it operates on the bit
stream itself, not on byte boundaries — which is why it can bit-stuff and why frames need not be
a whole number of bytes until the FCS is validated.

## In practice

On the receive side, an HDLC deframer slides a shift register through the incoming bits looking
for the flag pattern, then collects bits between two flags, removes stuffed zeros, and checks
that the result is byte-aligned and long enough to be a real frame before validating the FCS.
Note what HDLC does *not* do: it does not demodulate, and it does not undo line coding such as
[NRZI](/reference/nrzi/) — an AFSK modem handles NRZI in the demod-to-bits step, so by the time
bits reach the HDLC layer they are plain `{0,1}`.

## Relevance to SDR

HDLC is the framing an SDR decoder must reproduce to recover packet radio. Decoding an APRS
transmission means demodulating the 1200-baud [AFSK](/reference/afsk/), recovering bits,
undoing NRZI, and then running HDLC deframing to extract the [AX.25](/reference/ax25/) frame
whose FCS you verify before parsing the position report. The same framing underlies KISS TNCs
and classic packet BBS traffic. Beyond amateur radio, HDLC's flag-and-bit-stuffing idea reappears
in PPP and X.25 LAPB.

**GopherTrunk** implements HDLC directly in its AX.25/APRS path: the `internal/radio/aprs/hdlc`
package is a sync-aware framer that finds `0x7E` flags, removes bit-stuffing, enforces byte
alignment and length limits, and emits one frame body per flag-delimited sequence, leaving FCS
validation to the AX.25 parser. It is a clean, real instance of the flag/bit-stuffing/FCS
[packet framing](/reference/packet-framing/) model in a working pure-Go receiver.

## Sources

[^wiki]: [High-Level Data Link Control](https://en.wikipedia.org/wiki/High-Level_Data_Link_Control) — Wikipedia, on flag delimiting, bit-stuffing, and the frame check sequence.
[^ax25]: [AX.25](https://en.wikipedia.org/wiki/AX.25) — Wikipedia, on the amateur-radio data-link layer that uses HDLC framing with a 16-bit FCS.
