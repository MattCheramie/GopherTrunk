---
title: "Weak-Signal Engineering, Part 5: The Snapshot Trick — Frozen Taps & Differential Decoders"
description: Why a continuously-adapting CMA in front of a differential decoder scores exactly zero CRC-valid frames, why permanently frozen taps only ever match the baseline, and how SnapshotCMA's adapt-continuously-apply-frozen design resolves the dilemma — doubling TETRA voice yield and lifting the thread capture from twelve percent to one hundred.
category: deep-dives
keywords: snapshot cma, frozen equalizer taps, differential decoder phase, cma rotation invariance, pi/4-dqpsk differential decode, adaptive equalizer phase wander, tetra equalizer yield, blind equalizer differential safe, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, cma, equalizer, tetra, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 5
---

*Part 5 of **Weak-Signal Engineering**, a 14-part deep dive into decoding the
marginal regime, where a receiver locks but under-decodes.
[Part 4]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
built the Constant Modulus Algorithm and left one loose end swinging: CMA's
cost is rotation-invariant, so nothing constrains its output phase while the
taps adapt. This part is where that loose end meets TETRA's differential
decoder — and where the naive combination measures exactly zero. The fix is a
single structural idea, small enough to state in one sentence and consequential
enough that it reappears, in different clothes, in Parts 7 and 11: adapt
continuously, but apply a frozen snapshot. It is also the part where the
thread capture finally moves.*

> **TL;DR:** CMA's cost `J = E[(|y|²−R²)²]` doesn't change if every tap
> rotates by a common phase, so an adapting CMA's output phase **wanders** —
> and a time-varying phase does **not** cancel in the differential product
> `s[n]·conj(s[n−1])`. Measured on real captures: streaming-adaptive CMA ahead
> of the π/4-DQPSK differential decoder → **CRC 0**; taps frozen forever →
> baseline (a stale inverse for a moving channel). **`SnapshotCMA`**
> (`internal/dsp/equalizer/snapshot_cma.go`) keeps both tap sets: `wAdapt`
> updates every sample, `wApply` — the filter actually applied — is a snapshot
> of `wAdapt` refreshed every `snapEvery` symbols (default 200, on the order of
> a 255-symbol TETRA burst). Between snapshots the filter is constant, so it
> imposes only a constant phase — which the differential cancels; the one
> straddling symbol per snapshot is absorbed by the FEC. Results: soft-decision
> TCH/S bursts **410 → 778** (~1.9×) across six captures, and the thread
> capture's CRC-clean BSCH from **~12% to ~100%**.

**Key takeaways**

- **The failure is structural, not a tuning problem.** No step size makes an
  adapting filter phase-safe for a differential decoder — the phase wander is
  a null direction of the cost itself. Slower adaptation just wanders slower.
- **Differential decoding cancels constant phase, and only constant phase.**
  `s·conj(prev)` is immune to any rotation that is the same on both symbols;
  that exact algebraic property is what the frozen snapshot engineers for.
- **Two tap sets, two jobs.** `wAdapt` tracks the channel and is never
  applied; `wApply` filters the signal and never adapts. The design decouples
  "stay current" from "stay coherent."
- **The verdict came from CRC, at every step.** Each arm of the design space —
  streaming, frozen, snapshot — was scored by decoded frames on real captures,
  which is the only reason the middle option's failure (and the third option's
  win) was visible at all.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The differential-safe equalizer | adapt every sample, apply frozen snapshots | `internal/dsp/equalizer/snapshot_cma.go` (`SnapshotCMA.Process`) |
| Plain CMA (the contrast) | applies its live, adapting taps — coherent-slicer use only | `internal/dsp/equalizer/cma.go` (`CMA`) |
| Receiver wiring | equalizer between symbol timing and differential decode | `internal/radio/tetra/receiver/receiver.go` (`Options.EnableEqualizer`) |
| Defaults | 11 taps, μ=6e-3, snapshot every 200 symbols | `receiver.go` (`DefaultEqualizerTaps/Mu/Snapshot`) |
| Failing-first regression | multipath decode fails raw, passes equalized; clean stays clean | `snapshot_cma_test.go`, `receiver/receiver_equalizer_test.go` |
| Thread-capture A/B | one boolean, ~12% vs ~100% BSCH on the fixture | `internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go` |
| Re-sync hygiene | drop the stale channel estimate on re-acquisition | `snapshot_cma.go` (`Reset`) |

## In this post

- **The collision** — rotation invariance meets `s·conj(prev)`.
- **Three arms, three verdicts** — streaming zero, frozen baseline, snapshot win.
- **Inside `SnapshotCMA`** — two tap sets and a schedule.
- **The numbers** — voice 410→778, and the thread capture at ~100%.
- **The pattern beyond CMA** — where adapt-but-apply-frozen shows up next.

## The collision: rotation invariance meets `s·conj(prev)`

Two facts, each innocent alone. Fact one, from Part 4: rotate every CMA tap by
`e^{jθ}` and `|y|` is untouched, so the cost gradient has a null direction
along global phase. While the taps adapt, noise and ISI push them along that
null direction freely — the output constellation's absolute rotation drifts,
sample to sample. A slicer with its own carrier recovery shrugs; carrier loops
exist to track slow rotation.

Fact two: a differential decoder computes `d[n] = s[n]·conj(s[n−1])` and reads
information from the *phase of the product*. Inject a time-varying rotation
`θ[n]` and the product picks up `e^{j(θ[n]−θ[n−1])}` — the *derivative* of the
wander. A constant phase cancels perfectly; a changing phase lands its
per-sample increment directly onto the decision variable. π/4-DQPSK's decision
regions are 90° wide (45° to the nearest wrong decision), and the wander's
increment doesn't need to approach that to be fatal — it biases every single
dibit in a correlated way, and the convolutional decoder, built for scattered
errors, digests a systematic phase bias worst of all. The
`snapshot_cma.go` doc comment records the empirical endpoint bluntly: feeding
a continuously-adapting CMA to a differential decoder "empirically drives the
decode to zero."

Note that `cma.go`'s own centre-tap phase anchor (Part 4,
[#492](https://github.com/MattCheramie/GopherTrunk/issues/492)) doesn't save
you here: it pins the *equilibrium* phase, but every update still moves the
taps — and therefore the output phase — between consecutive symbols. Slow,
bounded wander is fine for a carrier loop and still poison for a differential.

## Three arms, three verdicts

The design space has two obvious corners and both were measured — by CRC
yield, per Part 2, because EVM had already demonstrated it would cheerfully
approve the zero-yield corner:

| Arm | Phase between consecutive symbols | Channel tracking | CRC yield |
|---|---|---|---|
| Streaming-adaptive CMA | changes every sample | perfect | **0** |
| Taps frozen once, forever | constant | none — estimate goes stale | baseline (no win) |
| **Snapshot: adapt always, apply frozen** | constant between snapshots | refreshed every `snapEvery` | **the win** |

The first row is the collision above. The second row fails softer but still
fails: a channel is only *approximately* time-invariant, and an inverse
estimated once decays as multipath geometry shifts — a frozen-at-startup
equalizer converged on the wrong (or an old) channel is just a fixed wrong
filter. The third row threads it: the *applied* filter is constant long enough
for every burst to see a coherent channel, yet the *estimate* underneath never
stops tracking. The one symbol that straddles a snapshot boundary sees a phase
step — deliberately bounded engineering debt, one or two dibits per 200
symbols, and exactly the kind of scattered error the FEC exists to absorb.

## Inside `SnapshotCMA`

The implementation is disarmingly small once the idea is stated. Two tap
vectors and a counter:

```go
// internal/dsp/equalizer/snapshot_cma.go (shape)
type SnapshotCMA struct {
    wAdapt    []complex64 // adapts every sample (phase wanders — never applied directly)
    wApply    []complex64 // frozen snapshot, applied to the output
    buf       []complex64 // normalised input history
    mu        float32     // CMA step size
    snapEvery int         // samples between wApply <- wAdapt snapshots
    /* … cumulative-mean normaliser state — Part 6 … */
}
```

`Process` does both jobs on every sample — the output always comes from the
frozen filter, the adaptation always lands on the tracking filter:

```go
// internal/dsp/equalizer/snapshot_cma.go (shape) — Process
// Output uses the frozen filter (phase-coherent between snapshots).
var y complex64
for k := 0; k < n; k++ { y += e.wApply[k] * e.buf[n-1-k] }

// Adapt the tracking filter by CMA(2,2): e = ya·(|ya|² − 1).
var ya complex64
for k := 0; k < n; k++ { ya += e.wAdapt[k] * e.buf[n-1-k] }
ge := real(ya)*real(ya) + imag(ya)*imag(ya) - 1
/* … w -= mu·e·conj(b), divergence guard — Part 6 … */

if e.since++; e.since >= e.snapEvery {
    copy(e.wApply, e.wAdapt)   // the snapshot
    e.since = 0
}
```

Both vectors start center-spike, so before first convergence the equalizer is
a pass-through — the no-harm property, inherited from Part 4 and pinned by
`TestSnapshotCMAHarmlessOnCleanSignal`. The receiver slots it between
symbol-timing recovery and the differential decoder — before the
nonlinearity, per Part 3's domain rule — behind one flag,
`Options.EnableEqualizer`, with defaults (11 taps, μ = 6e-3,
`snapEvery = 200`) sized so a snapshot interval is on the order of one
255-symbol TETRA burst. And `Reset` exists for a reason worth naming: on a
stream re-sync the channel estimate is stale by definition, and re-seeding to
pass-through beats confidently applying yesterday's inverse to today's
channel.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Timeline comparing output phase under three equalizer strategies. Top trace: streaming-adaptive CMA, phase wanders continuously, every differential corrupted, CRC zero. Middle trace: snapshot design — phase is a staircase, flat and constant across each burst-length interval with a small step at each snapshot refresh, and the differential decoder cancels the flat segments; snapshot instants are marked. Bottom band shows bursts aligned under the flat segments decoding cleanly.">
  <text x="10" y="26" fill="currentColor" font-size="10">streaming-adaptive:</text>
  <path d="M150,30 C190,12 220,52 260,34 C300,16 330,58 380,40 C430,22 460,60 520,38 C560,24 600,52 660,34" fill="none" stroke="currentColor"/>
  <text x="410" y="16" fill="var(--fg-muted)" font-size="9">output phase θ[n] wanders every sample → dθ lands in every differential → CRC 0</text>
  <text x="10" y="104" fill="var(--accent)" font-size="10">snapshot (wApply):</text>
  <polyline points="150,116 270,116 270,104 400,104 400,120 530,120 530,108 660,108" fill="none" stroke="var(--accent)"/>
  <g fill="var(--accent)">
    <circle cx="270" cy="110" r="3"/><circle cx="400" cy="112" r="3"/><circle cx="530" cy="114" r="3"/>
  </g>
  <text x="270" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="9">snapshot</text>
  <text x="400" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="9">snapshot</text>
  <text x="530" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="9">snapshot</text>
  <text x="410" y="140" fill="var(--fg-muted)" font-size="9">constant phase within each interval — cancels in s·conj(prev); one straddling symbol per step</text>
  <g stroke="var(--fg-muted)" fill="none">
    <rect x="160" y="160" width="100" height="26" rx="4"/>
    <rect x="285" y="160" width="100" height="26" rx="4"/>
    <rect x="415" y="160" width="100" height="26" rx="4"/>
    <rect x="545" y="160" width="100" height="26" rx="4"/>
  </g>
  <text x="210" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">burst ✓</text>
  <text x="335" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">burst ✓</text>
  <text x="465" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">burst ✓</text>
  <text x="595" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">burst ✓</text>
  <text x="345" y="206" text-anchor="middle" fill="var(--fg-muted)" font-size="10">snapEvery ≈ a burst: each 255-symbol burst decodes through one constant filter</text>
</svg>
<figcaption>The snapshot turns a continuously-wandering output phase into a staircase: flat across each burst (which the differential cancels), stepping only at refresh instants (which the FEC absorbs).</figcaption>
</figure>

## The numbers

The design earned its place with two yield results, both CRC-scored, both on
real operator captures. On the **voice path** — the garbled same-carrier
recordings that started the whole equalizer effort — inserting `SnapshotCMA`
ahead of the differential decoder roughly doubled CRC-valid TCH/S burst yield
across six captures: soft-decision **410 → 778** (~1.9×), including one call
that went 4 → 207 and another 42 → 134, with no loss on already-clean
captures. On the **control-channel path** — our thread capture — the story
came full circle: the primary single-channel TETRA CC pipeline turned out to
be the one TETRA CC path *not* running the equalizer (the voice composer and
the wideband path already did). Enabling it lifted the marginal fixture from
**~12% to ~100%** CRC-clean BSCH:

```go
// internal/scanner/ccdecoder/pipelines.go (shape) — newTETRAPipeline
// Blind SnapshotCMA equalizer between symbol timing and the
// differential decoder … On the reporter's ~10 dB re-acquisition
// capture it lifts CRC-clean BSCH yield from ~12% to ~100%, which
// is the difference between riding through a marginal dip and
// dropping lock → re-hunt (the ~210 CC transitions/hour symptom).
EnableEqualizer: true,
```

That second number is worth restating in operational terms: at ~12% BSCH the
sync machinery starves, declares the channel lost, and re-hunts — the 210
transitions per hour from Part 1. At ~100%, the same 10 dB channel just…
decodes. The lever didn't add a single dB of signal; it moved the cliff. The
full TETRA-side narrative of both fixes — voice and control channel — is told
protocol-first in the concurrent
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }})
series; here the point is the transferable mechanism.

## The pattern beyond CMA

File the shape of this fix, because the series reuses it twice. The general
statement: **when a downstream stage depends on a property being constant
(here: phase between consecutive symbols), never feed it a filter that is
changing — adapt an estimate on the side and apply it in frozen pieces sized
to the downstream stage's coherence window.** Part 7's trained `SnapshotLMS`
is the same principle with a better teacher: train on a burst's midamble,
freeze, apply to that burst — the snapshot window shrinking from "every 200
symbols" to "exactly one burst." And Part 11's diversity
`TrackingCalibrator` is the instructive *contrast*: it adapts continuously
ahead of the same differential decoder and is *safe*, because its reference
branch pins the output phase structurally — the difference between a cost
that can't see phase and an estimator whose phase is anchored is exactly the
difference between needing the snapshot trick and not.

## Where this goes next

Two loose ends remain inside `Process`: the input normalisation the code
quietly applied before every update, and the guard that re-seeds the tracking
filter when the taps blow up. Both look like hygiene; both turned out to be
the difference between the full win and CRC zero.
[Part 6]({{ '/blog/deep-dives/weak-signal-engineering-06-normalisation-guards/' | relative_url }})
is about why the CMA update's |x|³ scaling makes the normaliser part of the
algorithm, why an EMA that tracks TDMA slot power is a moving target that
converges to garbage, and why guards deserve tests of their own.

## FAQ

**Why not just re-derive the phase per burst instead of freezing taps?**
That's essentially what a coherent receiver's carrier loop does, and it works
for coherent slicers. But TETRA's decode chain is differential by design
(robustness to exactly this class of ambiguity), and bolting a per-burst phase
estimator onto a wandering equalizer adds an estimator — with its own failure
modes — to fix a problem the snapshot removes structurally, for free.

**How was `snapEvery = 200` chosen?**
Jointly with μ, against captures and the synthetic multipath fixtures: long
enough that a 255-symbol burst usually sees one filter (and the straddling
cost is a dibit or two per 200 symbols), short enough that the applied inverse
tracks a slowly-moving channel. The receiver exposes `EqualizerSnapshot` for
tuning, but the default has survived every capture A/B so far.

**Does the snapshot step ever land mid-burst and hurt?**
It can and does — the straddling symbol is real. It is bounded by design: one
symbol per `snapEvery`, carrying a phase step equal to the tap drift since the
last snapshot, and the RCPC/Viterbi layer treats it as an isolated error. The
measured results (410→778, ~12%→~100%) are net of this cost.

**Is plain `CMA` now dead code?**
No — it's the right tool ahead of a *coherent* slicer, where live taps plus
the centre-tap phase anchor are fine and maximal tracking speed helps. The
package deliberately keeps both: `CMA` for coherent chains, `SnapshotCMA` for
differential ones. Using the former where the latter belongs is the
documented, tested, career-limiting move.

**What happens at re-sync or when the scanner retunes?**
The pipeline calls `Reset`, returning both tap sets to center-spike
pass-through and clearing the normaliser. A channel estimate is only valid
for the channel it was learned on; carrying it across a retune would apply a
confident wrong inverse exactly when the receiver is most fragile.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Blind Equalization — CMA From First Principles]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
· Next →
[Part 6: Normalisation & Divergence Guards]({{ '/blog/deep-dives/weak-signal-engineering-06-normalisation-guards/' | relative_url }})
