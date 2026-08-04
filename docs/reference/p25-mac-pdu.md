---
slug: p25-mac-pdu
title: P25 MAC PDU
entry_type: term
category: trunked-radio
description: The MAC PDU is the P25 Phase 2 signalling unit that replaces the Phase 1 TSBK — an opcode-led message riding the two-slot TDMA MAC slots, carrying voice-channel grants, channel status, network broadcasts and the PTT/active/idle state a trunked call moves through.
keywords: P25 MAC PDU, Phase 2 MAC opcode, MAC_PTT, MAC_ACTIVE, network status broadcast, group voice channel grant, replaces TSBK, TIA-102.AABF BBAB
aka: ["MAC PDU", "Phase 2 MAC message", "MAC opcode"]
autolink: true
infobox:
  - { label: Role, value: Phase 2 signalling (replaces TSBK) }
  - { label: Rides on, value: TDMA MAC slots }
  - { label: After FEC, value: opcode + up to 17 payload bytes }
  - { label: Spec, value: TIA-102.AABF / BBAB }
see_also: [tsbk, p25-mac-vendor, p25-phase-2, control-channel, channel-grant, network-access-code, p25-reed-solomon, p25-trellis-code, p25-isch]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Protocol_data_unit
---

The **P25 MAC PDU** (Medium Access Control Protocol Data Unit) is the Phase 2 signalling unit that
does the job the [TSBK](/reference/tsbk/) does on [Phase 1](/reference/p25-phase-1/).[^wiki] Where
Phase 1 puts trunking blocks on a dedicated control channel, Phase 2 folds signalling into the same
two-slot TDMA traffic structure: MAC PDUs ride the **MAC slots** — the sub-frames the
[ISCH](/reference/p25-isch/) types as MAC rather than voice — and carry the
[channel grants](/reference/channel-grant/), status broadcasts, and call-state transitions that keep a
[trunked](/reference/control-channel/) system coordinated.[^pdu] After forward-error-correction removal
each PDU is an **opcode** followed by an opcode-specific payload.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A MAC PDU after FEC removal: one opcode byte, an optional manufacturer-ID byte present only for vendor opcodes, then up to 17 payload bytes; standard opcodes omit the MFID and start their payload at byte 1." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="60" height="28" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="50" y="51" text-anchor="middle" font-size="8.5" fill="currentColor">opcode</text>
  <rect x="80" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1" stroke-dasharray="3 2"/>
  <text x="115" y="49" text-anchor="middle" font-size="8" fill="currentColor">MFID</text>
  <text x="115" y="59" text-anchor="middle" font-size="6.5" fill="currentColor">vendor only</text>
  <rect x="150" y="34" width="290" height="28" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="295" y="51" text-anchor="middle" font-size="8.5" fill="currentColor">opcode-specific payload · up to 17 bytes</text>
  <text x="20" y="88" font-size="8" fill="currentColor">18 information bytes after outer RS(24,16,9) + trellis FEC removal</text>
  <text x="20" y="108" font-size="8" fill="currentColor">standard opcode → payload at byte 1 · manufacturer-specific opcode (0x80–0xBF) → MFID at byte 1</text>
</svg>
<figcaption>A decoded MAC PDU is an opcode plus payload; only opcodes in the 0x80–0xBF manufacturer-specific range insert an MFID byte between the two.</figcaption>
</figure>

## Standard opcodes

GopherTrunk enumerates the subset of TIA-102.AABF / BBAB standard opcodes the follow-along engine needs:

| Opcode | Name | Purpose |
| --- | --- | --- |
| 0x01 | GroupVoiceChannelUser (abbreviated) | In-call source-radio + service options |
| 0x02 | MAC_END | End of transmission |
| 0x03 | MAC_IDLE | Channel idle |
| 0x05 | MAC_HANGTIME | Hang-time |
| 0x06 | MAC_ACTIVE | Late-grant active update |
| 0x21 | GroupVoiceChannelUser (extended) | In-call user, carries SUID |
| 0x40 | GroupVoiceChannelGrantUpdate | Grant update |
| 0x44 | GroupVoiceChannelGrant | Group voice grant |
| 0x46 | UnitToUnitGrantUpdate (abbreviated) | Private-call grant update |
| 0x48 | UnitToUnitVoiceChannelGrant | Private-call grant |
| 0x49 | UnitToUnitVoiceChannelGrantUpdate | Private-call grant update |
| 0x4C | GroupAffiliationResponse | Affiliation response |
| 0x4D | UnitRegistrationResponse | Registration response |
| 0x70 | EncryptionSync | ALGID / KID / MI |
| 0x7D | IdentifierUpdate | Band-plan definition |
| 0xFA | RFSSStatusBroadcastUpdate | RFSS status |
| 0xFB | NetworkStatusBroadcastUpdate | WACN / System ID / Color Code |

A few of these carry state that the rest of the decoder depends on. The **Network Status Broadcast –
Update** (0xFB) publishes the WACN, System ID and [Color Code](/reference/network-access-code/) that seed
the [PN44 scrambler](/reference/pn44-scrambler/). **IdentifierUpdate** (0x7D) supplies the band plan that
turns a grant's channel number into a downlink frequency. And the encryption sync a follower needs is
not a distinct opcode on the grant — it arrives in the **MAC_PTT** message that begins a transmission,
which the ISCH marks with the `MAC_PTT` slot type, so a caller telling PTT signalling from ordinary
signalling must read the slot type alongside the PDU. The `GroupVoiceChannelUser` broadcasts (0x01 /
0x21) backfill the source RID and encryption flag in-call, because real systems often issue the initial
grant in a compressed form with those fields zeroed.

## Parsing and FEC

The MAC slot's coded burst is descrambled, de-interleaved, trellis-decoded and (optionally) checked
against the outer [Reed-Solomon RS(24,16,9)](/reference/p25-reed-solomon/) code before the 18 information
bytes reach `ParseMACPDU`. The parser reads byte 0 as the opcode; if the opcode is in the
manufacturer-specific range 0x80–0xBF it consumes byte 1 as the MFID and hands off to the
[vendor accessors](/reference/p25-mac-vendor/), otherwise the payload begins at byte 1. Callers then
dispatch on the opcode to a typed accessor — `AsGroupVoiceChannelGrant`, `AsNetworkStatusBroadcast`,
`AsIdentifierUpdate`, and so on — each of which validates the payload length before unpacking. A strict
mode drops any PDU whose opcode is outside the recognised set, which matters because the outer FEC's
[trellis](/reference/p25-trellis-code/) corrector can occasionally hand up a plausible-looking but
spurious burst.

## Relevance to SDR

`internal/radio/p25/phase2/mac.go` defines the `MACPDU` type, the `Opcode` enum, `ParseMACPDU`, and the
typed accessors for grants, network status, and in-call user broadcasts. It is the layer that turns a
decoded Phase 2 MAC slot into a trunking event — a grant to follow, a band plan to store, a call to end.
The spec is TIA-102.AABF (message content) and TIA-102.BBAB (Phase 2 framing).

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 Phase 2 trunking signalling.
[^pdu]: [Protocol data unit](https://en.wikipedia.org/wiki/Protocol_data_unit) — Wikipedia, on the PDU as a self-contained protocol message.
