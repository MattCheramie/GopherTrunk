---
title: "The Analog Edge, Part 13: Coherence, Not dBFS — Scale-Invariant Health"
description: The story of an operator who raised front-end gain seventeen decibels to push a number past a software constant, and the scale-invariant cross-correlation that replaced the gate — why coherence answers the question a calibrator actually asks, and why DC removal inside it is load-bearing.
category: tutorials
keywords: normalised cross correlation, coherence gate sdr, scale invariant signal quality, dbfs threshold trap, mrc calibration gate, dc offset correlator, rho snr mapping, gain staging trap, gophertrunk analog edge
tags: [analog-edge, coherence, dbfs, calibration, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 13
---

*Part 13 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk system. [Part 12]({{ '/blog/tutorials/analog-edge-12-front-end-classes/' | relative_url }})
classified front ends by watching `branch_phase_deg` — a log field standing in
for test equipment. This part is about the number that made the whole
diversity stack honest, and the incident that forced it into existence. Back
in [Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
we promised the full story of the operator who raised gain 65 → 82 dB to
clear a software constant by 0.2 dB. Here it is — and the scale-invariant
replacement is the most transferable idea this series has to offer.*

> **TL;DR:** MRC calibration used to be gated on the reference branch clearing
> **−40 dBFS** — so an operator raised front-end gain from 65.0 to 82.0 dB
> purely to push a meter past a constant (landing at −39.8 dBFS, clearing it
> by 0.2 dB). Any absolute-power gate is a gain-staging trap that re-fires on
> the next front end. The replacement is the **normalised cross-correlation**
> `|rho| = |Σ x1·conj(x0)| / sqrt(Σ|x0|²·Σ|x1|²)` (`diversity.CrossStats`),
> which is scale-invariant and answers the question the calibrator actually
> has: *are these two receivers seeing the same signal?* The mapping
> `|rho| = γ/(1+γ)` makes thresholds interpretable — **0.50 ≈ 0 dB** per-branch
> SNR, **0.35 ≈ −2.7 dB** — against a noise-only floor near `sqrt(π/4N)`.
> Inside the correlator, **DC removal is load-bearing, not hygiene**; absolute
> power survives only as a −100 dBFS digitally-dead-branch reject.

**Key takeaways**

- **An absolute dBFS gate makes operators tune the radio to the software.**
  The 17 dB gain raise bought nothing but intermod risk — the number it chased
  measured *level*, and the decision needed *evidence*. Every absolute
  threshold quietly re-fires when the hardware changes.
- **Coherence is the question, so measure the question.** "Are these branches
  the same signal through a constant complex gain?" has a direct statistic,
  and it reads the same at any gain setting — which is why the not-coherent
  WARN can honestly say raising gain will not help.
- **The γ/(1+γ) mapping turns a unitless number into dB.** A threshold of 0.5
  isn't folklore; it's "0 dB per-branch SNR," chosen on a scale where the
  noise floor for N samples is computable.
- **An uncentred correlator lies confidently.** Shared LO leakage is
  common-mode DC on both branches; without DC removal, two branches of pure
  independent noise read |rho| → 1 — a calibrator would lock onto nothing, in
  exactly the weak-signal regime the gate exists to protect.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The statistic | DC-removed normalised cross-correlation over a window | `internal/dsp/diversity/crossstats.go` (`CrossStats.Coherence`) |
| The gain beside it | least-squares branch gain from the same sums | `CrossStats.Gain` (`h = cov(x,ref)/var(ref)`) |
| Thresholds | lock 0.50 (≈0 dB), track 0.35 (≈−2.7 dB) | `internal/sdr/soapyremote/mrc.go` (`mrcCoherenceLockGate`, `mrcCoherenceTrackGate`) |
| DC-removal proof | common DC must not read as correlation | `TestCrossStatsRejectsCommonDCOffset` (`crossstats_test.go`) |
| The power floor that remains | reject a digitally dead branch only | `MinBranchPower` 1e-10 ≈ −100 dBFS (`tracking.go`) |
| Operator surface | `coherence` in the 30 s MRC health line | `diversityReporter` (`mrc.go`), [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }}) |
| The deep dive | full calibration design + measurements | [WSE Part 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }}) |

## In this post

- **The 0.2 dB incident** — how a threshold turns into a tuning target.
- **The scale-invariant question** — what coherence measures.
- **Reading |rho| in dB** — the γ/(1+γ) mapping and the noise floor.
- **DC removal is load-bearing** — the trap inside the correlator.
- **The general moral** — gating decisions on evidence, not level.

## The 0.2 dB incident

The old MRC calibration logic refused to estimate branch gains until the
reference branch's level cleared −40 dBFS. Reasonable-sounding — "don't
calibrate on noise" — and wrong in a way that only shows up in the field. An
operator with a healthy but conservatively-gained front end saw diversity
refuse to engage, read the number it wanted, and did the only thing the gate
rewarded: raised RF gain from 65.0 dB to 82.0 dB. They landed at −39.8 dBFS —
clearing the constant by 0.2 dB — and diversity engaged. Nothing about their
signal had improved. They had spent 17 dB of front-end headroom — the same
headroom [Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})
showed protecting you from intermod — to move a meter past a line in the
source code.

That is the anatomy of every absolute-power gate: it couples a *decision*
(can I trust this estimate?) to a *level* (how hot is the ADC running?), and
operators respond to what's rewarded. [Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
made the case that dBFS is a headroom meter, not a quality meter — the
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) captures both
peaked ≈−48 dBFS and differed by 10 dB of usable SNR. A gate on dBFS is that
category error, compiled in. And it re-fires on every new front end: the
constant that a USRP clears at gain 65 needs 82 on the next radio, and
whatever an RTL-SDR needs is nobody's guess. The fix was to delete the
question the gate was asking and ask the real one.

## The scale-invariant question

What the calibrator actually needs to know is: *are these two receivers
seeing the same signal through a constant complex gain?* That question has a
direct statistic — the normalised cross-correlation of the two branches over
an accumulation window:

```go
// internal/dsp/diversity/crossstats.go (shape)
// Gain()      the least-squares complex gain  h = cov(x,ref)/var(ref)
// Coherence() the normalised cross-correlation |rho| in [0,1]
//
// Coherence is the quality gate. It answers "are these two receivers seeing
// the same signal through a constant complex gain?" … and — unlike an
// absolute power threshold — it is SCALE-INVARIANT.
func (s *CrossStats) Coherence() float64
```

Scale-invariance is the whole point: multiply either branch by any constant —
turn any gain knob anywhere — and |rho| does not move, because the
denominator normalises both branches' energy away. The statistic reads the
*relationship* between the branches, not their level. This is why the
not-coherent WARN from [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})
can state flatly that raising RF gain will not help — mathematically, it
can't. The same accumulated sums also yield the branch gain `h` itself, so
the gate and the estimate come from one pass over one window: if |rho| says
the window is trustworthy, `h` from that window is the calibration.

## Reading |rho| in dB

A unitless 0-to-1 number invites folklore thresholds. What keeps this one
honest is that it maps to something physical: for two branches carrying a
common signal in independent noise at equal per-branch SNR γ,

**|rho| = γ / (1 + γ)**

So 0.50 is γ=1 — **0 dB** per-branch SNR. 0.35 is about **−2.7 dB**. And two
branches of pure independent noise over N samples don't read zero — they read
near `sqrt(π/4N)`, about **0.03 at N=1000**: the floor any threshold must
clear. GopherTrunk's gates sit exactly on this scale: the *first* estimate (a
one-shot calibration lives with it forever) must clear **0.50**; subsequent
tracking updates, whose errors average away, need only **0.35**
(`mrcCoherenceLockGate`, `mrcCoherenceTrackGate`).

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Coherence magnitude rho plotted against per-branch SNR in decibels, following the curve gamma over one plus gamma: rho rises from near zero at minus ten decibels through 0.35 at about minus 2.7 decibels and 0.5 at zero decibels, saturating toward one above ten decibels. The two gate thresholds at 0.35 and 0.5 are marked with horizontal dashed lines, and the noise-only floor near 0.03 is marked along the bottom.">
  <line x1="60" y1="15" x2="60" y2="165" stroke="var(--fg-muted)"/>
  <line x1="60" y1="165" x2="650" y2="165" stroke="var(--fg-muted)"/>
  <text x="14" y="25" fill="var(--fg-muted)" font-size="9">|rho|</text>
  <text x="560" y="182" fill="var(--fg-muted)" font-size="9">per-branch SNR (dB)</text>
  <text x="52" y="180" fill="var(--fg-muted)" font-size="9">−10</text>
  <text x="244" y="180" fill="var(--fg-muted)" font-size="9">0</text>
  <text x="432" y="180" fill="var(--fg-muted)" font-size="9">+10</text>
  <text x="622" y="180" fill="var(--fg-muted)" font-size="9">+20</text>
  <text x="40" y="169" fill="var(--fg-muted)" font-size="9">0</text>
  <text x="40" y="29" fill="var(--fg-muted)" font-size="9">1</text>
  <polyline points="60,152 155,131 199,116 250,95 345,59 440,38 535,29 650,26" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="60" y1="95" x2="250" y2="95" stroke="var(--accent)" stroke-dasharray="4,4"/>
  <line x1="250" y1="95" x2="250" y2="165" stroke="var(--accent)" stroke-dasharray="4,4"/>
  <circle cx="250" cy="95" r="4" fill="var(--accent)"/>
  <text x="66" y="90" fill="var(--accent)" font-size="9">lock gate 0.50 ⇔ 0 dB</text>
  <line x1="60" y1="116" x2="199" y2="116" stroke="var(--accent)" stroke-dasharray="4,4"/>
  <circle cx="199" cy="116" r="4" fill="var(--accent)"/>
  <text x="66" y="130" fill="var(--accent)" font-size="9">track gate 0.35 ⇔ −2.7 dB</text>
  <line x1="60" y1="161" x2="650" y2="161" stroke="var(--fg-muted)" stroke-dasharray="2,5"/>
  <text x="470" y="156" fill="var(--fg-muted)" font-size="9">noise-only floor ≈ sqrt(π/4N) (~0.03 at N=1000)</text>
  <text x="352" y="202" text-anchor="middle" fill="var(--fg-muted)" font-size="10">|rho| = γ/(1+γ): every threshold on this curve is a statement in dB — and no gain knob moves a point along it</text>
</svg>
<figcaption>The γ/(1+γ) curve makes coherence thresholds interpretable: 0.5 means 0 dB per-branch SNR, 0.35 means −2.7 dB, and the noise-only floor is computable.</figcaption>
</figure>

Absolute power didn't vanish entirely — it survives as exactly one check, at
a level no gain-staging decision will ever brush against: a branch below
`MinBranchPower` (1e-10 linear, ≈**−100 dBFS**, where one CS16 LSB is
−90.3 dBFS) is *digitally dead* — all zeros, a receiver that never digitised —
and there |rho| is 0/0. That's not a signal-presence test; it's a
divide-by-zero guard, which is all absolute power was ever qualified to be.

## DC removal is load-bearing

One trap hides inside the statistic itself, and it's worth knowing because it
generalizes. Every sum in `CrossStats` is a **DC-removed** covariance, not a
raw correlation — and that's not numerical hygiene. Both receivers of a
shared front end carry LO leakage and converter DC offset, and that offset is
perfectly **common-mode**: identical on both branches, always. Feed two
branches of pure independent noise plus a shared DC into an *uncentred*
correlator and it reports |rho| → 1 — mathematically correct, the DC really
is perfectly correlated — and hands back `h = dc1/dc0`. A calibrator gated on
that would confidently lock onto *nothing*, in exactly the weak-signal regime
the gate exists to protect, and the combined output would be two noise floors
aligned by their LO leakage. `TestCrossStatsRejectsCommonDCOffset` pins the
trap shut: the centred statistic reads that same input as the noise it is.
The cost of correctness is one extra running sum per branch. The cost of the
bug would have been invisible.

## The general moral

The series keeps meeting the same lesson wearing different clothes, so here
it is stated once, plainly: **never gate a DSP decision on an absolute
level; gate it on scale-invariant evidence.** Gain sweeps are scored by
decode error rate, not power
([The Hunt Part 5]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})).
Equalizers are judged by CRC yield, not EVM (the
[metrics-that-lie]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
problem). Calibration is gated on coherence, not dBFS. In every case the
absolute number was easier to compute and already on hand — and in every case
it measured the radio's *configuration* when the decision needed the
*signal's* testimony. When you find yourself adjusting hardware to satisfy a
software number, stop and ask what question the number was standing in for.
The full design that came out of this incident — window sizing, the hold-
don't-fallback rule, and what the gates measured on real captures — is
[Weak-Signal Engineering Part 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }}).

## Where this goes next

That's the last new instrument this series needed. Everything is now on the
bench: dBFS read correctly, gain staged by decode quality, overload and phase
noise recognized on sight, feedline and LNA budgets done in dB, captures that
prove things, diversity with honest health numbers.
[Part 14]({{ '/blog/tutorials/analog-edge-14-field-checklist/' | relative_url }})
folds all of it into one field checklist — the is-it-RF-or-is-it-software
triage tree — and closes the story of the reader whose hardware scanner used
to win.

## FAQ

**Isn't a coherence gate just a threshold too? Why is 0.5 better than −40 dBFS?**
Both are thresholds; they differ in what they measure. |rho|=0.5 is a
statement about the *signal* (0 dB per-branch SNR) that reads identically on
any hardware at any gain. −40 dBFS is a statement about the *ADC's operating
point*, which the operator controls — so it selects for operators who turn
the knob, not for trustworthy estimates.

**What coherence should I see on a healthy system?**
Track the mapping: branches at 5 dB SNR cohere at ~0.76, at 10 dB ~0.91. A
co-located pair on a live trunking band typically reads 0.6–0.9. The two
diagnostic regimes are near-floor (~0.03–0.1: branches see different signals
— band, polarization, clocking) and stuck mid-range (~0.3–0.5: the
wideband-scalar limit from [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})).

**Could the DC trap really happen, or is it theoretical?**
It's the default outcome, not an edge case: *every* zero-IF front end leaks
some LO, so the common-mode DC is always present, and the trap fires
strongest precisely when there's the least real signal to out-correlate it.
That's why the test pins it and why "hygiene" is the wrong word — remove the
centring and the diversity stack fails quietly on weak signals only.

**Where else does GopherTrunk apply this rule?**
Anywhere a decision used to want a power gate: autogain optimizes decode
error rate across the sweep; the MRC dead-branch check sits at −100 dBFS
(existence, not quality); and the equalizer work is validated exclusively by
CRC-clean decode counts on captures. The
[diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }})
collects the field-facing versions.

**I'm not running diversity. What do I take from this part?**
The trap generalizes to *your own* debugging: any time you're pushing gain,
rate, or thresholds to make a dashboard number cross a line, you're gating on
level. Ask instead for scale-invariant evidence — decode rate, CRC yield,
sync margin — the numbers that can't be flattered by a knob.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Front-End Classes — Shared LO vs Independent PLLs]({{ '/blog/tutorials/analog-edge-12-front-end-classes/' | relative_url }})
· Next →
[Part 14: The Field Checklist — Is It RF or Is It Software?]({{ '/blog/tutorials/analog-edge-14-field-checklist/' | relative_url }})
