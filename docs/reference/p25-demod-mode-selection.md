---
slug: p25-demod-mode-selection
title: P25 demod-mode selection
entry_type: term
category: fn-config
description: "P25 demod-mode selection is the choice between GopherTrunk's C4FM and CQPSK Phase 1 demodulation paths, made empirically — never from 'the site is simulcast', which is a transmitter-coordination fact that does not imply LSM modulation."
keywords: p25_phase1_demod_mode, c4fm, cqpsk, lsm, linear simulcast modulation, simulcast, demod mode, emission designator, p25 phase 1
aka: [p25_phase1_demod_mode, C4FM vs CQPSK, LSM mode selection]
infobox:
  - { label: Type, value: Config decision }
  - { label: Choices, value: c4fm (default) / cqpsk }
  - { label: Key rule, value: "Choose cqpsk only empirically — a strong, clean signal that refuses to lock in C4FM" }
  - { label: Trap, value: Simulcast does not imply LSM/CQPSK }
see_also: [c4fm, cqpsk, simulcast, p25-phase-1, wideband-voice-taps, control-channel, sdr-gain-overload]
related_reading:
  - { title: "From the Issue Tracker, Part 7: The LSM Myth — When Your Own Docs Are the Bug", url: /blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/ }
  - { title: "From the Issue Tracker, Part 6: CQPSK in Four Acts — Fixing the Linear Path One Layer at a Time", url: /blog/solution-postmortem/from-the-issue-tracker-06-cqpsk-four-acts/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/935
  - https://github.com/MattCheramie/GopherTrunk/issues/492
  - https://github.com/MattCheramie/GopherTrunk/issues/297
---

**P25 demod-mode selection** is the choice between GopherTrunk's two P25 Phase 1
symbol-recovery paths — [C4FM](/reference/c4fm/) (FM-discriminator, the default) and
[CQPSK](/reference/cqpsk/) (linear, for sites transmitting Linear Simulcast Modulation) —
set per system with `p25_phase1_demod_mode`, or per channel on a
[wideband](/reference/wideband-voice-taps/) dongle when sites of one system differ. The
setting matters because the wrong path does not merely degrade the decode; a C4FM
discriminator cannot recover a linearly-modulated signal and vice versa, so the control
channel simply never locks or reads as `poor`.

## The LSM myth

The single most damaging misconception — one GopherTrunk's own docs, config comments, and
web config-builder labels taught for a while — is that "simulcast site" implies "use
CQPSK/LSM." It does not. [Simulcast](/reference/simulcast/) is a *transmitter coordination*
technique (timing and phase alignment across towers), not a baseband modulation. Many
genuinely multi-tower simulcast sites transmit ordinary C4FM and decode reliably on the
C4FM path; forcing CQPSK on them kills the decode entirely
([#935](https://github.com/MattCheramie/GopherTrunk/issues/935)). LSM is one specific way
of running a simulcast cell, not a synonym for it.

Licensing databases cannot settle the question either: an emission designator such as
`10K1D7W` covers both C4FM and CQPSK, so no amount of reading the license tells you which
demod path a site needs ([#935](https://github.com/MattCheramie/GopherTrunk/issues/935)).

## The rule

Choose `cqpsk` **empirically, and only empirically**: a signal that is demonstrably strong
and clean (verify with `spectrum` or another receiver) yet refuses to lock on the C4FM path
is the one legitimate trigger. Never infer it from "the site is simulcast," from the
license, or from third-party site labels. Start with `c4fm`, and reach for `cqpsk` only
when the evidence forces you to.

| Symptom | Looks like | Actually | Fix / check |
| --- | --- | --- | --- |
| Site listed as simulcast | "Must need CQPSK" | Simulcast is transmitter coordination, not a modulation; most such sites decode in C4FM | Try `c4fm` first — it is the default for a reason |
| Strong, clean carrier; no C4FM lock | Weak signal or wrong frequency | Possibly a genuine LSM/linear site | Set `p25_phase1_demod_mode: cqpsk` and compare |
| No lock in either mode | Modulation mystery | Often gain: the value is in **tenths of a dB**, so a working ~36 dB from another app is `gain: "363"`, not `"36"` | Check gain units and front-end level — see [SDR gain and overload](/reference/sdr-gain-overload/) |
| CQPSK site: CC locks but everything downstream is garbage | FEC or signal problem | Historically, an odd-rotation recovery bug that *only* the CQPSK path could trigger (see below) | Fixed; on current builds suspect signal or configuration |

In the originating report the site that "needed LSM" was Melbourne's genuinely three-tower
simulcast CBD cell — which decodes reliably in C4FM in both GopherTrunk and SDRTrunk. The
actual blocker was the gain value; once translated correctly into tenths, the C4FM path
locked and granted normally ([#935](https://github.com/MattCheramie/GopherTrunk/issues/935)).

## Why the two paths really differ

The paths are not interchangeable front ends onto one decoder; they have different failure
modes, which is why the choice must be explicit:

- The CQPSK path is a linear receiver and needs carrier recovery. Early builds had none —
  a differential decoder cancels constant carrier phase but not per-symbol rotation, and at
  P25's 4800 baud even a small offset rotates symbols badly. Getting CQPSK to lock on real
  simulcast captures took carrier recovery, a multipath-gated frequency estimator, and a
  fractionally-spaced equalizer ([#492](https://github.com/MattCheramie/GopherTrunk/issues/492)).
- The C4FM discriminator path can only produce *even* constellation rotations, which
  coincidentally masked a sign error in rotation recovery for years: odd rotations —
  exactly the quadrant slips a CQPSK differential demod produces — came back with every
  dibit flipped, a failure that could only ever appear on simulcast/CQPSK sites
  ([#297](https://github.com/MattCheramie/GopherTrunk/issues/297)).

The practical consequence: "CQPSK doesn't work on this site" historically had several
causes that were bugs, not site properties. On current builds the empirical rule above is
the whole procedure.

## Per-channel overrides

Where sites of one system genuinely differ in modulation, set the system's
`p25_phase1_demod_mode` to what most sites use and add a per-channel
`p25_phase1_demod_mode:` override on the exceptions in the wideband `channels:` list. The
override drives both the control-channel decode and the voice grants that tap issues. It is
keyed by frequency, not site identity, because the demod path must be chosen before the
control channel can lock — see [wideband voice taps](/reference/wideband-voice-taps/) for
the details.

## Provenance

- [#935](https://github.com/MattCheramie/GopherTrunk/issues/935) — the LSM ≠ CQPSK correction, the emission-designator dead end, the gain-in-tenths root cause, and the per-channel override.
- [#492](https://github.com/MattCheramie/GopherTrunk/issues/492) — what it took to make the CQPSK path lock on real simulcast captures.
- [#297](https://github.com/MattCheramie/GopherTrunk/issues/297) — the odd-rotation recovery bug that only CQPSK quadrant slips could expose.
