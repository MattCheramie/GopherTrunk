---
slug: equalizer-gotchas
title: Equalizer gotchas
entry_type: term
category: fn-diagnostics
description: "Equalizer gotchas are the rules that separate an adaptive equalizer that raises decode yield from one that silently zeroes it: never adapt ahead of a differential decoder, judge by CRC yield not EVM, normalise by a constant, and never test blind CMA on noise-free input."
keywords: equalizer, CMA, LMS, differential decoder, frozen taps, EVM trap, CRC yield, normalisation, divergence guard, rotation, raw symbols, TETRA equalizer
aka: [adaptive equalizer gotchas, CMA gotchas]
infobox:
  - { label: Type, value: DSP facts }
  - { label: Key rule, value: Never feed an adapting equalizer to a differential decoder }
  - { label: Trap, value: "EVM improves while CRC yield stays zero" }
  - { label: Verdict metric, value: CRC-valid frames, nothing else }
see_also: [snapshot-equalization, cma-equalizer, lms-algorithm, differential-decoding, error-vector-magnitude, pi-4-dqpsk, tetra-lock-facts, signal-signatures]
related_reading:
  - { title: "Weak-Signal Engineering, Part 4: Blind CMA", url: /blog/deep-dives/weak-signal-engineering-04-blind-cma/ }
  - { title: "Weak-Signal Engineering, Part 5: The Snapshot Trick — Frozen Taps & Differential Decoders", url: /blog/deep-dives/weak-signal-engineering-05-snapshot-trick/ }
  - { title: "Weak-Signal Engineering, Part 6: Normalisation Guards", url: /blog/deep-dives/weak-signal-engineering-06-normalisation-guards/ }
  - { title: "Weak-Signal Engineering, Part 7: Trained LMS", url: /blog/deep-dives/weak-signal-engineering-07-trained-lms/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/1001
  - https://github.com/MattCheramie/GopherTrunk/issues/1003
---

**Equalizer gotchas** are the rules GopherTrunk learned bringing adaptive equalization into
its TETRA paths — where a [blind CMA](/reference/cma-equalizer/) roughly doubled voice
yield on marginal captures, but only after several variants that looked plausible, measured
well on the wrong metric, and decoded exactly nothing. Each rule below is structural, not a
tuning preference, and each is pinned by a test that fails against the broken variant.

## Never feed a continuously-adapting equalizer to a differential decoder

The CMA cost depends only on output magnitude, so nothing constrains its output phase — as
the taps adapt, the phase wanders. A
[differential decoder](/reference/differential-decoding/) cancels any *constant* rotation
in `s·conj(prev)`, but a phase moving between the two symbols of a pair corrupts every
dibit: streaming-adaptive CMA ahead of TETRA's
[π/4-DQPSK](/reference/pi-4-dqpsk/) differential decode scored **CRC 0**. Permanently
frozen taps only ever reproduce the unequalized baseline. The resolution — adapt a tracking
filter continuously, apply a frozen snapshot per burst — is its own article:
[snapshot equalization](/reference/snapshot-equalization/).

## EVM is a trap; CRC yield is the only trustworthy verdict

Blind CMA minimises *modulus*, not correctness, and its cost surface has spurious
constant-modulus minima. A numerically unstable variant once showed differential
[EVM](/reference/error-vector-magnitude/) collapsing 34% → 8% — a spectacular-looking
win — while CRC-valid frames stayed at **zero**. The rule generalises past equalizers: any
DSP change whose success metric is a statistic of the waveform can be gamed by a transform
that improves the statistic and destroys the information. Decode all the way to CRC and
count valid frames; nothing shallower is evidence.

## Normalise by a constant, not a local average

The CMA update scales with |x|³, so the normaliser is part of the loop. An EMA that tracks
a [TDMA](/reference/tdma/) downlink's slot-to-slot power swings hands the equalizer a
*moving* modulus target, and it converges to garbage — CRC 0 even though a global-RMS
normalise on the same capture gives the full win. Use a cumulative-mean (effectively
constant) normaliser, and pair it with a divergence guard that re-seeds the tracking filter
to pass-through if a transient or deep fade blows the taps up, so one bad patch cannot
poison later snapshots.

## Blind CMA is degenerate on noise-free input

A literally noise-free constant-modulus signal gives the CMA cost nothing to grade — every
input already sits on the circle, and the update direction is meaningless. This is a
*test-design* gotcha: a synthetic clean-channel fixture must add noise (GopherTrunk's add
30–40 dB AWGN) or the test exercises a degenerate case no real signal presents. A perfectly
clean synthetic passing proves nothing about the equalizer; see the
[self-consistent test trap](/reference/self-consistent-test-trap/) for the wider family.

## A frozen filter cannot invert a rotation ramp

TETRA burst decoding tries four carrier rotations; a burst decoded at non-zero residual
rotation has a per-symbol phase *ramp*, and a constant-tap filter cannot represent one. The
DMO trained equalizer therefore equalizes only rotation-0 bursts, where the de-rotation is
a no-op and the trained taps slot straight in. If an equalizer helps on most bursts and
zeroes a minority, check whether the minority share a residual rotation.

## Train in the raw-symbol domain, not the differential domain

The linear channel is a clean convolution only over the *raw* transmitted symbols;
`s·conj(prev)` is a nonlinear product of two channel outputs, and an equalizer trained on
differentials is fitting a model that does not hold. GopherTrunk's traffic extractor
carries raw symbols down parallel to its dibit and soft buffers precisely so the
midamble-trained `SnapshotLMS` can run before the differential decode and re-derive the
[soft LLRs](/reference/log-likelihood-ratio/) from equalized symbols.

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| Equalizer on, CRC drops to 0 | Broken equalizer maths | Adapting taps ahead of a differential decoder | Apply frozen snapshots ([snapshot equalization](/reference/snapshot-equalization/)) |
| EVM improves dramatically, decode unchanged/worse | Big win | Spurious constant-modulus minimum | Score CRC yield; distrust waveform statistics |
| Equalizer converges to garbage only on TDMA signals | Signal-dependent bug | EMA normaliser tracking slot power | Cumulative-mean normalisation + divergence guard |
| Clean-channel unit test passes, real air fails | Coverage gap | Noise-free constant-modulus input is degenerate | Add AWGN to synthetic fixtures |
| Helps most bursts, zeroes some | Flaky | Non-zero residual rotation vs frozen taps | Equalize rotation-0 only, or de-rotate first |

## Provenance

- [#1001](https://github.com/MattCheramie/GopherTrunk/issues/1001) — the TETRA equalizer thread: SnapshotCMA on the receiver paths (voice yield 410→778 across six captures; a marginal control-channel fixture ~12%→~100% BSCH) and the midamble-trained SnapshotLMS follow-up.
- [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) — the DMO path: rotation-0-only trained equalization, and the receiver-CMA lift on direct-mode signalling (CRC-valid SCH/S 6→64 on the first on-air capture).
