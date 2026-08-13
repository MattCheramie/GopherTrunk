---
title: "From the Issue Tracker, Part 17: Placeholder Constants — The TETRA Sync That Never Existed"
description: TETRA never locked because the training-sequence constants were placeholders — 76 bits declared into a 64-bit variable, bits conflated with dibits, and a linear dibit mapping where the air uses Gray — and every test passed because the fixtures were built from the same wrong constants.
category: solution-postmortem
keywords: tetra, training sequence, sync correlation, gray mapping, pi/4-dqpsk, placeholder constants, self-consistent tests, spectrum inversion, afc, gophertrunk issue tracker
tags: [from-the-issue-tracker, tetra, dsp, sync, testing, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 17
---

*Part 17 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 16]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
was two bugs where fixing one revealed the other. This one is the opposite
geometry: a decoder in which almost every layer of the sync path was wrong at once,
all the tests were green, and even the ending refuses to be tidy.*

> **TL;DR:** In [#553](https://github.com/MattCheramie/GopherTrunk/issues/553),
> the TETRA demodulator never locked on a real capture. The reporter blamed the
> Gardner timing loop's samples-per-symbol; that premise was wrong. The real
> problems were more embarrassing and more instructive: the training-sequence
> constants in `sync.go` were **placeholders** — a 64-bit variable declared to hold
> 76 bits, with bits and dibits conflated throughout — and the dibit mapping was
> linear where TETRA's π/4-DQPSK is Gray-coded on air. Every unit test passed
> because every fixture was generated from the same wrong constants. Along the way
> a spectrum-inversion misdiagnosis was caught by the reporter with SDR++, an AFC
> had to exist before the correlator could ever fire, and the proof of the fix was
> a metric worth memorizing: correlation hits at exactly one-frame cadence. The
> honest ending: even fully fixed, the capture itself was too noisy to lock.

## The symptom, and a wrong premise from the reporter

"TETRA demodulator fails to lock." The reporter came armed with a theory: the
Gardner timing-recovery loop looked hard-coded for 4 samples per symbol, wrong for
TETRA's rate.

The theory didn't survive contact with the code. `NewGardner(sps, gain)` takes
samples-per-symbol as a parameter, and the DDC's TETRA output rate —
`tetraDDCTargetRateHz = 144000.0` — is chosen deliberately: 144,000 samples per
second over TETRA's 18,000 symbols per second is exactly 8 sps, and 2 MS/s → 144
kS/s reduces to the exact rational 9/125, so the resampler is clean. Reporters'
theories are leads, not conclusions — this one was checked, credited, and retired
in the first exchange. But the symptom was real, so something else was broken.

## Constants that were never real

The something else was `internal/radio/tetra/sync.go`, and it did not require deep
DSP insight to find — just the willingness to compare constants against the
standard. The file held the training sequences the correlator hunts for, and they
were **placeholders that had never been checked against ETSI EN 300 392-2**. Two
separate confusions were fossilized in the code:

- `NormalSyncHex` was declared as a `uint64` — 64 bits — while the comment said it
  held 38 dibits. Thirty-eight dibits is **76 bits**. The variable physically could
  not contain the value it claimed to; six dibits were silently zero-filled.
- The naming conflated TETRA's 38-**bit** synchronization training sequence (19
  dibits) with "38 dibits." Bits and dibits had been swapped somewhere in the
  original transcription and never reconciled.

The real values, from EN 300 392-2 §9.4.4.3: the normal training sequences NTS1 and
NTS2 are 22 bits each, the extended sequence ETS is 30 bits, and the
synchronization sequence STS is 38 bits. Nothing in the file matched. The
correlator had spent its entire life searching real air for a pattern that no TETRA
transmitter has ever sent.

## The second layer: linear where the air is Gray

Fixing the constants wasn't enough, because underneath the correlator sat a second
transcription error. TETRA is π/4-DQPSK, and the phase-transition-to-bits mapping
on air is **Gray-coded**: demodulated symbol pair `(b1, b2)` maps as
`(b1<<1) | (b1^b2)`, i.e. `00→0, 01→1, 11→2, 10→3`. GopherTrunk's TETRA path
reused `framing.DibitsToBits` — the *linear* convention, correct for the 4800-baud
C4FM family (P25, DMR, NXDN) but wrong here. The linear convention effectively
flips B2 whenever B1 = 1: half of all symbols, statistically, delivered one bit
inverted.

Note what that does to debuggability. A wrong mapping downstream of a fixed
correlator means you can *find* sync — the sequence match tolerates it or is
patched around — and then every subsequent stage silently rots: descramble runs on
corrupted bits, deinterleave shuffles corrupted bits, and the CRC at the end fails
with no indication of which layer lied.

## Why every test was green

Both bugs — placeholder constants and the linear mapping — were covered by tests.
All passing. The fixtures were **generated by the same code under test**: bursts
built from the same placeholder sequences, modulated under the same linear
convention, then decoded by the mirror-image path. A round trip through a wrong
convention is still a round trip. Self-consistency is the property such tests
verify, and self-consistency is exactly what a transcription error preserves. The
only fixture that can catch a wrong constant is one derived from an *independent*
source: the standard's tables, another implementation, or real air.

## A misdiagnosis, caught by the reporter

Mid-investigation, the maintainer's "Finding 1" declared the reporter's capture
spectrum-inverted — I/Q swapped, a classic and common capture defect. The reporter
did the check that keeps a thread honest: loaded the same capture into SDR++ and
demonstrated that TETRA decoding there requires I/Q inversion **off**. The finding
was withdrawn. (Its one artifact survived on merit: a per-device `iq_invert`
config option, which is genuinely useful for hardware that does invert.)

Two other pieces had to land before the fixed correlator could prove anything on
real air. The live DDC had **no AFC** — a channel sitting a little off-center
biases every phase transition, and a biased transition stream never matches a
training sequence; automatic frequency correction had to be added ahead of timing
recovery. Only then came the measurement that settled the issue.

## The metric that proves a sync layer

After the real constants, the Gray mapping, and the AFC:

| Measurement | Before | After |
|---|---|---|
| Training-sequence correlation hits | 0 | 97 |
| Modal spacing between hits | — | exactly 1020 dibits |

Why 1020 matters more than 97: a TETRA frame is 4 slots × 255 dibits = **1020
dibits**. Training sequences recur once per frame, so a correlator that is truly
finding them must fire at exactly that cadence. Random false positives don't
cluster at one spacing; a broken mapping doesn't produce a comb at the frame
period. "Hits at exact frame cadence" validates the entire sync layer — mixing,
timing, mapping, correlation — using nothing downstream of it. No CRC, no
descrambler, no protocol decode required. It is the cleanest possible seam between
"the demodulator works" and "the rest of the chain works."

## The honest ending

And then the capture still didn't lock. With sync provably solid, the BSCH decode
behind it recovered nothing — and neither did a best-available reference
demodulator given the same file, which recovered **zero** BSCH frames. The
recording itself was SNR-limited: the decode chain had been broken *and* the test
material was insufficient to demonstrate the repaired chain end-to-end. Both things
were true. The issue closed on the strength of the frame-cadence evidence plus the
reference cross-check, not on a satisfying `cc.locked` — because that's what the
evidence actually supported. TETRA locking on real air got proven later, on better
captures, and today's lock criteria are collected in the Field Guide.

## What we keep

- **Placeholder constants are landmines with no expiry.** A 64-bit variable
  "holding" 76 bits should be impossible to write; a units mix-up (bits vs dibits)
  in a name is a standing invitation to the next one. Transcribe constants from
  the standard with the section number in a comment — see
  [TETRA training sequences]({{ '/reference/tetra-training-sequences/' | relative_url }}).
- **Know your symbol mapping per protocol family.** Linear for the C4FM family,
  [Gray]({{ '/reference/gray-code/' | relative_url }}) for TETRA's π/4-DQPSK —
  reusing a mapping across families corrupts every layer downstream of sync.
- **Self-consistent round-trip tests cannot catch a wrong constant.** Encode and
  decode through the same error and it cancels. At least one fixture per protocol
  must come from an independent source.
- **Correlation hits at exact frame cadence** is the decisive sync-layer metric —
  it isolates the demodulator from everything behind it. This and the rest of the
  lock checklist live in
  [TETRA lock facts]({{ '/reference/tetra-lock-facts/' | relative_url }}).
- **Verify capture-defect theories in a second tool.** One SDR++ session retired
  the spectrum-inversion misdiagnosis. And accept endings where the fix is real
  but the capture can't showcase it — a chain can be repaired and still be
  signal-limited, per
  [signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).

## Series navigation

← [Part 16: conventional FM and the IQ broker]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
· Next → [Part 18: the stall that wasn't]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }})
