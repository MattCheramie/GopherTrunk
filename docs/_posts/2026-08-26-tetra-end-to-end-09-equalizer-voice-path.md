---
title: "TETRA End to End, Part 9: The Equalizer on the Voice Path"
description: How a blind CMA equalizer with frozen-snapshot taps, inserted between symbol timing and the differential decoder, roughly doubled CRC-valid TCH/S yield on real concurrent-load TETRA captures — and why the naive way of wiring it drives the decode to exactly zero.
category: deep-dives
keywords: cma equalizer, blind equalization tetra, snapshot cma frozen taps, differential decoder equalizer, isi multipath tetra, crc yield vs evm, lms training sequence equalizer, pi/4-dqpsk equalization, gophertrunk tetra
tags: [tetra-end-to-end, tetra, equalizer, cma, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 9
---

*Part 9 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 8]({{ '/blog/deep-dives/tetra-end-to-end-08-soft-decision-tchs/' | relative_url }})
stopped throwing away the demod's confidence, and the marginal same-carrier call
got dramatically better — but not clean. What remained was not noise: the
π/4-DQPSK constellation on the reporter's concurrent-load captures was *smeared*,
each symbol dragged by its neighbours. That is a linear channel — multipath,
band-edge group delay, ISI — and no amount of per-bit confidence fixes it,
because the symbols themselves are in the wrong places. This part inverts the
channel: a blind equalizer on the voice path, and the one design constraint that
makes it survivable in front of a differential decoder.*

> **TL;DR:** The residual garble after soft decision was **linear channel / ISI**
> smearing the constellation — not front-end degradation. A blind CMA equalizer
> (**`equalizer.SnapshotCMA`**, wired via the receiver's `EnableEqualizer` in the
> voice composer) between symbol-timing recovery and the differential decoder
> roughly **doubled** CRC-valid TCH/S yield across six captures — soft-decision
> **410→778** (~1.9×), one call **4→207**, another **42→134** — with no loss on
> already-clean captures. The load-bearing design: **adapt a tracking filter
> continuously, apply a FROZEN snapshot** refreshed every 200 symbols, because
> CMA's phase wanders as it adapts and a time-varying phase does not cancel in
> `s·conj(prev)` — streaming-adaptive CMA drives CRC yield to exactly **zero**.
> The trained sibling **`SnapshotLMS`** (per-burst midamble training in the
> `TrafficExtractor`) is staged behind `GT_TETRA_LMS` for capture A/B.

**Key takeaways**

- **CRC yield is the only trustworthy metric; EVM is a trap.** A
  numerically-unstable CMA variant once showed differential EVM collapsing
  34%→8% while CRC-valid bursts stayed **0**. Blind CMA minimises *modulus*,
  not correctness — never conclude an equalizer helps from EVM.
- **Never feed a continuously-adapting equalizer to a differential decoder.**
  CMA's cost is rotation-invariant, so its output phase wanders with the taps;
  the wandering phase corrupts every `s·conj(prev)`. Frozen snapshots impose
  only a constant phase, which the differential cancels.
- **Normalise by a constant, not a local EMA.** The CMA update scales with
  |x|³; an EMA tracking the TDMA downlink's slot-to-slot power swings gives a
  moving modulus target and CMA converges to garbage. Cumulative-mean
  normalisation plus a divergence guard keeps it sane.
- **The equalizer can only help.** Center-spike initialisation means the output
  is the unmodified input until CMA converges — clean captures measure no
  regression.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Blind equalizer | CMA(2,2) tracking + frozen applied snapshot | `internal/dsp/equalizer/snapshot_cma.go` (`SnapshotCMA`) |
| Receiver wiring | insert between timing recovery and diff decode | `internal/radio/tetra/receiver/receiver.go` (`Options.EnableEqualizer`) |
| Defaults | 11 taps, µ=6e-3, snapshot every 200 symbols | `receiver.go` (`DefaultEqualizerTaps` / `Mu` / `Snapshot`) |
| Voice-chain enable | composer turns it on for every TETRA follow | `internal/voice/composer/tetra_voice.go` (`EnableEqualizer: true`) |
| Divergence guard | re-seed to pass-through if taps blow up | `snapshot_cma.go` (`snapshotDivergeGuard`) |
| Trained sibling | per-burst midamble-trained LMS, frozen taps | `internal/dsp/equalizer/snapshot_lms.go` (`SnapshotLMS`) |
| LMS wiring | train on NTS midamble, re-derive soft LLRs | `internal/radio/tetra/traffic.go` (`EnableLMSEqualizer`, `equalizedBurstDiffs`) |
| Proof | ISI recovery + clean-channel no-harm + capture sweep | `snapshot_cma_test.go`, `receiver_equalizer_test.go`, `traffic_lms_test.go` |

## In this post

- **Diagnosing ISI, not noise** — why the residual garble had structure.
- **Why plain CMA kills a differential decoder** — the phase-wander failure.
- **The snapshot trick** — adapt continuously, apply frozen.
- **Normalisation and the divergence guard** — the two quiet failure modes.
- **The trained sibling: LMS on the midamble** — staged, not yet default.

## Diagnosing ISI, not noise

After Part 8, the reporter's concurrent-load captures still garbled. The
tempting diagnosis was the one
[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) taught us —
front-end degradation baked into the captured samples. But the signature was
different. A signal-limited capture is *noisy*: the constellation is a fuzzy
cloud centered on the right points. These captures showed points dragged into
crescents and smears — each received symbol a weighted sum of its neighbours.
That is a **linear channel**: multipath reflections, band-edge group delay from
the channel filter, ISI. The distinction matters because the remedies are
disjoint — you cannot equalize thermal noise, and you cannot out-gain ISI. The
general taxonomy is
[Weak-Signal Engineering Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }});
here it meant one thing: the fix was a filter, and it had to be *blind*,
because a voice follow has no training preamble before the speech starts.

## Why plain CMA kills a differential decoder

The classic blind equalizer is CMA — the constant modulus algorithm.
π/4-DQPSK symbols all sit on the unit circle, so drive an adaptive FIR to make
its output's modulus constant and it must be inverting whatever smeared the
ring: `J = E[(|y|² − R²)²]`. The package's plain `CMA` does exactly this and
works fine ahead of a *coherent* slicer.

Ahead of a differential decoder it is fatal, and the reason is worth engraving.
CMA's cost function only sees modulus — it is **rotation-invariant**. Nothing
constrains the equalizer's output phase, so as the taps adapt, that phase
wanders freely. A coherent receiver's carrier loop tracks the wander out. But
the differential decode forms `s[n]·conj(s[n−1])`, and a phase that *changes
between consecutive symbols* does not cancel there — it adds directly to the
decision angle of every dibit. Measured, not argued: wiring the
continuously-adapting CMA into the TETRA voice path produced **CRC yield 0**.
Frozen taps produced baseline. The failure is structural, not a tuning issue —
[Weak-Signal Engineering Part 4]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})
derives it from the cost function.

## The snapshot trick

The resolution is to separate adapting from applying:

```go
// internal/dsp/equalizer/snapshot_cma.go (shape) — Process
// Output uses the frozen filter (phase-coherent between snapshots).
var y complex64
for k := 0; k < n; k++ {
    y += e.wApply[k] * e.buf[n-1-k]
}
// Adapt the tracking filter by CMA(2,2): e = ya·(|ya|² − 1).
var ya complex64
for k := 0; k < n; k++ {
    ya += e.wAdapt[k] * e.buf[n-1-k]
}
/* … CMA tap update on wAdapt, divergence guard … */
if e.since++; e.since >= e.snapEvery {
    copy(e.wApply, e.wAdapt) // refresh the frozen snapshot
    e.since = 0
}
```

`wAdapt` updates every sample and its phase wanders — it is never applied.
`wApply` is a snapshot of it refreshed every `snapEvery` samples (200 by
default, ≫ a 255-symbol burst's data span) and is the filter the output
actually sees. Held fixed, `wApply` imposes only a *constant* phase and scale —
exactly what the differential decode cancels by construction — while still
inverting the channel. The one symbol per snapshot that straddles the refresh
sees a phase step; the FEC absorbs a dibit or two per 200. And because
`wApply` starts as a center spike, the pre-convergence output is the unmodified
input: on a clean capture the equalizer is a near-noop, which is why the sweep
measured **no loss** where there was nothing to fix. The pattern is general
enough that it earned its own theory post —
[Weak-Signal Engineering Part 5]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Block diagram of the snapshot CMA equalizer. The normalised input feeds two FIR filters in parallel: a tracking filter whose taps adapt on every sample by the CMA rule and whose output phase wanders, and an applied filter whose taps are a frozen copy refreshed every two hundred symbols. Only the frozen filter's output goes to the differential decoder, so between snapshots the decoder sees a constant phase that cancels in s times conjugate of previous s.">
  <rect x="8" y="80" width="120" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="100" text-anchor="middle" fill="currentColor" font-size="10">input symbols</text>
  <text x="68" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cumulative-mean norm</text>
  <line x1="128" y1="92" x2="168" y2="52" stroke="var(--fg-muted)"/><polygon points="164,50 176,48 168,60" fill="var(--fg-muted)"/>
  <line x1="128" y1="116" x2="168" y2="156" stroke="var(--accent)"/><polygon points="170,150 176,162 162,158" fill="var(--accent)"/>
  <rect x="178" y="20" width="180" height="52" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="268" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="10">wAdapt — tracks every sample</text>
  <text x="268" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">phase wanders: NEVER applied</text>
  <rect x="178" y="140" width="180" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="268" y="160" text-anchor="middle" fill="var(--accent)" font-size="10">wApply — frozen snapshot</text>
  <text x="268" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">constant phase between copies</text>
  <line x1="268" y1="72" x2="268" y2="132" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="264,132 268,142 272,132" fill="var(--fg-muted)"/>
  <text x="282" y="106" fill="var(--fg-muted)" font-size="9">copy every 200 symbols</text>
  <line x1="358" y1="166" x2="404" y2="166" stroke="var(--accent)"/><polygon points="404,162 414,166 404,170" fill="var(--accent)"/>
  <rect x="414" y="140" width="150" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="489" y="160" text-anchor="middle" fill="currentColor" font-size="10">differential decode</text>
  <text x="489" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">s·conj(prev) — constant phase cancels</text>
  <line x1="564" y1="166" x2="606" y2="166" stroke="currentColor"/><polygon points="606,162 616,166 606,170" fill="currentColor"/>
  <text x="644" y="162" text-anchor="middle" fill="currentColor" font-size="10">dibits +</text>
  <text x="644" y="176" text-anchor="middle" fill="currentColor" font-size="10">LLRs</text>
  <text x="330" y="205" text-anchor="middle" fill="var(--fg-muted)" font-size="10">adapt continuously, apply frozen — the only wiring of blind CMA a differential decoder survives</text>
</svg>
<figcaption>SnapshotCMA keeps two tap sets: the tracking filter adapts every sample, the applied filter is a frozen copy — so the differential decoder only ever sees a constant phase.</figcaption>
</figure>

## Normalisation and the divergence guard

Two quieter failure modes cost real time before the yield numbers appeared,
and both are pinned by `snapshot_cma_test.go`.

**Normalisation.** The CMA tap update scales with |x|³, so the input must be
normalised toward unit power for the `R² = 1` target to mean anything. The
obvious choice — a local EMA power estimate — is wrong for TETRA specifically:
the TDMA downlink's power swings slot to slot, an EMA tracks the swings, and a
*moving* modulus target makes CMA converge to garbage (CRC 0, again, even
though a global-RMS normalise gave the full win). `SnapshotCMA` instead divides
by a **cumulative mean** — `cumSum/count`, converging to the whole-session RMS
and then staying put. A constant scale is a constant target.

**Divergence.** A normalisation transient or a deep fade can blow the tracking
taps up; without a guard, one bad patch poisons every later snapshot. The guard
is blunt and effective — if any tap's squared magnitude exceeds
`snapshotDivergeGuard` (|tap| > 3), the tracking filter re-seeds to a center
spike and starts over. The applied filter keeps its last sane snapshot in the
meantime. The general pattern is
[Weak-Signal Engineering Part 6]({{ '/blog/deep-dives/weak-signal-engineering-06-normalisation-guards/' | relative_url }}).

The verdict metric for all of it: **CRC-valid TCH/S bursts**, never EVM
([Weak-Signal Engineering Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})
is the full indictment). Across the six reporter captures: soft-decision yield
**410→778** (~1.9×), one call **4→207**, another **42→134**, clean captures
unchanged. The composer enables it for every TETRA voice follow
(`EnableEqualizer: true` in `runTETRAVoiceChain`), alongside the voice-only
`EnableDCBlock` — a distinction Part 10 returns to on the control-channel side.

## The trained sibling: LMS on the midamble

Blind CMA pins only the modulus; a *training sequence* pins the channel inverse
exactly, phase and all. Every TETRA burst carries a known 11-dibit midamble
(NTS1/NTS2), and the #1001 follow-up wires a trained equalizer around it:
`TrafficExtractor.EnableLMSEqualizer` trains an `equalizer.SnapshotLMS` on each
burst's midamble, freezes the taps (the same differential-safety principle),
equalizes the BKN1..BKN2 span with a taps-long FIR warm-up, and **re-derives the
soft LLRs from the equalized symbols** — the hard frame is untouched:

```go
// internal/radio/tetra/traffic.go (shape) — equalizedBurstDiffs
ref := te.midambleRef(L, ntsLen)         // ideal midamble from a unit anchor
rx := te.symBuf[m0 : m0+ntsLen+1]        // raw symbols via StashSymbols
te.lms.Reset()
te.lms.Train(rx, ref, te.lmsPasses)      // 80 passes over 11 dibits
span := te.lms.Equalize(te.lmsSpan[:0], te.symBuf[s0:s1])
/* … differential-decode consecutive equalized symbols → 216 diffs … */
```

The subtlety: the linear channel is a clean convolution only in the **raw
symbol** domain — `s·conj(prev)` is a nonlinear product in which it is not —
so the extractor carries raw symbols down a second parallel buffer
(`SymbolSink → StashSymbols`, the symbol analog of Part 8's soft bridge). The
reference is differentially encoded from an arbitrary unit anchor, because a
constant start-phase rotation cancels downstream — the same property that makes
frozen snapshots safe. `traffic_lms_test.go` pins synthetic multipath through
the real extractor at raw **13% → 0%** payload bit-error with no harm on clean.
It is opt-in and soft-path-only: production composers still run CMA alone, and
`GT_TETRA_LMS=1` in `TestTETRAMultiSlotReplay` runs the capture A/B (compare
`traffic_marked_crc_soft`) that gates flipping it on.
[Weak-Signal Engineering Part 7]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})
owns the trained-equalization theory.

## Where this goes next

The voice path now stacks three levers on a marginal carrier: soft decisions,
a blind equalizer, and a staged trained one. But the *control channel* — the
thing every voice follow depends on — was quietly running with none of them in
one specific configuration, and a reporter's one-hour session showed exactly
what that costs: ~210 control-channel transitions and 11 hard sync losses.
[Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})
reads that session's forensics, rules out the compute theory, and finds the one
TETRA CC path that wasn't running SnapshotCMA.

## FAQ

**How do I know my garble is ISI and not a weak signal?**
Look at the constellation and the yield pattern. Noise fuzzes points
symmetrically and degrades everything at once; ISI smears points into
structured crescents and hits the differential decode hardest. Practically: if
the equalizer roughly doubles CRC yield, it was ISI; if nothing moves, revisit
gain and antenna ([#764](https://github.com/MattCheramie/GopherTrunk/issues/764)'s
lesson — the equalizer mitigates a channel, not a front end).

**Why not just use the LMS equalizer everywhere and skip CMA?**
Different niches. CMA is framing-free — it runs in the receiver on the
continuous stream with no idea where bursts sit, which is what the control
channel and the pre-lock voice path need. LMS needs the midamble location, so
it lives in the extractor, per burst. And LMS is still gated on a capture A/B
(`GT_TETRA_LMS=1`) before it becomes a production default.

**What does the equalizer cost on a clean signal?**
Measurably nothing but CPU. Center-spike initialisation makes the
pre-convergence output the unmodified input, and on a clean ring CMA's gradient
is near zero, so the taps stay near pass-through.
`TestReceiverEqualizerCleanChannelNoHarm` pins it — with one caveat: a
literally noise-free constant-modulus synthetic is a degenerate CMA input, so
the test adds 30 dB AWGN to be a fair fixture.

**Why 200 symbols between snapshots?**
It must be long against a burst (so each 255-dibit burst's data span sees one
constant filter — the straddling symbol is FEC noise) and short against the
channel's drift. 200 at 18 ksym/s refreshes ~90×/s, plenty for the slowly
varying multipath these captures show, and converges within a few hundred
symbols from cold.

**Could the equalizer hide a real RF problem?**
It can mask moderate ISI from a bad antenna path, which is why the health
metrics still surface the underlying quality. The rule stands: raise the
signal too. An equalizer buys back the linear channel, never the noise floor.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Going Soft — Soft-Decision TCH/S]({{ '/blog/deep-dives/tetra-end-to-end-08-soft-decision-tchs/' | relative_url }})
· Next →
[Part 10: The Control Channel Under Stress — Sync Loss & the CC Equalizer]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})
