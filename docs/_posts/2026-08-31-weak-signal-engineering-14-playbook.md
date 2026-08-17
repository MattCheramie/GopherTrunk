---
title: "Weak-Signal Engineering, Part 14: The Weak-Signal Playbook"
description: "The finale — the whole method folded into one decision tree from symptom to lever: measure in-channel SNR and yield, classify the channel as ISI-limited, reliability-limited, fading-limited, or signal-limited, apply the matching lever, and hold every change to the failing-first, capture-A/B, yield-verdict discipline."
category: deep-dives
keywords: weak signal decision tree, decode yield verdict, equalize or go soft, diversity or fix rf, dsp testing discipline, failing-first regression, capture a/b, marginal signal playbook, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, playbook, dsp, testing, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 14
---

*Part 14 — the finale — of **Weak-Signal Engineering**. Thirteen posts ago we
met a capture that locked and then decoded almost nothing: ~10 dB in-channel
SNR, 22% of its BSCH bursts CRC-clean, a resync storm eating the rest. Along
the way we learned which metrics lie, inverted linear channels blind and
trained, froze taps in front of differential decoders, carried confidence
through the FEC, combined two antennas without breaking anything, and — twice
— proved a deficit wasn't ours to fix. That capture now decodes ~100% of its
BSCH. This closing part compresses the whole method into the thing you
actually need at the bench: a decision tree from symptom to lever, the
testing discipline that makes each lever safe to pull, and an honest list of
what is still open.*

> **TL;DR:** The playbook is four questions asked in order. **Is it
> signal-limited?** — measure in-channel SNR and dBFS headroom; if the
> samples don't contain the call, no DSP recovers it (fix RF, get captures —
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)'s lesson).
> **Is it ISI-limited?** — a smeared constellation at decent SNR is a linear,
> invertible channel: equalize (`SnapshotCMA` blind in the receiver,
> `SnapshotLMS` trained in the extractor). **Is it reliability-limited?** —
> clean-ish symbols failing hard CRC gates want soft decisions
> (`DecodeRCPCTetraMotherSoft` and friends). **Is it fading-limited?** — a
> second branch and coherence-gated MRC (`CrossStats`,
> `TrackingCalibrator`). Every lever lands the same way: a failing-first
> test, a byte-identical opt-out, a capture A/B where one exists, and
> **decode yield as the only verdict**. On the thread capture the CC-path
> equalizer alone took CRC-clean BSCH from ~12% to ~100%; the same
> equalizer nearly doubled voice TCH/S (410→778) across six captures, and
> soft decisions recovered the ~70% of marginal bursts the hard gate
> dropped.

**Key takeaways**

- **Classify before you fix.** The four regimes — signal-, ISI-,
  reliability-, and fading-limited — have distinct fingerprints and
  *disjoint* fixes. The costliest failure mode in this series was always
  applying a real lever to the wrong regime.
- **Yield is the only verdict; everything else is advisory.** EVM collapsed
  34%→8% on an equalizer that decoded zero frames. CRC-valid frames per
  opportunity is the number that cannot flatter you.
- **The discipline stack is what made the levers safe**: failing-first
  regressions, byte-identical opt-outs, capture A/Bs against operators' own
  recordings, and structural guards against the self-consistent-synthetic
  trap.
- **The method transfers.** Nothing in the four questions is
  TETRA-specific — the same triage indicts the P25 C4FM path (Part 13) and
  will grade its fix when the capture arrives.

## Cheat sheet

| Regime | The lever that matches it | Where it lives |
|---|---|---|
| Signal-limited (deficit survives an independent-path control) | fix RF, capture rate, gain staging — not DSP | [Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}), `internal/scanner/ccdecoder/ddc_highrate_test.go` |
| ISI-limited (smeared constellation at decent SNR) | blind `SnapshotCMA` / trained `SnapshotLMS` | `internal/dsp/equalizer/`, [Parts 4–7]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }}) |
| Reliability-limited (symbols mostly right, hard CRCs failing) | soft LLRs through depuncture + Viterbi | `internal/radio/framing/soft_tetra.go`, [Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }}) |
| Fading-limited (yield swings with time / position) | coherence-gated MRC across two branches | `internal/dsp/diversity/`, [Parts 10–11]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }}) |
| Any ("improved" metrics, unchanged yield) | distrust the metric — decode to CRC | [Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }}) |
| Any (green synthetic, live symptom) | operator capture A/B before closing | [Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}), `CONTRIBUTING.md` |

## In this post

- **The four questions** — the decision tree, in order.
- **The levers, ranked by cost** — what each one demands before it pays.
- **The discipline stack** — how a risky lever becomes a safe landing.
- **The thread capture, closed out** — the scorecard.
- **What's still open** — the honest punch list.

## The four questions

Ask them in order — each one is cheaper to answer than the one after it, and
a wrong answer early wastes every step later.

**1. Is it signal-limited?** Measure before theorising: in-channel SNR at
the demod (not wideband carrier SNR — [Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
showed the wideband number grading the *worse* #764 capture higher), peak
dBFS for headroom, and where possible the
[Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
control: route the failing samples through a proven path via an independent
resampler. If the deficit travels with the samples, stop — no equalizer,
soft decoder, or combiner manufactures information the front end never
delivered. The fix is on the other side of the ADC: gain staging, capture
rate, antenna, filtering — the territory of
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}).

**2. Is it ISI-limited?** Decent SNR but a smeared constellation — multipath,
band-edge group delay, simulcast — is a *linear* channel, and linear is
invertible ([Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})).
Equalize: blind `SnapshotCMA` in the streaming receiver where no reference
exists, trained `SnapshotLMS` in the burst extractor where a midamble does.
The confirmation is never the constellation looking rounder — it is yield
moving when the equalizer is enabled on the same capture.

**3. Is it reliability-limited?** Symbols mostly land in the right region
but hard CRC gates keep failing — noise, not smear, is eating the margin.
Go soft: carry LLRs through descramble, deinterleave, depuncture, and the
correlation-metric Viterbi, and let erasures be erasures
([Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})).
This lever stacks with the previous one — equalized symbols feed re-derived
LLRs, not re-sliced bits.

**4. Is it fading-limited?** Yield that swings with time or antenna position
while the average SNR looks adequate means the channel spends part of its
time in a null. One antenna cannot fix a null it is standing in; a second
branch, coherence-gated and MRC-combined
([Parts 10–11]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})),
buys the odds that somewhere on your roof the signal still exists.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The weak-signal decision tree. Start at measure SNR and yield. First branch: deficit survives an independent-path control means signal-limited, go fix RF and capture. Otherwise, smeared constellation at decent SNR means ISI-limited, equalize. Otherwise, hard CRC failures on mostly-correct symbols means reliability-limited, go soft. Otherwise, yield swinging with time means fading-limited, add a diversity branch. Every leaf feeds one verdict box: decode yield on the same capture.">
  <rect x="250" y="8" width="180" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="24" text-anchor="middle" fill="var(--accent)" font-size="10">measure: in-channel SNR,</text>
  <text x="340" y="37" text-anchor="middle" fill="var(--accent)" font-size="10">dBFS headroom, yield</text>
  <line x1="340" y1="44" x2="340" y2="58" stroke="currentColor"/><polygon points="336,56 340,64 344,56" fill="currentColor"/>
  <rect x="20" y="64" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="95" y="82" text-anchor="middle" fill="currentColor" font-size="9">deficit survives the</text>
  <text x="95" y="95" text-anchor="middle" fill="currentColor" font-size="9">independent-path control?</text>
  <rect x="190" y="64" width="140" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="260" y="82" text-anchor="middle" fill="currentColor" font-size="9">constellation smeared</text>
  <text x="260" y="95" text-anchor="middle" fill="currentColor" font-size="9">at decent SNR?</text>
  <rect x="350" y="64" width="140" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="420" y="82" text-anchor="middle" fill="currentColor" font-size="9">symbols right, hard</text>
  <text x="420" y="95" text-anchor="middle" fill="currentColor" font-size="9">CRC gates failing?</text>
  <rect x="510" y="64" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="585" y="82" text-anchor="middle" fill="currentColor" font-size="9">yield swings with</text>
  <text x="585" y="95" text-anchor="middle" fill="currentColor" font-size="9">time / position?</text>
  <line x1="95" y1="108" x2="95" y2="128" stroke="currentColor"/><polygon points="91,126 95,134 99,126" fill="currentColor"/>
  <line x1="260" y1="108" x2="260" y2="128" stroke="currentColor"/><polygon points="256,126 260,134 264,126" fill="currentColor"/>
  <line x1="420" y1="108" x2="420" y2="128" stroke="currentColor"/><polygon points="416,126 420,134 424,126" fill="currentColor"/>
  <line x1="585" y1="108" x2="585" y2="128" stroke="currentColor"/><polygon points="581,126 585,134 589,126" fill="currentColor"/>
  <rect x="20" y="134" width="150" height="42" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="95" y="151" text-anchor="middle" fill="currentColor" font-size="9">SIGNAL-limited</text>
  <text x="95" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fix RF / rate / gain — no DSP</text>
  <rect x="190" y="134" width="140" height="42" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="260" y="151" text-anchor="middle" fill="currentColor" font-size="9">ISI-limited</text>
  <text x="260" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="8">SnapshotCMA / SnapshotLMS</text>
  <rect x="350" y="134" width="140" height="42" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="420" y="151" text-anchor="middle" fill="currentColor" font-size="9">RELIABILITY-limited</text>
  <text x="420" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="8">LLRs → soft Viterbi</text>
  <rect x="510" y="134" width="150" height="42" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="585" y="151" text-anchor="middle" fill="currentColor" font-size="9">FADING-limited</text>
  <text x="585" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="8">coherence-gated MRC</text>
  <line x1="95" y1="176" x2="320" y2="210" stroke="var(--accent)"/>
  <line x1="260" y1="176" x2="330" y2="210" stroke="var(--accent)"/>
  <line x1="420" y1="176" x2="350" y2="210" stroke="var(--accent)"/>
  <line x1="585" y1="176" x2="360" y2="210" stroke="var(--accent)"/>
  <rect x="240" y="210" width="200" height="32" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="230" text-anchor="middle" fill="var(--accent)" font-size="10">verdict: decode yield, same capture</text>
</svg>
<figcaption>Four questions in cost order, four disjoint levers — and every branch terminates at the same verdict box: CRC-valid frames per opportunity on the capture that showed the symptom.</figcaption>
</figure>

## The levers, ranked by what they cost

- **Blind snapshot equalization** (Parts 4–6) costs CPU and two hard-won
  rules: frozen taps in front of anything differential, and a constant
  normalisation reference with a divergence guard. It needs no reference
  symbols and no new data plumbing — the cheapest real lever.
- **Trained equalization** (Part 7) costs the raw-symbol plumbing
  (`SymbolSink`/`StashSymbols`) and works only where a known training
  sequence exists — but it pins the true channel inverse, phase included,
  which blind CMA cannot on a short reference.
- **Soft decisions** (Part 8) cost a parallel float32 stream and a soft
  mirror of the channel-coding chain — paid once per protocol, then every
  marginal burst benefits.
- **Diversity** (Parts 10–11) is the only lever that costs *hardware* — a
  second coherent receive branch — plus the calibration machinery to avoid
  combining into a cancellation. It is also the only lever that helps a flat
  fade.
- The meta-lever is [Part 9]({{ '/blog/deep-dives/weak-signal-engineering-09-parallel-buffers/' | relative_url }})'s
  pattern — parallel buffers with byte-identical opt-outs — which is what
  kept the other four's risk off the working fleet.

## The discipline stack

Every lever above landed through the same gauntlet, and the gauntlet is the
part most worth stealing:

1. **Failing-first.** A regression test that fails against the old code and
   passes with the change — no reproduction, no fix.
2. **Byte-identical opt-out.** Off must be the identity function, and a test
   must compare the bytes (`TestTrafficExtractorSoftUnchangedWithoutEqualizer`
   is the template).
3. **No-harm on clean.** The lever must not tax the signals that already
   decode (`TestSnapshotCMAHarmlessOnCleanSignal`,
   `TestTrafficExtractorLMSNoHarmOnCleanChannel`).
4. **Capture A/B.** Where an operator capture exists, the lever is graded on
   it — `TestDiversityCombinerReplay`'s four arms, the
   `pipelines_tetra_equalizer_test.go` fixture — because a green synthetic
   is not proof of an on-air fix.
5. **Yield verdicts only.** Every harness in this series prints CRC-valid
   counts and says, in as many words, not to conclude anything from EVM.
6. **Guards against self-consistency.** Encode-side behaviour pinned to
   real-air behaviour, cross-checks through independent implementations,
   opcode constants pinned to upstream literals — because a test that
   shares its assumption with the code validates nothing.

## The thread capture, closed out

The capture introduced in
[Part 1]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }})
— a TETRA control channel that locks at ~10 dB in-channel SNR — decoded
~22% of its BSCH in the live session that produced it, with a
destructive-resync storm; the pinned 2-second fixture
(`testdata/tetra_cc_sync_loss_2s_144k.cs16`) baselines at ~12% through the
bare pipeline. Enabling the receiver's `SnapshotCMA` on that one remaining
un-equalized CC path lifted it to **~100% CRC-clean BSCH** — pinned by
`pipelines_tetra_equalizer_test.go`. One lever, matched to its regime,
recovered essentially everything the channel had to give; the residual
−44 dBFS front end remains an RF condition the equalizer mitigates but does
not replace — question 1 never goes away.

The same levers, scored on their own ground: the equalizer roughly doubled
CRC-valid voice TCH/S across the six concurrent-load captures
(soft-decision counts 410→778, with single calls going 4→207); soft
decisions recovered the ~70% of a marginal call's bursts the hard gate
dropped; and on the DMO capture the receiver equalizer lifted CRC-valid
SCH/S from 6 to 64. Different paths, different protocols, one pattern:
classify, lever, yield.

## What's still open

An honest playbook ends with its open items, each parked at a named gate:

- **Trained LMS in production TETRA voice** — staged and synthetic-pinned;
  gated on the `GT_TETRA_LMS=1` capture A/B showing it beats
  CMA-plus-soft on air.
- **Tracking MRC as the diversity default** — differential-safe by measured
  construction; gated on `TestDiversityCombinerReplay` on an operator's own
  pre-combine capture showing the tracking arm winning.
- **P25 Phase 1 C4FM voice** — the odd path out
  ([Part 13]({{ '/blog/deep-dives/weak-signal-engineering-13-p25-c4fm-gap/' | relative_url }}));
  gated on a real weak C4FM voice capture landing in `samples/p25/`.
- **Per-channel diversity combining** — the answer to the wideband-scalar
  caveat; unbuilt, and honestly scoped as a much larger change.

Every one of them is blocked on evidence, not effort — which is the series'
thesis restated: in the marginal regime, the scarce resource is not clever
DSP, it is a measurement you can trust.

## FAQ

**Where do I start when a system decodes badly and I know nothing else?**
Question 1, always: in-channel SNR at the demod and peak dBFS. Five minutes
of measurement sorts most cases into "the samples are fine, work the DSP
levers" or "no DSP will save this" — and the second answer, found early, is
the biggest time-saver in this series.

**How do the levers combine on one channel?**
Equalize first (it operates on symbols), decode soft second (it operates on
the equalized symbols' LLRs), diversify beneath both (it operates on the
wideband stream before any of this). They are complementary regimes, not
alternatives — the TETRA voice path today runs all of: CMA in the receiver,
soft TCH/S in the decoder, and optionally a second branch under MRC.

**What single habit most improves weak-signal debugging?**
Refusing any verdict that isn't decode yield on the capture that showed the
symptom. Every trap this series named — the EVM collapse, the flattering
wideband SNR, the self-consistent synthetic — is a version of accepting a
proxy because it was easier to obtain.

**Is any of this specific to Go or to GopherTrunk?**
The identifiers are; the method isn't. The four regimes, the yield rule, the
frozen-snapshot constraint ahead of differential decoders, the
independent-implementation cross-check — all of it transfers to any SDR
stack. GopherTrunk's contribution is having the tests in the tree so each
claim stays pinned.

**What should I contribute if I want to move an open item?**
Captures, over code. A weak C4FM P25 voice call, a pre-combine diversity
pair from `diversity_capture`, or a marginal TETRA call with
`GT_TETRA_LMS` A/B numbers each unblocks a parked lever — the analysis
machinery for all three is already merged and waiting.

## Series navigation

**Part 14 of 14** · ←
[Part 13: The Odd Path Out — P25 Phase 1 C4FM]({{ '/blog/deep-dives/weak-signal-engineering-13-p25-c4fm-gap/' | relative_url }})
· This is the finale — back to the [series index]({{ '/blog/series/weak-signal-engineering/' | relative_url }}).

*Where to next? The theory you've just finished has a protocol-length case
study — watch every lever land on one real network in
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }}) — or
cross to the other side of the ADC, where question 1 lives, with
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}).*
