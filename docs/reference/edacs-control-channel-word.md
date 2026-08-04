---
slug: edacs-control-channel-word
title: EDACS Control Channel Word
entry_type: term
category: trunked-radio
description: The EDACS Control Channel Word (CCW) is the 40-bit signalling block that rides every GE/Ericsson EDACS control-channel slot, packing a command, status, address, logical channel, and auxiliary field behind a 24-bit sync and a BCH(40,28,2) FEC code.
keywords: EDACS control channel word, CCW, EDACS signalling, GE Ericsson EDACS, BCH 40 28 2, EDACS opcode, ProVoice grant, EDACS sync 55D5AA, logical channel number
aka: [CCW, "EDACS CCW", "control channel word"]
autolink: true
infobox:
  - { label: Length, value: 40 bits (5 bytes) }
  - { label: Fields, value: "Command 4 · Status 4 · Address 16 · LCN 5 · Aux 11" }
  - { label: Sync, value: "0x55D5AA (24 bits)" }
  - { label: FEC, value: "BCH(40,28,2), generator 0x1539" }
see_also: [edacs, trunked-radio, control-channel, channel-grant, bch-code, motorola-osw, mpt-1327-codeword, frame-synchronization]
cite_urls:
  - https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System
  - https://en.wikipedia.org/wiki/BCH_code
---

The **EDACS Control Channel Word** (**CCW**) is the 40-bit signalling block that rides every
[EDACS](/reference/edacs/) control-channel slot after sync detection.[^edacs] EDACS — the
GE / Ericsson trunking system — sends a steady stream of these words on its outbound
[control channel](/reference/control-channel/), each announcing one piece of system business:
a voice or data [channel grant](/reference/channel-grant/), a system-identity broadcast, an
adjacent-site pointer, or an idle heartbeat. Every CCW is prefaced by a 24-bit sync sequence
and protected by a [BCH(40,28,2)](/reference/bch-code/) code so it survives a noisy channel.[^bch]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="An EDACS control-channel slot: a 24-bit sync word 0x55D5AA followed by the 40-bit Control Channel Word, which splits into a 4-bit command, 4-bit status, 16-bit address, 5-bit logical channel number, and 11-bit auxiliary field, with the same 40 bits doubling as a BCH codeword of 28 information bits plus 12 parity bits." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="26" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="51" y="43" text-anchor="middle" font-size="8" fill="currentColor">sync · 24</text>
  <rect x="86" y="26" width="44" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="108" y="43" text-anchor="middle" font-size="8" fill="currentColor">Cmd 4</text>
  <rect x="130" y="26" width="44" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="152" y="43" text-anchor="middle" font-size="8" fill="currentColor">Sts 4</text>
  <rect x="174" y="26" width="120" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="234" y="43" text-anchor="middle" font-size="8" fill="currentColor">Address 16</text>
  <rect x="294" y="26" width="52" height="26" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="320" y="43" text-anchor="middle" font-size="8" fill="currentColor">LCN 5</text>
  <rect x="346" y="26" width="108" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="400" y="43" text-anchor="middle" font-size="8" fill="currentColor">Aux 11</text>
  <path d="M130 60 L454 60" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
  <text x="240" y="78" text-anchor="middle" font-size="8" fill="currentColor">the same 40 wire bits = BCH(40,28,2) codeword</text>
  <text x="240" y="92" text-anchor="middle" font-size="7.5" fill="currentColor">bits 39..12 information (28) · bits 11..0 parity (12)</text>
</svg>
<figcaption>Each EDACS control slot is a 24-bit sync followed by the 40-bit CCW. GopherTrunk reads the same 40 bits two ways: as five logical fields, and — with FEC enabled — as a BCH(40,28,2) codeword whose low 12 bits are parity.</figcaption>
</figure>

## Field layout

The CCW packs five fields MSB-first across its 40 bits:

| Bits | Width | Field | Meaning |
|---|---|---|---|
| 39..36 | 4 | Command | Operation type — grant, system, idle, … |
| 35..32 | 4 | Status | Per-command flag bits (bit 0 encryption, bit 1 emergency) |
| 31..16 | 16 | Address | Talkgroup, radio ID, or site ID (per Command) |
| 15..11 | 5 | LCN | Logical channel number into the operator band plan |
| 10..0 | 11 | Aux | Command-specific auxiliary parameter |

The 4-bit Command field is the opcode. GopherTrunk models the subset most useful for
trunk-following:

| Command | Value | Meaning |
|---|---|---|
| Idle | 0x0 | Control-channel idle / heartbeat |
| GroupVoiceGrant | 0x1 | Group voice [channel grant](/reference/channel-grant/) |
| ProVoiceGrant | 0x2 | EDACS [ProVoice](/reference/provoice/) (digital) grant |
| IndividualCall | 0x3 | Unit-to-unit call |
| DataGrant | 0x4 | Data channel grant |
| SystemID | 0x5 | System-identity broadcast |
| AdjacentSite | 0x6 | Neighbour-site announcement |
| Emergency | 0x7 | Emergency call |
| Affiliation | 0x8 | Unit affiliation |
| Encryption | 0x9 | Encryption signalling |
| Reserved | 0xF | Reserved |

A voice grant carries the talkgroup in the Address field and an LCN that indexes an
operator-supplied band plan; the receiving radio retunes to the physical channel that LCN
names. The Status nibble layers flags onto a grant — encryption in bit 0, emergency in bit 1.

## Forward error correction

Under GopherTrunk's `BCHOn` mode the 40 on-wire bits are not raw field bits but a
**BCH(40,28,2)** codeword: 28 information bits in the high positions (39..12) and a 12-bit
parity field in the low positions (11..0). The code is a *shortened* form of the
BCH(63,51,2) mother code over GF(2^6) with primitive polynomial x^6 + x + 1. Its generator
is the product of the minimal polynomials of α and α^3:

- m₁(x) = x^6 + x + 1
- m₃(x) = x^6 + x^4 + x^2 + x + 1
- g(x) = m₁(x) · m₃(x) = x^12 + x^10 + x^8 + x^5 + x^4 + x^3 + 1 = **0x1539**

The code corrects up to t = 2 bit errors per codeword (designed minimum distance d = 5).
GopherTrunk's decoder computes the 12-bit syndrome, applies a single-bit correction from a
per-position syndrome table, and — failing that — searches the 780 ordered position pairs for
a matching double-bit error before re-deriving the fields. Per the canonical open reference
(lwvmobile/edacs-fm), BCH is the *only* on-wire FEC on the Standard EDACS CCW; no outer
interleaved or Reed-Solomon layer sits above it.

## In practice

The 24-bit outbound sync is documented as **0x55D5AA** across the public reference
implementations GopherTrunk tracks; a sliding [detector](/reference/frame-synchronization/)
locks onto it (tolerating a mismatched bit or two) and the adapter then slices the 40-bit CCW
that follows. Both the field-position and opcode assignments follow the most-cited public
reference (lwvmobile/edacs-fm) rather than a paywalled EDACS specification, so they are
best-effort — worth cross-checking against a fresh live capture before trusting an unusual
Command or Status interpretation. Strict-mode operators reject any CCW whose Command falls
outside the recognised set, since an unknown opcode is a strong signal of bit errors or a
misaligned codeword.

## Relevance to SDR

`internal/radio/edacs/ccw.go` holds the CCW packing (`ParseCCW`, `CCWFromBits`);
`opcodes.go` interprets the Command and Status fields into structured grants; and
`internal/radio/framing/bch_edacs.go` implements the BCH(40,28,2) encode/decode. Together
they let GopherTrunk follow an EDACS system: read the grant CCWs, map each LCN through the
band plan, and retune to the assigned voice channel in step with the fielded radios.

## Sources

[^edacs]: [Enhanced Digital Access Communications System](https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System) — Wikipedia, on the GE/Ericsson EDACS trunking system and its control channel.
[^bch]: [BCH code](https://en.wikipedia.org/wiki/BCH_code) — Wikipedia, on the error-correcting code family that protects the CCW.
