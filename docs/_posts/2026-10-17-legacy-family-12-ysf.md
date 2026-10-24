---
title: "The Legacy Family End to End, Part 12: Yaesu System Fusion — FICH Trellis, Frame Types and the Voice That Is Not Yet PCM"
description: "Inside GopherTrunk's YSF package: the 480-dibit frame and its 40-bit FSW 0xD471C9634D, the 32-bit FICH and its CRC with the XMODEM init, the K=5 trellis with puncture {0,1,102,103} and the 10×10 interleave ported from MMDVMHost, the DG-ID grant logic, and the two gaps that keep System Fusion at a synthetic lock: no FICH caller on the hot path, and no voice chain at all."
category: deep-dives
keywords: yaesu system fusion decoder go, ysf fich decode, ysf frame sync word 0xd471c9634d, ysf fich trellis k5 puncture, ysf 10x10 interleave mmdvmhost, ysf dg-id squelch code grant, ysf dn vw mode, c4fm 4800 baud sdr fusion, ysf voice not decoded, ysf capture dn mode, gophertrunk legacy family
tags: [legacy-family-end-to-end, system-fusion, ysf, c4fm, trellis, amateur-radio, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 12
---

*Part 12 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out, with each part naming
the verification rung its protocol stands on.
[Part 11]({{ '/blog/deep-dives/legacy-family-11-dstar/' | relative_url }})
read D-STAR's GMSK bit chain and its self-consistent header shell. This
part takes the other amateur mode, Yaesu System Fusion: familiar
4800-baud C4FM underneath, and above it the most thoroughly tested code in
this series that nothing on the hot path ever calls.
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
named that gap in a paragraph; this part reads the frame, the FICH, the
trellis and the state machine bit by bit.*

> **TL;DR:** `internal/radio/ysf` decodes a 480-dibit, 100 ms C4FM frame:
> 20 dibits of Frame Sync Word (`FSWBits` = `0xD471C9634D`), 100 of FICH,
> 360 of DCH (`FrameDibits`, `FICHDibits`, `PayloadDibits`). The receiver
> is `demod.FM` → `demod.C4FM` (RRC α 0.20, span 8) → `sync.MuellerMuller`
> → `SymbolToDibit` (the P25 Phase 1 Gray convention), sliced against a
> hard-coded `DeviationHz` 1800. The FICH is 32 info bits + CRC-16-CCITT
> with init `0x0000` (`TestFICHCRCUsesXMODEMInit`), carried on air as 100
> bits: K=5 ½-rate trellis → `FICHChannelBits` 104 → puncture
> `{0, 1, 102, 103}` → column-major 10×10 interleave, the MMDVMHost
> schedule; `DecodeFICHOnAir` inverts it and
> `TestFICHOnAirRecoversFromSingleBitFlip` corrects all 100 positions.
> `ProcessFICH` turns a Header + `CallTypeGroup` into a `trunking.Grant`
> with `Protocol "ysf"` and the DG-ID as `GroupID`. But `newYSFPipeline`
> wires the receiver to `Process` alone, which only `maybeLock`s on the
> FSW; `ProcessFICH`'s callers are all tests, and the composer maps `"ysf"`
> to `voiceKindUnsupported`. Rung: synthetic lock; reference-pinned FICH
> codec; **voice not decoded**.

**Key takeaways**

- **The frame is three fixed fields.** 20 + 100 + 360 dibits, pinned by
  `TestFrameLayoutAddsUp`.
- **The FICH codec is the one reference-pinned layer.** Puncture
  positions and the 10×10 permutation are MMDVMHost's, with DSDcc's
  alternate schedule documented as a two-line swap.
- **A grant exists on paper only.** `ProcessFICH` publishes DG-ID grants
  and clears them on Terminator; nothing slices the FICH region to call it.
- **No voice, no placeholder.** YSF is `voiceKindUnsupported`; the DCH
  carries AMBE+2 that GopherTrunk does not yet carve.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame layout | 480 dibits = FSW 20 + FICH 100 + DCH 360, 100 ms | `internal/radio/ysf/frame.go` (`FrameDibits`, `FICHOffset`, `PayloadOffset`) |
| Sync | 40-bit `0xD471C9634D` as 20 dibits, tolerance 2 | `internal/radio/ysf/sync.go` (`FSWBits`, `FSWPattern`, `SyncDetector`) |
| Dibits from IQ | FM → RRC C4FM → MM clock → `SymbolToDibit` | `internal/radio/ysf/receiver/receiver.go` (`RolloffAlpha`, `DeviationHz`) |
| FICH fields | 32 bits in 4 octets + CRC init `0x0000` | `internal/radio/ysf/fich.go` (`ParseFICH`, `AssembleFICH`) |
| FICH on-air codec | trellis 104 → puncture 4 → 10×10 interleave → 100 | `internal/radio/ysf/fich_trellis.go` (`EncodeFICHOnAir`, `DecodeFICHOnAir`) |
| Lock and grant | FSW → `maybeLock`; Header FICH → DG-ID grant | `internal/radio/ysf/control.go` (`Process`, `ProcessFICH`) |
| The missing join | receiver → `cc.Process` only | `internal/scanner/ccdecoder/pipelines.go` (`newYSFPipeline`) |
| Capture contract | DN-mode IQ ≥ 10 s, 100 % FICH CRC pass bar | `samples/ysf/README.md`, `docs/decoder-capture-needs.md` |

## In this post

- **Four hundred and eighty dibits** — the frame, the FSW and the receiver's slicer.
- **Thirty-two bits that name the frame** — FICH fields, the XMODEM CRC and the DN/VW modes.
- **The trellis, the puncture and the 10×10** — MMDVMHost's schedule and DSDcc's alternate.
- **A state machine with one live input** — `Process`, `ProcessFICH` and the join that is not there.
- **The voice that is not yet PCM** — what the composer does with a YSF grant.
- **The rung** — what is pinned, what a DN-mode capture would decide.

## Four hundred and eighty dibits

YSF runs 4800-baud 4-level C4FM, so one frame is 480 symbols — 960 bits,
100 ms. `frame.go` fixes the three fields and their offsets:

```go
// internal/radio/ysf/frame.go (shape)
const (
    FrameDibits   = 480
    FICHDibits    = 100
    PayloadDibits = FrameDibits - FSWDibits - FICHDibits // 360
    FSWOffset     = 0
    FICHOffset    = FSWOffset + FSWDibits   // 20
    PayloadOffset = FICHOffset + FICHDibits // 120
)
```

`TestFrameLayoutAddsUp` pins the sums and `TestFrameDurationMatches4800Baud`
checks `FrameDurationMs` = 100 against 480 symbols at 4800 sym/s. The Frame Sync Word is `FSWBits uint64 =
0xD471C9634D`, decomposed MSB-first into the 20-dibit `FSWPattern`
(`TestFSWPatternRoundTrips` reassembles it); `ysf.SyncDetector` matches it
within a tolerance `ControlChannel` defaults to 2, and a negative
tolerance clamps to 1 (`TestSyncDetectorNegativeToleranceClampsToOne`).
Like every detector in the family, the reported index is the dibit where
the FSW *ends*.

The receiver is the P25 Phase 1 chain under a different sync word:
`demod.FM` → `demod.C4FM` (`RolloffAlpha` 0.20, `PulseSpanSymbols` 8) →
`sync.MuellerMuller` → `SymbolToDibit`. That last function maps +1 → 0, +3 → 1, −1 → 2, −3 → 3 —
the TIA-102.BAAA Gray convention — and its own comment says it "tracks
the P25 / DSDcc convention pending real-air capture validation against
the FSWPattern". The slicer scale is `2π · DeviationHz / SampleRateHz`
when `DeviationHz` is set, and `newYSFPipeline` sets it to **1800.0**
("YSF spec peak deviation"). The capture documents say otherwise: `samples/ysf/README.md` and
`docs/decoder-capture-needs.md` both list "±2700 Hz peak deviation".
There is no `ysf_deviation_hz` key to bridge them — NXDN has
`nxdn_deviation_hz`, YSF does not — so a capture that slices bimodally
will be the first evidence of which number is right.

## Thirty-two bits that name the frame

The FSW says *that* a transmission is on air; the FICH says *what kind*.
`fich.go` lays out 32 information bits across four octets:

```text
octet 0  FT[7:6] frame type · CT[5:4] call type · BN[3:2] · BT[1:0]
octet 1  FN[7:5] frame number · FT2[4:2] frame total · DT[1:0] data type
octet 2  VoIP[7] · DT2[6:5] · SQM[4] squelch mode · SQ[3:0] (high 4 bits)
octet 3  SQ[7:5] (low 3 bits) · DEV[4:3] · reserved[2:0]
octets 4-5  CRC-16 (poly 0x1021, init 0x0000) over octets 0..3
```

`FrameType` is Header 0, Communications 1, Terminator 2, Test 3;
`CallType` Group 0 or RadioID 1; `DataType` is `VDMode1` 0, `DataFR` 1,
`VDMode2` 2, `VoiceFR` 3 — the V/D modes are what Yaesu markets as **DN**
and `VoiceFR` is **VW**, per the
[System Fusion reference]({{ '/reference/system-fusion-ysf/' | relative_url }}).
The 7-bit squelch code (the DG-ID) straddles octets 2 and 3, which
`TestFICHRoundTripExercisesAllFields` covers with `0x5A`. The CRC detail
is the one most likely to bite a port: `ParseFICH` computes
`framing.CRCCCITTWithInit(b[:4], 0x0000)`, not the `0xFFFF` init the rest
of the tree uses, and `TestFICHCRCUsesXMODEMInit` guards it — all-zero
info octets must produce CRC `0000`. A mismatch returns the parsed struct
with `CRCError`, so a caller can log fields without trusting them.

## The trellis, the puncture and the 10×10

Those 48 bits (32 + 16) never appear on air. `fich_trellis.go` derives the
on-air form from constants: `FICHInfoBits` 48 plus `FICHTailBits` 4 is
encoded by `framing.EncodeK5` into `FICHChannelBits` = 2 × 52 = 104; four
positions `fichPuncturePositions = {0, 1, 102, 103}` — flanking the
tail-bit boundary — are dropped to give `FICHOnAirBits` = 100; and a
column-major 10×10 permutation `fichInterleavePerm[k] = (k%10)*10 + (k/10)`
spreads them across the FICH region. `DecodeFICHOnAir` runs the inverse:

```go
// internal/radio/ysf/fich_trellis.go (shape)
depunctured[fichInterleavePerm[k]] = channel[k]      // deinterleave
out[i] = framing.DepunctureMark                       // at the 4 punctures
bits, metric := framing.ViterbiK5(out, FICHInfoBits+FICHTailBits)
return bits[:FICHInfoBits], metric, nil
```

The returned metric is the Viterbi path cost — 0 on a clean round trip
(`TestFICHOnAirRoundTrip`) — and `samples/ysf/README.md` turns it into an
acceptance bar: ≤ 4 per 100-bit block at ≥ 12 dB SNR. The tests are exhaustive where that is cheap:
`TestFICHOnAirRecoversFromSingleBitFlip` flips every one of the 100 on-air
positions and demands every one be corrected (K=5 rate-½ has free
distance 7); `TestFICHInterleavePermBijective` checks the permutation
covers 0..99 exactly once; `TestFICHPuncturePositionsExactly4` checks
`FICHChannelBits − FICHOnAirBits` equals the table length and the
positions are strictly increasing, because both loops rely on the
ordering.

This is the **reference-pinned** layer. The file says the schedule is
"per MMDVMHost YSFFICH.cpp (verified against DSDcc dsd_ysf.cpp)", and the
README documents the fallback if a real FICH fails its CRC: swap the
puncture table for DSDcc's spread variant `{0, 51, 52, 103}` and the
permutation for `(k%4)*25 + (k/4)`, then re-run the invariant tests. If
neither decodes, publish which K=5 generator pair the capture needed —
`(0o23, 0o35)` or `(0o31, 0o27)`. That is as far as pinning goes without
air: two open decoders agree, and the signal that could disagree with
both has not been recorded.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="A 480-dibit YSF frame drawn as three boxes: FSW 20 dibits, FICH 100 dibits, DCH 360 dibits. Below the FICH box a solid chain reads DecodeFICHOnAir, deinterleave 10 by 10, depuncture 0 1 102 103, Viterbi K equals 5, ParseFICH, ProcessFICH, grant. An arrow from the receiver reaches only the FSW box and the label cc.Process to maybeLock; the arrow into the FICH chain is dashed and labelled no caller on the hot path. Under the DCH box a dashed note reads AMBE+2 voice, not carved.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one YSF frame: 480 dibits · 100 ms at 4800 sym/s</text>
  <rect x="20" y="30" width="60" height="28" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="50" y="48" text-anchor="middle" fill="var(--accent)" font-size="8">FSW 20</text>
  <rect x="80" y="30" width="150" height="28" fill="none" stroke="currentColor"/>
  <text x="155" y="48" text-anchor="middle" fill="currentColor" font-size="8">FICH 100 dibits</text>
  <rect x="230" y="30" width="430" height="28" fill="none" stroke="var(--fg-muted)"/>
  <text x="445" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DCH 360 dibits (voice / data per DT)</text>
  <text x="50" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">cc.Process → maybeLock</text>
  <text x="50" y="92" text-anchor="middle" fill="var(--accent)" font-size="8">(the whole live path)</text>
  <line x1="155" y1="58" x2="155" y2="104" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="155" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no caller on the hot path</text>
  <rect x="40" y="104" width="600" height="26" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="340" y="121" text-anchor="middle" fill="currentColor" font-size="8">DecodeFICHOnAir: deinterleave 10×10 → depuncture {0,1,102,103} → ViterbiK5 (104→48) → ParseFICH (CRC init 0x0000) → ProcessFICH → grant</text>
  <text x="340" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="8">tested: every single-bit flip corrected · permutation bijective · puncture count = 104 − 100 · called only from control_test.go</text>
  <text x="445" y="178" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DCH: AMBE+2 voice frames — not carved, not decoded (composer: voiceKindUnsupported)</text>
</svg>
<figcaption>The solid path is what runs live: a Frame Sync Word match and a lock. The dashed chain is complete and tested, and ends at a grant no caller has ever requested.</figcaption>
</figure>

## A state machine with one live input

`control.go` has two entry points, and only one is wired.
`ControlChannel.Process` runs the FSW detector and, per hit, calls
`maybeLock`, which publishes `events.KindCCLocked` once with
`LockState{FrequencyHz}` and logs `ysf cc locked`. `LockedNAC` returns 0 — YSF has no NAC — and
`TestLockStateSatisfiesTrunkingLockedPayload` pins the hunter's interface. `MarkLost` publishes `cc.lost` and
clears the grant state. That is the entire live path: `newYSFPipeline` builds `ysfrx.New` with a
`DibitSink` that calls `opts.tapDibits` and `cc.Process`, nothing else.

The second entry point is where the protocol logic lives:

```go
// internal/radio/ysf/control.go (shape)
func (c *ControlChannel) ProcessFICH(f FICH) {
    switch f.FrameType {
    case FrameTypeHeader:     c.publishGrantFor(f)
    case FrameTypeTerminator: c.clearGrant()
    }
}
// publishGrantFor: CallTypeRadioID → return (private, not a talkgroup)
groupID := uint32(0)
if f.SquelchMode { groupID = uint32(f.SquelchCode) }   // the DG-ID
if c.hasGrant && c.lastDGID == uint8(groupID) { return } // same call
```

A Header frame for a group call publishes a `trunking.Grant` with
`Protocol "ysf"`, `GroupID` equal to the DG-ID when code squelch is active
and 0 (an "all calls" group) when the carrier is open, and `SourceID` 0 —
the comment notes the radio ID "lives in the DCH region, not the FICH".
Duplicate Headers for the same DG-ID are suppressed until a Terminator
clears `hasGrant` (`TestControlChannelDuplicateHeaderSuppressed`), and a
`CallTypeRadioID` Header is dropped so the engine does not spawn a
recorder for a private transmission (`TestControlChannelPrivateCallSkipsGrant`). The doc comment
on `ControlChannel` is explicit about the arrangement: "the caller is
expected to have decoded the 100-dibit FICH region into a parsed FICH
struct … and to hand it in. A future PR closes the IQ → dibit → FICH gap
on the hot path."

The daemon-level test shows the consequence. `TestDaemonCCDecodesYSF`
modulates 30 back-to-back frames with the `FSWPattern` at offset 0 and
**zero-filled** FICH and payload regions at 1800 Hz deviation, boots the
daemon, and asserts `events.KindCCLocked`. It cannot assert a grant: there
is no FICH in the fixture and no code that would read one.

## The voice that is not yet PCM

Follow a hypothetical YSF grant into the composer and it stops at the
door. `classifyVoiceKind` in `internal/voice/composer/composer.go` maps
`"dstar"`, `"nxdn"` and `"dpmr"` to their chains; `"ysf"` falls through to
`voiceKindUnsupported`, whose comment reads "no composer voice chain yet
(YSF, EDACS ProVoice)", and `handleStart` logs:

```text
composer: digital protocol not yet decoded; chain bypassed protocol=ysf group=…
```

`docs/status.md` says the same: "YSF voice and EDACS ProVoice are still
bypassed — their calls are followed and logged but not yet turned into
PCM." This is a different gap from the NXDN, dPMR and D-STAR chains of
[Part 13]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }}):
those have a placeholder deinterleave in front of a real vocoder; YSF has
no carve of the 360-dibit DCH at all, so there is nothing on the voice
side for a capture to confirm or refute yet. `docs/decoder-capture-needs.md`
files it under "Separate from captures" — an implementation gap.

What a capture *would* unblock is the FICH. The README is strict about
the input: "audio-only recordings (MP3, post-FM-demod WAV) cannot validate
the YSF decoder" because the 4-level constellation does not survive a
discriminator recording, and an earlier `Yaesu_sys_fusion.wav` upload was
removed for exactly that reason. The ask is a DN-mode IQ capture of ≥ 10 s at ≥ 48 kHz with a
`metadata.json` whose `expected.fich_sequence` lists `ft`/`dt`/`fn`/`ct`
per frame from a Pi-Star FICH log or DSDcc; the pass bar is 100 % FICH
CRC at clean SNR through the shipped schedule, or the documented swap.

## The rung

Placed on the ladder: **synthetic-green** for the lock
(`TestDaemonCCDecodesYSF`); **reference-pinned** for the FICH codec
(MMDVMHost's schedule, cross-checked against DSDcc) and the FSW constant
(shared by DSDcc, MMDVMHost and OP25, per `sync.go`); **unwired** for the
grant path; **absent** for voice. Two numbers in the
tree disagree — 1800 Hz in the pipeline, ±2700 Hz in the capture
documents — and one function admits its convention is pending
(`SymbolToDibit`). Every one of those is settled by the same ≥ 10 s DN-mode
IQ capture, which would be the first real System Fusion signal the package
has ever seen.

### How YSF shaped the Go code

- **Codec and state machine kept apart.** `fich.go` ships bit-level
  `Parse`/`Assemble` so the spec reading can be reviewed independently of
  the FEC math; `fich_trellis.go` adds the channel layer.
- **`ProcessFICH` takes a struct, not bits.** The grant contract is fixed
  now, so closing the join later does not change what the engine sees.
- **`PackBits`/`UnpackBits` as tiny bridges.** The Viterbi output is bits,
  `ParseFICH` wants six octets, and the seam has its own round-trip test.
- **Invariant tests over the tables.** Bijectivity and puncture-count
  checks make the DSDcc swap safe to perform.

## Where this goes next

Three protocols in this series decode voice through an AMBE FEC layer
whose only protocol-specific piece is a 72-bit interleave table, and in
all three that table is a sequential split marked `PLACEHOLDER`.
[Part 13]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }})
reads `voice_ambe.go` for NXDN, dPMR and D-STAR side by side, the composer
chains that feed them, and which DMR lessons — the silence-frame bug, the
Golay histogram, the b0 continuity metric — transfer the day a capture
lands.

## FAQ

**Why does a System Fusion repeater lock in GopherTrunk but never produce a call?**
Because the live pipeline stops at the Frame Sync Word. `newYSFPipeline`
wires the receiver's dibits to `ControlChannel.Process`, which only calls
`maybeLock`. `DecodeFICHOnAir` and `ProcessFICH` exist and are tested, but
their only callers are in `control_test.go`, so no grant, call or
recording follows a lock.

**What is the YSF FICH and how is it protected on air?**
The Frame Information Channel: 32 bits naming the frame type, call type,
block and frame counters, data type and a 7-bit squelch code (DG-ID), plus
a CRC-16-CCITT with init `0x0000`. On air it is K=5 ½-rate trellis
encoded to 104 bits, punctured at `{0, 1, 102, 103}` to 100, and
interleaved column-major 10×10 — the MMDVMHost schedule GopherTrunk ships.

**What does a YSF grant carry once the FICH path is wired?**
`ProcessFICH` publishes a `trunking.Grant` with `Protocol "ysf"` on a
Header frame with `CallTypeGroup`: `GroupID` is the DG-ID when
`SquelchMode` is set, else 0; `SourceID` is 0 because the radio ID lives
in the DCH. Private (RadioID) calls are skipped, and a Terminator clears
the duplicate-suppression state.

**Does GopherTrunk decode System Fusion voice?**
No. YSF maps to `voiceKindUnsupported` in the composer, which logs
"digital protocol not yet decoded; chain bypassed". Unlike NXDN, dPMR and
D-STAR, there is not even a placeholder carve of the 360-dibit DCH; the
voice chain is an implementation gap, not a capture gap.

**What capture would validate the YSF decoder?**
A ≥ 10 s DN-mode IQ recording at 48 kHz or above with a `metadata.json`
listing the expected FICH fields per frame. Audio recordings cannot work
— the 4-level constellation does not survive a discriminator. The pass
bar is 100 % FICH CRC at clean SNR; a failure triggers the two-line swap
to DSDcc's schedule.

## Series navigation

**Part 12 of 14** · ←
[Part 11: D-STAR — GMSK, the 660-Bit Header Shell and the 96-Bit DV Cadence]({{ '/blog/deep-dives/legacy-family-11-dstar/' | relative_url }})
· Next →
[Part 13: Three AMBE Chains With Honest Placeholders — What a Capture Would Pin]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }})
