---
title: "Signal Lab, Part 6: Synthesize — Building Ideal & Impaired References"
description: Signal Lab's synthesis engine generates an idealized IQ capture and then injects controlled impairments — SNR, carrier offset and drift, multipath, DC, and I/Q imbalance — so you can build a clean reference or a known-bad regression fixture and overlay them in the Compare tab.
category: tutorials
keywords: iq synthesis, synthesize capture, impairments, awgn snr, carrier offset drift, multipath, dc offset, iq imbalance, regression fixture, reference signal, siglab synthesize
tags: [siglab, dsp, synthesis, testing, regression, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 6
charts: true
---

*Part 6 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. So far we've measured captures we were given; now we
manufacture them on purpose.*

> **TL;DR:** Signal Lab's synthesis engine generates an *idealized* capture for a
> protocol, then lets you inject controlled impairments — SNR, carrier
> offset/drift, multipath, DC, and I/Q imbalance — with a fixed seed for
> reproducibility. Use it to build a pristine reference (what "perfect" looks
> like) or a known-bad fixture (a specific failure you can regress against), then
> overlay them in the **Compare** tab to prove a demod change did what you
> intended.

**Key takeaways**

- **Synthesis makes the answer known in advance.** You control the signal, so you
  know exactly what the metrics *should* read.
- **Impairments are a menu** — SNR, carrier offset/drift, multipath, DC, I/Q
  imbalance — applied on top of an ideal baseline.
- **A fixed seed makes it reproducible.** The same impairment draw every run is
  what makes a synthesized capture a valid fixture.
- **Compare overlays references** against real captures or against each other.
- **This is how demod changes get proven** — not by vibes, but by a fixture that
  fails before and passes after.

## Cheat sheet

| Concept / knob | What it does |
|---|---|
| Synthesize tab | Generate an idealized capture for a protocol |
| SNR impairment | Add AWGN to a target signal-to-noise |
| Carrier offset / drift | Shift or slowly move the carrier off center |
| Multipath | Add delayed, faded echoes |
| DC offset | Inject a DC spike at 0 Hz |
| I/Q imbalance | Add gain/phase imbalance between I and Q |
| Seed | Fix the random draw for reproducible fixtures |
| Compare tab | Overlay synthesized vs real (or vs ideal) |

## In this post

- **Why synthesize at all** — the value of a known answer.
- **The ideal baseline** — a perfect capture as a reference.
- **The impairment menu** — five ways to break a signal on purpose.
- **Reproducibility** — why the seed matters.
- **Compare** — proving a change with an overlay.

## Why synthesize at all

Every metric in this series answers "how good is this capture?" — but you can
only fully *trust* a measurement when you already know the right answer.
Synthesis gives you that: you specify the signal, so you know precisely what a
correct decoder should report. A pristine synthesized P25 capture *must* lock
instantly at near-zero EVM; if it doesn't, the bug is in the code, not the
signal. That inversion — from "measure an unknown" to "check against a known" —
is what makes the synthesis engine the backbone of GopherTrunk's demod testing.

Two jobs dominate. First, a **clean reference**: the "this is what perfect looks
like" baseline you overlay a real capture against to see how far short it falls.
Second, a **known-bad fixture**: a capture impaired in one specific, documented
way — say, exactly 8 dB SNR with a 1.5 kHz carrier offset — that a regression
test can decode and assert against. When a demod change lands, the fixture that
*failed to lock* before and *locks* after is the proof the change worked.

## The ideal baseline

Synthesis starts from an idealized capture for a chosen protocol and modulation —
a signal with no noise, no offset, no imbalance. On a constellation it's the
textbook picture: four rails (C4FM) or four clusters (CQPSK) so tight they're
nearly points. The synthesis engine and the analyzer share the same code paths as
the rest of the workbench, so an ideal capture run back through the decoder reads
the way theory says it should.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="constellation" width="480" height="360" role="img"
        aria-label="Ideal synthesized C4FM constellation with near-perfect rails"></canvas>
<script type="application/json" class="lab-chart-data">
{ "rails":[-3,-1,1,3], "dot":1.6,
"points":[[-3,0.05],[-3,-0.05],[-3,0.02],[-3,-0.03],[-1,0.04],[-1,-0.04],[-1,0.01],[-1,-0.02],[1,0.03],[1,-0.03],[1,0.02],[1,-0.01],[3,0.04],[3,-0.05],[3,0.02],[3,-0.03]] }
</script>
<figcaption>An ideal synthesized C4FM baseline: four near-perfect rails. This is the "what perfect looks like" reference every real capture is measured against.</figcaption>
</figure>

## The impairment menu

On top of the ideal baseline you inject impairments — individually or stacked —
each modeling a real-world defect:

| Impairment | Models | What it does to the picture |
|---|---|---|
| **SNR** | Weak signal / distance | AWGN thickens rails and clusters; the eye closes |
| **Carrier offset / drift** | Mistuned or drifting LO | Whole constellation rotates or slides; needs auto-tune/AFC |
| **Multipath** | Reflections, simulcast | Delayed echoes smear and fade symbols |
| **DC offset** | Front-end DC leakage | A spike at 0 Hz in the PSD; a pulled constellation |
| **I/Q imbalance** | Gain/phase mismatch | Rails skew; image-rejection drops |

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="constellation" width="480" height="360" role="img"
        aria-label="Impaired synthesized C4FM constellation with noise and imbalance"></canvas>
<script type="application/json" class="lab-chart-data">
{ "rails":[-3,-1,1,3], "dot":1.6,
"points":[[-2.5,0.9],[-3.5,-1.1],[-2.3,1.3],[-3.2,-1.5],[-1.4,1.0],[-0.7,-1.2],[-1.6,0.7],[-0.5,1.4],[0.7,1.1],[1.4,-1.3],[0.6,0.8],[1.5,1.2],[2.6,1.0],[3.4,-1.2],[2.4,1.4],[3.5,-1.5]] }
</script>
<figcaption>The same baseline after injecting SNR loss and I/Q imbalance: rails have thickened and skewed. Because you dialed in the impairment, you know exactly what the analyzer should report.</figcaption>
</figure>

The point is *control*. Because you set the impairment, the analyzer's job becomes
verifiable: dial 8 dB SNR and the SNR estimate had better read near 8 dB; add a
1.5 kHz offset and the lock line's residual frequency had better show it (or
`-auto-tune` had better remove it). When a metric *disagrees* with the impairment
you injected, you've found either a measurement bug or an interesting edge — and
either way you found it against a known answer.

The demod sweep in Part 10 is this idea industrialized: it synthesizes P25 across
a whole SNR ladder and checks measured demod quality against theory at every rung.

## Reproducibility: the seed

A fixture is only a fixture if it's the *same* every time. Random impairments —
AWGN especially — draw from a pseudorandom source, so synthesis takes a **seed**.
Fix the seed and the noise draw is identical run to run, which is what lets a
regression test assert exact-ish numbers instead of loose ranges. You can see the
pattern directly in the sweep's synthesis options:

```go
// cmd/gophertrunk/siglab_sweep.go
synth := siglab.SynthOptions{
    Protocol:    trunking.ProtocolP25,
    Format:      siglab.FormatF32,
    Modulation:  m.synthMod,
    Impairments: demod.Impairments{SNRdB: snr, Seed: *seed},
}
```

The default sweep seed is `0x5175`; any fixed value works, as long as it's
*fixed*. Change the seed and you get a different-but-equally-valid noise
realization — useful for checking a result isn't an artifact of one lucky draw.

## Compare: proving a change

The **Compare** tab (Part 3) is where synthesis pays off. It overlays the power
spectra of several analyzed captures and diffs their metrics, so you can put a
real capture next to a synthesized ideal and *see* the gap — the real one's
rails wider, its occupied bandwidth broader, its EVM higher. Or you overlay two
runs of the same fixture across a code change: same input, same seed, and any
difference in the output is attributable to your change and nothing else.

That's the workflow Reese insists on for any demod tweak. Build a known-bad
fixture that reproduces the symptom. Confirm the current code fails it. Make the
change. Confirm the fixture now passes, and that the *ideal* reference still
decodes perfectly (so you didn't fix the bad case by breaking the good one).
Overlay both in Compare, export the results, and you have evidence rather than a
hunch — which is exactly the standard GopherTrunk holds demod changes to.

## Stacking impairments to reproduce the field

The single-impairment fixtures are the cleanest teaching tools, but real captures
are rarely broken one way at a time — the field stacks defects, and so should a
fixture meant to mimic it. A weak signal from a distant tower isn't *just* low
SNR; it usually comes with a carrier offset (the two radios' oscillators don't
agree), some multipath (the signal arrives by more than one path), and whatever
I/Q imbalance the receiving SDR contributes. Synthesizing that combination —
say, 8 dB SNR *and* a 1.2 kHz offset *and* a light multipath tap — produces a
fixture that stresses the demod the way a genuinely marginal capture does, and
it's the kind of case where a naive demod passes each impairment alone but falls
over when they combine.

There's a discipline to it, though: **change one thing at a time when you're
diagnosing, stack them when you're stress-testing.** If you want to know *why* a
demod fails, isolate the impairment that breaks it — add offset alone, then
multipath alone — so the cause is unambiguous. Once you understand each failure
mode, the stacked fixture becomes your acceptance bar: "the demod must lock at
8 dB with 1.2 kHz offset and one multipath tap." Because every impairment carries
a known magnitude and the seed is fixed, that bar is exactly repeatable, and a CI
job can hold the demodulator to it forever.

## Synthesis as documentation

A last, underrated use: a synthesized fixture is *executable documentation* of a
bug. When a real capture reveals a demod weakness but you can't check the
multi-gigabyte file into a repo, you distill it into a small synthesized fixture
that reproduces the same symptom — a few seconds of IQ with a documented
impairment recipe and a fixed seed. The fixture *is* the bug report: anyone can
run it, see the failure, and confirm a fix. That's how the sweep in Part 10
industrializes the idea — it's nothing more than a whole ladder of synthesized
fixtures, each with a known answer, run against the production demod on every
build.

## Where this goes next

Synthesis gives you signals with known impairments; the natural next question is
how *precisely* the workbench can measure them. [Part 7]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }})
opens the vector signal analyzer — carrier-frequency error, RMS/peak EVM split
into magnitude and phase error, I/Q gain imbalance and quadrature skew, origin
offset, and the EVM-vs-symbol trace — the lab-grade instrumentation that turns
"the constellation looks fuzzy" into a number for each distinct defect. The
[SigLab docs]({{ '/siglab.html' | relative_url }}) enumerate the synthesis knobs.

## FAQ

**Why would I decode a signal I made up?**
Because you know the right answer in advance. A synthesized capture lets you
verify the analyzer and the demod against a controlled input — invaluable for
regression fixtures and for proving a demod change actually helped.

**What impairments can I inject?**
SNR (AWGN), carrier offset and drift, multipath, DC offset, and I/Q imbalance —
individually or stacked on top of the ideal baseline.

**Why does the seed matter?**
It fixes the pseudorandom draw (mostly the AWGN) so a synthesized capture is
byte-for-byte reproducible. Without a fixed seed you can't assert stable numbers
in a regression test.

**How do I prove my demod change worked?**
Build a fixture that fails before the change, confirm it passes after, verify the
ideal reference still decodes cleanly, and overlay the runs in the Compare tab.
That's evidence; a subjective "looks better" isn't.

## Series navigation

**Part 6 of 10** · ←[Part 5]({{ '/blog/tutorials/signal-lab-05-psd-spectrogram-occupancy/' | relative_url }}) · Next →
[Part 7: VSA & Modulation Quality]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }})
