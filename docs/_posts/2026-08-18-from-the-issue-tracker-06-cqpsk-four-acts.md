---
title: "From the Issue Tracker, Part 6: CQPSK in Four Acts — Fixing the Linear Path One Layer at a Time"
description: A P25 control channel that decoded perfectly in C4FM produced almost nothing in CQPSK mode. The fix took four rounds — a missing carrier-recovery stage, a multipath-poisoned frequency seed, a fractionally-spaced equalizer, and a BCH hot path that only appeared once locking finally worked — and the most convincing piece of evidence in the thread was a broken instrument.
category: solution-postmortem
keywords: p25 cqpsk, lsm demodulation, carrier recovery, costas loop, kay estimator, simulcast multipath, fractionally spaced equalizer, cma, bch decoder, nid, gophertrunk debugging
tags: [from-the-issue-tracker, p25, dsp, cqpsk, equalization, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 6
---

*Part 6 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that
fought back. [Part 5]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
ended with a rule: when a decode fails, prove which layer failed before you fix
one. This part is that rule applied four times in a row to the same issue —
[#492](https://github.com/MattCheramie/GopherTrunk/issues/492), the P25 Phase 1
CQPSK path — where each fix was real, each fix was necessary, and each fix
uncovered the next problem underneath it.*

> **TL;DR:** GopherTrunk's C4FM demodulator decoded a P25 control channel at
> 100% while the CQPSK path on the same capture managed 74% TSBK CRC and near-zero
> live lock. The reporter's meticulously argued "symbol clock timing slip" theory
> rested on a diagnostic line that was printing the *wrong demodulator's* state —
> a broken instrument, not a broken loop. The real story took four acts: the
> CQPSK path had **no carrier-frequency recovery at all**; the new carrier seed
> was then **poisoned by simulcast multipath**, railing the Costas loop; a
> **T/2 fractionally-spaced equalizer** was needed before real C4FM-shaped
> signals could open the constellation on the linear path; and once CQPSK finally
> locked, a **brute-force BCH decoder** that had never run at scale before
> surfaced as multi-second stalls. Locks on the reporter's eight real captures
> went 0/8 → 3/8 → 8/8.

## Cheat sheet

| Fact | Detail |
|---|---|
| Issue | [#492](https://github.com/MattCheramie/GopherTrunk/issues/492) |
| Symptom | Same capture: C4FM 100% TSBK; CQPSK 74% CRC, ~0% live lock — and replay ran 8× slower |
| Broken instrument | `mm_sps=0.00` — a C4FM gauge read on the CQPSK path, which uses a Gardner loop, not Mueller-Müller |
| Act one (#497) | The CQPSK path had no carrier-frequency recovery at all — coarse seed + NCO + Costas loop added |
| Act two (#529) | Simulcast multipath poisons the autocorrelation seed — modulus-CV gate + Costas anti-windup |
| Act three (#532) | T/2 fractionally-spaced equalizer replaces the symbol-spaced CMA |
| Act four | Brute-force BCH decoder (104 decodes per FSW hit) surfaced as 10–15 s stalls — 26 ns clean fast path |
| Score | Locks on eight real captures: 0/8 → 3/8 → 8/8 |

## In this post

- **The symptom** — one path perfect, the other losing a quarter of its TSBKs.
- **The smoking gun that was a broken instrument** — `mm_sps=0.00` measured the wrong demodulator.
- **Act one: the missing carrier recovery** — differential decoding forgives phase, never frequency.
- **Act two: multipath poisons the seed** — gating an estimator that can't be hardened.
- **Act three: the fractionally-spaced equalizer** — sub-symbol problems need sub-symbol taps.
- **Act four: the hot path that locking uncovered** — the dormant O(65,536) decoder.
- **What we keep** — the durable rules and their Field Guide entries.

## The symptom

The report was unusually well instrumented from the first message. The reporter
recorded 20 seconds of IQ from a live P25 Phase 1 system and replayed the same
file through both demodulators:

```
C4FM  (replay took 15.3 s):
replay: nid   trusted=390  marginal=0  uncorrectable=0  (100.0% ok)
replay: tsbk  decoded=417  trellis_failed=0  crc_failed=0 (100.0% ok)

CQPSK (replay took 127.3 s):
replay: nid   trusted=96   marginal=4  uncorrectable=6   (94.3% ok)
replay: tsbk  decoded=87   trellis_failed=0  crc_failed=30 (74.4% ok)
```

Same file, same samples: one path perfect, the other losing a quarter of its
TSBKs — and taking eight times longer to do it. Live, CQPSK was worse still:
roughly 0% control-channel decode, and it rarely achieved sync at all. File that
127-second replay time away; it comes back in act four.

## The smoking gun that was a broken instrument

The reporter then ran a 100-second stress pass over 200 million samples and
posted what looked like a definitive diagnosis: under CQPSK, the effective baud
measured 4796.2 against a true 4800 — exactly 383 symbols dropped over the run —
and the timing-recovery diagnostics were flat-lined:

```
mm_sps=0.00
```

The conclusion drawn from it was confident and specific: the Mueller-Müller
symbol-clock tracking loop was never activated on the CQPSK path, timing error
accumulated linearly, and the decision boundaries walked off the symbols.
"Definitive proof of symbol clock timing slip."

It was a beautifully argued theory about a number that meant nothing. The
`-diag` line was printing **C4FM state accessors** — Mueller-Müller residuals,
AFC state, slicer state — and every one of them legitimately returns zero on
the CQPSK path, because CQPSK doesn't *use* Mueller-Müller timing. It uses a
[Gardner loop]({{ '/reference/gardner-timing-recovery/' | relative_url }}),
whose state the diagnostic never displayed. `mm_sps=0.00` wasn't a stalled
loop; it was a gauge wired to the wrong engine. An earlier Gardner-gain tweak
motivated by this theory had, predictably, changed nothing.

The first fix in the series was therefore to the *instrument*: the replay
`-diag` line became demod-aware, printing `carrier_hz_est / gardner_mu /
gardner_sps / agc_gain / cma_err` on the CQPSK path. Every subsequent act of
this story was diagnosed off that corrected readout. When a debugging session
stalls, auditing what the diagnostics actually measure is not procrastination —
here it was the single highest-leverage change in the thread.

## Act one: the missing carrier recovery

With honest instrumentation, the real first-order problem was visible in the
dibit statistics: the four-level symbol histogram was badly skewed (dibits 0
and 2 over-represented, 1 and 3 starved), and the frame sync word never
correlated. The cause was structural: **the CQPSK path had no carrier-frequency
recovery whatsoever.** The `CoarseAFC` stage existed only on the C4FM branch.

A π/4-DQPSK differential decoder is famously tolerant of carrier *phase* — a
constant phase offset cancels in `s·conj(prev)`. What it cannot cancel is a
constant frequency offset, which shows up as a fixed **per-symbol rotation** of
`2π·Δf/baud`. That rotation spins the whole differential constellation, so
every decision lands in the wrong quadrant. And P25's 4800 baud makes it
acutely sensitive: the same tuner offset produces ~3.75× more rotation per
symbol than TETRA's 18,000-baud π/4-DQPSK — which is exactly why GopherTrunk's
TETRA path had gotten away without carrier recovery for so long.

[#497](https://github.com/MattCheramie/GopherTrunk/pull/497) added the missing
stage: a coarse carrier estimate from the lag-1 autocorrelation of the raw IQ
seeds an NCO that de-rotates before the matched filter, and a new decision-free
QPSK [Costas loop]({{ '/reference/costas-loop/' | relative_url }}) tracks the
residual and drift. Synthetic sweeps of ±200 to ±2500 Hz injected offset went
from decoding nothing to recovering the FSW throughout.

On the reporter's real captures, it didn't work.

## Act two: multipath poisons the seed

The new telemetry showed something strange: on capture after capture,
`carrier_hz_est` started plausibly and then *wound away* — 55 Hz, 213, 435,
550, then oscillating around ±600 Hz, which is precisely the Costas loop's
±baud/8 clamp. The loop wasn't failing to track an offset. It was being
actively steered to the rail and pinned there.

The culprit was the seed. These were simulcast-flavored captures, and
[simulcast]({{ '/reference/simulcast/' | relative_url }}) multipath carves deep
spectral nulls into the channel. A null shifts the *centroid* of the lag-1
(Kay) autocorrelation, and the coarse estimator faithfully reads that shifted
centroid as a spurious ~650–750 Hz carrier offset. It then mis-tunes the NCO by
that amount, and the Costas integrator — trying to pull back against a seed
that large — winds up at its clamp and stays railed. The uncomfortable truth:
**no autocorrelation-based estimator can distinguish a multipath-shifted
centroid from a real offset.** Hardening the estimate was not enough; it had to
be gated.

The gate that shipped in
[#529](https://github.com/MattCheramie/GopherTrunk/pull/529) exploits the one
property that separates the two cases. π/4-DQPSK is constant-modulus, and a
pure carrier offset only *rotates* it — the symbol modulus stays tight. Multipath
ISI *blurs* the modulus. So: de-rotate by the candidate estimate, run the
matched filter, and measure the **coefficient of variation of the symbol
modulus**. Low CV means the offset is real — trust the seed. High CV means the
"offset" is multipath bias — leave the NCO at identity and let the CMA-then-Costas
chain acquire the true (in-range) offset on its own. A multi-lag phase-ramp fit
sharpened trusted estimates, and Costas anti-windup meant a loop that did get
pinned could re-acquire instead of staying railed forever.

On a clean synthetic LSM capture with a 100 Hz offset, `carrier_hz` now
converged to 99.99 Hz instead of railing to ~700. On the reporter's eight real
captures (`drift1`–`drift8`), the rail was gone on all eight — the estimate
settled between −52 and +139 Hz, tracking the actual drift. And CQPSK locks
went from **0/8 to 3/8**, with two of the three reaching TSBK parity with
C4FM.

Three of eight is not a fixed demodulator.

## Act three: the fractionally-spaced equalizer

The five captures that still refused to lock shared a signature: the dibit
histogram collapsed toward the inner symbols, with the outer ±1800 Hz rails
consistently arriving under-shot. That is not noise and not carrier offset.
It's a pulse-shape problem.

Here is the deepest insight of the whole issue. These captures are real P25
**C4FM** — a continuous-phase FM waveform. The CQPSK/linear path
matched-filters its input with a fixed root-raised-cosine filter, because
that's the right receive filter for a linearly modulated LSM transmitter. But
the C4FM transmit pulse is *not* RRC. Filtering a C4FM waveform with an RRC
"matched" filter systematically under-shoots the outer deviation rails, the
constellation closes, and the carrier loop has nothing clean to lock to.
Crucially, the existing [CMA]({{ '/reference/cma-equalizer/' | relative_url }})
equalizer couldn't repair this, for a reason worth engraving somewhere: it was
**symbol-spaced** — one sample per symbol — and a pulse-shape mismatch is a
*sub-symbol* phenomenon. An equalizer that only ever sees one sample per symbol
cannot synthesize the missing fractional-delay structure, no matter how long
you let it adapt.

[#532](https://github.com/MattCheramie/GopherTrunk/pull/532) replaced it with a
**T/2 fractionally-spaced blind equalizer (FSE)**, fed two samples per symbol —
the Gardner loop's on-time and mid-symbol interpolants, via a new
`Gardner.Process2x` that emits both on the same timing loop and symbol cadence.
A fractionally-spaced equalizer synthesizes the receive matched filter
*implicitly*, so one structure opens both simulcast multipath ISI and the
C4FM-versus-RRC pulse mismatch, and it's insensitive to residual timing phase
into the bargain. It stays blind (CMA, rotation-invariant) ahead of the Costas
loop, keeps the center-tap phase pin, and adds a little tap leakage so the
larger fractionally-spaced null space can't wander on clean input.

The scoreboard, on the same eight captures:

| CQPSK locks | pre-fix | after carrier recovery (#529) | after T/2 FSE (#532) |
|---|---|---|---|
| | 0/8 | 3/8 | **8/8** |

| capture | CQPSK TSBK | C4FM TSBK |
|---|---|---|
| drift1 | 21 | 24 |
| drift2 | 23 | 28 |
| drift3 | 27 | 30 |
| drift4 | **28** | 28 |
| drift5 | 16 | 24 |
| drift6 | **28** | 28 |
| drift7 | 11 | 21 |
| drift8 | **17** | 16 |

Two captures at exact C4FM parity, one slightly above it, the rest decoding
usefully. One validation caveat became a permanent lesson: a purely synthetic
regression test for this fix **was not viable**. Synthetic C4FM generated at
the 48 kHz channel rate collapses ~94% of its samples onto a single point — the
signal is already constant-modulus, a degenerate constellation no CMA can open.
Only the real captures, roughed up by a real front end and the DDC, exercise
what the equalizer actually does. Some fixes can only be proven against air.

## Act four: the hot path that locking uncovered

The reporter came back with the best kind of bad news: "Finally, we got live
decoding! Both C4FM and CQPSK works." — followed by a new symptom. The whole
application now stalled for 10–15 seconds at a time, timestamps freezing
"when it's trying to correct crc for some NID."

That description pointed almost exactly at the culprit, and it wasn't new
code. `BCHDecode63_16` decoded the P25 NID's
[BCH(63,16,11)]({{ '/reference/bch-code/' | relative_url }}) codeword by
**brute-force minimum-distance search**: re-encode all 65,536 candidate
codewords, keep the closest, short-circuit only on an exact match — roughly a
millisecond per call. Worse, the NID search runs that decoder across its whole
alignment grid — ±6 dibits, strip and no-strip, every dibit rotation — which on
the CQPSK path means **104 BCH decodes per FSW hit** (CQPSK checks 4 rotations
where C4FM checks 2). And during warm-up or noisy stretches, bursts of *false*
FSW hits each triggered the full 65,536-scan worst case, all on the single
goroutine that holds the pipeline lock.

Before act three, CQPSK never locked, so this code path effectively never ran.
The moment the equalizer made locking routine, a dormant O(65,536) decoder
started executing a hundred times per frame. That's also the retroactive
explanation for the very first message's oddity: the 20-second capture that
took 127 seconds to replay. Performance bugs hide behind correctness bugs, and
fixing the second is what schedules the first.

The fix kept the decode **bit-for-bit identical** — none of the hard-won NID
behavior changed — and attacked only the cost:

1. **Clean-codeword fast path.** The code is systematic, so a zero-error NID
   decodes to its own info bits; re-encode and compare. The dominant case on a
   locked signal became O(1).
2. **Precompute the 65,536-codeword table once**, so the fallback search
   popcounts instead of re-encoding every candidate.
3. **Unique-decoding early return.** BCH(63,16) has minimum distance 23, so at
   most one codeword lies within the t=11 correction radius — the first one
   found inside it is provably *the* answer, identical to a full scan.

| case | before | after |
|---|---|---|
| clean codeword (locked-signal common case) | ~1 ms | **26 ns** |
| correctable (errors within t=11) | ~1 ms | ~85 µs |
| uncorrectable (false FSW hit) | ~1 ms | ~128 µs |

An equivalence test against a brute-force reference oracle pinned the
"identical outcomes" claim. The reporter re-ran the 100-second captures:
significantly faster, stalls gone, decode statistics unchanged. Four acts,
curtain.

## What we keep

- **Audit the instrument before you trust the smoking gun.** The most
  convincing evidence in this thread — `mm_sps=0.00` — was a C4FM gauge read on
  a CQPSK engine. The [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
  starts with verifying that a metric measures what its name implies *on the
  path you're debugging*.
- **Differential decoding forgives phase, never frequency.** A residual tuner
  offset becomes a per-symbol rotation of `2π·Δf/baud`, and the lower the baud,
  the harder it bites. Any π/4-DQPSK path without carrier recovery is a latent
  #492. See [signal signatures]({{ '/reference/signal-signatures/' | relative_url }})
  for the skewed-dibit-histogram tell.
- **Blind estimators need gates, not just hardening.** No autocorrelation
  estimator can tell a multipath-shifted centroid from a real offset — the
  discriminating physics (constant modulus rotates cleanly; ISI blurs it) has
  to be tested explicitly.
- **Symbol-spaced equalizers cannot fix sub-symbol problems.** A pulse-shape
  mismatch lives between the samples a T-spaced CMA ever sees; a T/2
  fractionally-spaced equalizer synthesizes the receive matched filter
  implicitly and fixes multipath and pulse mismatch with one structure.
- **Fixing a lock exposes everything downstream of the lock.** Budget for the
  performance profile of code that "never runs" — it's one bug fix away from
  running 104 times per frame.
- **Some regressions can only be pinned by real captures.** Synthetic C4FM at
  channel rate is constant-modulus by construction — a self-consistent test
  that can't fail. When and why to force each demod mode is covered in
  [P25 demod mode selection]({{ '/reference/p25-demod-mode-selection/' | relative_url }}).

## FAQ

**Why is CQPSK so much more sensitive to carrier offset than TETRA's π/4-DQPSK?**
Both cancel constant phase in the differential decode, but a frequency offset
survives as a per-symbol rotation of `2π·Δf/baud` — and P25 runs 4800 baud
against TETRA's 18,000. The same tuner offset rotates each P25 symbol ~3.75×
further, which is exactly why the TETRA path had gotten away without carrier
recovery and the CQPSK path could not.

**Why couldn't the existing CMA equalizer open the failing captures?**
It was symbol-spaced — one sample per symbol — and the C4FM-versus-RRC pulse
mismatch is a *sub-symbol* phenomenon. The fractional-delay structure an
equalizer needs to synthesize lives between the samples a T-spaced CMA ever
sees; no amount of adaptation reaches it. The T/2 fractionally-spaced equalizer
sees two samples per symbol and synthesizes the receive matched filter
implicitly.

**Was the 127-second replay time from the first message ever explained?**
Yes — retroactively, in act four. The brute-force BCH decoder cost ~1 ms per
call and the CQPSK NID search ran it up to 104 times per FSW hit (4 rotations
versus C4FM's 2), with false hits during noisy stretches each triggering the
full 65,536-candidate worst case. The first message's oddity was the fourth
act's evidence, filed 127 seconds at a time.

**If C4FM decodes these captures at 100%, why keep the CQPSK path at all?**
Because some transmitters really are linear (LSM), and the linear path is the
right receiver for them — once it works. But *which* sites should use it turned
out to be its own bug: the project's guidance said "simulcast means CQPSK," and
that was wrong. That story is the next part in this series.

**Why couldn't a synthetic regression test pin the equalizer fix?**
Synthetic C4FM generated at the 48 kHz channel rate collapses ~94% of its
samples onto one point — already constant-modulus, a degenerate constellation no
CMA can open or fail on. Only real captures, roughed up by a real front end and
the DDC, exercise what the equalizer actually does. Some fixes can only be
proven against air.

## Series navigation

**Part 6 of 22** · ←
[Part 5: Ten Megasamples — When the Bug Is in the Samples Themselves]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
· Next →
[Part 7: The LSM Myth — When Your Own Docs Are the Bug]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})
