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

## Cheat sheet

| Layer | What was wrong | How it was proved | Fix |
|---|---|---|---|
| Reporter's premise | "Gardner loop hard-coded for 4 sps" | `NewGardner(sps, gain)` takes sps as a parameter; 144 kHz ÷ 18,000 sym/s = exactly 8 sps by design | none needed — theory credited and retired |
| Training sequences | placeholder constants: a `uint64` "holding" 38 dibits = 76 bits; bits/dibits conflated | exhaustive search over the whole capture never correlated once | real ETSI EN 300 392-2 §9.4.4.3 sequences as bit arrays |
| Dibit mapping | linear `DibitsToBits` (C4FM convention) where the air is Gray-coded | half of all symbols deliver one inverted bit; CRC fails downstream of a found sync | `TetraBitsToDibits`/`TetraDibitsToBits` Gray convention |
| Carrier offset | live DDC had no AFC — off-center channel biases every transition | training sequence never correlates even when correct | AFC (blind acquisition + decision-directed tracking) before timing recovery |
| The tests | all green | fixtures generated from the same wrong constants — self-consistent | independent-literal guard `TestTrainingSequencesMatchETSI` |
| The capture | still no `cc.locked` after all fixes | reference demodulator also recovers **zero** BSCH frames | none possible — SNR limit of the recording, not the chain |

## In this post

- **The symptom, and a wrong premise from the reporter** — 8 samples per symbol was a choice, not a bug.
- **First, prove the demodulator** — baud, autocorrelation, and power say the front end is fine.
- **Constants that were never real** — the placeholder training sequences, and the table they should have been.
- **The second layer: linear where the air is Gray** — one mapping per protocol family, and what a wrong one rots.
- **Why every test was green** — the self-consistent fixture trap.
- **A misdiagnosis, caught by the reporter** — spectrum inversion, retired by SDR++.
- **The metric that proves a sync layer** — 97 hits, 1020 dibits apart.
- **The honest ending** — a repaired chain and a capture that can't showcase it.
- **What we keep** — the durable lessons.

## The symptom, and a wrong premise from the reporter

"TETRA demodulator fails to lock." The reporter came armed with a theory: the
Gardner timing-recovery loop looked hard-coded for 4 samples per symbol, wrong for
TETRA's rate.

The theory didn't survive contact with the code. `NewGardner(sps, gain)` takes
samples-per-symbol as a parameter, and the DDC's TETRA output rate —
`tetraDDCTargetRateHz = 144000.0` — is chosen deliberately: 144,000 samples per
second over TETRA's 18,000 symbols per second is exactly 8 sps, and 2 MS/s → 144
kS/s reduces to the exact rational 9/125, so the resampler is clean. The premise
was also stress-tested rather than just argued: an end-to-end simulation through
the production path swept clock offsets to 5000 ppm, carrier offsets to 300 Hz, and
SNR down to 6 dB across a grid of Gardner gains, and the 8-sps path locked in
every case, identically to a 4-sps variant. Reporters' theories are leads, not
conclusions — this one was checked, credited, and retired in the first exchange.
But the symptom was real, so something else was broken.

## First, prove the demodulator

The reporter supplied what makes a thread like this tractable: a raw IQ capture of
the site (`tetra_468775_2msps.cfile`, 2 MS/s). Before accusing any decode layer,
the investigation established what the *demodulator* recovered from it, using
measurements that don't depend on any downstream constant being right:

- Symbol timing converged to an effective baud of **17,999.7 Hz** against TETRA's
  nominal 18,000 — the timing loop was locked onto a real symbol clock.
- The recovered symbol stream's autocorrelation showed **0.98 periodicity at a
  fixed lag** (and its harmonics) — genuinely structured, repeating signal, not
  noise.
- Mean level was **−28.6 dBFS**, matching the reporter's logged `iq_power_dbfs`
  exactly — a healthy front end, no clipping.

So the front-end DSP was demonstrably fine: real TETRA symbols were coming out of
the timing loop. Amplitude and loop-gain hypotheses were tested against the
capture and ruled out — adding an AGC and varying the Gardner gain changed
nothing. Whatever was broken sat *between* good symbols and a sync that never
fired. That is a very short list, and at the top of it is the pattern the
correlator was searching for.

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

The real values, from EN 300 392-2 §9.4.4.3, are short enough to print in full —
which is part of the lesson, because printing them next to a section number is
exactly what the original code never did:

| Sequence | Bits | Carried by | On-air value (MSB first) |
|---|---|---|---|
| NTS1 (§9.4.4.3.2) | 22 | normal continuous downlink burst | `1101000011 1010011101 00` |
| NTS2 (§9.4.4.3.2) | 22 | normal continuous downlink burst | `0111101001 0000110111 10` |
| ETS (§9.4.4.3.3) | 30 | extended training | `1001110100 0011101001 1101001110` |
| STS (§9.4.4.3.4) | 38 | synchronization burst (SB), slot 1 of frame 18 of each multiframe | `1100000110 0111001110 1001110000 01100111` |

Nothing in the placeholder file matched any row of that table. And the mismatch
was proven, not assumed: an exhaustive search over the entire 2 MHz capture —
every carrier, both spectrum orientations, all 24 possible dibit relabelings, with
and without AGC — never found the placeholder pattern once. The correlator had
spent its entire life searching real air for a pattern that no TETRA transmitter
has ever sent.

## The second layer: linear where the air is Gray

Fixing the constants wasn't enough, because underneath the correlator sat a second
transcription error. TETRA is π/4-DQPSK, and the phase-transition-to-bits mapping
on air is **Gray-coded**: demodulated symbol pair `(b1, b2)` maps as
`(b1<<1) | (b1^b2)`, i.e. `00→0, 01→1, 11→2, 10→3`. GopherTrunk's TETRA path
reused `framing.DibitsToBits` — the *linear* convention, correct for the 4800-baud
C4FM family (P25, DMR, NXDN) but wrong here. The linear convention effectively
flips B2 whenever B1 = 1: half of all symbols, statistically, delivered one bit
inverted.

The two conventions differ on exactly the two middle rows:

| On-air bits (b1 b2) | Gray dibit (TETRA) | Linear dibit (C4FM family) |
|---|---|---|
| 0 0 | 0 | 0 |
| 0 1 | 1 | 1 |
| 1 1 | 2 | 3 |
| 1 0 | 3 | 2 |

Gray coding exists on phase modulations for a physical reason: adjacent phase
transitions differ in only one bit, so the most likely demodulation error — landing
one decision region over — costs one bit instead of two. Which means a wrong
mapping is *statistically camouflaged*: the stream looks plausible, the symbol
distribution is right, and only exact pattern matches and CRCs betray it. The fix
made `TetraBitsToDibits`/`TetraDibitsToBits` the single source of truth for the
TETRA convention, deliberately distinct from the C4FM family's linear helpers, and
routed the channel decode through them.

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
source: the standard's tables, another implementation, or real air. The repaired
code pins that lesson in place: `TestTrainingSequencesMatchETSI` compares the
constants against an independently-transcribed literal, so re-introducing a
placeholder fails a test that the code under test cannot influence.

## A misdiagnosis, caught by the reporter

Mid-investigation, the maintainer's "Finding 1" declared the reporter's capture
spectrum-inverted — I/Q swapped, a classic and common capture defect. It was a
seductive theory precisely because it fit the modulation: an inverted spectrum
reverses every differential phase transition, and without conjugation the
constellation genuinely collapsed toward two of four points. The reporter did the
check that keeps a thread honest: loaded the same capture into SDR++ and
demonstrated that TETRA decoding there requires I/Q inversion **off**. The finding
was withdrawn, and a fresh capture confirmed the non-inverted orientation both
ways. (Its one artifact survived on merit: a per-device `iq_invert` config option,
which is genuinely useful for hardware that does invert.)

Two other pieces had to land before the fixed correlator could prove anything on
real air. The live DDC had **no AFC** — a channel sitting a little off-center
biases every phase transition, and a biased transition stream never matches a
training sequence. Automatic frequency correction was added ahead of timing
recovery: a data-blind acquisition stage plus decision-directed tracking that
derotates the matched-filter output before the Gardner loop ever sees it. And a
channel-select filter (≈±15 kHz) went in ahead of the matched filter, because the
144 kHz channelized passband is wide enough to admit adjacent carriers that
pollute the transition statistics. Only then came the measurement that settled the
issue.

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

The same batch of work closed the configuration gap that would have greeted the
next user: the colour code that seeds TETRA's descrambler is now auto-learned
rather than operator-supplied. The BSCH — the one channel that is always
scrambled with colour zero — is decoded first to recover the colour code, MCC,
and MNC; the 30-bit scrambling code is formed from them; and the BNCH SYSINFO
descrambles from there with no manual configuration at all.

## The honest ending

And then the capture still didn't lock. With sync provably solid, the numbers said
why: at the true frame positions the symbol error rate ran 10–18% — only one of
the 97 correlation hits was an exact-match training sequence — which is more
corruption than the RCPC and CRC behind it can close, regardless of framing. The
decisive cross-check was a best-available reference demodulator given the same
file: best-phase search plus AFC managed 55/176 exact training-sequence matches
and recovered **zero** BSCH frames. The recording itself was SNR-limited: the
decode chain had been broken *and* the test material was insufficient to
demonstrate the repaired chain end-to-end. Both things were true. The issue closed
on the strength of the frame-cadence evidence plus the reference cross-check, not
on a satisfying `cc.locked` — because that's what the evidence actually supported.
TETRA locking on real air got proven later, on better captures, and the production
8-sps path now carries its own end-to-end lock regression so the original wrong
premise can never silently become true. Today's lock criteria are collected in the
Field Guide.

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

## FAQ

**How do placeholder constants survive in a decoder for so long?**
Because everything around them agreed with them. The tests built their fixtures
from the same constants, the encoder and decoder shared the same convention, and
no fixture ever came from the standard or from real air. Nothing in CI could
disagree. The first independent input the constants ever met was the reporter's
capture — and it disagreed completely.

**Why does π/4-DQPSK use Gray coding at all?**
The likeliest demodulation error on a phase modulation is mistaking a transition
for its nearest neighbor. Gray coding arranges the bit pairs so adjacent
transitions differ in exactly one bit, halving the bit cost of the most common
symbol error. The flip side, as this issue showed, is that a wrong mapping is
invisible to every statistical check — only exact matches and CRCs expose it.

**Couldn't the correlator have been fixed without the AFC and channel filter?**
The constants and mapping could have been corrected, but they couldn't have been
*proven* on real air: with no AFC, a channel a little off-center biases every
phase transition and the correct sequence still never correlates, and the wide
144 kHz passband lets adjacent carriers pollute the statistics. The AFC and the
±15 kHz channel-select filter aren't part of the bug — they're what made the
frame-cadence measurement possible.

**Why close the issue without a full control-channel lock?**
Because the remaining gap was measured to be in the recording, not the code: a
10–18% symbol error rate at the true frame positions, and a reference
demodulator that recovered zero BSCH frames from the same file. The frame-cadence
comb proved the sync layer; the reference cross-check proved the residual was
signal-limited. Holding the issue open for a better capture would have kept a
fixed decoder filed under "broken."

**What guards this from regressing?**
Two tests with independent ground truth: `TestTrainingSequencesMatchETSI` compares
the constants against a separately-transcribed literal from the standard, and an
end-to-end lock test pins the production 144 kHz / 8-sps receiver path so the
original "hard-coded sps" premise can never quietly become true.

## Series navigation

**Part 17 of 22** · ←
[Part 16: The Channel That Was Its Own Voice Channel — Conventional FM and the IQ Broker]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }})
· Next →
[Part 18: The Stall That Wasn't — A Dongle Off the Bus and an Opcode Off the Books]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }})
