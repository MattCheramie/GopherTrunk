---
title: "From the Issue Tracker, Part 1: The First P25 Lock — Eleven Fixes Between 'Trying' and 'Locked'"
description: A P25 control channel that SDRTrunk and OP25 decoded in five seconds took GopherTrunk eleven fixes over five days to lock — and the last two bugs were a wrong BCH generator polynomial and a wrong CRC variant that every synthetic round-trip test had happily agreed with.
category: solution-postmortem
keywords: p25 control channel, cc-hunt, frame sync word, nid bch, bch generator polynomial, crc-ccitt augmented, status symbols, c4fm matched filter, mueller-muller, gardner, iq capture replay, gophertrunk postmortem
tags: [from-the-issue-tracker, p25, dsp, trunking, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 1
---

*Part 1 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. Each part reconstructs one issue thread — the symptom as reported, the theories
that were wrong, the diagnostic that cracked it, and what the fix left behind in the
codebase. We start with the flagship:
[#275](https://github.com/MattCheramie/GopherTrunk/issues/275), the five days in
which GopherTrunk went from never locking a P25 control channel to decoding one —
through eleven distinct fixes, ending at two constants that had been wrong since the
day they were written.*

> **TL;DR:** A reporter's P25 control channel locked in SDRTrunk, OP25, and
> p25-survey within seconds, while GopherTrunk logged `cc-hunt: trying` forever with
> nothing after it. The failure turned out to be eleven bugs deep: an unprogrammed
> sample rate, a missing channelizer, four separate chunk-boundary and timing-loop
> bugs, a receive filter that didn't match the P25 spec, unstripped status symbols, a
> collapsed slicer — and at the very bottom, a wrong BCH(63,16,11) generator
> polynomial and the wrong CRC-CCITT variant. Every single one had been masked by
> encoder/decoder round-trip tests agreeing with each other. Fifteen seconds of real
> IQ from the reporter's antenna was what finally broke the spell.

## The report

The issue arrived as close to ideal as bug reports get. Two NESDR SMArt v5 dongles, a
high-gain Yagi, a busy P25 Phase 2 system (MMR, control channel 420.0875 MHz), a
minimal config — and one observation:

```text
time=2026-05-19T14:59:11.416+10:00 level=INFO msg="cc-hunt: trying" system=MMR freq_hz=420087500
```

Every 10–15 seconds, the same line again. No sync attempts, no framing failures, no
decode errors. Nothing downstream, ever. And the kicker: *the same dongles and
antenna decoded the same system in SDRTrunk.* OP25-based p25-survey locked it within
~5 seconds at BER 0.00%.

The first community theory was PPM drift — reasonable, wrong, and the cheapest of the
wrong theories this thread would collect. The real answer was that almost the entire
receive chain between the USB port and the trunking engine had never met a real
signal.

## The ladder

Eleven fixes, in the order the failures surfaced. Each rung was invisible until the
one above it was fixed.

| Rung | Fix | What was wrong | The tell |
|---|---|---|---|
| 1 | #281 | `SetSampleRate` never called on the production path | silence after `cc-hunt: trying` |
| 2 | #289 | no channelizer — decoder fed the full 2.048 MHz swath (~427 samples/symbol) | `no FSW hits in chunk` on both demods |
| 3 | #292 | frames couldn't cross IQ-chunk boundaries (~19 symbols per USB transfer, NID needs 32) | FSW hits silently discarded, `dibits=19/20` |
| 4 | #300 | Gardner timing loop emitted ~1 surplus symbol per call (~5% inflation) | CQPSK reached NID but never *held* sync |
| 5 | #303 | no coarse AFC — tuner offset became DC bias on the discriminator | PPM sweep changed nothing |
| 6 | #307 | CMA + Gardner error terms scale with amplitude² | lock only in a narrow gain window (197 broken, 49 partial, 28 broken) |
| 7 | #311 | Mueller-Müller dropped one sample per chunk | dibit error rate 0.66 chunked vs 0.04 one-shot |
| 8 | #316 | receive filter modeled as RRC; the P25 spec says sinc | ~5.75% residual ISI, `errs=11` ceiling |
| 9 | #318 | status symbols never stripped from the NID | ~21 of 32 NID dibits misaligned, both demods identical |
| 10 | #335 | slicer collapse — matched-filter DC gain put every sample past the outer threshold | dibit histogram 50/0/50/0 |
| 11 | #337 + #338 | wrong BCH(63,16,11) generator polynomial; wrong CRC-CCITT variant | NID byte-identical across frames, yet "not a codeword" |

## Rung 1: the radio was never told its sample rate

The pool programmed PPM, gain, and bias-tee on every device — but never the
resampler. The chip stayed at whatever divisor it powered up with while every decoder
did symbol-timing math against the configured 2.048 MS/s. SDRTrunk works on the same
hardware because librtlsdr's `rtlsdr_open()` ends with a default
`rtlsdr_set_sample_rate` call; GopherTrunk's pure-Go driver had no such safety net.

The fix (#281) also closed the diagnostic black hole: the P25 `Process` paths now
throttle-log `no FSW hits in chunk` instead of staying silent, and a new
`gophertrunk_sdr_iq_power_dbfs` gauge (#282) let the reporter confirm healthy IQ
(~−18 dBFS) before the decoder was even involved. From here on, every failure at
least left a fingerprint.

## Rungs 2–3: channelize, then let frames cross chunks

With the rate fixed, the decoder was still being fed the full un-channelized 2.048
MHz swath — roughly 427 samples per symbol against a ±1 MHz window. The symbol clock
produced dibits at about the right *rate*, but the values were noise. A digital
down-converter (#289) now decimates every per-protocol pipeline to the narrowband
rate its matched filter expects (~48 kHz for the 4800-baud C4FM family). A negative
result from that PR is worth keeping: an IQ-domain DC blocker was rejected, because
C4FM carries real energy at 0 Hz — a complex DC block measured over 60% RMS error on
a round-tripped stream.

Then the shape of real USB delivery bit. An RTL-SDR hands over IQ in 16 KiB
transfers — about **19 P25 symbols per decoder call**. The control-channel state
machine discarded every frame-sync hit unless the entire 154-dibit frame (FSW +
32-dibit NID + 98-dibit TSBK) landed inside a *single* call. Nineteen is less than
even the NID, so every hit was dropped. The synthetic tests fed ~819 dibits per
chunk and never saw it. #292 made frames accumulate across calls, and the regression
test now pumps real 8192-sample chunks.

## Rungs 4–7: the same bug class, one stage at a time

The chunk-boundary theme repeated up the DSP chain like a drumbeat:

- **Gardner** (#300): the CQPSK timing loop re-walked its cross-call look-back buffer
  and emitted roughly one surplus symbol per call — a ~5% inflation that kept
  desynchronizing the dibit stream, so the decoder could reach NID parsing but never
  hold it.
- **Mueller-Müller** (#311): the C4FM clock started its walk at `src[1]` on every
  call, so `src[0]` never advanced the loop — one dropped sample per chunk. Fed the
  whole signal in one block, the dibit error rate was 0.04; fed RTL-realistic
  ~19-symbol chunks, **0.66** — near-random. The fix carries look-back so the
  chunked stream is byte-identical to the one-shot stream, and it repaired the same
  latent bug in the DMR, NXDN, YSF, and dPMR receivers for free.
- **Coarse AFC** (#303): a residual tuner offset becomes a DC bias on the FM
  discriminator, and the fixed 4-level slicer thresholds have no defense — at ≥500 Hz
  the FSW stops correlating entirely. This is why the reporter's ±2 ppm sweep never
  changed anything: once the offset exceeds the slicer margin, no small correction
  recenters it.
- **Scale invariance** (#307): after a CMA equalizer landed on the CQPSK path (more
  on that below), lock suddenly depended on RTL gain — 197 broken, 49 partial
  framing, 28 broken. Both the Gardner and CMA update terms scale with amplitude², so
  the loops only converged when the front-end gain happened to land in a narrow band.
  A feed-forward AGC on the matched-filter output restored scale invariance, guarded
  by a regression test across a 0.05–20× amplitude range.

Rungs 4–7 also mark where the process changed. Every earlier fix had been verified
against *ideal* synthesized IQ in large blocks. #299 added a harness that drives the
real receiver chain against deliberately impaired IQ — carrier offset, DC spike, IQ
imbalance, AWGN — in RTL-realistic small chunks. The Gardner bug fell out of that
harness on day one.

## Rungs 8–9: the spec the code had never read

With timing solid, the C4FM path pegged at a new wall: `nid corrected errs=11` —
exactly the BCH(63,16,11) correction ceiling — with clearly bogus NACs (`0xFF5`)
surfacing as miscorrections.

**The filter was wrong** (#316). GopherTrunk modeled P25 C4FM as a root-raised-cosine
matched pair. RRC × RRC is raised-cosine and ISI-free — so the chain was perfectly
self-consistent in tests — but per TIA-102.BAAA (cross-checked against OP25's
`c4fm_const.py`), a real P25 transmitter shapes with raised-cosine α=0.2 cascaded
with an inverse-sinc, and the receiver's job is a plain **sinc** that cancels it.
An RRC receive filter on a spec signal leaves about 5.75% residual ISI — quietly
corrupting dibits at exactly the level that keeps BCH at its ceiling.

**The status symbols were never stripped** (#318). P25 interleaves a 2-bit status
symbol into the on-air stream every 70 information bits, through every data unit
including TSDUs. The 24-dibit FSW is contiguous — the first status symbol lands at
dibit 35, just after it — which is precisely why FSW detection worked while the NID
behind it was garbage: the decoder read the NID as 32 contiguous dibits, swallowed a
status symbol as data, and misaligned ~21 of the 32. The give-away was that C4FM and
CQPSK failed *identically* — the bug lived downstream of both demodulators, in the
one stage they shared.

## The wrong turns, named honestly

- **CQPSK/LSM was pushed twice** for a site that SDRTrunk labels C4FM. The theory —
  simulcast sites transmit LSM, so add a linear demod path (#288) and then a CMA
  equalizer (#306) — was plausible and the code was eventually valuable, but the
  reporter's site was a strong standalone C4FM transmitter. The equalizer's first
  contribution was the gain-window regression of rung 6.
- **`delta=2, rot=3` looked like a clue and was two artifacts.** When a bounded
  NID-alignment search (#320) went in, the diagnostics converged on
  `closest marginal errs=10 at delta=2 strip=true rot=3` every frame. `delta=2` was
  the edge of the ±2 search grid — the textbook signature of a bounded search pegged
  at its boundary — and `rot=3` is non-physical on an FM-discriminator stream, where
  only rotations 0 and 2 correspond to real signal symmetries. Both "signals" were
  the search machinery talking to itself; #326 widened the span to ±6 and restricted
  C4FM rotations to {0, 2}, and the phantom convergence vanished.
- **Two retest cycles were wasted on stale builds.** One retest ran a binary eight
  commits older than the fix it was testing; another carried a `-dirty` suffix that
  blocked fast-forward pulls. The countermeasures outlived the bug: the pipeline
  startup line now advertises `demod / rotations / nid_search_span / build=`, and
  `internal/version` falls back to `runtime/debug.ReadBuildInfo()` so even a bare
  `go build` produces a self-identifying binary.
- **The first capture was mislabeled.** A shared IQ file's symbol clock measured ~160
  samples per symbol — inconsistent with the stated 2.048 MS/s — and decoded to
  nothing through every demodulator at every plausible rate. The lesson became a
  feature: the `replay` tool now reports effective baud at EOF, so a mislabeled file
  diagnoses itself.

## The capture that ended the guessing

The turning point of the whole thread was 30 seconds of raw IQ captured with a known
command line: `mt-anakie-420087500-960k-g49.iq`, replayed offline through the new
`gophertrunk replay` subcommand. The failure reproduced identically off-air:

```text
no NID corroborated over 28 guesses; closest marginal errs=11 at delta=6 strip=true rot=0,
TSBK uncorroborated — best alignment at search boundary (±6); true offset may exceed span;
err_pattern=00100001102010200100000000100100
```

From here every hypothesis became a measurement instead of a field trip:

- **Alignment was ruled out by bisection** (#334): widening the search to ±36 still
  produced 253 NID-BCH failures with closest-miss errors flat at 9–11 across the
  whole range. Whatever was wrong, it wasn't framing.
- **The slicer had collapsed** (#335): a new `replay -diag` histogram showed the
  dibit distribution was **50/0/50/0** — only outer symbols, ever. The C4FM matched
  filter is normalized to a DC gain of one symbol period, which on real captures put
  every sample past the slicer's outer threshold. The FSW still correlated because it
  uses only outer symbols — which is exactly why the failure hid *behind* a working
  sync detector. A symbol-domain AGC between clock recovery and the slicer opened the
  eye: histogram 28/22/27/23, and suddenly ~200 perfect distance-0 FSW hits at exact
  360-dibit intervals.
- **The NID was constant, and that was the smoking gun** (#336): across all 197
  frames in the capture, the 32 NID dibits were *byte-identical* while the TSBK
  payload varied frame to frame. That's a real control channel — same NAC and DUID
  every frame, different payload — being decoded consistently and rejected
  consistently. The first 16 info bits even read out plausibly: NAC 0x164, DUID 7
  (TSDU). The remaining failure had to be a systematic transform — not noise.

## The bottom of the ladder: two constants

**Bug 1 — the BCH(63,16,11) generator polynomial was wrong** (#337). The constant in
`internal/radio/framing/bch.go` was off by ten exponents from the TIA-102.BAAA
Annex A generator. The correct value — derived from first principles by multiplying
the binary minimal polynomials of α, α³, α⁵, …, α²¹ over GF(2⁶) with primitive
polynomial x⁶ + x + 1 — is `0xCD930BDD3B2B`. Cross-verified by running OP25's
syndrome decoder against the captured Mt Anakie NID: 0 of 22 syndromes non-zero. The
on-air NID had been a valid spec codeword all along; GopherTrunk was checking it
against a code that exists nowhere but its own tests.

**Bug 1b — the 64th NID bit is not a parity bit.** The code computed the trailing bit
as even parity over the 63-bit codeword. Per spec (and OP25's `p25_framer`), it's a
fixed per-DUID flag: 0 for HDU/TDU/TSDU/PDU/TDULC, 1 for LDU1/LDU2. For a TSDU the
on-air bit is 0 — which the "even parity" computation produced as 1, flagging every
otherwise-clean decode as a parity failure.

**Bug 2 — the wrong CRC-CCITT variant** (#338). With the BCH fixed, the trellis
decoder reported `metric=0` — zero bit errors in the 196-bit channel block — and 195
of 197 TSBKs *still* failed CRC. Trellis right, verifier wrong: P25 uses the
"augmented codeword" CRC-CCITT for the TSBK trailer (init=0, MSB-first, final XOR
0xFFFF, evaluated over all 12 bytes, expecting 0), while the code used CRC-CCITT/FALSE.
Same 0x1021 polynomial, different answer on the same bytes. `metric=0` plus a CRC
failure is now a recognized signature: the FEC says the bits are perfect, so suspect
the checker.

The capture's full arc:

| | NID failures | TSBK CRC failures | TSBKs decoded | Grants |
|---|---:|---:|---:|---:|
| Pre-#335 (slicer collapse) | 197/197 | — | 0 | 0 |
| Post-#335 (symbol AGC) | 197/197 | — | 0 | 0 |
| Post-#337 (BCH polynomial + parity) | **2/197** | 195/197 | 0 | 0 |
| Post-#338 (augmented CRC) | **2/197** | **0/197** | **92** | **2** |

And on air, the line the thread had been chasing for five days:

```text
time=2026-05-24T07:16:10.555+10:00 level=INFO msg="control channel locked" nac=356 freq=420087500 rot=0 delta=0
time=2026-05-24T07:16:10.555+10:00 level=INFO msg="cc-hunt: locked" system=MMR freq_hz=420087500 nac=356
```

Followed within seconds by decoded TSBKs, Motorola vendor messages, and live patch
activity. Both constants are pinned by regression tests built from the capture's
actual on-air bytes (`TestEncodeNIDBitsMtAnakieVector`,
`TestTSBKAcceptsMtAnakieOnAirVector`), so a future regression names the byte that
changed.

## The meta-lesson

The maintainer's closing note on the thread deserves to be quoted rather than
paraphrased:

> Without 15 seconds of real ground-truth bits, every one of these bugs would have
> stayed silently masked by the encoder + decoder agreeing with each other inside
> synthetic round-trip tests.

Every rung of the ladder shares that anatomy. The wrong sample rate passed tests
that never touched hardware. The chunk-boundary bugs passed tests that fed
generously sized chunks. The RRC filter passed because the test modulator was also
RRC. The status symbols passed because the test frames were built without them. And
the wrong BCH polynomial and wrong CRC variant passed *hundreds* of round trips,
because an encoder and decoder that share the same wrong constant agree with each
other perfectly. A self-consistent system can only be falsified from outside — and
the outside, for a radio decoder, is 15 seconds of IQ from somebody's antenna.

## What we keep

- **Round-trip tests can't catch a shared wrong constant.** Validate codecs against
  external ground truth — a reference implementation, a captured on-air vector, or
  both. The spec constants this hunt corrected are recorded in the
  [P25 on-air constants]({{ '/reference/p25-onair-constants/' | relative_url }})
  Field Guide entry.
- **A capture plus offline replay beats any number of on-air retests.** The
  `gophertrunk replay` subcommand — with `-diag` histograms, effective-baud
  reporting, and per-failure `err_pattern` strings — turned each hypothesis into a
  measurement. The workflow is written up in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Learn the signatures.** A bounded search pegged at its boundary (`delta=2` on a
  ±2 grid), a non-physical rotation "winning" (rot=3 on an FM discriminator), a
  dibit histogram of 50/0/50/0, `metric=0` with a failing CRC — each of these names
  its own cause, and they're cataloged in
  [signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).
- **Make staleness self-evident.** Two retest cycles died on old binaries. Every
  pipeline now logs its decode parameters and build stamp on startup, so the first
  line of any pasted log answers "which code was this?"
- **A silent failure is a bug in its own right.** The original report's defining
  feature was *no logs at all*. Every stage that can produce nothing now says so, at
  a throttled cadence.

## Series navigation

**Part 1** · Next →
[Part 2: The Talker-Alias Hunt]({{ '/blog/solution-postmortem/from-the-issue-tracker-02-talker-alias-hunt/' | relative_url }})
