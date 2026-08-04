---
slug: mpt-1327-codeword
title: MPT 1327 codeword
entry_type: term
category: trunked-radio
description: The MPT 1327 codeword is the 48-bit address or data signalling unit of the UK MPT 1327 trunking standard, carrying a type flag, area prefix, fleet identity, opcode, and function payload behind a BCH(64,48) check with a trailing parity bit.
keywords: MPT 1327 codeword, MPT1327 signalling, address codeword, ALH AHY GTC, MPT 1327 opcode, BCH 64 48, CRC-15 0x6815, FFSK 1200 baud, go-to-channel
aka: ["MPT1327 codeword", "MPT 1327 address codeword", "control codeword"]
autolink: true
infobox:
  - { label: Length, value: 64 bits on wire (48 info) }
  - { label: Fields, value: "Type 1 · Prefix 7 · Ident 13 · Op 10 · Function 17" }
  - { label: FEC, value: "BCH(64,48) CRC-15 0x6815 + parity" }
  - { label: Modulation, value: 1200-baud FFSK }
see_also: [mpt-1327, trunked-radio, control-channel, channel-grant, ffsk, cyclic-redundancy-check, edacs-control-channel-word, motorola-osw, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/MPT-1327
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
---

The **MPT 1327 codeword** is the signalling unit of [MPT 1327](/reference/mpt-1327/), the UK
trunking standard once widespread across European and Commonwealth private mobile radio.[^mpt]
It rides the [control channel](/reference/control-channel/) as a stream of 1200-baud
[FFSK](/reference/ffsk/) codewords, each carrying one piece of system business — a
[channel grant](/reference/channel-grant/), a page, a broadcast, an acknowledgement, or a
control-channel idle. Every codeword is 64 bits on the wire: a 48-bit information field
followed by a BCH(64,48) check and a single overall parity bit, so a receiver can reject or
repair blocks corrupted by noise.[^crc]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="The MPT 1327 codeword: a 48-bit information field split into a 1-bit type flag, 7-bit prefix, 13-bit ident, 10-bit op, and 17-bit function field, followed on the wire by a 15-bit BCH check and one overall parity bit." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="26" width="30" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="31" y="43" text-anchor="middle" font-size="7.5" fill="currentColor">T 1</text>
  <rect x="46" y="26" width="46" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="69" y="43" text-anchor="middle" font-size="7.5" fill="currentColor">Pfx 7</text>
  <rect x="92" y="26" width="86" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="135" y="43" text-anchor="middle" font-size="7.5" fill="currentColor">Ident 13</text>
  <rect x="178" y="26" width="70" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="213" y="43" text-anchor="middle" font-size="7.5" fill="currentColor">Op 10</text>
  <rect x="248" y="26" width="110" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="303" y="43" text-anchor="middle" font-size="7.5" fill="currentColor">Function 17</text>
  <rect x="358" y="26" width="74" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="395" y="41" text-anchor="middle" font-size="7.5" fill="currentColor">BCH 15</text>
  <rect x="432" y="26" width="22" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="443" y="41" text-anchor="middle" font-size="6.5" fill="currentColor">P</text>
  <text x="16" y="76" font-size="8" fill="currentColor">48-bit information field (bits 47..0) · 15-bit BCH (bits 62..48) · even parity (bit 63)</text>
</svg>
<figcaption>The 48-bit MPT 1327 information field carries five fields; on the wire it is followed by a 15-bit BCH check and one overall even-parity bit, giving a 64-bit codeword that corrects a single flipped bit.</figcaption>
</figure>

## Field layout

GopherTrunk supports two wire-level layouts. The spec-complete **48-bit** information field
packs five fields MSB-first:

| Bits | Width | Field | Meaning |
|---|---|---|---|
| 47 | 1 | Type | 0 = address codeword, 1 = data codeword |
| 46..40 | 7 | Prefix | Area-code prefix |
| 39..27 | 13 | Ident | Radio / fleet identity (the address) |
| 26..17 | 10 | Op | Operation / opcode field |
| 16..0 | 17 | Function | Opcode-specific information bits |

The historical **38-bit** legacy layout drops the 10-bit Op field and shifts Function into
the low bits — GopherTrunk keeps it for back-compatible tests and fixtures that pre-date the
BCH wiring. The full 10-bit Op field only populates under the 48-bit / `BCHOn` path.

## Opcodes

MPT 1327 categorises an address codeword by the top 4 bits of its 17-bit Function field —
the spec's "Address Categorisation" subfield. GopherTrunk decodes the kinds most useful for
trunk-following:

| Kind | Mnemonic | Meaning |
|---|---|---|
| Aloha | ALH | Control-channel idle |
| Ahoy | AHY | Paging / inquiry |
| AhoyChan | AHYC | Broadcast / system info |
| GoToChannel | GTC | Voice [channel grant](/reference/channel-grant/) |
| Ack | ACK | Acknowledgement |
| Disconnect | DUL | Disconnect |
| Data | SAMO | Data-request shorthand |
| Emergency | — | Emergency call |

A **GTC** ("go to channel") is the grant a follower acts on: the codeword's Prefix + Ident
name the called party, and the lower 13 bits of Function carry the assigned channel number
the radio retunes to. An **AHYC** broadcast carries a sub-system identifier the engine treats
as the system ID when it locks to the control channel.

## Forward error correction

On the wire the 48-bit information field is wrapped in a 64-bit codeword: bits 0..47 are the
information field, bits 48..62 are a 15-bit BCH check (a CRC-15 with polynomial **0x6815** and
initial fill **0x0001**), and bit 63 is an overall even-parity bit over the 63-bit body. The
generator polynomial is g(x) = x^15 + x^14 + x^13 + x^11 + x^4 + x^2 + 1 — representable as
0xE815 with the implicit x^15 leading term, or 0x6815 without it, the form most
[CRC](/reference/cyclic-redundancy-check/) implementations use. GopherTrunk's primitive
follows the most-cited public reference, sdrtrunk's `CRCFleetsync` (Fleetsync and MPT 1327
share the code). The BCH structure corrects up to one bit error anywhere in the 64 codeword
bits: a single flip changes both the BCH syndrome (which locates a position in bits 0..62)
and the overall parity (which catches a flip in bit 63).

## In practice

MPT 1327 rides on 1200-baud FFSK layered over an otherwise ordinary NBFM channel, so it is
2-level rather than the 4-level modulation the newer digital trunking protocols use — the
receiver hands the codeword state machine a plain bit stream, not dibits. The field positions
follow the most-cited public reference; vendor extensions repurpose the Function field's
sub-layout, so the interpretation is best-effort and worth cross-checking against a live
capture before trusting an unusual codeword. Because each codeword is short and
CRC-protected, a monitor recovers useful activity even under marginal signal: good codewords
pass, bad ones drop, and the control stream re-announces enough that missed grants are
usually caught on a later slot.

## Relevance to SDR

`internal/radio/mpt1327/codeword.go` holds the 38- and 48-bit codeword packing;
`opcodes.go` decodes the Address Categorisation kinds into structured grants and broadcasts;
and `internal/radio/framing/bch_mpt1327.go` implements the BCH(64,48) check. Together they let
GopherTrunk follow an MPT 1327 system — read the GTC grants, retune to the assigned channel,
and track the site identity from the AHYC broadcasts.

## Sources

[^mpt]: [MPT-1327](https://en.wikipedia.org/wiki/MPT-1327) — Wikipedia, on the UK MPT 1327 trunking signalling standard.
[^crc]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, on the CRC/BCH construction that protects each codeword.
