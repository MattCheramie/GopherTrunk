---
title: "The Legacy Family End to End, Part 11: D-STAR — GMSK, the 660-Bit Header Shell and the 96-Bit DV Cadence"
description: "Inside GopherTrunk's D-STAR package: the 4800 bps GMSK bit chain, the 24-bit Frame Sync 0xEAA060, the 41-byte PCH header and its 660-bit FEC shell (K=5 Viterbi, PN15 scramble, 22×30 interleave), the callsign-driven grant, the Slow-Data-anchored 96-bit DV cadence, and which of those pieces no real capture has ever touched."
category: deep-dives
keywords: d-star decoder go, d-star header fec 660 bits, d-star frame sync 0xeaa060, d-star gmsk 4800 bps receiver, d-star slow data sync 0x552d16, dstar_fec_mode, d-star dv frame 72 voice 24 data, ambe 3600x2400 d-star, d-star callsign routing cqcqcq, sdr d-star decode, gophertrunk legacy family
tags: [legacy-family-end-to-end, d-star, gmsk, ambe, amateur-radio, fec, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 11
---

*Part 11 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out — the FM-era trunking
generation and the AMBE-era narrowband and amateur modes — with each part
naming the verification rung its protocol stands on.
[Part 10]({{ '/blog/deep-dives/legacy-family-10-dpmr-mode-3/' | relative_url }})
closed the narrowband PMR pair with dPMR Mode 3. This part crosses into
amateur radio and a different modulation: D-STAR, the only protocol in
GopherTrunk whose receiver emits bits rather than dibits.
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
placed it at arm's length from verification; this part reads the layouts,
the FEC chain, the state machine and the voice cadence one constant at a
time.*

> **TL;DR:** `internal/radio/dstar` decodes D-STAR DV mode in three
> independent layers. The **receiver** is `demod.FM` → `demod.GFSK`
> (`BT` 0.5) → `sync.MuellerMuller` → a `BitSink`, at `SymbolRate` 4800
> and 10 samples per symbol from the 48 kHz `ddcTargetRateHz`. The
> **header** path finds the 24-bit Frame Sync `0xEAA060` at tolerance 2,
> collects `HeaderBits` (328, `FECOff`, the default) or `FECOnHeaderBits`
> (660, `dstar_fec_mode: "on"`), and under `FECOn` runs
> `framing.DecodeDStarHeaderFEC` — K=5 ½-rate Viterbi (`G1=0x19`,
> `G2=0x17`), PN15 descramble, 22×30 deinterleave — before `ParseHeader`
> and a CRC-16-CCITT check (`ComputeCRC`). A `UR` of `CQCQCQ` or
> `/`-routing becomes a `trunking.Grant` with `Protocol "dstar"`. The
> **voice** path, `VoiceChannel`, anchors a 96-bit DV cadence (72 voice +
> 24 data) on the Slow Data sync `0x552D16` every 21 frames and feeds
> `DecodeDVVoiceBits` → the base `ambe2` (3600×2400) vocoder. Rung:
> polynomials and CRC are reference-matched; scrambler, interleaver and
> AMBE deinterleave are **self-consistent placeholders**; nothing has
> decoded real air.

**Key takeaways**

- **D-STAR is one bit per symbol, so every detector is a bit detector.**
  `dstar.SyncDetector` slides over bits and `BitSink` replaces `DibitSink`.
- **The header shell is complete and self-consistent, not calibrated.**
  `EncodeDStarHeaderFEC` and `DecodeDStarHeaderFEC` invert each other
  exactly; matching MMDVMHost/DSDcc's tables "is a follow-up calibration
  step", which is why `FECOff` is the default.
- **Voice never depends on the header.** `VoiceChannel` re-anchors on
  every Slow Data sync; a lost header costs a grant, not the cadence.
- **The rung is synthetic-green with documented placeholders.** No capture
  exists, `samples/` has no `dstar/` directory, and the AMBE deinterleave is
  labelled `PLACEHOLDER`.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Bits from IQ | FM → Gaussian MF + 2-level slice → MM clock → `BitSink` | `internal/radio/dstar/receiver/receiver.go` (`SymbolRate`, `BT`, `PulseSpanSymbols`) |
| Frame Sync | 24-bit `0xEAA060`, tolerance 2, bit-domain detector | `internal/radio/dstar/sync.go` (`FrameSyncHex`, `SyncDetector`) |
| Header window | 328 bits (`FECOff`) or 660 bits (`FECOn`), latched per match | `internal/radio/dstar/process.go` (`HeaderBits`, `FECOnHeaderBits`, `captureMode`) |
| Header FEC shell | conv K=5 → puncture 4 → PN15 → 22×30 interleave | `internal/radio/framing/dstar_header.go` (`DecodeDStarHeaderFEC`) |
| 41-byte PCH parse | flags, RPT2/RPT1/UR/MY1/MY2, CRC-16-CCITT | `internal/radio/dstar/header.go` (`ParseHeader`, `ComputeCRC`) |
| Lock and grant | first valid header locks; group `UR` publishes `Protocol "dstar"` | `internal/radio/dstar/control.go` (`Ingest`, `hashCallsign`) |
| DV cadence | 96-bit frames anchored on `0x552D16` every 21 frames | `internal/radio/dstar/voice.go` (`VoiceChannel`, `DVFramesPerSyncCycle`) |
| Daemon wiring | `dstar_fec_mode` → `ParseFECMode` → `newDStarPipeline`; `"dstar": "ambe2"` | `internal/scanner/ccdecoder/pipelines.go`, `internal/voice/recorder.go` |

## In this post

- **A receiver that emits bits** — GMSK at 4800 bps and the `BitSink`.
- **The 24-bit sync and the two header windows** — `FECOff`, `FECOn`, the per-match latch.
- **The 660-bit shell, layer by layer** — reference-matched versus self-consistent.
- **Forty-one bytes to a grant** — callsign fields, the CRC and `hashCallsign`.
- **The 96-bit DV cadence** — Slow Data sync anchoring and the voice placeholder.
- **The rung, honestly** — what the tests prove and what a capture would touch first.

## A receiver that emits bits

Every other receiver in the C4FM family hands its consumer dibits. D-STAR
is GMSK at 4800 bps with BT 0.5 — the JARL DV-mode parameters — so one
symbol carries one bit, and `sync.go` defines `BitSink func(bits []byte,
baseIdx int)` instead of borrowing `DibitSink`. `internal/radio/dstar/receiver/receiver.go` composes the
chain from primitives the other receivers already use:

```go
// internal/radio/dstar/receiver/receiver.go (shape)
const (
    SymbolRate       = 4800.0
    BT               = 0.5
    PulseSpanSymbols = 4
)
return &Receiver{
    fm:      demod.NewFM(),
    gfsk:    demod.NewGFSK(int(sps+0.5), span, bt),
    clock:   sync.NewMuellerMuller(sps, gain), // gain defaults to 0.05
    bitSink: opts.BitSink,
}
```

`Process` runs the discriminator, the Gaussian matched filter and the
clock, slices each symbol with `gfsk.Slice`, and calls the sink with a
monotonic `bitBase`. `New` panics below 2 samples per symbol; production
runs 10, because `ddcTargetForProtocol` channelises D-STAR to the same
48 kHz `ddcTargetRateHz` as the 4800-baud C4FM family.

The receiver is pinned the only way a receiver can be without air.
`TestReceiverRecoversModulatedBitStream` modulates a 200-bit pattern with
`demod.ModulateGFSK` at a deviation of `SymbolRate/4` (1200 Hz, modulation
index near 0.5) and demands a ≥ 90 % steady-state match;
`TestReceiverHandlesChunkedIQ` requires ≥ 95 % agreement between a
single-shot and a four-slice decode. Modulator and demodulator share a
package — the pairing the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
postmortem warns about — and the receiver file says so: "real-air
capture-from-RF will need the inner FEC layer later".

## The 24-bit sync and the two header windows

A DV transmission opens with a bit-sync preamble and then the Header Frame
Sync: `FrameSyncHex uint64 = 0xEAA060`, `FrameSyncBits = 24`. The prose in
`sync.go` and `process.go` still says "32-bit" in places, but the
constant, the pattern slice and the detector are 24 bits. The Process
adapter builds its detector lazily with tolerance 2, which the comment
justifies by the pattern being non-periodic: a near-match a few bits off
does not fire, unlike "the 0x55555555 toggle placeholder used to cause".
The fixtures carry the scar: `buildDStarHeaderStream` warms up with 64
**ones**, since zeros or an alternating pattern fire a false sync early.

What follows a match depends on `FECMode`. `fec.go` defines `FECOff`
(read `HeaderBits` = 41 × 8 = 328 information bits off the wire) and
`FECOn` (read `FECOnHeaderBits` = `framing.DStarHeaderChannelBits` = 660
and run the FEC chain first). `ParseFECMode` maps `""`, `off`, `false`,
`0` to `FECOff` and `on`, `true`, `1` to `FECOn`; anything else returns
`FECOff` with `ok = false`, and `newDStarPipeline` logs `ccdecoder:
unrecognised dstar_fec_mode; falling back to off`. The YAML key is
`dstar_fec_mode`; `config.example.yaml` carries it commented out with
"default off — flip on for raw 660-bit".

Two details of `Process` are load-bearing. The mode is **latched per
window** in `captureMode` at the match, so a mid-window flip cannot change
the count under a collection in progress; and a match that fires while
`remaining > 0` is skipped, so a duplicate detection inside a burst cannot
restart the window. Collection precedes the match check in the loop, for
the reason every adapter here shares: a match's index is the *last* bit
of the sync, so the window starts on the next iteration.

## The 660-bit shell, layer by layer

`internal/radio/framing/dstar_header.go` holds the chain, and its constants
read as a derivation: `DStarHeaderInfoBits` = 328 plus
`DStarHeaderTailBits` = 4 gives `DStarHeaderInputBits` = 332; rate ½
doubles that to `DStarHeaderRawChannelBits` = 664; four trailing bits are
punctured to reach `DStarHeaderChannelBits` = 660; and
`DStarHeaderInterleaveRows` × `DStarHeaderInterleaveCols` = 22 × 30 = 660,
an exact fit. The encoder is a four-register shift of the polynomials
`ViterbiK5` decodes:

```go
// internal/radio/framing/dstar_header.go (shape)
// G1=0x19 (1+x^3+x^4), G2=0x17 (1+x+x^2+x^4)
g1 := (input ^ d3 ^ d4) & 1
g2 := (input ^ d1 ^ d2 ^ d4) & 1
channel[2*s], channel[2*s+1] = g1, g2
d4, d3, d2, d1 = d3, d2, d1, input&1
```

Then `scrambleDStarHeader` XORs a PN15 keystream — Galois LFSR,
x¹⁵ + x + 1, register seeded `0x0001`, output from the LSB — and
`interleaveDStarHeader` writes the 660 bits column-major into the 22 × 30
grid and reads them row-major. `DecodeDStarHeaderFEC` runs the inverse
order, then applies the chain's one integrity check: the four recovered
tail bits must be zero, or the survivor did not terminate in the
encoder's zero state and the header is rejected.

Now the honest part, which the file states itself. The polynomials are
the pair MMDVMHost, DSDcc and OpenDV use — **reference-matched**. The
scrambler and interleaver are "self-consistent encode/decode pairs;
matching MMDVMHost / DSDcc's exact permutation tables for live-air decode
is a follow-up calibration step against a captured transmission". The
file's header comment even still describes a "24 × 28 block interleaver …
12 cells unused" while the constants and both grid functions are 22 × 30
— stale prose over code that disagrees with it, and a reminder that
nothing in the tree can say which grid a real repeater uses. Every test of
the shell — `TestDStarHeaderFECRoundTrip`, `TestProcess_DecodesHeaderUnderFECOn`,
`TestProcess_FECOnSurvivesSingleBitError`, `TestDaemonCCDecodesDStarFECOn` —
encodes with `EncodeDStarHeaderFEC`. They prove a bijection that corrects
one error; they cannot prove it is D-STAR's.

## Forty-one bytes to a grant

After the shell comes the structure every open decoder agrees on:

```text
byte  0      FLAG1   Data 0x80 · RepeaterMux 0x40 · Interrupted 0x20
                     Control 0x10 · Urgent 0x08 · EMR 0x04 · BreakIn 0x02
bytes 1-2    FLAG2, FLAG3
bytes 3-10   RPT2    destination repeater (8 chars, space-padded)
bytes 11-18  RPT1    gateway / source repeater
bytes 19-26  UR      "CQCQCQ" = group · "/<rpt>" = routing · else a station
bytes 27-34  MY1     own callsign
bytes 35-38  MY2     4-char suffix
bytes 39-40  CRC     CRC-16-CCITT (0x1021, init 0xFFFF) over bytes 0..38
```

`ParseHeader` preserves the space padding; `IsGroupCall` trims before
comparing. `ComputeCRC` is the one value pinned against an independent
vector:
`TestComputeCRCKnownVector` asserts `ComputeCRC([]byte("123456789"))` is
`0x29B1`, and `parseHeader` compares the trailer to
`ComputeCRC(bytesOut[:39])` before anything reaches the state machine
(`TestProcess_RejectsBadCRC` flips the final bit and demands silence).

`control.go` turns the header into engine events. `Ingest` drops a
`Flag1Data` header, then `maybeLock`s with `LockState{FrequencyHz,
Repeater: RPT2 trimmed}` — re-published only when that value changes —
and `LockedNAC` packs the callsign's first two bytes into the NAC slot. A
group call publishes:

```go
// internal/radio/dstar/control.go (shape)
trunking.Grant{
    System:      c.systemName,
    Protocol:    "dstar",
    GroupID:     hashCallsign(ur),   // up to 4 ASCII bytes packed
    SourceID:    hashCallsign(src),  // MY1
    FrequencyHz: c.freqHz,           // conventional: same carrier
    Emergency:   h.IsEmergency(),
}
```

`hashCallsign` packs up to four characters into a `uint32`; an individual
`UR` produces no grant (`TestControlChannelSilentOnIndividualCall`). D-STAR
is conventional, so the "grant" tells the engine a transmission has
started on the carrier it already camps on.

## The 96-bit DV cadence

After the header, a DV transmission is a free-running sequence of 96-bit
frames: `DVVoiceBits` = 72 bits of AMBE, then `DVDataBits` = 24 bits of
slow data. Frame 0 of every `DVFramesPerSyncCycle` = 21 carries the Slow
Data sync `SlowDataSyncHex` = `0x55_2D16` in its data field, and
`voice.go` builds the whole tracker on that fact: a sync match ending at
bit *p* means bits [p−95, p−24] were that frame's voice payload and the
next frame starts at p+1.

`VoiceChannel` keeps a rolling 96-bit `recent` window so the sync's own
frame decodes from history, free-runs between syncs, re-anchors
**unconditionally** on every match so a slip recovers within one cycle,
and drops the anchor after `2*DVFramesPerSyncCycle` sync-less frames so
noise after the transmission stops emitting garbage. A `lastFrameEnd`
guard stops the sync branch re-emitting a frame the free-run just produced. The tests enumerate exactly those behaviours:
`TestVoiceChannelDecodesAnchoredCadence` (22 frames, the boundary frame
once), `TestVoiceChannelDropsAnchorWithoutSync` (1 + 42 frames, then
nothing, then one more on a fresh sync),
`TestVoiceChannelSurvivesChunkedFeed` and
`TestVoiceChannelResyncsAfterBitSlip`.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A row of 96-bit DV frames, each a 72-bit voice field then a 24-bit data field; frames 0 and 21 carry the Slow Data sync in the data field, and a bracket under frame 0 marks the voice bits p minus 95 to p minus 24 relative to the sync end p.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">DV cadence: 96 bits per frame = 72 voice + 24 data · sync every 21 frames</text>
  <rect x="20" y="40" width="108" height="26" fill="none" stroke="currentColor"/>
  <rect x="128" y="40" width="36" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="74" y="57" text-anchor="middle" fill="currentColor" font-size="8">frame 0 · 72 voice</text>
  <text x="146" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">sync</text>
  <rect x="174" y="40" width="108" height="26" fill="none" stroke="currentColor"/>
  <rect x="282" y="40" width="36" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="228" y="57" text-anchor="middle" fill="currentColor" font-size="8">frame 1 · 72 voice</text>
  <text x="300" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">data</text>
  <text x="352" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">…</text>
  <rect x="386" y="40" width="108" height="26" fill="none" stroke="currentColor"/>
  <rect x="494" y="40" width="36" height="26" fill="none" stroke="var(--fg-muted)"/>
  <text x="440" y="57" text-anchor="middle" fill="currentColor" font-size="8">frame 20 · 72 voice</text>
  <text x="512" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="8">data</text>
  <rect x="540" y="40" width="96" height="26" fill="none" stroke="currentColor"/>
  <rect x="636" y="40" width="36" height="26" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="588" y="57" text-anchor="middle" fill="currentColor" font-size="8">frame 21 · voice</text>
  <text x="654" y="57" text-anchor="middle" fill="var(--accent)" font-size="8">sync</text>
  <path d="M20 76 L20 84 L128 84 L128 76" fill="none" stroke="var(--accent)"/>
  <text x="74" y="98" text-anchor="middle" fill="var(--accent)" font-size="8">bits [p−95, p−24] when the sync ends at bit p</text>
  <text x="340" y="130" text-anchor="middle" fill="currentColor" font-size="8">sync 0x552D16 · tolerance 1 · re-anchor on every match · anchor dropped after 2 × 21 sync-less frames</text>
  <text x="340" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="8">72 voice bits → DecodeDVVoiceBits → 49-bit ambe_d → "ambe2" (3600×2400)</text>
</svg>
<figcaption>The voice tracker needs only the Slow Data sync. The header decoder and the DV cadence never share state, so a header that fails its FEC or CRC costs the grant, not the audio.</figcaption>
</figure>

Each 72-bit voice field goes to `DecodeDVVoiceBits` in `voice_ambe.go`:
Golay(23,12) over C0, a C0-seeded keystream XORed onto C1 before its own
Golay, and the 12 + 12 + 11 + 14 = 49-bit `ambe_d` assembly — the layer
DMR's `DecodeAMBEFrame` runs too, because mbelib's 3600×2400 and
3600×2450 ECC paths share it. D-STAR's vocoder is the original AMBE
3600×2400, which is what `internal/voice/ambe2`'s base decoder
implements, so `DefaultVocoderForProtocol` maps `"dstar"` to `"ambe2"`.
The one protocol-specific piece — how 72 on-air bits map into C0..C3,
DSD's `dW`/`dX` schedule — is `dstarAMBEDeinterleave`, a direct sequential
split marked `PLACEHOLDER`.
[Part 13]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }})
takes that function, its two siblings and `runDStarVoiceChain` apart.

## The rung, honestly

Lay the package on the ladder and the picture is precise.
**Reference-matched**: the K=5 polynomial pair, the CRC (the `0x29B1`
vector), the header layout and the sync constants (the values "emitted by
MMDVMHost / DSDcc / OpenDV transmitters", per `sync.go`).
**Self-consistent, uncalibrated**: the PN15 seed and taps, the 22 × 30
grid orientation, the slicer's bit sense, and the AMBE `dW`/`dX`
deinterleave. **Synthetic end to end**: `TestDaemonCCDecodesDStar` and
`TestDaemonCCDecodesDStarFECOn` boot the daemon on modulated fixtures, and
v1.1.2 shipped the voice chain "experimental, unverified on air". **On
air**: nothing — no `samples/dstar/` directory, no skip-gated real-air
test; `docs/decoder-capture-needs.md` lists "dPMR / D-STAR voice (≥ 10 s
IQ, 48 kHz, clear voice)" as the capture that "confirms the placeholder
AMBE interleave tables + frame geometry".

That capture would be unusually informative because the layers fail
independently: a passing CRC under `dstar_fec_mode: "on"` settles the
scrambler and interleaver at once; Slow Data syncs arriving every
21 × 96 = 2016 bits settle the cadence without the header; and the
per-frame Golay counts the chain already reports (`DVFrame.Errors`) say
whether the deinterleave is right before anyone listens to a WAV. Until
then, `status.md`'s wording stands: D-STAR is "wired end-to-end through
the composer but not yet verified on air".

### How D-STAR shaped the Go code

- **A bit-domain detector beside the dibit ones.** `dstar.SyncDetector`
  keeps the other packages' shape over bits.
- **`FECMode` latched per window.** `captureMode` is sampled at the match,
  so `SetFECMode` never desynchronises a collection in progress.
- **Lock state is a value.** `maybeLock` compares `LockState` structs: a
  repeater change re-publishes, a repeated header does not.
- **Voice and header are peers.** `VoiceChannel` and `ControlChannel`
  consume the same `BitSink` independently.

## Where this goes next

D-STAR's amateur sibling runs the opposite physical layer — 4800-baud
C4FM under a different sync word — and a different incompleteness: a FICH
trellis codec that is built, reference-pinned and exhaustively tested, yet
has no caller on the hot path.
[Part 12]({{ '/blog/deep-dives/legacy-family-12-ysf/' | relative_url }})
reads Yaesu System Fusion's frame, FICH and trellis, the DN/VW modes, and
why its voice is not yet PCM.

## FAQ

**What does GopherTrunk decode from a D-STAR transmission today?**
On synthetic fixtures, the whole DV header — FLAG1 bits, RPT2, RPT1, UR,
MY1, MY2, CRC-16-CCITT checked. A group `UR` (`CQCQCQ` or `/`-routing)
becomes a `trunking.Grant` with `Protocol "dstar"` and callsign-hashed
IDs; voice decodes through `DecodeDVVoiceBits` to the `ambe2` vocoder. No
real-air capture has exercised any of it.

**Why is `dstar_fec_mode` off by default?**
Because the shell's scrambler and interleaver are self-consistent pairs
whose match to MMDVMHost/DSDcc's tables is "a follow-up calibration
step". `FECOff` reads 328 information bits — right for fixtures and
pre-decoded inputs — while `dstar_fec_mode: "on"` selects the 660-bit
window and `framing.DecodeDStarHeaderFEC`, which a live JARL header needs.

**What is the D-STAR Slow Data sync for in GopherTrunk?**
It anchors the voice cadence. `VoiceChannel` matches `0x552D16` at
tolerance 1; a match ending at bit p marks bits [p−95, p−24] as that
frame's 72 voice bits and restarts the 96-bit free-run, dropping the
anchor after 42 sync-less frames. The header decoder is never consulted,
so a lost header does not lose the audio.

**Which vocoder renders D-STAR voice, and why not `ambe2-dmr`?**
D-STAR carries the original AMBE 3600×2400, and `internal/voice/ambe2`'s
base decoder ports mbelib's `ambe3600x2400.c`, so the recorder maps
`"dstar"` to `"ambe2"`. `"ambe2-dmr"` is the 3600×2450 variant DMR, NXDN
and dPMR use, with a different bit layout and codebooks.

## Series navigation

**Part 11 of 14** · ←
[Part 10: dPMR Mode 3 — FS1/FS2 Syncs, CSBK Trunking and the 6.25 kHz Channel]({{ '/blog/deep-dives/legacy-family-10-dpmr-mode-3/' | relative_url }})
· Next →
[Part 12: Yaesu System Fusion — FICH Trellis, Frame Types and the Voice That Is Not Yet PCM]({{ '/blog/deep-dives/legacy-family-12-ysf/' | relative_url }})
