---
title: "Beyond Voice, Part 13: D-STAR, System Fusion & dPMR"
description: "Three AMBE-era digital modes at three distances from on-air verification: D-STAR's GMSK header behind a 660-bit FEC shell, System Fusion's FICH trellis codec that is built but not yet on the hot path, and dPMR Mode 3's CSBK trunking — plus the two voice chains whose interleave tables are honest placeholders."
category: deep-dives
keywords: d-star decoder sdr, d-star header fec, system fusion ysf decoder, ysf fich trellis, dpmr mode 3 decoder, dpmr csbk, ambe 3600x2400 d-star, gmsk 4800 bps decoder, amateur digital voice sdr, gophertrunk d-star ysf dpmr
tags: [beyond-voice, d-star, system-fusion, dpmr, ambe, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 13
---

*Part 13 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats,
paging, APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice
family — and the one eleven-place wiring pattern that carries each of them
from a burst on the air to a row in the web console.
[Part 12]({{ '/blog/deep-dives/beyond-voice-12-m17-open-digital-voice/' | relative_url }})
took the open end of digital voice; this part takes the three modes built
on AMBE — D-STAR, Yaesu System Fusion and ETSI dPMR — and is precise about
what each decodes today, what is built but unwired, and which tables are
placeholders waiting for a capture.*

> **TL;DR:** Three packages, one shape, three verification states.
> **D-STAR** (`internal/radio/dstar`): GMSK at 4800 bps, BT 0.5, a 24-bit
> Frame Sync `0xEAA060`, a 41-byte PCH header (flags, RPT2, RPT1, UR, MY1,
> MY2, CRC-CCITT) whose 660-bit FEC shell — `framing.DecodeDStarHeaderFEC`:
> K=5 ½-rate Viterbi, PN15 descramble, 22×30 deinterleave — runs under
> `dstar_fec_mode: "on"`; a `CQCQCQ` or `/`-routed UR becomes a
> `trunking.Grant` with `Protocol "dstar"`.
> **YSF** (`internal/radio/ysf`): C4FM 4800 baud, 40-bit FSW `0xD471C9634D`,
> a full FICH codec (`DecodeFICHOnAir`: 10×10 deinterleave, puncture
> `{0,1,102,103}`, `ViterbiK5`, CRC init `0x0000`) that **only tests call**
> — the live pipeline locks on the FSW and stops. **dPMR Mode 3**
> (`internal/radio/dpmr`): 4FSK at 2400 sym/s, FS3 → 80-bit CSBK → grants
> resolved through a `LinearBandPlan`; the CSBK FEC is not implemented.
> The D-STAR and dPMR voice chains (v1.1.2) decode AMBE 3600×2400 and AMBE+2
> 3600×2450 on placeholder deinterleave tables — **unverified on air**.

**Key takeaways**

- **The same sync → slice → parse → grant shape covers all three.**
  `SyncDetector`, a countdown `Process` adapter, a `ControlChannel`
  publishing lock and grant — D-STAR merely runs it on bits.
- **D-STAR's FEC chain is complete but self-consistent.** Matching
  MMDVMHost's exact scrambler and interleaver tables "is a follow-up
  calibration step".
- **YSF is one wiring step from a real decoder.** `DecodeFICHOnAir`
  survives every single-bit flip, yet `ProcessFICH` has no caller outside
  its tests.
- **A placeholder table is a placeholder, not a guess.** Both voice chains
  deinterleave with a sequential split labelled `PLACEHOLDER`, so the swap
  is one function — and the first capture confirms it.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| D-STAR bits | FM → Gaussian matched filter → MM timing → 2-level slice | `internal/radio/dstar/receiver/receiver.go` |
| D-STAR header | Frame Sync `0xEAA060` → 328 or 660 bits → `ParseHeader` | `internal/radio/dstar/process.go`, `header.go` |
| YSF frame + FICH | FSW 20 dibits, FICH 100, DCH 360 | `internal/radio/ysf/frame.go`, `fich.go`, `fich_trellis.go` |
| dPMR CSBK | FS3 → 40 dibits → `CSBKFromBits` → `Ingest` → grant | `internal/radio/dpmr/process.go`, `control.go` |
| dPMR channel → Hz | `LinearBandPlan` / `TableBandPlan` | `internal/radio/dpmr/bandplan.go` |
| Voice chains | `VoiceChannel` (96-bit DV) / `TrafficChannel` (4×72 TCH) | `dstar/voice.go`, `dpmr/traffic.go`, `internal/voice/recorder.go` |

## In this post

- **One shape, three modulations** — where the protocols enter.
- **D-STAR** — the GMSK header and its 660-bit shell.
- **System Fusion** — a FICH codec waiting for its caller.
- **dPMR Mode 3** — CSBKs, band plans and the FEC that is not there.
- **The voice chains** — two AMBE paths and their placeholder tables.

## One shape, three modulations

All three are trunking-engine protocols rather than message decoders:
`ProtocolDPMR`, `ProtocolYSF` and `ProtocolDStar` live in
`internal/trunking/site.go`, each has a `ccdecoder` pipeline factory, and
all three run at the 48 kHz `ddcTargetRateHz` the C4FM family shares. That is the
shape
[Protocol Decoders Part 1]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
laid down: a `SyncDetector` finds frame boundaries, a `Process` adapter
counts out the payload after each match, a `ControlChannel` parses it and
publishes `events.KindCCLocked` and `events.KindGrant`.

What differs is the physics. YSF is 4800-baud C4FM with RRC α = 0.20 — the
P25 Phase 1 receiver with a different sync word. dPMR is the same 4-level
scheme at **2400 sym/s** with 900 Hz peak deviation. D-STAR is **GMSK at
4800 bps, BT 0.5**: one bit per symbol, so its receiver emits a `BitSink`.
D-STAR and YSF are conventional — a "grant" announces a transmission on the
monitored repeater; only dPMR Mode 3 has a real control channel and a band
plan.

## D-STAR: a GMSK header and its 660-bit shell

`internal/radio/dstar/receiver` composes `demod.FM` → `demod.GFSK` (a
Gaussian matched filter and 2-level slicer) → `sync.MuellerMuller` →
`dstar.BitSink` ([reference]({{ '/reference/gmsk/' | relative_url }})).
`ControlChannel.Process` runs a `SyncDetector` on the 24-bit Header Frame
Sync at tolerance 2, then counts down a header window whose size `FECMode`
chooses: `HeaderBits` (328) under `FECOff`, or
`framing.DStarHeaderChannelBits` (660) under `FECOn`, where
`framing.DecodeDStarHeaderFEC` recovers the 41 bytes first. Either way
`ParseHeader` and a `ComputeCRC` check follow.

`FECOff` is the default and reads 328 information bits straight off the
wire — right for synthesised fixtures. `FECOn`, selected per system with
`dstar_fec_mode: "on"` and threaded through `dstar.ParseFECMode` in
`newDStarPipeline`, is "the path that lights up on a live-air capture". Its chain
([reference]({{ '/reference/dstar-header-fec/' | relative_url }})): 328
info bits plus a 4-bit tail → K=5 rate-½ convolutional code (`G1=0x19`,
`G2=0x17`, the `ViterbiK5` pair) → 664 channel bits → puncture four → 660 →
PN15 scramble → 22×30 block interleave, inverted on receive;
`TestProcess_FECOnSurvivesSingleBitError` and
`TestDaemonCCDecodesDStarFECOn` pin it. The file's own caveat: the
scrambler and interleaver are "self-consistent encode/decode pairs" whose
match to MMDVMHost's exact tables "is a follow-up calibration step" — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
named in advance, and why `FECOff` stays the default.

The 41 recovered bytes are the PCH header
([reference]({{ '/reference/d-star/' | relative_url }})): three flag
bytes, four 8-character callsigns — `RPT2`, `RPT1`, `UR`, `MY1` — a
4-character `MY2`, and a CRC-16-CCITT (poly `0x1021`, init `0xFFFF`),
pinned by `TestComputeCRCKnownVector` to the standard `"123456789"` →
`0x29B1` vector. `Ingest` decides everything from the header: a data header
(`Flag1Data`) is ignored; the first valid header locks with `RPT2` as the
repeater; a `UR` of `CQCQCQ` or any `/`-prefixed routing publishes a grant
with `Protocol "dstar"`, the trimmed `UR` hashed into `GroupID` and `MY1`
into `SourceID` (`hashCallsign`). `LockedNAC` packs two bytes of the RPT2
callsign into the NAC slot as a stable per-repeater identity.

## System Fusion: a FICH codec waiting for its caller

YSF's frame is 480 dibits — 100 ms at 4800 baud
([reference]({{ '/reference/system-fusion-ysf/' | relative_url }})): 20
dibits of Frame Sync Word (`FSWBits` `0xD471C9634D`), 100 of FICH, 360 of
DCH. `ysf.SyncDetector` matches the 20-dibit `FSWPattern` within a
tolerance of 2, and `ControlChannel.Process` does one thing with a hit:
`maybeLock`, logging `ysf cc locked`. That is the whole live path.

The FICH is where the interesting code sits
([reference]({{ '/reference/ysf-fich/' | relative_url }})): 32 information
bits — frame type, call type, block and frame counters, data type, squelch
mode and a 7-bit squelch code — plus a CRC-CCITT computed with
`framing.CRCCCITTWithInit(b[:4], 0x0000)`, the XMODEM initial value
(`TestFICHCRCUsesXMODEMInit`). On the air those 48 bits
pass through a K=5 ½-rate trellis with four tail bits (`FICHChannelBits`
104), lose the punctured positions `{0, 1, 102, 103}` (`FICHOnAirBits` 100)
and are permuted by a column-major 10×10 interleaver — MMDVMHost's
schedule. `DecodeFICHOnAir` inverts it: deinterleave, depuncture with
`framing.DepunctureMark`, `framing.ViterbiK5` to 48 info bits plus a metric. `TestFICHOnAirRecoversFromSingleBitFlip` confirms every one of the
100 on-air positions is corrected. And `ControlChannel.ProcessFICH` knows
what to do with a decoded FICH: a Header frame with `CallTypeGroup`
publishes a `trunking.Grant` with `Protocol "ysf"` and `GroupID` = the
squelch code (the DG-ID) when `SquelchMode` is set, else 0; a Terminator
clears the dedup state.

The gap is the join. `newYSFPipeline` wires the receiver's `DibitSink` to
`cc.Process` only; nothing slices the 100-dibit FICH region, runs
`DecodeFICHOnAir` and calls `ProcessFICH` — its only callers are in
`control_test.go`. So a live YSF repeater locks, and no grant, call or
recording follows; the control comment says so: "A future PR closes the IQ
→ dibit → FICH gap on the hot path". `samples/ysf/README.md` names the
fallback if MMDVMHost's schedule fails a real FICH CRC: DSDcc's spread
puncture `{0, 51, 52, 103}`.

## dPMR Mode 3: CSBKs, band plans and the FEC that is not there

[Protocol Decoders Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
introduced dPMR's three sync words and its 80-bit CSBK.
`internal/radio/dpmr/receiver` is the C4FM chain at
`SymbolRate` 2400 with `DeviationHz` 900
([reference]({{ '/reference/dpmr/' | relative_url }})). The control path
in `process.go` runs one `SyncDetector` on `FS3Dibits()` at tolerance 1
and, after each match, collects 40 dibits, converts them with
`framing.DibitsToBits`, parses with `CSBKFromBits` and calls `Ingest`
([sync]({{ '/reference/dpmr-frame-sync/' | relative_url }}),
[CSBK]({{ '/reference/dpmr-csbk/' | relative_url }})).

`Ingest` is the state machine: `MsgIdle` is dropped;
`MsgStandingServiceStatus` locks the channel with the broadcast's `DestID`
as `SystemID` (`dpmr cc locked`); `MsgVoiceServiceAllocation` or
`MsgIndividualVoiceAllocation` locks if needed and publishes a grant. The 16-bit opcode-specific field carries the channel
number, and a `Resolver` turns it into hertz: `LinearBandPlan` applies
`BaseHz + (channel + Offset) × SpacingHz` — for PMR446, base 446 006 250 Hz,
6 250 Hz spacing, offset −1 — and `TableBandPlan` maps explicit entries.
With no resolver the grant carries `FrequencyHz` 0.
`SetStrictValidation` drops any CSBK whose 5-bit type is outside the
documented set (`TestStrictValidationDropsUnknownMessageType`), the defence
against a misaligned-but-passing window producing a phantom grant.

What is *not* there is the channel coding
([reference]({{ '/reference/dpmr-channel-coding/' | relative_url }})): Mode
3 CSBKs carry a short-block cyclic code under a rate-¾ convolutional outer
code plus an interleaver, and the package doc says "the parsing here assumes
the upstream caller has already corrected errors". The 40 dibits after FS3
are read as 80 clean bits — which is why strict validation exists and why
`TestDaemonCCDecodesDPMR` is a synthetic lock, not a field result.

## The voice chains and their placeholder tables

Both voice chains landed in v1.1.2 (2026-09-12) labelled "experimental,
unverified on air".

**D-STAR** voice is the original AMBE 3600×2400 — exactly the base
`internal/voice/ambe2` decoder, so the recorder's map has `"dstar": "ambe2"`
([Voice Coding]({{ '/blog/deep-dives/sdr-internals-12-voice-coding-vocoders/' | relative_url }}),
[AMBE]({{ '/reference/ambe/' | relative_url }})). `VoiceChannel` free-runs a
96-bit DV cadence — 72 voice bits then 24 slow-data bits — anchored on the
24-bit Slow Data sync (`0x552D16`) that every 21st DV frame carries; it
re-anchors on every match and drops the anchor after two sync-less cycles.
`DecodeDVVoiceBits` then runs the AMBE FEC shared with DMR and NXDN:
Golay(23,12) over C0, a C0-seeded keystream XORed onto C1 before its own
Golay, and the 12 + 12 + 11 + 14 = 49-bit `ambe_d` assembly.

**dPMR** voice is AMBE+2 3600×2450 — the DMR/NXDN codec — so `"dpmr"` maps to
`"ambe2-dmr"` ([AMBE+2 FEC]({{ '/reference/ambe-plus-2-fec/' | relative_url }}))
— v1.1.2 fixed a map that had pointed at the 2400 codebook.
`TrafficChannel` runs FS1 and FS2 detectors, collects 168 dibits per 80 ms
frame — 24 of CCH, skipped, then 144 of TCH — and `ExtractTCHFrames` carves
four 72-bit AMBE+2 frames. `runDPMRVoiceChain` and `runDStarVoiceChain`
decimate to 48 kHz and feed the recorder 7-byte frames.

The placeholder is the same in both: the function that maps 72 on-air bits
into the C0..C3 sub-vectors. `dstarAMBEDeinterleave` and
`dpmrAMBEDeinterleave` are direct sequential splits — C0 = bits 0..23,
C1 = 24..46, C2 = 47..57, C3 = 58..71 — each marked `PLACEHOLDER`, to be
replaced "once a capture is available to verify against".
`TestDVVoiceBitsRoundTrip` and `TestTCHFrameRoundTrip` are green — and
prove only that the encoder inverse matches the decoder. The source cites
CLAUDE.md's TETRA-CRC lesson for why the real tables are not guessed: a
wrong table that round-trips is the worst bug, because nothing in the tree
can disagree with it
([From Spec to Shipping Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})).
A voice capture confirms the table first and, for dPMR, the 2450-versus-2400
codebook second.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Three lanes, one per protocol. D-STAR: GMSK bits and sync, header with optional FEC, grant, then a dashed DV cadence and AMBE voice. YSF: C4FM dibits and FSW, lock, then a dashed FICH codec with no caller and dashed undecoded voice. dPMR: 4FSK and FS3, CSBK and band plan, grant, then dashed traffic frames and AMBE+2 voice. Solid boxes are wired; dashed boxes are built but unwired or placeholder-backed.">
  <text x="8" y="41" fill="currentColor" font-size="9" font-weight="bold">D-STAR</text>
  <rect x="62" y="24" width="140" height="28" fill="none" stroke="currentColor"/>
  <text x="132" y="42" text-anchor="middle" fill="currentColor" font-size="8">GMSK bits · sync 0xEAA060</text>
  <rect x="212" y="24" width="150" height="28" fill="none" stroke="currentColor"/>
  <text x="287" y="42" text-anchor="middle" fill="currentColor" font-size="8">41-byte header · FEC 660→328</text>
  <rect x="372" y="24" width="60" height="28" fill="none" stroke="var(--accent)"/>
  <text x="402" y="42" text-anchor="middle" fill="var(--accent)" font-size="8">grant</text>
  <rect x="442" y="24" width="230" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="557" y="42" text-anchor="middle" fill="var(--fg-muted)" font-size="8">96-bit DV cadence → AMBE 2400</text>
  <text x="8" y="111" fill="currentColor" font-size="9" font-weight="bold">YSF</text>
  <rect x="62" y="94" width="140" height="28" fill="none" stroke="currentColor"/>
  <text x="132" y="112" text-anchor="middle" fill="currentColor" font-size="8">C4FM dibits · FSW</text>
  <rect x="212" y="94" width="60" height="28" fill="none" stroke="var(--accent)"/>
  <text x="242" y="112" text-anchor="middle" fill="var(--accent)" font-size="8">lock</text>
  <rect x="282" y="94" width="200" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="382" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FICH codec — no caller</text>
  <rect x="492" y="94" width="180" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="582" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DCH voice — not decoded</text>
  <text x="8" y="181" fill="currentColor" font-size="9" font-weight="bold">dPMR</text>
  <rect x="62" y="164" width="140" height="28" fill="none" stroke="currentColor"/>
  <text x="132" y="182" text-anchor="middle" fill="currentColor" font-size="8">4FSK 2400 sym/s · FS3</text>
  <rect x="212" y="164" width="150" height="28" fill="none" stroke="currentColor"/>
  <text x="287" y="182" text-anchor="middle" fill="currentColor" font-size="8">CSBK · band plan</text>
  <rect x="372" y="164" width="60" height="28" fill="none" stroke="var(--accent)"/>
  <text x="402" y="182" text-anchor="middle" fill="var(--accent)" font-size="8">grant</text>
  <rect x="442" y="164" width="230" height="28" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="557" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FS1/FS2 → 4×72 TCH → AMBE+2 2450</text>
  <text x="340" y="228" text-anchor="middle" fill="currentColor" font-size="9">solid: wired, synthetic-verified · dashed: unwired or placeholder-backed</text>
</svg>
<figcaption>Three lanes, one shape. Every solid box is exercised by the daemon today; every dashed box is code that exists — a FICH codec with no caller, two voice chains on placeholder tables — and waits for an IQ capture.</figcaption>
</figure>

### How these three shaped the Go code

- **`FECMode` as a per-window latch.** `Process` samples the D-STAR FEC
  mode at each Frame Sync match, so a mid-window flip cannot desynchronise
  the bit count.
- **Voice trackers independent of the header.** `VoiceChannel` and
  `TrafficChannel` anchor on their own syncs; a lost header costs a grant,
  never the audio cadence.
- **Placeholders isolated to one function each**, with encoder inverses in
  lock-step, so the capture-driven swap is a one-function change.
- **Strictness as a knob, not a default.** `SetStrictValidation` and the
  `FECOff` default keep fixtures green with the live-air posture one config
  line away.

## Where this goes next

Twelve physical layers later, the pattern is the story.
[Part 14]({{ '/blog/deep-dives/beyond-voice-14-adding-a-decoder-checklist/' | relative_url }})
closes the series with the checklist: the eleven places a new decoder
touches, the three tests that police them, and the honest open list —
including the FICH caller and the placeholder tables above.

## FAQ

**Does GopherTrunk decode D-STAR voice?**
Experimentally. The `dstar` voice chain anchors a 96-bit DV cadence on the
Slow Data sync, FEC-decodes 72-bit AMBE frames and renders them through the
base `ambe2` (3600×2400) vocoder. Its deinterleave table is a documented
placeholder validated only by synthetic round-trip, so the path is
unverified on air until a real capture lands.

**What does `dstar_fec_mode` do?**
It selects how many bits follow a Frame Sync. Off (the default) reads 328
information bits and parses them directly — right for pre-decoded fixtures.
On reads the 660 on-wire bits and runs `framing.DecodeDStarHeaderFEC`
(deinterleave, PN15 descramble, depuncture, K=5 Viterbi) first, which a
live JARL DV-mode header needs.

**Why does a System Fusion repeater lock but never record?**
Because the live YSF pipeline stops at the Frame Sync Word. The FICH codec
(`DecodeFICHOnAir`) and the grant logic (`ProcessFICH`) both exist and are
tested, but nothing on the hot path slices the FICH region and joins them,
so YSF produces `cc.locked` and nothing else until that wiring lands.

**How does dPMR turn a channel number into a frequency?**
Through a `Resolver`. `LinearBandPlan` applies base + (channel + offset) ×
spacing — for PMR446, base 446.006250 MHz, 6.25 kHz spacing, offset −1 so
channel 1 lands on the base — and `TableBandPlan` maps explicit entries.
Without a resolver the grant carries frequency 0.

**Are the D-STAR and dPMR AMBE interleave tables correct?**
Unknown — that is the point of labelling them placeholders. Both use a
sequential C0|C1|C2|C3 split so the FEC machinery can be exercised end to
end; the real D-STAR dW/dX schedule and the dPMR TS 102 658 §6 table await a
capture, because a guessed table that round-trips would hide the bug.

## Series navigation

**Part 13 of 14** · ←
[Part 12: M17 — The Open Digital Voice Link Layer]({{ '/blog/deep-dives/beyond-voice-12-m17-open-digital-voice/' | relative_url }})
· Next →
[Part 14: Adding a Decoder in a Day — The Checklist]({{ '/blog/deep-dives/beyond-voice-14-adding-a-decoder-checklist/' | relative_url }})
