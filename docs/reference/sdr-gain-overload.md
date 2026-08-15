---
slug: sdr-gain-overload
title: SDR gain & front-end overload
entry_type: term
category: fn-hardware
description: "SDR gain & front-end overload covers the trap that GopherTrunk gain values are tenths of a decibel, plus the overload signature — rail-pinned samples behind a deceptively clean FFT — that AGC makes worse and fixed low gain fixes."
keywords: gain, tenths of db, agc, overload, saturation, clipping, rail-pinned, dbfs, histogram, c4fm, front end, intermodulation, wideband
aka: [gain in tenths of a dB, front-end overload, ADC saturation]
infobox:
  - { label: Type, value: Config + RF trap }
  - { label: Key rule, value: "gain is tenths of a decibel — 300 means 30 dB" }
  - { label: Signature, value: Rail-pinned sample histogram behind a clean-looking FFT }
  - { label: Fix, value: "Fixed low gain, not AGC" }
see_also: [automatic-gain-control, dbfs, intermodulation, c4fm, rtl-sdr, airspy-rate-selection, rtlsdr-usb-recovery, usrp-soapyremote-notes, signal-signatures, diagnostic-playbook]
related_reading:
  - { title: "From the Issue Tracker, Part 8: Nineteen Dibits — A Perfect Hypothesis Meets a Rail-Pinned ADC", url: /blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/881
  - https://github.com/MattCheramie/GopherTrunk/issues/935
  - https://github.com/MattCheramie/GopherTrunk/issues/876
---

**SDR gain and front-end overload** cause two of the most common "nothing decodes"
reports, and both hide behind healthy-looking spectrum displays. The first is a units
trap: GopherTrunk's `gain` is in **tenths of a decibel**. The second is that an
overloaded front end produces samples that *still look fine on an FFT* — because
constant-envelope modulations like [C4FM](/reference/c4fm/) survive hard limiting —
while the live decode path is dead.

## Gain is tenths of a dB

`gain: "300"` means 30 dB. A bare `"30"` means 3.0 dB — enough attenuation to make a
workable signal undecodable, with no error anywhere
([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)).

This trap decided a real case: in
[#935](https://github.com/MattCheramie/GopherTrunk/issues/935) a Melbourne CBD P25
site refused to lock, and the investigation went deep into demodulator-mode theories
before the actual problem surfaced — the gain that worked in SDRTrunk (~36 dB) had
never been ported into GopherTrunk's tenths format. `gain: "363"` produced an
immediate lock and roughly 2,000 grants in five minutes.

Practical rules:

- When porting a known-good setup from SDR++ or SDRTrunk, multiply the dB figure by 10.
- For survey and field work, prefer a **manual** gain value found empirically in SDR++
  over auto/[AGC](/reference/automatic-gain-control/)
  ([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)).

## The overload signature

[#881](https://github.com/MattCheramie/GopherTrunk/issues/881) is the reference case.
A wideband VHF device with four control channels never decoded live, while a
near-identical sibling device decoded thousands of TSBKs. The reporter built a
compelling hypothesis about chunk sizes ("~19 dibits per chunk is less than the
24-dibit frame sync"), backed by log evidence — which turned out to be a lock-state
artifact: the `no FSW hits in chunk` line only fires while *unlocked*, so the working
device stopped emitting it the moment it locked, not because its chunks grew.

A raw capture settled it. The IQ was hard-saturated:

| Symptom | Looks like | Actually | Fix/Check |
|---|---|---|---|
| ~25% of u8 samples exactly 0 **and** ~25% exactly 255 (about half rail-pinned) | Strong healthy signal | AGC-driven front-end saturation | Histogram the raw capture; anything beyond a fraction of a percent at the rails is overload |
| Raw IQ RMS near or above 0 [dBFS](/reference/dbfs/) (measured +1.3 dBFS) | — | The ADC is clipping continuously | Target comfortable headroom; see the working sibling for a baseline |
| FFT shows clean, well-shaped carriers | RF path is fine | Constant-envelope C4FM keeps its *shape* through hard limiting — spectrum is the wrong instrument for overload | Look at the amplitude histogram, not the spectrum |
| The same capture decodes offline (3 of 4 control channels) | Live pipeline bug | Phase/frequency information survives limiting well enough for a forgiving offline decode, while the live path starves | Believe the histogram over the decode result |
| `wideband front end overloaded` WARN | Noise | The correct diagnosis, emitted by GopherTrunk itself | Act on it: reduce gain |

The confusing part of #881 was that overload *coexisted* with partial decodability:
clipping is not all-or-nothing for angle modulations, so "the FFT looks great and
offline replay half-works" does not rule it out.

## The counter-intuitive fix: less gain, fixed

On a front end being overloaded by strong in-band or near-band signals, **more gain
makes it worse** — and AGC, which responds to total power, will happily drive the
device into the rails. The fix in #881 was a fixed `gain: "200"` (20 dB), replacing
AGC. The general rule:

- On a hot band (VHF high sites, close transmitters), use fixed low gain, not AGC.
- If reducing gain changes the decode markedly, the problem was compression or
  [intermodulation](/reference/intermodulation/); if a quality deficit is *independent*
  of gain across a wide range, it is not overload — see
  [Airspy sample-rate selection](/reference/airspy-rate-selection/) for the
  phase-noise case that gain cannot touch.
- Dropping `sample_rate` to escape an oversampling warning is not a lever here: the
  RTL wideband path snaps back to 2.048 MS/s
  ([#881](https://github.com/MattCheramie/GopherTrunk/issues/881)).

## Provenance

- [#881](https://github.com/MattCheramie/GopherTrunk/issues/881) — the rail-pinned-histogram overload signature; the plausible-but-wrong short-chunk hypothesis and its lock-state log artifact; fixed low gain over AGC.
- [#935](https://github.com/MattCheramie/GopherTrunk/issues/935) — the tenths-of-a-dB port from SDRTrunk (`gain: "363"`) that turned a dead site into ~2K grants in five minutes.
- [#876](https://github.com/MattCheramie/GopherTrunk/issues/876) — gain units on the USRP/SoapyRemote path and the prefer-manual-gain field advice.
