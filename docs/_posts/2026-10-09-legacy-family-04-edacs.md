---
title: "The Legacy Family End to End, Part 4: EDACS — 9600-Baud Control Channel Words, BCH, and the LCN Map"
description: "GopherTrunk's EDACS decoder read at the bit level — the 24-bit sync, the two readings of a 40-bit Control Channel Word, the shortened BCH(40,28,2) with its generator 0x1539 and 780-pair double-error search, the command nibble and grant flags, the LCN resolver the production factory never wires to a frequency, the strict tests, the ProVoice bypass, and the honest rung."
category: deep-dives
keywords: edacs decoder, edacs control channel word, edacs ccw 40 bit, bch 40 28 2, edacs 9600 baud gfsk, edacs sync 55d5aa, edacs lcn band plan, edacs provoice, edacs_bch_mode, ge ericsson edacs sdr, gophertrunk legacy family
tags: [legacy-family-end-to-end, edacs, bch, trunking, fec, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 4
---

*Part 4 of **The Legacy Family End to End**, a 14-part deep dive into the
FM-era trunking generation and the AMBE-era narrowband and amateur modes
through GopherTrunk.
[Part 3]({{ '/blog/deep-dives/legacy-family-03-smartnet-air-interface/' | relative_url }})
read the SmartNet control channel, a protocol with no opcodes and a band
plan as its parser. EDACS — GE-Marc, later Ericsson — is the opposite
design: a fast GFSK control channel of fixed 40-bit words, each with a
real command nibble and its own block code. This part reads
`internal/radio/edacs` from the sync word to the grant, and ends on two
gaps the earlier overview in
[Protocol Decoders Part 8]({{ '/blog/deep-dives/protocol-decoders-08-edacs-ltr-mpt1327/' | relative_url }})
did not have room for: the frequency a grant never gets, and the rung
the decoder actually stands on.*

> **TL;DR:** EDACS is a continuous **9600-baud GFSK** control channel,
> BT = 0.3, carrying 40-bit **Control Channel Words** behind a 24-bit
> sync `0x55D5AA` (`OutboundSyncHex`, matched at tolerance 1). GopherTrunk
> reads the same 40 bits two ways: the legacy layout `Command 4 · Status
> 4 · Address 16 · LCN 5 · Aux 11` (`CCWFromBits`), and — under the
> default `edacs_bch_mode: on` — a shortened **BCH(40,28,2)** codeword,
> 28 info bits high and 12 parity low, generator `0x1539` from
> `lwvmobile/edacs-fm`'s `bch3.h`, corrected by a 40-entry syndrome
> table and a 780-pair double-error search. `Ingest` drops `CmdIdle`,
> locks on `CmdSystemID`, folds `CmdAdjacentSite` into the topology and
> publishes `CmdGroupVoiceGrant` / `CmdProVoiceGrant` with `Encrypted`
> (Status bit 0), `Emergency` (bit 1) and `ProVoice`. The package has
> `LinearBandPlan` and `TableBandPlan` resolvers, but `newEDACSPipeline`
> wires none, so a live grant publishes with `FrequencyHz: 0` and the
> engine drops it. ProVoice voice is bypassed. Rung: **reference-pinned**
> for the BCH; the sync and field layout are labelled best-effort in the
> code itself.

**Key takeaways**

- **One word, two readings.** Under `BCHOff` the 40 bits are five
  fields; under `BCHOn` the low 11-bit `Aux` and the LCN's low bit are
  parity, and only 28 bits are data.
- **The BCH is the only on-wire FEC.** The package doc calls an earlier
  claim of a Reed-Solomon layer above it "a documentation error."
- **A lock works; a follow does not.** The control channel locks on a
  system ID and decodes grants, but with no resolver wired the grant
  carries no frequency, and `HandleGrant` logs `dropping grant with
  zero frequency`.
- **Honest rungs differ by layer.** The BCH generator is reference-pinned;
  `sync.go` calls the sync constant "best-effort" and `ccw.go` sources
  the field layout from "the most-cited public reference."

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Sync | 24-bit `0x55D5AA`, sliding `SyncDetector`, production tolerance 1 | `internal/radio/edacs/sync.go`, `process.go` |
| CCW fields | `Command` 39..36, `Status` 35..32, `Address` 31..16, `LCN` 15..11, `Aux` 10..0 | `ccw.go` (`AssembleCCW`, `ParseCCW`, `CCWFromBits`) |
| BCH codeword | 28 info bits at 12..39, 12 parity at 0..11, generator `0x1539`, t = 2 | `internal/radio/framing/bch_edacs.go` |
| Mode switch | `BCHOff` / `BCHOn`; `ParseBCHMode("")` → `BCHOn` | `control.go` (`SetBCHMode`), key `edacs_bch_mode` |
| Commands | `CmdIdle` 0x0 … `CmdEncryption` 0x9, `CmdReserved` 0xF; `IsKnown` | `opcodes.go` |
| Grant flags | `IsEncrypted` Status bit 0, `IsEmergency` bit 1, `ProVoice` from `CmdProVoiceGrant` | `opcodes.go` (`AsGroupVoiceGrant`) |
| LCN → Hz | `LinearBandPlan{BaseHz, SpacingHz, Offset}`, `TableBandPlan` — unwired in production | `bandplan.go`, `ccdecoder/pipelines.go` (`newEDACSPipeline`) |
| Receiver | FM → `demod.GFSK` matched filter (BT 0.3) → Mueller-Müller → `gfsk.Slice` | `receiver/receiver.go` |

## In this post

- **The control channel on the wire** — 9600 baud, BT 0.3, a best-effort sync.
- **Two readings of 40 bits** — legacy fields and the BCH codeword.
- **BCH(40,28,2) in practice** — generator, syndrome table, 780 pairs.
- **Commands, grants and topology** — what `Ingest` acts on.
- **The LCN map that stops short** — resolvers in the package, none in the factory.
- **Strict tests, ProVoice and the rung** — pinned and deferred.

## The control channel on the wire

The receiver subpackage states the physical layer in its header: "EDACS
runs a continuous 9600-baud GFSK control channel with the standard
BT = 0.3 Gaussian premod filter." The chain is the skeleton from
[Part 2]({{ '/blog/deep-dives/legacy-family-02-the-shared-skeleton/' | relative_url }})
with a Gaussian matched filter as its shaping stage: `demod.FM`, then
`demod.GFSK` at `int(sps+0.5)` samples per symbol with a
`PulseSpanSymbols` = 4 half-span and `BT` = 0.3, then `sync.MuellerMuller`
and `gfsk.Slice`. At the production 48 kHz rate that is 5 samples per
symbol; `New` panics below 19200 Hz. The receiver tests drive an
alternating NRZ stream at 96 kHz and ±2400 Hz, and
`TestDaemonCCDecodesEDACS` modulates the same parameters through
`demod.ModulateGFSK` with the BCH chain on.

Framing starts at the sync. `sync.go` declares `OutboundSyncHex uint32 =
0x55D5AA` and `SyncBits = 24`; `SyncDetector` slides a 24-bit history
over the stream and reports where the pattern ends within `tolerance`
mismatches, which `process.go` sets to 1. `TestSyncDetectorTolerance`
flips two bits and checks that tolerance 2 matches once and tolerance 0
not at all. On a match the adapter sets `remaining = 40` and collects
the next forty bits.

The constant's own comment is the package's first honest line: the sync
"is documented as 0x55D5AA across multiple public reference
implementations. As with the other protocol packages, the constant is
best-effort and should be cross-checked against an authoritative
reference before trusting live captures." No test pins it to an upstream
literal the way `TestOutboundSyncBitsMatchReference` pins SmartNet's
`0xAC`; `TestSyncDetectorMatchesCleanSync` checks the detector against
the package's own `OutboundSyncBits()`.

## Two readings of 40 bits

`ccw.go` models a CCW as `{Command, Status, Address, LCN, Aux}` and packs
it MSB-first into 40 bits:

```go
// internal/radio/edacs/ccw.go — AssembleCCW
v := cmd<<36 | status<<32 | addr<<16 | lcn<<11 | aux
//     4 bits   4 bits       16 bits   5 bits    11 bits
```

That is the `BCHOff` reading, "useful only for synthesized test fixtures
whose codewords are not BCH-protected" per the `BCHMode` doc, and the
in-package zero-value default. `TestCCWAssembleParseRoundTrip` and
`TestCCWFromBitsRoundTrip` cover it — round-trips, which prove
consistency and nothing more.

Under `BCHOn`, the production default, the same 40 bits are a shortened
BCH codeword: "info at bits 12..39 high; 12-bit BCH parity at bits
0..11 low." The 28 information bits are `Command` at 36..39, `Status` at
32..35, `Address` at 16..31 and the **high four bits** of the LCN at
12..15. The doc comment is explicit about what that costs: "The existing
`Aux` field at codeword bits 0..10 is BCH parity under BCHOn — not data.
Callers that depend on the legacy 5-bit LCN range or the Aux payload
should keep using BCHOff."

`parseCCW` is where the two readings meet. It packs the 40 wire bits
into a `uint64` with `wire[0]` at bit 39, calls `framing.BCHDecodeEDACS`,
drops the word on `errs == -1`, re-encodes the corrected 28 bits with
`BCHEncodeEDACS` into a clean codeword and hands *that* to `CCWFromBits`.
The field reader therefore sees a 5-bit LCN whose low bit is parity
bit 11 of the re-encoded codeword. For the three vectors in
`process_bch_test.go` — LCN 10, 8 and 4 — that bit is zero, so
`TestProcessBCHOnDecodesEncodedCCW` compares `ChannelNum` against
`LCN & 0x1E` and passes; the `BCHOn` comment documents the low bit as
"BCH parity", and the EDACS LCN is effectively a 4-bit value under the
default mode.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A 40-bit EDACS Control Channel Word drawn twice. The upper reading, BCHOff, splits it into Command 4, Status 4, Address 16, LCN 5 and Aux 11 bits. The lower reading, BCHOn, splits the same bits into 28 information bits (Command, Status, Address, LCN high 4) and 12 BCH parity bits, with the generator 0x1539 and the correction capacity t equals 2 noted.">
  <text x="20" y="18" fill="var(--fg-muted)" font-size="10" font-weight="bold">BCHOff — legacy five-field reading (bit 39 … bit 0)</text>
  <rect x="20" y="26" width="64" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="52" y="45" text-anchor="middle" fill="currentColor" font-size="9">Cmd 4</text>
  <rect x="84" y="26" width="64" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="116" y="45" text-anchor="middle" fill="currentColor" font-size="9">Status 4</text>
  <rect x="148" y="26" width="256" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="276" y="45" text-anchor="middle" fill="currentColor" font-size="9">Address 16</text>
  <rect x="404" y="26" width="80" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="444" y="45" text-anchor="middle" fill="currentColor" font-size="9">LCN 5</text>
  <rect x="484" y="26" width="176" height="30" rx="3" fill="none" stroke="currentColor"/>
  <text x="572" y="45" text-anchor="middle" fill="currentColor" font-size="9">Aux 11</text>
  <text x="20" y="92" fill="var(--accent)" font-size="10" font-weight="bold">BCHOn (default) — shortened BCH(40,28,2) codeword reading</text>
  <rect x="20" y="100" width="64" height="30" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="52" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">Cmd 4</text>
  <rect x="84" y="100" width="64" height="30" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="116" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">Status 4</text>
  <rect x="148" y="100" width="256" height="30" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="276" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">Address 16</text>
  <rect x="404" y="100" width="64" height="30" rx="3" fill="none" stroke="var(--accent)"/>
  <text x="436" y="119" text-anchor="middle" fill="var(--accent)" font-size="9">LCN hi 4</text>
  <rect x="468" y="100" width="192" height="30" rx="3" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="564" y="119" text-anchor="middle" fill="currentColor" font-size="9">BCH parity 12 (bits 11..0)</text>
  <line x1="20" y1="140" x2="404" y2="140" stroke="var(--accent)"/>
  <text x="212" y="152" text-anchor="middle" fill="var(--accent)" font-size="8">28 information bits (codeword bits 39..12)</text>
  <line x1="468" y1="140" x2="660" y2="140" stroke="var(--fg-muted)"/>
  <text x="564" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Aux and LCN bit 0 are parity, not data</text>
  <text x="340" y="178" text-anchor="middle" fill="currentColor" font-size="9">g(x) = x¹² + x¹⁰ + x⁸ + x⁵ + x⁴ + x³ + 1 = 0x1539 · corrects t = 2 errors per word · errs = −1 ⇒ word dropped</text>
</svg>
<figcaption>The same 40 bits under the two modes: the default trades Aux and one LCN bit for twelve parity bits and two correctable errors.</figcaption>
</figure>

## BCH(40,28,2) in practice

The code itself lives in `internal/radio/framing/bch_edacs.go`, and its
header is the package's clearest provenance statement. The shortened
code derives from the BCH(63,51,2) mother code over GF(2⁶) with
primitive polynomial x⁶ + x + 1; the generator is the product of the
minimal polynomials of α and α³:

```text
m₁(x) = x^6 + x + 1
m₃(x) = x^6 + x^4 + x^2 + x + 1
g(x)  = m₁(x) · m₃(x) = x^12 + x^10 + x^8 + x^5 + x^4 + x^3 + 1 = 0x1539
```

The source named is `lwvmobile/edacs-fm`'s `bch3.h`, "the most-cited
public reference for the EDACS channel coding." At `init` the package
builds `bchEDACSSyndromes[40]`, the syndrome `x^i mod g(x)` of a single
error at each position, and decoding is a lookup: zero means clean; a
table match corrects one bit; otherwise the decoder walks the 780
ordered pairs `(i, j)` matching `syndromes[i] ^ syndromes[j]`; no match
returns `errs = -1`. `TestBCHEDACSCorrectsAnySingleBitError` and
`TestBCHEDACSCorrectsAnyDoubleBitError` exhaust every position and pair.
The encoder is systematic — parity is the syndrome of the info shifted
up 12 — which is what `parseCCW` uses to re-encode.

The package doc closes a loop earlier documentation had opened: "Per the
canonical open reference (`lwvmobile/edacs-fm`), the BCH(40, 28, 2) per
CCW is the only on-wire FEC layer on the Standard EDACS control channel
— earlier package comments that referenced an 'interleaved
Reed-Solomon-derived FEC' above the BCH were a documentation error."
`process_bch_test.go` runs the path from stream to bus: an encoded
`CmdGroupVoiceGrant` publishes; one flipped bit at wire index 19 still
publishes; two at 5 and 25 still publish; three at 1, 15 and 30 publish
nothing. All four encode with `framing.BCHEncodeEDACS` — the
shared-encoder caveat from
[the self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
applies to the layout, while the generator is what the reference pins.

## Commands, grants and topology

`opcodes.go` names eleven of the sixteen command values: `CmdIdle` 0x0,
`CmdGroupVoiceGrant` 0x1, `CmdProVoiceGrant` 0x2, `CmdIndividualCall`
0x3, `CmdDataGrant` 0x4, `CmdSystemID` 0x5, `CmdAdjacentSite` 0x6,
`CmdEmergency` 0x7, `CmdAffiliation` 0x8, `CmdEncryption` 0x9 and
`CmdReserved` 0xF. `IsKnown` returns true for exactly those; 0xA..0xE
are unallocated. The enum is wider than the state machine: `Ingest`
drops idle words first (before the strict check), then acts on three
kinds only.

```go
// internal/radio/edacs/control.go — Ingest (shape)
if sys, ok := w.AsSystemID(); ok {
    c.topo.applySystemID(sys.ID)
    c.maybeLock(LockState{FrequencyHz: c.freqHz, SystemID: sys.ID})
    return
}
if adj, ok := w.AsAdjacentSite(); ok { c.topo.applyAdjacent(adj); return }
if grant, ok := w.AsGroupVoiceGrant(); ok { c.publishGrant(grant); return }
```

A `CmdSystemID` word carries the system identifier in `Address` and a
site/network subfield in `Aux` — parity under `BCHOn`, so `SystemID.Aux`
is meaningful only on the legacy reading. The lock publishes
`LockState{FrequencyHz, SystemID}` and logs `edacs cc locked`. A
`CmdAdjacentSite` word carries the neighbour's `SiteID` in `Address` and
its control-channel `LCN`; `applyAdjacent` de-duplicates by site, and
`edacsPipeline.TopologySnapshot` maps each to a `TopoNeighborRef` with
a frequency from `NeighborFrequency` when a resolver exists.
`TestTopologyAccumulation` feeds sites 11, 12 and a duplicate 11,
expecting two.

A voice grant is `CmdGroupVoiceGrant` or `CmdProVoiceGrant`;
`AsGroupVoiceGrant` lifts the talkgroup from `Address`, the LCN, and the
two Status-nibble flags — `IsEncrypted` bit 0, `IsEmergency` bit 1 —
plus `ProVoice` when the command was 0x2. `publishGrant` emits
`trunking.Grant{Protocol: "edacs", GroupID, ChannelNum: LCN, Encrypted,
Emergency, ProVoice}`; `TestControlChannelPropagatesProVoiceFlag` checks
the flag is grant-type-specific. `CmdIndividualCall`, `CmdDataGrant`,
`CmdEmergency`, `CmdAffiliation` and `CmdEncryption` are parsed as names
and otherwise ignored.

## The LCN map that stops short

Every EDACS grant references a Logical Channel Number, and `bandplan.go`
provides the two resolvers the skeleton promised:

```go
// internal/radio/edacs/bandplan.go
type Resolver interface { Frequency(lcn uint8) (uint32, error) }
type LinearBandPlan struct { BaseHz, SpacingHz uint32; Offset int } // BaseHz + (lcn+Offset)*SpacingHz
type TableBandPlan map[uint8]uint32
```

`TestLinearBandPlan` checks LCN 10 on an 851.000 MHz / 25 kHz plan →
851.250 MHz; `TestControlChannelPublishesGrant` wires a
`LinearBandPlan{866_000_000, 25_000}` and expects LCN 8 at 866.200 MHz.
In the package, the map works.

In the daemon it is never built. `newEDACSPipeline` constructs
`edacs.New(edacs.Options{Bus, Log, SystemName, FrequencyHz})` — no
`Resolver` — and `config.go` has no `edacs_band_plan` key; `trunking.System`
carries `DMRBandPlan` and `NXDNBandPlan` but no EDACS mirror.
`publishGrant` then emits `FrequencyHz: 0`, which
`TestControlChannelGrantWithoutResolverHasZeroFreq` pins as the package
behaviour, and `Engine.HandleGrant`'s first check logs `dropping grant
with zero frequency` and returns. The chain therefore locks on a live
EDACS site and publishes grants the engine refuses to follow.
`TestDaemonCCDecodesEDACS` asserts lock only. The fix has a template —
`dmr_band_plan`'s config section, `trunking` mirror and
`tier3.ResolverFromPlan` — and no EDACS instance of it exists.

## Strict tests, ProVoice and the rung

`strict_test.go` pins the "soft FEC" lever:
`TestStrictValidationDropsUnknownCommand` sends `Command(0xA)` under
`SetStrictValidation(true)` and expects silence,
`TestStrictValidationKeepsKnownCommand` sends a voice grant and expects
one, and `TestCommandIsKnownCoversAllConstants` checks the eleven known
values and the five unknown. As Part 2 noted, nothing in production
calls `SetStrictValidation`; with `BCHOn` the block code is the gate,
and a corrected word with an unallocated command reaches `Ingest`,
which does nothing with it.

ProVoice is the voice gap. A `CmdProVoiceGrant` publishes with
`ProVoice: true`; the composer's `classifyVoiceKind` sends an EDACS
grant to `voiceKindFM` only when `!cs.Grant.ProVoice`, and otherwise to
`voiceKindUnsupported`, logging `composer: digital protocol not yet
decoded; chain bypassed`; `discoveredMode` labels the talkgroup `"D"`.
The package doc's deferral names the starting points — "closest open
reference is `lwvmobile/edacs-fm` (pv_*.c) and `szechyjs/dsd`" — and
`docs/status.md` lists "YSF voice and EDACS ProVoice are still bypassed."

The rung, by layer. The BCH generator and its correction capacity are
**reference-pinned** to `edacs-fm`'s `bch3.h` and exhaustively tested.
The sync word and five-field layout are what the code calls best-effort
and "the most-cited public reference" — pinned to the package's own
constants, not an upstream literal, with the cross-check caveat written
into `sync.go`. No real-air capture has been through the chain;
`docs/decoder-capture-needs.md` lists EDACS among the control chains
that "ship; FEC is on by default with no outstanding capture", a
statement about blockers, not verification. Any clean control-channel IQ
with a known system ID would pin the sync and layout in one pass — and
surface the zero-frequency grant the moment the first call keyed up.

## Where this goes next

EDACS has a control channel and a block code; the next protocol has
neither. LTR puts a 41-bit status word under every repeater's voice at
300 bits per second, with a CRC-7 as its only check and no central
channel to hunt.
[Part 5]({{ '/blog/deep-dives/legacy-family-05-ltr/' | relative_url }})
reads `internal/radio/ltr` — the sub-audible receiver, the Manchester
modes, the FCS that covers 24 of the word's bits, the per-repeater lock
the hunter mistakes for a control channel, and an integration test that
runs with both production defaults switched off.

## FAQ

**What is an EDACS Control Channel Word and how does GopherTrunk parse it?**
A 40-bit block behind a 24-bit sync `0x55D5AA`. `CCWFromBits` reads it
as `Command 4 · Status 4 · Address 16 · LCN 5 · Aux 11`; under the
default `edacs_bch_mode: on`, `parseCCW` first treats the 40 bits as a
BCH(40,28,2) codeword — 28 info bits high, 12 parity low — corrects up
to two errors with `framing.BCHDecodeEDACS`, and drops uncorrectable
words.

**Why doesn't an EDACS voice grant get followed in GopherTrunk?**
`newEDACSPipeline` constructs the control channel with no `Resolver`
and no `edacs_band_plan` config key exists, so `publishGrant` emits
`FrequencyHz: 0` and `Engine.HandleGrant` logs `dropping grant with
zero frequency`. The package's `LinearBandPlan` and `TableBandPlan` work
in tests; wiring them from config is the missing piece.

**Does GopherTrunk decode EDACS ProVoice?**
No. A `CmdProVoiceGrant` publishes a grant flagged `ProVoice: true`, the
composer classifies it `voiceKindUnsupported` and logs `chain bypassed`,
and the call is logged without audio. Analog EDACS grants go to the FM
chain.

**Is the EDACS decoder verified on air?**
No. The BCH(40,28,2) generator `0x1539` is reference-pinned to
`edacs-fm`'s `bch3.h` and exhaustively tested; the sync word and field
layout are labelled best-effort in `sync.go` and `ccw.go`. A clean
control-channel capture with a known system ID is the step that moves
it.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Motorola Type II — 3600-Baud OSWs, Sync 0xAC and the Band Plans]({{ '/blog/deep-dives/legacy-family-03-smartnet-air-interface/' | relative_url }})
· Next →
[Part 5: LTR — Subaudible Data on Every Repeater, No Control Channel]({{ '/blog/deep-dives/legacy-family-05-ltr/' | relative_url }})
