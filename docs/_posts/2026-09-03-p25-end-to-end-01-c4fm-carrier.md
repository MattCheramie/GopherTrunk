---
title: "P25 End to End, Part 1: C4FM & the Shape of a P25 Carrier"
description: What a P25 Phase 1 carrier actually is — 4800 symbols per second of four-level FSK in a 12.5 kHz channel, dibits riding on ±600 and ±1800 Hz deviations, and the FM-discriminator → matched-filter → slicer chain GopherTrunk sizes around a 48 kHz channel rate.
category: deep-dives
keywords: p25 c4fm modulation, c4fm four level fsk, p25 4800 symbols per second, p25 deviation 1800 hz, fm discriminator demodulation, p25 phase 1 decoder, c4fm matched filter, p25 12.5 khz channel, gophertrunk p25
tags: [p25-end-to-end, p25, c4fm, dsp, demodulation, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 1
---

*Part 1 of **P25 End to End**, a 14-part deep dive that follows North America's
dominant trunking protocol through GopherTrunk — from a raw C4FM carrier to
recorded, named, multi-site voice. Where
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }})
followed one European carrier, this series follows the protocol GopherTrunk
locks most often — and its running thread is that P25 is a *family* of twins:
Phase 1 and Phase 2, C4FM and CQPSK, single-channel and wideband, live and
replay. Every twin pair is a place where a fix can land on one side and miss
the other — the
[Two Pipelines lesson]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}),
applied layer by layer. This opener is the physical layer: what a P25 carrier
actually is, and why information riding in *amplitudes* — not phase
transitions — shapes everything downstream.*

> **TL;DR:** P25 Phase 1 is **C4FM: four-level FSK at 4800 symbols/s** in a
> 12.5 kHz channel — one dibit per symbol, carried as a deviation of ±600 or
> ±1800 Hz. An FM discriminator turns that into a four-level waveform, a
> matched filter cleans it, and a slicer maps {−3, −1, +1, +3} to dibits
> (`phase1.SymbolToDibit`). GopherTrunk channelizes the 4800-baud family to
> **48 kHz — 10 samples/symbol** (`ddcTargetForProtocol`,
> `internal/scanner/ccdecoder/ddc.go`), and the receiver
> (`internal/radio/p25/phase1/receiver`) is a discriminator → spec matched
> filter → CoarseAFC → Mueller-Müller → slicer chain sized entirely from that
> rate. One early surprise: the matched filter is **not an RRC** — real P25
> shapes with a raised-cosine plus inverse-sinc, and modelling it as RRC was
> self-consistent in tests while leaving residual ISI on every real capture
> (issue #275).

**Key takeaways**

- **The dibit lives in the amplitude, not the transition.** After the
  discriminator a C4FM symbol is a level — an absolute question, where TETRA
  asks a relative one. That decides what the chain needs: level calibration,
  offset control and an AFC, rather than a rotation-tolerant differential
  decode.
- **C4FM is not an RRC matched-pair system.** The transmitter shapes with a
  raised-cosine cascaded with an inverse-sinc; the correct receive filter is
  a sinc (`demod.P25C4FMRxTaps`). The original RRC model passed every
  synthetic test and failed real captures — the series' first
  self-consistency trap.
- **A carrier offset is a slicer bias, not a nuisance.** The discriminator
  maps frequency to amplitude, so tuner error becomes a DC shift that pushes
  inner symbols over outer thresholds — why this chain carries a `CoarseAFC`
  stage TETRA's differential path never needed.
- **48 kHz is a design rate, not a detail.** Ten samples per symbol sizes the
  matched filter, the AFC time constants and the Mueller-Müller loop. Feed
  the receiver raw 2.048 MHz IQ and the frame sync word never correlates —
  the failure mode issue #275 started from.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| FM discriminator | IQ → per-sample frequency (rad/sample) | `internal/dsp/demod/fm.go` (`demod.FM`) |
| Matched filter | spec C4FM receive filter (sinc, not RRC) | `internal/dsp/demod/c4fm_p25.go` (`P25C4FMRxTaps`, `NewC4FMP25`) |
| Carrier-offset removal | tracks/subtracts the discriminator DC bias | `internal/dsp/demod/afc.go` (`demod.CoarseAFC`) |
| Symbol timing | Mueller-Müller clock recovery on the real waveform | `internal/dsp/sync` (`sync.MuellerMuller`) |
| 4-level slicer + dibit map | {−3,−1,+1,+3} → dibits per TIA-102.BAAA | `demod.C4FM.Slice`; `phase1.SymbolToDibit` (`sync.go`) |
| Channel rate | 48 kHz for the 4800-baud C4FM family | `internal/scanner/ccdecoder/ddc.go` (`ddcTargetForProtocol`) |
| The composed chain | IQ → dibits, both demod paths | `internal/radio/p25/phase1/receiver/receiver.go` (`Receiver.Process`) |

## In this post

- **The amplitude question** — C4FM vs π/4-DQPSK, the TETRA table with the roles reversed.
- **The deviation ladder** — ±600/±1800 Hz, and how four frequencies become dibits.
- **The filter that isn't an RRC** — the transmit/receive pair real P25 actually uses.
- **48 kHz and ten samples per symbol** — why the channel rate is a named constant.
- **The receiver in one pass** — the chain from IQ to dibits, and the hooks later parts use.

## The amplitude question

[TETRA End to End Part 1]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }})
opened with a table contrasting the two great demodulation families from
TETRA's side. Here it is with the roles reversed — this time the C4FM column
is the protagonist:

| Axis | P25 Phase 1 (C4FM) | TETRA TMO |
|---|---|---|
| Modulation | 4-level FSK (frequency → amplitude) | π/4-DQPSK (differential phase) |
| Symbol rate | 4800 sym/s | 18000 sym/s |
| Channel width | 12.5 kHz | 25 kHz |
| Channel bit rate | 9.6 kbps | 36 kbps before slot muxing |
| The demod question | "what amplitude is this symbol?" | "how far did the phase move?" |
| Sensitive to | carrier offset (slicer bias), level error | phase wander between symbols |
| GT channel rate | 48 kHz (10 sps) | 144 kHz (8 sps) |
| Vocoder | IMBE (MBE family) | ACELP (EN 300 395-2) |

A TETRA demodulator asks a *relative* question, so a constant phase rotation
cancels in `s·conj(last)`. A C4FM slicer asks an *absolute* one: after the FM
discriminator each symbol is a level, and the receiver must know where the
four levels sit. That difference explains most of what this chain carries
that TETRA's doesn't — a symbol AGC to calibrate levels, a `CoarseAFC` to
remove the DC bias a tuner error injects, and slicer thresholds that are
physical quantities derived from the deviation. The
[demodulation primer]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
covers both families in the abstract, and the
[DSP learning track]({{ '/learn/dsp/' | relative_url }}) builds the theory up
from IQ; here we care about what the absolute question costs on air.

## The deviation ladder

C4FM — *Compatible 4-level FM* — encodes each dibit as one of four deviations
from the carrier: outer symbols at ±1800 Hz, inner symbols at ±600 Hz. The
mapping is fixed by TIA-102.BAAA and lives in one place in the tree:

```go
// internal/radio/p25/phase1/sync.go (shape)
// C4FM symbol-to-dibit mapping per TIA-102.BAAA: +3→01, +1→00, -1→10, -3→11.
func SymbolToDibit(sym int8) uint8 {
    switch sym {
    case 1:
        return 0 // +600 Hz
    case 3:
        return 1 // +1800 Hz
    case -1:
        return 2 // −600 Hz
    case -3:
        return 3 // −1800 Hz
    }
    return 0
}
```

Note the ordering: the dibit values do **not** climb the ladder with the
deviation. `+1` maps to dibit 00 and `+3` to dibit 01 — the high bit is the
sign, the low bit selects inner/outer, so the most likely slicer error
(inner ↔ outer on the same side) corrupts only one bit.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="The C4FM deviation ladder: four frequency levels at plus and minus 1800 and 600 hertz around the carrier, each labelled with its symbol and dibit, dashed slicer thresholds between them, and a note that a carrier offset shifts the whole ladder against the fixed thresholds">
  <line x1="60" y1="120" x2="430" y2="120" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="434" y="124" fill="var(--fg-muted)" font-size="10">carrier (0 Hz)</text>
  <!-- levels -->
  <line x1="80" y1="40" x2="400" y2="40" stroke="var(--accent)" stroke-width="2"/>
  <text x="60" y="44" text-anchor="end" fill="var(--accent)" font-size="10">+1800 Hz</text>
  <text x="408" y="44" fill="currentColor" font-size="10">+3 → dibit 01</text>
  <line x1="80" y1="93" x2="400" y2="93" stroke="currentColor" stroke-width="2"/>
  <text x="60" y="97" text-anchor="end" fill="currentColor" font-size="10">+600 Hz</text>
  <text x="408" y="97" fill="currentColor" font-size="10">+1 → dibit 00</text>
  <line x1="80" y1="147" x2="400" y2="147" stroke="currentColor" stroke-width="2"/>
  <text x="60" y="151" text-anchor="end" fill="currentColor" font-size="10">−600 Hz</text>
  <text x="408" y="151" fill="currentColor" font-size="10">−1 → dibit 10</text>
  <line x1="80" y1="200" x2="400" y2="200" stroke="var(--accent)" stroke-width="2"/>
  <text x="60" y="204" text-anchor="end" fill="var(--accent)" font-size="10">−1800 Hz</text>
  <text x="408" y="204" fill="currentColor" font-size="10">−3 → dibit 11</text>
  <!-- thresholds -->
  <line x1="80" y1="66" x2="400" y2="66" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <line x1="80" y1="174" x2="400" y2="174" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="240" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">slicer threshold ±1200 Hz</text>
  <text x="240" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">slicer threshold</text>
  <!-- offset annotation -->
  <text x="530" y="60" fill="currentColor" font-size="11" font-weight="bold">absolute levels,</text>
  <text x="530" y="76" fill="currentColor" font-size="11" font-weight="bold">fixed thresholds</text>
  <text x="530" y="100" fill="var(--fg-muted)" font-size="10">a tuner offset shifts the</text>
  <text x="530" y="114" fill="var(--fg-muted)" font-size="10">whole ladder against the</text>
  <text x="530" y="128" fill="var(--fg-muted)" font-size="10">thresholds — inner symbols</text>
  <text x="530" y="142" fill="var(--fg-muted)" font-size="10">mis-slice as outer</text>
  <text x="530" y="166" fill="var(--fg-muted)" font-size="10">high bit = sign,</text>
  <text x="530" y="180" fill="var(--fg-muted)" font-size="10">low bit = inner/outer —</text>
  <text x="530" y="194" fill="var(--fg-muted)" font-size="10">a one-step slice error</text>
  <text x="530" y="208" fill="var(--fg-muted)" font-size="10">costs one bit, not two</text>
</svg>
<figcaption>The C4FM ladder: four absolute frequency levels sliced against fixed thresholds — why a few hundred hertz of carrier offset is a decode problem, not a cosmetic one.</figcaption>
</figure>

The figure also shows the trap. The discriminator maps frequency linearly to
amplitude, so a static tuner error of Δf shifts every level by the same
amount — and against ±600 Hz inner deviations, offsets ≥ ~1 kHz start pushing
inner symbols across outer thresholds (issue #402's territory). That is the
structural reason this chain carries a `CoarseAFC` tracking and subtracting
the discriminator's DC bias, and why later parts keep returning to carrier
offset as a first-class failure mode.

## The filter that isn't an RRC

Between the discriminator and the slicer sits the matched filter, and here
P25 holds the series' first self-consistency lesson. The obvious model — the
one GopherTrunk originally shipped — is a root-raised-cosine pair. Textbook.
Also wrong for P25:

```go
// internal/dsp/demod/c4fm_p25.go (shape)
// P25 C4FM is NOT a root-raised-cosine matched-pair system. The transmit
// baseband filter is a raised-cosine (α=0.2) cascaded with an inverse-sinc
// compensation; the receive filter is a sinc (a one-symbol-period
// integrate-and-dump). The transmit inverse-sinc and the receive sinc
// cancel, so the cascade transmit×receive is a plain raised-cosine —
// ISI-free at the symbol instants.
func NewC4FMP25(sampleRate, deviation float64) *C4FM {
    return NewC4FMWithTaps(P25C4FMRxTaps(sampleRate), deviation)
}
```

The RRC model was *self-consistent*: GopherTrunk's synthetic C4FM modulator
shaped with an RRC too, so every round-trip test passed while real captures
carried residual inter-symbol interference the receiver could never remove
(issue #275). The fix cross-checked the filter against OP25's
`c4fm_const.py` and now generates the taps from the spec transfer functions —
raised-cosine flat to 1920 Hz rolling off to 2880 Hz, times the inverse-sinc,
on transmit; `sinc(f/4800)` on receive (`P25C4FMRxTaps`). This is the pattern
the whole series keeps meeting: **a test whose encoder and decoder share an
assumption validates the assumption against nothing.** The villain returns in
Part 3 (a byte-offset bug) and Part 13 (the testing playbook), wearing
different clothes each time.

One subtlety worth planting: `P25C4FMRxTaps` is normalised to a DC gain of
sps, so the matched-filtered symbol centres land at ±2π·1800/48000 ≈ 0.2356
rad/sample — a *physical* level the slicer thresholds are calibrated against,
with a symbol AGC bridging the gap when real signal levels drift.

## 48 kHz and ten samples per symbol

GopherTrunk channelizes every protocol to a fixed per-protocol rate, chosen
in one function — the same one the TETRA opener showed from the other side:

```go
// internal/scanner/ccdecoder/ddc.go (shape)
func ddcTargetForProtocol(p trunking.Protocol) float64 {
    switch p {
    case trunking.ProtocolTETRA, trunking.ProtocolTETRADMO:
        return tetraDDCTargetRateHz // 144_000 — 8 sps at 18000 baud
    case trunking.ProtocolMotorola:
        return motorolaDDCTargetRateHz // 18_000 — 5 sps at 3600 baud
    }
    return ddcTargetRateHz // 48_000 — the 4800-baud C4FM family
}
```

At 4800 sym/s, 48 kHz is exactly **10 samples per symbol**, and everything in
the receiver is sized from it: the matched-filter span (13 symbols per the
OP25 reference), the CoarseAFC time constant, the Mueller-Müller loop gain,
the AGC's ~256-symbol settling. The constant's doc comment records what
happens without it: feed the receiver a raw 2.048 MHz SDR stream and you get
≈427 samples per symbol, a matched filter spanning a ±1 MHz swath, and a frame
sync word that never correlates (issue #275). The exported `DDCTargetRateHz`
exists so the replay subcommand builds an *identical* down-converter — a
replay path that channelizes differently from the live path is a twin pair
waiting to drift, and Part 11 covers the time exactly that happened between
the single-channel `Downconverter` and the wideband `DDCBank`.

## The receiver in one pass

`internal/radio/p25/phase1/receiver` composes the chain. On the default
`DemodC4FM` path, `Receiver.Process` runs: an optional pre-discriminator DC
blocker (voice chains only — never the control-channel DDC path) → FM
discriminator → spec C4FM matched filter → `CoarseAFC` → Mueller-Müller
symbol clock
([timing recovery in general form]({{ '/blog/deep-dives/sdr-internals-07-symbol-timing-sync-recovery/' | relative_url }}))
→ symbol AGC → 4-level slicer → `SymbolToDibit`. The output dibit stream, in
the TIA-102.BAAA convention, feeds a `DibitSink` (the control-channel state
machine, Part 2) and/or an LDU assembler (the voice path, Part 8).

Three hooks for later parts:

- **There is a second demod path.** `Options.DemodMode` selects `DemodCQPSK`:
  a complex RRC → Gardner → blind-equalizer → differential-decode chain for
  P25 sites transmitting a linear/LSM waveform, on which the FM discriminator
  produces near-random dibits. Same dibits out, entirely different physics —
  the first twin pair this series tracks, and Part 6's whole subject.
- **The soft information already flows.** `SoftSink` surfaces per-symbol soft
  samples, `EyeSink` the oversampled eye, and `BitLLRSink` two
  log-likelihood ratios per dibit — sign-axis distance for the high bit,
  inner/outer-threshold distance for the low — feeding the soft-decision TSBK
  path. All nil by default, all free when unused.
- **The dibit stream is demod-agnostic by contract.** FSW detection, NID
  parsing and the TSBK trellis (Parts 2–3) never know which physics produced
  their dibits — one control-channel state machine serves both paths, a seam
  [Protocol Decoders Part 2]({{ '/blog/deep-dives/protocol-decoders-02-p25-phase-1-physical-layer/' | relative_url }})
  surveyed in brief.

### How the amplitude question shaped the Go code

- **Levels are calibrated, not assumed.** `Options.DeviationHz` derives the
  slicer scale, the matched filter's DC gain restores it, and the symbol AGC
  holds mean|x| at the slicer's expected value — a calibration chain a
  differential decoder doesn't need.
- **Carrier offset gets a dedicated stage.** `CoarseAFC` exists because
  frequency error *is* slicer bias here; its decision-directed refinement
  (issue #402) ships opt-in behind `EnableDecisionDirectedAFC` because it can
  stably false-lock.
- **One mapping, one owner.** `SymbolToDibit` is the single source of truth
  for the ladder, exactly as `TetraBitsToDibits` is for TETRA's Gray map —
  deliberately separate functions, because mixing the conventions produces
  garbage that looks like a demod bug.

## Where this goes next

A dibit stream at 4800/s is a firehose with no punctuation.
[Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})
adds the structure: the 48-bit frame sync word, the BCH-protected NID naming
each frame's NAC and type, the status symbols interleaved every 36th dibit —
and what GopherTrunk requires before it logs `control channel locked`.

## FAQ

**Is C4FM the same thing as 4FSK?**
It's a constrained form of it: four-level FSK with a specific pulse shaping
(raised-cosine + inverse-sinc at the transmitter) chosen so the signal fits a
12.5 kHz channel *and* demodulates on a simple FM discriminator. The
"compatible" in the name is the point — the same family of waveforms also
receives as a phase modulation, the door Phase 1's CQPSK/LSM twin path
(Part 6) walks through.

**Why does GopherTrunk demodulate with an FM discriminator instead of
coherently?**
It's simple, robust and sufficient on a clean channel: frequency maps
straight to amplitude and a slicer finishes the job, with no carrier-phase
loop to lose lock. The honest cost is Part 12's subject — the discriminator
path has no equalizer and hard-decision FEC, and on weak or simulcast-smeared
signals that is the gap between GopherTrunk and better hardware.

**What does a P25 carrier look like on a spectrum display?**
A roughly 8–10 kHz-wide hump inside 12.5 kHz spacing — no sub-carriers, no
TDMA slot rhythm on Phase 1 (the control channel transmits continuously;
voice channels key up per call). The four deviation levels are invisible in
the spectrum; they only appear in the demodulated eye diagram, which is why
the receiver exposes an `EyeSink` fold for the diagnostics panels.

**Why 48 kHz and not something lower?**
Ten samples per symbol is a comfortable operating point for the
Mueller-Müller loop, and every loop constant in the receiver was tuned at
10 sps. The rate is also shared by the whole 4800-baud family (DMR, NXDN,
dPMR), so one DDC design serves five protocols.

**Does the receiver care about the ±600/±1800 numbers, or just their ratio?**
Both. The 3:1 ratio fixes the slicer geometry (thresholds at ±2/3 of the
outer level); the absolute values calibrate the slicer scale in physical
units via `DeviationHz` — and the small inner deviation is what makes a
~1 kHz carrier offset dangerous here while TETRA, whose information rides in
phase differences, shrugs off far more.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Frame Sync, the NID & What 'Locked' Means]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})
