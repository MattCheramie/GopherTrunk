---
slug: dmr-cach
title: DMR CACH
entry_type: term
category: trunked-radio
description: The Common Announcement Channel (CACH) is the short 24-bit field a DMR base station interleaves between outbound bursts to carry TACT signalling — access type and logical-channel state — though GopherTrunk currently uses it only as a timing marker and does not decode TACT.
keywords: DMR CACH, common announcement channel, TACT, access type bits, TDMA channel, DMR outbound, 24-bit CACH, ETSI TS 102 361-1
aka: ["CACH", "common announcement channel"]
autolink: true
infobox:
  - { label: Length, value: 24 bits (12 dibits) }
  - { label: Carries, value: "TACT (access type, LCN)" }
  - { label: Direction, value: BS outbound only }
  - { label: Spec, value: ETSI TS 102 361-1 §6.2 }
see_also: [dmr-burst, dmr-voice-superframe, dmr-sync-patterns, tdma, control-channel, dmr, dmr-tier-2, dmr-tier-3]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

The **DMR CACH** (**Common Announcement Channel**) is the short 24-bit field a base station
interleaves *between* the two [TDMA](/reference/tdma/) timeslots on its outbound
carrier.[^wiki] Because a DMR base station transmits continuously — filling both slots — it uses
the gap between one slot's burst and the next to insert a CACH, giving mobiles a channel that
is present on every outbound frame regardless of which timeslot they follow.[^tdma] The CACH
carries **TACT** bits describing the access state of the channels, and it also serves as a
reliable inter-burst timing landmark.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="A DMR outbound carrier alternating slot-1 and slot-2 bursts with a short 12-dibit CACH inserted before each burst, so same-slot bursts sit 288 dibits apart instead of 264." xmlns="http://www.w3.org/2000/svg">
  <rect x="14" y="34" width="26" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/>
  <text x="27" y="51" text-anchor="middle" font-size="6.5" fill="currentColor">CACH</text>
  <rect x="40" y="34" width="90" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="85" y="51" text-anchor="middle" font-size="8" fill="currentColor">TS1 burst</text>
  <rect x="130" y="34" width="26" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/>
  <text x="143" y="51" text-anchor="middle" font-size="6.5" fill="currentColor">CACH</text>
  <rect x="156" y="34" width="90" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="201" y="51" text-anchor="middle" font-size="8" fill="currentColor">TS2 burst</text>
  <rect x="246" y="34" width="26" height="26" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1"/>
  <text x="259" y="51" text-anchor="middle" font-size="6.5" fill="currentColor">CACH</text>
  <rect x="272" y="34" width="90" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="317" y="51" text-anchor="middle" font-size="8" fill="currentColor">TS1 burst</text>
  <path d="M85 74 L85 84 L317 84 L317 74" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="201" y="100" text-anchor="middle" font-size="8" fill="currentColor">same-slot stride = 288 dibits (2×132 + 2×12 CACH)</text>
</svg>
<figcaption>On an outbound DMR carrier a 12-dibit CACH precedes each burst, so two bursts of the same logical call sit 288 dibits apart rather than the 264 of a CACH-free stream.</figcaption>
</figure>

## Structure and TACT

The CACH is 24 bits — 12 dibits — laid out per ETSI TS 102 361-1 §6.2. Seven of those bits
form the **TACT** (Terminator / Access-Type / Channel-Type) block, protected by a short
Hamming/QR code, with the remaining bits assigned to the interleaved payload of a slow
signalling channel. TACT encodes, among other things, the **AT** (access type — whether the
associated logical channel is idle or busy) and the **TC** (TDMA channel number, identifying
which of the two logical slots the following burst belongs to), plus an **LCSS**-style pair
marking the position of the CACH's own payload fragment within a longer message. A mobile reads
TACT to know, frame by frame, whether a slot is free before it attempts access.

## GopherTrunk's use: cadence only

GopherTrunk does **not** currently decode TACT. In the voice decode path
(`internal/radio/dmr/voice/superframe.go`) the CACH is treated purely as a *timing* artefact:
its presence changes the dibit distance between two same-slot bursts. On a CACH-free stream
(direct mode, or a replayed capture) same-slot bursts are `2 × 132 = 264` dibits apart; on a
live BS-sourced outbound carrier a 12-dibit CACH before each burst stretches that to
`2 × (132 + 12) = 288`. The interleaved decoder auto-detects which stride is in use per call
(`cachDibits = 12`, `NewInterleavedDecoder`) and locks it, so the CACH never has to be decoded
for the cadence to be recovered. **Impl-gap:** the TACT access-type and channel bits are not
parsed — the field is consumed as a spacing constant, not as signalling. Reading TACT would let
GopherTrunk track slot busy/idle state directly rather than inferring cadence from AMBE FEC
quality.

## Relevance to SDR

For a scanner the CACH's most valuable property today is exactly the timing role GopherTrunk
uses: distinguishing an outbound (CACH-bearing, 288-stride) carrier from a direct-mode or
replayed (264-stride) stream is what lets the voice decoder slice bursts B–F at the correct
same-slot cadence and avoid pulling dibits from the wrong timeslot. Promoting the CACH to a
fully-decoded TACT channel is a natural extension: it would surface per-slot access state and
give a second, independent cross-check on which logical channel each burst carries.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its outbound common signalling.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on the two-slot framing between which the CACH is interleaved.
