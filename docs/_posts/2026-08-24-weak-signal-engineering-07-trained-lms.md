---
title: "Weak-Signal Engineering, Part 7: Trained Equalization — LMS on the Midamble"
description: When the burst carries known symbols, stop guessing — how GopherTrunk trains a SnapshotLMS on each TETRA burst's midamble in the raw-symbol domain, freezes the taps, equalizes the payload with a FIR warm-up, and re-derives the soft LLRs, taking a synthetic multipath burst from 13% payload bit-error to zero while staying byte-identical when switched off.
category: deep-dives
keywords: lms equalizer, training sequence equalization, tetra midamble, snapshot lms, per-burst equalizer, raw symbol domain, wirtinger gradient conjugate, decision directed lms, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, lms, equalizer, tetra, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 7
---

*Part 7 of **Weak-Signal Engineering**, a 14-part deep dive into decoding the
marginal regime, where a receiver locks but under-decodes.
[Part 6]({{ '/blog/deep-dives/weak-signal-engineering-06-normalisation-guards/' | relative_url }})
closed out the blind equalizer — cost, snapshot, references, guards — a lever
that asks nothing of the signal but its envelope, and pays for that humility
with spurious minima it cannot detect from inside. But most burst-mode radio
protocols are not actually that coy: every TETRA normal burst carries a
training sequence — a midamble of known symbols at a known position —
broadcast in the clear, burst after burst. This part is about accepting the
gift: train an equalizer on symbols you *know*, freeze it, and equalize the
payload around them. It is also where Part 3's domain fact — the channel is
only a convolution over raw symbols — stops being theory and starts dictating
plumbing.*

> **TL;DR:** **`SnapshotLMS`** (`internal/dsp/equalizer/snapshot_lms.go`) is
> the trained counterpart to `SnapshotCMA`: adapt an
> [LMS]({{ '/reference/lms-algorithm/' | relative_url }}) filter
> (`w += μ·e·conj(x)`, `e = d − y`) on a burst's known **midamble**, freeze the
> taps, apply them to the whole burst. Training kills blind CMA's ambiguity —
> `e = d − y` is only zero at the *right* inverse, phase included. Because the
> linear channel exists only in the **raw-symbol domain** (not in the
> differential products), the TETRA `TrafficExtractor` carries raw symbols
> down parallel to its dibits (`SymbolSink → StashSymbols`), trains on the
> 11-dibit midamble against a reference built by differentially encoding the
> ideal sequence **from a unit anchor**, equalizes BKN1..BKN2 with a taps-long
> FIR **warm-up**, and **re-derives the soft LLRs** from equalized symbols —
> hard frame untouched, byte-identical when off. Synthetic multipath:
> payload bit-error **13% → 0%**, no harm on clean. Status honesty: a staged
> lever — the capture A/B (`GT_TETRA_LMS=1`) is still pending, so production
> composers run CMA only.

**Key takeaways**

- **Known symbols collapse the blind ambiguity.** CMA accepts any
  constant-modulus output; LMS training accepts only the output that maps the
  reference to itself. The 34%→8%-EVM-with-CRC-0 failure class is structurally
  impossible against a good training sequence.
- **The extractor trains, not the receiver — because only the extractor knows
  where the midamble is.** Blind equalization lives in the framing-free
  receiver stream; trained equalization needs burst geometry, so it lives
  where bursts are found. Placement follows knowledge.
- **Eleven dibits is a short teacher, so the lesson is repeated.** Five taps,
  eighty passes over the midamble: sweeping a short reference many times is
  how an LMS converges on so few known symbols.
- **The result is real and the claim is scoped.** 13% → 0% is a synthetic,
  failing-first regression through the real extractor; the on-air win is
  unproven until an operator capture A/B says so. The lever is wired, opt-in,
  and waiting — that's a status, not a victory lap.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The LMS core | `e = d − y`; `w += μ·e·conj(x)` per sample | `internal/dsp/equalizer/lms.go` (`LMS.Process`) |
| Train-then-freeze wrapper | `Train(rx, ref, passes)` → frozen `Equalize` | `internal/dsp/equalizer/snapshot_lms.go` (`SnapshotLMS`) |
| Raw-symbol plumbing | symbols parallel to dibits, same `baseIdx` | `internal/radio/tetra/receiver/receiver.go` (`SymbolSink`) → `traffic.go` (`StashSymbols`) |
| Per-burst orchestration | train on midamble, equalize span, re-derive LLRs | `internal/radio/tetra/traffic.go` (`equalizedBurstDiffs`, `softFrame`) |
| The ideal reference | differentially encode NTS1/NTS2 from a unit anchor | `traffic.go` (`midambleRef`, `idealDiff`) |
| Opt-in switch + defaults | 5 taps, 80 passes, μ = 0.02 | `traffic.go` (`EnableLMSEqualizer`, `defaultLMS*`) |
| Failing-first regressions | 13%→0% payload bit-error; byte-identical off | `traffic_lms_test.go`, `snapshot_lms_test.go` |
| DMO variant | rotation-0 DNBs only, different burst geometry | `internal/radio/tetra/dmo_equalizer.go` (`dmEqualizeDNBSoft`) |

## In this post

- **From blind to trained** — what a known reference buys.
- **LMS, and the conjugate that almost got away** — the update rule's fine print.
- **Where the trained equalizer must live** — receiver vs. extractor.
- **One burst, start to finish** — reference, training, warm-up, re-derived LLRs.
- **Results and honest status** — synthetic wins, pending air.

## From blind to trained

Part 4 ended with a sober admission: CMA's own diagnostics cannot distinguish
the right channel inverse from a spurious constant-modulus impostor. The
`SnapshotLMS` doc comment states the escape in one clause — training to a
known reference "pins the true channel inverse (phase and all), not merely
its modulus." The error `e = d − y` is zero *only* when the composite
channel-plus-equalizer maps the known transmitted symbols back to themselves.
No rotation ambiguity, no spurious minima worth the name, and convergence you
can verify against ground truth per burst.

The price is the information itself: you need known symbols at known
positions, which means you need framing. TETRA obliges generously — the
[training sequences]({{ '/reference/tetra-training-sequences/' | relative_url }})
NTS1 and NTS2 sit mid-burst (a midamble, 11 dibits) in every normal downlink
burst, and their whole reason for existing in the standard is exactly this:
they are the part of the burst the receiver is *supposed* to already know.
GopherTrunk's burst extractor was already correlating against them to find
bursts at all; the trained equalizer just stops throwing away what the
correlation proves it knows.

## LMS, and the conjugate that almost got away

The Least-Mean-Squares update is the oldest tool in the adaptive-filter
drawer: FIR output `y = Σ w_k·x[n−k]`, error against the reference
`e = d − y`, taps nudged along the gradient. `lms.go` carries the rule — and,
in its comments, a war story that belongs in this series' trap collection:

```go
// internal/dsp/equalizer/lms.go (shape) — Process
// Weight update: w_k += μ · e · conj(x[n-k]).
//
// For the non-Hermitian filter y = Σ_k w_k·x_k, Wirtinger calculus
// gives ∂J/∂w_k* = −e·conj(x_k) with J = |d−y|², so steepest descent
// is w_k += μ·e·conj(x_k). The conjugate is on x, NOT on e: the two
// differ only in the sign of the imaginary cross-term, so a real-only
// channel can't tell them apart — but on a COMPLEX channel the wrong
// sign turns the update into ascent on the imaginary axis and the taps
// diverge. (The earlier code computed x·conj(e), which is that wrong
// sign; it survived only because the package tests used a real-valued
// channel coefficient. See TestLMSConvergesOnComplexChannel.)
```

Savor that failure mode, because it is the self-consistent-synthetic trap in
miniature: `x·conj(e)` and `e·conj(x)` agree exactly on any real-valued
channel, so a test suite whose synthetic channels were all real passed
forever while the update was, on any *complex* channel — i.e., any real one —
gradient *ascent* in the imaginary direction. The fix came with a
failing-first test whose channel coefficient is complex, and the lesson
generalises: test inputs must span the dimensions your math can be wrong in.
`SnapshotLMS` wraps this corrected core in the Part 5 pattern — `Train` runs
`adapt.Process(rx[i], ref[i])` over the aligned pairs (repeating `passes`
times), then `copy(e.apply, e.adapt.Taps())` freezes the result; `Equalize`
is a plain frozen FIR. A held-constant filter imposes only a constant
phase/scale, which the downstream differential cancels — the same safety
argument as `SnapshotCMA`, with the snapshot window now exactly one burst.

## Where the trained equalizer must live

Here the two Part 3 threads braid together. The **blind** equalizer sits in
the *receiver* — a continuous, framing-free stream is precisely where a
reference-free algorithm belongs, and the receiver has no idea where bursts
are. The **trained** equalizer inverts that logic: training requires knowing
where the midamble sits, and the only component that knows is the
`TrafficExtractor`, downstream of the dibit stream, *after* the nonlinear
differential decode. But Part 3 proved the channel is only a convolution over
**raw symbols** — equalizing differentials is algebra that doesn't exist.

So the raw symbols have to travel. The receiver grew a `SymbolSink` — the
symbol analog of the soft path's `SoftSink` — emitting the post-timing/AFC
complex symbols aligned 1:1 with the dibits, same `baseIdx`; the extractor
grew `StashSymbols` and a `symBuf` carried parallel to its dibit and soft
buffers. (That three-buffer architecture — dibits, differentials, raw
symbols, all index-aligned — is Part 9's subject in its own right.) The
placement rule deserves its one-line form: **blind in the receiver, trained
in the extractor — each equalizer lives where its information lives.**

## One burst, start to finish

`softFrame` — the function that builds each burst's 432-LLR soft frame —
now tries the trained path first and falls back cleanly:

```go
// internal/radio/tetra/traffic.go (shape) — softFrame
diffs := te.equalizedBurstDiffs(L)   // nil unless LMS enabled + symbols buffered
if diffs == nil {
    diffs = te.rawBurstDiffs(L)      // the pre-#1001 soft path, untouched
}
llr := softType5FromDiffs(diffs, 0)
```

Inside `equalizedBurstDiffs`, one burst's worth of work, in order. **Build
the reference**: `midambleRef` picks NTS1 or NTS2 (whichever the received
dibits match more closely), then differentially encodes the ideal dibits from
a **unit anchor** — `out[0] = 1`, each next symbol the previous times the
ideal differential. The anchor phase is arbitrary, and that's the point: a
constant rotation cancels in the differential decode — the same invariance
that makes frozen snapshots safe now makes the reference constructible
without knowing the burst's absolute phase. **Train**: five taps swept eighty
passes over the 11-symbol midamble (`defaultLMSTaps`, `defaultLMSPasses`) —
a short teacher, repeated until the estimate converges — then freeze.
**Equalize with a warm-up**: the span starts `taps` symbols *before* BKN1's
differential anchor, so the FIR's delay line is full — past its transient —
before the first payload symbol anyone will use:

```go
// internal/radio/tetra/traffic.go (shape) — equalizedBurstDiffs
warm := te.lmsTaps
spanStart := L + ndbBKN1Start - 1 - warm // one before BKN1 start, minus FIR warm-up
te.lms.Reset()
te.lms.Train(rx, ref, te.lmsPasses)
span := te.lms.Equalize(te.lmsSpan[:0], te.symBuf[s0:s1])
// diff for burst dibit m: span[i]·conj(span[i-1]), i = m - spanStart
```

**Re-derive, don't patch**: the 216 differentials for BKN1 and BKN2 are
recomputed from *equalized* symbols and flow into the same
`softType5FromDiffs` → descramble → soft depuncture → Viterbi chain as
before. The hard frame never changes. If any precondition fails — equalizer
off, no symbols stashed, span not fully buffered — the function returns nil
and the burst decodes exactly as it would have in the previous release,
pinned by `TestTrafficExtractorSoftUnchangedWithoutEqualizer`.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A TETRA normal burst laid out as blocks: payload block BKN1, the known eleven-dibit midamble in the centre, payload block BKN2. The trained equalizer flow is drawn around it: an arrow from the midamble down to a train box, LMS five taps eighty passes against the ideal reference built from a unit anchor; then freeze; then a frozen-taps arrow applying across a span that begins a warm-up region before BKN1 and runs through the end of BKN2; the output is re-derived differentials feeding the soft decoder.">
  <rect x="30" y="40" width="90" height="34" rx="4" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="75" y="61" text-anchor="middle" fill="var(--fg-muted)" font-size="9">warm-up (taps)</text>
  <rect x="120" y="40" width="180" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="210" y="61" text-anchor="middle" fill="currentColor" font-size="10">BKN1 — 108 dibits</text>
  <rect x="300" y="40" width="80" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="340" y="55" text-anchor="middle" fill="var(--accent)" font-size="9">midamble</text>
  <text x="340" y="68" text-anchor="middle" fill="var(--accent)" font-size="9">NTS1/2 · known</text>
  <rect x="380" y="40" width="180" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="470" y="61" text-anchor="middle" fill="currentColor" font-size="10">BKN2 — 108 dibits</text>
  <line x1="340" y1="74" x2="340" y2="108" stroke="var(--accent)"/><polygon points="336,108 340,118 344,108" fill="var(--accent)"/>
  <rect x="240" y="118" width="200" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="133" text-anchor="middle" fill="var(--accent)" font-size="10">Train: LMS, 5 taps × 80 passes</text>
  <text x="340" y="147" text-anchor="middle" fill="var(--fg-muted)" font-size="9">vs ideal ref from unit anchor</text>
  <line x1="240" y1="136" x2="150" y2="136" stroke="currentColor"/><polygon points="150,132 140,136 150,140" fill="currentColor"/>
  <rect x="40" y="118" width="100" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="90" y="133" text-anchor="middle" fill="currentColor" font-size="10">freeze taps</text>
  <text x="90" y="147" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one burst, one filter</text>
  <path d="M90,118 C90,96 110,88 130,88 L555,88" fill="none" stroke="currentColor" stroke-dasharray="5 3"/>
  <polygon points="555,84 565,88 555,92" fill="currentColor"/>
  <text x="330" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">apply frozen FIR across warm-up + BKN1 + midamble + BKN2</text>
  <line x1="565" y1="88" x2="600" y2="88" stroke="currentColor"/>
  <text x="620" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">re-derived</text>
  <text x="620" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">216 diffs</text>
  <text x="620" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ soft decode</text>
  <text x="340" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="10">train on what you know, freeze, equalize what you don't — the hard frame is never touched</text>
</svg>
<figcaption>Per burst: train a short LMS on the known midamble against a unit-anchored ideal reference, freeze the taps, equalize the whole burst with a taps-long warm-up, and re-derive the payload's soft differentials from equalized symbols.</figcaption>
</figure>

## Results — and honest status

The synthetic verdict is clean and failing-first.
`TestTrafficExtractorLMSRecoversMultipathBurst` pushes modulated bursts
through a multipath channel and the *real* extractor: raw soft path **13%
payload bit-error**, LMS path **0%** — and the no-harm and byte-identical-off
companions hold (`TestTrafficExtractorLMSNoHarmOnCleanChannel`,
`TestTrafficExtractorSoftUnchangedWithoutEqualizer`). At the package level,
`TestSnapshotLMSBeatsBlindCMAOnBurst` pins the comparative claim: on a single
burst-length reference, training beats blind — CMA simply cannot pin a
channel from 11 symbols of nothing-but-envelope.

The same lever is wired into TETRA's direct mode with two DMO-specific twists
(`dmo_equalizer.go`): the DNB burst geometry differs, and it equalizes
**rotation-0 bursts only** — a frozen constant-tap filter cannot invert the
per-symbol phase *ramp* of a residual rotation, and at rotation 0 the soft
decoder's de-rotation is a no-op, so the rotation-0-trained differentials
slot straight in.

And now the scoping, stated the way this project's
[issue-closing discipline](https://github.com/MattCheramie/GopherTrunk/issues/764)
demands. This is a **staged lever, not a verified win**: the production voice
composer still runs blind CMA only, and `SnapshotLMS` in the traffic path is
opt-in (`EnableLMSEqualizer` + `StashSymbols`), flipped on in the replay
harness by `GT_TETRA_LMS=1` for capture A/Bs that compare soft CRC yield. On
the first DMO capture it was tried against, it did not move the number (35 →
32 CRC-valid — that capture's ceiling was elsewhere). A green synthetic
through the real extractor is necessary evidence, not sufficient; the lever
graduates when an operator capture shows the yield delta, and not before.

## Where this goes next

Twice now, the equalizers have ended their work by "re-deriving the soft
LLRs" — and the phrase has gone unexamined. It's time. [Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})
descends into soft decisions themselves: what a log-likelihood ratio carries
that a hard bit throws away, how LLRs survive descrambling, deinterleaving,
and depuncturing to reach a soft Viterbi (`DecodeRCPCTetraMotherSoft`), the
classic ~2 dB framing and the measured TETRA outcome — the ~70% of a marginal
call's bursts that hard decisions failed and soft ones recovered.

## FAQ

**Why five taps for the trained equalizer when the blind one uses eleven?**
The teacher is short. Eleven midamble symbols have to pin every tap; more
taps on so few reference symbols under-determine the estimate and overfit
noise, and the eighty passes already work the reference hard. The blind CMA,
adapting over an unbounded stream, can afford the longer filter. Channel span
you can model is bounded by reference you can train on.

**Doesn't re-running Train per burst throw away what the last burst learned?**
`te.lms.Reset()` per burst is deliberate: consecutive bursts on a moving
channel (or from different transmitters — this is trunked radio) need not
share an inverse, and a stale estimate is a confident wrong one, the same
argument as Part 5's `Reset`-on-resync. The cost is re-convergence from
center-spike each burst — which the 80 passes over the midamble pay for.

**Why does the reference need an anchor symbol at all?**
The midamble is *defined* as dibits — phase differences. To train in the
raw-symbol domain you must integrate the differences into symbols, and
integration needs a starting value. Any unit-modulus start works because the
resulting constant rotation of the whole reference cancels in the downstream
differential — the same invariance Part 5 leaned on, now used constructively.

**Could the trained equalizer replace the blind one?**
No — they cover different ground. `SnapshotCMA` improves the *receiver's*
stream before any framing exists (it's part of why bursts are found at all on
marginal captures); `SnapshotLMS` refines *found* bursts. The staged design
runs both: blind in the receiver, trained per burst on top, each measured
separately — which is exactly how the DMO A/B could report "CMA lifted DSB
6→64; LMS didn't move TCH" on the same capture.

**What would make the LMS lever graduate to on-by-default?**
The same thing that graduated the CMA: an operator capture A/B where the only
variable is the lever and the CRC yield moves. The harness exists
(`GT_TETRA_LMS=1` in the multislot replay test, comparing
`traffic_marked_crc_soft`), the fallback is byte-identical, and the synthetic
regression guards the mechanism. What's missing is the capture where linear
per-burst ISI is the binding constraint — Part 12 is, in part, about how to
recognise one.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Normalisation & Divergence Guards]({{ '/blog/deep-dives/weak-signal-engineering-06-normalisation-guards/' | relative_url }})
· Next →
[Part 8: Soft Decisions — LLRs Through Depuncture & Viterbi]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})
