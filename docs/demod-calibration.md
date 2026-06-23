---
layout: page
title: P25 Phase 1 demodulator calibration
description: Operator's guide to the P25 Phase 1 demodulator measurement harness — EVM, SNR estimate, pre-FEC error rate, and FSW sync-margin that show whether a weak decode is the demodulator or a marginal signal.
nav_group: Technical docs
---

# P25 Phase 1 demodulator calibration & measurement

This is the operator's guide to the P25 Phase 1 **demodulator measurement
harness** — the instrument that answers, with numbers, whether a weak decode is
the demodulator leaving SNR on the table or the signal simply being marginal.
Before this harness the only signal was "the audio sounds bad" and a binary
locked / not-locked; now every stage reports EVM, an SNR estimate, a pre-FEC
error rate, and an FSW sync-margin, and the demod can be benchmarked against
both the theoretical limit and a second real decoder.

The harness has three legs. Each localizes the gap from a different direction.

## 1. Versus theory — the synthetic sweep

`internal/radio/p25/phase1/receiver/sweep_test.go` drives the real C4FM and
CQPSK receivers across a seeded SNR ladder and measures the recovered
symbol-error rate and SNR estimate at each step, comparing to the closed-form
references in `internal/radio/p25/phase1/metrics` (coherent QPSK / 4-PAM).

- `TestSweepImplementationLossBudget` is a **hard CI gate**: SER must fall
  monotonically with SNR, the Es/N0 needed to reach 1% SER must stay within a
  committed loss budget of theory, and the SNR estimator must track (CQPSK) or
  rise with (C4FM) the injected SNR. The budgets are *regression ceilings*
  calibrated from the measured baseline — re-run the test and read its `loss=`
  log lines to re-derive them.

Run it ad hoc, with a human-readable curve, via the CLI:

```
gophertrunk siglab sweep                       # both paths, default 2–30 dB
gophertrunk siglab sweep -snr-min 6 -snr-max 20 -snr-step 1 -csv sweep.csv
```

Measured baseline at the time of writing: **CQPSK** sits ~3.85 dB off coherent
QPSK at 1% SER (the ~2.3 dB π/4-DQPSK differential penalty plus ~1.5 dB
implementation); **C4FM** sits ~24 dB off coherent 4-PAM — the FM-discriminator
path is sharply noise-fragile and its soft-axis SNR estimate saturates near
20 dB. That C4FM gap is the headroom any future demod work would target.

## 2. Versus the field — real captures

Drop a control-channel capture into `samples/p25/` (see `samples/p25/README.md`
for the `.cfile` + `.metadata.json` schema). The integration-tagged, skip-gated
`TestReplayP25RealCaptureMetrics` runs it through the production receiver and
reports the pre-FEC EVM, estimated SNR, FSW sync-margin distribution, and
NID/TSBK yields — asserting whichever `max_evm_pct` / `min_snr_db` /
`min_sync_margin` bounds the metadata declares.

```
go test -tags integration -run TestReplayP25RealCaptureMetrics -v ./cmd/gophertrunk/
```

The same EVM/SNR appear on every `gophertrunk analyze` / `replay` run (and in
the siglab web Compare tab) via the `demod` block of the result's signal
quality, so you can read them off a live capture without writing a test.

## 3. Versus another decoder — OP25 / DSD-FME cross-check

`TestP25ReferenceCrossCheck` (integration-tagged, doubly skip-gated) runs an
external reference decoder on the *same* capture and diffs the decoded NAC sets
and frame yields, modelled on `internal/voice/calibrate`'s pass-bar. Combined
with leg 1 it localizes a gap: if GopherTrunk trails both theory and the
reference, the gap is GopherTrunk-specific; if GopherTrunk ≈ the reference and
both trail theory, the signal is inherently marginal.

Configure via environment and run:

```
export P25_REFERENCE_CMD='dsd-fme -i {} -fp -o /dev/null'   # {} = capture path
export P25_REFERENCE_DEMOD=c4fm                              # or cqpsk
export P25_REFERENCE_MIN_AGREE=1.0                           # NAC-set overlap bar
go test -tags integration -run TestP25ReferenceCrossCheck -v ./cmd/gophertrunk/
```

The comparison math (`internal/radio/p25/phase1/refcompare`) is pure and
unit-tested; only the binary invocation and output scraping live in the
skip-gated test, so the tool/version-specific brittleness never reaches CI.

## What this harness is *not*

It does not re-architect the demodulator. It is the prerequisite that tells you
whether — and precisely where — that is warranted. The large C4FM-vs-theory gap
in leg 1 is the evidence such work would start from; the real-capture and
reference legs say whether closing it would actually help a given site.
