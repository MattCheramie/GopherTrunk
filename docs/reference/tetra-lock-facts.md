---
slug: tetra-lock-facts
title: TETRA lock facts
entry_type: term
category: fn-protocol
description: "TETRA lock facts are the protocol and DSP truths that separate a TETRA demodulator that locks real air from one that only passes its own tests — training-sequence bit lengths, the Gray dibit mapping, the 4th-power AFC's ±2250 Hz alias limit, and why colour code 0 still scrambles."
keywords: tetra, training sequence, nts, ets, sts, bsch, gray mapping, pi/4-dqpsk, afc, frequency offset, aliasing, colour code, scrambler, dmo, frame cadence, etsi en 300 392-2
aka: [TETRA bring-up facts, TETRA sync gotchas]
infobox:
  - { label: Type, value: Protocol + DSP facts }
  - { label: Key rule, value: Training sequences are counted in bits (22/30/38) }
  - { label: Trap, value: Colour code 0 still scrambles (LFSR seed 0xC0000000) }
  - { label: Proof of sync, value: Correlation hits at exactly 1020-dibit frame cadence }
see_also: [tetra, tetra-training-sequences, tetra-scrambler, tetra-extended-colour-code, pi-4-dqpsk, gray-code, automatic-frequency-control, dibit, dmr-grant-field-gotchas, signal-signatures]
related_reading:
  - { title: "From the Issue Tracker, Part 17: Placeholder Constants — The TETRA Sync That Never Existed", url: /blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/553
  - https://github.com/MattCheramie/GopherTrunk/issues/940
  - https://github.com/MattCheramie/GopherTrunk/issues/648
  - https://github.com/MattCheramie/GopherTrunk/issues/1003
---

**TETRA lock facts** are the small set of protocol and DSP truths that separate a
[TETRA](/reference/tetra/) demodulator that locks real air from one that only passes its own
tests. Each was learned during GopherTrunk's TETRA bring-up, and several contradict the
"obvious" reading of the spec.

## Training sequences are counted in bits, not dibits

ETSI EN 300 392-2 §9.4.4.3 defines the [training sequences](/reference/tetra-training-sequences/)
as **NTS1/NTS2 = 22 bits, ETS = 30 bits, STS = 38 bits**. The trap is unit confusion: "38"
is bits (19 [dibits](/reference/dibit/)), not 38 dibits. GopherTrunk's first implementation
carried placeholder constants that declared a 64-bit variable to hold "38 dibits" (76 bits) —
six dibits were silently zero-filled, and the correlator could never fire on real signal
([#553](https://github.com/MattCheramie/GopherTrunk/issues/553)).

The decisive proof that a sync layer works, independent of anything downstream: training
correlation hits at a modal spacing of **exactly 1020 dibits** — one TETRA frame
(4 slots × 255 dibits). Hits at exact frame cadence validate sync even while every CRC still
fails.

## π/4-DQPSK uses the Gray mapping

On air, a [π/4-DQPSK](/reference/pi-4-dqpsk/) symbol pair `(B1, B2)` maps to a dibit via the
[Gray](/reference/gray-code/) convention `(B1<<1) | (B1^B2)`:

| Bits | Dibit |
|---|---|
| 00 | 0 |
| 01 | 1 |
| 11 | 2 |
| 10 | 3 |

Using the *linear* convention (correct for the C4FM family) flips B2 whenever B1 = 1. Sync
still correlates, but descrambling, deinterleaving, and CRC all see corrupted bits — a "locks
but nothing decodes" state ([#553](https://github.com/MattCheramie/GopherTrunk/issues/553)).

## AFC: the 4th-power estimator aliases at ±f_sym/8

Raising the per-symbol differential phase to the 4th power collapses π/4-DQPSK data to a
constant — but because 4·ω wraps at 2π, the offset is only recoverable modulo π/2 per symbol.
At TETRA's 18,000 sym/s that is **±f_sym/8 = ±2250 Hz**. Larger offsets *alias*: the
estimator returns a plausible-looking value that is wrong by a multiple of f_sym/4 = 4500 Hz
(a 2500 Hz offset reads as ≈ −2000 Hz), and lock silently fails
([#940](https://github.com/MattCheramie/GopherTrunk/issues/940)).

The fix is a coarse acquisition stage — the angle of the block's lag-1 autocorrelation, which
has no ±π/4 wrap — that picks the correct 4500 Hz alias bucket before the 4th-power stage
refines. Two subtleties:

- **The coarse stage must read pre-matched-filter samples.** The matched filter is centred at
  0 Hz, so a spectral estimate taken after it is pulled toward zero (measured at roughly half
  the true offset).
- Total range is bounded near ±6 kHz by the 15 kHz channel-filter cutoff.

Also note ordering: [AFC](/reference/automatic-frequency-control/) has to run *before* timing
recovery — an off-centre channel biases every dibit and the training sequence never
correlates in the first place ([#553](https://github.com/MattCheramie/GopherTrunk/issues/553)).

## Colour code 0 is not "no scrambling"

TETRA [scrambling](/reference/tetra-scrambler/) is **non-identity at colour code 0**: the
colour-0 scrambler seeds its LFSR to `0xC0000000` (EN 300 392-2 §8.2.5.2). Two consequences
that each cost real debugging time:

- **Blind identification must recover the colour code from the BSCH.** The BSCH is always
  transmitted colour-0-scrambled and is therefore always decodable; every other channel needs
  the site's real [colour code](/reference/tetra-extended-colour-code/). GopherTrunk's hunt
  scored TETRA candidates with the configured colour (0 during blind identify), so every
  SCH/HD CRC failed and the confidence score forfeited its entire FEC term — recover the
  colour from the BSCH first ([#648](https://github.com/MattCheramie/GopherTrunk/issues/648)).
- **Never skip descrambling "because the colour is 0."** The DMO (direct mode) voice path
  inherited an `if colour != 0` shortcut from the trunked-mode code — safe there only because
  a trunked extended colour code is never 0. On clear colour-0 DMO traffic it left bursts
  scrambled going into the FEC, producing a uniform chance-floor CRC rate that was misread as
  encryption ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)).

A related DMO fact: the DM colour code is *not* recoverable from the SCH/S (which is always
colour-0-scrambled and so always reads 0). When signalling and radio programming disagree
with reality, GopherTrunk recovers the colour empirically — sweep candidate colours and keep
the one that maximizes CRC-valid speech frames; the correct colour wins by a wide margin
([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)).

## Symptom table

| Symptom | Looks like | Actually | Fix / check |
|---|---|---|---|
| Training sequence never correlates on real air | Wrong sample rate, inverted spectrum | Placeholder/miscounted sync constants, or no AFC before timing | Bit lengths 22/30/38; look for hits at 1020-dibit cadence ([#553](https://github.com/MattCheramie/GopherTrunk/issues/553)) |
| Sync locks, every CRC fails | Scrambler or colour bug | Linear instead of Gray dibit mapping | Apply the Gray table ([#553](https://github.com/MattCheramie/GopherTrunk/issues/553)) |
| AFC reports a small, stable offset but never locks | AFC converged, signal weak | 4th-power alias — residual is a multiple of 4500 Hz | Coarse lag-1 autocorrelation stage, pre-matched-filter ([#940](https://github.com/MattCheramie/GopherTrunk/issues/940)) |
| Colour 0, so descrambling skipped | Harmless shortcut | Colour-0 scrambling is non-identity (seed `0xC0000000`) | Descramble unconditionally ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)) |
| Known-clear traffic decodes at the CRC chance floor | Encryption | Wrong colour code for the descramble | Recover colour from the BSCH, or sweep for max CRC yield ([#648](https://github.com/MattCheramie/GopherTrunk/issues/648), [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)) |

The DMR sync layer has an analogous blind spot — its sync words are closed under polarity
flip, so sync alone can't reveal spectrum inversion either; see
[DMR grant-field gotchas](/reference/dmr-grant-field-gotchas/) and the wider catalog in
[signal signatures](/reference/signal-signatures/).

## Provenance

- [#553](https://github.com/MattCheramie/GopherTrunk/issues/553) — TETRA fails to lock: placeholder training sequences, the bit-vs-dibit trap, the Gray mapping, and the frame-cadence proof.
- [#940](https://github.com/MattCheramie/GopherTrunk/issues/940) — AFC coarse-acquisition range: the ±2250 Hz 4th-power alias and the pre-matched-filter rule.
- [#648](https://github.com/MattCheramie/GopherTrunk/issues/648) — hunt misclassification: blind identify at colour 0 forfeits all FEC confidence; recover the colour from the BSCH.
- [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003) — DMO clear voice misread as encrypted: the colour-0 descramble skip and empirical colour recovery.
