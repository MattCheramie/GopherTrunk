---
title: "P25 End to End, Part 13: Testing P25 Without a Tower"
description: How GopherTrunk tests a P25 decoder with no transmitter in reach — literal byte vectors that catch what round-trips cannot, impairment-swept synthetic carriers with a hard CI loss budget, committed real-air capture fixtures, and rate-invariance tests that prove where a bug is not.
category: deep-dives
keywords: testing p25 decoder, p25 test vectors, sdr regression testing, synthetic iq impairments, p25 capture fixture, round trip test trap, ber snr sweep ci, replay testing sdr, gophertrunk p25
tags: [p25-end-to-end, p25, testing, replay, fixtures, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 13
---

*Part 13 of **P25 End to End**, a 14-part deep dive that follows North America's
dominant trunking protocol through GopherTrunk — from a raw C4FM carrier to
recorded, named, multi-site voice.
[Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})
ended with a fix deliberately parked behind a capture — which only makes sense
if tests have sharply different powers of proof. This part lays out that
hierarchy for P25: four layers of testing, what each can and cannot catch,
and the bugs that define each boundary. The villain of the whole series —
the test that passes because both sides share the same wrong assumption —
gets its full anatomy here.*

> **TL;DR:** GopherTrunk tests P25 in four layers. **Literal vectors** pin
> wire layouts against independent references — the TSBK SCCB (0x39) parser
> read channel B one byte early and its round-trip test *passed*, because the
> assembler encoded the same wrong layout; the hand-built byte vector in
> `TestParseSecondaryControlChannelBroadcastLayout` fails against the bug.
> **Synthetic streams** add honest RF impairments (`demod.Impairments`:
> multipath FIR, carrier offset + drift, DC spike, IQ imbalance, AWGN) under
> a hard BER-sweep gate with committed loss budgets. **Committed captures**
> (`testdata/mmr-s9-cc.cfile`, `samples/p25/`) are field truth in CI. And
> **rate-invariance tests** (`internal/scanner/ccdecoder/ddc_highrate_test.go`)
> prove where a defect is *not* — the control that settled
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764). None of
> them prove on-air behaviour, which is why the issue-closing policy demands
> a verified symptom, not a green suite.

**Key takeaways**

- **A round-trip test validates consistency, not correctness.** Encoder and
  decoder written by the same hand share the same misreading of the spec —
  the SCCB byte-offset bug rode a green round-trip into production.
- **Synthetic streams are only as honest as their impairments.** An ideal
  modulator output hides gain sensitivity, carrier-offset bias and multipath
  failure; `demod.Impairments` puts the RTL-SDR's sins back in.
- **A committed capture is a regression test with the real world inside it.**
  One second of real control channel (~400 KB) catches level bugs, filter
  mismatches and timing-loop defects no synthetic ever tripped.
- **Some tests exist to prove where a bug is NOT.** The rate-invariance suite
  pins the verified conclusion that the DDC neither creates nor hides an SNR
  deficit, so the next #764-shaped report starts at the right suspect.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Literal wire vectors | pins the 0x39 SCCB layout against SDRTrunk offsets | `internal/radio/p25/phase1/mbt_test.go` (`TestParseSecondaryControlChannelBroadcastLayout`) |
| On-air vector | a real captured TSBK must parse forever | `phase1/tsbk_test.go` (`TestTSBKAcceptsMtAnakieOnAirVector`) |
| RF impairment model | multipath, offset, drift, DC, IQ imbalance, AWGN | `internal/dsp/demod/impair.go` (`demod.Impairments`) |
| Hard demod gate | BER-vs-SNR sweep with committed loss budgets | `phase1/receiver/sweep_test.go` (`TestSweepImplementationLossBudget`) |
| Capture fixtures | real CC slices replayed in CI | `cmd/gophertrunk/replay_realcapture_test.go` (`TestReplayMMRSite9DecodesRealP25`) |
| Drop-in field truth | skip-gated metrics on any contributed capture | `cmd/gophertrunk/p25_realcapture_metrics_test.go` |
| Where a bug is NOT | DDC level/SNR invariance across capture rates | `internal/scanner/ccdecoder/ddc_highrate_test.go` |

## In this post

- **The trap at the bottom** — why round-trips lie, via the SCCB bug.
- **Literal vectors** — independent references, on-air frames.
- **Synthetic streams** — honest impairments and the hard sweep gate.
- **Committed captures** — field truth as a permanent regression.
- **Rate invariance** — tests that prove where a defect is not.
- **The top of the pyramid** — what only air can verify.

## The trap at the bottom: why round-trips lie

Start with the bug
[Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})
introduced — the cleanest specimen of the failure class this whole part is
armour against. The TSBK Secondary Control Channel Broadcast (opcode 0x39)
parser read channel B from payload bytes 4–5; the correct layout puts it at
bytes 5–7, so the old code spliced service class A into the channel field.
And the round-trip test **passed**, for the oldest reason in the book:
`AssembleSecondaryControlChannelBroadcast` encoded the same wrong layout, so
parse(assemble(x)) == x held perfectly while every real site's SCCB decoded
to a phantom channel.

That is the self-consistent-synthetic trap, and it is not a P25 quirk — the
same shape produced the RRC-matched-filter model that passed every synthetic
test while leaving ISI on every real capture (issue #275,
[Part 1]({{ '/blog/deep-dives/p25-end-to-end-01-c4fm-carrier/' | relative_url }})),
and it recurs often enough to have earned
[its own postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).
A test whose two sides share an assumption validates it against nothing.

## Literal vectors: bytes that cannot agree with you

The countermeasure is structural: at least one test per wire format whose
expected bytes were produced by something that is *not your code*. The SCCB
regression is the template —

```go
// internal/radio/p25/phase1/mbt_test.go (shape)
// Pins the 0x39 payload layout against SDRTrunk's bit offsets (RFSS 16-23,
// SITE 24-31, CH A 32-47, SC A 48-55, CH B 56-71, SC B 72-79). The previous
// parser read channel B from bytes 4-5 — one byte early — and its
// round-trip test passed because the assembler encoded the same wrong
// layout. A literal byte vector cannot be fooled that way.
func TestParseSecondaryControlChannelBroadcastLayout(t *testing.T) {
    p := [8]byte{0x01, 0x01, 0x26, 0x9C, 0x70, 0x2D, 0x0E, 0x50}
    s := ParseSecondaryControlChannelBroadcast(p)
    if s.ChannelBID != 2 || s.ChannelBNumber != 3342 {
        t.Errorf("channel B = %d-%d, want 2-3342 (old layout read bytes 4-5: 7-45)",
            s.ChannelBID, s.ChannelBNumber)
    }
    /* … RFSS/site, channel A, both service classes, assemble round-trip … */
}
```

The vector's provenance is the point: those offsets come from SDRTrunk, an
independent implementation proven on air. The test *fails against the old
parser* — the failing-first property every bug fix here must have — and no
future refactor can drift the layout without tripping it.

One rung up sits the even better vector: bytes that came off the air.
`TestTSBKAcceptsMtAnakieOnAirVector` feeds the TSBK pipeline a block a real
transmitter emitted, pinning trellis decode, deinterleave and CRC-CCITT16 to
the world rather than the spec-reading. The same philosophy anchors
[Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})'s
MBT/AMBT decoder to OP25's `process_PDU`, and it is the founding rule of
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }}):
a spec plus an independent decoder beats a spec plus your own inverse.

## Synthetic streams with honest impairments

Literal vectors test parsers, not demodulators. For those you need signal —
and naive "generate a carrier, decode it" re-creates the trap one layer
down, because a mathematically ideal modulator output is a channel no real
capture resembles. GopherTrunk models the sins explicitly:

```go
// internal/dsp/demod/impair.go (shape)
type Impairments struct {
    Multipath         []complex64 // complex channel FIR — simulcast ISI
    FreqOffsetHz      float64     // tuner ppm error
    FreqDriftHzPerSec float64     // LO warm-up wander (issue #492)
    DCOffset          complex64   // the R820T2/E4000 centre spike
    IQGainImbalance   float64     // analog quadrature mismatch
    IQPhaseSkewRad    float64
    SNRdB             float64     // AWGN, seeded for reproducibility
    Scale             float64     // front-end gain level
}
```

Each field is a war story. `Scale` exists because the CQPSK path once locked
only in a narrow RTL-SDR gain window (issue #275's regression report);
`FreqDriftHzPerSec` because a fixed coarse carrier seed cannot track a
warming LO (issue #492); `Multipath` because a flat-fading model cannot
produce simulcast ISI. The harness in `receiver/harness_test.go` modulates a
canonical control-channel stream (`ModulateP25C4FM`, NAC 0x293 at the
production 48 kHz rate), applies an `Impairments`, and runs the **real
receiver** — never an idealised stand-in.

On top of that sits the hard gate from
[Part 12]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }}):
`TestSweepImplementationLossBudget` sweeps injected SNR from 2 to 40 dB on
both demod paths, measures symbol-error rate against closed-form references,
and fails CI if either path regresses past its committed loss budget (6 dB
for CQPSK against coherent QPSK, 27 dB for C4FM against coherent 4-PAM).
The sweep cannot prove the demod is good, but it guarantees it never
silently gets worse — and it validates the SNR *estimator* alongside the
demod, so the diagnostic numbers operators see are themselves under test.

## Committed captures: field truth in the repo

The third layer embeds the real world in CI. `testdata/mmr-s9-cc.cfile` is a
~1.09 s slice of a real P25 control channel — small enough to commit, long
enough for several FSW+NID+TSBK frames — and
`TestReplayMMRSite9DecodesRealP25` replays it through the production chain
asserting decode-yield floors. A second fixture, `mmr-city-cc.cfile`
(420.0125 MHz, NAC 0x164 — the issue #402 site), pins the clean-decode case.
Both are reproducible from their raw recordings via `TestGenerateP25Fixture`
(`p25_make_fixture_test.go`), so fixture generation is itself tested rather
than a one-off script.

For captures the repo can't commit, `samples/p25/` is the drop point:
`TestReplayP25RealCaptureMetrics` (tag `integration`) skips when empty and,
when a `.cfile` + `.metadata.json` pair appears, reports pre-FEC EVM,
estimated SNR, FSW sync-margin and NID/TSBK yields — asserting only the
bounds the metadata declares. This is the instrument Part 12's weak-signal
ask feeds, and the general pattern
[Protocol Decoders Part 12]({{ '/blog/deep-dives/protocol-decoders-12-testing-decoders-without-radios/' | relative_url }})
surveys across every protocol GopherTrunk speaks.

What captures caught that synthetics never did, from this series alone: the
matched filter's ~sps× level mismatch that collapsed the slicer to outer
symbols (issue #275 — invisible when the synthetic pre-scales its levels),
the Gardner gain that only locked on symbol-aligned input (issue #492 —
fixtures start aligned; real captures never do), and the eye asymmetry that
made the adaptive slicer *worse* on the #402 site.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A four-layer test pyramid for P25. The base layer is literal vectors, pinning wire layouts against independent references. Above it, synthetic streams with modelled RF impairments and a hard BER sweep gate. Above that, committed real-air captures replayed in CI. The apex is on-air verification with an operator. An arrow up the side is labelled increasing power of proof, and an arrow down the other side is labelled increasing cost and scarcity. A note at the base recalls that round-trip tests alone validated the SCCB bug.">
  <polygon points="340,18 620,222 60,222" fill="none" stroke="var(--fg-muted)"/>
  <line x1="130" y1="172" x2="550" y2="172" stroke="var(--fg-muted)"/>
  <line x1="200" y1="120" x2="480" y2="120" stroke="var(--fg-muted)"/>
  <line x1="270" y1="70" x2="410" y2="70" stroke="var(--fg-muted)"/>
  <text x="340" y="52" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">on-air A/B</text>
  <text x="340" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="8">operator's own signal</text>
  <text x="340" y="96" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">committed captures</text>
  <text x="340" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">mmr-s9-cc.cfile · samples/p25/</text>
  <text x="340" y="146" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">synthetic streams + impairments</text>
  <text x="340" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="8">demod.Impairments · BER sweep hard gate</text>
  <text x="340" y="196" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">literal vectors</text>
  <text x="340" y="210" text-anchor="middle" fill="var(--fg-muted)" font-size="8">independent references · on-air bytes · failing-first</text>
  <line x1="52" y1="210" x2="130" y2="40" stroke="var(--accent)"/>
  <polygon points="124,44 132,34 133,46" fill="var(--accent)"/>
  <text x="30" y="120" fill="var(--accent)" font-size="9" transform="rotate(-65 30 120)">power of proof →</text>
  <line x1="550" y1="40" x2="628" y2="210" stroke="var(--fg-muted)"/>
  <polygon points="622,204 630,214 618,214" fill="var(--fg-muted)"/>
  <text x="600" y="90" fill="var(--fg-muted)" font-size="9" transform="rotate(65 600 90)">cost &amp; scarcity →</text>
  <text x="340" y="242" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a round-trip test lives below the base — it proved the SCCB bug "correct"</text>
</svg>
<figcaption>Four layers, each catching what the one below cannot — and the round-trip test sits underneath them all, able to validate only its own consistency.</figcaption>
</figure>

## Rate invariance: proving where a defect is not

The fourth kind of test is the strangest and, in this repo's history, one of
the most valuable: tests whose job is to *exonerate* a component.
`internal/scanner/ccdecoder/ddc_highrate_test.go` exists because of the
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764)/#771 saga —
the same P25 site locking from a 2.5 MS/s capture and failing from a
10 MS/s one, in pure offline replay. Three tests pin what the investigation
proved:

| Test | Pins |
|---|---|
| `TestDownconverterC4FMLevelRateInvariant` | the same C4FM channel reaches the receiver at the same symbol-domain level from 2.5 and 10 MS/s (the reported failure was a 10–20× AGC level gap) |
| `TestDownconverterRejectsWidebandNeighbours` | strong off-channel carriers and a DC spike at 10 MS/s must not fold into the ±24 kHz output |
| `TestDownconverterSNRInvariantAcrossRate` | a noisy channel decimated 4:1 by an *independent* resampler shows the same in-channel SNR as the native path |

The third is the decisive control in miniature: on the real captures,
decimating the failing 10 MS/s file with an independent resampler and
replaying it through the proven 2.5 MS/s path reproduced the *same* ~9.5 dB
SNR — the deficit was baked into the samples (front-end phase noise at the
Airspy's native clock), not GopherTrunk's DDC. The suite holds that
conclusion permanently, so the next capture-rate mystery starts at the right
suspect — the "prove it's the signal" method the
[weak-signal series formalised]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}),
kin to the hardware-free rigs of
[RF Front End Part 13]({{ '/blog/deep-dives/rf-front-end-13-testing-radios-without-radios/' | relative_url }}).

## The top of the pyramid: what only air can prove

Every layer below the apex has a blind spot the layer above covers, and the
apex's blind spot is covered by nothing but a transmitter and an operator.
The project learned this the expensive way: #764 was closed twice on
unverified fixes while the symptom was still live, and the repo carries the
scar as policy — a hook asks for human confirmation before any
close-as-completed, PRs prefer `Refs #N` over `Closes #N` until a fix is
verified, and "verified" means a failing-first regression *plus* the reporter
confirming, or the original symptom reproduced and shown resolved. Green CI,
at any layer of this pyramid, is not that.

That is why Part 12's equalizer port waits for a capture, and why the
[testing learn module]({{ '/learn/testing/' | relative_url }}) and this
series keep repeating one sentence: **green synthetic ≠ on-air correct**.
The pyramid doesn't eliminate the need for air; it minimises how often you
need it and maximises what each on-air session proves.

### How the pyramid shaped the Go code

- **Assemblers are exported for tests, and distrusted by policy.** Every
  `Assemble*` has a `Parse*` round-trip — *and* at least one literal-vector
  test per format family, because the round-trip alone once cost a real bug.
- **The impairment model is shared, not per-test.** One `demod.Impairments`
  serves the sweep, harness and regressions, so a new field (like
  `FreqDriftHzPerSec`) upgrades every consumer's honesty at once.
- **Capture tests skip, never rot.** With no capture present the harness
  skips cleanly, so contributed field truth slots into an existing socket.
- **Exonerating tests carry their issue numbers.** `ddc_highrate_test.go`
  names #764/#771 in its comments — the next investigator finds the verdict,
  not just the assertion.

## Where this goes next

Thirteen parts down, one to go.
[Part 14]({{ '/blog/deep-dives/p25-end-to-end-14-playbook/' | relative_url }})
folds the series into what you actually want at the bench: the full layer
map from antenna to WAV, a failure-signature quick reference, and an honest
list of what remains open.

## FAQ

**How do I test a P25 decoder without owning a P25 system?**
In layers: pin wire formats with literal vectors cross-checked against an
independent implementation (OP25, SDRTrunk), synthesise carriers with
realistic impairments through the production receiver, and replay real
captures — your own or contributed ones from `samples/p25/`. GopherTrunk
ships all three, runnable with `make vet test` plus the `integration`-tagged
replay suite.

**Why isn't a passing round-trip test good enough for a protocol parser?**
Because both directions were written from the same reading of the spec. If
the reading is wrong, encoder and decoder agree with each other and disagree
with the air — the SCCB 0x39 bug shipped exactly that way. A literal vector
from an independent source breaks the symmetry.

**What makes a good capture fixture?**
Short, real, annotated: ~1 second of on-channel IQ (about 400 KB as a 48 kHz
`.cfile`) covers several sync/NID/TSBK cycles, and a metadata sidecar with
sample rate, NAC and expected yields turns it into a permanent regression.
Larger contributions go through the git-ignored `samples/` route with
committed sidecars.

**Can synthetic tests measure demod quality, or just catch regressions?**
Both, carefully. The BER sweep measures real implementation loss against
closed-form references — that's where the ~3.85 dB (CQPSK) and ~23.8 dB
(C4FM) figures come from. But its *gate* is deliberately a ceiling, not a
target: it proves the demod never gets worse, while improvement claims are
reserved for capture A/Bs.

**What can only on-air testing prove?**
That a fix addresses the reporter's actual conditions — the impairments you
didn't think to model, the traffic your synthetic never generated, the
hardware you don't own. That's why issue-closing is gated on a verified
symptom, not a green suite.

## Series navigation

**Part 13 of 14** · ←
[Part 12: The Weak-Signal Gap — P1 Voice's Missing Levers]({{ '/blog/deep-dives/p25-end-to-end-12-weak-signal-gap/' | relative_url }})
· Next →
[Part 14: The P25 Playbook]({{ '/blog/deep-dives/p25-end-to-end-14-playbook/' | relative_url }})
