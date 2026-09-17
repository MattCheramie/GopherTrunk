---
title: "DMR End to End, Part 8: Tier III Trunking — C_ALOHA, Grants & LCNs"
description: How GopherTrunk reads a DMR Tier III control channel — the 96-bit CSBK and the 0x5A5A CRC mask that once rejected every real one, locking on C_ALOHA, topology from C_BCAST, voice grants that carry a 12-bit LPCN instead of a frequency, vendor feature IDs, multi-block control, and the −56 dBFS WARN that contradicted live decodes.
category: deep-dives
keywords: dmr tier iii decoder, dmr csbk opcodes, c_aloha lock, dmr tv_grant lpcn, dmr lcn band plan, dmr band plan resolver, capacity plus fid, dmr multi block control, dmr c_bcast adjacent site, gophertrunk dmr tier 3
tags: [dmr-end-to-end, dmr, tier-3, trunking, csbk, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 8
---

*Part 8 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }})
met the CSBK as a keep-alive on a conventional repeater. This part gives it
a control channel: DMR Tier III, where the same 96-bit block advertises the
site, describes its neighbours, and hands out voice calls — by logical
channel number, never by frequency.*

> **TL;DR:** `tier3.ControlChannel` (`internal/radio/dmr/tier3/`) reads
> CSBKs off a Tier III control channel: BPTC(196,96) → 12 bytes → `ParseCSBK`
> (LB, PF, 6-bit CSBKO, FID, 8-octet payload, CRC-CCITT XOR **`0x5A5A`** — the
> earlier "init 0xFFFF, complement" convention rejected every real CSBK).
> **C_ALOHA** (0x19) locks the channel on its raw `SystemID`
> (`dmr cc locked`); **C_BCAST** (0x28) sub-types fill the topology;
> **TV_GRANT / PV_GRANT** (0x31 / 0x30) lead with a **12-bit LPCN + timeslot
> bit** — #639 read the LCN from the source address's low byte, so it changed
> with every radio. An LCN resolves through `dmr_band_plan` (`LinearBandPlan`
> / `TableBandPlan`) or the learned plan of
> [The Hunt Part 8]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }});
> unresolved grants publish `decode.error stage=no-bandplan`. Vendor FIDs
> dispatch before opcodes; MBC assembles under LB=1; and the wideband
> low-power WARN no longer fires against a CC decoding every C_ALOHA at
> −56 dBFS (`lowPowerDecodeGrace`).

**Key takeaways**

- **Tier III is the same bursts with a CSBK reader on top.** Sync, slot type
  and BPTC are shared with Tier II.
- **A CRC convention is a claim about the air, and only air can check it.**
  Real off-air vectors from two live TSCCs pin the `0x5A5A` mask; round-trips
  passed the wrong one for as long as they were the only test.
- **A DMR grant names a channel number, not a frequency.** The 12-bit LPCN
  leads the payload; the band plan — configured or learned — turns it into Hz.
- **Never gate a health WARN on absolute dBFS when decode evidence exists.**
  A control channel decoding every beacon is healthy at any power reading.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| CSBK parse + pin | CRC ^ `0x5A5A`, pinned by off-air blocks | `tier3/csbk.go` (`ParseCSBK`), `csbk_realvectors_test.go` |
| Lock + topology | C_ALOHA → `maybeLock`; C_BCAST → `topologyModel` | `control.go` (`handleCSBK`, `handleBroadcast`), `topology.go` |
| Grants | LPCN + TS + 24-bit target + 24-bit source | `payloads.go` (`ParseTVGrant`, `ParsePVGrant`) |
| LCN → Hz | linear base/spacing/offset or table; hot-swappable | `bandplan.go` (`Resolver`, `LinearBandPlan`), `SetResolver` |
| Vendor + MBC | FID before opcode; header + continuations under LB=1 | `vendor.go` (`VendorFromFID`), `mbc.go` (`handleMBC`) |
| Low-power WARN gate | decode within 15 s ⇒ healthy at any dBFS | `widebandt2/engine.go` (`lowPowerDecodeGrace`) |

## In this post

- **Same bursts, a different reader** — where Tier III forks from Tier II.
- **The CSBK and its CRC** — the mask that rejected every real block.
- **Lock on C_ALOHA, learn from C_BCAST** — identity and topology.
- **Grants carry an LCN** — the LPCN layout, #639, the band plan.
- **Vendor FIDs and MBC** — dispatch order and honest limits.
- **The −56 dBFS lesson** — decode evidence beats a power gauge.

## Same bursts, a different reader

[Protocol Decoders Part 5]({{ '/blog/deep-dives/protocol-decoders-05-dmr-tier-2-3/' | relative_url }})
made the point: the tier is a policy, not a wire format. `tier3.Process` is
the Tier II adapter's twin — buffer dibits, nine-pattern `SyncDetector`,
132-dibit slices, Hamming(20,8) slot type — except that it tries **both**
discriminator polarities on every burst and never locks one, letting the
CSBK CRC drop the wrong image. `IngestBurst` routes on data type:

```go
// internal/radio/dmr/tier3/control.go (shape)
func (c *ControlChannel) IngestBurst(b *dmr.Burst, slot dmr.SlotType) {
    switch slot.DataType {
    case dmr.DTCSBK:
        info, ok := c.decodeInfoBlock(b)          // BPTC(196,96) → 12 bytes
        csbk, err := ParseCSBK(info)               // CRC ^ 0x5A5A
        if err != nil { c.log.Debug("dmr/tier3: CSBK CRC failed"); return }
        c.csbkDecoded.Add(1); c.noteActivity()
        c.handleCSBK(slot.ColorCode, csbk)
    case dmr.DTMBCHeader, dmr.DTMBCContinuation:
        c.handleMBC(slot.ColorCode, slot.DataType, b)
    }
}
```

Two counters fall out of that switch: `csbkDecoded` is the `DecodedFrames`
figure the wideband engine polls, and `noteActivity` stamps the heartbeat the
`ccdecoder` drought watchdog compares.

## The CSBK, and the CRC that rejected every real one

A [CSBK]({{ '/reference/csbk/' | relative_url }}) is 96 information bits:

```go
// internal/radio/dmr/tier3/csbk.go (shape)
//	bit 0 LB · bit 1 PF · bits 2-7 CSBKO · bits 8-15 FID (0x00 = ETSI)
//	bits 16-79 payload · bits 80-95 CRC-CCITT(init 0x0000) of bits 0-79 ^ 0x5A5A
const (
    OpAloha    CSBKOpcode = 0x19 // C_ALOHA — the TSCC beacon the CC locks on
    OpBcast    CSBKOpcode = 0x28 // C_BCAST — Gen_Site, CallTimer, Adjacent_Site …
    OpPVGrant  CSBKOpcode = 0x30 // Private Voice Channel Grant
    OpTVGrant  CSBKOpcode = 0x31 // TalkGroup Voice Channel Grant
    OpBTVGrant CSBKOpcode = 0x32 // Broadcast TalkGroup Voice Grant
    OpPDGrant  CSBKOpcode = 0x33 // Private Data Grant (observed, not followed)
    OpTDGrant  CSBKOpcode = 0x34 // TalkGroup Data Grant
)
const csbkCRCMask uint16 = 0x5A5A // ETSI TS 102 361-1 §B.3.11, Table B.21
```

The [CRC reference]({{ '/reference/dmr-csbk-crc/' | relative_url }}) describes
the mask; the code comment describes the scar. The first convention — init
`0xFFFF`, store the complement — **rejected every real CSBK while passing
every synthesized round-trip**, so the control channel never locked on air.
`TestParseCSBKRealOffAirVectors` is the counter-measure: 12-byte blocks from
2 MS/s captures of two live TSCCs (440.5625 and 440.2625 MHz), zero BPTC
corrections, repeating identically. They validate under `0x5A5A` and nothing
else — and pinned the opcode table too, where `0x28` had been mislabelled
"Preamble" and is C_BCAST (dsd-neo agrees). That is
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
in one test file: a spec plus your own inverse proves nothing about the air.

## Lock on C_ALOHA, learn from C_BCAST

`handleCSBK` dispatches standard-FID blocks by opcode. **C_ALOHA** is the
TSCC beacon; `ParseAloha` reads the raw 16-bit `SystemID` from payload
octets 2–3 and `maybeLock` publishes `cc.locked`:

```
INF dmr cc locked freq=440262500 cc=1 sysid=18689
DBG dmr/tier3: csbk opcode=C_ALOHA fid=0 cc=1 repeats_suppressed=163
```

That second line is parked on purpose. An idle TSCC repeats the same Aloha
~16 times a second — the field report counted **2177 C_ALOHA lines in two
minutes**. A change of `(opcode, FID, cc)` logs immediately; an unchanged
repeat summarises every `csbkRepeatLogInterval` (10 s) with the suppressed
count.

**C_BCAST** carries sub-types in a 5-bit `anncd_type` at the top of the
first payload octet, validated on real bursts (Gen_Site_Params, type 7,
decodes from `payload[0]=0x3A`; CallTimer_Parms, type 1, from `0x0F`).
`handleBroadcast` folds two into the topology model
[Trunking Engine Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
consumes: **Gen_Site_Params** gives the camped site's identity, RFSS and site
number; **Adjacent_Site** (type 6) adds a
[neighbour]({{ '/reference/neighbor-site/' | relative_url }}) with system ID,
site ID, CC LCN and colour code. Locking stays Aloha-driven — the two identity
fields differ. A third sub-type is handled by *not* handling it: **Announce
Channel-Frequency** (type 5) is the LCN↔frequency relationship, but its layout
has never been validated on a real burst, so the raw payload is logged once
(`dmr/tier3: announce channel-frequency (raw, layout unvalidated)`) and
nothing is guessed from it.

## Grants carry an LCN, not a frequency

The payload that matters most, and the bug it replaced:

```go
// internal/radio/dmr/tier3/payloads.go (shape)
//	payload bits 0-11 LPCN (12-bit) · bit 12 Timeslot (0 = TS1, 1 = TS2)
//	octets 2-4 target / group · octets 5-7 source subscriber
func ParseTVGrant(p [8]byte) TVGrant {
    return TVGrant{
        LCN:          uint16(p[0])<<4 | uint16(p[1])>>4,
        Timeslot:     (p[1] >> 3) & 0x01,
        GroupAddress: uint32(p[2])<<16 | uint32(p[3])<<8 | uint32(p[4]),
        SourceID:     uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7]),
    }
}
```

The channel-grant CSBKs **lead with the LPCN**, cross-checked against
dsd-neo's `dmr_csbk_parse.c`. The earlier layout read an "LCN" from octet 7 —
the low byte of the source address — so the decoded channel **changed with
every transmitting radio**
([#639](https://github.com/MattCheramie/GopherTrunk/issues/639), catalogued in
[grant-field gotchas]({{ '/reference/dmr-grant-field-gotchas/' | relative_url }})).
The timeslot bit is real here, unlike Tier II's synthetic token in
[Part 6]({{ '/blog/deep-dives/dmr-end-to-end-06-conventional-ipsc-two-calls/' | relative_url }}):
`publishGrant` maps it to `Grant.Timeslot = slot + 1`. PV_GRANT shares the
layout with `Individual = true`; data grants are observed, never followed. What a grant does *not* carry is service options: the ETSI
content has no emergency or privacy bits, so the voice chain backfills
`Encrypted`, `Emergency`, `Priority` and the source RID from the embedded LC
of [Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }}).

Then the LCN must become a frequency, and DMR never broadcasts the map:

```yaml
# config.example.yaml (shape)
- name: "Example-DMR-T3"
  protocol: dmr
  control_channels: [851_037_500]
  dmr_band_plan:
    linear: { base_hz: 851_012_500, spacing_hz: 25_000, offset: 1 }  # or a table:
```

`ResolverFromPlan` builds a `LinearBandPlan` (`BaseHz + (LCN − Offset) ×
SpacingHz`) or a `TableBandPlan`. `publishGrant` **first** publishes
`KindDMRGrantObserved` so the autoconfig learner sees every grant even with
no plan configured — the case it exists to fix; only then does `resolveLCN`
run. No resolver, or an
LCN outside the plan, drops the grant with `decode.error stage=no-bandplan`
(`dmr/tier3: grant dropped, no band-plan resolver configured` /
`dmr/tier3: band-plan miss`) — how a configuration gap shows in metrics. Omit
`dmr_band_plan` and the `dmrlcn.Learner` of
[The Hunt Part 8]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }})
correlates onsets with granted LCNs, confirms each by decoding a DMR sync,
fits a grid snapped to 6.25 / 12.5 / 25 kHz once four LCNs confirm, and
hot-swaps it in via `SetResolver`.
[Cookbook Part 2]({{ '/blog/tutorials/operator-cookbook-02-dmr-tier3/' | relative_url }})
walks both routes.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A Tier III CSBK flows from BPTC decode through the 0x5A5A CRC check into a dispatcher reading feature ID before opcode; C_ALOHA locks the channel, C_BCAST feeds the topology, and a TV_GRANT yields a 12-bit LPCN that is published for the LCN learner and resolved through the band plan into a frequency.">
  <rect x="14" y="30" width="118" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="73" y="46" text-anchor="middle" fill="currentColor" font-size="10">CSBK burst</text>
  <line x1="132" y1="50" x2="160" y2="50" stroke="currentColor"/><polygon points="160,46 168,50 160,54" fill="currentColor"/>
  <rect x="168" y="30" width="110" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="223" y="46" text-anchor="middle" fill="var(--accent)" font-size="10">CRC ^ 0x5A5A</text>
  <line x1="278" y1="50" x2="306" y2="50" stroke="currentColor"/><polygon points="306,46 314,50 306,54" fill="currentColor"/>
  <rect x="314" y="30" width="118" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="373" y="46" text-anchor="middle" fill="currentColor" font-size="10">FID, then opcode</text>
  <line x1="373" y1="70" x2="120" y2="120" stroke="var(--fg-muted)"/>
  <line x1="373" y1="70" x2="330" y2="120" stroke="var(--fg-muted)"/>
  <line x1="373" y1="70" x2="560" y2="120" stroke="var(--accent)"/>
  <rect x="50" y="120" width="140" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="120" y="136" text-anchor="middle" fill="currentColor" font-size="10">C_ALOHA 0x19</text>
  <text x="120" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">→ dmr cc locked</text>
  <rect x="260" y="120" width="140" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="330" y="136" text-anchor="middle" fill="currentColor" font-size="10">C_BCAST 0x28</text>
  <text x="330" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Gen_Site · Adjacent_Site</text>
  <rect x="490" y="120" width="140" height="40" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="560" y="136" text-anchor="middle" fill="var(--accent)" font-size="10">TV_GRANT 0x31</text>
  <text x="560" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">12-bit LPCN + TS bit</text>
  <line x1="560" y1="160" x2="560" y2="186" stroke="var(--accent)"/><polygon points="556,186 560,194 564,186" fill="var(--accent)"/>
  <rect x="470" y="194" width="180" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="560" y="210" text-anchor="middle" fill="var(--accent)" font-size="10">Resolver: LCN → Hz</text>
  <text x="560" y="224" text-anchor="middle" fill="var(--fg-muted)" font-size="8">linear / table / learned</text>
  <line x1="490" y1="150" x2="420" y2="214" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="330" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dmr.grant.observed → dmrlcn.Learner</text>
</svg>
<figcaption>One CSBK, three fates: C_ALOHA locks, C_BCAST maps the site, a grant yields an LPCN that only a band plan — configured or learned — can turn into a frequency.</figcaption>
</figure>

## Vendor FIDs and multi-block control

`handleCSBK` dispatches on **FID before opcode**, and the order is
load-bearing: a vendor block whose opcode happens to be `0x30` would
otherwise parse as a standard PV_GRANT and emit a bogus grant
([vendor FID reference]({{ '/reference/dmr-vendor-fid/' | relative_url }})).
Motorola **Capacity Plus / Max** (FID `0x10`) carries grants in the
ETSI-shaped payload, so those decode through the same parsers; its
system-info CSBK advertises the
[rest channel]({{ '/reference/rest-channel/' | relative_url }}) in the
`SiteID` octet. **Connect Plus** (`0x06`) and
**Hytera XPT** (`0x08` / `0x68`) are recognised and logged (`dmr/tier3:
vendor csbk recognised (payload decode pending capture validation)`), never
force-parsed.

[Multi-Block Control]({{ '/reference/dmr-mbc/' | relative_url }}) spreads a
CSBK-opcode message across a `DTMBCHeader` burst and `DTMBCContinuation`
bursts, the last flagged LB=1. `handleMBC` assembles per colour code — at
most `mbcMaxBlocks` (8), evicting a partial older than `mbcMaxAge` (1 s) —
and `dispatchMBC` structurally parses **only the grant opcodes**, whose
header-block octets 2–9 reuse the validated grant layout. Everything else is
assembled and logged (`mbc payload decode pending capture validation`): no
off-air MBC capture pins the last-block CRC convention, and the CSBK history
above is what an unverified convention costs.

## The −56 dBFS lesson

One more Tier III story, from the wideband engine
(`internal/scanner/widebandt2/`), which WARNs when a tap's IQ power sits
below `iqpower.LowPowerThresholdDbFS`. A reporter's Tier III control channel
drew that WARN **every 5 s at −56 dBFS while decoding every C_ALOHA** — an
absolute-dBFS gate contradicting live decode evidence:

```go
// internal/scanner/widebandt2/engine.go (shape)
// A channel with recent protocol decodes is healthy whatever its absolute
// dBFS says — never warn "outside the passband" against live decode evidence.
const lowPowerDecodeGrace = 15 * time.Second

decodingRecently := !ec.lastDecodeAt.IsZero() &&
    now.Sub(ec.lastDecodeAt) <= lowPowerDecodeGrace
if dbfs < iqpower.LowPowerThresholdDbFS && !decodingRecently { /* WARN */ }
```

`lastDecodeAt` advances whenever `DecodedFrames` — Tier III's `csbkDecoded`
— moves in a diagnostics window. `TestLowPowerDecodingControlChannelIsSilent`
drives a `dmr-tier3` channel whose counter climbs every window and asserts
zero WARNs (the DEBUG power line stays);
`TestLowPowerNonDecodingControlChannelStillWarns` pins that silence *without*
decodes still warns. It is the principle
[Weak-Signal Part 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
learned for MRC: absolute power is a gain-staging number; a gate built on it
re-fires on the next front end.

### How Tier III shaped the Go code

- **Real vectors sit beside the parser.** `csbk_realvectors_test.go` is the
  only test that could have caught the CRC convention; mask and opcodes are
  pinned to on-air bytes, not to `AssembleCSBK`.
- **Observation precedes resolution.** `KindDMRGrantObserved` is published
  before `resolveLCN`, so the learner works with no plan configured.
- **The resolver is an interface behind a lock.** `SetResolver` swaps it
  from another goroutine, so a learned plan lands without a restart.
- **Unvalidated layouts are logged raw.** Announce Channel-Frequency,
  Connect Plus, Hytera and non-grant MBC payloads surface as bytes.

## Where this goes next

Everything so far ran on a continuous repeater carrier.
[Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
goes to the other extreme: a simplex handheld transmitting one 27.5 ms burst
per 60 ms frame, and the gaps between bursts that blinded the receiver for
the whole life of #836.

## FAQ

**How does GopherTrunk lock onto a DMR Tier III control channel?**
On C_ALOHA, the periodic TSCC beacon (CSBKO 0x19). Once a CSBK clears
BPTC(196,96) and the 0x5A5A-masked CRC, `ParseAloha` reads its 16-bit system
ID and the control channel publishes `cc.locked` — logged as `dmr cc locked`
with frequency, colour code and system ID.

**Why does a DMR Tier III grant need a band plan?**
Because the grant carries a 12-bit logical channel number, not a frequency,
and the map is never broadcast. GopherTrunk resolves it through
`dmr_band_plan` (a linear grid or a table) or, if the plan is omitted, learns
it by correlating grants with carriers keying up.

**What was the DMR CSBK CRC bug?**
The first implementation used CRC-CCITT with init 0xFFFF and stored the
complement — self-consistent with its own encoder, so every synthetic test
passed while every real CSBK failed and the control channel never locked.
Off-air vectors from two live TSCCs pinned the correct convention: init
0x0000, XOR 0x5A5A.

**Does GopherTrunk decode Capacity Plus and Connect Plus?**
Capacity Plus / Max (FID 0x10) yes — its grants share the ETSI payload layout
and its system-info CSBK carries the rest channel. Connect Plus (0x06) and
Hytera XPT (0x08 / 0x68) are recognised and logged, not force-parsed, pending
on-air captures.

**Why did the low-power WARN fire on a working DMR control channel?**
It was gated on absolute dBFS: a Tier III CC at −56 dBFS decoding every
C_ALOHA read as "outside the passband". The engine now suppresses the
WARN for any channel whose decode counter advanced within 15 s
(`lowPowerDecodeGrace`); a channel that decodes nothing keeps it.

## Series navigation

**Part 8 of 14** · ←
[Part 7: The Idle Beacon & the CSBK Train]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }})
· Next →
[Part 9: Direct Mode — The Gaps That Blinded the Receiver]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
