---
title: "The Analog Edge, Part 1: Where the Software Ends"
description: The signal chain from antenna to ADC and why no code change can fix a problem on the analog side of it — an inventory of GopherTrunk issue-tracker bugs that turned out to live in the samples, and the map of a 14-part field guide to the front end.
category: tutorials
keywords: sdr signal chain, antenna to adc, rf front end basics, samples not software, sdr troubleshooting, front end overload, phase noise capture, trunking scanner rf, gophertrunk analog edge
tags: [analog-edge, rf, front-end, sdr, hardware, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 1
---

*Part 1 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation — everything between the tower and the first sample
the software ever sees. The whole series exists because of one recurring
operator: the one whose hardware scanner decodes a system cleanly while
GopherTrunk, on the same antenna, produces garble. Over fourteen parts we walk
that reader's marginal system through every analog cause, one at a time, until
it's fixed. The motto we'll keep coming back to: **the decoder can only be as
good as the samples — and half the hard bugs in the issue tracker were in the
samples.***

> **TL;DR:** Your signal passes through **antenna → feedline → filter/LNA →
> tuner → ADC** before a single line of Go runs, and no code change can repair
> damage done in that chain. The issue tracker proves it: the
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) "all taps go
> dark at 10 MS/s" mystery was **front-end phase noise recorded into the
> capture**; the "nineteen dibits" report was a capture **50% pinned to the ADC
> rails**; a DMR two-slot fix is still waiting on air time because the only
> capture supplied sits at **−75 dBFS with no frame sync**. GopherTrunk
> surfaces the analog side with numbers — `gophertrunk_sdr_iq_power_dbfs`,
> `gophertrunk_sdr_iq_clip_ratio`, demod SNR — and this series teaches you to
> read them.

**Key takeaways**

- **The chain has a hard boundary at the ADC.** Everything before it is
  physics you configure with hardware and gain; everything after it is
  deterministic math. A fix on one side never fixes the other.
- **The samples carry the damage forward, permanently.** Noise folded onto a
  channel by an overloaded amplifier or a noisy oscillator is
  indistinguishable, downstream, from a weak signal — the decoder cannot
  subtract it back out.
- **GopherTrunk's decode path is deliberately rate-invariant**, which is what
  makes "works at 2.4 MS/s, fails at 10 MS/s" a statement about the *capture*,
  not the DSP — the reasoning that closed #764.
- **The instruments already exist.** dBFS gauges, clip ratios, decode error
  rates, and demod SNR are exported today; the series is about which one to
  read for which suspicion.

## Cheat sheet

| Stage | What goes wrong there | Where GopherTrunk shows it |
|---|---|---|
| Antenna | wrong band, wrong pattern, indoors | `gophertrunk_sdr_iq_power_dbfs` stuck near idle (≈ −45) |
| Feedline & connectors | loss that adds straight to noise figure | Part 8; no direct meter — the math is the meter |
| Filter / LNA | overload, intermod from strong neighbors | `gophertrunk_sdr_iq_clip_ratio` (`internal/metrics/prom.go`) |
| Tuner / LO | phase noise, reciprocal mixing | demod SNR low while the FFT looks clean — [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) |
| ADC | clipping, rail-pinned samples | `iq_clip_ratio` sustained above ~0.002 |
| Software | everything after the ADC | logs, tests, and the rest of this blog |

## In this post

- **The chain, end to end** — five analog stages and one bold line.
- **Why a software fix never fixes the samples** — the one-way valve.
- **The bugs that were in the samples** — the issue-tracker inventory.
- **The reader we're writing for** — the marginal system we'll carry through
  the series.
- **The series map** — fourteen parts, front to back.

## The chain, end to end

Every recording GopherTrunk ever makes starts as a voltage on an antenna
element. From there the signal runs a fixed gauntlet: down a length of coax
(losing a little at every foot and every adapter), optionally through a filter
and a low-noise amplifier (gaining level, and — if you're unlucky — gaining
garbage), into the tuner chip where a local oscillator mixes it down to
baseband, and finally into the ADC, which measures the waveform a few million
times a second and emits numbers.

That last step is the boundary this series is named for. Left of the ADC,
you're doing RF engineering: antennas, cable, gain, oscillators. Right of it,
you're doing arithmetic — and arithmetic is repeatable, testable, and
replayable. GopherTrunk leans hard on that: the same capture replayed through
the same code produces the same result, every time. Which is exactly why, when
a symptom *survives* into offline replay, the suspect list collapses to one
side of the line.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="The receive signal chain drawn as five boxes: antenna, feedline, filter and LNA, tuner, and ADC, followed by a software box. A bold vertical line sits between the ADC and the software box, marking where the analog world ends and deterministic code begins. A caption under the analog side reads physics you configure, and under the software side reads math you can replay.">
  <rect x="8" y="60" width="88" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="52" y="78" text-anchor="middle" fill="currentColor" font-size="10">antenna</text>
  <text x="52" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P7</text>
  <line x1="96" y1="82" x2="112" y2="82" stroke="currentColor"/><polygon points="112,78 120,82 112,86" fill="currentColor"/>
  <rect x="120" y="60" width="88" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="164" y="78" text-anchor="middle" fill="currentColor" font-size="10">feedline</text>
  <text x="164" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P8</text>
  <line x1="208" y1="82" x2="224" y2="82" stroke="currentColor"/><polygon points="224,78 232,82 224,86" fill="currentColor"/>
  <rect x="232" y="60" width="92" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="278" y="78" text-anchor="middle" fill="currentColor" font-size="10">filter / LNA</text>
  <text x="278" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P4, P9</text>
  <line x1="324" y1="82" x2="340" y2="82" stroke="currentColor"/><polygon points="340,78 348,82 340,86" fill="currentColor"/>
  <rect x="348" y="60" width="88" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="392" y="78" text-anchor="middle" fill="currentColor" font-size="10">tuner / LO</text>
  <text x="392" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P5</text>
  <line x1="436" y1="82" x2="452" y2="82" stroke="currentColor"/><polygon points="452,78 460,82 452,86" fill="currentColor"/>
  <rect x="460" y="60" width="72" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="496" y="78" text-anchor="middle" fill="currentColor" font-size="10">ADC</text>
  <text x="496" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P2, P4</text>
  <line x1="548" y1="28" x2="548" y2="140" stroke="var(--accent)" stroke-width="3"/>
  <text x="548" y="18" text-anchor="middle" fill="var(--accent)" font-size="10">software begins</text>
  <line x1="532" y1="82" x2="556" y2="82" stroke="currentColor"/><polygon points="556,78 564,82 556,86" fill="currentColor"/>
  <rect x="564" y="60" width="108" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="618" y="78" text-anchor="middle" fill="var(--accent)" font-size="10">GopherTrunk</text>
  <text x="618" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DDC → demod → FEC</text>
  <text x="270" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="10">physics you configure — damage done here rides in the samples forever</text>
  <text x="618" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="10">math you can replay</text>
  <text x="340" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="10">a symptom that reproduces in offline replay lives LEFT of the line — that one rule closed #764</text>
</svg>
<figcaption>The receive chain with the series' one bold line: everything left of the ADC is analog, and this series is about that half.</figcaption>
</figure>

## Why a software fix never fixes the samples

The ADC is a one-way valve for damage. Once a strong FM broadcast station has
driven your LNA into intermodulation and splattered phantom energy across your
control channel, the samples *contain* that energy — it has the same units,
the same statistics, the same everything as real signal. Once a noisy local
oscillator has smeared its phase jitter onto the carrier, the constellation
the software receives is already blurred. There is no field in a complex
sample that says "this part was added by your front end"; subtracting it back
out is not hard, it's *impossible* in the general case.

The reverse is just as true, and just as often missed: no antenna upgrade
fixes a bug in the Viterbi decoder, and no LNA fixes a wrong CRC polynomial.
The two sides fail independently and must be debugged independently. That's
why GopherTrunk's discipline — replay everything, pin everything with a
capture — matters so much. A capture freezes the analog side at the moment of
failure, so the software side can be interrogated separately, forever. The
project's own [capture-needs page]({{ '/decoder-capture-needs.html' | relative_url }})
exists because "get the raw capture" is the single most repeated sentence in
the tracker.

## The bugs that were in the samples

The motto isn't rhetoric; here is the inventory. Each of these arrived as a
plausible software bug report, consumed real debugging effort on the software
side, and resolved on the analog side:

| The report | Where it actually lived | The full story |
|---|---|---|
| "All four P25 taps go dark at 10 MS/s" ([#764](https://github.com/MattCheramie/GopherTrunk/issues/764), [#771](https://github.com/MattCheramie/GopherTrunk/issues/771)) | front-end phase noise at the Airspy's native 10 MS/s clock, recorded into the capture — ~10 dB of demod SNR gone with no clipping anywhere | [Ten Megasamples]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}) |
| "The wideband DDC starves the demod of dibits" | the raw capture was 50% pinned to the ADC rails; the fix was turning the gain *down* | [Nineteen Dibits]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }}) |
| "Simulcast systems need the CQPSK demod" | the real fault was a gain value; the modulation myth had leaked into our own docs | [The LSM Myth]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }}) |
| DMR two-slot decode, awaiting on-air verification | the only IQ grab supplied is a dead capture — ~−75 dBFS RMS, carrier 11 kHz off, no frame sync to be found | the A/B still waits on a decodable capture |
| TETRA control-channel sync losses over a 1-hour session | a weak front end (peak −44 dBFS) in the marginal-SNR regime; an equalizer now recovers it, but the RF condition is real | Part 7 of this series, and the signal-level fix it points at |

Notice what the middle column has in common: **none of these were visible as
bugs in the code**, because there were no bugs in the code to see. The
symptom was real, the report was honest, and the samples were the problem.
Ruling the analog side in or out *first* — with measurements, not vibes — is
the skill this series teaches.

## The reader we're writing for

Here is the operator we'll carry through all fourteen parts. They have a
hardware scanner — a purpose-built receiver with a tuned front end — that
decodes the local trunked system flawlessly. They stood up GopherTrunk on an
SDR dongle, pointed it at the same system from the same desk, and got a lock
that comes and goes, audio that's garbled when it arrives, and a creeping
suspicion that the software is broken.

Sometimes it is! We keep a whole
[postmortem series]({{ '/blog/solution-postmortem/from-the-issue-tracker-01-first-p25-lock/' | relative_url }})
of times it was. But the hardware scanner's advantage is not better math — it
is a band-filtered, purpose-tuned analog front end, against a wideband SDR
whose front end you have to *stage yourself*. The gap between those two front
ends is measured in exactly the numbers this series covers: headroom (Part 2),
gain (Part 3), overload (Part 4), oscillator quality (Part 5), rate (Part 6),
and the antenna system (Parts 7–9). Close the gap and the general-purpose SDR
plus GopherTrunk usually *beats* the scanner, because the software half can do
things no scanner firmware will. If you're still choosing hardware, start with
[what you need for GopherTrunk]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})
— this series assumes you have something plugged in already.

## The series map

Fourteen parts, in the order the signal (and the debugging) flows:

| Part | Topic | The one thing you'll take away |
|---|---|---|
| 1 | Where the software ends | this post — the line, and the inventory |
| 2 | dBFS | it's a headroom meter, not a quality meter |
| 3 | Gain staging | never chase a software threshold |
| 4 | Clipping, overload & intermod | gain can manufacture signals |
| 5 | Phase noise & reciprocal mixing | carrier-clean but modulation-degraded |
| 6 | Sample rate | the decode path doesn't care; the front end does |
| 7 | Antennas | gain is a shape, not a magnitude |
| 8 | Feedline & connectors | where dB go to die |
| 9 | Filters & LNAs | amplify before loss, never after |
| 10 | Capture discipline | a capture turns "sounds bad" into a test |
| 11 | Two antennas — diversity & MRC | what a second branch buys and costs |
| 12 | Front-end classes | shared LO vs independent PLLs |
| 13 | Coherence, not dBFS | the scale-invariant health number |
| 14 | The field checklist | is it RF or is it software? |

Parts 2–6 are the desk work — numbers you can read tonight on the system you
already have. Parts 7–9 are the hardware work. Parts 10–13 are the advanced
kit: captures, diversity, and the health numbers that don't lie. Part 14
folds it all into one triage flowchart. If you want the theory under the
practice as we go, the [RF & SDR learning module]({{ '/learn/rf-sdr/' | relative_url }})
runs a parallel track from first principles.

## Where this goes next

The first instrument every operator meets — and the first one that misleads
them — is the dBFS meter. [Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
defines full scale precisely, separates peak from RMS, and shows why a signal
peaking at −48 dBFS can be perfectly healthy while another at the *same*
reading is 10 dB short of decodable — the exact pair of captures that closed
#764. You'll leave with a table mapping every dBFS regime to a likely
condition and an action.

## FAQ

**My hardware scanner works and GopherTrunk doesn't. Doesn't that prove the
software is the problem?**
It proves the *systems* differ, and the receiver front end is the biggest
difference. A scanner ships a band-filtered, factory-staged analog chain; an
SDR ships a wideband one you stage yourself. Before filing an issue, measure
where the chains diverge: dBFS and clip ratio (Part 2, Part 4), then a gain
sweep scored by decode quality (Part 3). If those come back clean, *then* it's
software — and Part 10 shows how to capture the evidence.

**How do I know if a problem is left or right of the ADC?**
Replay. Record raw IQ at the moment of failure and play it back through
`gophertrunk replay`. A symptom that reproduces from the file lives in the
samples or the code — and since the code is deterministic and heavily
regression-tested, the next question is whether a *known-good* capture of the
same channel replays cleanly. That two-capture A/B is the exact experiment
that resolved [#764](https://github.com/MattCheramie/GopherTrunk/issues/764).

**Can't the DSP just clean up front-end damage?**
Sometimes, partially. GopherTrunk ships equalizers that recover real losses
from linear channel distortion, and soft-decision decoding buys margin. But
those are mitigations with hard information-theoretic limits — noise folded
into the channel is signal lost forever. The project's own notes on the TETRA
equalizer say it plainly: it mitigates a weak front end but does not replace
it — raise the signal level too.

**Do I need new hardware to follow this series?**
No. Parts 2–6 use only the metrics and logs a running daemon already exports,
plus config changes. Hardware spending starts at Part 7, and Part 14 gives an
explicit order to spend in — antenna first, coax second, filter/LNA third, a
second dongle last.

**Where do the numbers in this series come from?**
From the repo: metric definitions in `internal/metrics/prom.go`, config
guidance in `config.example.yaml`, and the measured postmortems in the issue
tracker. Where a fact is general RF craft rather than GopherTrunk code (coax
loss tables, antenna patterns), we present standard published figures and say
so.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: dBFS — What the Number Means (& What It Doesn't)]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
