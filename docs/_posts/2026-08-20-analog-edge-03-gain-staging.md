---
title: "The Analog Edge, Part 3: Gain Staging — Never Chase a Software Threshold"
description: How to set SDR front-end gain the measured way — the operator who raised gain 17 dB to clear a software constant by 0.2 dB, the ladder method scored by decode quality, autogain's lock-then-error-rate-then-lower-gain tie-break, and the tenths-of-a-dB config trap.
category: tutorials
keywords: sdr gain staging, rtl-sdr gain setting, gain sweep decode quality, agc vs manual gain, tenths of a db, gain config trap, front end headroom, autogain sweep, gophertrunk analog edge
tags: [analog-edge, gain, sdr, configuration, front-end, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 3
---

*Part 3 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk installation. [Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
established dBFS as a headroom meter and nothing more — which leaves our
reader with the marginal system holding a gain knob and no target to turn it
toward. This part gives them one, by way of the tracker's best cautionary
tale: an operator who cranked real RF gain to satisfy an arbitrary software
constant, and what GopherTrunk changed so nobody ever has to do that again.
The rule this part plants: gain is staged against decode quality, and any
absolute threshold you find yourself chasing is a trap.*

> **TL;DR:** An operator once raised front-end gain from **65.0 to 82.0 dB**
> so a branch would read **−39.8 dBFS** and clear a **−40 dBFS** software
> gate — by 0.2 dB. Seventeen decibels of real RF strain to move a number
> past a constant. That gate is gone (its replacement, a scale-invariant
> coherence check, is Part 13's story). The right method is a **ladder
> sweep scored by decoding**: `AutoGainSweep`
> (`internal/hunt/autogain.go`) decodes a dwell at each gain and `pickGain`
> chooses by **lock first, then lowest error rate, then LOWER gain**. And
> mind the config trap: GopherTrunk gain values are **tenths of a dB** —
> `gain: "300"` is 30.0 dB, not 300.

**Key takeaways**

- **Any absolute-power gate is a gain-staging trap.** A threshold in dBFS
  invites the operator to move the *number* instead of the *signal*, and the
  same constant re-fires wrongly on the next front end. GopherTrunk now gates
  DSP decisions on scale-invariant evidence instead.
- **Sweep, don't reason.** The interaction between your antenna, your band,
  and your tuner's gain ladder is empirical. Decode a short dwell at each
  rung and let the error rate rank them.
- **On a tie, take the lower gain.** Two gains that decode equally well are
  not equally safe: the lower one leaves headroom before a strong neighbor
  drives the front end into intermod — that preference is coded into
  `pickGain`.
- **The unit is tenths of a dB.** Coming from SDRTrunk/OP25/gqrx, multiply
  your dB figure by ten. `gain: "496"` = 49.6 dB. This single conversion
  error has produced entire bug reports.

## Cheat sheet

| Concern | What to do | Where it lives |
|---|---|---|
| Set gain in config | tenths of a dB, or `"auto"` for AGC | `sdr.devices[].gain` in `config.example.yaml` |
| Discover the gain ladder | print each device's supported steps | `gophertrunk sdr list --probe` |
| Sweep gains empirically | decode a dwell per rung, recommend one | `internal/hunt/autogain.go` (`AutoGainSweep`) |
| Rank the rungs | lock → error rate → lower gain | `internal/hunt/autogain.go` (`pickGain`) |
| Catch overload while sweeping | rail-pinned fraction, not RMS | `gophertrunk_sdr_iq_clip_ratio` (Part 2) |
| The trap, in one page | tenths-of-dB + overload signatures | [SDR gain & overload]({{ '/reference/sdr-gain-overload/' | relative_url }}) |

## In this post

- **The 0.2 dB story** — seventeen decibels against a constant.
- **Why absolute thresholds are traps** — and what replaced this one.
- **The ladder method** — sweeping gain against decode quality.
- **The tie-break that respects the front end** — `pickGain`'s ordering.
- **The tenths-of-a-dB trap** — the config unit that bites migrators.
- **AGC or fixed?** — the wideband shared-front-end rule.

## The 0.2 dB story

The diversity combiner used to gate its calibration on a simple check: the
reference branch had to clear −40 dBFS before the gain estimate was trusted.
Reasonable-sounding — don't calibrate on noise. But an operator with a
healthy, decodable setup found diversity refusing to engage, read the code,
and did the only thing the gate rewarded: raised the front-end gain from
65.0 dB to 82.0 dB until the branch read −39.8 dBFS. They cleared the
constant by 0.2 dB, at the price of 17 dB of added front-end strain —
17 dB less headroom against every strong neighbor on the band, for zero
improvement in the actual signal.

Nothing about the RF was wrong. The *gate* was wrong, and it converted a
software design flaw into an operator's gain problem. That's the defining
signature of a gain-staging trap: the knob moves to satisfy the instrument
instead of the signal.

## Why absolute thresholds are traps

The deep reason is the one Part 2 ended on: dBFS is relative to *this*
converter behind *this* gain chain. A constant like −40 dBFS encodes an
assumption about antenna, coax, LNA, tuner, and gain setting all at once —
so it is wrong on every rig except the one it was tuned on, and it re-fires
on the next front end forever. Any time you find yourself adjusting real RF
gain to move a reading past a fixed number — a squelch you found in a config,
a threshold you read in a forum post, a gate you found in source code — stop
and ask what the threshold is a *proxy for*, and measure that instead.

In GopherTrunk's case the question the gate was proxying was "is this gain
estimate trustworthy?", and the honest answer is scale-invariant: the
normalized cross-correlation between branches, which doesn't change when you
turn the gain knob — the config comments say it outright: *raising gain to
make diversity engage is never the answer*. That number, `|rho|` from
`diversity.CrossStats`, is [Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})'s
whole subject. Absolute power survives in that code for exactly one job —
rejecting a digitally dead branch at −100 dBFS — because "the cable is
unplugged" is the one condition a level *can* diagnose.

## The ladder method

So what do you turn the knob against? Decode quality, measured, per rung.
Your tuner offers a discrete ladder of gains (print it with
`gophertrunk sdr list --probe`); the interaction of that ladder with your
antenna and your band is not something to reason about from first principles
— it's something to *sweep*. The procedure, whether you run it by hand or
let [autogain]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
do it:

1. Pick the channel you actually care about — your control channel.
2. At each gain rung, dwell a second or two and decode.
3. Record: did it lock? what was the decode error rate? did anything clip
   (`iq_clip_ratio`, not the RMS gauge)?
4. Plot rung against error rate: you'll get the U-curve — errors high at low
   gain (carrier buried in noise), a flat healthy bottom, errors rising again
   at high gain (intermod from strong neighbors).
5. Settle on the bottom of the U, biased toward its *low-gain* edge.

The crucial discipline is step 3's scoring: decode quality, never level. A
higher rung that lifts your carrier also lifts every neighbor and every
intermod product; the level meter goes up while the decode gets worse. The
sweep sees that; a power reading can't.

## The tie-break that respects the front end

GopherTrunk's automated version encodes the whole philosophy in one readable
cascade. `AutoGainSweep` decodes a dwell at each rung through the same
identify→decode path the live hunt uses, then `pickGain` ranks the outcomes:

```go
// internal/hunt/autogain.go (shape) — pickGain ordering
b := scores[best]
switch {
case s.Locked != b.Locked:
    if s.Locked { best = i }              // 1. a gain that LOCKS beats one that doesn't
case s.ErrorRate != b.ErrorRate:
    if s.ErrorRate < b.ErrorRate { best = i } // 2. lower decode error rate
case s.GainTenthDB < b.GainTenthDB:
    best = i                              // 3. lower gain (less front-end strain)
}
```

Rule 3 is the one operators skip when staging by hand, and it's the one this
series cares most about: **among gains that decode equally well, prefer the
lower one.** The rungs at the bottom of the U are equal *today*; they are not
equal the day a new pager transmitter lights up two channels over. The lower
rung is carrying spare headroom against that day. Recommendations are
advisory — the sweep applies nothing behind your back — so the number it
hands you is a starting point you confirm, not a setting that moves itself.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="A bar chart with a dashed horizontal gate line at minus 40 dBFS. A left bar at 65 dB gain reaches minus 43 dBFS, below the gate. A right bar at 82 dB gain reaches minus 39.8 dBFS, just above the gate. An arrow labelled plus 17 dB of real RF gain connects the bars, and a note explains the reading moved only 3.2 dB and the signal quality not at all — the gate, not the gain, was wrong.">
  <line x1="60" y1="20" x2="60" y2="180" stroke="var(--fg-muted)"/>
  <line x1="60" y1="180" x2="640" y2="180" stroke="var(--fg-muted)"/>
  <text x="30" y="30" fill="var(--fg-muted)" font-size="9">dBFS</text>
  <line x1="60" y1="70" x2="640" y2="70" stroke="var(--accent)" stroke-dasharray="6 4"/>
  <text x="632" y="62" text-anchor="end" fill="var(--accent)" font-size="10">the −40 dBFS gate (a constant in software)</text>
  <rect x="140" y="92" width="90" height="88" fill="none" stroke="currentColor"/>
  <text x="185" y="86" text-anchor="middle" fill="currentColor" font-size="10">−43.0</text>
  <text x="185" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="10">gain 65.0 dB</text>
  <rect x="410" y="66" width="90" height="114" fill="none" stroke="var(--accent)"/>
  <text x="455" y="60" text-anchor="middle" fill="var(--accent)" font-size="10">−39.8</text>
  <text x="455" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="10">gain 82.0 dB</text>
  <line x1="235" y1="120" x2="402" y2="120" stroke="currentColor"/><polygon points="402,116 410,120 402,124" fill="currentColor"/>
  <text x="320" y="112" text-anchor="middle" fill="currentColor" font-size="10">+17 dB of real RF gain</text>
  <text x="320" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">…to clear a constant by 0.2 dB — headroom spent, signal quality unchanged</text>
  <text x="350" y="214" text-anchor="middle" fill="var(--fg-muted)" font-size="10">when a knob moves to satisfy a threshold instead of a signal, the threshold is the bug</text>
</svg>
<figcaption>The 0.2 dB story: 17 dB of front-end strain spent pushing a reading past a software constant. The constant was removed; the lesson stays.</figcaption>
</figure>

## The tenths-of-a-dB trap

One more way gain goes wrong before the RF is even involved: the unit.
GopherTrunk's `gain:` values are **tenths of a decibel**, matching the
granularity real tuner ladders expose. From `config.example.yaml`:

```yaml
gain: "auto"   # TENTHS of a dB — NOT dB. "496" = 49.6 dB. SDRTrunk/
               # OP25/gqrx users: multiply your dB figure by 10
               # (32 dB → "320"). "auto" (AGC) is the safe default.
```

A migrator who carries `gain: "300"` over from a tool that means 300 tenths
elsewhere gets 30.0 dB — or, in the other direction, someone intending 30 dB
who writes `30` gets 3.0 dB and a dead-quiet capture. One of the tracker's
documentation-scarred postmortems,
[The LSM Myth]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }}),
pivots on exactly this: a system blamed on modulation modes for weeks, where
the real fault was a gain value. Typical ladders for reference: RTL-SDR
R820T 0–496, Airspy R2/Mini 0–500, HackRF One 0–560 — all in tenths, all
printable with `--probe`.

## AGC or fixed?

For a single-channel dongle parked on one control channel, a fixed, swept-in
gain is the predictable choice. The calculus changes on a wideband dongle
hosting several taps, because every tap shares one gain: a fixed value chosen
so the strongest site doesn't clip leaves weaker co-tenants flat at the ADC
floor — and SNR lost at the converter is gone; no downstream AGC recovers it.
The project's guidance (issue #749, written into `config.example.yaml`) is to
**prefer `gain: "auto"` when one dongle hosts sites of differing strength**;
the daemon even logs a startup WARN when a multi-tap wideband dongle is
pinned to a fixed gain. If the input clip ratio is non-zero, lower the gain
or add attenuation — never raise it. And a genuinely weak distant site may
simply need its own dongle: no gain setting serves two sites 30 dB apart
from one converter. The [AGC]({{ '/reference/automatic-gain-control/' | relative_url }})
Field Guide entry and the
[gain & AGC lesson]({{ '/learn/rf-sdr/gain-and-agc/' | relative_url }})
cover the mechanism itself.

## Where this goes next

The ladder method's step 3 quietly assumed you can recognize overload when
the sweep walks into it. [Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
makes that explicit: what clipping actually looks like in IQ, how an
overdriven front end *manufactures* phantom signals through intermodulation,
why the error-vs-gain curve turns back up, and why "no clipping" still
doesn't clear the front end — the distinction #764 turned on.

## FAQ

**Is there one right gain for my dongle?**
There's one right gain for your dongle *on this antenna, at this frequency,
in this RF neighborhood* — which is why the answer is a sweep, not a lookup
table. Re-sweep after any change to the chain: new antenna, new coax, added
LNA, or even a new strong transmitter moving into the neighborhood.

**Why prefer the lower gain when two rungs decode identically?**
Headroom. The band you swept today is the quietest it will ever be — strong
neighbors come and go, and the higher rung has already spent the margin
you'll want when one appears. `pickGain` encodes that as its final
tie-break; it costs nothing today and saves a re-stage later.

**Should I just use `gain: "auto"` everywhere?**
It's the safe default, and on multi-site wideband dongles it's actively
recommended. The case for fixed gain is a single known channel where a sweep
found a clearly better rung than AGC settles on, or where you want run-to-run
reproducibility (captures for regression tests, A/B experiments — Part 10).
When you do fix it, write down why and what the clip ratio read.

**My squelch/threshold config has dBFS numbers in it. Are those traps too?**
A threshold is a trap when it makes you adjust *RF gain* to satisfy it.
Squelch on a settled, staged system is fine — it's downstream of a chain you
already fixed. The rule of thumb: thresholds may respond to your staging;
your staging must never respond to a threshold.

**The sweep says gain 0 decodes best. Is that believable?**
Yes, near a strong site — it means every higher rung was already degrading
the decode through front-end compression or intermod, and it's a hint your
system would benefit from attenuation or filtering rather than amplification
(Part 9). Believe the decode; it has no incentive to lie.

## Series navigation

**Part 3 of 14** · ←
[Part 2: dBFS — What the Number Means (& What It Doesn't)]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
· Next →
[Part 4: Clipping, Overload & Intermod]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
