---
slug: diagnostic-playbook
title: Diagnostic playbook
entry_type: term
category: fn-diagnostics
description: "The diagnostic playbook is GopherTrunk's escalation ladder for decode failures: verify the build, probe the hardware, capture ground truth, replay offline, census the decoder, and only then suspect the OS or the DSP."
keywords: diagnostics, troubleshooting, escalation ladder, replay, capture, spectrum, census logging, usb trace, sdr probe, ground truth, offline decode
aka: [escalation ladder, decode-failure triage]
infobox:
  - { label: Type, value: Troubleshooting procedure }
  - { label: Rungs, value: "9 (0–8), from build check to independent cross-check" }
  - { label: Principle, value: Each rung replaces a guess with a measurement }
  - { label: First step, value: Confirm the build before interpreting any result }
see_also: [signal-signatures, audio-pipeline-tells, carrier-offset-adjacent-lock, rtlsdr-usb-recovery, sdr-gain-overload, airspy-rate-selection, iq-recording-playback, error-vector-magnitude]
related_reading:
  - { title: "From the Issue Tracker, Part 21: Census Everything — The Silence of a Success-Only Log Line Carries No Information", url: /blog/solution-postmortem/from-the-issue-tracker-21-census-everything/ }
  - { title: "From the Issue Tracker, Part 1: The First P25 Lock — Eleven Fixes Between 'Trying' and 'Locked'", url: /blog/solution-postmortem/from-the-issue-tracker-01-first-p25-lock/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/275
  - https://github.com/MattCheramie/GopherTrunk/issues/376
  - https://github.com/MattCheramie/GopherTrunk/issues/813
  - https://github.com/MattCheramie/GopherTrunk/issues/248
  - https://github.com/MattCheramie/GopherTrunk/issues/764
  - https://github.com/MattCheramie/GopherTrunk/issues/345
  - https://github.com/MattCheramie/GopherTrunk/issues/598
---

The **diagnostic playbook** is the escalation ladder that GopherTrunk's hardest
issues converged on. Its organizing principle: **each rung replaces a guess with a
measurement**, and the cheap rungs come first. The pattern behind almost every
long investigation in the tracker — the first P25 lock
([#275](https://github.com/MattCheramie/GopherTrunk/issues/275)), the talker-alias
hunt ([#376](https://github.com/MattCheramie/GopherTrunk/issues/376)), the Phase 2
encryption fields ([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)) —
is that time was lost whenever a theory ran ahead of the evidence. The ladder
exists so the evidence arrives first.

| Rung | Tool | Question it answers |
|---|---|---|
| 0 | Build check | Am I even testing the fix? |
| 1 | `gophertrunk sdr list --probe` | Can the device be opened and its tuner initialized? |
| 2 | `gophertrunk sdr doctor` | Is the device bound to a usable driver? |
| 3 | `gophertrunk capture` + `gophertrunk spectrum` | What is actually on the air, off-pipeline? |
| 4 | `gophertrunk replay -diag` | Does the decoder work on this exact recording? |
| 5 | Unconditional census logging | Which decode stage is failing — and how often? |
| 6 | Direct API probes (`curl`) | Is data flowing between components? |
| 7 | OS-level evidence (`dmesg`, paired USB traces) | Is the hardware or transport misbehaving? |
| 8 | Independent-implementation cross-check | Is the fault in GopherTrunk at all? |

## Rung 0: confirm the build

Two retest cycles in [#275](https://github.com/MattCheramie/GopherTrunk/issues/275)
and two more in [#376](https://github.com/MattCheramie/GopherTrunk/issues/376) were
wasted on stale binaries — a fix that was never actually running. The pipeline
startup log line now advertises its `build=` identifier for exactly this reason.
Before interpreting any result, confirm the version string matches the code you
think you are testing.

## Rungs 1–2: probe the hardware before the theory

`sdr list --probe` opens each discovered device long enough to run demodulator and
tuner initialization, so transport faults surface as one explicit error line
instead of a silent downstream failure. `sdr doctor` checks that each known device
is bound to a driver GopherTrunk can use. If either rung fails, the problem is in
the USB/driver layer — see [RTL-SDR USB recovery](/reference/rtlsdr-usb-recovery/) —
and no amount of DSP investigation will help.

## Rung 3: capture ground truth off-pipeline

`capture` records raw IQ and `spectrum` renders it **without involving the decode
pipeline**, so what they show is the air, not the software. In
[#275](https://github.com/MattCheramie/GopherTrunk/issues/275) the maintainer
called fifteen seconds of real ground-truth IQ "the linchpin" — every one of that
issue's eleven bugs had been masked by synthetic round-trip tests where encoder
and decoder agreed with each other. Off-pipeline spectrum also settles "is the
carrier even there?" questions instantly: in
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) it proved all four
carriers were present and healthy-looking while the decoder saw nothing, and in
the adjacent-lock case it showed no carrier at all at the configured frequency
(see [carrier offset & adjacent-channel lock](/reference/carrier-offset-adjacent-lock/)).

## Rung 4: replay the capture offline

`gophertrunk replay -diag` decodes a recorded IQ file through the real decode
chain and prints a demod-quality report at EOF — symbol histogram, sync landscape,
soft-sample eye, and effective baud rate. The effective-baud figure alone caught a
mislabeled capture in [#275](https://github.com/MattCheramie/GopherTrunk/issues/275)
(~160 samples per symbol meant the file was not the sample rate its name claimed).
Replay turns a slow on-air guess-and-retest loop into a fast, repeatable
experiment: the same bytes in, a comparable
[EVM](/reference/error-vector-magnitude/)/[SNR](/reference/signal-to-noise-ratio/)
report out. See [IQ recording & playback](/reference/iq-recording-playback/). One
caveat: replay and the live daemon are not identical code paths, so a warning or
fix present in one may not exist in the other — verify in the path where the
symptom lives.

## Rung 5: census, don't infer from silence

The silence of a success-only log line carries no diagnostic information. In
[#813](https://github.com/MattCheramie/GopherTrunk/issues/813) the line
`composer: p25p2 mac pdu` only fired on a *successful* MAC decode, so zero lines
looked identical whether superframe sync never locked, ISCH never classified, or
the MAC FEC failed. The fix was an **unconditional per-call census** —
`superframes=N voice_subframes=N mac_subframes=N mac_pdus=N` plus a slot
histogram, logged once per call *even at zero* — which immediately disambiguated
the three cases (`superframes=0` on 67 of 67 calls: the failure was upstream of
MAC entirely). The same technique cracked
[#376](https://github.com/MattCheramie/GopherTrunk/issues/376): an info-level
per-(opcode, MFID) census with capped raw-payload hex proved the talker alias was
*not* on the control channel and decoded two previously dropped grant types along
the way. When a stage is suspect, make it count everything it sees, not just what
it accepts.

## Rung 6: probe the APIs directly

When components disagree about whether data is flowing, ask the wire. The two
`curl` probes from [#598](https://github.com/MattCheramie/GopherTrunk/issues/598)
— stream a few kilobytes of `/api/v1/audio/stream` and read the publisher
counters from `/api/v1/audio` — located a missing-PCM bug in minutes after two
plausible browser-side theories had failed. The specific probes are in
[audio-pipeline tells](/reference/audio-pipeline-tells/).

## Rung 7: ask the operating system

A decoder that goes silent with no error may not be your bug. In
[#345](https://github.com/MattCheramie/GopherTrunk/issues/345) the "pipeline
stall" was the dongle repeatedly dropping off the USB bus — proven by matching
`dmesg` disconnect/re-enumeration timestamps against the last decoded log line.
For faults inside the USB transport itself, the technique from
[#248](https://github.com/MattCheramie/GopherTrunk/issues/248) is **paired
traces**: run GopherTrunk with `RTLSDR_DEBUG_USB=1` and a known-good reference
(`rtl_test` with `LIBUSB_DEBUG=4`) in the same session, then diff the transfer
sequences. Six rounds of that comparison found a missing 5 ms chip-settle delay —
and reading which error *wrapper string* fired told exactly which layer had moved
after each attempted fix.

## Rung 8: cross-check against an independent implementation

The final rung removes GopherTrunk from the experiment. In
[#345](https://github.com/MattCheramie/GopherTrunk/issues/345), running
`p25_survey` against the same site printed all sixteen band-plan entries and
exposed the one TSBK opcode GopherTrunk never dispatched. In
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764), decimating a
suspect 10 MS/s capture with an *independent* resampler and feeding the result to
the proven 2.5 MS/s path reproduced the same 10 dB deficit — proving the damage
was baked into the captured samples, not GopherTrunk's DSP (see
[Airspy rate selection](/reference/airspy-rate-selection/)). If an independent
tool sees the same fault, stop debugging the software.

## Provenance

- [#275](https://github.com/MattCheramie/GopherTrunk/issues/275) — the first P25 control-channel lock; real-capture ground truth and the `replay` subcommand ended a five-day guess-and-retest loop.
- [#376](https://github.com/MattCheramie/GopherTrunk/issues/376) — the talker-alias hunt; a per-opcode census disproved three wrong transport theories.
- [#813](https://github.com/MattCheramie/GopherTrunk/issues/813) — P25 Phase 2 encryption fields; the unconditional per-call census as a three-way stage disambiguator.
- [#248](https://github.com/MattCheramie/GopherTrunk/issues/248) — NESDR tuner-init failures; six rounds of paired USB traces.
- [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) — Airspy 10 MS/s deficit; the independent-resampler cross-check.
- [#345](https://github.com/MattCheramie/GopherTrunk/issues/345) — the "pipeline stall" that was a USB disconnect (`dmesg`) plus a band-plan gap found via `p25_survey`.
- [#598](https://github.com/MattCheramie/GopherTrunk/issues/598) — silent live audio; the two `curl` probes that located the missing PCM.
