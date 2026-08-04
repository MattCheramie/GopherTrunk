---
slug: p25-packet-data-unit
title: P25 Packet Data Unit (PDU)
entry_type: term
category: trunked-radio
description: The P25 Packet Data Unit (PDU) is the data-channel frame — a header block plus data blocks carrying confirmed or unconfirmed user data, an SNDCP convergence layer, and commonly an IPv4 packet underneath.
keywords: P25 PDU, packet data unit, confirmed data, unconfirmed data, SNDCP, NSAPI, LLID, SAP, P25 data channel, IPv4 over P25, TIA-102.BAAA
aka: [PDU, "packet data unit"]
autolink: true
infobox:
  - { label: DUID, value: "0xC (PDU)" }
  - { label: Structure, value: Header block + N data blocks }
  - { label: Block size, value: 12 bytes post-FEC }
  - { label: Spec, value: TIA-102.BAAA / BAEA }
see_also: [p25-tsbk-opcodes, p25-nid-duid, tsbk, p25-logical-data-unit, radio-id, p25-phase-1, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **P25 Packet Data Unit** (**PDU**) is the data-channel counterpart of the voice
[LDU](/reference/p25-logical-data-unit/) and the trunking [TSBK](/reference/tsbk/): the frame
that carries user *data* — text messages, location reports, IP traffic — across a P25
system.[^wiki] It is identified by [DUID](/reference/p25-nid-duid/) `0xC` and is structured
as a **header block** followed by a run of **data blocks**, each a 12-byte (96-bit) post-FEC
unit. GopherTrunk decodes the structured PDU from already-FEC-decoded block bytes and
reassembles the data blocks into a payload, which is typically an SNDCP message wrapping a
network packet.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A P25 PDU as a header block followed by several data blocks, which reassemble into an SNDCP message that in turn encapsulates an IPv4 packet, showing the layered data stack." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="90" height="26" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="65" y="37" text-anchor="middle" font-size="8" fill="currentColor">header</text>
  <rect x="112" y="20" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="147" y="37" text-anchor="middle" font-size="8" fill="currentColor">block 1</text>
  <rect x="184" y="20" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="219" y="37" text-anchor="middle" font-size="8" fill="currentColor">block 2</text>
  <rect x="256" y="20" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="291" y="37" text-anchor="middle" font-size="8" fill="currentColor">block N</text>
  <path d="M112 46 L112 66 L326 66 L326 46" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <rect x="90" y="78" width="260" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="220" y="94" text-anchor="middle" font-size="8" fill="currentColor">SNDCP header + payload</text>
  <rect x="130" y="112" width="180" height="24" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="220" y="128" text-anchor="middle" font-size="8" fill="currentColor">IPv4 packet (src → dst)</text>
</svg>
<figcaption>Data blocks reassemble into an SNDCP message, which peels off to reveal the encapsulated network packet — commonly IPv4 — layering a data call from the air interface up to who is talking to whom.</figcaption>
</figure>

## Header and blocks

The PDU header names the format and the destination. GopherTrunk's `PDUHeader`
(`internal/radio/p25/phase1/pdu.go`) decodes:

| Field | Meaning |
|---|---|
| Format | `0x03` response, `0x15` unconfirmed data, `0x16` confirmed data |
| Confirmed | flag: does the receiver acknowledge each block? |
| SAP | Service Access Point — which upper-layer service (e.g. packet data) |
| MFID | manufacturer ID |
| LLID | 24-bit logical link (destination) ID — the target [radio](/reference/radio-id/) |
| Block count | number of data blocks that follow |

A **confirmed** PDU (`0x16`) carries per-block serial numbers and CRCs so the sender can be
told which blocks arrived and retransmit the rest; an **unconfirmed** PDU (`0x15`) is
fire-and-forget. `ReassemblePDU` concatenates the data blocks in order into the payload. The
block bit layout here is GopherTrunk's working model — TIA-102.BAAA's PDU tables are not in
the repository — so it is confined to one file with symmetric encoders and defensive parsers.

## SNDCP and IP transport

When the header's SAP marks the payload as packet data, the reassembled bytes begin with an
**SNDCP** (Sub-Network Dependent Convergence Protocol) header. `ParseSNDCP` peels off a
1-octet header that packs a 4-bit PDU type and a 4-bit **NSAPI** (Network Service Access
Point Identifier), then exposes the encapsulated network-layer packet. That packet is most
often IPv4: `ParseIPv4` decodes the standard 20-byte RFC 791 header to surface the source and
destination addresses and the protocol (ICMP/TCP/UDP) — the "who is talking to whom" of a
data call. Unlike the PDU and SNDCP layers, the IPv4 header is a fully-specified format, so
that layer is exact rather than a working model.

## Relevance to SDR

GopherTrunk decodes the PDU from FEC-decoded blocks and walks the stack — PDU header →
reassembled payload → SNDCP → IPv4 — to characterise packet-data activity on a system. The
receiver-level framing that produces those blocks from the raw dibit stream (the FSW → NID →
trellis-coded-block chain for DUID `0xC`) is a documented follow-up best built against real
packet-data captures; the structured decode above is what runs once the blocks exist. Because
much P25 data is short and bursty, and confirmed PDUs are acknowledged, a monitor can often
recover useful metadata (destination LLID, IP endpoints) even without reconstructing the full
application payload.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its data services. PDU/SNDCP layouts follow TIA-102.BAAA / BAEA as a working model; the IPv4 header follows RFC 791.
