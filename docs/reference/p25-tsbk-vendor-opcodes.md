---
slug: p25-tsbk-vendor-opcodes
title: P25 TSBK vendor opcodes
entry_type: term
category: trunked-radio
description: P25 vendor TSBK opcodes are manufacturer-specific control-channel messages — Motorola (MFID 0x90) patch/regroup and Harris (MFID 0xA4) regroup — that reuse the 6-bit opcode space under a vendor manufacturer ID.
keywords: P25 vendor TSBK, MFID, manufacturer ID, Motorola 0x90, Harris 0xA4, patch group, regroup, super-group, talker alias, P25 vendor opcode
aka: [MFID, "manufacturer TSBK", "vendor opcode"]
autolink: true
infobox:
  - { label: Selector, value: TSBK MFID byte }
  - { label: Motorola, value: "MFID 0x90" }
  - { label: Harris, value: "MFID 0xA4" }
  - { label: Status, value: Working model (partly RE) }
see_also: [p25-tsbk-opcodes, patch-multigroup, motorola-type-ii, tsbk, channel-grant, talkgroup, radio-id, p25-talker-alias, control-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

**P25 vendor TSBK opcodes** are manufacturer-specific control-channel messages that live in
the same 6-bit opcode space as the standard [OSP opcodes](/reference/p25-tsbk-opcodes/) but
are selected by a non-standard **MFID** (manufacturer ID) byte.[^wiki] The same opcode value
decodes differently depending on the MFID: under MFID `0x90` (Motorola) opcode `0x02` is a
patch-group voice grant, while under the standard MFID it is a plain group-voice update. The
two vendors GopherTrunk decodes are Motorola (`0x90`) and L3Harris (`0xA4`); both are common
on public-safety systems built on [Motorola](/reference/motorola-type-ii/) or Harris
infrastructure.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="A TSBK MFID byte routing decode down one of three branches: standard OSP table, Motorola vendor table, or Harris vendor table, with the same opcode value meaning different things on each branch." xmlns="http://www.w3.org/2000/svg">
  <rect x="185" y="16" width="100" height="26" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="235" y="33" text-anchor="middle" font-size="8.5" fill="currentColor">MFID byte</text>
  <path d="M235 42 L90 74 M235 42 L235 74 M235 42 L390 74" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="30" y="74" width="120" height="30" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="90" y="90" text-anchor="middle" font-size="8" fill="currentColor">0x00/0x01</text>
  <text x="90" y="100" text-anchor="middle" font-size="7" fill="currentColor">standard OSP</text>
  <rect x="175" y="74" width="120" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="235" y="90" text-anchor="middle" font-size="8" fill="currentColor">0x90 Motorola</text>
  <text x="235" y="100" text-anchor="middle" font-size="7" fill="currentColor">patch / regroup</text>
  <rect x="320" y="74" width="120" height="30" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="380" y="90" text-anchor="middle" font-size="8" fill="currentColor">0xA4 Harris</text>
  <text x="380" y="100" text-anchor="middle" font-size="7" fill="currentColor">dynamic regroup</text>
  <text x="235" y="126" text-anchor="middle" font-size="7.5" fill="currentColor">the same 6-bit opcode is read against the branch the MFID selects</text>
</svg>
<figcaption>The MFID byte routes a block into the standard or a vendor opcode table; GopherTrunk dispatches on MFID before opcode, so a vendor block is never misread as a standard one.</figcaption>
</figure>

## MFID dispatch

Every TSBK begins with an MFID byte that GopherTrunk parses before the opcode. `IsVendorMFID`
reports whether it is a recognised vendor (`0x90` or `0xA4`); the vendor accessors
(`AsMotorolaPatchGroup`, `AsHarrisRegroup`, …) each re-check the MFID and opcode before
decoding, so a standard block can never fall through a vendor path. This keeps the vendor
layouts — which are the project's working model rather than published spec — confined to one
file, `internal/radio/p25/phase1/tsbk_vendor.go`, where a correction stays local.

## Notable vendor opcodes

| MFID | Opcode | Name | Carries |
|---|---|---|---|
| `0x90` | `0x00` | Motorola Patch Group Add | Super-group address + up to 3 member talkgroups |
| `0x90` | `0x01` | Motorola Patch Group Delete | Super-group address to cancel |
| `0x90` | `0x02` | Motorola Patch Group Channel Grant | Patch (super-group) voice grant; grant-shaped payload |
| `0x90` | `0x03` | Motorola Patch Group Grant Update | Two (channel, super-group) activity pairs |
| `0x90` | `0x05` | Motorola Traffic Channel ID | Raw payload only — not field-decoded |
| `0x90` | `0x09` | Motorola System Loading | Raw payload only — not field-decoded |
| `0x90`/`0xA4` | `0x15` | Vendor Talker Alias | One fragment of a radio's display name |
| `0xA4` | `0x00` | Harris Dynamic Regroup | Regroup talkgroup + target unit |

The Motorola patch opcodes carry a [multigroup / patch](/reference/patch-multigroup/)
super-group — several member [talkgroups](/reference/talkgroup/) temporarily merged so they
share one call. The patch-grant opcodes reuse the standard group-voice grant and update
layouts exactly, differing only in that the group address is a super-group. The
[talker-alias](/reference/p25-talker-alias/) opcode `0x15` is MFID-agnostic, carrying one
block of a [radio's](/reference/radio-id/) human-readable name to be reassembled across
several TSBKs.

## Provenance

TIA-102 does not publish the vendor extensions, so these layouts are reverse-engineered
against the two mature open decoders, **OP25** and **SDRTrunk**, and cross-checked against
field captures where a ground-truth event is known (for example the Motorola patch-grant
layout was confirmed against MMR/CBD captures in issue #376). Two Motorola opcodes —
Traffic Channel ID (`0x05`) and System Loading (`0x09`) — are emitted continuously but have
no trustworthy field layout in either reference decoder, so GopherTrunk names them for the
probe log and captures the raw bytes but decodes no fields. GopherTrunk mirrors SDRTrunk's
per-manufacturer taxonomy (standard / Motorola / Harris / unknown) so an unrecognised vendor
block is reported with its MFID rather than silently misdecoded.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard. Vendor opcode layouts are reverse-engineered against OP25 and SDRTrunk and are the project's working model.
