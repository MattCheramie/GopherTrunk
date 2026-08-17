---
title: "Weak-Signal Engineering, Part 1: The Marginal Regime"
description: Between full quieting and no signal lies the marginal regime — where a receiver locks but decodes only a fraction of what it hears — and this opener defines that regime with a real TETRA capture, previews the four levers that roughly doubled GopherTrunk's yield there, and sets the one verdict rule the whole series lives by.
category: deep-dives
keywords: weak signal decoding, marginal signal regime, in-channel snr, crc yield, control channel sync loss, tetra bsch decode, decode yield vs snr, blind equalizer, soft decision fec, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, dsp, tetra, snr, equalizer, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 1
---

*Part 1 of **Weak-Signal Engineering**, a 14-part deep dive into the regime
where most real radio traffic actually lives: not full quieting, not silence,
but the marginal middle — where a receiver locks and then decodes a fraction of
what it hears. Over fourteen posts we work through the four levers that roughly
doubled GopherTrunk's decode yield in that regime — blind equalization, trained
equalization, soft-decision FEC, and diversity combining — and the measurement
discipline that keeps each one honest. The series has a running thread: a single
two-second capture of a struggling TETRA control channel, introduced below,
whose yield number we move part by part.*

> **TL;DR:** The **marginal regime** is where the receiver *locks* but decodes
> only a fraction of the frames it demodulates. GopherTrunk's canonical example
> is a real on-air capture (`testdata/tetra_cc_sync_loss_2s_144k.cs16` in
> `internal/scanner/ccdecoder`): in-channel SNR ~**10 dB**, a solid lock, and
> only ~**12%** of its BSCH synchronisation bursts CRC-clean — while a healthy
> period on the *same site* measures ~18 dB and decodes ~100%. Four levers move
> that number: a blind **`SnapshotCMA`** equalizer, a trained **`SnapshotLMS`**
> equalizer, **soft-decision** channel decoding, and **diversity** combining.
> And one rule governs all of them: **decode yield — CRC-valid frames per
> opportunity — is the only trustworthy verdict.** EVM, SNR, and constellation
> beauty are advisory at best and traps at worst.

**Key takeaways**

- **Marginal means "locks but under-decodes," not "weak."** A no-signal channel
  is easy to diagnose; a channel that locks, decodes 12%, and thrashes the
  resync logic is the hard case — and the common one.
- **The regime is narrow in dB and wide in reality.** The gap between ~100%
  yield and ~12% yield on our thread capture is about 8 dB of in-channel SNR —
  the difference between a fixed antenna on a good day and the same antenna on
  a bad one.
- **Four levers, one method.** Equalize what is linear, train where you know
  the transmitted symbols, keep reliability information the slicer would throw
  away, and combine antennas where the math allows — each lever lands with a
  failing-first test and, where possible, an operator-capture A/B.
- **Yield is the verdict.** Every lever in this series is scored by CRC-valid
  frames per opportunity on real captures, never by how pretty the
  constellation got. Part 2 is entirely about why.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The thread capture | ~2 s of a marginal on-air TETRA CC, 144 kHz cs16 | `internal/scanner/ccdecoder/testdata/tetra_cc_sync_loss_2s_144k.cs16` |
| Yield harness | replay the fixture, count CRC-clean BSCH | `internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go` (`decodeCCBSCHYield`) |
| Blind equalizer | invert linear ISI with no reference | `internal/dsp/equalizer/snapshot_cma.go` (`SnapshotCMA`) — Parts 4–6 |
| Trained equalizer | train on the known midamble, freeze, apply | `internal/dsp/equalizer/snapshot_lms.go` (`SnapshotLMS`) — Part 7 |
| Soft decisions | carry LLRs through depuncture + Viterbi | `internal/radio/framing/soft_tetra.go` (`DecodeRCPCTetraMotherSoft`) — Part 8 |
| Diversity | combine two branches, gated by coherence | `internal/dsp/diversity` (`CrossStats`, `TrackingCalibrator`) — Parts 10–11 |

## In this post

- **What "marginal" actually means** — locked, demodulating, and losing.
- **The capture that runs this series** — one bad night on system 250_013.
- **Why real traffic lives here** — fringe sites, mobiles, band edges.
- **Four levers, one method** — the series map.
- **Yield is the only verdict** — the rule everything else obeys.

## What "marginal" actually means

Every receiver spec sheet implies a binary: above sensitivity you decode, below
it you don't. Reality has a third state, and it is enormous. In the marginal
regime the synchronisation machinery *works* — the correlator finds the training
sequence, the timing loop tracks, the AFC pulls the carrier in — but the
per-frame error correction loses more often than it wins. The receiver is
locked, demodulating symbols at full rate, and throwing most of them away at
the [CRC]({{ '/reference/crc-16-ccitt/' | relative_url }}).

That state is qualitatively different from "no signal," and it fails in a
qualitatively nastier way. A dead channel produces silence and an obvious
diagnosis. A marginal channel produces *partial* everything: partial voice
recordings, a control channel that follows some grants and misses others, and —
worst — sync-tracking logic that keeps being fed just enough valid frames to
believe the lock is real while the failure counters climb. On a TETRA control
channel that pattern has a specific signature GopherTrunk operators have watched
live: `bsch_fail` climbing, `sb_bursts` collapsing, repeated
`tetra: dsp resync (signal-time decode drought)` lines, and finally the 5-second
stale watchdog declaring the channel lost and sending the scanner back to the
hunt. One reporter's one-hour session logged ~210 control-channel transitions
and 11 hard sync losses that way — and the postmortem found *zero* correlation
with CPU or call load. It was never compute starvation. It was this regime.

## The capture that runs this series

When that reporter's daemon lost sync, the `on_cc_sync_loss` auto-recorder did
exactly what it exists to do: it grabbed IQ. The two-second slice checked into
the repo is the series' thread capture, and its header comment is a complete
character sketch:

```go
// internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go (shape)
// marginalCCFixture is a ~2 s / 144 kHz cs16 slice of a real on-air TETRA
// control channel captured by gophertrunk's on_cc_sync_loss auto-recorder
// during a re-acquisition (system 250_013, CC 467.9125 MHz, Airspy). In-channel
// SNR is ~10 dB (a healthy period on the same site measures ~18 dB) and the
// π/4-DQPSK constellation is ISI-smeared. Replayed through the current CC path
// it LOCKS but decodes only ~22 % of its BSCH synchronisation bursts — the
// marginal-yield regime that produced the field's ~210 CC transitions/hour.
const marginalCCFixture = "testdata/tetra_cc_sync_loss_2s_144k.cs16"
```

The numbers to hold onto: the full sync-loss capture decodes ~**22%** of its
BSCH bursts; the two-second fixture slice, replayed through the baseline chain,
manages ~**12%**. A healthy capture from the same site — same antenna, same
Airspy, same software — measures ~**18 dB** in-channel and decodes ~**100%**.
Two more facts matter because of what they rule out: the capture peaks at about
−44 dBFS with no clipping (so this is not front-end overload), and the carrier
sits ~3 kHz off centre, well within the AFC's pull-in. Nothing is *broken*.
The channel is simply about 8 dB worse than it needs to be, and the
[π/4-DQPSK]({{ '/reference/pi-4-dqpsk/' | relative_url }}) constellation is
visibly smeared by inter-symbol interference on top of the noise.

The harness that scores it is deliberately minimal — the same wiring as the
production pipeline with exactly one variable exposed:

```go
// internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go (shape)
// decodeCCBSCHYield hand-wires a TETRA control-channel receiver + decoder at the
// 144 kHz channel rate with the equalizer toggled, feeds the IQ in chunks, and
// returns the BSCH decode counts. The wiring mirrors newTETRAPipeline exactly
// … EXCEPT that EnableEqualizer is a parameter — so a single fixture isolates
// the equalizer as the only variable.
func decodeCCBSCHYield(t *testing.T, iq []complex64, enableEqualizer bool) (ok, fail int64)
```

`ok / (ok + fail)` is the yield, and yield is the number this series moves.

## Why real traffic lives here

It would be comforting to treat the marginal regime as an edge case. It isn't —
for structural reasons. Coverage planning puts sites where *most* users get a
strong signal, which by construction leaves the fringe populated. Mobiles and
handhelds spend their lives behind buildings, in vehicles, and at the bottom of
the fade distribution. Band-edge channels pick up group-delay distortion from
the very filters that protect them. And a scanner operator is usually *not* a
subscriber standing where the coverage map is green — you are listening from
one fixed antenna at whatever distance the geography dealt you.

The practical consequence: the difference between a scanner that "mostly works"
and one that feels professional is almost entirely how it behaves between
roughly 8 and 18 dB of in-channel SNR. Above that band, everything decodes and
the DSP is unheroic. Below it, no software can help and the fix is RF — better
antenna, better siting, the territory of
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}), running
concurrently with this series. In between is where engineering pays, and in
between is exactly where our thread capture sits.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Decode yield versus in-channel SNR is a cliff: near zero yield at low SNR, a steep transition region, and a plateau near one hundred percent at high SNR. The marginal TETRA capture sits on the cliff face at ten decibels and twelve percent yield; the healthy capture from the same site sits on the plateau at eighteen decibels and one hundred percent. An arrow shows the series' four levers moving the cliff left, so the same ten decibel capture lands on the plateau.">
  <line x1="50" y1="20" x2="50" y2="180" stroke="var(--fg-muted)"/>
  <line x1="50" y1="180" x2="650" y2="180" stroke="var(--fg-muted)"/>
  <text x="18" y="30" fill="var(--fg-muted)" font-size="9">yield</text>
  <text x="18" y="42" fill="var(--fg-muted)" font-size="9">(CRC)</text>
  <text x="44" y="34" text-anchor="end" fill="var(--fg-muted)" font-size="9">100%</text>
  <text x="44" y="182" text-anchor="end" fill="var(--fg-muted)" font-size="9">0%</text>
  <text x="590" y="198" fill="var(--fg-muted)" font-size="9">in-channel SNR (dB)</text>
  <polyline points="60,176 160,172 240,160 300,130 350,80 400,44 470,32 640,30" fill="none" stroke="currentColor"/>
  <polyline points="60,174 120,168 180,148 230,104 280,52 330,34 400,31 640,30" fill="none" stroke="var(--accent)" stroke-dasharray="5 3"/>
  <circle cx="312" cy="120" r="5" fill="currentColor"/>
  <text x="312" y="142" text-anchor="middle" fill="var(--fg-muted)" font-size="9">thread capture: ~10 dB, ~12%</text>
  <circle cx="470" cy="32" r="5" fill="var(--accent)"/>
  <text x="470" y="20" text-anchor="middle" fill="var(--accent)" font-size="9">healthy: ~18 dB, ~100%</text>
  <line x1="360" y1="70" x2="300" y2="70" stroke="var(--accent)"/><polygon points="300,66 290,70 300,74" fill="var(--accent)"/>
  <text x="370" y="74" fill="var(--accent)" font-size="9">the four levers move the cliff left</text>
  <text x="350" y="214" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the marginal regime is the cliff face — locked, demodulating, and losing at the CRC</text>
</svg>
<figcaption>Decode yield against in-channel SNR is a cliff, and the marginal regime is its face. The series' levers do not add signal — they move the cliff left, so the same 10 dB capture lands on the plateau.</figcaption>
</figure>

## Four levers, one method

[SDR Internals Part 8]({{ '/blog/deep-dives/sdr-internals-08-equalization-diversity-fft/' | relative_url }})
surveyed equalization and diversity in a single episode and promised each was
"worth its own series." This is that series. The four levers, and where each
one lives in the fourteen parts:

| Lever | What it exploits | Measured effect | Parts |
|---|---|---|---|
| Blind equalization (`SnapshotCMA`) | PSK's constant envelope — no reference needed | CC fixture ~12% → ~100% BSCH; voice bursts 410 → 778 | 4, 5, 6 |
| Trained equalization (`SnapshotLMS`) | known training sequences (midambles) | synthetic multipath: 13% → 0% payload bit-error | 7 |
| Soft decisions | reliability the hard slicer discards | recovered ~70% of a marginal call's failed bursts | 8, 9 |
| Diversity (MRC) | two antennas, coherence-gated combining | scored per-capture by CRC-clean BSCH | 10, 11 |

The remaining parts are the method around the levers: Part 2 on the metrics
that lie, Part 3 on what a linear channel is and is not, Part 12 on
experimental design (how [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)
was proven to be the *signal's* fault, not the DSP's), Part 13 on the one
GopherTrunk decode path that still has none of these levers, and Part 14 as the
assembled playbook. The concurrently-running
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }}) series
is the protocol-level case study these levers were proven on; here we keep the
treatment cross-protocol, because nothing in an FIR tap knows what a TETRA is.

## Yield is the only verdict

State the rule once, up front, because every later part leans on it: **the only
trustworthy measure of a weak-signal lever is decode yield — CRC-valid frames
per decode opportunity — on real captures.** Not EVM. Not SNR. Not how round the
constellation looks. GopherTrunk learned this the expensive way: a
numerically-unstable equalizer variant once collapsed differential EVM from 34%
to 8% — a spectacular apparent win — while CRC yield stayed exactly **zero**.
The signal got beautiful and carried no information. Part 2 dissects that
failure and the related trap from
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764), where the
*worse* capture had the *higher* wideband carrier SNR.

The companion rule is the testing discipline this project has carried since
[#771](https://github.com/MattCheramie/GopherTrunk/issues/771): every lever
lands with a **failing-first regression test** — one that demonstrably fails
against the old code — and, wherever an operator can supply one, an A/B on
their own capture. A synthetic test that encodes and decodes with the same
assumptions can pass forever while the air disagrees; the
[postmortem series]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
has already met that trap, and this series will keep meeting it. When we say
the blind equalizer takes the thread capture from ~12% to ~100%, that claim is
`TestTETRACCPipelineEqualizer`-shaped: same fixture, one boolean, two counts.

## Where this goes next

Before touching a single tap weight we have to fix the measurement problem,
because a wrong metric will cheerfully steer months of work into a ditch.
[Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
is about the metrics that lie: what EVM actually measures and where it's
measured, how an equalizer can make EVM collapse while decoding nothing, why
the worse #764 capture had the better carrier SNR — and the short list of
numbers you can actually trust.

## FAQ

**How do you measure "in-channel SNR" — isn't SNR one of the metrics you just said not to trust?**
It's measured in the channel bandwidth after the channel-select filter, not on
a wideband FFT — and it's *advisory*, useful for characterizing a capture and
placing it on the cliff. The rule is narrower than "never look at SNR": never
use SNR (or EVM) as the *verdict* on whether a change helped. The verdict is
yield. See [noise and SNR]({{ '/learn/rf-sdr/noise-and-snr/' | relative_url }})
for the measurement itself.

**If the capture peaks at −44 dBFS, why not just add gain?**
Raising front-end gain raises signal and noise together once you're past the
point where the front end's own noise figure dominates — and the thread
capture's ~10 dB is in-channel SNR, which more gain does not improve. The weak
front-end level is a real RF condition worth fixing (better antenna, less
feedline loss), but it is a separate problem from the ISI smear the equalizer
addresses. The two compound; fixing either helps.

**Is the marginal regime a TETRA thing?**
No. TETRA is the case study because that's where the operator captures came
from, but the regime is universal: the same locks-but-under-decodes behaviour
shows up on P25 Phase 1 voice (Part 13's subject), DMR, and anything else with
sync acquisition cheaper than payload FEC. The levers are protocol-agnostic —
they operate on complex symbols and LLRs, not on protocol fields.

**Why does the fixture decode ~12% when the full capture decodes ~22%?**
The fixture is a two-second slice of the worst of it, chosen to be a compact,
reproducible regression target. The full capture averages in some healthier
seconds. Both numbers describe the same regime; the fixture's baseline is
simply the harder version, which makes it the better test.

**Couldn't the resync storm be fixed by making the sync logic more patient?**
Patience was already correct: the investigation confirmed the signal-time
resync design was untouched precisely because the losses had zero correlation
with load. Tuning sync thresholds to tolerate a 12%-yield channel just delays
the inevitable and slows recovery on genuinely dead channels. The fix that
mattered raised the yield itself — which is the whole thesis of this series.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: Metrics That Lie — EVM vs CRC Yield]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
