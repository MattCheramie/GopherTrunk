---
title: "The Legacy Family End to End, Part 2: The Shared Skeleton — ControlChannel.Process, Band Plans, Topology, strict_test"
description: "What every legacy decoder package in GopherTrunk shares — a receiver subpackage with one Options shape, a ControlChannel.Process adapter with three framing strategies, a LockState the hunter can read, a band-plan resolver, topology and strict validation — and the two places where the production factory leaves that skeleton unwired."
category: deep-dives
keywords: controlchannel process adapter, bitsink baseidx, mueller muller legacy receiver, lcn to frequency band plan, linearbandplan tablebandplan, lockedpayload cchunt, resyncguard decode drought, strict validation opcode, fec opt-out defaults, ddc target rate per protocol, gophertrunk legacy family
tags: [legacy-family-end-to-end, architecture, receiver, band-plan, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 2
---

*Part 2 of **The Legacy Family End to End**, a 14-part deep dive into the
FM-era trunking generation and the AMBE-era narrowband and amateur modes
through GopherTrunk.
[Part 1]({{ '/blog/deep-dives/legacy-family-01-what-legacy-means/' | relative_url }})
sorted the eight protocols into two generations, found them in the
protocol enum and the pipeline factories, and placed each on a four-rung
verification ladder. This part reads the skeleton they all share — the
receiver subpackage, the `Process` adapter, the band plan, the lock
payload, the topology accumulator and the strict-validation pattern — so
the protocol parts that follow can spend their words on what differs.
It also names the two places where that skeleton is present in the
package but never wired by the production factory.*

> **TL;DR:** Every legacy package has the same bones. A `receiver/`
> subpackage takes `Options{SampleRateHz, BitSink, ClockGain}`, panics
> below two samples per symbol, runs FM discriminator → protocol-specific
> shaping → `sync.MuellerMuller` → slicer, and emits bits with a
> monotonic `baseIdx`. `ControlChannel.Process(bits, baseIdx) int`
> frames them one of three ways — SmartNet's sync-bracketed 84-bit ring,
> EDACS's 24-bit `SyncDetector` plus a 40-bit countdown, LTR's sync-bit
> alignment of 41-bit words — and hands each unit to `Ingest`. A
> `LockState` satisfies `trunking.LockedPayload` so the hunter can read
> `cc.locked`; Motorola and EDACS also expose a `TopologySnapshot`. The
> band plan is where the skeleton has gaps: `motorola.BandPlan` is wired
> from `motorola_band_plan`, but `newEDACSPipeline` and `newLTRPipeline`
> construct with no `Resolver`, so their grants carry `FrequencyHz: 0`
> and `Engine.HandleGrant` drops them with `dropping grant with zero
> frequency`. `resyncGuard` is wired to no legacy pipeline because none
> has a `LastActivityNano` heartbeat.

**Key takeaways**

- **One receiver shape, four shaping stages.** The chains differ only
  between the discriminator and the timing loop: a DC tracker and boxcar
  (SmartNet), a Gaussian matched filter (EDACS), a 300 Hz Kaiser low-pass
  (LTR), an FFSK tone discriminator (MPT 1327).
- **`Process` is an adapter; `Ingest` is the contract.** Framing lives
  in a lazily built `processState`; the state machine never sees bits.
- **Band plans resolve; factories decide.** The resolver interfaces are
  identical across packages, but only SmartNet's is constructed from
  config. EDACS and LTR grants reach the bus with no frequency.
- **FEC defaults are on; strict validation is dormant.** Every
  `Parse*Mode("")` selects the spec chain, while `SetStrictValidation`
  has tests in six packages and no production caller.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Receiver options | `SampleRateHz`, `BitSink`, `ClockGain` (default 0.05); panics below 2 sps | `internal/radio/{motorola,edacs,ltr,mpt1327}/receiver/receiver.go` |
| Framing adapters | bracket ring / sync countdown / sync-bit alignment | `motorola/process.go`, `edacs/process.go`, `ltr/process.go` |
| Lock payload | `LockedFrequencyHz()` / `LockedNAC()` for the hunter | `trunking.LockedPayload` (`internal/trunking/cchunt.go`), each package's `LockState` |
| Band plans | `motorola.BandPlan`; `edacs.Resolver` / `ltr.Resolver` with `LinearBandPlan`, `TableBandPlan` | `*/bandplan.go`; wiring in `ccdecoder/pipelines.go` |
| Zero-frequency drop | grants with `FrequencyHz == 0` are logged and discarded | `internal/trunking/engine.go` (`HandleGrant`) |
| Topology | `topologyModel` → `TopologySnapshot` on the pipeline | `motorola/topology.go`, `edacs/topology.go`, `nxdn/topology.go` |
| Strict validation | drop unknown opcodes / malformed words at `Ingest` | `*/strict_test.go`, `SetStrictValidation` |
| Channel rates | 48 kHz default, 18 kHz Motorola, 144 kHz TETRA | `ccdecoder/ddc.go` (`ddcTargetForProtocol`) |

## In this post

- **The receiver subpackage** — one `Options`, four shaping stages, one timing loop.
- **Three ways to frame a bit stream** — bracket, countdown, alignment.
- **Lock, topology and the hunter** — what `cc.locked` must carry.
- **Band plans, and where they are not wired** — the zero-frequency grant.
- **Strict validation and the FEC defaults** — what is on, what is dormant.
- **Rates, resync and reset** — the pipeline wrapper's small differences.

## The receiver subpackage

Each FM-era package carries a `receiver/` subpackage with the same
constructor contract. `Options` names the channelized `SampleRateHz`,
the `BitSink`, and a Mueller-Müller `ClockGain` defaulting to 0.05;
`New` panics when the rate or sink is missing or the samples per symbol
fall below two — the panic strings name the floor: `7200 Hz` for
SmartNet, `19200 Hz` for EDACS, `600 Hz` for LTR, `2400 Hz` for
MPT 1327. EDACS adds `PulseSpanSymbols` and `BTProduct`; LTR adds
`LPFCutoffHz` and `LPFLen`. Every receiver test file opens with
`TestReceiverConstructsAndProcessesSilence` and
`TestReceiverConstructorPanicsOnBadParams`.

Between the shared ends — `demod.FM` in front, `sync.MuellerMuller` and
a slicer behind — the chains differ by one shaping stage:

```go
// internal/radio/motorola/receiver/receiver.go (shape)
r.disc = r.fm.Process(r.disc, iq)
for i, x := range r.disc {
    r.dcMean += r.dcAlpha * (x - r.dcMean) // dcTrackSeconds = 0.1
    x -= r.dcMean
    r.boxSum += x - r.boxHist[r.boxPos]    // one-symbol boxcar, taps = round(sps)
    // …
}
r.symbols = r.clock.Process(r.symbols, r.matched)
```

SmartNet's stage is a slow DC tracker plus a one-symbol boxcar, a port
of trunk-recorder's `smartnet_fsk2_demod`. EDACS runs `demod.GFSK`'s
Gaussian matched filter at BT = 0.3 over a 4-symbol half-span and slices
through `gfsk.Slice`. LTR's is a 101-tap Kaiser low-pass
(`LPFCutoffHz` = 300, `LPFBeta` = 8.6) that keeps the sub-audible word
and rejects the voice above it. MPT 1327 uses `demod.FFSK`, a
1200/1800 Hz tone discriminator. All four emit `0`/`1` bytes and advance
`bitBase` by the batch length; `Reset` zeroes `bitBase` and the stage's
filter history — `TestReceiverBitSinkBaseIdxMonotonic` pins both halves.
None carries an AGC: the slicer threshold is zero and the Mueller-Müller
error is amplitude-normalised, as the SmartNet receiver's doc comment
says.

## Three ways to frame a bit stream

The parent package's `ControlChannel.Process(bits []byte, baseIdx int) int`
turns the receiver's stream into protocol units. The signature is shared
— it returns `baseIdx + len(bits)` "to match the YSF / P25 Phase 1 /
dPMR / NXDN / EDACS ControlChannel.Process contracts", as the SmartNet
adapter's comment puts it — and so is the shape: a `processState` built
lazily on the first call so a frame straddling two chunks still decodes.
What differs is how each protocol tells a frame from noise.

SmartNet has only an 8-bit sync, so `motorola/process.go` keeps an
84-bit ring and trusts a frame only when the *next* sync lands exactly
`PayloadBits` later; a missing bracket drops framing entirely. EDACS has
a 24-bit sync and no bracket: `edacs/process.go` runs a `SyncDetector`
at tolerance 1 and, at each match, sets `remaining = 40` and collects
the CCW bit by bit. LTR has no sync word, so `ltr/process.go` searches
for a `1` with room for 41 bits, commits to that alignment, and unlocks
when a later word's first bit reads `0`:

```go
// internal/radio/ltr/process.go (shape)
if !p.aligned {
    for p.off+statusBits <= len(p.buf) && p.buf[p.off] != 1 { p.off++ }
    st, _ := StatusFromBits(p.buf[p.off : p.off+statusBits])
    c.Ingest(st); p.aligned = true; p.off += statusBits
    continue
}
window := p.buf[p.off : p.off+statusBits]
if window[0] != 1 { p.aligned = false; continue } // Sync invariant broken
```

Each adapter ends at `Ingest`, the seam the package tests use:
`edacs_test.go`, `ltr_test.go` and `motorola_test.go` publish decoded
words directly and assert what lands on the bus, while each
`process_test.go` feeds bit streams, including chunk-boundary cases such
as `TestProcessHandlesSyncSpanningCalls` (EDACS) and
`TestProcessSurvivesChunkBoundaries` (SmartNet).

<figure class="lab-figure">
<svg viewBox="0 0 680 236" width="680" height="236" role="img" aria-label="A horizontal pipeline from IQ through the DDC and the receiver to a BitSink, then three framing adapters stacked beneath it — SmartNet bracketed sync ring, EDACS sync detector countdown, LTR sync-bit alignment — each ending at Ingest, which publishes cc.locked and grant events to the bus.">
  <rect x="16" y="22" width="60" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="46" y="41" text-anchor="middle" fill="currentColor" font-size="9">IQ</text>
  <rect x="90" y="22" width="90" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="135" y="36" text-anchor="middle" fill="currentColor" font-size="9">DDC</text>
  <text x="135" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="8">48 k · 18 k (Motorola)</text>
  <rect x="194" y="22" width="250" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="319" y="36" text-anchor="middle" fill="currentColor" font-size="9">receiver/: FM → shaping stage → Mueller-Müller → slicer</text>
  <text x="319" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="8">DC+boxcar · GFSK matched · 300 Hz LPF · FFSK tones</text>
  <rect x="458" y="22" width="90" height="30" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="503" y="36" text-anchor="middle" fill="var(--accent)" font-size="9">BitSink</text>
  <text x="503" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bits, baseIdx</text>
  <line x1="76" y1="37" x2="90" y2="37" stroke="currentColor"/>
  <line x1="180" y1="37" x2="194" y2="37" stroke="currentColor"/>
  <line x1="444" y1="37" x2="458" y2="37" stroke="currentColor"/>
  <line x1="503" y1="52" x2="503" y2="70" stroke="var(--accent)"/>
  <text x="340" y="84" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">ControlChannel.Process(bits, baseIdx) → one of three framers</text>
  <rect x="16" y="96" width="200" height="54" rx="4" fill="none" stroke="currentColor"/>
  <text x="116" y="112" text-anchor="middle" fill="currentColor" font-size="9">SmartNet: bracket ring</text>
  <text x="116" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="8">sync 0xAC … 76 bits … next sync</text>
  <text x="116" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="8">no bracket ⇒ drop framing</text>
  <rect x="240" y="96" width="200" height="54" rx="4" fill="none" stroke="currentColor"/>
  <text x="340" y="112" text-anchor="middle" fill="currentColor" font-size="9">EDACS: detector + countdown</text>
  <text x="340" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="8">24-bit sync, tolerance 1</text>
  <text x="340" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="8">remaining = 40 → parseCCW</text>
  <rect x="464" y="96" width="200" height="54" rx="4" fill="none" stroke="currentColor"/>
  <text x="564" y="112" text-anchor="middle" fill="currentColor" font-size="9">LTR: sync-bit alignment</text>
  <text x="564" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="8">first 1 with 41 bits of room</text>
  <text x="564" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="8">word[0] == 0 ⇒ re-search</text>
  <line x1="116" y1="150" x2="300" y2="180" stroke="var(--fg-muted)"/>
  <line x1="340" y1="150" x2="340" y2="180" stroke="var(--fg-muted)"/>
  <line x1="564" y1="150" x2="380" y2="180" stroke="var(--fg-muted)"/>
  <rect x="250" y="180" width="180" height="44" rx="4" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="197" text-anchor="middle" fill="var(--accent)" font-size="10">Ingest(word)</text>
  <text x="340" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="8">cc.locked · KindGrant · cc.lost on MarkLost</text>
</svg>
<figcaption>The skeleton every FM-era package shares; only the shaping stage and the framer are protocol-specific.</figcaption>
</figure>

## Lock, topology and the hunter

A decoder's first job on a freshly tuned frequency is to tell the
control-channel hunter it has found something. Each package publishes
`events.KindCCLocked` with its own `LockState` — `{FrequencyHz,
SystemID}` for SmartNet and EDACS, `{FrequencyHz, Area, Repeater}` for
LTR — and each type implements the two methods `cchunt` type-asserts:

```go
// internal/trunking/cchunt.go
type LockedPayload interface {
    LockedFrequencyHz() uint32
    LockedNAC() uint16
}

// internal/radio/ltr/control.go — (Area, Repeater) packed into the NAC slot
func (s LockState) LockedNAC() uint16 { return uint16(s.Area)<<8 | uint16(s.Repeater) }
```

The comment on those methods is the same in all three packages: without
them "the supervisor's type-assertion on cc.locked silently drops the
event and /api/v1/scanner never surfaces state=locked."
`TestLockStateSatisfiesLockedPayload` in the SmartNet package pins the
contract. `maybeLock` publishes on the first identity or a changed one;
`MarkLost` publishes `KindCCLost` with the last state and clears the
flag, pinned per package by `TestControlChannelMarkLost` or
`TestMarkLostPublishesCCLost`.

SmartNet and EDACS also accumulate topology. `topologyModel` is a
mutex-guarded `{SystemID, Neighbors}`: `applySystemID` keeps the first
non-zero value and `applyNeighbor` / `applyAdjacent` de-duplicate — by
channel number for SmartNet (its OSW broadcasts carry no site ID) and by
`SiteID` for EDACS. The pipeline wrapper maps that to the neutral
`trunking.TopologySnapshot`, resolving each neighbour's frequency with
`NeighborFrequency`, which is how a legacy system's neighbours reach the
systems report. LTR and MPT 1327 have no `topology.go`.

## Band plans, and where they are not wired

A grant in this family carries a channel number, not a frequency, so
every package has a resolver. SmartNet's is the richest because the
channel number *is* the discriminator of its state machine:
`motorola.BandPlan` offers `Frequency(ch)`, `IsChannel(ch)` and `Name()`,
`ParseBandPlan` selects `800_standard`, `800_rebanded`, `800_splinter`
or `900`, and `newMotorolaPipeline` wires the result from the
`motorola_band_plan` key. EDACS and LTR share a simpler shape:

```go
// internal/radio/edacs/bandplan.go (ltr/bandplan.go is identical in shape)
type Resolver interface { Frequency(lcn uint8) (uint32, error) }

type LinearBandPlan struct { BaseHz, SpacingHz uint32; Offset int }
// freq = BaseHz + (lcn + Offset) * SpacingHz; negative index is an error
type TableBandPlan map[uint8]uint32
```

Here the skeleton stops. `trunking.System` carries `P25BandPlan`,
`DMRBandPlan` and `NXDNBandPlan`, and `config.go` maps `dmr_band_plan`
and `nxdn_band_plan`, but there is no EDACS or LTR equivalent — and
`newEDACSPipeline` and `newLTRPipeline` construct their control channels
with no `Resolver` at all. `publishGrant` in both packages then emits
`freq := uint32(0)`. The package tests pin the outcome —
`TestControlChannelGrantWithoutResolverHasZeroFreq` exists in both
`edacs_test.go` and `ltr_test.go` — and the engine's first check in
`HandleGrant` decides what happens next:

```go
// internal/trunking/engine.go
if g.FrequencyHz == 0 {
    e.log.Warn("dropping grant with zero frequency", "grant", g.String())
    return
}
```

So on a live EDACS or LTR system today the control channel can lock and
the grant can publish with the right talkgroup and `ChannelNum`, and the
engine will log a WARN and not follow it. `TestDaemonCCDecodesEDACS` and
`TestDaemonCCDecodesLTR` assert lock only, which is why they are green.
The fix is the shape `dmr_band_plan` already has — a config section, a
`trunking` mirror, a `ResolverFromPlan` — and it is not built. Parts 4
and 5 return to this with each protocol's numbers.

## Strict validation and the FEC defaults

Every legacy `ControlChannel` has a `SetStrictValidation(bool)` and a
`strict_test.go`. The filter is what EDACS's comment calls "soft FEC":
it corrects nothing, but drops words that violate a protocol invariant.
For EDACS that is `Command.IsKnown()` — the opcodes `0xA`..`0xE` are
unallocated, and `TestStrictValidationDropsUnknownCommand` feeds
`Command(0xA)`. For LTR it is `Status.IsWellFormed()` — `Sync` set,
`Channel` and `Home` both in 1..20 —
`TestStrictValidationDropsOutOfRangeHome` sends `Home: 25`. dPMR,
MPT 1327, TETRA and P25 Phase 2 carry the same file. What no package
carries is a production caller: `SetStrictValidation` is invoked from
tests only, so the filter is a tested, dormant lever rather than a
shipping default.

The FEC layers are the opposite: on by default, opt-out per system.
`docs/opt-in-features.md` records the rule and its implementation note —
the connector "goes through `Parse*Mode(opts.System.X)` which maps empty
strings to the new on-defaults, then calls `SetXMode(parsed)`", while
"the in-package `ControlChannel` constructors still zero-value to `Off`
mode so direct callers (primarily unit tests) see the legacy behaviour."
For this family the keys are:

| Key | Default | Opt-out |
|---|---|---|
| `edacs_bch_mode` | `BCHOn` — BCH(40,28,2) per CCW | `off` |
| `ltr_fcs_mode` | `FCSOn` — CRC-7 over the 24-bit message | `off` |
| `ltr_manchester_mode` | `ManchesterSoft` | `off` / `nrz`; `strict` |
| `mpt1327_bch_mode` | `BCHOn` | `off` |
| `mpt1327_cwsc_tolerance` | 2 of 16 sync bits | `0` / `exact` |
| `motorola_band_plan` | `800_standard` | — |
| `motorola_bch_mode` | obsolete, accepted and ignored | — |

Each factory follows one pattern: parse, warn-log on an unrecognised
value, fall back to the default, call the setter. The warn strings are
greppable — `ccdecoder: unrecognised edacs_bch_mode; falling back to on`
— and `newMotorolaPipeline` logs an INFO line when the obsolete key is
present.

## Rates, resync and reset

The pipeline wrapper around each receiver-plus-state-machine pair is a
dozen lines. The channel rate comes from `ddcTargetForProtocol`: 48 kHz
for everything here except SmartNet's 18 kHz, which gives EDACS 5
samples per symbol, MPT 1327 40 and LTR 160.
`TestMotorolaPipelineDecodesThroughProductionDDC` is the one legacy test
that drives a wideband 90 kHz stream through the real `Downconverter` to
that target, the configuration the pre-#1143 tests never exercised.

The signal-time decode-drought watchdog, `resyncGuard`
(`ccdecoder/resyncguard.go`), is the TETRA `checkResync` design
generalised: a full window of *processed* signal with no CRC-clean decode
forces a receiver reset. It is wired to P25 Phase 1 and 2, DMR Tier III
and NXDN, with windows from `p25Phase1ResyncWindow` = 2 s to
`dmrTier3ResyncWindow` = 3 s. None of the FM-era pipelines carries it,
for a structural reason: the guard reads a `LastActivityNano` heartbeat,
and only the DMR Tier III, NXDN, P25 and TETRA control channels stamp
one. A legacy pipeline's `Process` is just `p.rx.Process(iq)`.

`Reset` is where the wrappers differ most. `motorolaPipeline.Reset`
calls `p.rx.Reset()` then `p.cc.ResetFraming()`, dropping the bracket
ring so a half-frame cannot bridge two IQ epochs. The EDACS, LTR and
MPT 1327 wrappers reset the receiver only; EDACS's `SyncDetector` has a
`Reset` method and LTR's adapter has a buffer, but neither is cleared
from the pipeline, so on a retune `bitBase` restarts at zero while the
framing state carries over — worth knowing when a legacy system's first
decode after a hunt looks odd.

## Where this goes next

With the skeleton in place, the series can read each protocol's flesh.
[Part 3]({{ '/blog/deep-dives/legacy-family-03-smartnet-air-interface/' | relative_url }})
takes the Motorola Type II control channel bit by bit — the 27-bit OSW
and its inverted fields, the stride-19 interleave and parity ECC, the
CRC-10 registers, the multi-OSW sequencer that makes a grant out of two
words, the four band plans and the 18 kHz front end — and ends on the
one capture that would move it off the reference-pinned rung.

## FAQ

**What does ControlChannel.Process do in GopherTrunk's legacy decoders?**
It adapts the receiver's bit stream to the protocol state machine: a
lazily built `processState` frames the stream one of three ways —
SmartNet's sync-bracketed 84-bit ring, EDACS's 24-bit `SyncDetector`
with a 40-bit countdown, LTR's sync-bit alignment of 41-bit words — and
hands each unit to `Ingest`, returning `baseIdx + len(bits)`.

**Why do EDACS and LTR grants have FrequencyHz 0?**
`newEDACSPipeline` and `newLTRPipeline` construct their control channels
without a `Resolver`, and no `edacs_band_plan` or `ltr_band_plan` config
key exists. `publishGrant` emits `FrequencyHz: 0`, pinned by
`TestControlChannelGrantWithoutResolverHasZeroFreq`, and
`Engine.HandleGrant` drops it with `dropping grant with zero frequency`.
The lock works; the voice follow does not.

**What is trunking.LockedPayload and why does each LockState implement it?**
An interface in `internal/trunking/cchunt.go` with `LockedFrequencyHz()`
and `LockedNAC()`. The hunter type-asserts every `cc.locked` payload
against it; a `LockState` without the methods is silently dropped and
`/api/v1/scanner` never shows `state=locked`. SmartNet and EDACS return
their system ID as the NAC; LTR packs `(Area << 8) | Repeater`.

**Is strict validation on by default for legacy protocols?**
No. `SetStrictValidation` exists on the EDACS, LTR, MPT 1327, dPMR,
TETRA and P25 Phase 2 control channels, each with a `strict_test.go`,
but nothing in production calls it. The on-by-default layers are the
FEC ones — `edacs_bch_mode`, `ltr_fcs_mode`, `ltr_manchester_mode`,
`mpt1327_bch_mode` — selected by `Parse*Mode("")`.

## Series navigation

**Part 2 of 14** · ←
[Part 1: What 'Legacy' Means]({{ '/blog/deep-dives/legacy-family-01-what-legacy-means/' | relative_url }})
· Next →
[Part 3: Motorola Type II — 3600-Baud OSWs, Sync 0xAC and the Band Plans]({{ '/blog/deep-dives/legacy-family-03-smartnet-air-interface/' | relative_url }})
