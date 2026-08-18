---
title: "Weak-Signal Engineering, Part 11: Diversity II — Tracking Without Breaking the Differential"
description: "Why independent-PLL front ends make a frozen diversity calibration decay, and how TrackingCalibrator re-estimates the branch gain continuously yet stays safe ahead of a differential decoder — the anchored phase, the hold-don't-fallback rule, the step clamp, and the four-arm capture A/B."
category: deep-dives
keywords: tracking calibrator, diversity phase drift, independent pll front end, differential decoder phase, one-pole smoothing, mrc static vs tracking, coherence hold, capture a/b testing, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, diversity, mrc, tracking, dsp, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 11
---

*Part 11 of **Weak-Signal Engineering**, a 14-part series on decoding the
marginal regime — where the receiver locks but only a fraction of frames
survive. [Part 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
built the coherence-gated calibration that decides when two branches can be
combined at all. It left one assumption standing: that the branch gain, once
measured, stays put. On some hardware it does. On a USRP with two TwinRX
daughterboards it does not — the relative phase is random at every LO lock
and walks afterwards. So the calibration must track. But
[Part 5]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }})
spent an entire post on why a continuously-adapting filter ahead of a
differential decoder corrupts every dibit. This part is about why tracking
MRC survives that rule — and the claim is measured, not asserted.*

> **TL;DR:** `diversity.TrackingCalibrator` re-estimates the branch gain from
> each ~2 ms `CrossStats` window and smooths one-pole with α = 0.01 (τ ≈
> 200 ms). It is safe ahead of a differential decoder for a **structural**
> reason CMA lacked: the reference branch's weight is pinned to exactly
> `1+0j`, so the output phase is ANCHORED to the reference branch's own —
> only the estimate *error* can move it, and `arg(y/x0) ≈ −ε·|h|²/(1+|h|²)`,
> zero at convergence. `TestTrackingCalibratorIsDifferentialSafe` bounds the
> per-window output phase step against the 45° π/4-DQPSK decision spacing. A
> rejected window **holds** the previous gains — never falls back to
> passthrough, because a mid-stream fallback is itself a phase step.
> `mrc-static` (α = 0) is the one-shot escape hatch: the first accepted
> window is snapped, not smoothed, bit-identical to a one-shot least-squares
> calibration. The verdict instrument is `TestDiversityCombinerReplay`: four
> decode arms scored by CRC-clean BSCH — and tracking-as-default is **not yet
> verified on air**.

**Key takeaways**

- **Hardware class decides whether a constant is a lie.** Shared-LO
  front ends (one AD9361, a B210) have genuinely fixed branch phase;
  separate daughterboards with independent PLLs are frequency-locked but
  phase-random per lock, walking after. Tracking is required for the second
  class and a variance-reducing no-op for the first.
- **The CMA lesson does not transfer — structurally.** CMA's cost is
  rotation-invariant, so nothing constrains its output phase. Here `h₀ ≡ 1+0j`
  anchors the output to the reference branch; only estimate error perturbs
  it, and that error is gated, damped by α, and hard-clamped.
- **Hold, never fall back.** Dropping to passthrough mid-stream is a large
  phase discontinuity — the exact failure class being avoided. A rejected
  window keeps the previous gains; a clamped |h| cannot diverge the way CMA
  taps can, so holding is safe where CMA needed a re-seed.
- **Yield is the verdict, and the on-air verdict is still open.** The replay
  harness scores each-branch-alone, static, and tracking by CRC-clean BSCH.
  Until an operator's own capture shows tracking winning, it is a lever, not
  a default.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Tracking loop | window estimate → gate → one-pole → clamp | `internal/dsp/diversity/tracking.go` (`TrackingCalibrator.Observe`) |
| Phase anchor | reference weight pinned to `1+0j` | `tracking.go` (`estimate`, `applied[0]`) |
| Gates | phase-error bounds 0.10/0.16 rad (|rho| ≥ 1/sqrt(1+2Nσ²)), dead branch −100 dBFS | `tracking.go` (`TrackingOptions.LockGate`/`TrackGate`) |
| Step clamp | ≤ 0.05 rad (~2.9°) phase, ≤ 1.25× magnitude per update | `tracking.go` (`clampStep`, `trackingMaxStepRad`) |
| Differential safety | per-window output phase step ≤ 1° measured | `tracking_test.go` (`TestTrackingCalibratorIsDifferentialSafe`) |
| One-shot mode | α = 0 snaps the first window, then freezes | `tracking.go` (`estimate`), `mrc.go` in `internal/sdr/soapyremote` (`mrc-static`) |
| Capture A/B | four arms scored by CRC-clean BSCH | `cmd/gophertrunk/diversity_replay_test.go` (`TestDiversityCombinerReplay`) |
| Pre-combine tap | per-branch cs16 files, alignment invariant | `internal/sdr/soapyremote/branchcapture.go` (`branchRecorder`) |

## In this post

- **Why a frozen constant decays** — two front-end classes, one config knob.
- **Anchored, not rotation-invariant** — the structural difference from CMA.
- **The window loop** — estimate, gate, smooth, clamp.
- **Holds, not fallbacks** — and the reference-flapping fix.
- **The four-arm A/B** — how the verdict gets decided, and what's still open.

## Why a frozen constant decays

A one-shot calibration measures `h` once and applies it forever. Whether that
is correct is a property of the *front end*, not the algorithm. On a
shared-LO, single-chip design — an AD9361-class device, both receive chains
clocked and mixed from one synthesizer — the branch-to-branch phase is fixed
trace skew plus a power-up divider phase, and it genuinely does not move
while the synthesizer stays locked. Freeze away.

Now put the branches on separate daughterboards with independent PLLs — the
reported configuration was `rx_subdev_spec=B:0 A:0` on a TwinRX pair. The
PLLs are frequency-locked to a common reference, so there is no beat note;
but the relative phase is **random at each lock and walks afterwards**. A
constant measured at t=0 is a little wrong at t=10 s and arbitrarily wrong
eventually. The tracking calibrator's docstring compresses it: "a constant
measured once decays."

Because the hardware class decides, the config exposes both:
`diversity: mrc` tracks; `diversity: mrc-static` (α = 0) snaps the first
accepted window and freezes — and the field instrument for choosing is
`branch_phase_deg` in the periodic MRC health line
(`internal/sdr/soapyremote/mrc.go`): constant ⇒ shared-LO, walking ⇒
independent PLLs. The operator-seat reading of that line is covered in
[The Analog Edge, Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }});
this post owns the algorithm.

## Anchored, not rotation-invariant

Part 5's rule was absolute: never feed a continuously-adapting filter to a
differential decoder, because a time-varying output phase does not cancel in
`s·conj(prev)`. Streaming-adaptive CMA ahead of a differential decode
measured CRC 0 — not degraded, zero. So why is a calibrator that updates
every 2 ms window acceptable?

Because the failure was never "adaptation" — it was **unconstrained phase**.
CMA's constant-modulus cost is rotation-invariant: rotate every tap by any
angle and the cost is identical, so the output phase is a free parameter that
wanders as the taps adapt. The tracking combiner has no such freedom. The
reference branch's weight is pinned to exactly `1+0j`, so the combined output
phase is anchored to the reference branch's own phase. Write the second
branch as `x1 = h·x0 + n` and let the estimate carry an error,
`ĥ = h·e^{jε}`. Then

y = (x0 + conj(ĥ)·x1) / (1+|ĥ|²)  ⇒  arg(y/x0) ≈ −ε·|h|²/(1+|h|²)

— at most about ε/2, and **exactly zero when the estimate is right**. The
only phase motion the differential decoder can ever see is the motion of the
estimate *error*, and three mechanisms bound it: the coherence gate keeps
garbage windows out, the one-pole scales each accepted step by α = 0.01, and
`clampStep` hard-limits any single update to 0.05 rad (~2.9°) of phase and
1.25× of magnitude regardless of what slipped through.

And per this series' discipline, the claim is measured, not argued:

```go
// internal/dsp/diversity/tracking_test.go (shape) — TestTrackingCalibratorIsDifferentialSafe
// Per-window residual phase relative to the reference branch. The step
// between consecutive windows is what a differential decode actually sees.
for i := window * 2; i+window <= n; i += window { // skip the lock window
    cur := residualPhaseDeg(y[i:i+window], refN[i:i+window])
    /* … track the max step between consecutive windows … */
}
// pi/4-DQPSK decisions are 45° apart. Anything approaching a degree here
// would already be suspicious; the design budget is hundredths.
if maxStep > 1.0 {
    t.Errorf("max per-window output phase step %.3f°, want <=1° (45° decision spacing)", maxStep)
}
```

At a 2 ms window and α = 0.01 the per-burst perturbation is a hundredth of a
degree against a 45° decision spacing — four orders of magnitude of margin.

## The window loop: estimate, gate, smooth, clamp

`Observe` accumulates branch chunks into `CrossStats` until `WindowSamples`
(default 8192) is reached, then runs one decision:

```go
// internal/dsp/diversity/tracking.go (shape) — estimate
n := c.stats[0].Samples()
gate := coherenceGateFor(c.opts.TrackPhaseSigmaRad, n) // once calibrated…
if !c.calibrated {
    gate = coherenceGateFor(c.opts.LockPhaseSigmaRad, n) // …stricter FIRST
}
/* … per branch: reject dead power, reject non-finite / out-of-bounds h,
   take worst-branch coherence … */
if worst < gate {
    c.hold(worst)
    return false
}
for k := 1; k < c.branches; k++ {
    if !c.calibrated {
        // First accepted window: snap. No smoothing and no clamp, so this
        // is bit-identical to a one-shot least-squares calibration on the
        // same window.
        c.applied[k] = proposed[k]
        continue
    }
    target := smoothGain(c.applied[k], proposed[k], c.opts.Alpha)
    c.applied[k] = clampStep(c.applied[k], target)
}
```

Every constant earns its value. The gates bound the estimate's projected
**phase error** `sqrt((1−ρ²)/(2Nρ²))` rather than |rho| itself — wideband
|rho| is diluted by noise-only bandwidth around the carrier, and an 18 Aug
operator A/B showed a fixed |rho| constant forcing gain-staging games the
same way the old −40 dBFS gate did — so the minimum |rho| falls as
`1/sqrt(N)` while staying ~8× (lock) / ~5× (track) above the `sqrt(π/4N)`
noise-only floor. The **lock bound is stricter than the track bound**
(0.10 vs 0.16 rad) because the first estimate is what a one-shot calibration
lives with forever, while a tracking update's error averages away over ~1/α
windows. The **window is 8192 samples**, not one datagram, because a
184-sample window estimates phase to about 9.5° at |rho| = 0.35 while an
8192-sample window gets ~1.2°. The **time constant τ ≈ 200 ms** sits far
above one TETRA burst (14 ms, so intra-burst phase is effectively constant)
and far below the drift rate of two independently-locked PLLs. The
**divergence bounds** on |h|² (1/1024 … 1024, i.e. ±30 dB) reject estimates
past the point where the weaker branch contributes nothing anyway — far more
likely numerical garbage than measurement. And the **first accepted window is
snapped, not smoothed**, so `mrc` and `mrc-static` start bit-identically —
`TestTrackingFirstUpdateMatchesStaticCalibrate` pins it.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Timeline of branch phase versus time on an independent-PLL front end. The true branch phase walks steadily upward. A frozen static calibration is a horizontal line that diverges from the walking phase, with the growing gap shaded as decode-degrading error. The tracking estimate follows the walking phase closely in small clamped steps, with two hold intervals marked where incoherent windows kept the previous gain instead of falling back.">
  <line x1="50" y1="20" x2="50" y2="180" stroke="var(--fg-muted)"/>
  <line x1="50" y1="180" x2="640" y2="180" stroke="var(--fg-muted)"/>
  <text x="20" y="30" fill="var(--fg-muted)" font-size="9">branch</text>
  <text x="20" y="42" fill="var(--fg-muted)" font-size="9">phase</text>
  <text x="600" y="196" fill="var(--fg-muted)" font-size="9">time</text>
  <polyline points="60,150 130,142 200,131 270,123 340,110 410,100 480,86 550,74 630,64" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="632" y="58" text-anchor="end" fill="currentColor" font-size="9">true phase (walks)</text>
  <line x1="60" y1="150" x2="630" y2="150" stroke="var(--fg-muted)" stroke-dasharray="6 4"/>
  <text x="628" y="164" text-anchor="end" fill="var(--fg-muted)" font-size="9">mrc-static: frozen at first lock</text>
  <polyline points="60,150 130,145 200,134 270,126 340,113 410,113 480,89 550,77 630,67" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="330" y="212" text-anchor="middle" fill="var(--accent)" font-size="9">tracking: one-pole steps, ≤2.9° per update</text>
  <rect x="340" y="104" width="70" height="18" fill="none" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="375" y="98" text-anchor="middle" fill="var(--accent)" font-size="8">hold (|rho| &lt; 0.35)</text>
  <line x1="550" y1="74" x2="550" y2="150" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="556" y="120" fill="var(--fg-muted)" font-size="8">static's growing error</text>
</svg>
<figcaption>On independent-PLL hardware the true branch phase walks; the frozen calibration's error grows without bound while tracking follows in clamped, gated steps — and holds rather than falling back when a window is incoherent.</figcaption>
</figure>

## Holds, not fallbacks — and the flapping fix

What happens when a window fails a gate? The obvious answer — drop back to
reference-only passthrough until coherence returns — is wrong, and the
docstring says why: "falling back mid-stream is itself a large phase step,
the exact failure class being avoided." A stale-but-bounded weight is
strictly better than a step. So a rejected window **holds**: gains stay,
`holds` increments, and `TestTrackingCalibratorHoldsOnIncoherentWindow`
pins that pure-noise windows neither move the gains nor drop the combiner to
passthrough. This is a deliberate divergence from `SnapshotCMA`'s
re-seed-to-passthrough guard
([Part 6]({{ '/blog/deep-dives/weak-signal-engineering-06-normalisation-guards/' | relative_url }})),
and the difference is principled: CMA's taps can diverge without bound, so a
poisoned filter must be reset; a clamped |h| cannot, so it is safe to keep.
The `Counters()` pair makes the state legible — a climbing hold count against
a flat update count is the signature of two receivers not seeing the same
signal.

A related fix from the same review: while uncalibrated, the reference branch
was re-chosen by `argmax` of branch power on **every datagram**. Two healthy
receivers routinely cross within ~1 dB, so the phase anchor itself swapped
branches mid-stream — a discontinuity injected by the very machinery meant to
prevent one. The reference is now chosen stably rather than per-datagram.

## The four-arm A/B — and what is still open

Every claim above is synthetic-pinned, and this series has a name for
trusting that alone: the self-consistent-synthetic trap. The on-air verdict
needs pre-combine truth, which is why the capture tap exists where it does —
`branchRecorder` (`internal/sdr/soapyremote/branchcapture.go`) dumps each
branch to its own headerless cs16 file *before* the combiner, with an
alignment invariant: a datagram that did not carry every branch is dropped
from **both** files and counted, never written short, because one short
write silently desynchronises the pair and invalidates every later
conclusion. Every other IQ tap in GopherTrunk sits downstream of the
combiner and can answer nothing about it.

`TestDiversityCombinerReplay` (`cmd/gophertrunk/diversity_replay_test.go`,
driven by `GT_DIVERSITY_CAPTURE=<prefix>.diversity.json`) then does two
things. It prints a windowed coherence/gain/**phase** trace with an explicit
verdict line — flat phase ⇒ "a frozen calibration is sufficient here";
walking ⇒ "tracking is doing real work", in degrees per second. And it
decodes four arms through the identical TETRA BSCH scorer:

```go
// cmd/gophertrunk/diversity_replay_test.go (shape) — the four arms
arms := []struct{ name string; iq []complex64 }{
    {"branch0-only", br0},
    {"branch1-only", br1},
    {"static-combine", combineWith(t, br0, br1,
        diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window, Alpha: 0}))},
    {"tracking-combine", combineWith(t, br0, br1,
        diversity.NewTrackingCalibrator(2, diversity.TrackingOptions{WindowSamples: window}))},
}
// "CRC-clean BSCH count is the verdict. Do NOT conclude anything from EVM…"
```

Each branch alone, one-shot static, and tracking — scored by CRC-clean BSCH,
per [Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})'s
rule that a combiner can improve every intermediate metric while decoding
nothing. The honest status line: **tracking-as-default is not yet verified on
air.** The harness, the tap, and the verdict criteria are all in the tree;
the gate is an operator's own capture showing the tracking arm winning.

## Where this goes next

Diversity closes the lever list. What remains is the discipline that decides
*which* lever a symptom actually calls for — and the sharpest example in the
project's history is a signal that decoded at one capture rate and not
another. [Part 12]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
turns [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) into a
worked example of experimental design: find the invariant your DSP
guarantees, isolate it with an independent implementation, and prove the
deficit lives in the samples before you touch the code.

## FAQ

**If tracking is safe and degrades to static behaviour on stable hardware, why keep `mrc-static` at all?**
As the explicit escape hatch and the A/B control. α = 0 freezes the first
accepted window — bit-identical to a one-shot least-squares calibration — so
an operator on shared-LO hardware can rule the tracking loop out of any
comparison, and the replay harness gets its static arm from the same code
path rather than a second implementation.

**Doesn't holding stale gains during a long fade apply a wrong calibration?**
It applies a *bounded* one. |h| is clamped within ±30 dB and the phase error
grows only as fast as the hardware drifts — while the alternative, a
passthrough fallback, injects an immediate large phase step into a live
differential decode. When the fade lifts, gated windows resume and the
one-pole walks the estimate back.

**How do I know which front-end class I have without capturing anything?**
Watch `branch_phase_deg` in the periodic `soapyremote: MRC diversity
branches` log line. Essentially constant across minutes ⇒ shared LO, and
`mrc-static` is sufficient; walking ⇒ independent PLLs, and tracking is doing
real work. The replay harness prints the same verdict from a capture.

**Why is the window's coherence taken from the worst branch?**
One incoherent branch is enough to make the joint estimate untrustworthy —
combining a good branch with a garbage-calibrated one can cancel signal. The
gate is only as strong as its weakest input, so that is what it measures.

**What would change if the branches sat on separate dongles with free-running clocks?**
Everything — free-running LOs differ in *frequency*, so the branch phase
spins continuously rather than walking, and no per-window scalar tracker
holds. This design assumes a common reference (frequency lock); without one
you need per-sample frequency alignment first, which is a different machine.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Diversity I — MRC & Coherence-Gated Calibration]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
· Next →
[Part 12: Proving It's the Signal — Rate Invariance & Independent Resamplers]({{ '/blog/deep-dives/weak-signal-engineering-12-proving-signal/' | relative_url }})
