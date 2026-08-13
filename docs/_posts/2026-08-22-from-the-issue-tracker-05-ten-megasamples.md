---
title: "From the Issue Tracker, Part 5: Ten Megasamples — When the Bug Is in the Samples Themselves"
description: Raising an Airspy R2 from 2.5 to 10 MS/s made every wideband tap go dark, including the strong one — two real bugs got fixed, an alarming AGC number turned out to be normal, and an independent-resampler A/B proved the remaining 10 dB deficit was recorded into the capture itself.
category: solution-postmortem
keywords: airspy r2, sample rate, wideband ddc, channelizer, agc red herring, phase noise, reciprocal mixing, evm, independent resampler, offline replay, gophertrunk postmortem
tags: [from-the-issue-tracker, airspy, dsp, wideband, p25, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 5
---

*Part 5 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 4]({{ '/blog/solution-postmortem/from-the-issue-tracker-04-blog-v4-register-table/' | relative_url }})
ended with a one-line hardware fix. This one ends somewhere stranger: after two
real bugs were found and fixed, the remaining symptom turned out not to be in the
code at all. It was recorded into the capture files themselves — and proving that
required decimating a capture with a resampler that wasn't ours.*

> **TL;DR:** With four P25 control-channel taps on one Airspy R2, 2.5 MS/s could
> only "see" the tap nearest center — but raising the rate to 10 MS/s made **all
> four taps go dark, including the previously strong one**
> ([#764](https://github.com/MattCheramie/GopherTrunk/issues/764)). Two genuine
> rate-dependent bugs were fixed (per-tap CPU blowup and a hardcoded channelizer
> bin count), yet the symptom survived into pure offline replay
> ([#771](https://github.com/MattCheramie/GopherTrunk/issues/771)). The reported
> "AGC stuck 10× above target" was a red herring — the working capture shows the
> identical value. The decisive experiment: decimate the 10 MS/s capture 4:1 with
> an *independent* resampler and feed the proven 2.5 MS/s path — the ~10 dB
> in-channel deficit survived, so it was baked into the samples. Carrier-clean but
> modulation-degraded, independent of gain: the signature of sampling-clock phase
> noise at the Airspy's native 10 MS/s rate. Not our DSP. Also fixed along the
> way: this project's issue-closing discipline.

## The report

The setup: an Airspy R2 in wideband role at 420.9 MHz center, four MMR
control-channel taps spread from −887.5 kHz to +937.5 kHz. At
`sample_rate: 2500000`, only Mt Anakie (−812.5 kHz, closest to usable passband)
decoded; the other three sat at −82 to −86 dBFS with zero frame-sync hits. The
reporter had already done the elimination themselves: `airspy_rx` tuned directly
at a failing frequency showed −5.3 dBFS of strong signal (RF is present), clip
ratio was zero (no overload), and a `capture` + `spectrum` snapshot at the
daemon's own settings confirmed only one carrier visible at 2.5 MS/s but **19
carriers at 10 MS/s** — including all four taps. The band was simply wider than
2.5 MS/s could deliver.

So: raise the rate. And here the report earns its keep, because the result was
the *opposite* of a partial improvement:

> Running the live daemon at `sample_rate: 10000000` broke decode for **all four
> sites, including Mt Anakie**, which decodes cleanly at 2.5 MS/s. Reverting
> immediately restored it.

A change that kills the previously strong tap is a different beast from one that
fails to help the weak ones. It smells like a systemic, rate-dependent defect —
and it was. Two of them.

## Two real bugs

**Per-tap CPU.** Each tap mixed and ran a single-stage rational resampler at the
*full* input rate — a 208:1 decimation per tap at 10 MS/s — all inline on **one
goroutine**. At 10 MS/s × 4 taps the engine could no longer keep real time, the
USB ring overran, and dropped samples broke symbol sync on *every* tap. That is
why the strong site died with the weak ones: symbol timing doesn't care about
SNR when the sample stream has holes in it. The fix runs one **shared decimation
stage** down to ~2.5 MS/s before the per-tap mixers, pinning per-tap cost to the
proven regime regardless of dongle rate.

**Channelizer bins.** Separately, the polyphase channelizer's bin count was
hardcoded to 16 — sized for 2.4 MS/s ≈ 150 kHz per bin. At 10 MS/s that became
625 kHz bins, merging adjacent carriers. (A wrinkle worth knowing: these four
taps, spaced 25–75 kHz apart, sit *below* the channelizer's minimum bin width, so
they were routed to the per-tap DDC and never touched the channelizer — the two
halves of the issue title were unrelated code paths all along.)

Both fixes merged. The reporter rebuilt, retested — same symptom. And then did
the single most useful thing in the whole thread: took the live daemon, the USB
bus, and real time out of the loop entirely.

## Offline replay, and a process failure

`gophertrunk capture` at both rates, same antenna, same gain, same center — then
`gophertrunk replay` against the static files. The 2.5 MS/s capture locked and
decoded the full band plan. The 10 MS/s capture did not lock, at any tap offset.
No USB ring, no CPU pressure, no live component: the fix that targeted the
CPU/USB-overrun mechanism *could not* have addressed this, because this
reproduces with none of those in play.

Two things went wrong at this point, and both are worth owning in print. First,
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) was **closed twice
on an unverified fix while the symptom was still live**, with close comments that
re-stated the original fix instead of engaging the follow-up — which is why
[#771](https://github.com/MattCheramie/GopherTrunk/issues/771) exists as an issue
at all. That failure directly reshaped the project's issue-closing policy: no
close-as-completed until a failing-first test passes *and* the symptom is
verifiably gone, and follow-ups get addressed on their own terms, never by
repeating the original fix description.

Second, a subtler confusion: the merged fix changed the multi-tap wideband
`DDCBank` (`internal/dsp/tuner/ddc.go`) — but `gophertrunk replay -tune-hz` runs
the single-channel `ccdecoder.Downconverter` (`internal/scanner/ccdecoder/ddc.go`),
a separate path the fix never touched. The "fix doesn't work" replay result was
**structurally impossible to be about that fix**. Two code paths that look like
one is a trap that recurred in later issues; now it's documented.

## The red herring: "AGC stuck 10× high"

The replay diagnostics showed `agc_level≈1.47` against `agc_target=0.15708` —
roughly 10× above target, never converging. It looked like a smoking gun for a
gain-normalization bug in the new shared decimation stage, and the follow-up
issue was framed around it.

Then someone checked the *passing* case. The working 2.5 MS/s capture shows the
**identical** `agc_level≈1.47`. That ~9× ratio is simply the symbol-domain AGC's
normal operating gain — the matched filter has gain proportional to
samples-per-symbol, and the AGC compensates. The metric was converging fine in
both captures.

> A metric that looks alarming in the failing case must be checked in the
> passing case before it's allowed to be evidence.

That one sentence would have saved days. It's now a standing rule.

## The experiment that settled it

If the replay path is rate-invariant (and a synthetic harness said it was — same
±24 kHz passband, same 48 kHz internal rate, unity DC gain at both input rates),
the only remaining variable is the captured data itself. The decisive control:
take the 10 MS/s capture, decimate it 4:1 with an **independent Kaiser polyphase
resampler — not GopherTrunk's DSP at all** — and feed the result through the
proven 2.5 MS/s path, the exact path that locks the native 2.5 MS/s capture.

| Capture | Demod EVM | Demod SNR | Result |
|---|---|---|---|
| 2.5 MS/s native | 7.4% | 19.7 dB | Locks — full band-plan decode |
| 10 MS/s native | 22.5% | 9.5 dB | No lock |
| 10 MS/s → independent 4:1 decimate → 2.5 MS/s path | 22.5% | 9.5 dB | No lock |

The ~10 dB in-channel deficit survived a clean decimation through a completely
different DSP chain. It was baked into the 10 MS/s samples before any of our code
touched them. That exonerates the DDC, the AGC, and the decode path in one move —
and a regression test (`TestDownconverterSNRInvariantAcrossRate`) now pins the
invariant so a rate-dependent gain-staging bug can't sneak back in silently.

Alongside it, the systematic rule-outs: band-limiting the 10 MS/s input first
(no change — not aliasing of out-of-band neighbors), float32 vs float64
(bit-identical — not precision), channel-filter width swept 24 kHz down to
6.25 kHz (<0.5 dB — not filtering).

## Carrier-clean, modulation-degraded

So what *is* wrong with the 10 MS/s samples? Not clipping — both captures peak
near −48 dBFS, nowhere close to full scale, so overload and intermod are out. And
here is the genuinely counterintuitive part: by wideband FFT, the carrier looks
**cleaner** at 10 MS/s (SNR 36.2 dB) than at 2.5 MS/s (33.1 dB). The energy is
all there. Only the *in-channel modulation quality* is ~10 dB worse — and that
deficit sits co-band with the signal, inside ±2.4 kHz of the discriminator
output, where no filter can remove it.

Carrier-clean but modulation-degraded is the classic signature of **reciprocal
mixing — phase noise on the sampling clock**, smearing the modulation without
touching the carrier's apparent strength. The confirmation came from a
gain-independence test the reporter ran on request: gain 600 gave EVM 22.5% /
SNR 9.5 dB; gain 300 gave 22.7% / 9.4 dB, despite ~6 dB less capture power. A
deficit that doesn't track gain isn't compression or IMD. It's fixed to the
clock path.

The hardware picture closes the loop: the Airspy R2 exposes exactly two native
rates, 10 and 2.5 MS/s — and 2.5 is the FPGA's decimate-by-4 of the same 10 MS/s
ADC, which averages down the clock phase noise. The deficit appeared at the R2's
*top native rate*. That generalizes into a purchasing caution: the Airspy Mini
shares the R820T2 tuner but samples on a different clock, and its 6 MS/s ceiling
is the same untested top-rate regime — measure before you rely on it. The
qualification recipe (capture the same channel at 2.4 MS/s and the rate under
test, replay both with `-diag`, compare the EVM/SNR line) is now in
[Airspy rate selection]({{ '/reference/airspy-rate-selection/' | relative_url }}).

What shipped, besides the two real fixes: a **no-lock reason string** in replay
("EVM 22.5% / demod SNR 9.5 dB is below the lock threshold — SNR-limited capture,
not a tuning or AGC issue"), so the next person sees the cause instead of a
misleading AGC number, and an opt-in `-soft-sync` mode that extends reach into
marginal captures without being able to manufacture a false lock. The reporter's
verdict: staying on 2.5 MS/s for this site — which the analysis confirms is the
right call on this hardware.

## What we keep

- **Check the alarming metric in the passing case.** The "AGC stuck 10× high"
  reading was identical in the capture that locked. This rule alone reorders
  most investigations, and it opens the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **An independent implementation is the ultimate control.** Decimating with a
  resampler that isn't yours separates "our DSP degrades the data" from "the data
  arrived degraded" in one experiment.
- **Carrier-clean + modulation-degraded = clock phase noise.** The wideband FFT
  can look *better* while the channel is unusable; the deficit lives co-band and
  ignores gain. See
  [signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).
- **Know which code path your repro exercises.** The daemon's wideband `DDCBank`
  and replay's single-channel `Downconverter` are separate; a fix to one proves
  nothing about the other. Name the path in the log line.
- **A dongle's top native rate is its least-tested regime.** Qualify it
  empirically per
  [Airspy rate selection]({{ '/reference/airspy-rate-selection/' | relative_url }})
  before building a site plan on it.
- **Don't close what you haven't verified.** This issue was closed twice while
  the symptom was live. The policy that came out of it — failing-first test,
  reporter confirmation, address the latest follow-up — is why later postmortems
  in this series are shorter.
