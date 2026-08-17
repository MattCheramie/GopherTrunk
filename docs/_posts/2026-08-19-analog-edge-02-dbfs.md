---
title: "The Analog Edge, Part 2: dBFS — What the Number Means (& What It Doesn't)"
description: A precise definition of dBFS for SDR operators — full scale, peak versus RMS, the readings GopherTrunk exports, and why two captures peaking at the same −48 dBFS can differ by 10 dB of usable signal, which makes dBFS a headroom meter and never a quality meter.
category: tutorials
keywords: dbfs meaning, dbfs sdr, adc full scale, peak vs rms, crest factor, iq power dbfs, clip ratio, sdr signal level, headroom meter, gophertrunk analog edge
tags: [analog-edge, dbfs, adc, metrics, sdr, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 2
---

*Part 2 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation. [Part 1]({{ '/blog/tutorials/analog-edge-01-where-software-ends/' | relative_url }})
drew the line at the ADC and inventoried the tracker bugs that lived on the
analog side of it. This part picks up the first instrument on that side of the
line — the dBFS meter — for our running reader with the marginal system,
because the very first thing they'll do is look at a level number and draw
exactly the wrong conclusion from it. dBFS answers one question with total
authority and a different question not at all, and telling those apart is the
whole part.*

> **TL;DR:** **dBFS is level relative to the ADC's full scale** — 0 dBFS is
> the rail, everything real is negative, and the only thing the number
> measures is **headroom**. GopherTrunk exports it as
> `gophertrunk_sdr_iq_power_dbfs` (RMS over a ~1 s window: idle ≈ −45,
> healthy ≈ −25) with `gophertrunk_sdr_iq_clip_ratio` beside it, because
> **RMS averages away peak clipping** — the clip ratio is the authoritative
> overload signal. And dBFS says nothing about quality: the two
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) captures
> both peaked ≈ **−48 dBFS**, yet one locked at ~19.7 dB demod SNR and the
> other sat undecodable at ~9.5 dB. Same meter reading, 10 dB of usable
> signal apart.

**Key takeaways**

- **0 dBFS is a cliff, not a target.** Full scale is where the ADC stops
  being a measuring instrument; every dB of margin below it is headroom
  against clipping, and you want plenty.
- **Peak and RMS are different numbers, and GopherTrunk's gauge is RMS.** A
  high-crest signal can pin the rails on peaks while the average still reads
  a merely-hot −5 dBFS — which is why the clip ratio exists as a separate,
  authoritative metric.
- **dBFS cannot rank two signals by decodability.** It measures how loud the
  digitized waveform is, not how much of that loudness is *your channel*
  versus noise, neighbors, and front-end artifacts.
- **Learn the regimes, not a single magic number.** Idle ≈ −45, healthy
  ≈ −25, > −3 clipping — those anchors come straight from the metric's own
  help text, and the table below fills in the space between.

## Cheat sheet

| Concern | What it tells you | Where it lives |
|---|---|---|
| Mean level, per control SDR | RMS dBFS, ~1 s window, labelled per system | `gophertrunk_sdr_iq_power_dbfs` (`internal/metrics/prom.go`) |
| Overload, authoritatively | fraction of samples pinned to the rail | `gophertrunk_sdr_iq_clip_ratio` — sustained > ~0.002 is overload |
| Whole wideband capture | pre-DDC level per dongle serial | `gophertrunk_sdr_wideband_input_iq_power_dbfs` |
| Wideband overload | a strong site burying weaker taps | `gophertrunk_sdr_wideband_input_clip_ratio` (issue #749) |
| DC contamination | DC-bin power vs total; ≤ −20 dB is clean | `gophertrunk_sdr_iq_dc_ratio_db` |
| The definition itself | full scale, headroom, clipping | [dBFS]({{ '/reference/dbfs/' | relative_url }}) in the Field Guide |

## In this post

- **What full scale actually is** — the ADC's rail and the sign convention.
- **Peak vs RMS** — crest factor, and why the gauge needs a partner metric.
- **Where GopherTrunk shows the number** — the gauges and their anchors.
- **What dBFS cannot tell you** — the two −48 dBFS captures of #764.
- **The regime table** — reading → likely condition → action.

## What full scale actually is

An ADC measures voltage against a fixed reference. Drive it with exactly that
reference voltage and every bit of the sample is used — that's **full scale**,
and we call it 0 dBFS. Everything smaller is expressed as decibels *below*
that ceiling, so real-world readings are always negative: −20 dBFS is a
waveform reaching a tenth of the rail voltage, −40 dBFS a hundredth. (If
decibels themselves are shaky ground, the
[decibels lesson]({{ '/learn/rf-sdr/decibels/' | relative_url }}) in the
learning module is a fifteen-minute fix.)

Two properties follow immediately. First, dBFS is a *digital* unit: it says
where the waveform sits relative to this converter's rail, and nothing about
absolute RF power at the antenna — the same station moves up and down the
dBFS scale as you turn the gain knob. Second, 0 dBFS is not a maximum you
approach for best results; it's the point where the ADC stops measuring and
starts lying. A sample that *would have been* larger than full scale is
recorded at the rail, its true value gone. That's clipping, and Part 4 is
about how catastrophically it spreads across a band.

## Peak vs RMS: why the gauge needs a partner

"The level" is actually two numbers. **Peak** is the largest single sample in
a window; **RMS** is the energy average. The gap between them is the signal's
**crest factor**, and it varies enormously: a single constant-envelope carrier
has a small crest factor, while a wideband capture full of independent
carriers adds up like noise, with rare peaks far above its average.

GopherTrunk's `gophertrunk_sdr_iq_power_dbfs` gauge is RMS over a roughly
one-second window, and its own help text carries the warning that matters:

```text
gophertrunk_sdr_iq_power_dbfs   Mean IQ power on the control SDR in dBFS
    (window ~= 1 s). Idle ≈ -45; healthy signal ≈ -25. This is RMS and
    averages away peak clipping — watch iq_clip_ratio for the
    authoritative overload signal.
gophertrunk_sdr_iq_clip_ratio   Fraction of IQ samples pinned to the ADC
    rail. 0 = no clipping; a sustained value above ~0.002 means front-end
    overload — reduce gain or add attenuation, do not raise gain.
```

Read that pair as designed: a high-crest capture can clip on peaks while the
RMS reads a merely "hot" −5 dBFS, so **the clip ratio, not the power gauge,
is the overload verdict**. This is not a hypothetical — the
[Nineteen Dibits postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }})
turned on a capture that was 50% rail-pinned while every downstream symptom
pointed, convincingly, at software.

## Where GopherTrunk shows the number

Four gauges cover the level story end to end. Per decoded system, the
narrowband `iq_power_dbfs` (labelled `"<system> @ <freq> MHz"`) tracks the
channel actually feeding a decoder. Per dongle, the wideband pair —
`wideband_input_iq_power_dbfs` and `wideband_input_clip_ratio` — measures the
*whole capture before channelization*, which matters because every tap on a
wideband dongle shares one gain: a strong site can overload the shared ADC
and bury its weaker siblings, the issue #749 failure mode the
`config.example.yaml` comments walk through. A tap sitting far below the
wideband input power, or any sustained non-zero clip ratio, is also logged as
a throttled WARN so you don't need a Prometheus dashboard to catch it.

The anchors to memorize come from the help text: **idle ≈ −45 dBFS** (an
antenna hearing only the noise floor), **healthy ≈ −25 dBFS**, and anything
above **−3 dBFS** means clipping. Near-zero readings in the *other* direction
— down at −70 and below — mean the chain is delivering essentially nothing:
no antenna, gain at zero, or a dead cable. The DMR two-slot work from Part 1
is still blocked on exactly such a capture: ~−75 dBFS RMS, no frame sync
anywhere in it.

## What dBFS cannot tell you

Here is the part that saves you a week of mis-aimed debugging. In
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764), the reporter
supplied two captures of the *same site* from the *same antenna*: one at
2.5 MS/s, one at 10 MS/s. Both peaked at approximately **−48 dBFS** — neither
clipped, neither was weak enough to suspect a dead branch. On the level
meter, they are identical twins.

Decoded, they are nothing alike. The 2.5 MS/s capture replays at ~19.7 dB
demod SNR (EVM 7.4%) and locks; the 10 MS/s capture manages ~9.5 dB (EVM
22.5%) and never locks. Ten decibels of usable signal separate two files the
dBFS meter cannot distinguish — because the missing 10 dB wasn't *level*, it
was signal *replaced by* front-end phase noise, sample for sample, at the
same amplitude. The full detective story is in
[Ten Megasamples]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }}),
and Part 5 of this series retells its physics for operators.

The moral generalizes: **dBFS is a headroom meter.** It tells you whether the
ADC has room to represent the waveform without lying. It does not tell you
what fraction of that waveform is your channel, whether the modulation
survived the front end, or whether a decoder will lock. Quality questions
belong to quality numbers — demod SNR, decode error rate, and ultimately CRC
yield — which is a thread that runs all the way to Part 13's
coherence-over-dBFS argument.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A vertical dBFS scale from 0 down to minus 80. The top region above minus 3 is marked clipping. Minus 25 is marked healthy. Minus 45 is marked idle noise. Below minus 70 is marked dead chain. Two markers sit together at minus 48: capture A, which decodes at 19.7 dB SNR, and capture B, which fails at 9.5 dB SNR, illustrating that the same dBFS reading cannot distinguish signal quality.">
  <line x1="90" y1="20" x2="90" y2="210" stroke="var(--fg-muted)"/>
  <line x1="86" y1="20" x2="94" y2="20" stroke="currentColor"/>
  <text x="80" y="24" text-anchor="end" fill="currentColor" font-size="10">0 dBFS</text>
  <rect x="90" y="20" width="180" height="12" fill="var(--accent)" opacity="0.35"/>
  <text x="280" y="30" fill="var(--accent)" font-size="10">&gt; −3: clipping — the ADC is lying</text>
  <line x1="86" y1="78" x2="94" y2="78" stroke="var(--fg-muted)"/>
  <text x="80" y="82" text-anchor="end" fill="var(--fg-muted)" font-size="10">−25</text>
  <text x="100" y="82" fill="currentColor" font-size="10">healthy signal (the gauge's own anchor)</text>
  <line x1="86" y1="130" x2="94" y2="130" stroke="var(--fg-muted)"/>
  <text x="80" y="134" text-anchor="end" fill="var(--fg-muted)" font-size="10">−48</text>
  <circle cx="130" cy="126" r="4" fill="var(--accent)"/>
  <text x="140" y="124" fill="currentColor" font-size="9">capture A: locks, demod SNR ≈ 19.7 dB</text>
  <circle cx="130" cy="138" r="4" fill="none" stroke="var(--accent)"/>
  <text x="140" y="142" fill="currentColor" font-size="9">capture B: no lock, demod SNR ≈ 9.5 dB — same meter reading</text>
  <line x1="86" y1="118" x2="94" y2="118" stroke="var(--fg-muted)"/>
  <text x="80" y="116" text-anchor="end" fill="var(--fg-muted)" font-size="10">−45</text>
  <text x="100" y="108" fill="var(--fg-muted)" font-size="10">idle: antenna hears only the noise floor</text>
  <line x1="86" y1="186" x2="94" y2="186" stroke="var(--fg-muted)"/>
  <text x="80" y="190" text-anchor="end" fill="var(--fg-muted)" font-size="10">−75</text>
  <text x="100" y="190" fill="var(--fg-muted)" font-size="10">dead chain — the unusable DMR capture from Part 1 lives here</text>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="10">dBFS places you on this ladder and nothing more — quality lives in demod SNR, error rate, and CRC yield</text>
</svg>
<figcaption>The dBFS ladder with its anchors — and the #764 twins at −48 dBFS, identical on this meter and 10 dB apart in usable signal.</figcaption>
</figure>

## The regime table

What to do with a reading, from the top of the ladder down:

| `iq_power_dbfs` reads… | Likely condition | Action |
|---|---|---|
| above −3 | clipping — the ADC is saturated | reduce gain or add attenuation *now*; never raise gain (Part 4) |
| −3 to −15 | hot; RMS may hide peak clipping | check `iq_clip_ratio`; if sustained > ~0.002, back the gain off |
| −15 to −35 | the healthy band (anchor: −25) | leave it alone; judge by decode quality, not by pushing level higher |
| −35 to −55 | workable but lean — both #764 captures peaked ≈ −48 here | fine *if* it decodes; if not, the deficit is SNR, not headroom — sweep gain against decode quality (Part 3), then look at antenna and rate (Parts 5–7) |
| −55 to −70 | very weak; decoders will be marginal | more signal, not more software: gain staging first, then antenna/LNA (Parts 7, 9) |
| below −70 | effectively dead: no antenna, gain 0, broken coax | fix the chain before reading anything else — no capture from here answers any question |

Two habits complete the table. First, always read the clip ratio *with* the
power gauge — the pair is designed as an instrument, and either one alone can
mislead. Second, treat the healthy band as a plateau, not a slope: once
you're clear of both the noise floor and the rails, adding level buys nothing
and eventually costs plenty. The
[SDR gain & overload]({{ '/reference/sdr-gain-overload/' | relative_url }})
Field Guide entry condenses this into one page you can keep open.

## Where this goes next

If dBFS is a headroom meter, what should you actually *turn the gain knob*
against? [Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
answers with a cautionary tale from the tracker — an operator who raised
front-end gain 17 dB to push a reading 0.2 dB past a software constant — and
the method that replaces threshold-chasing entirely: sweep the ladder and
score each rung by decode quality, the way `AutoGainSweep` does it.

## FAQ

**Is a stronger dBFS reading always better?**
No. Between the noise floor and the rails there is a wide plateau where level
changes decode quality not at all, and above it every extra dB eats headroom
until peaks clip. The gauges' anchors — healthy ≈ −25, clipping > −3 — bracket
that plateau. Once you're on it, the knob that matters is covered in Part 3,
and it isn't level.

**My waterfall looks strong but nothing decodes. How?**
Because an FFT display and a decoder measure different things. A carrier can
be tall in the spectrum while its *modulation* is degraded — by phase noise,
intermod, or multipath — and #764 is the canonical case: the wideband FFT
carrier SNR was actually *higher* in the capture that failed. Trust demod SNR
and error rate over any picture of the band.

**What's a normal reading with no antenna connected?**
The gauge should drop toward its idle anchor or below — you're measuring the
receiver's own noise. If it doesn't move when you disconnect the antenna,
you were never measuring the antenna in the first place: suspect a stuck
gain stage, a strong local interferer getting in past the connector, or a
mislabeled device.

**Why does GopherTrunk use RMS for the gauge instead of peak?**
RMS over a ~1 s window is stable enough to alert on and matches how energy
(and therefore SNR) accumulates. Peak alone is jumpy and dominated by single
samples. The design splits the job: RMS for level, and a dedicated clip ratio
for the rail — the one question where peaks are exactly what matters.

**Does dBFS relate to dBm?**
Only through your entire gain chain. dBm is absolute power at a reference
point; dBFS is relative to one converter's full scale, after antenna, coax,
LNA, and tuner gain have all had their say. Change any of them and the same
dBm at the antenna lands at a different dBFS. That's why this series treats
dBFS as a *staging* instrument, never a field-strength meter.

## Series navigation

**Part 2 of 14** · ←
[Part 1: Where the Software Ends]({{ '/blog/tutorials/analog-edge-01-where-software-ends/' | relative_url }})
· Next →
[Part 3: Gain Staging — Never Chase a Software Threshold]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
