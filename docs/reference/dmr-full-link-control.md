---
slug: dmr-full-link-control
title: DMR Full Link Control
entry_type: term
category: trunked-radio
description: The DMR Full Link Control is the 72-bit PDU that identifies a voice call — its FLCO opcode, feature-set ID, service options, and 24-bit destination and source addresses — carried in the Voice LC Header, the Terminator with LC, and the embedded signalling of a superframe.
keywords: DMR Full Link Control, FLC, FLCO opcode, group voice channel user, unit to unit, talker alias, GPS info, terminator, ETSI TS 102 361-2 Table 7.1
aka: ["FLC", "Full Link Control", "FLCO"]
autolink: true
infobox:
  - { label: Length, value: 72 bits (9 octets) }
  - { label: Opcode, value: 6-bit FLCO }
  - { label: Carries, value: "FID, service opts, dst + src (24-bit)" }
  - { label: Spec, value: ETSI TS 102 361-2 §7.1 Table 7.1 }
see_also: [dmr-rs-12-9, dmr-embedded-lc, dmr-slot-type, dmr-voice-superframe, talkgroup, radio-id, channel-grant, dmr-vendor-fid, dmr, dmr-encryption]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
---

The **DMR Full Link Control** (**FLC**) is the 72-bit PDU that says who is talking to whom on a
DMR voice call.[^wiki] It appears in three places: the **Voice LC Header** ([Data
Type](/reference/dmr-slot-type/) `0x1`) that opens a call, the **Terminator with LC** (`0x2`) that
closes it, and — spread across four fragments — the [embedded LC](/reference/dmr-embedded-lc/) of a
[voice superframe](/reference/dmr-voice-superframe/). Its leading 6 bits are the **FLCO**
(Full Link Control Opcode) naming the block type; the remaining octets carry a feature-set ID,
service options, and the 24-bit destination and source addresses that identify the
[talkgroup](/reference/talkgroup/) and [radio](/reference/radio-id/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 120" role="img" aria-label="The nine octets of a DMR Full Link Control laid out left to right: a byte holding a protect flag and the six-bit FLCO opcode, a feature-set ID byte, a service-options byte, three bytes of destination address, and three bytes of source address." xmlns="http://www.w3.org/2000/svg">
  <rect x="12" y="34" width="56" height="30" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1"/>
  <text x="40" y="49" text-anchor="middle" font-size="7" fill="currentColor">PF·FLCO</text>
  <text x="40" y="60" text-anchor="middle" font-size="6.5" fill="currentColor">oct 0</text>
  <rect x="68" y="34" width="46" height="30" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/>
  <text x="91" y="49" text-anchor="middle" font-size="7" fill="currentColor">FID</text>
  <text x="91" y="60" text-anchor="middle" font-size="6.5" fill="currentColor">oct 1</text>
  <rect x="114" y="34" width="46" height="30" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1"/>
  <text x="137" y="49" text-anchor="middle" font-size="6.5" fill="currentColor">svc opt</text>
  <text x="137" y="60" text-anchor="middle" font-size="6.5" fill="currentColor">oct 2</text>
  <rect x="160" y="34" width="120" height="30" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="220" y="49" text-anchor="middle" font-size="7.5" fill="currentColor">destination · 24-bit</text>
  <text x="220" y="60" text-anchor="middle" font-size="6.5" fill="currentColor">oct 3–5</text>
  <rect x="280" y="34" width="120" height="30" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1"/>
  <text x="340" y="49" text-anchor="middle" font-size="7.5" fill="currentColor">source · 24-bit</text>
  <text x="340" y="60" text-anchor="middle" font-size="6.5" fill="currentColor">oct 6–8</text>
  <text x="12" y="88" font-size="8" fill="currentColor">72 info bits + 24-bit RS(12,9) parity trailer (12 octets total)</text>
</svg>
<figcaption>Nine FLC octets: the FLCO opcode, feature-set ID, service options, and 24-bit destination and source addresses, followed on the wire by an RS(12,9) parity trailer.</figcaption>
</figure>

## The FLCO opcode table

GopherTrunk parses the FLC in `internal/radio/dmr/flc.go`. Octet 0 holds a 1-bit protect flag
(PF), a reserved bit, and the 6-bit FLCO; the common voice-call opcodes per ETSI TS 102 361-2
§7.1.1 Table 7.1 are:

| FLCO | Opcode | Meaning |
|---|---|---|
| `0x00` | Group Voice Channel User | group call — destination is a [talkgroup](/reference/talkgroup/) |
| `0x03` | Unit-to-Unit Voice Channel User | private call — destination is a subscriber |
| `0x04` | Talker Alias header | first block of the alias text |
| `0x08` | GPS Info | source location |
| `0x30` | Terminator | end-of-transmission link control |

Vendor extensions live behind a non-zero feature-set ID (FID — see
[vendor FID](/reference/dmr-vendor-fid/)); the opcodes above are the manufacturer-independent
set (FID 0). Octet 2, the service options, carries the emergency (bit 7) and
[encryption](/reference/dmr-encryption/)/privacy (bit 6) flags that
`FLC.AsGroupVoiceUser` surfaces on a decoded group call.

## Structure and FEC

The 9 FLC octets are the data portion of a 12-octet [RS(12,9,4)](/reference/dmr-rs-12-9/)
[Reed–Solomon](/reference/reed-solomon-code/) frame; the trailing 3 octets are parity, XOR-masked
with a context-specific seed. `ParseFLC` reads the leading 9 octets a
[BPTC(196,96)](/reference/bptc/) or [embedded-LC](/reference/dmr-embedded-lc/) decode reconstructs.
In the current pass GopherTrunk relies on the BPTC / embedded-BPTC layer for error correction over
the same bits and treats RS(12,9) verification as a separate, self-contained follow-up
(`dmr-rs-12-9`), so `ParseFLC` decodes the FLC without failing it on the RS trailer.

## Relevance to SDR

The FLC is where a DMR voice decode turns into a usable scanner event. From one recovered FLC,
GopherTrunk learns the call type (group vs unit-to-unit from the FLCO), the talkgroup or
destination subscriber, the source radio ID, and whether the traffic is encrypted or an emergency —
everything a scanner needs to name and log the call. Because the same FLC arrives redundantly (the
Voice LC Header, the embedded LC every superframe, and the Terminator), a receiver has repeated
chances to recover it even if the opening header is lost mid-tune, which is what lets GopherTrunk
label a call it joined late.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on the DMR standard and its link-control signalling.
[^rs]: [Reed–Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, on the code that protects the FLC's parity trailer.
