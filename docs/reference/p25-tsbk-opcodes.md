---
slug: p25-tsbk-opcodes
title: P25 TSBK opcodes
entry_type: term
category: trunked-radio
description: The P25 TSBK opcode is the 6-bit field naming a control-channel message type — a channel grant, a registration response, a status broadcast, or a band-plan update — per the standard TIA-102.AABC-D OSP table.
keywords: P25 TSBK opcode, OSP, outbound signalling packet, GRP_V_CH_GRANT, NET_STS_BCST, IDEN_UP, ADJ_STS_BCST, TIA-102.AABC, P25 control channel opcode table
aka: [OSP, "outbound signalling packet", "TSBK opcode"]
autolink: true
infobox:
  - { label: Field, value: 6-bit opcode }
  - { label: Namespace, value: "MFID = standard (0x00/0x01)" }
  - { label: Direction, value: OSP (outbound, FNE → unit) }
  - { label: Spec, value: TIA-102.AABC-D Table 7-1 }
see_also: [tsbk, channel-grant, csbk, control-channel, network-access-code, wacn, system-id, p25-tsbk-vendor-opcodes, p25-identifier-update, p25-phase-1]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **P25 TSBK opcode** is the 6-bit field that names the message type carried in a
[TSBK](/reference/tsbk/) on the [control channel](/reference/control-channel/).[^wiki] Every
standard opcode is an **OSP** (**Outbound Signalling Packet**) — a message the fixed network
equipment sends outbound to radios — enumerated in the TIA-102.AABC-D **Table 7-1** registry.
The opcode tells the decoder how to read the eight argument bytes that follow: a Group Voice
Channel Grant carries a [talkgroup](/reference/talkgroup/) and a channel number, while a
Network Status Broadcast carries the [WACN](/reference/wacn/), [system ID](/reference/system-id/),
and control-channel identity.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="A TSBK block split into a manufacturer ID byte, a 6-bit opcode field selecting the message type, an 8-byte argument payload, and a trailing CRC; the opcode value indexes the standard OSP table." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="34" width="70" height="28" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="55" y="52" text-anchor="middle" font-size="8.5" fill="currentColor">MFID</text>
  <rect x="90" y="34" width="78" height="28" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="129" y="49" text-anchor="middle" font-size="8.5" fill="currentColor">opcode</text>
  <text x="129" y="58" text-anchor="middle" font-size="6.5" fill="currentColor">6 bits</text>
  <rect x="168" y="34" width="210" height="28" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="273" y="52" text-anchor="middle" font-size="8.5" fill="currentColor">8 argument bytes</text>
  <rect x="378" y="34" width="62" height="28" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="409" y="52" text-anchor="middle" font-size="8.5" fill="currentColor">CRC</text>
  <text x="90" y="86" font-size="8" fill="currentColor">0x00 → GRP_V_CH_GRANT · 0x3B → NET_STS_BCST · 0x3D → IDEN_UP …</text>
  <text x="90" y="100" font-size="7.5" fill="currentColor">the opcode selects how the 8 argument bytes are read</text>
</svg>
<figcaption>The 6-bit opcode selects one row of the OSP table, which fixes the meaning of the TSBK's eight argument bytes; the MFID byte before it switches between the standard and vendor namespaces.</figcaption>
</figure>

## The standard OSP opcodes

The table below lists the opcodes GopherTrunk names in
`internal/radio/p25/phase1/opcodes.go`, with the canonical TIA mnemonic and what each
message carries. Hex values are the 6-bit opcode.

| Opcode | Mnemonic | Carries |
|---|---|---|
| `0x00` | GRP_V_CH_GRANT | Group voice grant: service options, channel, talkgroup, source unit |
| `0x02` | GRP_V_CH_GRANT_UPDT | Up to two active (channel, talkgroup) pairs |
| `0x03` | GRP_V_CH_GRANT_UPDT_EXP | Explicit-channel group update (grant-shaped payload) |
| `0x04` | UU_V_CH_GRANT | Unit-to-unit grant: channel, target unit, source unit |
| `0x05` | UU_ANS_REQ | Answer request for a private call (no channel yet) |
| `0x08` | TELE_INT_CH_GRANT | Telephone-interconnect grant + call timer |
| `0x14` | SNDCP_DAT_CH_GRANT | Packet-data (SNDCP) channel grant |
| `0x1F` | CALL_ALRT | Call-alert page to a unit |
| `0x20` | ACK_RSP_FNE | Acknowledge response from the network |
| `0x27` | DENY_RSP | Service denied |
| `0x28` | GRP_AFF_RSP | Group [affiliation](/reference/affiliation/) response |
| `0x2B` | LOC_REG_RSP | Location registration response (RFSS + site) |
| `0x2C` | U_REG_RSP | Unit [registration](/reference/registration/) response (system ID + WUID) |
| `0x2F` | U_DE_REG_ACK | Deregistration acknowledgement |
| `0x30` | SYNC_BCST | System synchronisation broadcast |
| `0x33` | IDEN_UP_TDMA | Band-plan slot, Phase 2 TDMA form |
| `0x34` | IDEN_UP_VU | Band-plan slot, VHF/UHF form |
| `0x38` | SYS_SRV_BCST | System-service capability broadcast |
| `0x39` | SCCB | Secondary control-channel broadcast |
| `0x3A` | RFSS_STS_BCST | Camped RFSS + site identity |
| `0x3B` | NET_STS_BCST | WACN, system ID, control channel |
| `0x3C` | ADJ_STS_BCST | Adjacent (neighbour) site + its control channel |
| `0x3D` | IDEN_UP | Band-plan slot, 700/800/900 MHz form |

The grant opcodes drive a scanner's trunk-following: a
[channel grant](/reference/channel-grant/) points radios (and a monitor) at a voice
channel, and the [Identifier Update](/reference/p25-identifier-update/) opcodes
(`0x3D`/`0x34`/`0x33`) supply the base frequency and spacing needed to turn the grant's
channel number into a tunable frequency.

## Namespace and dispatch

The opcode alone is not enough to decode a block — the same 6-bit value means different
things under different manufacturers. The TSBK's MFID (manufacturer ID) header byte selects
the namespace: MFID `0x00`/`0x01` is the standard OSP table above, while MFID `0x90`
(Motorola) and `0xA4` (Harris) reinterpret the opcode against the
[vendor table](/reference/p25-tsbk-vendor-opcodes/). GopherTrunk dispatches on MFID first,
then opcode, so a Motorola opcode `0x02` decodes as a patch-group grant rather than a
standard group-voice update.

## Relevance to SDR

GopherTrunk's `Opcode` type and its `String()` method render each block in spec terms
(`GRP_V_CH_GRANT`, `NET_STS_BCST`, …) so logs read the way the standard documents do;
unnamed opcodes fall back to `OSP(0xNN)`. The trunking engine acts on the grant and update
opcodes to follow calls, folds the status-broadcast opcodes into a running picture of the
system's identity and neighbours, and feeds the Identifier Update opcodes into the band
plan. Decoding this opcode set correctly is the core of following a P25 trunked system.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its control-channel signalling. Opcode values follow TIA-102.AABC-D Table 7-1.
