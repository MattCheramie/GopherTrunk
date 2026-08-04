---
slug: tetra-cmce-mle-pdu
title: TETRA CMCE & MLE PDUs
entry_type: term
category: trunked-radio
description: "The TETRA CMCE (Circuit Mode Control Entity) PDUs — D-SETUP, D-CONNECT, D-RELEASE, D-TX-GRANTED and more — carry call control on the control channel, and the MLE SYSINFO broadcast carries the network identity; a 4-bit discriminator selects the sub-protocol."
keywords: TETRA CMCE, MLE PDU, D-SETUP, D-CONNECT, D-RELEASE, D-TX-GRANTED, discriminator, protocol identifier, SYSINFO, voice grant, EN 300 392-2 14
aka: [CMCE, "circuit mode control entity", "MLE PDU"]
autolink: true
infobox:
  - { label: Discriminator, value: 4 bits (MLE / MM / CMCE / SDS) }
  - { label: Key CMCE PDUs, value: "D-SETUP, D-CONNECT, D-RELEASE, D-TX-GRANTED" }
  - { label: Carries, value: "Call parties, emergency flag, grant" }
  - { label: Spec, value: "ETSI EN 300 392-2 §14 (CMCE) / §18 (MLE)" }
see_also: [tetra, tetra-mac-pdu, tetra-llc, channel-grant, group-call, private-call, emergency-call, control-channel, radio-id, talkgroup]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Protocol_data_unit
---

The **TETRA CMCE PDUs** are the circuit-mode call-control messages that a
[TETRA](/reference/tetra/) infrastructure sends on the [control channel](/reference/control-channel/)
to set up, connect and tear down voice calls.[^tetra][^pdu] They ride in the TL-SDU handed up
by the [LLC](/reference/tetra-llc/) layer, beneath a 3-bit MLE protocol discriminator. A
receiver following a system reads them to learn who is calling whom and — paired with the
[MAC](/reference/tetra-mac-pdu/) channel-allocation element — where the call's traffic is,
which together form a [voice grant](/reference/channel-grant/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A Layer-3 PDU opens with a 4-bit discriminator whose top two bits are the protocol identifier selecting MLE, MM, CMCE or SDS, followed by the PDU type; a CMCE D-CONNECT carries the call identifier and parties, and the MAC channel-allocation element in the same slot supplies the carrier and timeslot, together forming a voice grant." xmlns="http://www.w3.org/2000/svg">
  <rect x="14" y="34" width="60" height="26" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="44" y="47" text-anchor="middle" font-size="7" fill="currentColor">disc 4</text>
  <text x="44" y="56" text-anchor="middle" font-size="6.5" fill="currentColor">= CMCE</text>
  <rect x="74" y="34" width="52" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="100" y="50" text-anchor="middle" font-size="7" fill="currentColor">type</text>
  <rect x="126" y="34" width="130" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="191" y="50" text-anchor="middle" font-size="7.5" fill="currentColor">call id · source · dest</text>
  <text x="285" y="41" font-size="7.5" fill="currentColor">+ MAC chan-alloc</text>
  <rect x="285" y="46" width="120" height="16" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="345" y="57" text-anchor="middle" font-size="7" fill="currentColor">carrier · timeslot</text>
  <path d="M191 60 L191 82 L300 82" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <path d="M345 62 L345 82" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <rect x="150" y="86" width="180" height="22" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="240" y="101" text-anchor="middle" font-size="8" fill="currentColor">voice grant → retune to traffic slot</text>
  <text x="235" y="128" text-anchor="middle" font-size="7.5" fill="currentColor">disc top 2 bits = protocol id: 00 MLE · 01 MM · 10 CMCE · 11 SDS</text>
</svg>
<figcaption>A Layer-3 PDU opens with a 4-bit discriminator (its top two bits the protocol identifier); a CMCE D-CONNECT names the call's parties, and the MAC channel-allocation element from the same slot supplies the carrier and timeslot — together a voice grant.</figcaption>
</figure>

## The discriminator

Every Layer-3 PDU opens with a 4-bit **discriminator** (§14.7). Its top two bits are the
**protocol identifier** that selects the sub-protocol, and the low two bits are the upper
part of the PDU type within it:

| Discriminator | Protocol |
| --- | --- |
| `00xx` | MLE — Mobile Link Entity (system broadcast) |
| `01xx` | MM — Mobility Management |
| `10xx` | CMCE — Circuit Mode Control Entity (call control) |
| `11xx` | SDS — Short Data Service |

Only CMCE and MLE carry trunking-relevant messages a voice follower acts on; MM and SDS are
dropped by strict validation.

## The CMCE PDUs

| Type | PDU | Role |
| --- | --- | --- |
| 0x1 | D-SETUP | Incoming call setup — announces a call is forming |
| 0x2 | D-CONNECT | Call connected — carries the traffic grant |
| 0x4 | D-RELEASE | Call released — teardown |
| 0x5 | D-TX-CEASED | Talker stopped transmitting |
| 0x7 | D-TX-GRANTED | Late-grant transmission permission |
| 0x9 | D-INFO | Supplementary services |
| 0xA | D-CALL-PROCEEDING | Call proceeding |

**D-SETUP** and **D-CONNECT** are the pair that opens a call: the setup announces it and the
connect confirms it and carries the grant. A follower assembles the
[VoiceGrant](/reference/channel-grant/) field by field — the physical resource (carrier +
timeslot + [usage marker](/reference/tetra-aach/)) from the MAC allocation element, and the
call identifier, source and destination SSIs and emergency flag from the CMCE TM-SDU in the
same PDU. Whether the destination is a [talkgroup](/reference/talkgroup/) GSSI (a
[group call](/reference/group-call/)) or an individual ISSI (a
[private call](/reference/private-call/)) is classified from that destination address so the
UI does not surface a [radio ID](/reference/radio-id/) as a phantom talkgroup, and the
emergency bit marks an [emergency call](/reference/emergency-call/). **D-RELEASE** and
**D-TX-CEASED** drive teardown; the latter is treated as idle filler the state machine
absorbs.

## MLE SYSINFO

Under the MLE discriminator, PDU type 0x3 is **SYSINFO**, the network-broadcast that carries
the system identity: a 10-bit Mobile Country Code, a 14-bit Mobile Network Code, and a 14-bit
Location Area. The MCC and MNC uniquely tag a TETRA system, and GopherTrunk treats the first
SYSINFO as the trigger that locks the control channel, surfacing the identifier in the lock
state.

## Relevance to SDR

`internal/radio/tetra/cmce.go` defines the CMCE and MLE PDU types, the `VoiceGrant` carrier
the MAC path fills, and `AsSystemBroadcast` for the MLE SYSINFO; `pdu.go` defines the
`Discriminator` and the `PDU` shape. A real CMCE TM-SDU is bit-packed (a 3-bit MLE
discriminator plus a 5-bit PDU type), so grants are decoded bit-accurately off the MAC layer
rather than from the byte-aligned `ParsePDU` view, which cannot frame the sub-byte fields.
`IsKnown` and strict validation gate the state machine to the documented (discriminator,
type) pairs so a PDU whose type field lands in an unallocated range is dropped rather than
mis-dispatched.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA call-control signalling.
[^pdu]: [Protocol data unit](https://en.wikipedia.org/wiki/Protocol_data_unit) — Wikipedia, on the PDU concept the CMCE/MLE messages instantiate.
