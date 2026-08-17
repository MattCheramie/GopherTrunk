---
title: "Weak-Signal Engineering, Part 2: Metrics That Lie — EVM vs CRC Yield"
description: Why error vector magnitude, wideband carrier SNR, and constellation beauty can all improve while a decoder recovers nothing — the equalizer whose EVM collapsed 34% to 8% with CRC stuck at zero, the #764 capture whose carrier SNR was higher on the worse file, and the short list of numbers you can actually trust.
category: deep-dives
keywords: evm trap, error vector magnitude, crc yield, decode yield metric, blind equalizer evm, wideband carrier snr, in-channel snr measurement, constellation quality, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, evm, metrics, dsp, tetra, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 2
---

*Part 2 of **Weak-Signal Engineering**, a 14-part deep dive into decoding the
marginal regime — where a receiver locks but under-decodes, and four levers
roughly doubled GopherTrunk's yield.
[Part 1]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }})
defined that regime and introduced the thread capture: a ~10 dB TETRA control
channel that locks and decodes ~12% of its BSCH. Before we touch a single
equalizer tap, this part fixes the measurement problem — because two of the
most natural quality metrics in DSP have already, on this project, pointed
confidently in the wrong direction. If the verdict number is wrong, every
lever after it optimizes the wrong thing.*

> **TL;DR:** **EVM is a trap in front of a blind equalizer, and wideband
> carrier SNR is a trap in front of a phase-noise problem.** A
> numerically-unstable CMA variant once drove differential EVM from **34% to
> 8%** — textbook "constellation opened" — while CRC-valid frames stayed at
> **zero**: blind CMA minimises *modulus error*, not *correctness*, and it has
> spurious constant-modulus minima that look gorgeous and carry nothing. In
> [#764](https://github.com/MattCheramie/GopherTrunk/issues/764), the capture
> that decoded *worse* had the *higher* wideband FFT carrier SNR — carrier-clean
> but modulation-degraded is the signature of reciprocal mixing. The numbers to
> trust, in order: **CRC-valid frames per opportunity** (the verdict), then
> demod-side in-channel SNR (advisory), then everything else (scenery). Rule of
> the series: *never conclude an equalizer helps from EVM; decode to CRC.*

**Key takeaways**

- **Every metric is an answer to a specific question — and usually not yours.**
  EVM answers "how far are symbols from the ideal grid?"; a decoder asks "were
  the bits right?". Those correlate on well-behaved channels and diverge
  exactly when you're doing something interesting.
- **A blind equalizer can optimize the metric instead of the signal.** CMA's
  cost is constant modulus; an output that is perfectly constant-modulus and
  perfectly wrong scores perfectly. The 34%→8%-with-CRC-0 incident is pinned in
  the `SnapshotLMS` doc comment so nobody re-learns it.
- **Where you measure matters as much as what.** #764's wideband FFT said the
  10 MS/s capture was *cleaner*; the in-channel demod said it was 10 dB worse.
  Both were right — about different bandwidths.
- **Yield needs a denominator.** "778 CRC-valid bursts" means nothing without
  the opportunity count; GopherTrunk's harnesses always report `ok` and `fail`
  together, and the ratio is the number.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| EVM, measured properly | reference-grid error vs ideal constellation | Signal Lab VSA — [signal-lab-07]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }}), [`/reference/error-vector-magnitude/`]({{ '/reference/error-vector-magnitude/' | relative_url }}) |
| The EVM-trap record | doc-comment history: EVM 34%→8%, CRC 0 | `internal/dsp/equalizer/snapshot_lms.go` (package doc) |
| What CMA optimizes | error proxy `\|y\|² − R²`, not bit correctness | `internal/dsp/equalizer/cma.go` (`Process`) |
| The verdict harness | CRC-clean BSCH counts, one variable isolated | `internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go` (`decodeCCBSCHYield`) |
| Voice-path verdict | CRC-valid TCH/S per burst opportunity | `internal/radio/tetra/tch.go` (`DecodeTCHSSoft` → `crcOK`) |
| The carrier-SNR trap | wideband-clean, in-channel-degraded (#764) | `internal/scanner/ccdecoder/ddc_highrate_test.go` |

## In this post

- **What EVM actually measures** — and the assumptions riding along.
- **Beauty with zero truth** — the equalizer that polished a corpse.
- **The carrier-SNR trap** — #764's cleaner-but-worse capture.
- **The numbers you can trust** — a short, ordered list.
- **Instrumenting for yield** — what the harnesses report and why.

## What EVM actually measures

[Error vector magnitude]({{ '/reference/error-vector-magnitude/' | relative_url }})
is the RMS distance between your received symbols and the nearest ideal
constellation points, normalised to the constellation's scale. It is a fine
instrument — [Signal Lab's VSA]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }})
computes it, and on a *cooperative* signal it tracks decode quality closely
enough that lab equipment vendors quote sensitivity in EVM.

But look at what the definition smuggles in. EVM assumes the "nearest ideal
point" is the *transmitted* point — that your symbols are merely perturbed, not
relabeled. It assumes the reference grid itself is right: correct phase,
correct scale, correct decision boundaries. On a marginal channel, after an
adaptive stage that is allowed to rotate, scale, and reshape the signal, every
one of those assumptions is up for negotiation. A symbol that lands crisply on
the *wrong* ideal point contributes almost nothing to EVM and everything to the
bit error rate. EVM measures *tidiness*. Decoders consume *identity*.

On a differential modulation like
[π/4-DQPSK]({{ '/reference/pi-4-dqpsk/' | relative_url }}) there's a further
subtlety: you can compute EVM on the differentials (the `s·conj(prev)`
products) rather than the raw symbols, which conveniently cancels any constant
rotation. GopherTrunk's equalizer experiments did exactly that — and it still
lied, which is the story that gives this post its title.

## Beauty with zero truth

The incident is preserved, deliberately, in the doc comment of the *trained*
equalizer — because it is the reason the trained equalizer exists:

```go
// internal/dsp/equalizer/snapshot_lms.go (shape) — package history
// CMA's cost J = E[(|y|²−R²)²] is satisfied by any equalizer output that has
// the right modulus, so it has spurious minima and can converge to a
// constant-modulus but WRONG solution — the failure the package history
// records (a numerically-unstable CMA once drove differential EVM 34%→8%
// while CRC stayed 0). Training to a known reference removes that ambiguity…
```

Unpack what happened. A blind CMA variant — since replaced — was inserted ahead
of the TETRA differential decoder and evaluated the obvious way: measure
differential EVM before and after. Before: 34%, a smeared mess. After: 8%,
which for π/4-DQPSK is a genuinely clean-looking constellation. Anyone
grading on appearance ships that change. But the CRC-valid frame count on the
same capture was **zero**. Not reduced — zero. The equalizer had found an
output that satisfied its cost function beautifully: constant modulus, tight
clusters, and a time-varying relationship to the transmitted symbols that
destroyed every single dibit on its way through the differential decode.

The mechanism has two halves, and the series treats each in its own part.
First, CMA's cost `J = E[(|y|² − R²)²]` genuinely does not mention
correctness — any constant-modulus output is a candidate minimum, including
spurious ones (Part 4). Second, an *adapting* filter in front of a
*differential* decoder injects a time-varying phase that no differential can
cancel (Part 5). Either half alone can make EVM improve while yield dies. The
one-line lesson entered the project's permanent vocabulary: **never conclude an
equalizer helps from EVM; decode to CRC.**

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Two panels compare metrics on the same capture. The left panel shows two constellations: a smeared one labeled before with EVM thirty-four percent, and a tight clean one labeled after with EVM eight percent — the metric says the equalizer helped. The right panel shows the verdict metric: a bar chart of CRC-valid frames, with the baseline bar small but nonzero and the after bar at exactly zero. The pretty constellation decoded nothing.">
  <text x="170" y="18" text-anchor="middle" fill="currentColor" font-size="11">what EVM saw</text>
  <text x="510" y="18" text-anchor="middle" fill="currentColor" font-size="11">what the CRC saw</text>
  <circle cx="90" cy="90" r="52" fill="none" stroke="var(--fg-muted)"/>
  <g fill="var(--fg-muted)">
    <circle cx="122" cy="62" r="3"/><circle cx="105" cy="52" r="3"/><circle cx="66" cy="55" r="3"/><circle cx="55" cy="78" r="3"/>
    <circle cx="52" cy="112" r="3"/><circle cx="78" cy="130" r="3"/><circle cx="115" cy="124" r="3"/><circle cx="128" cy="98" r="3"/>
    <circle cx="98" cy="70" r="3"/><circle cx="72" cy="95" r="3"/><circle cx="95" cy="115" r="3"/><circle cx="112" cy="85" r="3"/>
  </g>
  <text x="90" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="9">before: EVM 34%</text>
  <line x1="160" y1="90" x2="185" y2="90" stroke="currentColor"/><polygon points="185,86 195,90 185,94" fill="currentColor"/>
  <circle cx="255" cy="90" r="52" fill="none" stroke="var(--fg-muted)"/>
  <g fill="var(--accent)">
    <circle cx="292" cy="53" r="3"/><circle cx="291" cy="55" r="3"/>
    <circle cx="218" cy="53" r="3"/><circle cx="220" cy="55" r="3"/>
    <circle cx="218" cy="127" r="3"/><circle cx="220" cy="125" r="3"/>
    <circle cx="292" cy="127" r="3"/><circle cx="290" cy="125" r="3"/>
    <circle cx="255" cy="38" r="3"/><circle cx="255" cy="142" r="3"/>
    <circle cx="203" cy="90" r="3"/><circle cx="307" cy="90" r="3"/>
  </g>
  <text x="255" y="164" text-anchor="middle" fill="var(--accent)" font-size="9">after: EVM 8% — “fixed”</text>
  <line x1="60" y1="200" x2="640" y2="200" stroke="var(--fg-muted)"/>
  <rect x="440" y="120" width="60" height="60" fill="var(--fg-muted)"/>
  <text x="470" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="9">baseline yield</text>
  <rect x="560" y="179" width="60" height="1" fill="currentColor"/>
  <text x="590" y="196" text-anchor="middle" fill="currentColor" font-size="9">after: CRC = 0</text>
  <text x="530" y="110" text-anchor="middle" fill="currentColor" font-size="10">CRC-valid frames</text>
</svg>
<figcaption>The same change, two metrics: EVM collapsed 34% → 8% and declared victory; the CRC-valid frame count went to zero. Tidiness is not identity.</figcaption>
</figure>

## The carrier-SNR trap

EVM is not the only metric with a blind spot.
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) — the
investigation the
[Ten Megasamples postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-05-ten-megasamples/' | relative_url }})
narrates in full — turned on a pair of captures of the same site: one at
2.5 MS/s that locked cleanly (demod SNR ≈19.7 dB, EVM 7.4%), one at 10 MS/s
that never locked (≈9.5 dB, EVM 22.5%). The natural first move was a wideband
FFT of both files to compare signal quality — and the FFT said the 10 MS/s
capture had the *higher* carrier SNR. Cleaner file, by that measurement.
Worse decode, by 10 dB.

Both measurements were correct. A wideband FFT bin integrates the carrier's
*power* against the noise floor around it; it is nearly blind to energy that
stays close to the carrier. Oscillator phase noise — in that case, reciprocal
mixing at the Airspy's native 10 MS/s clock — smears energy from the carrier
onto its own modulation sidebands. Carrier-clean but modulation-degraded is
that mechanism's exact signature, and only an *in-channel, demod-side*
measurement sees it, because only the demodulator asks the question at the
bandwidth where the damage lives. The proof that the deficit was baked into
the samples (an independent 4:1 decimation replayed through the proven
2.5 MS/s path reproduced the same ≈9.5 dB) is Part 12's subject; the metric
lesson stands alone: **a measurement made in the wrong bandwidth answers a
different question, confidently.**

## The numbers you can trust

The series' working hierarchy, from verdict to scenery:

| Rank | Metric | Role | Caveat |
|---|---|---|---|
| 1 | CRC-valid frames / opportunities | **the verdict** | needs an honest denominator |
| 2 | Payload bit-error rate vs known truth | verdict, synthetic only | only where truth is known (tests) |
| 3 | In-channel demod SNR | advisory — places you on Part 1's cliff | not a verdict; measure post-channel-filter |
| 4 | EVM | advisory — useful on *unprocessed* signals | never after a blind adaptive stage |
| 5 | Wideband carrier SNR, dBFS | scenery — capture hygiene | blind to phase noise, says nothing about decode |

Two design notes on rank 1. First, the *opportunity* count has to be honest:
counting CRC passes per *detected* burst lets a change that suppresses burst
detection look like a yield improvement. GopherTrunk's harnesses count
`ok` and `fail` from the same detector so the denominator can't be gamed by
the thing under test. Second, CRC yield inherits the CRC's own false-positive
floor — a 16-bit check passes garbage at ~1/65536 — which is negligible at
healthy yields but worth remembering when a sweep reports single-digit counts:
Part 8 and the DMO colour-recovery story both lean on *dominance* over that
chance floor, not mere non-zero counts.

## Instrumenting for yield

The principle only bites if the code makes yield cheap to read. Three places
it shows up, all of which later parts use as scoreboards:

```go
// internal/radio/tetra/tch.go (shape) — the per-burst verdict
func DecodeTCHSSoft(type5LLR []float32) (frameA, frameB []byte, crcOK bool, errs float32, ok bool)
```

Every TCH/S decode returns `crcOK` alongside the speech frames — the composer
and the replay harnesses tally it per opportunity, which is how "soft-decision
410 → 778 bursts" was ever a statement anyone could make. The CC-side
equivalent is `decodeCCBSCHYield` from Part 1, which returns `(ok, fail)`
with exactly one boolean of configuration difference between the two arms.
And the periodic `tetra: decode status` line carries the same counters into
production logs (`bsch_fail`, `sb_bursts`, and friends), so an operator's
debug log *is* a yield instrument — the ~210-transitions-per-hour session was
diagnosed from those lines alone, before any capture was replayed.

### How that principle shaped the Go code

- **Verdict functions return their own denominator.** Decoders return
  `crcOK`/`ok` per call rather than logging aggregate percentages, so any
  caller — test, composer, status line — can build an honest ratio over
  exactly the population it cares about.
- **A/B harnesses isolate one boolean.** `decodeCCBSCHYield` mirrors the
  production pipeline except for the single parameter under test, so a yield
  delta is attributable to one change, not to drifted wiring.
- **The trap is documented where you'd re-fall into it.** The EVM incident
  lives in the `SnapshotLMS` doc comment — the exact place someone evaluating
  a new equalizer will be reading when they're tempted to grade it by EVM.

## Where this goes next

With the measurement rules fixed, we can finally look at the enemy itself.
[Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})
is about inter-symbol interference and the linear channel model `y = h∗x + n`:
what multipath and band-edge group delay actually do to a constellation, which
impairments are invertible and which are not — and the one domain fact
(a linear channel is only a convolution over *raw* symbols, never over
differentials) that quietly dictates the architecture of every equalizer in
the rest of the series.

## FAQ

**Is EVM ever the right metric?**
Yes — on signals that haven't passed through an adaptive stage that could game
it. For grading a capture's raw modulation quality, comparing antennas, or
watching a transmitter drift, [Signal Lab's VSA]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }})
EVM is exactly the tool. The rule is scoped: EVM must not be the *verdict on a
decode-chain change*, because decode-chain changes can optimize it directly.

**Why does CRC yield beat bit-error rate as the field verdict?**
Because on-air you don't know the transmitted bits, so you can't count bit
errors — the CRC is the only ground truth the air gives you. In synthetic
tests, where truth is known, GopherTrunk does use payload bit-error
(Part 7's 13% → 0% result is exactly that); the two verdicts are used where
each is available.

**Couldn't a change also game the CRC metric?**
Only by actually decoding correctly — which is the point of using it. The two
gameable edges are the denominator (fixed by counting opportunities from the
same detector) and the CRC's chance floor (~1/65536 for a 16-bit check), which
matters only for near-zero counts and is why marginal sweeps demand dominance,
not presence.

**What's "demod SNR" versus the FFT SNR my waterfall shows?**
The waterfall number is carrier power over the noise floor in FFT-bin
bandwidth — a wideband, pre-demod view. Demod SNR is estimated inside the
decode chain, in the channel bandwidth, after filtering — it sees everything
the decoder sees, including phase noise folded onto the modulation. #764 is
the canonical case where they disagree by design.

**Do I need special builds to read yield on my own system?**
No. The TETRA decode-status debug line carries the burst and failure counters,
and every capture-replay harness prints its `ok/fail` tallies. If you can
reproduce a symptom in a capture, you already have a yield instrument —
which is why the auto-recorders exist.

## Series navigation

**Part 2 of 14** · ←
[Part 1: The Marginal Regime]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }})
· Next →
[Part 3: ISI & the Linear Channel — What an Equalizer Can & Can't Fix]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})
