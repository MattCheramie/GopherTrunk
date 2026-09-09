---
title: "P25 End to End, Part 3: TSBKs — The Control Channel's Workhorse"
description: Inside the P25 Trunking Signaling Block — 12 bytes through a 1/2-rate trellis and interleaver into 98 channel dibits, the CRC variant many references get wrong, the opcode families that run a trunked system, and the SCCB byte-offset bug a round-trip test blessed.
category: deep-dives
keywords: p25 tsbk decoding, trunking signaling block, p25 trellis code, p25 opcode list, group voice channel grant, p25 crc ccitt, p25 vendor mfid 0x90, sccb secondary control channel, gophertrunk p25
tags: [p25-end-to-end, p25, tsbk, trunking, fec, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 3
---

*Part 3 of **P25 End to End**, a 14-part deep dive that follows North America's
dominant trunking protocol through GopherTrunk — from a raw C4FM carrier to
recorded, named, multi-site voice.
[Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})
found the frames and defined what "locked" means. This part opens the frame
that earns the lock: the TSDU and its Trunking Signaling Blocks — the 12-byte
messages that grant calls, publish band plans, and describe whole systems.
It also tells two of this series' best cautionary tales: a CRC algorithm the
references document wrong, and a byte-offset bug that a passing round-trip
test protected for as long as nobody checked a literal vector.*

> **TL;DR:** A TSBK is **12 bytes** — LB/P flags, a 6-bit opcode, an MFID, 8
> payload bytes and a CRC-16 — trellis-encoded 1/2-rate into **98 channel
> dibits** and block-interleaved (`EncodeTSBKChannel` /
> `DecodeTSBKChannel`, `internal/radio/p25/phase1/tsbk.go`). One TSDU packs
> up to **three** blocks; decoding only the first dropped two-thirds of a
> busy site's signalling (issue #402). The trailer is the **augmented-
> codeword CRC-CCITT** (init 0, final XOR 0xFFFF), not the widely-documented
> CRC-CCITT/FALSE — the wrong variant failed 195 of 197 real TSBKs at trellis
> metric 0. Grants flow through the band plan into `trunking.Grant`; vendor
> opcodes (MFID 0x90) must never render through the standard
> `Opcode.String()` map; and the SCCB (0x39) parser once read channel B a
> byte early — with a green round-trip test — until literal vectors
> cross-checked against SDRTrunk caught it.

**Key takeaways**

- **The TSBK is the atom of P25 trunking.** Every grant, band-plan entry,
  neighbour announcement and registration reply on a Phase 1 control channel
  is one of these 12-byte blocks, arriving several per TSDU, many TSDUs per
  second.
- **The trellis is table-driven, not textbook.** P25's 1/2-rate code is a
  16-entry constellation table whose state is the previous input dibit
  (TIA-102.BAAA-A Annex A) — not the standard (7,5) convolutional code — and
  its Viterbi path metric doubles as a per-block quality gauge.
- **Two independent validators beat one strong one.** The trellis can decode
  garbage confidently; the CRC-16 behind it is what actually gates
  acceptance — and Part 2's marginal NID tier borrows it as a witness.
- **A round-trip test cannot see a layout bug.** The SCCB parser and
  assembler shared the same wrong byte offsets, so encode→decode was
  perfect while every real 0x39 produced a phantom secondary control
  channel. Pin parsers with literal vectors from an independent decoder.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Block parse + CRC | 12 bytes → LB/P/opcode/MFID/payload, augmented CRC | `internal/radio/p25/phase1/tsbk.go` (`ParseTSBK`) |
| Trellis code | 48 info dibits ↔ 98 channel dibits, Viterbi metric | `trellis.go` (`EncodeTrellis`, `DecodeTrellis`) |
| Interleave | block permutation over the 98 dibits | `interleaver.go` (`InterleaveTSBK`, `DeinterleaveTSBK`) |
| Soft-decision twin | per-bit LLRs through the same pipeline | `tsbk.go` (`DecodeTSBKChannelSoft`) |
| Opcode namespace | 6-bit OSP opcodes, TIA mnemonics | `opcodes.go` (`Opcode`, `Opcode.String`) |
| Vendor TSBKs | MFID 0x90/0xA4 patches, aliases, grants | `tsbk_vendor.go` (`MFIDMotorola`, `IsVendorMFID`) |
| Grant publish | band-plan lookup → `trunking.Grant` on the bus | `control.go` (`publishVoiceGrant`) |

## In this post

- **Twelve bytes that run a system** — the TSBK layout, and why three per TSDU matters.
- **Through the trellis** — the channel pipeline, and the CRC the references get wrong.
- **The opcode families** — grants, identifiers, status, unit business.
- **From opcode to `trunking.Grant`** — what leaves this layer, and where it goes.
- **Names that lie** — the vendor namespace and the AMBT labelling rule.
- **The SCCB bug** — the round-trip test that validated nothing.

## Twelve bytes that run a system

Strip away the channel coding and a TSBK is compact:

| Field | Size | Meaning |
|---|---|---|
| LB | 1 bit | last block in this TSDU |
| P | 1 bit | protected (encrypted signalling) |
| Opcode | 6 bits | what this block says |
| MFID | 8 bits | manufacturer ID — 0x00 standard, 0x90 Motorola, 0xA4 Harris |
| Payload | 8 bytes | opcode-specific |
| CRC | 16 bits | trailer over the whole block |

One TSDU carries up to **three** TSBKs back to back (`maxTSBKBlocks`), the
last flagged LB=1. That plural is load-bearing: GopherTrunk originally
decoded only the first block per frame, and on a busy site — which fills all
three — that silently discarded roughly two-thirds of the signalling (fixed
under issue #402, with `resumeTSBKBlocks` continuing a block train across IQ
chunks). "Locks fine but misses grants" is exactly what a first-block-only
decoder looks like from outside.

## Through the trellis

On the air, each 12-byte block becomes 98 channel dibits: 96 info bits split
into 48 dibits, trellis-encoded at rate 1/2, then block-interleaved to spread
burst errors. The code itself is worth a look because it is *not* the
convolutional code you'd guess:

```go
// internal/radio/p25/phase1/trellis.go (shape)
// This is NOT the standard (7,5) octal NASA convolutional code; it's a
// table-driven code whose state IS the most-recent input dibit and whose
// transition outputs come from a 16-entry constellation table
// (TIA-102.BAAA-A Annex A Table A.1).
func EncodeTrellis(info []uint8) []uint8 {
    out := make([]uint8, 0, 98)
    state := 0
    for _, d := range info {
        next := int(d & 0x3)
        idx := trellisStates[state][next]
        out = append(out, trellisPairs[idx][0], trellisPairs[idx][1])
        state = next
    }
    /* … finisher dibit flushes state to 0 → the 98th output dibit … */
    return out
}
```

`DecodeTrellis` runs hard-decision Viterbi and returns a **path metric** —
0 for a clean channel, climbing with corrections — which downstream code uses
as a per-block confidence gauge, and `DecodeTSBKChannelSoft` runs the same
pipeline on per-bit LLRs from the receiver's `BitLLRSink` (Part 1's hook,
closing the loop). The general theory lives in the
[framing & FEC deep dive]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }});
what's P25-specific is the trap behind the trellis.

The trailer CRC looks like a solved problem — CRC-CCITT, every reference
says so. But the variant matters: many P25 references document
CRC-CCITT/FALSE (init 0xFFFF, no final XOR), and that is **not what the air
uses**. Real TSBKs verify under the *augmented-codeword* variant — init 0,
final XOR 0xFFFF, run over all 12 bytes, expect 0
(`framing.CRCCCITTAugmented`). The field symptom that forced the fix: on the
Mt Anakie capture, **195 of 197 TSBKs failed CRC while the trellis reported
metric 0** — a clean channel whose every block was rejected by arithmetic.
When a strong FEC says clean and a checksum says corrupt, suspect the
checksum's pedigree before the radio.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="The TSBK receive pipeline as five boxes: 98 channel dibits enter a deinterleaver, then a Viterbi decoder over the table-driven trellis emitting 48 info dibits and a path metric, then a repack into 12 bytes, then the augmented CRC-CCITT gate, then the opcode dispatch; a note marks the CRC as the acceptance gate and the metric as advisory">
  <rect x="12" y="60" width="108" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="66" y="79" text-anchor="middle" fill="currentColor" font-size="10">98 channel dibits</text>
  <text x="66" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per block, ×3 per TSDU</text>
  <line x1="120" y1="82" x2="140" y2="82" stroke="currentColor"/><polygon points="138,78 146,82 138,86" fill="currentColor"/>
  <rect x="146" y="60" width="100" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="196" y="79" text-anchor="middle" fill="currentColor" font-size="10">deinterleave</text>
  <text x="196" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DeinterleaveTSBK</text>
  <line x1="246" y1="82" x2="266" y2="82" stroke="currentColor"/><polygon points="264,78 272,82 264,86" fill="currentColor"/>
  <rect x="272" y="60" width="110" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="327" y="79" text-anchor="middle" fill="currentColor" font-size="10">Viterbi (trellis)</text>
  <text x="327" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">48 dibits + metric</text>
  <line x1="382" y1="82" x2="402" y2="82" stroke="currentColor"/><polygon points="400,78 408,82 400,86" fill="currentColor"/>
  <rect x="408" y="60" width="96" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="456" y="79" text-anchor="middle" fill="currentColor" font-size="10">repack</text>
  <text x="456" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">12 bytes</text>
  <line x1="504" y1="82" x2="524" y2="82" stroke="currentColor"/><polygon points="522,78 530,82 522,86" fill="currentColor"/>
  <rect x="530" y="60" width="138" height="44" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="599" y="79" text-anchor="middle" fill="var(--accent)" font-size="10">augmented CRC-16</text>
  <text x="599" y="93" text-anchor="middle" fill="var(--accent)" font-size="9">the acceptance gate</text>
  <line x1="599" y1="104" x2="599" y2="128" stroke="currentColor"/><polygon points="595,126 599,134 603,126" fill="currentColor"/>
  <rect x="470" y="134" width="198" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="569" y="151" text-anchor="middle" fill="currentColor" font-size="10">dispatch on opcode + MFID</text>
  <text x="569" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grants · identifiers · status · vendor</text>
  <text x="200" y="140" fill="var(--fg-muted)" font-size="10">the Viterbi metric is advisory (0 = clean);</text>
  <text x="200" y="155" fill="var(--fg-muted)" font-size="10">the CRC decides — the wrong CRC variant</text>
  <text x="200" y="170" fill="var(--fg-muted)" font-size="10">rejected 195/197 clean blocks (Mt Anakie)</text>
</svg>
<figcaption>The TSBK receive pipeline: interleaving and the trellis fight the channel, but the augmented CRC-16 is the only gate — get its variant wrong and a clean site decodes nothing.</figcaption>
</figure>

## The opcode families

The 6-bit opcode space (`opcodes.go`, per TIA-102.AABC-D Table 7-1) sorts
into four working families:

| Family | Representative opcodes | What they do |
|---|---|---|
| Voice grants & updates | 0x00 GRP_V_CH_GRANT, 0x02/0x03 updates, 0x04 UU_V_CH_GRANT | start and refresh calls — the reason to listen |
| Channel identifiers | 0x3D IDEN_UP, 0x34 IDEN_UP_VU, 0x33 IDEN_UP_TDMA | publish the band plan (Part 5's subject) |
| Site & system status | 0x3B NET_STS_BCST, 0x3A RFSS_STS_BCST, 0x3C ADJ_STS_BCST, 0x39 SCCB | identity, neighbours, secondary CCs (Part 10) |
| Unit business | 0x28 GRP_AFF_RSP, 0x2C U_REG_RSP, 0x18 STS_UPDT, 0x1F CALL_ALRT | registration, affiliation, paging |

`Opcode.String()` renders the canonical TIA mnemonic so logs read in spec
terms, falling back to `OSP(0xNN)` for unnamed opcodes — with a scope rule
we'll return to below: it is only meaningful for **standard** (MFID 0x00)
outbound opcodes.

## From opcode to `trunking.Grant`

A grant TSBK carries no frequency — only a 4-bit channel ID and 12-bit
channel number that the band plan (built from IDEN_UP blocks) resolves to
hertz. `publishVoiceGrant` does the translation and publishes the result:

```go
// internal/radio/p25/phase1/control.go (shape) — publishVoiceGrant
freq, err := c.bandPlan.Frequency(g.channelID, g.channelNumber)
if err != nil {
    // Grant before its IDEN_UP: buffer it and surface a
    // stage="no-bandplan" decode error, then drain when the slot lands.
    c.pendingGrants.add(g.channelID, g, nac, c.now())
    return
}
c.bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
    System: c.systemName, Protocol: protocol, // "p25", or "p25-phase2" on a TDMA channel
    GroupID: g.groupID, SourceID: g.sourceID, FrequencyHz: freq,
    NAC: nac, Encrypted: so.Encrypted(), Emergency: so.Emergency(),
    /* … RFSS/Site, priority, demod mode, Phase 2 decode config … */
}})
```

Two details preview later parts. A grant arriving *before* its identifier
update isn't dropped — it waits in `pendingGrants` and publishes when the
band plan fills in (Part 5 covers the band plan itself). And a grant on a
channel advertised via IDEN_UP_TDMA (0x33) ships as
`Protocol: "p25-phase2"` so the voice composer routes it into the H-DQPSK
chain — the first place the Phase 1/Phase 2 twins meet (Part 7). What the
engine does with the event — dedup, voice pool, recording — is the
[grant-to-call story]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }})
and its
[grants deep dive]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }});
this layer's job ends at a well-formed `trunking.Grant`.

## Names that lie

The MFID byte partitions the opcode space: under MFID 0x90 (Motorola) or
0xA4 (Harris), the same 6-bit value means something entirely different —
patch-group adds, dynamic regroups, talker aliases (`tsbk_vendor.go`). Which
makes logging a trap with teeth: **never render a vendor opcode through the
standard `Opcode.String()` map.** MFID 0x90 opcode 0x00 is a Motorola
patch-group message; the OSP map would happily label it `GRP_V_CH_GRANT`,
and an operator chasing a grant bug would be reading fiction. The same rule
covers AMBT opcodes (Part 4): `ambtOpcodeLabel` names only the three AMBT
forms GopherTrunk decodes and renders everything else numerically —
`AMBT(0x00)`, never a borrowed name — pinned by a test asserting exactly the
mislabel case. A wrong-but-plausible name in a log is worse than a hex
number.

## The SCCB bug: a round-trip test that lied

Opcode 0x39, the Secondary Control Channel Broadcast, announces the site's
alternate control channels — two (channel, service class) pairs in eight
payload bytes. The parser read channel B from `p[4:6]`. The correct offset is
`p[5:7]`:

```go
// internal/radio/p25/phase1/opcodes.go (shape) — the fixed layout
//	bytes 2-3 : secondary control channel A (4-bit ID + 12-bit number)
//	byte 4    : system service class A
//	bytes 5-6 : secondary control channel B (4-bit ID + 12-bit number)
//	byte 7    : system service class B
cA := binary.BigEndian.Uint16(p[2:4])
cB := binary.BigEndian.Uint16(p[5:7]) // was p[4:6] — one byte early
```

Reading a byte early spliced service class A into the channel field and
produced a **phantom secondary control channel** on every real broadcast. And
the round-trip test was green the whole time, because
`AssembleSecondaryControlChannelBroadcast` encoded the *same wrong layout* —
parse(assemble(x)) == x held perfectly for a format that existed nowhere but
this codebase. It is the purest small specimen of the
[self-consistent-synthetic trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}):
a test whose encoder and decoder share an assumption validates it against
nothing. The fix — and the standing rule — is to pin parsers with **literal
byte vectors cross-checked against an independent decoder** (here SDRTrunk's
field offsets), the discipline
[From Spec to Shipping Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
develops in full. `tsbk_test.go` now carries the mnemonic and layout pins
that would have caught this on day one.

### How the workhorse shaped the Go code

- **Parse and validate are one call.** `ParseTSBK` returns the parsed block
  *and* the CRC verdict together — and returns the partial parse on failure,
  so diagnostics can log what almost arrived.
- **The encoder exists for tests, and that cuts both ways.** `AssembleTSBK` /
  `EncodeTSBKChannel` make synthetic streams easy — which is exactly why
  layout facts need literal vectors from outside the codebase.
- **Metrics ride along.** Every decode carries its Viterbi metric, so a
  marginal channel is visible per block (`TSBKErrorRate`, issue #858's
  frame-error instrument) instead of averaged into folklore.
- **The vendor split is structural.** `IsVendorMFID` gates every vendor
  parse, and vendor decoding lives in its own file — the namespace boundary
  in the spec is a file boundary in the tree.

## Where this goes next

TSBKs are single blocks — 8 payload bytes, take it or leave it. Some systems
put their richest data where a single block can't hold it: multi-block PDUs
on the control channel, which GopherTrunk once logged as `non-control DUID
duid=PDU` and threw away.
[Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})
is the story of learning to read them — and why one neighbour site out of
twelve was ever visible before.

## FAQ

**How many TSBKs per second does a control channel send?**
The channel runs at 9.6 kbps, and a 98-dibit block plus its share of FSW/NID
overhead works out to a few dozen blocks per second — a busy site fills
nearly all of them. That density is why decoding all three blocks per TSDU
matters.

**What does the Viterbi path metric actually tell me?**
How far the received dibits sat from the nearest valid trellis path — 0
means the channel was clean, small values mean corrected noise, large values
mean the decoder guessed hard. GopherTrunk treats it as advisory (the CRC
decides) but logs it, because a rising metric across blocks is the earliest
sign of a degrading control channel.

**Why did the wrong CRC variant ever work in tests?**
Because the tests generated their own TSBKs with the same wrong variant —
encode and check agreed with each other, disagreeing only with the air. Same
shape as the SCCB bug: round-trips validate consistency, not correctness.
Only a real capture (Mt Anakie's 195/197 failure rate) exposed it.

**Is the P (protected) bit the same as an encrypted call?**
No — P marks encrypted *signalling* in the TSBK itself, which is rare on
public-safety systems. Whether a granted *call* is encrypted arrives in the
grant's service options (and later in the voice frames' encryption sync,
Part 9); GopherTrunk stamps `Encrypted` onto the `trunking.Grant` from the
service options at grant time.

**What happens to a grant whose channel ID has no band-plan entry yet?**
It parks in `pendingGrants` and publishes a `stage="no-bandplan"` decode
error so the gap is measurable. When the matching IDEN_UP arrives, the
pending grant drains through the normal path. On a healthy system that wait
is seconds; a *persistent* no-bandplan counter usually means the operator's
seeded band plan (or the site's identifier set) needs Part 5's attention.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Frame Sync, the NID & What 'Locked' Means]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})
· Next →
[Part 4: Multi-Block Trunking — The PDU That Isn't Noise]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})
