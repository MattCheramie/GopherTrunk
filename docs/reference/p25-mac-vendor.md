---
slug: p25-mac-vendor
title: P25 vendor MAC opcodes
entry_type: term
category: trunked-radio
description: P25 Phase 2 reserves a manufacturer-specific MAC opcode range where Motorola and Harris carry their own messages — group-regroup/patch, patch-delete, and the FACCH-S talker alias (header 0x91 / data 0x95) — disambiguated by an MFID byte and reverse-engineered against SDRTrunk.
keywords: P25 vendor MAC opcode, manufacturer specific MAC, MFID, Motorola patch group, Harris regroup, FACCH-S talker alias, 0x91 0x95, TIA-102 vendor extension
aka: ["vendor MAC opcode", "manufacturer-specific MAC PDU", "Motorola MAC opcode", "Harris MAC opcode"]
autolink: true
infobox:
  - { label: Opcode range, value: 0x80–0xBF (MFID-tagged) }
  - { label: Motorola MFID, value: "0x90" }
  - { label: Harris MFID, value: "0xA4" }
  - { label: Alias opcodes, value: 0x91 header / 0x95 data }
see_also: [p25-mac-pdu, p25-phase-2, p25-sacch-facch, tsbk, cyclic-redundancy-check, channel-grant]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Protocol_data_unit
---

**Vendor MAC opcodes** are the manufacturer-specific [MAC PDUs](/reference/p25-mac-pdu/) that P25
Phase 2 systems emit outside the standard opcode set.[^wiki] The standard reserves an opcode range
(0x80–0xBF) in which each manufacturer defines its own messages, and a PDU with such an opcode carries
an **MFID** (Manufacturer ID) byte right after the opcode so the same numeric opcode can mean different
things per vendor.[^pdu] GopherTrunk decodes the two vendors that dominate deployed P25 —
**Motorola** (MFID `0x90`) and **Harris** (MFID `0xA4`) — dispatching on the `(MFID, Opcode)` pair.
Because TIA-102 vendor extensions are not in the project's spec PDFs, these layouts are a working model
reverse-engineered against SDRTrunk, and are confined to one file so a correction stays local.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A vendor MAC PDU: opcode 0x81 followed by MFID 0x90 decodes as a Motorola patch group, while the same opcode with MFID 0xA4 decodes as a Harris regroup; the talker alias uses a 0x91 header PDU followed by 0x95 data PDUs on FACCH-S during hangtime." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="50" height="24" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
  <text x="45" y="46" text-anchor="middle" font-size="8" fill="currentColor">0x81</text>
  <rect x="70" y="30" width="60" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="100" y="46" text-anchor="middle" font-size="8" fill="currentColor">MFID</text>
  <path d="M132 42 L160 30" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M132 42 L160 54" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="240" y="30" font-size="8" fill="currentColor">0x90 → Motorola patch group (super-group + members)</text>
  <text x="240" y="58" font-size="8" fill="currentColor">0xA4 → Harris regroup (regroup TG + target unit)</text>
  <text x="20" y="92" font-size="8" fill="currentColor">talker alias (Motorola 0x90):  HEADER 0x91  →  DATA 0x95 …  →  DATA 0x95</text>
  <text x="20" y="112" font-size="7.5" fill="currentColor">rides FACCH-S during hangtime · fragments reassemble to WACN|System|RID|cipher-alias|CRC-16</text>
</svg>
<figcaption>The MFID byte disambiguates a shared vendor opcode; the Motorola talker alias is a 0x91 header followed by 0x95 data fragments that reassemble into the same cipher-alias framing as Phase 1.</figcaption>
</figure>

## The vendor opcode set

| Opcode | MFID | Decodes as |
| --- | --- | --- |
| 0x81 | 0x90 (Motorola) | Group-regroup / patch group (super-group + up to 3 members) |
| 0x81 | 0xA4 (Harris) | Regroup (regroup talkgroup → target unit ID) |
| 0x82 | 0x90 / 0xA4 | Talker-alias (speculative plain-ASCII model — never matches real air) |
| 0x83 | 0x90 (Motorola) | Patch-delete (cancels a super-group established by 0x81) |
| 0x91 | 0x90 (Motorola) | Talker-alias HEADER (FACCH-S) |
| 0x95 | 0x90 (Motorola) | Talker-alias DATA (FACCH-S) |

The **patch / regroup** opcode (0x81) is the clearest example of MFID disambiguation: under Motorola it
aggregates member talkgroups under one super-group so a patched call is heard on every member (a
[channel-grant](/reference/channel-grant/)-adjacent construct), while under Harris the identical opcode
points a single regroup talkgroup at a target unit. `AsMotorolaPatchGroup` and `AsHarrisRegroup` each
guard on both the opcode and the MFID before unpacking, so a Harris regroup is never mis-parsed as a
Motorola patch.

## The FACCH-S talker alias

The most important vendor path is the Motorola talker alias — the radio's display name. An early working
model assumed a single plain-ASCII opcode (0x82), but that never matched on-air traffic. SDRTrunk ground
truth (from Victorian MMR) showed the real form rides on **FACCH-S during hangtime** as a **HEADER PDU
(opcode 0x91)** followed by one or more **DATA PDUs (opcode 0x95)**, both under MFID 0x90. The fragments
are not ASCII: they reassemble into the same Motorola message framing as Phase 1 —
`WACN | System | RadioID | cipher-alias | CRC-16` — so the alias is deciphered through the shared
Motorola alias cipher and validated by a trailing
[CRC-16](/reference/cyclic-redundancy-check/) rather than read literally. A subtlety that cost time: the
data fragment is *nibble*-aligned, with the first cipher nibble in the low nibble of a payload byte, so
reassembly concatenates a nibble stream, not whole bytes. An assembler collects the header and data
blocks for one call and emits the finished alias once the CRC validates. The 0x82 opcode is retained only
as a dead reference the dispatch still names; the real alias is the 0x91/0x95 path.

## Relevance to SDR

`internal/radio/p25/phase2/mac_vendor.go` holds the MFID constants, the vendor opcode enum, and the
`(MFID, Opcode)` accessors, with the alias reassembly in `talker_alias.go`. Decoding these is what lets
GopherTrunk report a patch super-group or a caller's alias on a Motorola or Harris Phase 2 system —
information the standard opcodes alone do not carry. All of it is a working model against SDRTrunk, not a
published spec, and is deliberately isolated so a future correction is one local edit.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 and its manufacturer extensions.
[^pdu]: [Protocol data unit](https://en.wikipedia.org/wiki/Protocol_data_unit) — Wikipedia, on the PDU as a self-contained protocol message.
