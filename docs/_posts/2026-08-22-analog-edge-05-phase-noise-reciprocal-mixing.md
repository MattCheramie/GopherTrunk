---
title: "The Analog Edge, Part 5: Phase Noise & Reciprocal Mixing — The Ten-Megasamples Lesson"
description: "The front-end failure no level meter can see: how oscillator phase noise smears modulation while leaving the carrier looking clean, retold for operators through the #764 verdict — same antenna, same signal, 10 dB of demod SNR gone at one sample rate and not the other."
category: tutorials
keywords: phase noise sdr, reciprocal mixing, local oscillator noise, carrier clean modulation degraded, demod snr vs fft snr, airspy 10 msps, evm degradation, sample rate signal quality, gophertrunk analog edge
tags: [analog-edge, phase-noise, oscillator, sdr, debugging, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 5
---

*Part 5 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation. [Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
closed with a warning: clearing clipping and intermod narrows the front-end
search but doesn't end it, because the worst analog failure leaves every
level statistic pristine. This part is that failure — the one that cost the
project its longest hunt, [#764](https://github.com/MattCheramie/GopherTrunk/issues/764),
and taught the signature our marginal reader most needs to recognize:
**carrier-clean but modulation-degraded**. If your waterfall looks great and
your decoder disagrees, this is the part to read twice.*

> **TL;DR:** Your tuner's **local oscillator** is multiplied into every
> sample, so its **phase noise** — tiny random jitter in when it crosses
> zero — lands directly on the received phase. On a phase-modulated signal
> that *is* the payload. The [#764](https://github.com/MattCheramie/GopherTrunk/issues/764)
> verdict, from the reporter's own captures of one site on one antenna: at
> **2.5 MS/s**, demod SNR ≈ **19.7 dB**, EVM 7.4%, locks; at **10 MS/s**,
> demod SNR ≈ **9.5 dB**, EVM 22.5%, never locks. Neither clips (both peak
> ≈ −48 dBFS), and the wideband FFT carrier SNR was actually **higher** at
> 10 MS/s. An independent-resampler A/B pinned the deficit into the samples
> themselves — the signature of **front-end phase noise / reciprocal
> mixing** at the Airspy's native 10 MS/s clock. Operator rule: **prefer
> rates your front end is clean at, and never trust an FFT to grade
> modulation.**

**Key takeaways**

- **The LO's flaws are multiplied into every sample.** Mixing doesn't add
  the oscillator's jitter, it *transfers* it — every received symbol
  inherits the LO's phase wander, and no downstream filter can remove what
  is now part of the signal.
- **Carrier-clean but modulation-degraded is the tell.** Power metrics, the
  waterfall, even wideband FFT SNR stay healthy while EVM balloons and
  demod SNR collapses. It's the one front-end failure the whole level
  toolkit of Parts 2–4 cannot see.
- **Reciprocal mixing turns neighbors into noise.** A noisy LO also smears
  every *strong nearby* signal's energy across your channel — the receiver
  equivalent of everyone on the band shouting through your oscillator.
- **The capture rate can change the analog front end.** Same chip, same
  antenna, different sampling configuration, different oscillator
  behavior — which is why "works at 2.5 MS/s, fails at 10 MS/s" can be a
  true statement about *hardware*, not software.

## Cheat sheet

| Concern | What it tells you | Where it lives |
|---|---|---|
| The physics, one page | oscillator jitter → carrier skirts → reciprocal mixing | [Phase noise]({{ '/reference/phase-noise/' | relative_url }}) |
| The full detective story | two real bugs, one red herring, one verdict | [Ten Megasamples]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}) |
| The quality number that matters | demod SNR / EVM from the actual receiver | `gophertrunk replay` metrics; the rate-invariance test in `internal/scanner/ccdecoder/ddc_highrate_test.go` |
| Proving it's the samples | independent-resampler A/B, in depth | [Weak-Signal Engineering, Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}) |
| Rate choice on Airspy hardware | which rates the front end favors | [Airspy rate selection]({{ '/reference/airspy-rate-selection/' | relative_url }}) |
| Why levels can't see it | both #764 captures peaked ≈ −48 dBFS | Part 2's regime table |

## In this post

- **The symptom that didn't fit** — everything measured healthy, nothing
  decoded.
- **What phase noise actually does** — skirts, smear, and reciprocal
  mixing.
- **The verdict, in numbers** — the two-capture table.
- **Why the wideband FFT lied** — integrated power vs symbol accuracy.
- **What an operator does about it** — rates, A/Bs, and when to stop
  blaming software.

## The symptom that didn't fit

Rewind the story to where an operator would have stood. Four P25 control
channels on one Airspy R2. At 2.5 MS/s, the nearest tap decodes cleanly. To
cover all four, the rate goes up to 10 MS/s — and every tap goes dark,
*including the strong one that just worked*. Levels: fine, ≈ −48 dBFS peak,
no clipping (Part 4's tests all pass). Waterfall: the carriers are right
there, plainly visible. Two genuine software bugs were found and fixed along
the way — a per-tap CPU blowup and a hardcoded channelizer bin count — and
the symptom *survived them*, even in pure offline replay of the captures
([#771](https://github.com/MattCheramie/GopherTrunk/issues/771)). Every
level instrument in Parts 2–4 said "healthy"; the decoder said no. When your
instruments and your outcome disagree that stubbornly, you're missing an
instrument — and the missing one here measures *phase*.

## What phase noise actually does

A receiver tunes by multiplying the incoming spectrum against a local
oscillator. In the idealized diagram the LO is a perfect sinusoid — a
single infinitely thin spectral line. A real oscillator wobbles: each zero
crossing arrives a few picoseconds early or late, at random. In the
frequency domain that timing jitter becomes **skirts** — a pedestal of
noise spreading out from the carrier line, described in dBc/Hz at various
offsets (the [Field Guide entry]({{ '/reference/phase-noise/' | relative_url }})
unpacks the units).

Multiplication transfers those skirts onto everything the receiver hears,
and that hurts you twice:

- **Directly:** your wanted carrier is convolved with the LO's smear, so
  each symbol's phase arrives with the oscillator's wander added. For a
  phase-carrying modulation — P25's C4FM ultimately conveys frequency/phase
  trajectories; TETRA's π/4-DQPSK is explicitly differential phase — that
  wander is indistinguishable from noise *in exactly the dimension the data
  lives in*.
- **Reciprocally:** every *strong neighbor* in the front end's view is also
  convolved with the skirts, and the edges of a strong neighbor's smear
  land on your channel as broadband noise. This is **reciprocal mixing** —
  a clean signal into a noisy oscillator produces the same in-channel floor
  as a noisy signal into a clean oscillator. On a busy trunking band with
  strong carriers everywhere, a modest LO can set your noise floor all by
  itself.

Note what neither mechanism touches: total power. The energy isn't removed,
it's *relocated* — from crisp symbol positions into a blur around them.
Every level metric of Parts 2–4 conserves; only symbol-accuracy metrics
(EVM, demod SNR, error rate, CRC yield) see the loss.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Left: a clean local oscillator drawn as a single thin spectral line yields a tight four-point constellation. Right: a noisy local oscillator drawn with wide skirts around its line yields the same constellation smeared into arcs, with a note that total power is unchanged while symbol accuracy is destroyed. Underneath, a strong neighbor's skirt is shown overlapping the wanted channel, labelled reciprocal mixing.">
  <line x1="30" y1="110" x2="300" y2="110" stroke="var(--fg-muted)"/>
  <polyline points="150,110 155,20 160,110" fill="none" stroke="currentColor"/>
  <text x="155" y="14" text-anchor="middle" fill="currentColor" font-size="9">clean LO</text>
  <circle cx="245" cy="45" r="4" fill="var(--accent)"/><circle cx="275" cy="45" r="4" fill="var(--accent)"/>
  <circle cx="245" cy="75" r="4" fill="var(--accent)"/><circle cx="275" cy="75" r="4" fill="var(--accent)"/>
  <text x="260" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tight symbols</text>
  <line x1="380" y1="110" x2="650" y2="110" stroke="var(--fg-muted)"/>
  <path d="M 460 110 Q 495 90 500 22 Q 505 90 540 110" fill="none" stroke="currentColor"/>
  <polyline points="495,110 500,22 505,110" fill="none" stroke="currentColor"/>
  <text x="500" y="14" text-anchor="middle" fill="currentColor" font-size="9">noisy LO: skirts</text>
  <path d="M 585 38 A 22 22 0 0 1 605 52" fill="none" stroke="var(--accent)" stroke-width="3" opacity="0.7"/>
  <path d="M 585 82 A 22 22 0 0 0 605 68" fill="none" stroke="var(--accent)" stroke-width="3" opacity="0.7"/>
  <path d="M 565 52 A 22 22 0 0 0 565 68" fill="none" stroke="var(--accent)" stroke-width="3" opacity="0.7"/>
  <text x="592" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">smeared symbols</text>
  <text x="340" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="10">same total power on both sides — the energy moved from symbol positions into the blur</text>
  <line x1="120" y1="196" x2="560" y2="196" stroke="var(--fg-muted)"/>
  <path d="M 350 196 Q 385 176 390 152 Q 395 176 430 196" fill="none" stroke="currentColor"/>
  <text x="390" y="146" text-anchor="middle" fill="currentColor" font-size="9">strong neighbor × noisy LO</text>
  <polyline points="465,196 470,178 475,196" fill="none" stroke="var(--accent)"/>
  <text x="500" y="176" fill="var(--accent)" font-size="9">your channel, under its skirt</text>
  <text x="340" y="216" text-anchor="middle" fill="var(--fg-muted)" font-size="10">reciprocal mixing: the neighbor stays clean on the display while its smear becomes your noise floor</text>
</svg>
<figcaption>Phase noise relocates energy rather than removing it: the constellation smears, the neighbors' skirts land in your channel, and every level meter stays green.</figcaption>
</figure>

## The verdict, in numbers

Here is #764 reduced to its evidence table — one site, one antenna, the
reporter's own captures, replayed offline through the same code:

| Measurement | 2.5 MS/s capture | 10 MS/s capture |
|---|---|---|
| Peak level | ≈ −48 dBFS | ≈ −48 dBFS |
| Clipping | none | none |
| Wideband FFT carrier SNR | high | **higher** |
| Demod SNR | ≈ 19.7 dB | ≈ 9.5 dB |
| EVM | 7.4% | 22.5% |
| Control-channel lock | yes | no |

The clincher was the independent-resampler experiment: decimate the
10 MS/s file 4:1 with a resampler that isn't GopherTrunk's, feed the result
to the proven 2.5 MS/s decode path, and the *same* ≈ 9.5 dB deficit comes
out — so the missing 10 dB is baked into the captured samples, not
introduced by our DDC. (One sentence here; the full methodology of proving
"it's the samples" by rate-invariance and independent resamplers is
[Weak-Signal Engineering, Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }}).)
Carrier-clean, modulation-degraded, level-innocent, rate-correlated: that
constellation of facts points at the front end's oscillator behavior at the
Airspy's native 10 MS/s clock — phase noise / reciprocal mixing recorded
permanently into every sample.

## Why the wideband FFT lied

The most disorienting row of that table is the FFT one: the capture that
*failed* had the **higher** wideband carrier SNR. It isn't a paradox once
you know what each instrument integrates. An FFT bin sums power — and phase
noise conserves power, merely relocating it within and around the channel.
A tall, proud carrier in the waterfall is a statement about *energy*, not
about whether that energy's phase trajectory still encodes symbols. The
demodulator, meanwhile, is an interferometer: it compares each symbol's
phase against where a clean symbol would be, and it feels every picosecond
of LO jitter. Same signal, two instruments, opposite verdicts — and the
decoder's is the one correlated with reality. This is the series' Part 2
lesson escalated one level: first we learned dBFS can't grade quality; now
even *spectral* SNR can't. Only demodulation-domain numbers grade
modulation.

## What an operator does about it

You cannot fix an oscillator in YAML, but you have real levers:

- **Prefer rates your front end is clean at.** The sampling configuration
  is part of the analog design — clocking, decimation, and PLL settings
  shift with it. If a lower rate covers your channels, take it; the
  [Airspy rate-selection notes]({{ '/reference/airspy-rate-selection/' | relative_url }})
  exist because of exactly this history. The decode path itself is
  rate-invariant (Part 6), so rate choice is purely a front-end decision.
- **A/B by capture, not by vibes.** Record the same channel at both
  candidate rates, replay both, compare demod SNR/EVM/lock. Ten minutes,
  and the answer is durable evidence instead of an impression.
- **Weigh oscillator quality when buying.** Part of what separates SDR
  tiers is exactly LO cleanliness under real clocking loads — worth as much
  as any dB of gain on a phase-modulated trunking band. (Our
  [hardware comparison]({{ '/airspy-vs-rtl-sdr-vs-hackrf/' | relative_url }})
  is the buying-side companion.)
- **Keep strong neighbors out of the front end.** Reciprocal mixing needs a
  strong neighbor to reciprocate with; Part 9's filters reduce the supply.
- **Stop blaming software once the signature matches.** Carrier-clean,
  modulation-degraded, reproducible from a capture, correlated with a
  front-end configuration change: that's an analog verdict. File the issue
  *with the capture* (Part 10) — but point it at the right side of Part 1's
  line.

## Where this goes next

This part leaned on a claim that deserves its own post: that GopherTrunk's
decode path treats every capture rate identically, so rate-correlated
symptoms indict the front end. [Part 6]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }})
opens that up — how both down-converters normalize to one per-protocol
channel rate, when higher rates genuinely help, what they cost, and the two
log lines (`decode can't keep up with real time`, soapyremote `host_drops`)
that look like driver bugs and are actually downstream signals.

## FAQ

**Is phase noise a defect in my SDR?**
It's a budget line in every oscillator ever built — the question is degree,
and price roughly tracks it. The operational point isn't "buy perfection,"
it's that phase-noise behavior can differ between *configurations of the
same device* (as #764 showed across sample rates), so measure your rig at
the settings you actually run.

**How do I check for this signature without lab gear?**
With the decoder as the instrument. Capture the same channel under both
configurations you're comparing, replay, and read demod SNR/EVM and lock.
Carrier visible in the waterfall + healthy dBFS + collapsed demod SNR that
follows one configuration is the fingerprint. No spectrum analyzer
required — the receiver is a phase-accurate one already.

**Could multipath or ISI produce the same "clean carrier, bad decode" look?**
Yes, and that's a fair confounder — linear channel distortion also degrades
EVM while conserving power. The discriminators: ISI is equalizable (and
GopherTrunk's equalizers recovering most of the loss points that way),
tends to follow *location and antenna*, and doesn't track front-end
configuration. Phase noise follows the *hardware configuration* and no
linear equalizer can undo it. The #764 deficit tracked the sample rate on
one antenna — hardware.

**Why did this take an independent resampler to prove?**
Because the obvious A/B — decode the 10 MS/s capture through our own
10 MS/s path — can't separate "the samples are bad" from "our high-rate
path is bad." Decimating with a *third-party* resampler and replaying
through the already-proven 2.5 MS/s path removes our high-rate code from
the loop entirely. The deficit survived, so it lived in the samples. It's
the same self-consistency discipline the project applies to synthetic
tests, aimed at hardware.

**Does this mean I should always run the lowest rate?**
No — it means rate is a front-end quality choice, not a free coverage knob.
Part 6 gives the full trade: what higher rates buy (more taps per dongle,
wideband hunting), what they cost (front-end cleanliness, CPU, USB), and
how to verify your hardware's clean rates empirically.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Clipping, Overload & Intermod]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
· Next →
[Part 6: Sample Rate — The Decode Path Doesn't Care; the Front End Does]({{ '/blog/tutorials/analog-edge-06-sample-rate/' | relative_url }})
