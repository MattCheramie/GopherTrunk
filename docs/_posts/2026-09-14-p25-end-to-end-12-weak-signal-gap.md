---
title: "P25 End to End, Part 12: The Weak-Signal Gap — P1 Voice's Missing Levers"
description: Why P25 Phase 1 C4FM voice is GopherTrunk's least-equipped decode path — an FM-discriminator chain with no channel equalizer and hard-decision IMBE FEC, a measured ~24 dB gap to the coherent reference, and the weak-signal voice capture that gates the fix.
category: deep-dives
keywords: p25 weak signal decode, c4fm equalizer missing, p25 phase 1 voice decode, imbe hard decision fec, fm discriminator weak signal, p25 voice capture contribution, soft decision p25 tsbk, p25 marginal call frames, gophertrunk p25
tags: [p25-end-to-end, p25, c4fm, weak-signal, imbe, equalizer, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 12
---

*Part 12 of **P25 End to End**, a 14-part deep dive that follows North America's
dominant trunking protocol through GopherTrunk — from a raw C4FM carrier to
recorded, named, multi-site voice.
[Part 11]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }})
watched a whole system through one wide capture and met the twin pair that let
a fix land on one side and miss the other. This part is the one the opener
promised: the honest inventory. On a marginal Phase 1 voice call, a hardware
radio decodes clean audio while GopherTrunk recovers a handful of frames — and
we know exactly which two stages are missing, why they haven't been built, and
what single recording unblocks them.*

> **TL;DR:** The default P25 Phase 1 **C4FM voice receiver**
> (`internal/radio/p25/phase1/receiver/receiver.go`) is FM discriminator → spec
> matched filter → `CoarseAFC` → Mueller-Müller timing → symbol AGC → 4-level
> slicer → **hard** Golay/Hamming IMBE FEC — no channel equalizer, no
> soft-decision voice FEC. The synthetic sweep
> (`TestSweepImplementationLossBudget`) puts a number on the fragility: at 1%
> symbol-error rate the C4FM path needs **~23.8 dB more Es/N0 than the
> coherent 4-PAM reference**, while the CQPSK twin — which carries a T/2 blind
> equalizer (the `fse` field in `cqpsk.go`) — sits ~3.85 dB off coherent QPSK.
> The missing levers are the pair that roughly
> [doubled TETRA's marginal yield]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }});
> the port is gated on a real weak C4FM **voice** capture, and
> `samples/p25/README.md` + `TestReplayP25RealCaptureMetrics` are the baseline
> instrument waiting for it.

**Key takeaways**

- **The gap is structural, and it is measured.** Every stage of the C4FM
  voice chain works as designed; it simply lacks the equalization and
  soft-FEC stages the marginal regime demands, and the BER sweep quantifies
  the headroom against a reference the CQPSK path tracks within 4 dB.
- **This is the twin-path thread inverted.** Usually a fix lands on one twin
  and misses the other; here a *capability* — the blind equalizer — was only
  ever built on the CQPSK/LSM twin and on Phase 2.
- **The soft machinery is half-installed.** The C4FM receiver already emits
  two log-likelihood ratios per dibit (`BitLLRSink`) and the control channel
  spends them on soft TSBK decode — but the IMBE Golay/Hamming decoders never
  see a soft bit.
- **No fix without a capture.** A green synthetic is not proof of an on-air
  fix — the #764/#771 lesson. The highest-leverage contribution to
  GopherTrunk's P25 support right now is a recording, not a patch.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| C4FM voice chain (all hard) | discriminator → MF → AFC → M&M → AGC → slicer | `internal/radio/p25/phase1/receiver/receiver.go` (`DemodC4FM` branch) |
| The equalizer next door | T/2 fractionally-spaced blind CMA on the linear path | `internal/radio/p25/phase1/receiver/cqpsk.go` (`fse *equalizer.FSE`) |
| Soft plumbing, half-installed | per-dibit LLRs feed soft TSBKs, not IMBE | `receiver.go` (`BitLLRSink`) → `control.go` (`StashSoft`) |
| Measured fragility | C4FM ~23.8 dB off coherent 4-PAM at 1% SER | `receiver/sweep_test.go` (`TestSweepImplementationLossBudget`) |
| Baseline instrument | EVM / SNR / FSW-margin / NID / TSBK from a real capture | `cmd/gophertrunk/p25_realcapture_metrics_test.go` |
| The ask | a weak C4FM voice `.cfile` + metadata sidecar | `samples/p25/README.md` (weak-signal voice section) |

## In this post

- **The report** — a hardware Astro Spectra sets the bound.
- **The chain, audited** — what the C4FM voice path runs and what it discards.
- **The twin that got the equalizer** — `fse`, Phase 2, and the inverted twin thread.
- **The gap, measured** — what the BER sweep's loss budgets actually say.
- **Why the port is parked** — proven levers, missing evidence.
- **The instrument is waiting** — what to record and what happens next.

## The report: a hardware radio sets the bound

The report that defines this part is a controlled experiment an operator ran
without meaning to. A marginal P25 Phase 1 voice call decodes cleanly on a
hardware **Astro Spectra**; the same call, on the same antenna, through
GopherTrunk yields about **4–5 IMBE frames** from a whole transmission. Not
silence, and not a failure to lock — a lock that extracts almost nothing, the
signature of the
[marginal regime]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }})
in its purest form.

The hardware radio matters because it bounds the problem: the information
survived the antenna, so whatever is lost is lost after the ADC. There is a
telling brush in the numbers, too — an LDU carries 9 IMBE frames in 180 ms,
so 4–5 frames is under one LDU, well short of the `minAutotuneLDUs = 5`
threshold in `internal/voice/composer/p25p1_voice.go` that skips the autotune
measurement for calls too short or noisy to trust. The guard is correct; that
real calls keep tripping it is the indictment.

Commercial subscriber radios earn their weak-signal margin with two tools:
adaptive equalization and soft-decision error correction. Which does the
default C4FM voice path run?

## The chain, audited

The receiver's own doc comment is the inventory:

```go
// internal/radio/p25/phase1/receiver/receiver.go (shape) — the C4FM branch
//	DemodC4FM (default):
//	  IQ
//	    → FM discriminator (internal/dsp/demod.FM)
//	    → spec P25 C4FM matched filter (demod.P25C4FMRxTaps — not RRC)
//	    → coarse AFC: residual carrier-offset removal (demod.CoarseAFC)
//	    → Mueller-Müller symbol clock recovery (sync.MuellerMuller)
//	    → 4-level slicer → C4FM symbol → 0..3 dibit
```

Downstream, the LDU voice path
([Part 8]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }}))
runs `ExtractVoiceFrames`: hard
[Golay]({{ '/reference/golay-code/' | relative_url }}) and
[Hamming]({{ '/reference/hamming-code/' | relative_url }}) decoding per IMBE
subframe. Audited against the marginal regime:

- **No equalizer, anywhere on the C4FM path.** No stage between timing
  recovery and the slicer inverts inter-symbol interference; a
  multipath-smeared 4-level eye goes into symbol decisions smeared.
- **A hard slicer feeding hard FEC.** The slicer collapses each symbol to a
  dibit and throws away its distance from the thresholds — the confidence a
  soft FEC would spend. Golay/Hamming then defend a coin-flip bit as firmly
  as a certain one.
- **The soft machinery exists — for the control channel.** The C4FM receiver
  *already* derives two log-likelihood ratios per dibit (`Options.BitLLRSink`
  — sign-axis distance for the high bit, inner/outer-threshold distance for
  the low), and the control channel's `StashSoft` path spends them on a
  per-bit soft TSBK Viterbi behind the `p25_phase1_soft_decision` config key.
  The plumbing runs right past the voice path: IMBE's Golay and Hamming
  decoders take hard bits, full stop.

Nothing in that list is a bug. It is a fair-weather design meeting a use
case that is not fair weather.

## The twin that got the equalizer

This series' running thread is that P25 is a family of twins, and every twin
pair is a place fixes drift apart. The weak-signal gap is that thread
**inverted**: not a fix that missed one twin, but a capability that only ever
landed on one.

[Part 6]({{ '/blog/deep-dives/p25-end-to-end-06-cqpsk-lsm/' | relative_url }})
covered the linear path in full: the opt-in `DemodCQPSK` receiver carries a
**T/2 fractionally-spaced blind equalizer**, and its configuration is worth
re-reading with this part's eyes:

```go
// internal/radio/p25/phase1/receiver/cqpsk.go (shape)
const (
    cqpskFSESymbolSpan = 6     // 12 T/2 taps, centre tap on-time
    cqpskFSEStep       = 0.025 // brisk CMA step — short captures must converge
    cqpskFSELeak       = 5e-4  // pulls taps toward identity on clean input
)
/* … */
fse: equalizer.NewFSE(cqpskFSESymbolSpan, cqpskFSEStep, 1.0, cqpskFSELeak),
```

The comments above those constants record why fractional spacing matters: a
T/2 equalizer synthesizes the receive matched filter implicitly, opening both
simulcast multipath ISI *and* pulse-shape mismatch — things a symbol-spaced
equalizer cannot (issue #492,
[told in four acts]({{ '/blog/solution-postmortem/from-the-issue-tracker-06-cqpsk-four-acts/' | relative_url }})).
P25 Phase 2 wires an equalizer as well
([Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }})).
Default C4FM Phase 1 voice is, literally, the odd path out.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two decode chains compared. Top row, the P25 Phase 1 C4FM voice chain: FM discriminator, matched filter, CoarseAFC, Mueller-Muller timing with AGC, hard 4-level slicer, hard Golay and Hamming IMBE FEC — with two dashed boxes marking the missing equalizer and missing soft FEC stages. Bottom row, the TETRA voice chain for contrast: RRC matched filter, Gardner timing, snapshot CMA equalizer, differential decode, soft LLRs, and soft Viterbi with CRC — every stage solid. A note says the dashed boxes are the two levers that roughly doubled TETRA's marginal yield.">
  <text x="10" y="24" fill="currentColor" font-size="11" font-weight="bold">P25 Phase 1 C4FM voice (default)</text>
  <rect x="10" y="36" width="86" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="53" y="53" text-anchor="middle" fill="currentColor" font-size="9">FM</text>
  <text x="53" y="66" text-anchor="middle" fill="currentColor" font-size="9">discriminator</text>
  <line x1="96" y1="55" x2="108" y2="55" stroke="currentColor"/>
  <rect x="108" y="36" width="80" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="148" y="53" text-anchor="middle" fill="currentColor" font-size="9">matched filter</text>
  <text x="148" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+ CoarseAFC</text>
  <line x1="188" y1="55" x2="200" y2="55" stroke="currentColor"/>
  <rect x="200" y="36" width="88" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="244" y="53" text-anchor="middle" fill="currentColor" font-size="9">Mueller-Müller</text>
  <text x="244" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">timing + AGC</text>
  <rect x="300" y="30" width="96" height="24" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="5 4"/>
  <text x="348" y="46" text-anchor="middle" fill="var(--accent)" font-size="8">equalizer — MISSING</text>
  <line x1="288" y1="55" x2="420" y2="55" stroke="currentColor"/>
  <rect x="420" y="36" width="80" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="460" y="53" text-anchor="middle" fill="currentColor" font-size="9">hard 4-level</text>
  <text x="460" y="66" text-anchor="middle" fill="currentColor" font-size="9">slicer</text>
  <line x1="500" y1="55" x2="512" y2="55" stroke="currentColor"/>
  <rect x="512" y="36" width="100" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="562" y="53" text-anchor="middle" fill="currentColor" font-size="9">hard Golay/</text>
  <text x="562" y="66" text-anchor="middle" fill="currentColor" font-size="9">Hamming IMBE</text>
  <rect x="560" y="86" width="110" height="24" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="5 4"/>
  <text x="615" y="102" text-anchor="middle" fill="var(--accent)" font-size="8">soft FEC — MISSING</text>
  <line x1="588" y1="86" x2="575" y2="74" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="10" y="140" fill="currentColor" font-size="11" font-weight="bold">TETRA voice (for contrast — every stage built)</text>
  <rect x="10" y="152" width="130" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="75" y="169" text-anchor="middle" fill="currentColor" font-size="9">RRC matched filter</text>
  <text x="75" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Gardner timing + AFC</text>
  <line x1="140" y1="171" x2="154" y2="171" stroke="currentColor"/>
  <rect x="154" y="152" width="110" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="209" y="169" text-anchor="middle" fill="var(--accent)" font-size="9">SnapshotCMA</text>
  <text x="209" y="182" text-anchor="middle" fill="var(--accent)" font-size="8">equalizer</text>
  <line x1="264" y1="171" x2="278" y2="171" stroke="currentColor"/>
  <rect x="278" y="152" width="110" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="333" y="169" text-anchor="middle" fill="currentColor" font-size="9">differential decode</text>
  <line x1="388" y1="171" x2="402" y2="171" stroke="currentColor"/>
  <rect x="402" y="152" width="194" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="499" y="169" text-anchor="middle" fill="var(--accent)" font-size="9">soft LLRs → soft Viterbi → CRC</text>
  <text x="340" y="226" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the two dashed boxes are the levers that roughly doubled TETRA's marginal yield —</text>
  <text x="340" y="240" text-anchor="middle" fill="var(--fg-muted)" font-size="10">proven in this codebase, absent from this path</text>
</svg>
<figcaption>Side by side: the C4FM voice chain versus the TETRA voice chain. Every solid block on the P25 row works; the two dashed blocks are the diagnosis — and both already exist elsewhere in the tree.</figcaption>
</figure>

## The gap, measured

"Noise-fragile" is not a vibe here; it is a committed CI number. The
synthetic BER-vs-SNR sweep in `receiver/sweep_test.go` drives the **real
receiver** on both demod paths across an injected-SNR ladder and gates the
result against closed-form references:

```go
// internal/radio/p25/phase1/receiver/sweep_test.go (shape) — sweepGates
// CQPSK lands ~3.85 dB off coherent QPSK at 1% SER;
// C4FM ~23.8 dB off coherent 4-PAM. Budgets add ~2–3 dB of margin.
{mode: DemodCQPSK, ref: metrics.SERCoherentQPSK, targetSER: 1e-2, lossBudgetDB: 6.0},
{mode: DemodC4FM,  ref: metrics.SER4PAM,         targetSER: 1e-2, lossBudgetDB: 27.0},
```

Read the two numbers together. The linear path — coherent demod, equalizer,
Gardner timing — reaches 1% symbol errors within ~4 dB of the coherent QPSK
bound. The C4FM discriminator path needs ~24 dB more Es/N0 than coherent
4-PAM. The honest caveat is in the test's own comment: coherent 4-PAM is a
bound no FM discriminator reaches, and the budget "is a *regression ceiling*
… NOT a claim that the gap is small" — part physics of non-coherent
detection, part timing-slip behaviour the measurement folds in. But the
ceiling stops the path silently getting worse, and it quantifies the headroom
that justifies demod work. Twenty-plus dB of headroom is why a subscriber
radio and GopherTrunk disagree so completely about the same marginal call.

## Why the port is parked

Both missing stages are proven technology *in this codebase*. The blind
snapshot equalizer and soft-decision decoding
([Weak-Signal Engineering Parts 4]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
[and 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }}))
roughly **doubled** TETRA's CRC-valid voice yield across six operator
captures (410 → 778 soft-decision TCH/S, one call going 4 → 207), and the
soft path recovered the ~70% of a marginal call's bursts the hard gate
dropped. The physics doesn't care which trunking protocol rides on top.

So why not port it this afternoon? Because the project has been burned,
twice, by exactly that move: issue
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) was closed
twice on fixes that were green against synthetic fixtures while the live
symptom persisted (issue #771 is the receipt). A synthetic channel exercises
the impairments you *imagined*; the operator's terrain exercises the ones you
didn't. The standing gate that came out of it: a weak-signal fix lands only
with a baseline on a real capture and an A/B where **decode yield is the
verdict** — never EVM, which
[has lied to this project before]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }}).
The sibling series covers this path from the method's side
([The Odd Path Out]({{ '/blog/deep-dives/weak-signal-engineering-13-p25-c4fm-gap/' | relative_url }}));
this series adds the protocol context: it is the last unequipped twin.

## The instrument is waiting

Everything buildable without the capture is built.
`TestReplayP25RealCaptureMetrics`
(`cmd/gophertrunk/p25_realcapture_metrics_test.go`, tag `integration`)
replays any `.cfile` dropped into `samples/p25/` through the production
receiver and reports pre-FEC EVM, estimated SNR, the FSW sync-margin
distribution, and NID/TSBK yields — every metadata bound optional, so a
capture can be measured before anyone commits to a pass/fail threshold. The
pipeline is validated end to end: a live UHF C4FM control-channel capture
(449.875 MHz, NAC 0x2C1) grades at EVM ≈ 12.7%, SNR ≈ 14.5 dB, NID trusted
31 / failed 0, TSBK decoded 36 / CRC-failed 0.

What's missing is the **voice** capture. `samples/p25/README.md` spells out
the ask: tune the *granted voice frequency* during a call that is genuinely
weak or multipath-degraded — a strong, clean call cannot exercise a missing
equalizer — note the hardware radio's contemporaneous decode in
`tool_cross_check`, and drop the `.cfile` + `.metadata.json` pair in (the
binary stays git-ignored; the sidecar is committed). From there the sequence
is mechanical: baseline, port one lever — the capture's own numbers decide
whether ISI says equalizer-first or a clean eye with FEC failures says
soft-first — and A/B the LDU/IMBE yield.

## Where this goes next

A series that leans this hard on "the test that passes because both sides
share the same bug" owes you the full testing story.
[Part 13]({{ '/blog/deep-dives/p25-end-to-end-13-testing-p25/' | relative_url }})
climbs the whole pyramid — literal byte vectors, impairment-swept synthetic
streams, committed captures, on-air verification — using P25's own bugs as
the case studies.

## FAQ

**Why can a hardware scanner decode a P25 call GopherTrunk can't?**
Because commercial subscriber hardware ships the two stages this path lacks:
adaptive equalization ahead of the symbol decisions and soft-decision error
correction behind them. On clean signals the difference is invisible; in the
marginal regime those stages are worth many dB of effective sensitivity.

**Is the P25 control channel affected the same way?**
Less so. The control channel can opt into per-bit soft TSBK decode
(`p25_phase1_soft_decision`), fed by the same `BitLLRSink` LLRs the voice
path ignores — and TSBKs repeat, so a missed block usually costs nothing.
IMBE frames don't repeat: every hard-FEC failure is 20 ms of audio gone,
which is why the operator-visible gap is voice.

**Couldn't I just run the CQPSK receiver on the weak C4FM site?**
No. The equalizer lives on a *linear* demod chain that expects an LSM-style
phase trajectory, and the two paths are not interchangeable (the
[LSM myth postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})
covers why; issue #935 covers when each applies). The equalizer has to come
to the C4FM chain, not the signal to the equalizer.

**What exactly should I record to help?**
A P25 Phase 1 C4FM **voice-channel** IQ recording of a marginal call — the
granted frequency, not the control channel — as a `.cfile` at ≥ 48 kHz with a
metadata sidecar per `samples/p25/README.md`, ideally noting what a hardware
radio made of the same call. That one file converts a parked fix into a
measurable A/B.

**Does the 23.8 dB figure mean the C4FM demod is broken?**
No — the reference is coherent and the demod is not, plus real
implementation loss on top. The number's job is to be a ceiling (the gate
fails past 27 dB) and a motivator: the measured size of the pool the
equalizer and soft FEC would fish in.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Wideband P25 — Watching the Whole System at Once]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }})
· Next →
[Part 13: Testing P25 Without a Tower]({{ '/blog/deep-dives/p25-end-to-end-13-testing-p25/' | relative_url }})
