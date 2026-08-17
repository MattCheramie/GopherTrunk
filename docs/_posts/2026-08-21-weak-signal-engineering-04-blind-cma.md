---
title: "Weak-Signal Engineering, Part 4: Blind Equalization — CMA From First Principles"
description: The Constant Modulus Algorithm built from its cost function up — how the Godard gradient becomes a five-line tap update in GopherTrunk's cma.go, why constant-envelope PSK is the perfect prey, why the algorithm needs a noise floor to be well-posed, and where the spurious minima behind Part 2's EVM trap come from.
category: deep-dives
keywords: constant modulus algorithm, cma equalizer, blind equalization, godard cost function, stochastic gradient descent dsp, adaptive fir taps, pi/4-dqpsk equalizer, spurious minima, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, cma, equalizer, dsp, adaptive-filters, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 4
---

*Part 4 of **Weak-Signal Engineering**, a 14-part deep dive into decoding the
marginal regime, where a receiver locks but under-decodes.
[Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})
established that the smear on our thread capture is a convolution — invertible
by another filter, if only we could find it. The catch: on a live control
channel there's no training sequence conveniently aligned for us, no pilot, no
known symbols at all from the receiver's point of view. This part is about
finding the inverse filter *blind* — using nothing but a statistical property
the transmitted signal is known to have — and about the exact price that
bargain exacts, which Parts 5 and 6 then pay down.*

> **TL;DR:** PSK-family signals leave the transmitter with **constant
> modulus** — every symbol on one circle. A dispersive channel destroys that
> property; the **Constant Modulus Algorithm** exploits it by adapting FIR taps
> to minimise the Godard cost **`J = E[(|y|² − R²)²]`** by stochastic gradient
> descent: `w ← w − μ·(|y|²−R²)·y·conj(x)`, five lines in
> `internal/dsp/equalizer/cma.go`. It is genuinely blind — no reference, no
> framing — which is why it can run inside a receiver that hasn't synchronised
> yet. The costs: the cost function never mentions *correctness* (spurious
> constant-modulus minima — Part 2's EVM trap is one), its output **phase is
> unconstrained** (fatal before a differential decoder — Part 5), the update
> scales with **|x|³** so normalisation is load-bearing (Part 6), and a
> literally noise-free constant-modulus input is a **degenerate case** — which
> is why GopherTrunk's synthetic equalizer tests deliberately add noise.

**Key takeaways**

- **CMA needs to know one thing about the signal, and PSK guarantees it.**
  Constant envelope is a property of the *modulation*, known before any
  decode. That's what makes blind operation possible on an unsynchronised
  channel.
- **The update is the cost's own gradient, one sample at a time.** No matrix
  inversions, no block processing: an 11-tap FIR and a rank-1 update per
  symbol. Cheap enough to leave on permanently.
- **Center-spike initialisation makes it fail-safe.** Starting as a
  pass-through means an unconverged (or unneeded) equalizer is a wire, not a
  hazard — the "no regression on clean captures" half of every result.
- **Blindness has structural costs, not tuning costs.** Spurious minima,
  phase indifference, and scale sensitivity are properties of the cost
  function itself. The next two parts are engineering around them, not
  parameter-tweaking them away.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The classic algorithm | Godard/CMA-2 cost, gradient, caveats | `internal/dsp/equalizer/cma.go` (`CMA`, `NewCMA`) |
| Per-sample update | FIR output + `w −= μ·err·y·conj(x)` | `cma.go` (`Process`) |
| Differential-safe wrapper | adapt always, apply frozen snapshots | `internal/dsp/equalizer/snapshot_cma.go` (`SnapshotCMA`) — Part 5 |
| Failing-first ISI regression | multipath channel: hard decode fails raw, passes equalized | `snapshot_cma_test.go` (`TestSnapshotCMARecoversISIChannel`) |
| No-harm guard | clean signal in ⇒ essentially unchanged out | `snapshot_cma_test.go` (`TestSnapshotCMAHarmlessOnCleanSignal`) |
| Concept reference | CMA in the reference library | [`/reference/cma-equalizer/`]({{ '/reference/cma-equalizer/' | relative_url }}), [`/reference/adaptive-filter/`]({{ '/reference/adaptive-filter/' | relative_url }}) |

## In this post

- **The property the channel can't hide** — constant modulus as prior knowledge.
- **From cost to update rule** — deriving the five lines in `cma.go`.
- **Reading the real implementation** — history ring, error proxy, center spike.
- **The fine print** — spurious minima, and why blind can mean confidently wrong.
- **The degenerate case** — why the synthetic tests add noise on purpose.

## The property the channel can't hide

Everything blind equalization knows about the world fits in one sentence: *the
transmitter emitted symbols of constant magnitude.* For
[π/4-DQPSK]({{ '/reference/pi-4-dqpsk/' | relative_url }}), QPSK, and the rest
of the PSK family, information rides entirely in phase; every transmitted
symbol sits on one circle of radius R. A dispersive channel — Part 3's
convolution — sums each symbol with scaled, rotated echoes of its neighbours,
and a sum of vectors of fixed length has *variable* length. Magnitude spread
at the receiver is therefore not noise-like bad luck; it is a **measurement of
the ISI**. Squeeze the output magnitudes back onto one circle and you have —
up to the fine print below — undone the mixing.

That inversion of perspective is the elegant part: the modulation's most
boring property (we deliberately put no information in amplitude) becomes the
observable that lets a receiver learn the channel with no cooperation from the
transmitter, no frame sync, and no decoded bits. Which is exactly what a
control-channel receiver needs, since the equalizer has to help *before*
decoding succeeds — on the thread capture, decoding mostly *isn't* succeeding.

## From cost to update rule

Formalise "squeeze onto one circle" and the algorithm falls out. Let `y[n]`
be the equalizer output — an FIR over the received samples,
`y = Σ_k w_k·x[n−k]`. Godard's CMA-2 cost penalises squared deviation of the
squared modulus from a target `R²`:

`J = E[(|y|² − R²)²]`

Differentiate with respect to the conjugate taps (the Wirtinger convention
that `lms.go`'s comments walk through for the trained case) and the gradient
comes out proportional to `(|y|² − R²)·y·conj(x)`. Descend it one sample at a
time — stochastic gradient, the same move LMS makes — and you get the whole
algorithm. GopherTrunk's `cma.go` states it in its doc comment exactly as the
math reads:

```go
// internal/dsp/equalizer/cma.go (shape) — the contract
// Cost function and gradient (Godard / CMA-2):
//
//	J = E[(|y|^2 - R^2)^2]
//	∂J/∂w*  ∝  (|y|^2 - R^2) · y · conj(x)
//	w[n+1]  =  w[n]  -  μ · (|y|^2 - R^2) · y · conj(x)
//
// Pick R^2 so the equilibrium weight scaling matches the expected
// constellation. For unit-magnitude PSK use R^2 = 1 …
```

Read the update rule's anatomy, because each factor earns its keep.
`(|y|² − R²)` is the *signed* modulus error: outputs outside the circle push
taps one way, inside the other, and on the circle the update vanishes.
`y·conj(x)` steers the correction along the input history — the same
directional term every adaptive FIR uses. And `μ` is the eternal step-size
trade: fast convergence versus a noisy equilibrium. Notice what the rule never
consults: a reference symbol, a decision, a frame boundary. That absence *is*
the blindness — and it's also the door the failure modes walk through.

## Reading the real implementation

`Process` in `cma.go` is the derivation transcribed, plus the practicalities:
a ring buffer for the input history, and manual complex arithmetic so the hot
loop stays allocation-free:

```go
// internal/dsp/equalizer/cma.go (shape) — Process
// Error proxy = |y|^2 - R^2.
mag2 := yr*yr + yi*yi
err := mag2 - c.target

// Weight update: w_k -= μ · err · y · conj(x[n-k]).
mu := c.stepSize
/* … walk the history ring … */
ur := yr*xr + yi*xi        // y·conj(x), real part
ui := yi*xr - yr*xi        //            imag part
c.taps[i] = complex(tr-mu*err*ur, ti-mu*err*ui)
```

Two constructor-time choices matter more than any parameter. First,
**center-spike initialisation**: `NewCMA` sets `taps[N/2] = 1` with all others
zero, so tap zero state is an identity filter with a fixed delay. An
unconverged equalizer passes the signal through untouched — which is why
`TestSnapshotCMAHarmlessOnCleanSignal` can demand a clean channel stay clean,
and why enabling the equalizer fleet-wide was a defensible default rather than
a gamble. Second, `Process` returns the error proxy `|y|²−R²` alongside the
output: watch it settle toward zero and you are watching convergence, no
decoded bits required.

One more block in `cma.go` deserves a mention because it previews Part 5's
theme. The constant-modulus cost is invariant to rotating *all* taps by a
common phase — rotate every `w_k` by `e^{jθ}` and `|y|` doesn't change — so
the gradient has a null direction and the tap phase random-walks, driven by
noise. `cma.go` handles the symptom that bit first (a downstream carrier loop
integrating the drift, [#492](https://github.com/MattCheramie/GopherTrunk/issues/492))
by re-anchoring the centre tap to the positive real axis after each update.
That keeps the output phase *slowly varying* — good enough for a coherent
loop, and, as Part 5 will show in detail, *not* good enough for a
differential decoder.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Three constellations showing CMA converging. Left: received symbols scattered at many radii around the unit circle, labeled ISI in, cost J large. Middle: partially converged, symbols pulled toward the circle. Right: converged, symbols sitting on the circle, cost near zero — with a note that the ring's rotation is unconstrained. Beneath, an arrow labeled per-sample tap updates, w minus mu times modulus error times y conj x, spans the three panels.">
  <circle cx="115" cy="90" r="52" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <g fill="currentColor">
    <circle cx="140" cy="60" r="3"/><circle cx="90" cy="52" r="3"/><circle cx="150" cy="110" r="3"/><circle cx="82" cy="122" r="3"/>
    <circle cx="115" cy="90" r="3"/><circle cx="170" cy="82" r="3"/><circle cx="62" cy="86" r="3"/><circle cx="126" cy="146" r="3"/>
    <circle cx="103" cy="30" r="3"/><circle cx="155" cy="140" r="3"/>
  </g>
  <text x="115" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ISI in: |y| spread wide, J large</text>
  <circle cx="340" cy="90" r="52" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <g fill="currentColor">
    <circle cx="388" cy="76" r="3"/><circle cx="378" cy="122" r="3"/><circle cx="340" cy="40" r="3"/><circle cx="296" cy="66" r="3"/>
    <circle cx="290" cy="106" r="3"/><circle cx="318" cy="136" r="3"/><circle cx="362" cy="46" r="3"/><circle cx="348" cy="140" r="3"/>
  </g>
  <text x="340" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">adapting: pulled toward the circle</text>
  <circle cx="565" cy="90" r="52" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <g fill="var(--accent)">
    <circle cx="617" cy="90" r="3.5"/><circle cx="602" cy="53" r="3.5"/><circle cx="565" cy="38" r="3.5"/><circle cx="528" cy="53" r="3.5"/>
    <circle cx="513" cy="90" r="3.5"/><circle cx="528" cy="127" r="3.5"/><circle cx="565" cy="142" r="3.5"/><circle cx="602" cy="127" r="3.5"/>
  </g>
  <text x="565" y="168" text-anchor="middle" fill="var(--accent)" font-size="9">converged: J ≈ 0 — rotation still free</text>
  <line x1="60" y1="188" x2="620" y2="188" stroke="currentColor"/><polygon points="620,184 630,188 620,192" fill="currentColor"/>
  <text x="340" y="202" text-anchor="middle" fill="var(--fg-muted)" font-size="10">per-sample updates: w ← w − μ·(|y|²−R²)·y·conj(x) — no reference symbols consulted, ever</text>
</svg>
<figcaption>CMA drives output magnitudes onto the modulation's known circle using only the modulus error — but nothing in the cost pins the ring's rotation, or guarantees the phases on it are the transmitted ones.</figcaption>
</figure>

## The fine print: spurious minima

Here is where Part 2's trap stops being an anecdote and becomes a theorem-
shaped fact. `J` is a fourth-order surface over the tap space, and it is *not*
convex. Its global minima are the channel inverses we want (at arbitrary
rotation and delay — all equally good). But it also has **local minima that
satisfy constant modulus without inverting the channel** — tap vectors whose
output sits beautifully on a circle while remaining a scrambled function of
the transmitted symbols. A finite-length FIR trying to invert a channel it
can't perfectly represent makes more of them, and noise plus a finite step
size lets the descent wander into one and stay.

Sit with what that means operationally: **CMA's own convergence diagnostics
cannot distinguish success from a spurious minimum.** The error proxy settles
near zero either way. EVM improves either way — that is precisely how a
numerically-unstable variant printed 34%→8% while decoding *nothing*. The only
external observer that can tell the difference is the decoder: CRC yield, the
Part 2 verdict, measured through the whole chain. This is also the honest case
for Part 7's trained equalizer: a known reference collapses the ambiguity —
`e = d − y` is only zero at the *right* answer — which is why the midamble
path exists even with a working blind path in production.

## The degenerate case: why the tests add noise

One last subtlety, discovered the practical way when a daemon-level synthetic
TETRA test started exercising the CC equalizer. A synthesized, perfectly
clean, perfectly constant-modulus input is a **degenerate input for CMA**: the
modulus error is zero for the identity filter *and* for a continuum of other
tap vectors that happen to preserve modulus on that pristine signal — the cost
surface goes flat in directions that real signals would penalise, and the taps
drift on numerical noise with nothing to anchor them. Blind CMA is well-defined
only *against a noise floor*.

So GopherTrunk's synthetic fixtures add one on purpose: the receiver-level
clean-channel test runs at 30 dB SNR, and the daemon's synthetic TETRA CC test
adds 40 dB AWGN — not to make the tests harder, but to make the algorithm's
problem well-posed. It's a pleasing inversion of the usual instinct, and a
small instance of the self-consistent-synthetic trap this project keeps
guarding against: a test input can be *too* ideal to represent the air, and
"add realistic impairment" is sometimes a correctness requirement, not
pessimism.

## Where this goes next

We now have a filter that finds the channel inverse blind — and a cost
function that is provably indifferent to output rotation. Between snapshots of
adaptation, that indifference is fatal to exactly the decoder TETRA uses.
[Part 5]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }})
is the snapshot trick: why a continuously-adapting CMA ahead of a differential
decoder measures CRC **zero**, why frozen taps alone measure baseline, and how
`SnapshotCMA`'s adapt-continuously-apply-frozen design gets both — taking the
thread capture from ~12% to ~100% BSCH in the process.

## FAQ

**How many taps, and what step size, does GopherTrunk actually run?**
The TETRA receiver's defaults are 11 taps, μ = 6e-3, refreshing the applied
snapshot every 200 symbols (`DefaultEqualizerTaps` / `DefaultEqualizerMu` /
`DefaultEqualizerSnapshot` in `internal/radio/tetra/receiver/receiver.go`),
validated on the reporter captures and the synthetic multipath fixtures. At
18k symbols/s, 11 symbol-spaced taps span ±5 symbols of channel memory —
generous for the delay spreads a 25 kHz land-mobile channel serves up.

**Why CMA-2 (squared modulus) instead of penalising |y| − R directly?**
The squared form makes the cost a smooth polynomial in the taps — the gradient
is algebraic (`err·y·conj(x)`), with no square roots or divisions per sample.
The classic CMA(1,2)/CMA(2,2) family trades exactly these smoothness-vs-
robustness properties; CMA-2 is the conventional choice for exactly this
gradient simplicity.

**Does CMA work on C4FM or QAM?**
Not as-is. `cma.go`'s caveat list is explicit: on non-constant-modulus signals
(FM-family, OQPSK, QAM) the cost isn't zero at the right answer. C4FM is
four-level FSK demodulated through a discriminator — a different signal model,
and part of why P25 Phase 1's equalizer story (Part 13) isn't a copy-paste of
TETRA's. QAM needs modified costs or decision-directed
[LMS]({{ '/reference/lms-algorithm/' | relative_url }}).

**Could I detect a spurious minimum without running the decoder?**
Not reliably from inside the equalizer — that's the sober conclusion of the
34%→8% incident. Modulus error, EVM, even constellation appearance are all
satisfied at spurious minima. Structural defenses exist (center-spike restarts
make you start at the identity, Part 6's guards re-seed on divergence), but
verification is external by nature: decode to CRC.

**If blind equalization is this fragile, why not always train?**
Training needs known symbols at known positions — which requires burst framing,
which on a control channel you're still *acquiring* may not exist yet. Blind
CMA works pre-sync, streaming, with no framing at all; that's an operating
regime training can't reach. The two are complementary, not competing: blind
in the receiver (Parts 5–6), trained in the extractor where midambles are
locatable (Part 7).

## Series navigation

**Part 4 of 14** · ←
[Part 3: ISI & the Linear Channel — What an Equalizer Can & Can't Fix]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})
· Next →
[Part 5: The Snapshot Trick — Frozen Taps & Differential Decoders]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }})
