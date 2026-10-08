---
title: "The Legacy Family End to End, Part 10: dPMR Mode 3 — FS1/FS2 Syncs, CSBK Trunking and the 6.25 kHz Channel"
description: "dPMR Mode 3 as GopherTrunk decodes it: 4FSK at 2400 symbols per second with 900 Hz deviation in a 6.25 kHz channel, the three 48-bit frame syncs, an 80-bit CSBK read raw because its cyclic and rate-¾ convolutional coding is not implemented, a control channel that locks and grants but installs no band plan, and a voice path on a placeholder AMBE+2 interleave — each claim placed on the verification ladder."
category: deep-dives
keywords: dpmr mode 3 decoder, dpmr csbk, dpmr fs1 fs2 fs3 sync, etsi ts 102 658, dpmr 6.25 khz 4fsk 2400, dpmr trunking sdr, pmr446 digital, dpmr ambe+2 voice placeholder, dpmr band plan, gophertrunk legacy family end to end
tags: [legacy-family-end-to-end, dpmr, 4fsk, trunking, ambe, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 10
---

*Part 10 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out, each placed honestly
on one verification ladder.
[Part 9]({{ '/blog/deep-dives/legacy-family-09-nxdn-cac-and-trunking/' | relative_url }})
finished NXDN at a grant with a frequency and a harness waiting for a
capture. This part takes NXDN's European sibling, dPMR Mode 3 — the same
4-level FSK philosophy at half the symbol rate in half the channel — whose
package decodes a clean control channel end to end and says, in its own
header, which two layers it has not built.*

> **TL;DR:** `internal/radio/dpmr` decodes ETSI TS 102 658 Mode 3, the only
> dPMR mode with a control channel. `dpmr/receiver` is the C4FM chain at
> `SymbolRate` 2400 (20 sps at the 48 kHz `ddcTargetRateHz`) with
> `DeviationHz` 900 — half of P25 / DMR / NXDN — and no AGC, AFC or taps.
> Three 48-bit syncs: `FS1Hex` 0x57FF5F75F575 (superframe start), `FS2Hex`
> 0x5F7F77FD7DFD (mid-superframe), `FS3Hex` 0x7DDFFD5F55D5 (CSBK burst).
> `ControlChannel.Process` matches FS3 within one dibit, collects 40 dibits
> and parses an 80-bit CSBK — 5-bit `MessageType`, 3 flags, 24-bit source,
> 24-bit destination, 8-bit service info, 16-bit opcode-specific field —
> **with no FEC**: the spec's short-block cyclic code and rate-¾
> convolutional outer code are an unimplemented deferral, and
> `SetStrictValidation` is the only defence. `Ingest` locks on
> `StandingServiceStatus` or any voice allocation and publishes a
> `Protocol "dpmr"` grant whose frequency comes from a `Resolver` that
> `newDPMRPipeline` never installs — so live grants carry `FrequencyHz` 0
> and the engine drops them. Voice: FS1/FS2 → 24 CCH + 144 TCH dibits →
> 4 × 72-bit AMBE+2 frames on a placeholder interleave → `ambe2-dmr`. Rung:
> **spec-derived, synthetic-verified, no capture, no reference decoder.**

**Key takeaways**

- **Half the rate, same chain, fewer instruments.** dPMR runs 2400 sym/s
  at 900 Hz through the NXDN-shaped receiver minus its AGC, AFC and taps;
  it is in neither the symbol scope nor the wideband engine.
- **An 80-bit CSBK read raw is a decoder for clean fixtures.** The spec's
  channel coding is named and absent; strict validation drops unknown
  message types but cannot correct a bit.
- **Lock works, follow does not.** `TestDaemonCCDecodesDPMR` proves the
  pipeline locks on a synthesised `StandingServiceStatus`; nothing wires the
  band plan a voice grant needs.
- **Two docs disagree and the code decides.** `docs/decoder-capture-needs.md`
  says dPMR control "FEC is on by default"; the package header and
  `newDPMRPipeline` show no CSBK FEC exists.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Receiver | FM → RRC(α 0.20, span 8) → MM(20 sps) → slice(scale from 900 Hz) → Gray dibits | `internal/radio/dpmr/receiver/receiver.go` |
| Syncs | FS1 / FS2 / FS3, 24 dibits each, `SyncDetector` within tolerance | `sync.go` (`FS1Hex`, `FS2Hex`, `FS3Hex`, `SyncDibits`) |
| Control burst | FS3 (tol 1) → 40 dibits → `CSBKFromBits` → `Ingest` | `process.go` (`processState`) |
| CSBK | Type 5 · Flags 3 · Source 24 · Dest 24 · ServiceInfo 8 · Extra 16 | `csbk.go` (`CSBK`, `ParseCSBK`), `opcodes.go` (`MessageType`) |
| Lock / grant | `StandingServiceStatus` or voice allocation → lock; grant via `Resolver` (none installed) | `control.go` (`Ingest`, `maybeLock`, `publishGrant`), `bandplan.go` |
| Voice | FS1/FS2 → `CCHDibits` 24 + `TCHFieldDibits` 144 → 4 × `DecodeTCHFrame` | `traffic.go`, `voice.go`, `voice_ambe.go` (`dpmrAMBEDeinterleave` PLACEHOLDER) |
| Composer | `runDPMRVoiceChain`, ±3125 Hz channel select, `"dpmr": "ambe2-dmr"` | `internal/voice/composer/dpmr_voice.go`, `internal/voice/recorder.go` |
| Pins | `TestDaemonCCDecodesDPMR`, `TestProcessHandlesSyncSpanningCalls`, `TestStrictValidationDropsUnknownMessageType`, `TestTCHFrameRoundTrip` | `cmd/gophertrunk/`, `internal/radio/dpmr/` |

## In this post

- **The 6.25 kHz channel** — 2400 symbols per second, 900 Hz, and what the receiver lacks.
- **Three syncs** — FS1, FS2, FS3 and the countdown adapter.
- **The CSBK, read raw** — fields, message types, flags, and the FEC that is not there.
- **Lock, grant, and the missing band plan** — the state machine and its zero.
- **The voice path** — FS1/FS2 frames, four AMBE+2 carves, one placeholder.
- **The rung** — spec-derived, synthetic-verified, and the capture that moves it.

## The 6.25 kHz channel

dPMR ([reference]({{ '/reference/dpmr/' | relative_url }})) is ETSI
TS 102 658's digital successor to analogue PMR446: 4-level FSK for 6.25 kHz
spacing, with Mode 1 peer-to-peer, Mode 2 managed direct, and Mode 3
centralised trunking. The package targets Mode 3 alone, "the only mode
where the standard 'see CC opcode → retune Voice device → follow grant'
loop applies".
[Protocol Decoders Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
introduced the framing and
[Beyond Voice Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }})
placed dPMR at its distance from verification; this part stays inside the
package.

```go
// internal/radio/dpmr/receiver/receiver.go (shape)
const (
    SymbolRate       = 2400.0 // half of P25 P1 / DMR / YSF — matches 6.25 kHz spacing
    RolloffAlpha     = 0.20
    PulseSpanSymbols = 8
)
// newDPMRPipeline passes DeviationHz: 900.0 — half of P25 / DMR / YSF
slicerScale := 2.0 * math.Pi * opts.DeviationHz / opts.SampleRateHz
r.disc    = r.fm.Process(r.disc, iq)
r.matched = r.mf.MatchedFilter(r.matched, r.disc)
r.symbols = r.clock.Process(r.symbols, r.matched)   // 20 sps at 48 kHz
r.sliced  = r.mf.SliceMany(r.sliced, r.symbols)
for i, sym := range r.sliced { r.dibits[i] = SymbolToDibit(sym) } // +3→01 +1→00 −1→10 −3→11
```

Compare it with the NXDN receiver of
[Part 8]({{ '/blog/deep-dives/legacy-family-08-nxdn-physical-layer/' | relative_url }})
and the differences are all absences. No `C4FMSymbolAGC` — the stage NXDN
gained because the unit-energy RRC overshoots a rectangular stream ~3.1× —
so a real capture meets the fixed thresholds at full matched-filter gain.
No `CoarseAFC`. No `SoftSink` / `EyeSink` taps, and `Reset` rewinds only
`dibitBase`. `internal/scanner/symbolscope` rejects the protocol — its
error names "P25 Phase 1/2, DMR, NXDN, and TETRA only" — and the widebandt2
channel builder lists `dmr-tier2, dmr, p25, p25-phase2, nxdn, and tetra`.
The DDC target is the shared 48 kHz, 20 samples per symbol; the daemon
test notes that this halves the dibits per IQ chunk and is why
`TestDaemonCCDecodesDPMR` carries a 30 s deadline instead of the siblings'
5 s. The composer's `newDPMRVoiceFrontEnd` channel-selects to
±`dpmrChannelSelectHz` 3125, half the spacing.

## Three syncs

TS 102 658 §4.4 defines three 48-bit frame syncs, stored as hex and
materialised into 24 dibits MSB-first
([frame sync reference]({{ '/reference/dpmr-frame-sync/' | relative_url }})):

```go
// internal/radio/dpmr/sync.go
const (
    FS1Hex uint64 = 0x57FF5F75F575 // start of a voice / data superframe
    FS2Hex uint64 = 0x5F7F77FD7DFD // mid-superframe sync (every 4th burst)
    FS3Hex uint64 = 0x7DDFFD5F55D5 // start of a CSBK burst on the control channel
    SyncDibits    = 24
)
```

`TestSyncDibitsAndDetector` pins only what a self-generated test can — 24
dibits each, every value ≤ 3, FS1 ≠ FS2 ≠ FS3. No fixture carries a sync
word typed in from an independent decoder; they are spec transcriptions,
the category the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
post shows a wrong P25 Phase 2 sync hiding in for months.

`ControlChannel.Process` is the countdown adapter every narrowband package
in this series uses, hardwired to FS3:

```go
// internal/radio/dpmr/process.go (shape)
p.det = NewSyncDetector(FS3Dibits(), 1)
for i, d := range dibits {
    if p.remaining > 0 {                       // collect FIRST: a match index is the sync's LAST dibit
        p.csbk = append(p.csbk, d)
        if p.remaining--; p.remaining == 0 {
            if csbk, err := CSBKFromBits(framing.DibitsToBits(p.csbk)); err == nil { c.Ingest(csbk) }
        }
    }
    for matchIdx < len(p.matchScratch) && p.matchScratch[matchIdx] == baseIdx+i {
        p.remaining = 40; p.csbk = p.csbk[:0]; matchIdx++
    }
}
```

Collect-before-match matters because a match's index is the sync's last
dibit, so the CSBK starts one iteration later; the countdown survives chunk
boundaries (`TestProcessHandlesSyncSpanningCalls`), and `SyncDetector.Reset`
clears the ring on a retune. FS1 and FS2 never match the FS3 detector within
tolerance 1, so voice bursts are invisible to the control path by
construction — and the reverse holds for the traffic path.

<figure class="lab-figure">
<svg viewBox="0 0 680 196" width="680" height="196" role="img" aria-label="Two dPMR burst layouts as dibit maps. Top: a control-channel burst, a 24-dibit FS3 sync followed by 40 dibits of CSBK, expanded below into its 80-bit fields: message type 5 bits, flags 3, source ID 24, destination ID 24, service info 8, opcode-specific 16, with a note that no FEC is applied and the bits are read raw. Bottom: a voice frame of 192 dibits, a 24-dibit FS1 or FS2 sync, 24 dibits of CCH that are skipped, and 144 dibits of TCH carved into four 36-dibit AMBE plus 2 frames on a placeholder deinterleave.">
  <text x="8" y="18" fill="currentColor" font-size="9" font-weight="bold">control channel burst (FS3)</text>
  <rect x="8" y="26" width="96" height="24" fill="none" stroke="var(--accent)"/>
  <text x="56" y="42" text-anchor="middle" fill="var(--accent)" font-size="8">FS3 · 24 dibits</text>
  <rect x="104" y="26" width="160" height="24" fill="none" stroke="currentColor"/>
  <text x="184" y="42" text-anchor="middle" fill="currentColor" font-size="8">CSBK · 40 dibits = 80 bits</text>
  <text x="280" y="42" fill="var(--fg-muted)" font-size="8">read raw — no cyclic / rate-¾ FEC implemented</text>
  <rect x="8" y="62" width="40" height="22" fill="none" stroke="currentColor"/>
  <text x="28" y="77" text-anchor="middle" fill="currentColor" font-size="7">type 5</text>
  <rect x="48" y="62" width="26" height="22" fill="none" stroke="currentColor"/>
  <text x="61" y="77" text-anchor="middle" fill="currentColor" font-size="7">flg 3</text>
  <rect x="74" y="62" width="170" height="22" fill="none" stroke="currentColor"/>
  <text x="159" y="77" text-anchor="middle" fill="currentColor" font-size="7">source ID · 24</text>
  <rect x="244" y="62" width="170" height="22" fill="none" stroke="currentColor"/>
  <text x="329" y="77" text-anchor="middle" fill="currentColor" font-size="7">destination ID · 24</text>
  <rect x="414" y="62" width="60" height="22" fill="none" stroke="currentColor"/>
  <text x="444" y="77" text-anchor="middle" fill="currentColor" font-size="7">svc 8</text>
  <rect x="474" y="62" width="116" height="22" fill="none" stroke="var(--accent)"/>
  <text x="532" y="77" text-anchor="middle" fill="var(--accent)" font-size="7">opcode-specific · 16</text>
  <text x="532" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="7">= channel number on a voice allocation</text>
  <line x1="8" y1="112" x2="672" y2="112" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="8" y="130" fill="currentColor" font-size="9" font-weight="bold">voice frame (FS1 at superframe start, FS2 mid-superframe) · 192 dibits · 80 ms</text>
  <rect x="8" y="138" width="80" height="24" fill="none" stroke="var(--accent)"/>
  <text x="48" y="154" text-anchor="middle" fill="var(--accent)" font-size="8">FS1/FS2 · 24</text>
  <rect x="88" y="138" width="80" height="24" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="128" y="154" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CCH · 24 (skipped)</text>
  <rect x="168" y="138" width="120" height="24" fill="none" stroke="currentColor"/>
  <text x="228" y="154" text-anchor="middle" fill="currentColor" font-size="8">AMBE+2 · 36 db</text>
  <rect x="288" y="138" width="120" height="24" fill="none" stroke="currentColor"/>
  <text x="348" y="154" text-anchor="middle" fill="currentColor" font-size="8">AMBE+2 · 36 db</text>
  <rect x="408" y="138" width="120" height="24" fill="none" stroke="currentColor"/>
  <text x="468" y="154" text-anchor="middle" fill="currentColor" font-size="8">AMBE+2 · 36 db</text>
  <rect x="528" y="138" width="120" height="24" fill="none" stroke="currentColor"/>
  <text x="588" y="154" text-anchor="middle" fill="currentColor" font-size="8">AMBE+2 · 36 db</text>
  <text x="408" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TCH · 144 dibits = 4 × 72 bits → DecodeTCHFrame on a PLACEHOLDER C0|C1|C2|C3 split</text>
</svg>
<figcaption>Two bursts, one countdown adapter each. The control path sees only FS3 and reads 80 clean bits; the voice path sees only FS1/FS2 and carves four AMBE+2 frames whose bit order is a placeholder.</figcaption>
</figure>

## The CSBK, read raw

The Common Signalling Block
([CSBK reference]({{ '/reference/dpmr-csbk/' | relative_url }})) is the
80-bit unit Mode 3 transmits between grants, laid out per TS 102 658 §6.5:
bits 0–4 `MessageType`, 5–7 three flags, 8–31 a 24-bit source, 32–55 a
24-bit destination, 56–63 service info, 64–79 opcode-specific
(`TestCSBKByteRoundTrip`, `TestCSBKBitRoundTrip`). The flags are
`FlagGroupCall` 0x4, `FlagEmergency` 0x2, `FlagEncrypted` 0x1. `opcodes.go`
enumerates the §6.5.2 types the trunking layer uses — `RegistrationRequest`
0x01, `RegistrationResponse` 0x02, `VoiceServiceAllocation` 0x03,
`IndividualVoiceAllocation` 0x04, `DataServiceAllocation` 0x05,
`ServiceRequest` 0x06, `StandingServiceStatus` 0x07, `Release` 0x0F, `Idle`
0x1F — and two accessors shape them: `AsVoiceGrant` answers for either voice
allocation, with `Group` forced true for the group variant
(`TestAsVoiceGrantGroupForcesGroupFlag`) and `Channel` taken from `Extra`;
`AsSiteBroadcast` answers for `StandingServiceStatus`, treating `DestID` as
the system identifier.

What the package does not do, its header says under "honest deferrals":
"The interleaver + FEC over CSBK bits. Mode 3 CSBKs use a short-block
cyclic code with rate-3/4 convolutional outer coding; the parsing here
assumes the upstream caller has already corrected errors"
([channel coding reference]({{ '/reference/dpmr-channel-coding/' | relative_url }})).
The 40 dibits after FS3 are read as 80 clean bits. The only defence is
`SetStrictValidation(true)`, which drops a CSBK whose type is outside the
documented set (`MessageType.IsKnown`;
`TestStrictValidationDropsUnknownMessageType` uses 0x10) — a filter against
a misaligned-but-passing window, not a corrector.

Here the docs disagree, and the series rule is to say so.
`docs/decoder-capture-needs.md`'s "not capture-blocked" list reads "EDACS,
LTR, Motorola Type II, dPMR control — control chains ship; FEC is on by
default with no outstanding capture." For dPMR there is no FEC to be on.
The package header, `newDPMRPipeline` (which sets no mode — there is no
`dpmr_*` key) and `docs/opt-in-features.md`'s FEC table (no dPMR row) agree;
the capture-needs line does not. The code decides.

## Lock, grant, and the missing band plan

`Ingest` is six lines of policy. Strict mode drops unknown types. `IsIdle`
— `Idle` or `Release` — is absorbed silently
(`TestControlChannelSilentOnIdle`). A `StandingServiceStatus` locks with
its `DestID` as `SystemID` (`TestControlChannelEmitsLockOnSiteBroadcast`).
A voice allocation locks too, "even if we haven't seen a SiteBroadcast
yet", then publishes a grant. `maybeLock` keeps a previously learned
`SystemID` when a later state has none, so a grant-first lock followed by a
broadcast does not flap (`TestControlChannelNoRepublishOnSameLockState`),
and `LockedNAC` packs the low 16 bits of the SystemID into the supervisor's
NAC slot.

```go
// internal/radio/dpmr/control.go (shape) — publishGrant
freq := uint32(0)
if c.resolver != nil {
    if hz, err := c.resolver.Frequency(g.Channel); err == nil { freq = hz }
}
c.bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
    System: c.systemName, Protocol: "dpmr",
    GroupID: g.DestID, SourceID: g.SourceID, FrequencyHz: freq, ChannelNum: g.Channel,
    Encrypted: g.Encrypted, Emergency: g.Emergency, At: c.now(),
}})
```

`bandplan.go` names the canonical PMR446 layout —
`LinearBandPlan{BaseHz: 446_006_250, SpacingHz: 6_250, Offset: -1}`, so
channel 1 is 446.006250 MHz — and `TestControlChannelEmitsGrant` resolves
channel 4 through it. `TestControlChannelGrantNoResolverFallsBackToZero`
pins the other branch: no resolver, `FrequencyHz` 0, `ChannelNum` 9. That
other branch is production. `newDPMRPipeline` constructs `dpmr.New` with
`Bus`, `Log`, `SystemName` and `FrequencyHz` and nothing else;
`config.SystemConfig` has `motorola_band_plan`, `p25_band_plan`,
`dmr_band_plan` and `nxdn_band_plan` and no dPMR entry; so every live Mode 3
voice allocation reaches `Engine.HandleGrant` with a zero and is dropped at
its first gate — `dropping grant with zero frequency` — exactly as
[Part 7]({{ '/blog/deep-dives/legacy-family-07-analog-voice-on-trunked-fm/' | relative_url }})
found for EDACS, LTR and MPT 1327. The daemon test is consistent:
`TestDaemonCCDecodesDPMR` synthesises thirty `StandingServiceStatus` CSBKs
with `demod.ModulateC4FM` at 20 sps, 900 Hz, on 446.018750 MHz, and asserts
a `dpmr.LockState` with `SystemID` 0x123456 — a lock, never a followed call.

## The voice path

The traffic side is independent of the control side by design.
`TrafficChannel` runs FS1 and FS2 detectors (tolerance 1), collects
`postSyncDibitsTraffic` = 24 + 144 = 168 dibits after either — a 192-dibit,
80 ms frame minus the sync — skips the 24-dibit CCH ("like NXDN's SACCH,
its call-control content is not needed to render audio") and hands the
144-dibit TCH to `ExtractTCHFrames`, which carves four 72-bit segments
through `DecodeTCHFrame` (`TestTrafficChannelDecodesFS1Frame`, `…FS2Frame`,
`…IgnoresCSBKBurst`, `…SurvivesChunkedFeed`). The composer's
`runDPMRVoiceChain` builds that `TrafficChannel` behind a 900 Hz receiver,
packs each 49-bit payload to seven bytes, and writes it with the
Golay-corrected bit count (`WriteRawFrameWithErrors`); the recorder maps
`"dpmr"` to `"ambe2-dmr"`, the AMBE+2 3600×2450 decoder
([AMBE+2 FEC]({{ '/reference/ambe-plus-2-fec/' | relative_url }})).
`TestComposerRunsDPMRVoiceChain` confirms a `Protocol "dpmr"` grant at
446.1 MHz launches the chain rather than the bypass.

`voice_ambe.go` is where the honesty is loudest. The FEC inside a 72-bit
AMBE+2 frame — Golay(23,12) over C0 and C1, the C0-seeded keystream XORed
onto C1 (`dpmrC1Keystream`), the 12 + 12 + 11 + 14 = 49-bit assembly — is
"a property of the AMBE+2 codec, not of the radio protocol", identical to
the DMR and NXDN paths. The one protocol-specific piece is how 72 on-air
bits map into the four sub-vectors, and `dpmrAMBEDeinterleave` is "a
documented *placeholder* (a direct sequential split C0|C1|C2|C3, mirroring
the NXDN placeholder)", isolated in one function with its encoder inverse in
lock-step. `TestTCHFrameRoundTrip` says what it proves: "everything except
the (capture-unknown) on-air interleave table, which round-trips by
construction here." `TestTCHFrameCorrectsErrors` flips three bits in each of
C0 and C1 and recovers the payload — true of the Golay layer on any table.
The first things a capture confirms, per the file, are this table and "the
3600×2450-vs-2400 codebook choice";
[Part 13]({{ '/blog/deep-dives/legacy-family-13-ambe-chains-with-placeholders/' | relative_url }})
compares the three placeholder chains side by side.

## The rung

dPMR Mode 3 stands lower than NXDN, for reasons each visible in the tree.
The physical layer is **spec-derived and synthetic-verified**: receiver
tests drive a phase ramp (`TestReceiverEmitsDibitsFromPhaseRamp`), the Gray
mapping is pinned to the P25 convention, and the daemon test modulates its
own CSBKs. No sample exists under `samples/dpmr/`, and no test cites an
independent decoder's table for a sync word or CSBK field — NXDN can point
at MMDVMHost and dsdcc for its SACCH; this package points at ETSI section
numbers.

The control layer is **structurally complete and FEC-less**: it locks and
grants on a clean fixture and a clean-enough carrier; on a marginal one,
with an 80-bit window accepted on a 24-dibit sync at tolerance 1, strict
validation is all that stands between a bit error and a phantom grant with
a wrong source ID. The follow-the-call layer is **unwired**: no band plan,
so no frequency, so no call. The voice layer is **placeholder-backed and
experimental**, by the file's label and by `docs/status.md`'s — "wired
end-to-end through the composer but not yet verified on air: each chain's
AMBE interleave table is a documented placeholder".

The capture that moves it is sixth of seven in
`docs/decoder-capture-needs.md`'s priority list: "dPMR / D-STAR voice
(≥ 10 s IQ, 48 kHz, clear voice) — confirms the placeholder AMBE interleave
tables + frame geometry". A Mode 3 control-channel capture with a known
system ID would do for the layers above it what the NXDN harness is waiting
to do; nobody has written a `TestReplayDPMRRealCapture`, because nothing
exists to replay.

### How dPMR shaped the Go code

- **One detector per burst family.** The control path owns FS3; the
  traffic path owns FS1 and FS2; neither sees the other's bursts.
- **Accessors encode the spec's intent.** `AsVoiceGrant` forces `Group`
  for a `VoiceServiceAllocation`; `IsIdle` folds `Release` into silence.
- **Nil resolver is tolerated, zero frequency is not.** `publishGrant`
  publishes with 0; the engine's first gate drops it.
- **The placeholder has an inverse.** `dpmrAMBEInterleave` exists only so
  the round-trip can run; both must change together.

## Where this goes next

dPMR shares its vocoder with DMR and NXDN; the next protocol shares its
vocoder with nobody in the trunked world.
[Part 11]({{ '/blog/deep-dives/legacy-family-11-dstar/' | relative_url }})
takes D-STAR — GMSK at 4800 bps, a 41-byte header inside a 660-bit FEC
shell, a 96-bit DV cadence anchored on the slow-data sync, and the original
AMBE 3600×2400 — the one AMBE-era mode where `FECOff` is still the default
and the file says why.

## FAQ

**What does GopherTrunk decode on a dPMR control channel?**
Mode 3 CSBKs: `ControlChannel.Process` matches the 24-dibit FS3 sync
(0x7DDFFD5F55D5) within one dibit, collects 40 dibits, and parses an 80-bit
block with a 5-bit message type, flags, 24-bit source and destination,
service info and a 16-bit opcode field. A `StandingServiceStatus` locks the
channel; a voice allocation publishes a `Protocol "dpmr"` grant.

**Does the dPMR decoder correct bit errors in a CSBK?**
No. TS 102 658 wraps Mode 3 CSBKs in a short-block cyclic code with a
rate-¾ convolutional outer code, which the package header lists as an
unimplemented deferral — the 80 bits after FS3 are read raw.
`SetStrictValidation` drops CSBKs with an undocumented message type, which
filters misalignment but cannot repair a flipped bit.

**Why are dPMR voice grants dropped by the engine?**
`publishGrant` resolves the CSBK's channel number through an optional
`Resolver` — a `LinearBandPlan` such as PMR446's 446.006250 MHz base,
6.25 kHz spacing, offset −1 — and `newDPMRPipeline` installs none; no dPMR
band-plan config key exists. The grant is published with `FrequencyHz` 0
and `Engine.HandleGrant` drops it with "dropping grant with zero frequency".

**Is dPMR Mode 3 verified on air?**
No, at any layer. The sync words and CSBK layout are spec transcriptions
pinned by self-generated fixtures; `TestDaemonCCDecodesDPMR` locks on
synthesised CSBKs; no capture exists under `samples/dpmr/`; the voice
deinterleave is a documented placeholder. A ≥ 10 s, 48 kHz IQ capture of a
clear Mode 3 voice call is the named gate.

## Series navigation

**Part 10 of 14** · ←
[Part 9: NXDN CAC and Trunking — Interleaver, Puncture, Viterbi and the Soft-Decision Lever]({{ '/blog/deep-dives/legacy-family-09-nxdn-cac-and-trunking/' | relative_url }})
· Next →
[Part 11: D-STAR — GMSK, the 660-Bit Header Shell and the 96-Bit DV Cadence]({{ '/blog/deep-dives/legacy-family-11-dstar/' | relative_url }})
