---
slug: signal-signatures
title: Signal-quality signatures
entry_type: term
category: fn-diagnostics
description: "Signal-quality signatures are the recurring measurement patterns from GopherTrunk's issue tracker that identify a fault class on sight — overload, misalignment, phase noise, adjacent-channel lock — before any code is read."
keywords: signal quality, symptom table, image rejection, slicer collapse, uncorrectable frames, overload, rail-pinned, phase noise, evm trap, carrier offset, diagnosis
aka: [signal fingerprints, symptom signatures, diagnostic fingerprints]
infobox:
  - { label: Type, value: Diagnostic catalog }
  - { label: Format, value: "Numeric fingerprint → what it looks like → what it actually means" }
  - { label: Examples, value: "Rail-pinned histogram, +78° phase imbalance, ±12.5 kHz offset" }
see_also: [diagnostic-playbook, carrier-offset-adjacent-lock, sdr-gain-overload, airspy-rate-selection, error-vector-magnitude, iq-imbalance, image-rejection, cma-equalizer]
related_reading:
  - { title: "From the Issue Tracker, Part 12: Seventy-Eight Degrees — The Phase Angle That Named the Bug", url: /blog/solution-postmortem/from-the-issue-tracker-12-seventy-eight-degrees/ }
  - { title: "From the Issue Tracker, Part 8: Nineteen Dibits — A Perfect Hypothesis Meets a Rail-Pinned ADC", url: /blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/ }
  - { title: "From the Issue Tracker, Part 5: Ten Megasamples — When the Bug Is in the Samples Themselves", url: /blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/454
  - https://github.com/MattCheramie/GopherTrunk/issues/275
  - https://github.com/MattCheramie/GopherTrunk/issues/489
  - https://github.com/MattCheramie/GopherTrunk/issues/881
  - https://github.com/MattCheramie/GopherTrunk/issues/764
  - https://github.com/MattCheramie/GopherTrunk/issues/553
  - https://github.com/MattCheramie/GopherTrunk/issues/815
  - https://github.com/MattCheramie/GopherTrunk/issues/1001
---

**Signal-quality signatures** are measurement patterns that recur in GopherTrunk's
issue tracker often enough to be diagnostic on sight. Each row below pairs a
specific numeric fingerprint with what it *looks* like at first glance and what it
*actually* meant when it was finally run to ground. Recognizing a row saves the
detour the original investigation took; the
[diagnostic playbook](/reference/diagnostic-playbook/) covers how to collect these
measurements in the first place.

| Signature | What it looks like | What it actually means | Source issue |
|---|---|---|---|
| Phase imbalance ≈ +78°, image rejection ≈ 3 dB | A badly miscalibrated tuner | A real-sample stream misread as interleaved I/Q | [#454](https://github.com/MattCheramie/GopherTrunk/issues/454) |
| Sample histogram rail-pinned (~25% at 0 and ~25% at 255), RMS near 0 dBFS, FFT looks clean | Weak or absent signal, since nothing decodes | Front-end overload; the fix is *less* gain, fixed | [#881](https://github.com/MattCheramie/GopherTrunk/issues/881) |
| Wideband FFT carrier SNR *higher*, in-channel EVM much worse | The DSP breaks at the higher sample rate | Sampling-clock phase noise; the damage is in the captured samples | [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) |
| Dibit histogram 50/0/50/0 — only outer symbols | Corrupt demodulator output | Slicer collapse: symbol amplitude far above the slicer's calibrated levels | [#275](https://github.com/MattCheramie/GopherTrunk/issues/275) |
| Sync correlation hits at exactly one-frame cadence | Just another intermediate metric | Proof the sync layer works, independent of downstream CRC | [#553](https://github.com/MattCheramie/GopherTrunk/issues/553) |
| `offset_hz` ≈ ±12.5 kHz on a locked control channel | A drifty crystal | Locked onto an *adjacent channel's* carrier; the decoded identity is another site's | [#815](https://github.com/MattCheramie/GopherTrunk/issues/815) |
| Uncorrectable frames at exactly 100% | Hopelessly weak signal | A structural bit-alignment bug — signal degrades gradually, never to a deterministic 100% | [#489](https://github.com/MattCheramie/GopherTrunk/issues/489) |
| EVM improving while CRC yield stays at 0 | An equalizer that is helping | The blind-equalizer trap: better modulus is not better bits | [#1001](https://github.com/MattCheramie/GopherTrunk/issues/1001) |

## +78° phase, 3 dB image rejection: real samples read as complex

The Airspy R2/Mini stream bare real ADC samples at twice the IQ rate; converting
to complex baseband is the host's job. A driver that pairs adjacent real samples
as I and Q produces this textbook fingerprint: neighbouring samples of an
oversampled signal are highly correlated, so the apparent
[phase imbalance](/reference/iq-imbalance/) lands near 90° (measured +78.1°) with
[image rejection](/reference/image-rejection/) of only ~3.3 dB. After a proper
Hilbert-pair converter the same metrics read −0.0007° and ~70.8 dB. See
[Airspy rate selection](/reference/airspy-rate-selection/).

## Rail-pinned histogram: overload that decodes anyway

In [#881](https://github.com/MattCheramie/GopherTrunk/issues/881) a raw capture
had 24.9% of its 8-bit samples at exactly 0 and 24.9% at exactly 255 — half the
stream pinned to the rails, RMS at +1.3 [dBFS](/reference/dbfs/) — yet the FFT
showed clean-looking carriers and an offline decode still recovered 3 of 4 control
channels, because constant-envelope [C4FM](/reference/c4fm/) carries its
information in phase and survives hard limiting. That combination (clean FFT,
saturated histogram, dead live decode) misdirected the investigation toward a
compelling but wrong DSP hypothesis. The counter-intuitive rule: on an overloading
front end, more gain makes it worse — the fix was a fixed low gain, not
[AGC](/reference/automatic-gain-control/). See
[SDR gain & overload](/reference/sdr-gain-overload/).

## Carrier-clean but modulation-degraded: clock phase noise

In [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) the wideband FFT
showed the carrier *cleaner* at 10 MS/s (SNR 36.2 dB vs 33.1 dB) while in-channel
demod quality was ~10 dB worse ([EVM](/reference/error-vector-magnitude/) 22.5%
vs 7.4%). Neither capture clipped (peaks near −48 dBFS), ruling out overload. A
high carrier-to-noise floor with smeared modulation is the signature of
[phase noise](/reference/phase-noise/) on the sampling clock (reciprocal mixing):
the energy is all there, but its phase is jittered. The deficit was proven to be
in the captured samples themselves via an independent-resampler cross-check.

## 50/0/50/0 dibit histogram: slicer collapse

When the four-level slicer's calibrated thresholds sit far below the actual symbol
amplitude, every symbol reads as an outer level — the dibit histogram collapses to
two bins. The treacherous part: frame-sync correlation *still works*, because the
P25 sync word uses only outer symbols, so the decoder looks half-alive. A
symbol-domain AGC restored a healthy 28/22/27/23 spread in
[#275](https://github.com/MattCheramie/GopherTrunk/issues/275). Compare the
[eye diagram](/reference/eye-diagram/) view of the same failure: all trajectories
pass wide of the inner levels.

## Sync hits at exact frame cadence: the sync layer is proven

The decisive positive signature from
[#553](https://github.com/MattCheramie/GopherTrunk/issues/553): TETRA
training-sequence correlation went from 0 hits to 97 hits with a modal spacing of
exactly 1020 dibits — one TETRA frame. Random correlation noise does not arrive on
a frame grid. This proves sync acquisition works on real air *independent of
downstream [CRC](/reference/cyclic-redundancy-check/) results*, which cleanly
splits "sync is broken" from "descramble/deinterleave/FEC is broken."

## `offset_hz` near ±12.5 kHz: you locked the neighbour

GopherTrunk locks whatever compatible carrier the matched filter finds in the
channel passband and reports the *configured* frequency. A measured carrier
offset sitting at the channel spacing (±12.5 kHz for narrowband P25) means the
lock is an adjacent channel — and every decoded site identity belongs to the
neighbour. Full treatment in
[carrier offset & adjacent-channel lock](/reference/carrier-offset-adjacent-lock/).

## Exactly 100% uncorrectable: structural, not signal

Weak signal produces a *distribution* of frame quality. In
[#489](https://github.com/MattCheramie/GopherTrunk/issues/489),
`ldus=1622 uncorrectable_ldus=1622` — exactly 100%, deterministically — meant the
voice-frame bit offsets were structurally wrong (the decoder omitted the link
control blocks interleaved between voice subframes), so the FEC never saw aligned
codewords. When *every* frame fails, suspect alignment tables and bit ordering
before antennas. The round-trip tests passed because encoder and decoder shared
the same wrong offset table.

## EVM improving while CRC stays 0: the blind-equalizer trap

A blind [CMA equalizer](/reference/cma-equalizer/) minimizes deviation from a
constant modulus — not decoding correctness — and its cost surface has spurious
minima. During the TETRA equalizer work
([#1001](https://github.com/MattCheramie/GopherTrunk/issues/1001)), a numerically
unstable variant showed differential EVM collapsing from 34% to 8% while CRC-valid
frame yield stayed at exactly zero: the equalizer had made the constellation
*rounder*, not *righter*. The durable rule recorded in the project's engineering
notes: never conclude an equalizer helps from EVM — decode all the way to CRC
yield, the only trustworthy metric.

## Provenance

- [#454](https://github.com/MattCheramie/GopherTrunk/issues/454) — Airspy real-sampling stream misread as I/Q; the +78°/3 dB fingerprint.
- [#275](https://github.com/MattCheramie/GopherTrunk/issues/275) — first P25 lock; the 50/0/50/0 slicer-collapse histogram.
- [#489](https://github.com/MattCheramie/GopherTrunk/issues/489) — 100% uncorrectable LDUs as a structural-misalignment tell.
- [#881](https://github.com/MattCheramie/GopherTrunk/issues/881) — rail-pinned overload behind a clean-looking FFT.
- [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) — carrier-clean but modulation-degraded = sampling-clock phase noise.
- [#553](https://github.com/MattCheramie/GopherTrunk/issues/553) — TETRA lock; correlation hits at exact frame cadence as sync proof.
- [#815](https://github.com/MattCheramie/GopherTrunk/issues/815) — the ±12.5 kHz adjacent-channel lock fingerprint.
- [#1001](https://github.com/MattCheramie/GopherTrunk/issues/1001) — TETRA equalizer work where the EVM-vs-CRC trap was pinned down.
