---
title: "The Legacy Family End to End, Part 9: NXDN CAC and Trunking — Interleaver, Puncture, Viterbi and the Soft-Decision Lever"
description: "From 150 dibits after an NXDN frame sync to a voice grant: the §4.5.1.1 CAC chain (155-bit info block, CRC-16, K=5 rate-½ code, 50-of-350 puncture, 25×12 interleaver) in its three ViterbiMode shapes, the opt-in per-bit soft Viterbi that lifts CAC CRC yield from 26/200 to 151/200 on the synthetic channel, why nxdn_afc stays off, the RCCH state machine with its band plan and drought guard, and the GT_NXDN_* replay harness waiting for a capture."
category: deep-dives
keywords: nxdn cac decode, nxdn viterbi k5 puncture interleaver, nxdn_viterbi_mode spec, nxdn_soft_decision, nxdn_afc opt in, nxdn rcch vcall_assgn grant, nxdn_band_plan, TestReplayNXDNRealCapture, nxdn resync guard, gophertrunk legacy family end to end
tags: [legacy-family-end-to-end, nxdn, fec, viterbi, trunking, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Legacy Family End to End"
series_part: 9
---

*Part 9 of **The Legacy Family End to End**, a 14-part deep dive through
the protocols the P25, DMR and TETRA series left out, each placed honestly
on one verification ladder.
[Part 8]({{ '/blog/deep-dives/legacy-family-08-nxdn-physical-layer/' | relative_url }})
stopped at a stream of dibits and a LICH that says "this is a control
channel". This part decodes what follows — the Common Access Channel's
convolutional shell, the three ways the code can open it, the
soft-decision lever ported from TETRA, the one carrier-tracking stage that
had to stay off — and the state machine that turns an RCCH message into a
lock, a topology and a grant.*

> **TL;DR:** On an NXDN RCCH the 150 dibits after the LICH are 300 channel
> bits. `nxdn.DecodeCACChannel` inverts NXDN-TS-1-A §4.5.1.1: 25×12
> deinterleave → depuncture the 50 positions a `1111111 / 1011101` period-7
> matrix dropped (`framing.DepunctureMark`) → `framing.ViterbiK5` over 175
> stages (g1 0x19, g2 0x17) → strip 4 tail bits → verify a bit-level CRC-16
> (0x1021, init 0xFFFF) over the 155-bit info block (8 SR + 144 L3 + 3
> Null). `ControlChannel.Process` selects that chain as `ViterbiSpec`
> (`nxdn_viterbi_mode` empty or `spec`), keeping `ViterbiOn` (92 dibits,
> bare K=5) and `ViterbiOff` (44 raw dibits) for older fixtures.
> `nxdn_soft_decision` carries two LLRs per dibit through `ProcessSoft`
> into `DecodeCACChannelSoft` (`framing.ViterbiK5Soft`): CAC CRC yield
> 26/200 hard → 151/200 soft at σ = 0.7. `nxdn_afc` stays off because the
> unwhitened CAC's constant-dibit runs drag a plain coarse tracker onto the
> data mean. `IngestFrame` locks on `SITE_INFO` / `CCH`, resolves
> `VCALL_ASSGN` through `nxdn_band_plan`, and a 2 s `resyncGuard`
> reacquires on decode drought. Rung: **capture-gated** —
> `TestReplayNXDNRealCapture` is written; no capture has run through it.

**Key takeaways**

- **Only the spec chain can light up on air.** `ViterbiOff` and
  `ViterbiOn` are fixture layouts whose comments say live CAC frames
  "typically fail their CRC and the adapter silently drops them".
- **Soft decision is the portable lever for the C4FM family.** Complex
  equalizers do not model a post-discriminator channel; per-bit LLRs into a
  soft Viterbi do.
- **An always-on AFC was tried, measured, and made opt-in.** NXDN's CAC
  has no whitening; a zero-heavy payload encodes to constant runs the
  tracker followed, collapsing CAC CRC yield to zero on the SITE_INFO
  fixture.
- **A grant needs a band plan, and the code says so.** Without
  `nxdn_band_plan` a `VCALL_ASSGN` is dropped with a WARN
  (`TestControlChannelDropsGrantWithoutBandPlan`).

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| CAC coding | 155 info → +CRC-16 → +4 tail → K=5 R=½ → −50 puncture → 25×12 interleave → 300 bits | `internal/radio/nxdn/cac_channel.go` |
| Modes | `ViterbiOff` 84 / `ViterbiOn` 132 / `ViterbiSpec` 158 post-sync dibits | `process.go`, `control.go` (`ParseViterbiMode`) |
| Soft path | LLR pairs per dibit → `ProcessSoft` → `DecodeCACChannelSoft` | `receiver.go` (`SoftDibitSink`), `cac_channel_soft.go`, key `nxdn_soft_decision` |
| Carrier AFC | post-clock `demod.CoarseAFC`, opt-in | `receiver.go` (`Options.EnableAFC`), key `nxdn_afc` |
| RCCH dispatch | `SITE_INFO` → lock + topology, `CCH` → lock, `VCALL_ASSGN` → grant | `control.go` (`IngestFrame`, `publishGrant`), `cac.go` (`RCCHType`) |
| Band plan | `LinearBandPlan` / `TableBandPlan` from `nxdn_band_plan` | `bandplan.go` (`ResolverFromPlan`) |
| Drought guard | 2 s of signal with no CRC-clean CAC → `rx.Reset` + `cc.ResyncReset` | `ccdecoder/pipelines.go`, `resyncguard.go` (`nxdnResyncWindow`) |
| Harness | `GT_NXDN_IQ`, `_IQ_RATE`, `_SOFT`, `_AFC`, `_ALLOW_EMPTY` | `cmd/gophertrunk/nxdn_realcapture_test.go` |

## In this post

- **Three ways to read a CAC** — the modes, their frame lengths, and which one is real.
- **The §4.5.1.1 chain** — sizes, generators, the puncture matrix, the interleaver, the CRC.
- **The soft-decision lever** — LLRs from the slicer, erasures at the punctures, measured yield.
- **Why `nxdn_afc` is off** — unwhitened runs and the data-mean drift.
- **From RCCH message to grant** — lock, topology, band plan, and the drought guard.
- **The harness and the rung** — what `TestReplayNXDNRealCapture` would say.

## Three ways to read a CAC

`ControlChannel.Process` counts dibits after each outbound FSW, and how
many depends on `ViterbiMode`
([CAC reference]({{ '/reference/nxdn-cac/' | relative_url }})):

```go
// internal/radio/nxdn/process.go
const postSyncDibits            = 84       // ViterbiOff: 8 LICH + 32 SACCH (skipped) + 44 raw CAC
const postSyncDibitsViterbi     = 8+32+92  // ViterbiOn:  … + 92 K=5-encoded dibits (88 info + 4 tail, ×2)
const postSyncDibitsViterbiSpec = 8+150    // ViterbiSpec: 8 LICH + 150 CAC = 300 channel bits (§4.6)
```

`ViterbiOff` reads 44 dibits = 88 raw information bits into the byte-level
`ParseCAC` — right for a fixture that placed CAC bits on the wire uncoded;
on real air, its comment says, "the CAC CRC almost always fails and the
adapter silently drops the frame". `ViterbiOn` collects 92 dibits and runs
a bare K = 5 rate-½ decode with no deinterleave or depuncture, the layout
older MMDVMHost / DSDcc fixtures used. `ViterbiSpec` collects 150 and runs
the full chain. `ParseViterbiMode` maps the empty string and `spec` to
`ViterbiSpec`, `off` and `on` to the legacy modes, and garbage to
`ViterbiSpec` with `ok = false` so `newNXDNPipeline` warns. The bare
`NewControlChannel` still zero-values to `ViterbiOff`
(`TestSetViterbiModeRetainsViterbiOffDefault`) — the in-package discipline
from `docs/opt-in-features.md`: the connector's `Parse*Mode` makes the
operator default spec-correct while direct callers keep the legacy
behaviour.

## The §4.5.1.1 chain

`cac_channel.go` lays the chain out as arithmetic. One CAC burst carries
152 user bits — an 8-bit SR and 144 bits of L3 data — padded by three Null
zeros to `CACInfoBits` = 155. A 16-bit CRC-CCITT covers those 155 bits
(polynomial 0x1021, init 0xFFFF, evaluated bit by bit because 155 is not
byte-aligned; `TestCACCRC16SanityAgainstByteWiseCRC` anchors it to
`framing.CRCCCITT`). Four zero tail bits flush the encoder, so 175 bits
enter a K = 5 rate-½ code with g1(D) = 1 + D³ + D⁴ (0x19) and g2(D) =
1 + D + D² + D⁴ (0x17) — the SACCH's primitive — producing 350 bits.

Then the puncture. The period-7 matrix keeps every G1 bit (`1111111`) and
drops G2 at sub-columns 1 and 5 (`1011101`); 2 drops × 25 periods = 50
positions, enumerated by `computeCACPuncturePositions` as `2i+1` for every
encoder step with `i mod 7 ∈ {1,5}` (`TestCACPuncturePositionsMatchMatrix`).
350 − 50 = `CACChannelBits` = 300. Finally a 25 × 12 block interleaver,
written row by row and read column by column:
`channel[k] = pre[(k % 25) × 12 + (k / 25)]`
(`TestCACInterleavePermIsBijection`). A package `init` panics if the sums
drift — 50 positions, 175 pre-encode bits, 2 × 175 − 50 = 300 — so a wrong
edit fails at load, not on air.

<figure class="lab-figure">
<svg viewBox="0 0 680 196" width="680" height="196" role="img" aria-label="The NXDN CAC outbound coding chain as a row of boxes with bit counts: 152 user bits plus 3 null make 155; a 16-bit CRC makes 171; 4 tail bits make 175; the K equals 5 rate one half encoder doubles it to 350; the puncture drops 50 to reach 300; a 25 by 12 interleaver reorders the 300 channel bits sent as 150 dibits. A lower row shows the receive side with two branches after deinterleave: a hard branch where punctured positions become DepunctureMark and ViterbiK5 decodes, and a soft branch where they become zero-valued erasures and ViterbiK5Soft decodes per-bit LLRs; both end at the CRC-16 verify. A note gives the measured yield at sigma 0.7: 26 of 200 hard, 151 of 200 soft.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">CAC outbound, NXDN-TS-1-A §4.5.1.1</text>
  <rect x="8" y="26" width="84" height="30" fill="none" stroke="currentColor"/>
  <text x="50" y="40" text-anchor="middle" fill="currentColor" font-size="8">8 SR + 144 L3</text>
  <text x="50" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">+3 Null = 155</text>
  <line x1="92" y1="41" x2="104" y2="41" stroke="currentColor"/>
  <rect x="104" y="26" width="70" height="30" fill="none" stroke="currentColor"/>
  <text x="139" y="40" text-anchor="middle" fill="currentColor" font-size="8">CRC-16</text>
  <text x="139" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">0x1021 · FFFF → 171</text>
  <line x1="174" y1="41" x2="186" y2="41" stroke="currentColor"/>
  <rect x="186" y="26" width="60" height="30" fill="none" stroke="currentColor"/>
  <text x="216" y="40" text-anchor="middle" fill="currentColor" font-size="8">4 tail</text>
  <text x="216" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">→ 175</text>
  <line x1="246" y1="41" x2="258" y2="41" stroke="currentColor"/>
  <rect x="258" y="26" width="96" height="30" fill="none" stroke="var(--accent)"/>
  <text x="306" y="40" text-anchor="middle" fill="var(--accent)" font-size="8">K=5 R=½</text>
  <text x="306" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">g1 0x19 g2 0x17 → 350</text>
  <line x1="354" y1="41" x2="366" y2="41" stroke="currentColor"/>
  <rect x="366" y="26" width="96" height="30" fill="none" stroke="var(--accent)"/>
  <text x="414" y="40" text-anchor="middle" fill="var(--accent)" font-size="8">puncture</text>
  <text x="414" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">1111111/1011101 → 300</text>
  <line x1="462" y1="41" x2="474" y2="41" stroke="currentColor"/>
  <rect x="474" y="26" width="96" height="30" fill="none" stroke="currentColor"/>
  <text x="522" y="40" text-anchor="middle" fill="currentColor" font-size="8">interleave 25×12</text>
  <text x="522" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">row-write, column-read</text>
  <line x1="570" y1="41" x2="582" y2="41" stroke="currentColor"/>
  <rect x="582" y="26" width="90" height="30" fill="none" stroke="currentColor"/>
  <text x="627" y="40" text-anchor="middle" fill="currentColor" font-size="8">300 ch bits</text>
  <text x="627" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="7">150 dibits on air</text>
  <line x1="20" y1="76" x2="660" y2="76" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="340" y="92" text-anchor="middle" fill="currentColor" font-size="9" font-weight="bold">receive: DecodeCACChannel (hard) · DecodeCACChannelSoft (LLRs)</text>
  <rect x="8" y="104" width="110" height="26" fill="none" stroke="currentColor"/>
  <text x="63" y="121" text-anchor="middle" fill="currentColor" font-size="8">deinterleave 25×12</text>
  <line x1="118" y1="117" x2="140" y2="117" stroke="currentColor"/>
  <rect x="140" y="104" width="150" height="26" fill="none" stroke="currentColor"/>
  <text x="215" y="121" text-anchor="middle" fill="currentColor" font-size="8">hard: 50 × DepunctureMark</text>
  <rect x="140" y="150" width="150" height="26" fill="none" stroke="var(--accent)"/>
  <text x="215" y="167" text-anchor="middle" fill="var(--accent)" font-size="8">soft: 50 × 0.0 erasure</text>
  <line x1="130" y1="117" x2="130" y2="163" stroke="var(--accent)"/>
  <line x1="130" y1="163" x2="140" y2="163" stroke="var(--accent)"/>
  <line x1="290" y1="117" x2="312" y2="117" stroke="currentColor"/>
  <rect x="312" y="104" width="130" height="26" fill="none" stroke="currentColor"/>
  <text x="377" y="121" text-anchor="middle" fill="currentColor" font-size="8">ViterbiK5 · 175 stages</text>
  <line x1="290" y1="163" x2="312" y2="163" stroke="var(--accent)"/>
  <rect x="312" y="150" width="130" height="26" fill="none" stroke="var(--accent)"/>
  <text x="377" y="167" text-anchor="middle" fill="var(--accent)" font-size="8">ViterbiK5Soft · 175</text>
  <line x1="442" y1="117" x2="464" y2="140" stroke="currentColor"/>
  <line x1="442" y1="163" x2="464" y2="140" stroke="var(--accent)"/>
  <rect x="464" y="127" width="110" height="26" fill="none" stroke="currentColor"/>
  <text x="519" y="144" text-anchor="middle" fill="currentColor" font-size="8">strip tail · CRC-16</text>
  <line x1="574" y1="140" x2="596" y2="140" stroke="currentColor"/>
  <rect x="596" y="127" width="76" height="26" fill="none" stroke="currentColor"/>
  <text x="634" y="144" text-anchor="middle" fill="currentColor" font-size="8">155 info · ok</text>
  <text x="340" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">σ = 0.7 AWGN, 200 bursts: hard CRC-ok 26 · soft CRC-ok 151</text>
</svg>
<figcaption>The encode chain as sizes, and the two receive branches that share everything but the metric. Soft decoding changes what a punctured position and a weak bit are worth to the Viterbi — and that is worth most of the yield on a noisy channel.</figcaption>
</figure>

`DecodeCACChannel` inverts it and returns the 155-bit info block with a
CRC flag (`TestDecodeCACChannelCorrectsSingleBitError`,
`TestDecodeCACChannelDetectsHeavyCorruption`). `packCACBlockFromInfo` then
bridges to the older parser: drop the SR, pack the first 72 L3 bits — the
8-bit RCCH type and 64-bit payload — into 9 bytes, and synthesise the inner
CRC the 11-byte `ParseCAC` layout expects, "a no-op here" because the outer
CRC already validated the block. `ParseCAC` yields a `CACMessage` with an
`RCCHType` — `VCALL` 0x01, `VCALL_ASSGN` 0x04, `DCALL` 0x09, `SITE_INFO`
0x3C, `CCH` 0x3F among them.

One honest note belongs here. The chain is transcribed from the spec, and
`samples/nxdn/README.md` names what the round-trip in `process_spec_test.go`
cannot catch: "bit-ordering / endianness mismatches against on-air
transmitters", vendor forks that "diverge slightly in puncture index
ordering", and the corrector's noise margin. One labelled RCCH burst closes
all three — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
named in the README, not discovered later.

## The soft-decision lever

`docs/protocol-feature-parity.md` records why NXDN got soft decision and not
an equalizer: the TETRA levers that doubled yield on ISI-smeared captures
are complex linear equalizers, and after an FM discriminator "a multipath
channel is no longer a linear complex convolution". The portable lever is
soft-decision FEC
([soft decision]({{ '/reference/soft-decision/' | relative_url }})), in
three pieces. The receiver derives two log-likelihood ratios per dibit from
the 4-level soft symbol, LLR > 0 ⇒ bit 0:

```go
// internal/radio/nxdn/receiver/receiver.go (shape) — SoftDecision path
r.llrBuf[2*i]   = y                    // MSB: sign axis, distance from 0
r.llrBuf[2*i+1] = r.llrThreshold - ay  // LSB: magnitude axis, t − |y|, t = 2·slicerScale/3
r.softDibitSink(r.dibits, r.llrBuf, r.dibitBase)
```

`TestReceiverSoftDibitSinkContract` pins the contract — two LLRs per dibit,
index-aligned, sign-consistent, the hard sink never fired.
`ControlChannel.ProcessSoft` keeps the LLRs in lockstep with the frame it
collects and falls back to the hard `Process` for any chunk where
`len(soft) != 2*len(dibits)` — the TETRA `StashSoft` discipline.
`DecodeCACChannelSoft` is `DecodeCACChannel` with float inputs: the 50
punctured positions become 0.0 erasures, and `framing.ViterbiK5Soft` runs
the same 175 stages.

The numbers are on the synthetic channel and are the ones the parity doc
quotes. `TestDecodeCACChannelSoftBeatsHardOnNoisyChannel` adds Gaussian
noise at σ = 0.7 to the ideal ±1 LLRs of 200 random bursts, hard-slices one
copy, and counts CRC passes: **26/200 hard, 151/200 soft**.
`TestProcessSoftLocksWhereHardFails` raises σ to 0.9, feeds 30 SITE_INFO
bursts through both paths on one seeded channel, and requires the hard path
to lock **zero** times while the soft path locks — the failing-first shape;
`TestProcessSoftCleanMatchesHard` keeps the no-harm side. The
`cac_channel_soft.go` comment gives the primitive-level figure: ~8× fewer
info-bit errors where the hard path loses one bit in eleven. It ships
**off by default** — `nxdn_soft_decision: on` in `newNXDNPipeline` and the
widebandt2 tap alike — under the posture TETRA's `tetra_traffic_lms`
shipped: a synthetic gain ships a lever; only an operator's capture A/B
pulls it by default.

## Why `nxdn_afc` is off

The other port from the DMR receiver is the post-clock `CoarseAFC` of
issue #836 — subtract the DC bias a tuner's ppm error leaves on the
discriminator. DMR and P25 run it always-on; NXDN runs it only with
`nxdn_afc: on`, and the receiver's `Options.EnableAFC` comment is the
finding: NXDN's CAC carries **no air-interface whitening**, so a zero-heavy
L3 payload convolutionally encodes to long constant-dibit runs, and "the
plain coarse tracker drifts onto that data mean (the issue #402 mode) — an
always-on port collapsed CAC CRC yield to zero on the repo's own synthetic
SITE_INFO round-trip". That fixture — `buildNXDNSpecEncodedDibits`, a
SITE_INFO whose 72 trailing L3 bits are all zero — is exactly the payload
that produces the runs, and it now pins the default-off path. The flag-on
path is pinned the other way: `TestReceiverIsCarrierOffsetInvariant`
rotates a balanced 600-dibit stream by 400 Hz and requires ≥ 0.95 agreement
with the unrotated decode; the DMR port measured ~0.99 collapsing to ~0.79
without the stage. The parity doc's conclusion: turn it on for a rig with a
real tuner error; a default change needs a mistuned NXDN capture and
"likely the P25-style decision-directed pairing".

## From RCCH message to grant

`IngestFrame` receives a parity-clean `RFChControl` LICH and a CRC-clean
`CACMessage`, stamps `lastActivityNano`, bumps `framesDecoded`, and
dispatches. `SITE_INFO` parses LocationID, SiteID and SystemID, feeds the
`topologyModel` (first non-zero value wins) and locks with a `LockState`
whose `LockedNAC` is the SiteID; `CCH` locks with identity empty;
`VCALL_ASSGN` becomes a grant:

```go
// internal/radio/nxdn/control.go (shape) — publishGrant
if c.bandPlan == nil {
    c.log.Warn("nxdn: voice grant dropped; no band plan configured", …); return
}
freq, err := c.bandPlan.Frequency(a.Channel)   // LinearBandPlan or TableBandPlan
if err != nil { c.log.Warn("nxdn: voice grant dropped; channel outside band plan", …); return }
c.bus.Publish(events.Event{Kind: events.KindGrant, Payload: trunking.Grant{
    System: c.systemName, Protocol: "nxdn",
    GroupID: uint32(a.GroupAddress), SourceID: uint32(a.SourceID),
    FrequencyHz: freq, ChannelNum: a.Channel, At: now(),
}})
```

Unlike the FM-era four of
[Part 7]({{ '/blog/deep-dives/legacy-family-07-analog-voice-on-trunked-fm/' | relative_url }}),
NXDN's resolver *is* wired: `newNXDNPipeline` calls
`SetBandPlan(nxdn.ResolverFromPlan(opts.System.NXDNBandPlan))`, and
`nxdn_band_plan` takes one of `linear` (`base_hz`, `spacing_hz`, `offset` —
`offset: 1` for sites numbering from channel 1, so `TestLinearBandPlanResolves`
maps channel 1 → 461.000 MHz and 2 → 461.0125 at 12.5 kHz) or `table`
(`channel` / `freq_hz`). Without it the channel locks, decodes and warns
per grant. `VCallAssignPayload`'s comment keeps the ladder honest: the
offsets "match the common Type-C trunked variant" and "are structural and
not yet validated against an on-air capture".

`LastActivityNano` is the heartbeat the pipeline's `resyncGuard` compares
across chunks: after `nxdnResyncWindow` = 2 s of *processed signal* — "≈ 25
missed frames" — with no CRC-clean decode, `nxdnPipeline.Process` calls
`rx.Reset()` and `cc.ResyncReset()` and logs `nxdn: dsp resync (signal-time
decode drought; reacquiring from centre)`. The window counts samples, not
wall clock, so a starved goroutine cannot trip it, and `ResyncReset` exists
because `Process` keys on absolute dibit indices — a receiver reset under a
stale mid-frame countdown would splice pre- and post-reset dibits into one
garbage frame. The widebandt2 NXDN tap wraps the same guard as a
`droughtGuardReceiver`, and `DecodedFrames` is the counter the engine uses
to gate per-channel power logging.

## The harness and the rung

`cmd/gophertrunk/nxdn_realcapture_test.go` is the NXDN analogue of
`TestDMRIPSCReplay`, written for a file that does not exist. `GT_NXDN_IQ`
points at a cs16 capture, `GT_NXDN_IQ_RATE` gives its rate (default
48 000); the test builds `ccdecoder.NewDownconverter(inRate, 48000)` as the
daemon does, runs the production receiver at 1800 Hz, and feeds every dibit
batch to the `ControlChannel` in `ViterbiSpec` (locks and grants) and to a
standalone FSW/CAC slicer counting `fsw_hits`, `cac_total`, `cac_crc_ok`,
`cac_parsed` and a histogram of RCCH types. `GT_NXDN_SOFT=1` adds
`cac_crc_ok_soft` off the same stream — the opt-in's A/B metric;
`GT_NXDN_AFC=1` enables the AFC; zero FSW hits fail with "check
GT_NXDN_IQ_RATE / tuning / capture" unless `GT_NXDN_ALLOW_EMPTY=1`
downgrades it to a weak-signal baseline. Above it sits Part 8's daemon
gate, `TestDaemonCCDecodesNXDNRealAir`, asserting `LockState` against a
`samples/nxdn/` sidecar's `system_id`, `site_id` and `center_freq_hz`.

So the rung for everything here is **capture-gated**: the chain is
spec-transcribed and synthetic-verified, the soft lever synthetic-measured
and shipped off, the AFC synthetic-measured and shipped off for a reason,
the state machine and band plan unit-tested, and the two harnesses that
would move any of it waiting. The parity doc's list of what was
deliberately *not* changed — "NXDN voice deinterleave / scramble / CAC
structural changes. All flagged UNVERIFIED ON AIR placeholders;
capture-gated" — is the same sentence from the other side.

### How the CAC shaped the Go code

- **Puncture positions are computed, not typed.** The 50 indices derive
  from the matrix, and `init` panics on a wrong count.
- **Hard and soft share one geometry.** `cacInterleavePerm` and
  `cacPuncturePositions` serve both; only the erasure value and the Viterbi
  differ.
- **The soft path falls back, never desyncs.** A mismatched LLR slice
  sends the whole chunk to `Process`.
- **The legacy parser stayed.** `packCACBlockFromInfo` re-synthesises the
  11-byte block so `ParseCAC` and every RCCH accessor are untouched.

## Where this goes next

NXDN's European sibling shares its 4FSK philosophy and halves its symbol
rate.
[Part 10]({{ '/blog/deep-dives/legacy-family-10-dpmr-mode-3/' | relative_url }})
takes dPMR Mode 3 — three 48-bit frame syncs, an 80-bit CSBK read without
the FEC the spec puts around it, a 6.25 kHz channel at 2400 symbols per
second — and the same resolver gap Part 7 found in the FM-era packages.

## FAQ

**What does `nxdn_viterbi_mode: spec` do?**
It selects `ViterbiSpec`, the full NXDN-TS-1-A §4.5.1.1 outbound CAC chain:
150 dibits after the LICH are deinterleaved 25×12, depunctured at 50
positions, Viterbi-decoded at K = 5 over 175 stages, tail-stripped and
CRC-16-verified to a 155-bit info block. It is the default; `off` and `on`
are fixture layouts that do not decode real air.

**How much does `nxdn_soft_decision` help?**
On the synthetic AWGN channel at σ = 0.7, CAC CRC yield rises from 26/200
hard to 151/200 soft, and at σ = 0.9 the soft control channel locks where
the hard one never does. It ships off; A/B a capture with `GT_NXDN_IQ=…
GT_NXDN_SOFT=1 go test ./cmd/gophertrunk -run TestReplayNXDNRealCapture -v`
and compare `cac_crc_ok` with `cac_crc_ok_soft`.

**Why is `nxdn_afc` off when DMR and P25 run the same AFC always-on?**
Because NXDN's CAC has no air-interface whitening: a zero-heavy L3 payload
encodes to long constant-dibit runs, and the plain coarse tracker follows
that data mean instead of the carrier — an always-on port collapsed CAC CRC
yield to zero on the synthetic SITE_INFO fixture. Enable it only for a rig
with a real tuner frequency error.

**Why are NXDN voice grants dropped with "no band plan configured"?**
A `VCALL_ASSGN` carries a traffic-channel number, not a frequency.
`publishGrant` resolves it through `nxdn_band_plan` — one `linear` or
`table` block per system — and without a plan, or for a channel outside it,
logs a WARN and publishes nothing, because a grant with no frequency cannot
be followed.

## Series navigation

**Part 9 of 14** · ←
[Part 8: NXDN Physical Layer — 4FSK at 4800, FSW, LICH and Scrambling]({{ '/blog/deep-dives/legacy-family-08-nxdn-physical-layer/' | relative_url }})
· Next →
[Part 10: dPMR Mode 3 — FS1/FS2 Syncs, CSBK Trunking and the 6.25 kHz Channel]({{ '/blog/deep-dives/legacy-family-10-dpmr-mode-3/' | relative_url }})
