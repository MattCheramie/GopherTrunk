---
title: "Weak-Signal Engineering, Part 6: Normalisation & Divergence Guards"
description: The unglamorous half of a working blind equalizer — why the CMA update's cubic scaling in input amplitude makes normalisation part of the algorithm, how an EMA tracking a TDMA downlink's slot-to-slot power swings turned the full equalizer win into CRC zero, and why the divergence guard that re-seeds blown-up taps deserves a test of its own.
category: deep-dives
keywords: cma normalisation, adaptive filter divergence, equalizer divergence guard, tdma power swings, cumulative mean normalisation, ema normalisation trap, blind equalizer stability, adaptive dsp guards, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, cma, equalizer, stability, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 6
---

*Part 6 of **Weak-Signal Engineering**, a 14-part deep dive into decoding the
marginal regime, where a receiver locks but under-decodes.
[Part 5]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }})
delivered the headline mechanism — adapt continuously, apply frozen — and the
headline numbers: TETRA voice bursts 410→778, the thread capture ~12%→~100%.
This part is about the two lines of `SnapshotCMA` that made those numbers
possible and that nobody would put on a slide: the input normaliser and the
divergence guard. Both were discovered the hard way — one of them by watching
the *entire* equalizer win evaporate to CRC zero over a choice of averaging
window. The lesson generalises past CMA: in adaptive DSP, the reference frame
and the failure handling are not packaging around the algorithm. They are the
algorithm.*

> **TL;DR:** The CMA tap update scales like **|x|³** in the input amplitude
> (error `|y|²−R²` is quadratic, the steering term `y·conj(x)` adds another
> power) — so the input's *scale* sets the effective step size and the
> equilibrium. Normalising with a **local EMA** that tracks a TDMA downlink's
> slot-to-slot power swings hands CMA a **moving modulus target**: measured
> result, **CRC 0**, even though a global-RMS normalise on the same capture
> gives the full win. `SnapshotCMA` therefore normalises by a
> **cumulative-mean** power estimate — converging to the session RMS and then
> staying put — and backs it with a **divergence guard**: if any tap's squared
> magnitude exceeds `snapshotDivergeGuard = 9` (|tap| > 3), the tracking
> filter re-seeds to center-spike pass-through, so one normalisation transient
> or deep fade cannot poison every later snapshot. Constant references,
> guarded recovery — both pinned by tests.

**Key takeaways**

- **An adaptive algorithm is only as good as its reference frame.** CMA
  measures "modulus error" against R² = 1 *in normalised units* — if the
  normaliser moves, the target moves, and the equalizer chases its own
  denominator instead of the channel.
- **TDMA is the worst case for local power tracking.** A downlink whose slots
  swing power every 14.17 ms is precisely the signal that makes an EMA
  normaliser oscillate — burst structure and adaptation dynamics interlock.
- **Divergence is an operating condition, not an exception.** Deep fades and
  transients *will* blow up a cubic-scaled update eventually; the design
  question is only whether the filter recovers to neutral or keeps applying
  garbage. The guard makes recovery structural.
- **Guards get tests too.** The no-harm, recovery, and yield regressions pin
  the guard and normaliser behaviour the same way they pin the taps — because
  a silently disabled guard is indistinguishable from a working one until the
  night it matters.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Cumulative-mean normaliser | divide input by √(running mean power) — a constant-ish scale | `internal/dsp/equalizer/snapshot_cma.go` (`Process`: `cumSum`, `count`) |
| Why not EMA | local tracking ⇒ moving modulus target ⇒ CRC 0 | `snapshot_cma.go` (doc comment) |
| Divergence guard | re-seed tracking taps to pass-through at \|tap\| > 3 | `snapshot_cma.go` (`snapshotDivergeGuard = 9.0`) |
| Guarded state only | guard hits `wAdapt`; frozen `wApply` unaffected until next snapshot | `snapshot_cma.go` (`Process`) |
| Full-state reset | re-sync: taps, buffer, *and* normaliser state | `snapshot_cma.go` (`Reset`) |
| Yield regressions | ISI recovery + clean-channel no-harm, CRC/bit-error scored | `snapshot_cma_test.go`, `internal/radio/tetra/receiver/receiver_equalizer_test.go` |

## In this post

- **Why CMA cares about scale at all** — the |x|³ update.
- **The EMA trap** — a normaliser that danced with the TDMA frame.
- **The cumulative mean** — boring on purpose.
- **The divergence guard** — failing back to a wire.
- **Guards are part of the algorithm** — the general lesson.

## Why CMA cares about scale at all

Nothing in Part 4's derivation looked scale-sensitive — until you count powers
of the input. The update is `w ← w − μ·(|y|²−R²)·y·conj(x)`. With `y` linear
in the input, the error factor is quadratic in input amplitude, and `y·conj(x)`
contributes another power: the whole correction scales like **|x|³**. Feed the
same equalizer a signal at half amplitude and the update shrinks 8×; at double
amplitude it grows 8×. The input scale isn't a detail the algorithm tolerates —
it *is* the effective step size, and it also decides where equilibrium lands,
because CMA drives `|y|²` toward a fixed `R²` in whatever units the input
arrives in.

So every practical CMA normalises its input. `SnapshotCMA` bakes the
normaliser into `Process` itself rather than trusting upstream AGC — the
receiver's AGC settles at its own time constant for its own purposes, and
"roughly unit power, eventually" is not the contract a cubic update wants.
The normaliser's *form*, though, turned out to be a live design decision with
a measured, catastrophic wrong answer.

## The EMA trap: a normaliser that danced with the frame

The reflexive choice is an exponential moving average of input power — track
the level, divide it out, self-tuning, done. On a continuous carrier it even
works. On a **TDMA downlink** it walked straight into the signal's structure:
a TETRA base station's four-slot downlink can swing power slot to slot (this
carrier's traffic loud, that idle period quieter), so a local EMA faithfully
tracks a *stair-step* power profile. Divide by a stair-step and the
"normalised" signal's modulus jumps at every slot boundary — which hands CMA a
**moving modulus target**. The algorithm spends every slot re-converging
toward a circle whose radius just changed, and its equilibrium never exists
long enough to reach. The doc comment preserves the measured outcome:

```go
// internal/dsp/equalizer/snapshot_cma.go (shape) — doc
// Input is normalised by a CUMULATIVE mean power estimate — a constant-ish
// scale that converges to the whole-session RMS and stays put. This matters:
// the CMA update scales with |x|³, and a *local* (EMA) normalisation that
// tracks a bursty TDMA downlink's slot-to-slot power swings gives a moving
// modulus target and CMA converges to garbage.
```

And "garbage" here is Part 2's currency: **CRC 0** — while the identical
equalizer, on the identical capture, normalised by a single global RMS,
delivered the full ~2× yield win. The entire difference between total failure
and total success lived in the averaging window of a divisor. It's the EVM
trap's quieter sibling: nothing in the equalizer's own diagnostics flags it,
because the equalizer is dutifully minimising its cost — against a reference
frame that won't hold still.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Three aligned traces over a TDMA downlink. Top: input power as a stair-step, slots alternating loud and quiet. Middle: an EMA normaliser tracking the stair-step, so the modulus target seen by CMA jumps at every slot boundary — labeled moving target, CRC zero. Bottom: the cumulative-mean normaliser as a nearly flat line converging to the session RMS — labeled constant target, full win.">
  <text x="8" y="26" fill="var(--fg-muted)" font-size="9">input power</text>
  <polyline points="90,44 170,44 170,26 250,26 250,50 330,50 330,30 410,30 410,46 490,46 490,28 570,28 570,44 650,44" fill="none" stroke="currentColor"/>
  <text x="370" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TDMA slots swing power every 14.17 ms</text>
  <text x="8" y="96" fill="currentColor" font-size="9">EMA scale</text>
  <polyline points="90,112 150,112 175,98 245,96 255,116 325,118 335,102 405,100 415,114 485,114 495,100 565,98 575,112 650,112" fill="none" stroke="currentColor"/>
  <text x="370" y="88" text-anchor="middle" fill="currentColor" font-size="9">normaliser tracks the slots → modulus target moves → CMA chases it → CRC 0</text>
  <text x="8" y="166" fill="var(--accent)" font-size="9">cumulative</text>
  <text x="8" y="177" fill="var(--accent)" font-size="9">mean</text>
  <path d="M90,186 C150,172 220,166 300,164 C420,162 540,162 650,162" fill="none" stroke="var(--accent)"/>
  <text x="370" y="152" text-anchor="middle" fill="var(--accent)" font-size="9">converges to the session RMS, then stays put → constant R² → full ~2× yield win</text>
  <text x="345" y="208" text-anchor="middle" fill="var(--fg-muted)" font-size="10">same equalizer, same capture — the averaging window of the divisor was the whole difference</text>
</svg>
<figcaption>On a slot-power-swinging TDMA downlink, an EMA normaliser hands CMA a target that moves at frame rate; the cumulative mean gives it one that converges and holds. Measured outcomes: CRC 0 versus the full win.</figcaption>
</figure>

## The cumulative mean: boring on purpose

The fix is aggressively unsophisticated — a running mean over *everything seen
so far*:

```go
// internal/dsp/equalizer/snapshot_cma.go (shape) — Process, the normaliser
px := float64(real(x)*real(x) + imag(x)*imag(x))
e.cumSum += px
e.count++
mean := e.cumSum / e.count
if mean < 1e-9 { mean = 1e-9 }
scale := float32(1 / math.Sqrt(mean))
e.buf[n-1] = complex(real(x)*scale, imag(x)*scale)
```

Early in a stream it adapts quickly (small denominator); as `count` grows, each
new slot's power moves the mean less and less, and the scale converges to the
whole-session RMS and effectively freezes. That trajectory is exactly right
for the job: CMA gets a *constant* reference frame in steady state — R² = 1
means one thing all night — at the cost of the normaliser being deliberately
sluggish about genuine long-term level changes. And that trade is safe
precisely because of what's downstream: `Reset` clears `cumSum`/`count` along
with the taps on every re-sync/retune, so the estimate never spans two
different channels, and the divergence guard (next) catches the transient
where a stale scale meets a step change in level. The design rhymes with
Part 5 deliberately: *snapshot in time* (frozen taps) and *anchor in scale*
(cumulative mean) are the same instinct — give the adaptive core references
that hold still.

## The divergence guard: failing back to a wire

Even with a constant scale, a stochastic-gradient loop with cubic input
scaling lives one bad interval from instability: a deep fade drops the input
near zero (the normalised signal then rears up when it returns), a
normalisation transient at stream start meets a hot slot, and the update takes
a few huge steps. Left alone, blown-up taps don't just fail *now* — under the
snapshot design they get copied into `wApply` at the next refresh and fail
*for the rest of the stream*. Hence the guard, inline in the update loop:

```go
// internal/dsp/equalizer/snapshot_cma.go (shape) — the guard
// snapshotDivergeGuard is the squared-tap-magnitude threshold at which the
// tracking filter is re-seeded to a pass-through (|tap| > 3).
const snapshotDivergeGuard = 9.0

if mx > snapshotDivergeGuard { // taps blew up — re-seed to pass-through
    for k := range e.wAdapt { e.wAdapt[k] = 0 }
    e.wAdapt[n/2] = 1
}
```

Three design choices, each carrying weight. **The threshold is far from
normal operation**: a well-converged inverse for a plausible mobile channel
keeps tap magnitudes near unity, so |tap| > 3 (9× energy) is unambiguous
pathology, not a tuning knob that clips honest adaptation. **The guard
re-seeds `wAdapt` only** — the frozen `wApply` keeps filtering with the last
good snapshot while the tracking filter re-converges from pass-through, so a
one-interval blow-up costs at most the staleness of one snapshot period.
**And it fails toward neutral, not toward history**: the recovery state is
the center-spike wire, whose worst case is "no equalization" — the baseline —
rather than a re-application of whatever state preceded the blow-up. One bad
patch cannot poison later snapshots; the doc comment says exactly that, and
the capture sweeps that scored 410→778 ran with this guard armed.

## Guards are part of the algorithm

Zoom out, because this is the series' quietest recurring theme. The
textbook presentation of an adaptive algorithm is the update rule; everything
else is "implementation detail." The measured record says otherwise: on this
one small struct, the *normaliser* was the difference between CRC 0 and a 2×
win, and the *guard* is what makes an always-on deployment survivable. The
same pattern reappears in Part 11's diversity calibrator — where a rejected
measurement window **holds** the previous gains rather than falling back to
passthrough, because the fallback *is itself* a phase step, and where the
reference branch is pinned rather than re-elected per datagram. Different
algorithm, same discipline: decide what must stay constant, decide what
failure falls back to, and test both decisions as first-class behaviour.
An adaptive algorithm without its guards isn't a leaner version of the same
algorithm. It's a different, worse algorithm that usually behaves the same.

### How that principle shaped the Go code

- **References are state, so `Reset` owns them.** `cumSum`/`count` reset with
  the taps — a channel estimate and its scale estimate live and die together,
  never spanning a retune.
- **The guard is in the hot loop, not a supervisor.** Divergence is detected
  the same sample it happens (max tap magnitude tracked inside the update
  walk), because a guard that runs "periodically" grants garbage a grace
  period — and the next snapshot might land inside it.
- **Every safety property has a regression.** No-harm on clean signals,
  recovery on ISI, byte-identical off-state — the tests treat "does not make
  things worse" as a feature with the same standing as "makes things better."

## Where this goes next

Blind CMA is now complete: cost, snapshot, references, guards — a lever that
needs nothing from the signal but its envelope. But TETRA bursts *do* carry
something better: a known training sequence at a known position, transmitted
in the clear in every normal burst.
[Part 7]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})
takes the step from blind to trained: LMS on the midamble — train on the
known symbols, freeze per burst, re-derive the soft decisions from equalized
symbols — and the architectural consequence Part 3 foretold: the raw-symbol
plumbing that had to be built before any of it could run.

## FAQ

**Why not just use the receiver's AGC as the normaliser?**
The AGC serves the demodulator and settles at its own time constant; nothing
guarantees the scale it delivers is constant on the timescale CMA's
equilibrium needs, and coupling the equalizer's reference frame to another
loop's dynamics invites exactly the interaction the EMA trap demonstrated.
`SnapshotCMA` owning its normaliser makes its correctness self-contained.

**Doesn't the cumulative mean go stale if the signal level genuinely changes?**
Slowly, yes — by design. A genuine sustained level change (retune, gain
change) should arrive with a `Reset` from the pipeline anyway; within one
locked stream, slow staleness only means the effective μ drifts modestly,
which the guard bounds in the worst case. The failure mode of the *responsive*
alternative was total; the failure mode of the sluggish one is mild and
bounded. Easy trade.

**How was the EMA failure actually diagnosed?**
By the series' standing rule: yield. The equalizer "worked" by its own
telemetry (cost decreasing per slot), but the capture A/B showed CRC 0 with
the EMA and the full win with a global-RMS normalise — same taps, same μ, same
capture. Once the divisor was the only difference between the arms, the moving
target explanation followed and the cumulative mean formalised it.

**Why re-seed to pass-through instead of the last-good taps?**
Because "last-good" is a guess about *why* the blow-up happened. If the
channel genuinely changed, last-good taps are a confident wrong inverse; the
wire is never confidently wrong, and `wApply` is still carrying the last
snapshot regardless. Neutral recovery costs one re-convergence (a few hundred
symbols at μ = 6e-3); wrong recovery can cost the stream.

**Do other adaptive pieces of GopherTrunk follow this pattern?**
Yes, deliberately. The diversity `TrackingCalibrator` (Part 11) holds gains on
rejected windows and pins its reference branch; the autotune manager rejects
implausible AFC measurements before averaging and gates its correction behind
a warm-up. The shared shape — constant references, bounded failure, tested
guards — is house style, learned mostly the expensive way.

## Series navigation

**Part 6 of 14** · ←
[Part 5: The Snapshot Trick — Frozen Taps & Differential Decoders]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }})
· Next →
[Part 7: Trained Equalization — LMS on the Midamble]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})
