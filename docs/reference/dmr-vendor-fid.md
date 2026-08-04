---
slug: dmr-vendor-fid
title: DMR vendor FID
entry_type: term
category: trunked-radio
description: The DMR Feature-set ID (FID) is a CSBK octet that tags a control block as standard ETSI or as a vendor extension — Capacity Plus, Connect Plus, or Hytera — so a decoder dispatches on FID before opcode and never misreads a proprietary block as a standard grant.
keywords: DMR FID, feature set ID, Capacity Plus, Connect Plus, Hytera XPT, MotoTRBO, vendor CSBK, FID dispatch
aka: ["FID", "feature-set ID", "vendor FID"]
autolink: true
infobox:
  - { label: Field, value: CSBK octet 1 (FID) }
  - { label: Standard, value: "0x00 (ETSI)" }
  - { label: Motorola, value: "0x10 Cap+, 0x06 Connect Plus" }
  - { label: Hytera, value: "0x08 / 0x68 (XPT)" }
see_also: [capacity-plus, connect-plus, csbk, dmr-tier-3, dmr-csbk-payloads, rest-channel, multisite-trunking]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/
---

The **DMR Feature-set ID** (**FID**) is the second octet of a [CSBK](/reference/csbk/), and it
tags the block as either standard ETSI signalling or a vendor extension.[^wiki] ETSI TS 102 361-4
reserves FID `0x00` for standard trunking; the major MotoTRBO and Hytera trunking products tag
their proprietary CSBKs with a vendor FID.[^etsi] A decoder must read the FID **before** the
opcode, because a vendor block's 6-bit opcode can collide with a standard one — dispatching on
opcode first would misread a proprietary block as, say, a standard TalkGroup Voice Channel Grant
and emit a bogus grant.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A CSBK's feature-set ID octet routing to one of four handlers: standard ETSI, Motorola Capacity Plus, Motorola Connect Plus, or Hytera XPT; only the standard and Capacity Plus paths parse grants, while Connect Plus and Hytera are recognised and logged but not force-parsed." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="46" width="90" height="28" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="61" y="64" text-anchor="middle" font-size="8.5" fill="currentColor">FID octet</text>
  <g stroke="currentColor" stroke-width="1.1" font-size="7.5" fill="currentColor" text-anchor="middle">
    <path d="M106 60 L128 22" fill="none"/><rect x="128" y="10" width="150" height="24" fill="currentColor" fill-opacity="0.14"/><text x="203" y="25">0x00 · ETSI standard</text>
    <path d="M106 60 L128 48" fill="none"/><rect x="128" y="42" width="150" height="24" fill="currentColor" fill-opacity="0.14"/><text x="203" y="57">0x10 · Capacity Plus/Max</text>
    <path d="M106 60 L128 78" fill="none"/><rect x="128" y="74" width="150" height="24" fill="none"/><text x="203" y="89">0x06 · Connect Plus</text>
    <path d="M106 60 L128 108" fill="none"/><rect x="128" y="102" width="150" height="16" fill="none"/><text x="203" y="114">0x08 / 0x68 · Hytera XPT</text>
  </g>
  <text x="360" y="30" font-size="7.5" fill="currentColor">grants parsed</text>
  <text x="360" y="90" font-size="7.5" fill="currentColor">logged only</text>
</svg>
<figcaption>The FID octet routes a CSBK to its vendor handler before the opcode is read; standard and Capacity Plus blocks are parsed, while Connect Plus and Hytera blocks are recognised and logged pending capture validation.</figcaption>
</figure>

## The recognised FIDs

GopherTrunk maps four vendor FIDs onto three trunking feature sets, with everything else treated
as standard ETSI:

| FID | Vendor | Trunking product |
|-----|--------|------------------|
| 0x00 | ETSI standard | Dedicated-control Tier III |
| 0x10 | Motorola | Capacity Plus / Capacity Max |
| 0x06 | Motorola | Connect Plus |
| 0x08 | Hytera | XPT |
| 0x68 | Hytera | XPT (alternate feature-set tag) |

## How each vendor is handled

The handling deliberately differs by how much of each vendor's signalling has been validated
against real signals. **Capacity Plus / Capacity Max** (FID `0x10`) carry their voice grants in the
same ETSI-shaped 8-octet payload as standard trunking, so those blocks decode straight through the
standard `ParseTVGrant` / `ParsePVGrant` parsers and publish normally; the
[Capacity Plus](/reference/capacity-plus/) rest channel is tracked from its system-info CSBK, whose
site-ID field doubles as a pointer to the current [rest channel](/reference/rest-channel/)'s LCN.

**Connect Plus** (FID `0x06`) and **Hytera XPT** (FID `0x08` / `0x68`) use materially different
proprietary signalling. Their CSBKs are recognised and logged at debug level, but they are **not**
force-parsed as standard grants — doing so would emit garbage, since their payload layouts are not
the ETSI ones. On-air capture validation of the [Connect Plus](/reference/connect-plus/) and XPT
payload formats is the remaining follow-up; until then, the honest posture is to surface that a
recognised vendor block arrived without pretending to understand its contents. Publishing a vendor
grant does not change the voice path — the protocol stays "dmr-tier3" because vendor trunking
alters the control layer, not the voice codec, so the recorder and vocoder are unaffected.

## Why order matters

The single reason FID dispatch comes first is opcode collision. The 6-bit CSBKO space is small and
vendors reuse values freely, so a Connect Plus block with opcode `0x30` is not a Private Voice
Channel Grant even though the standard set assigns that value to one. Reading the FID first routes
the block to the right handler before any opcode-based parsing runs; only after a block is confirmed
to carry FID `0x00` (or a vendor whose grants are known to match the ETSI layout) is the standard
opcode table consulted. This ordering is what keeps a mixed-vendor RF environment from producing
phantom grants.

## Relevance to SDR

`internal/radio/dmr/tier3/vendor.go` defines the FID constants, `VendorFromFID` mapping, and
`handleVendorCSBK`, which switches on the recognised vendor: Motorola grants flow through the shared
grant parsers and the Capacity Plus rest channel is latched from the broadcast/Aloha system-info,
while Connect Plus and Hytera blocks are logged as recognised-but-unparsed. Dispatching on FID
before opcode is the guard that lets GopherTrunk sit on a real trunked system carrying several
vendors' traffic and only act on the blocks it can decode correctly.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR trunking and the MotoTRBO and Hytera product families.
[^etsi]: [ETSI TS 102 361-4 (DMR Tier III)](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/) — ETSI, which reserves FID 0x00 for standard trunking and leaves other values to vendors.
