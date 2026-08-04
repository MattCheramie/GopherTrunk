---
slug: dmr-csbk-payloads
title: DMR CSBK payloads
entry_type: term
category: trunked-radio
description: The DMR CSBK payloads are the opcode-specific contents of a Tier III control block — the channel-grant, Aloha, and broadcast layouts a decoder reads after the CSBKO opcode to follow a trunked DMR system.
keywords: DMR CSBK payload, CSBKO opcode, C_ALOHA, C_BCAST, TV_GRANT, PV_GRANT, LPCN timeslot, adjacent site, ETSI TS 102 361-4
aka: ["CSBK opcodes", "CSBKO", "CSBK payload layouts"]
autolink: true
infobox:
  - { label: Opcode field, value: CSBKO (6 bits) }
  - { label: Aloha, value: "0x19 (C_ALOHA)" }
  - { label: Voice grants, value: "0x30–0x32" }
  - { label: Broadcast, value: "0x28 (C_BCAST) + sub-types" }
see_also: [csbk, channel-grant, dmr-tier-3, talkgroup, radio-id, neighbor-site, rest-channel, dmr-vendor-fid, dmr-bandplan]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/
---

The **DMR CSBK payloads** are the opcode-specific contents of a Tier III
[CSBK](/reference/csbk/) — what a decoder reads once the 6-bit CSBKO opcode has told it which
kind of control message the 96-bit block is.[^wiki] The opcode selects the layout of the
64-bit payload between the block's header octets and its CRC: a channel grant carries an
address and a channel number, an Aloha advertises the control channel, and a broadcast carries
one of several system-parameter sub-types.[^etsi] Reading these correctly is what lets a
monitor follow a trunked DMR system.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A CSBK information block split into a last-block and protect flag, a six-bit CSBKO opcode, a feature-set ID octet, a 64-bit opcode-specific payload, and a 16-bit CRC; the opcode selects how the payload octets are interpreted." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="34" width="40" height="26" fill="currentColor" fill-opacity="0.30"/><text x="40" y="50">LB·PF</text>
    <rect x="60" y="34" width="60" height="26" fill="currentColor" fill-opacity="0.22"/><text x="90" y="50">CSBKO</text>
    <rect x="120" y="34" width="46" height="26" fill="currentColor" fill-opacity="0.14"/><text x="143" y="50">FID</text>
    <rect x="166" y="34" width="200" height="26" fill="none"/><text x="266" y="50">payload · 64 bits</text>
    <rect x="366" y="34" width="74" height="26" fill="currentColor" fill-opacity="0.22"/><text x="403" y="50">CRC</text>
  </g>
  <path d="M90 60 L90 74 L266 74 L266 62" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="230" y="92" text-anchor="middle" font-size="8" fill="currentColor">opcode selects the payload layout</text>
  <text x="230" y="106" text-anchor="middle" font-size="7.5" fill="currentColor">grant · Aloha · broadcast sub-type · acknowledgement</text>
</svg>
<figcaption>The CSBKO opcode names the message; the 64-bit payload that follows is interpreted differently for a grant, an Aloha beacon, or a broadcast sub-type.</figcaption>
</figure>

## The opcode set

GopherTrunk decodes the standard-FID (0x00) opcode set defined in ETSI TS 102 361-4 §7. The
grant family and the two beacons are the load-bearing ones for following a system:

| CSBKO | Name | Meaning |
|-------|------|---------|
| 0x19 | C_ALOHA | Periodic control-channel beacon the receiver locks onto |
| 0x1C | C_AHOY | Outbound poll / service request |
| 0x1E | C_ACKVIT | Acknowledged response (vital) |
| 0x1F | C_RAND | Random-access service request (inbound) |
| 0x20 / 0x21 | C_ACKD / C_ACKU | Acknowledged downlink / uplink response |
| 0x26 | C_NACK | Negative acknowledgement |
| 0x28 | C_BCAST | Broadcast / announcement (carries sub-types) |
| 0x2A / 0x2E / 0x2F | P_MAINT / P_CLEAR / P_PROTECT | Maintenance, channel clear, protect |
| 0x30 | PV_GRANT | Private voice channel grant |
| 0x31 | TV_GRANT | TalkGroup voice channel grant |
| 0x32 | BTV_GRANT | Broadcast TalkGroup voice channel grant |
| 0x33 / 0x34 | PD_GRANT / TD_GRANT | Private / TalkGroup data channel grant |
| 0x39 | C_MOVE | Move to another control channel |
| 0x3D | Preamble | Lead-in block before a following CSBK/MBC chain |

## Grant and Aloha layouts

Every voice- and data-grant opcode (0x30–0x34) shares one content layout, and getting its
lead field right matters: the payload **starts with the Logical Physical Channel Number
(LPCN)**, not with service options. Within the 8-octet payload:

| Field | Position | Size |
|-------|----------|------|
| LPCN (logical physical channel number) | bits 0–11 | 12 bits |
| Timeslot (0 = TS1, 1 = TS2) | bit 12 | 1 bit |
| Target address (talkgroup or subscriber) | octets 2–4 | 24 bits |
| Source / subscriber address | octets 5–7 | 24 bits |

The LPCN feeds a per-system [band-plan resolver](/reference/dmr-bandplan/) to recover the
downlink frequency; the target and source are the [talkgroup](/reference/talkgroup/) or
[radio ID](/reference/radio-id/) the [channel grant](/reference/channel-grant/) is for. An
earlier layout read the LCN from octet 7, which is actually the low byte of the 24-bit source
address, so the decoded channel changed with every transmitting radio (GopherTrunk issue
\#639) — a reminder that these octet offsets are exact, not approximate. A **C_ALOHA** payload
instead leads with a Site Time Slot nibble and a set of random-access parameters, then a
16-bit system identity the control-channel state machine uses as a stable lock key.

## Broadcast sub-types

The single trap for newcomers is that the trunking-system parameters, general site
parameters, adjacent-site lists, and call-timer values are **not** standalone opcodes. They
are sub-types (an `anncd_type` field in the top of the first payload octet) carried *inside*
C_BCAST (0x28):

| anncd_type | Sub-type |
|------------|----------|
| 0 | Announce / withdraw TSCC |
| 1 | Call-timer parameters |
| 2 | Vote-now advice |
| 3 | Broadcast local time |
| 4 | Mass registration |
| 5 | Logical-channel / frequency relationship |
| 6 | Adjacent-site information |
| 7 | General site parameters |

The adjacent-site sub-type (6) is how a decoder learns a [neighbour site](/reference/neighbor-site/)'s
system, site, control-channel LCN, and colour code; the general-site sub-type (7) carries the
camped site's own identity. Because the sub-field octets follow the `anncd_type` octet, they
are read from `payload[1:]` — a detail validated against real off-air C_BCAST bursts.

## Relevance to SDR

`internal/radio/dmr/tier3/csbk.go` defines the opcode constants and `internal/radio/dmr/tier3/payloads.go`
holds the per-opcode parsers — `ParseTVGrant`, `ParsePVGrant`, `ParseDataGrant`, `ParseAloha`,
`ParseBroadcast`, `ParseAdjacentSite`, and the site-parameter readers — each pinned against the
dsd-neo reference decoder and, where captures exist, real bursts. The grant parsers drive the
engine's retune-and-follow; the Aloha and broadcast parsers keep the control channel locked and
build the neighbour map. Opcodes carrying a vendor feature-set ID are dispatched separately
before this standard set is consulted (see [DMR vendor FID](/reference/dmr-vendor-fid/)).

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR control signalling and Tier III trunking.
[^etsi]: [ETSI TS 102 361-4 (DMR Tier III)](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/) — ETSI, §7 defining the CSBK opcode set and payload layouts.
