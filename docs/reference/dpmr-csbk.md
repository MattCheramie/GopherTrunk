---
slug: dpmr-csbk
title: dPMR CSBK
entry_type: term
category: trunked-radio
description: "The dPMR Mode-3 CSBK (Common Signalling Block) is the 80-bit control-channel unit that coordinates a trunked system — a 5-bit message type, 3-bit flags, 24-bit source and destination IDs, 8-bit service info, and a 16-bit opcode-specific field — carrying registration, voice/data grants, release, and idle."
keywords: dPMR CSBK, common signalling block, Mode 3, 80-bit, message type, source ID, destination ID, voice allocation, ETSI TS 102 658, trunking grant
aka: [CSBK, "common signalling block", "dPMR CSBK"]
autolink: true
infobox:
  - { label: Size, value: 80 bits (10 bytes) }
  - { label: Fields, value: "type · flags · src · dst · service · extra" }
  - { label: Mode, value: Mode 3 (trunking) }
  - { label: Spec, value: ETSI TS 102 658 §6.5 }
see_also: [csbk, dpmr, dpmr-frame-sync, dpmr-channel-coding, channel-grant, radio-id, talkgroup, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/DPMR
  - https://en.wikipedia.org/wiki/Trunked_radio_system
---

The **dPMR CSBK** (**Common Signalling Block**) is the 80-bit control-channel unit a
[dPMR](/reference/dpmr/) Mode-3 trunked system transmits between voice grants — the same role
the [CSBK](/reference/csbk/) plays in DMR.[^wiki] It is the message that registers radios,
grants voice and data calls onto traffic channels, and keeps the system coordinated. After
[FEC](/reference/dpmr-channel-coding/) removal the block resolves into six fixed fields: a
message type, a small flags field, a source and a destination address, service info, and a
16-bit opcode-specific tail.[^trunk]

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 130" role="img" aria-label="The 80-bit dPMR CSBK laid out left to right as six fields: a 5-bit message type, a 3-bit flags field, a 24-bit source ID, a 24-bit destination ID, an 8-bit service info field, and a 16-bit opcode-specific field." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor">
    <rect x="12" y="40" width="34" height="30" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
    <text x="29" y="55" text-anchor="middle">type</text>
    <text x="29" y="82" text-anchor="middle" font-size="6.5">5</text>
    <rect x="46" y="40" width="28" height="30" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1"/>
    <text x="60" y="55" text-anchor="middle">flags</text>
    <text x="60" y="82" text-anchor="middle" font-size="6.5">3</text>
    <rect x="74" y="40" width="120" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
    <text x="134" y="58" text-anchor="middle">Source ID</text>
    <text x="134" y="82" text-anchor="middle" font-size="6.5">24</text>
    <rect x="194" y="40" width="120" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1"/>
    <text x="254" y="58" text-anchor="middle">Destination ID</text>
    <text x="254" y="82" text-anchor="middle" font-size="6.5">24</text>
    <rect x="314" y="40" width="52" height="30" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
    <text x="340" y="58" text-anchor="middle">service</text>
    <text x="340" y="82" text-anchor="middle" font-size="6.5">8</text>
    <rect x="366" y="40" width="102" height="30" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="417" y="58" text-anchor="middle">opcode-specific</text>
    <text x="417" y="82" text-anchor="middle" font-size="6.5">16 (e.g. channel #)</text>
  </g>
  <text x="12" y="106" font-size="8" fill="currentColor">80 bits total · flags = group / emergency / encrypted</text>
</svg>
<figcaption>The dPMR CSBK packs six fields into 80 bits: the message type names the function, the flags mark group/emergency/encrypted calls, the source and destination carry 24-bit radio addresses, and the opcode-specific tail carries the granted channel number on a voice allocation.</figcaption>
</figure>

## Field layout

The 80 bits are read MSB-first as ten bytes with the fields packed across byte boundaries:

| Bits | Width | Field | Contents |
| --- | --- | --- | --- |
| 0–4 | 5 | Message type | opcode (see below) |
| 5–7 | 3 | Flags | group / emergency / encrypted |
| 8–31 | 24 | Source ID | calling subscriber [radio ID](/reference/radio-id/) |
| 32–55 | 24 | Destination ID | callee — group or subscriber |
| 56–63 | 8 | Service info | priority / payload type |
| 64–79 | 16 | Opcode-specific | channel number, status, … |

The three flag bits are `0x4` group call, `0x2` emergency, and `0x1` encrypted. The source and
destination are 24-bit addresses — the destination is a [talkgroup](/reference/talkgroup/) on a
group call and a subscriber on a unit-to-unit call, distinguished by the group flag. On a voice
allocation, the 16-bit opcode-specific field carries the physical channel number the radios
retune to, which the band-plan resolver turns into a frequency.

## Message types

The 5-bit message type opens every CSBK. GopherTrunk enumerates the subset the trunking state
machine acts on:

| Opcode | Name | Purpose |
| --- | --- | --- |
| `0x01` | RegistrationRequest | radio registers to the system |
| `0x02` | RegistrationResponse | system acknowledges registration |
| `0x03` | VoiceServiceAllocation | group voice channel grant |
| `0x04` | IndividualVoiceAllocation | unit-to-unit voice grant |
| `0x05` | DataServiceAllocation | data channel grant |
| `0x06` | ServiceRequest | subscriber requests a service |
| `0x07` | StandingServiceStatus | periodic site broadcast |
| `0x0F` | Release | tear down / release |
| `0x1F` | Idle | control-channel filler |

A VoiceServiceAllocation or IndividualVoiceAllocation is the
[channel grant](/reference/channel-grant/) the engine follows to a traffic channel;
StandingServiceStatus is the broadcast that lets the state machine declare the control channel
locked and learn the system identifier; Release and Idle are absorbed silently. GopherTrunk can
run in a strict mode that drops any CSBK whose 5-bit type falls outside this documented set, so
a corrupted block does not masquerade as a grant.

## Relevance to SDR

`internal/radio/dpmr/csbk.go` parses the 80 bits into a typed `CSBK` — `CSBKFromBits` packs the
bit slice, `ParseCSBK` unpacks the fields, and the `IsGroup` / `IsEmergency` / `IsEncrypted`
accessors read the flags — while `internal/radio/dpmr/opcodes.go` holds the `MessageType` enum
and the `AsVoiceGrant` / `AsSiteBroadcast` helpers that turn a CSBK into a structured grant or
site broadcast. Those grants are what the trunking engine emits on the event bus with
`Protocol = "dpmr"`, driving the "see grant → retune → follow" loop. Vendor extensions
repurpose the service-info and trailing fields, so the source flags that live captures should be
cross-checked against the deployment before those two fields are trusted; the message type,
addresses, and flags, however, are the stable core the state machine depends on.

## Sources

[^wiki]: [dPMR](https://en.wikipedia.org/wiki/DPMR) — Wikipedia, on the ETSI dPMR standard and its Mode 3 trunking signalling.
[^trunk]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on control-channel signalling and channel grants in trunked systems.
