---
title: "Weak-Signal Engineering, Part 10: Diversity I — MRC & Coherence-Gated Calibration"
description: "Maximal-ratio combining from two receive branches, the wideband-scalar caveat that bounds what one complex gain can deliver, and the scale-invariant coherence gate — |rho| = γ/(1+γ), a noise floor you can compute, and the DC-removal detail that turns out to be load-bearing."
category: deep-dives
keywords: maximal ratio combining, diversity combining sdr, coherence gate, normalised cross-correlation, complex gain calibration, dc offset correlator, scale-invariant threshold, dual receiver combining, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, diversity, mrc, coherence, calibration, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 10
---

*Part 10 of **Weak-Signal Engineering**, a 14-part series on decoding the
marginal regime — where the receiver locks but only a fraction of frames
survive. [Part 9]({{ '/blog/deep-dives/weak-signal-engineering-09-parallel-buffers/' | relative_url }})
closed out the single-antenna levers with the architecture that let them land
safely. This part adds hardware: a second receive branch, combined
coherently so the SNRs add. But combining two streams that are *not* seeing
the same signal is worse than useless — it cancels — so the real engineering
is in the gate that decides when calibration can be trusted. That gate is a
normalised cross-correlation, it is scale-invariant by design, and getting
there meant deleting an absolute-power threshold that had already pushed one
operator into a 17 dB gain hike to satisfy a software constant.*

> **TL;DR:** GopherTrunk's `diversity.MRC` combines branches as
> `y = Σ conj(h_k)·x_k / Σ|h_k|²` — in pilot mode the complex gains `h_k` come
> from a calibrator, and the output SNR of a correct combine is the **sum** of
> the branch SNRs. The calibration question — "are these two receivers seeing
> the same signal through a constant complex gain?" — is answered by
> `diversity.CrossStats`: a single-pass, multi-chunk accumulator whose
> `Coherence()` is the normalised cross-correlation |rho| and whose `Gain()`
> is the least-squares `h = cov(x,ref)/var(ref)`. |rho| = γ/(1+γ) at equal
> per-branch SNR γ makes thresholds interpretable (0.5 ≈ 0 dB, 0.35 ≈
> −2.7 dB) against a noise-only floor near sqrt(π/4N). **Every statistic is a
> DC-removed covariance** — an uncentred correlator on two noise branches
> sharing LO-leakage DC reports |rho| → 1 and calibrates confidently on
> nothing (`TestCrossStatsRejectsCommonDCOffset`). Absolute power survives
> only as a −100 dBFS digitally-dead-branch reject.

**Key takeaways**

- **MRC is optimal only when the gains are right — and destructive when they
  are wrong.** Two equal-power branches combined in anti-phase cancel. The
  combiner is the easy part; the calibration gate is the product.
- **Gate on coherence, never on absolute power.** An operator raised
  front-end gain 65.0 → 82.0 dB to clear a −40 dBFS calibration gate by
  0.2 dB. Any dBFS threshold is a gain-staging trap that re-fires on the next
  front end; |rho| asks the actual question and is immune to gain.
- **DC removal in the correlator is load-bearing, not hygiene.** Shared LO
  leakage is perfectly common-mode; uncentred, it makes independent noise look
  perfectly correlated exactly in the weak-signal regime the gate protects.
- **One scalar gain serves the whole wideband stream — know the caveat.**
  Antennas metres apart give each carrier its own phase; a scalar aligns the
  loudest and can partially cancel others. Coherence stuck around 0.3–0.5 is
  that signature, and per-channel combining is honestly not built.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Combiner | `y = Σ conj(h_k)x_k / Σ|h_k|²`, power or pilot mode | `internal/dsp/diversity/mrc.go` (`MRC`, `SetGain`) |
| Statistics | single-pass DC-removed covariances across chunks | `internal/dsp/diversity/crossstats.go` (`CrossStats.Accumulate`) |
| Quality gate | normalised cross-correlation \|rho\| in [0,1] | `crossstats.go` (`Coherence`) |
| Gain estimate | least-squares `h = cov(x,ref)/var(ref)` | `crossstats.go` (`Gain`) |
| Dead-branch reject | −100 dBFS linear AC-power floor | `internal/dsp/diversity/tracking.go` (`TrackingOptions.MinBranchPower`) |
| DC-offset pin | shared DC over independent noise ⇒ \|rho\| ≤ 0.05 | `crossstats_test.go` (`TestCrossStatsRejectsCommonDCOffset`) |
| Operator surface | per-branch health line, WARNs that name the fix | `internal/sdr/soapyremote/mrc.go` (`diversityReporter.observe`) |

## In this post

- **Why a second antenna helps** — and how MRC spends it.
- **The wideband-scalar caveat** — the honest bound on one complex gain.
- **Coherence, the scale-invariant gate** — |rho|, its SNR mapping, its floor.
- **The gain-staging trap** — the −40 dBFS gate and why it had to die.
- **DC removal is load-bearing** — the failure the centred correlator prevents.

## Why a second antenna helps — and how MRC spends it

Fading is not uniform in space. Two antennas a few wavelengths apart see
partially independent channels: when one sits in a multipath null, the other
usually doesn't. A selection combiner (`diversity.Selection`) picks the
stronger branch and buys you the better of two rolls. Maximal-ratio combining
does better: weight each branch by the conjugate of its complex channel gain,
sum, and — in AWGN with correct gains — the output SNR is the *sum* of the
branch SNRs. Two equal branches: +3 dB. On the yield cliff of
[Part 1]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }}),
3 dB is a different regime.

```go
// internal/dsp/diversity/mrc.go (shape) — pilot-mode combine
//   y[i] = ( sum_k  conj(h_k) · x_k[i] ) / sum_k |h_k|^2
if m.usePilot {
    for k := 0; k < m.branches; k++ {
        h := m.gains[k]
        weights[k] = complex(real(h), -imag(h)) // conj(h)
        denom += float64(real(h))*float64(real(h)) + float64(imag(h))*float64(imag(h))
    }
}
```

The conjugate is the whole trick: multiplying branch k by `conj(h_k)` rotates
its signal component back into phase alignment with every other branch, so
signals add coherently (amplitudes sum) while the independent noises add
incoherently (powers sum). That is also the failure mode in one sentence:
**get the phase wrong by 180° and the same machinery subtracts your signal.**
The docstring in `mrc.go` is explicit that the combiner is "sensitive to
phase mis-alignment: with two equal-power branches in anti-phase, MRC cancels
the signal." Everything else in this post exists to make sure the `h_k`
handed to `SetGain` deserve to be trusted.

## The wideband-scalar caveat

Before the math, the honest scope. GopherTrunk combines the **wideband
stream** — the full multi-megahertz capture, before any per-channel DDC — so
one complex scalar per branch has to serve every carrier in the band at once.
That is exact only if the branches differ by a frequency-flat constant:
same antenna feed split two ways, or co-located antennas into a
frequency-locked front end.

Two antennas metres apart break the assumption. Each carrier arrives from its
own direction, so each gets its own inter-antenna phase difference set by
geometry; a single scalar aligns whichever carrier dominates the estimate and
*partially cancels* others. The signature is a coherence that no amount of
tracking improves — stuck around 0.3–0.5 while both branches are healthy.
The fix for that regime would be combining after the per-channel DDC, one
gain per narrowband channel — a much larger change, and not built. The
`diversityReporter` health line says as much in its WARN text: widely
separated antennas give each carrier its own phase, which one wideband
complex gain cannot align. Surfacing that number instead of silently
underperforming is arguably the bigger contribution — the operator-seat view
of these log lines lives in the concurrent
[Analog Edge series]({{ '/blog/series/analog-edge/' | relative_url }}).

## Coherence: the scale-invariant gate

The question a calibrator must answer is not "is the signal strong?" but
"are these two branches related by a constant complex gain?" The statistic
that answers it is the normalised cross-correlation

|rho| = |Σ x·conj(ref)| / sqrt(Σ|ref|² · Σ|x|²)

computed over DC-removed samples. `CrossStats` accumulates the six needed
sums in a single pass across arbitrarily many stream chunks, so a
calibration window can span many datagrams:

```go
// internal/dsp/diversity/crossstats.go (shape)
type CrossStats struct {
    n                int
    sumRefR, sumRefI float64 // Σ ref
    sumXR, sumXI     float64 // Σ x
    sumRefRef        float64 // Σ |ref|²
    sumXX            float64 // Σ |x|²
    sumXRefR         float64 // Re Σ x·conj(ref)
    sumXRefI         float64 // Im Σ x·conj(ref)
}

func (s *CrossStats) Coherence() float64 // |rho| in [0,1]
func (s *CrossStats) Gain() (complex64, bool) // h = cov(x,ref)/var(ref)
```

Three properties make |rho| the right gate. **It is scale-invariant** —
multiply either branch by any gain and it does not move, so no front-end
knob can game it. **It maps to SNR**: for a common signal in independent
noise at equal per-branch SNR γ, |rho| = γ/(1+γ), so a threshold of 0.5
means "0 dB per branch" and 0.35 means "about −2.7 dB" — thresholds you can
reason about instead of tune (`TestCrossStatsCoherenceTracksSNR` pins the
mapping). **It has a computable floor**: two branches of pure independent
noise over N samples sit near sqrt(π/4N) — about 0.03 at N = 1000 — so you
know how much coherence is "none" for any window length
(`TestCrossStatsIndependentNoiseFloor`). And the same accumulation yields the
least-squares gain `h = cov(x,ref)/var(ref)`, so the gate and the estimate it
gates come from one measurement of one window.

> **Update (18 Aug):** the thresholds themselves are no longer fixed
> constants. Wideband |rho| is diluted by noise-only bandwidth around the
> coherent carrier, so a fixed 0.5 turned out to be a bandwidth-staging trap
> of the same species as the −40 dBFS gate below — an operator raised RF gain
> 5 dB purely to clear it. The gates now bound the estimate's projected phase
> error `sqrt((1−ρ²)/(2Nρ²))`, which puts the minimum |rho| at ~8× (lock) /
> ~5× (track) the sqrt(π/4N) floor for whatever window length the stream
> runs. The figure below still shows the γ/(1+γ) mapping with the original
> fixed thresholds for illustration.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="A curve of coherence magnitude rho against per-branch SNR in dB, rising from near zero through 0.35 at about minus 2.7 dB and 0.5 at 0 dB toward 1 at high SNR. Two horizontal threshold lines mark the track gate at 0.35 and the lock gate at 0.5, and a shaded band near the bottom marks the noise-only floor around square root of pi over 4N.">
  <line x1="60" y1="20" x2="60" y2="180" stroke="var(--fg-muted)"/>
  <line x1="60" y1="180" x2="640" y2="180" stroke="var(--fg-muted)"/>
  <text x="30" y="30" fill="var(--fg-muted)" font-size="9">|rho|</text>
  <text x="30" y="44" fill="var(--fg-muted)" font-size="9">1.0</text>
  <text x="600" y="196" fill="var(--fg-muted)" font-size="9">per-branch SNR</text>
  <text x="140" y="196" fill="var(--fg-muted)" font-size="9">−10 dB</text>
  <text x="330" y="196" fill="var(--fg-muted)" font-size="9">0 dB</text>
  <text x="480" y="196" fill="var(--fg-muted)" font-size="9">+10 dB</text>
  <polyline points="70,172 120,166 170,152 220,136 270,118 330,104 390,78 450,58 510,46 570,40 630,37" fill="none" stroke="currentColor"/>
  <rect x="60" y="168" width="580" height="12" fill="var(--fg-muted)" opacity="0.25"/>
  <text x="470" y="176" fill="var(--fg-muted)" font-size="8">noise-only floor ≈ sqrt(π/4N)</text>
  <line x1="60" y1="104" x2="640" y2="104" stroke="var(--accent)" stroke-dasharray="5 4"/>
  <text x="66" y="98" fill="var(--accent)" font-size="9">lock gate 0.5 (= 0 dB): |rho| = γ/(1+γ)</text>
  <line x1="60" y1="126" x2="640" y2="126" stroke="var(--accent)" stroke-dasharray="2 4"/>
  <text x="66" y="140" fill="var(--accent)" font-size="9">track gate 0.35 (≈ −2.7 dB)</text>
  <circle cx="330" cy="104" r="4" fill="var(--accent)"/>
  <text x="350" y="214" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the gate reads SNR off the stream itself — no front-end gain setting can move it</text>
</svg>
<figcaption>|rho| = γ/(1+γ) turns coherence thresholds into SNR statements: 0.5 is 0 dB per branch, 0.35 is −2.7 dB, and the noise-only floor is computable from the window length alone.</figcaption>
</figure>

## The gain-staging trap the gate replaced

The gate was not always coherence. An earlier design refused to calibrate
until the reference branch cleared **−40 dBFS** — a sane-sounding "make sure
there's signal" check. Then a field report showed what any absolute-power
gate eventually does: an operator raised front-end gain from 65.0 to 82.0 dB
— a 17 dB hike chosen not for RF reasons but to push a number past a software
constant — and landed at −39.8 dBFS, clearing the gate by 0.2 dB. The radio
was fine; the threshold was the problem, and it would have re-fired on the
next front end with a different full-scale mapping. This is the same lesson
the [Analog Edge series]({{ '/blog/series/analog-edge/' | relative_url }})
teaches from the operator's seat: never chase a software threshold with a
gain knob.

Absolute power survives in exactly one place, doing the one job it is fit
for: `MinBranchPower` defaults to 1e-10 — **−100 dBFS**, where one CS16 LSB
is −90.3 dBFS — and rejects only a *digitally dead* branch: all zeros, a
receiver that never digitised, a server that delivered one channel of a
two-channel request. There |rho| is 0/0 and genuinely has no opinion. The
docstring is explicit that this is deliberately far below any signal level,
"not a signal-presence test, because an absolute power threshold is exactly
the gain-staging trap this design exists to remove."

## DC removal is load-bearing

The subtlest line in `crossstats.go` is the one that makes every statistic a
*centred* covariance rather than a raw sum. It reads like numerical hygiene.
It is not — it is the difference between a gate and a hazard.

Both receivers of a shared front end carry LO leakage and converter DC
offset, and that offset is perfectly **common-mode**: identical on both
branches, always. Feed two branches of pure independent noise plus a common
DC into an *uncentred* correlator and the DC term dominates every sum — the
correlator reports |rho| → 1 and hands back `h = dc1/dc0`. The system would
conclude the branches are perfectly coherent, "calibrate" on the ratio of
two leakage terms, and start coherently combining noise — precisely in the
weak-signal regime where the signal is too small to out-vote the DC, which
is precisely the regime the gate exists to protect.

`TestCrossStatsRejectsCommonDCOffset` pins the defence with a hostile
fixture: two branches of independent noise sharing a DC offset **20× the
noise amplitude** must read |rho| ≤ 0.05, and the branch AC power must read
~the noise power, not the ~544× figure the DC would contribute. The cost of
the fix is two extra running sums per branch (the per-branch means), folded
into the same single pass. `RefPower` and `BranchPower` are AC powers for
the same reason: a branch carrying nothing but DC reads zero, "which is the
correct reading for a receiver that is not delivering signal."

### How that principle shaped the Go code

- **One accumulator answers both questions.** `Gain()` and `Coherence()`
  derive from the same six sums over the same window, so the estimate can
  never be graded by a different measurement than produced it.
- **Chunk-tolerant by construction.** `Accumulate` folds arbitrary-length
  chunks and truncates a ragged pair to the shorter length, so the window
  schedule is owned by the caller and a torn final datagram skews nothing.
- **The thresholds carry their own justification.** 0.5 and 0.35 are not
  tuned magic — they are γ/(1+γ) statements pinned by a test, which is why
  they can be documented as dB and reasoned about at 2 a.m.

## Where this goes next

`CrossStats` grades one window. The harder question is time: on front ends
with independent PLLs the true branch phase *walks*, so a constant measured
once decays — but Part 5 taught that a continuously-adapting filter ahead of
a differential decoder is fatal. 
[Part 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }})
resolves the apparent contradiction: why `TrackingCalibrator` is structurally
safe where CMA was not, the hold-don't-fallback rule, the step clamp, and the
capture A/B — four decode arms scored by CRC-clean BSCH — that decides
whether tracking earns default-on.

## FAQ

**How much does a second branch actually buy?**
With correct gains in AWGN, the combined SNR is the sum of branch SNRs — +3 dB
for two equal branches, more when the branches fade independently and one
would otherwise be in a null. With *wrong* gains it can cost everything: MRC
in anti-phase cancels. That asymmetry is why the gate is conservative and why
an uncalibrated combiner passes the reference branch through untouched.

**Why does GopherTrunk combine wideband instead of per channel?**
The combiner lives in the SDR driver, ahead of channelisation, so every
downstream consumer benefits without knowing diversity exists. The price is
the frequency-flat assumption — one scalar per branch. Co-located antennas on
a locked front end fit it; widely separated antennas don't, and the coherence
figure is what tells you which situation you are in.

**What does a coherence stuck at 0.3–0.5 mean when both branches look healthy?**
Usually the wideband-scalar caveat: the branches see the same band but with
per-carrier phase differences one scalar cannot align — separated antennas,
or mixed polarisations. It is not a gain problem; the health-line WARN says
so explicitly, because raising RF gain is the reflex that never helps here.

**Could the DC problem be solved by a DC-blocking filter before the correlator?**
A high-pass helps the stream, but the correlator cannot assume every caller
runs one, and residual DC after an imperfect block still biases a raw
correlator. Centring inside `CrossStats` costs two sums and makes the
statistic correct regardless of what upstream did — an invariant, not a
configuration.

**Is selection diversity ever the better choice?**
Selection (`diversity.Selection`) needs no phase calibration at all, so it is
immune to every failure mode in this post — at the cost of the coherent gain.
When coherence gates keep holding on your hardware, the better of two
branches beats a combine you cannot trust.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Parallel Buffers — SymbolSink, SoftSink & Opt-In Soft Paths]({{ '/blog/deep-dives/weak-signal-engineering-09-parallel-buffers/' | relative_url }})
· Next →
[Part 11: Diversity II — Tracking Without Breaking the Differential]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }})
