---
title: "Signal Lab, Part 7: VSA — Lab-Grade Modulation Quality"
description: Signal Lab's vector signal analyzer measures modulation quality like bench gear — carrier-frequency error, RMS and peak EVM split into magnitude and phase error, I/Q gain imbalance and quadrature skew, origin offset, an EVM-vs-symbol trace, and an error-vector spectrum.
category: tutorials
keywords: vector signal analyzer, vsa, evm, rms evm, peak evm, magnitude error, phase error, carrier frequency error, iq gain imbalance, quadrature skew, origin offset, error vector spectrum
tags: [siglab, dsp, vsa, evm, modulation, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 7
charts: true
---

*Part 7 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. Part 4 let you *see* modulation quality; the VSA lets
you *measure* it to bench-instrument precision.*

> **TL;DR:** Signal Lab's vector signal analyzer reports the same metrics a
> bench VSA does: **carrier-frequency error**; **RMS and peak EVM**, each split
> into **magnitude** and **phase** error; **I/Q gain imbalance** and
> **quadrature skew**; **origin (DC) offset**; an **EVM-vs-symbol trace**; and an
> **error-vector spectrum**. Each isolates a *different* defect, so instead of
> "the constellation looks off" you get "carrier is 1.2 kHz high and there's 0.4
> dB of gain imbalance."

**Key takeaways**

- **EVM is one number; the VSA is the breakdown.** Splitting EVM into magnitude
  and phase error tells you *what kind* of error dominates.
- **Carrier-frequency error** isolates tuning from modulation quality.
- **Gain imbalance and quadrature skew** are front-end defects, not signal
  weakness — the VSA separates them cleanly.
- **The EVM-vs-symbol trace** finds *when* quality degraded across the burst.
- **The error-vector spectrum** shows *where in frequency* the error energy
  lives.

## Cheat sheet

| VSA metric | What it isolates |
|---|---|
| Carrier-frequency error | Residual tuning offset |
| RMS EVM | Overall modulation quality (average) |
| Peak EVM | Worst-case symbol |
| Magnitude error | Amplitude component of EVM |
| Phase error | Phase component of EVM |
| I/Q gain imbalance | Amplitude mismatch between I and Q |
| Quadrature skew | Departure from 90° between I and Q |
| Origin offset | DC / carrier feedthrough |
| EVM-vs-symbol trace | Quality over the burst |
| Error-vector spectrum | Error energy vs frequency |

## In this post

- **What a VSA measures** and why the split matters.
- **Carrier-frequency error** vs modulation quality.
- **EVM decomposed** into magnitude and phase.
- **Front-end defects** — gain imbalance, quadrature skew, origin offset.
- **Traces** — EVM-vs-symbol and the error-vector spectrum.

## What a VSA measures

A **vector signal analyzer** treats each received symbol as a vector in the I/Q
plane and measures the **error vector** — the difference between where the symbol
landed and where it ideally should have. The length of that error vector,
normalized and averaged, is **EVM** (error-vector magnitude). You already met EVM
as a single dashboard number in Part 2; the VSA is what happens when you stop
averaging everything into one figure and start asking *what the error is made of*.

That decomposition is the whole value. Two captures can share the same 8% RMS EVM
and be broken in completely different ways — one by tuning error, one by a
front-end gain mismatch, one by low SNR. A single EVM number can't tell them
apart; the VSA's breakdown can. Signal Lab exposes the full set the way bench gear
does, computed off the analyzed capture.

## Carrier-frequency error vs modulation quality

The first thing the VSA separates out is **carrier-frequency error** — the
residual offset between where the signal actually sits and 0 Hz baseband. This is
a *tuning* problem, not a *modulation* problem, and conflating the two is a
classic mistake. A perfectly modulated signal recorded slightly off-center will
show a rotating constellation and inflated EVM even though the transmitter is
flawless; the fix is to tune it out (`-auto-tune`, or the receiver's AFC), not to
chase a nonexistent modulation defect.

By reporting carrier-frequency error as its own number, the VSA lets you subtract
the tuning question entirely: correct the offset, and whatever EVM remains is
*genuinely* modulation quality. Reese's first VSA question is always the same —
*is the carrier where it should be?* — because until it is, every other metric is
contaminated.

## EVM decomposed: magnitude and phase

Once tuning is accounted for, the VSA splits EVM into two orthogonal components:

- **Magnitude error** — the *amplitude* part of the error vector: symbols landing
  too close to or too far from the origin along their ideal direction. It points
  at amplitude problems — compression, AGC misbehavior, fading.
- **Phase error** — the *angular* part: symbols rotated off their ideal phase.
  It points at phase noise, residual frequency error, and phase-tracking trouble.

It also reports **RMS EVM** (the average, your headline quality number) alongside
**peak EVM** (the single worst symbol). The gap between them is diagnostic: RMS
and peak close together means uniformly noisy; a low RMS with a high peak means
mostly clean with occasional bad symbols — often a transient interferer rather
than a steady-state weakness. Whether magnitude or phase error dominates tells
you which physical mechanism to suspect, which is exactly the kind of pointer a
lone EVM number can never give.

## Front-end defects: imbalance, skew, offset

Three VSA metrics describe the *receiver*, not the signal — defects introduced by
the I/Q front-end itself:

- **I/Q gain imbalance** — the amplitude of the I channel doesn't match the Q
  channel. On a constellation the rails or clusters stretch along one axis; the
  image-rejection number from Part 2 drops.
- **Quadrature skew** — I and Q aren't exactly 90° apart. The constellation
  shears, as if the plane were pushed out of square.
- **Origin offset** — residual DC / carrier feedthrough that pulls the whole
  constellation off center; it shows up as a spike at 0 Hz in the PSD.

These are the fingerprints of a particular SDR and gain setting rather than of the
signal in the air, which is why isolating them matters: a capture with fine SNR
but visible gain imbalance and skew was recorded on a front-end that needs I/Q
correction, and no amount of resignal-hunting will improve it. Synthesis (Part 6)
lets you inject each of these deliberately and confirm the VSA reads them back —
the cleanest way to build intuition for what each defect looks like.

## Traces: EVM-vs-symbol and the error-vector spectrum

The last two outputs are traces rather than single numbers, and they answer
*when* and *where*.

The **EVM-vs-symbol trace** plots error magnitude across the burst, symbol by
symbol. A flat trace means steady quality; a ramp or a spike means quality
changed *during* the capture — a fade, a collision, a transmitter settling. This
is how you catch a capture that averages "fine" but was briefly awful at one
instant.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="line" width="560" height="300" role="img"
        aria-label="EVM versus symbol index, mostly flat with a mid-burst spike"></canvas>
<script type="application/json" class="lab-chart-data">
{ "series":[{"label":"EVM per symbol","points":[[0,6.8],[200,7.1],[400,6.9],[600,7.4],[800,12.5],[1000,18.2],[1200,11.0],[1400,7.2],[1600,6.9],[1800,7.0],[2000,7.1]]}],
"xlabel":"symbol index","ylabel":"EVM (%)","xmin":0,"xmax":2000,"ymin":0,"ymax":25 }
</script>
<figcaption>An EVM-vs-symbol trace: steady near 7% except for a mid-burst spike. The average would read "fine"; the trace exposes the transient the average hid — likely a brief collision or fade.</figcaption>
</figure>

The **error-vector spectrum** takes the error signal itself and shows its energy
versus frequency. A flat error spectrum is consistent with white noise (just low
SNR); a *peaked* error spectrum means a specific spur or tone is injecting error
at a particular frequency — an interferer or a front-end artifact you can then go
find. Where the EVM-vs-symbol trace localizes error in *time*, the error-vector
spectrum localizes it in *frequency*, and together they turn a raised EVM into an
actionable lead.

Ada's takeaway from her first VSA session: the dashboard's single EVM told her
*something* was wrong; the VSA told her the carrier was 1.2 kHz high and the rest
was a clean 7% — so the fix was a retune, not a new antenna.

## A worked diagnosis

Put the whole panel together on one capture and watch it name a fault. Ada's
recording reads 9.5% RMS EVM — not great, not terrible — and she wants to know
what's costing her. The VSA breaks it down:

| VSA metric | Reading | What it says |
|---|---|---|
| Carrier-frequency error | +180 Hz | Small; not the culprit |
| Magnitude error | 3.1% | Modest amplitude spread |
| Phase error | 8.8% | **Dominant** — the error is mostly angular |
| I/Q gain imbalance | 0.15 dB | Front-end is fine |
| Quadrature skew | 0.3° | Front-end is fine |
| Origin offset | −42 dB | Negligible DC |

The story is unambiguous: the carrier is nearly centered, the front-end is clean,
and magnitude error is small — but **phase error dominates**. That points at phase
noise or a phase-tracking loop that's working too hard, not at a weak signal or a
misbehaving SDR. Ada wouldn't have reached that conclusion from a 9.5% EVM number
or even from staring at the constellation, where a phase-heavy error just looks
like a general smear. The decomposition did what the picture couldn't: it named
the *mechanism*. And because synthesis (Part 6) can inject phase noise
deliberately, she can confirm the theory — dial in known phase noise, watch phase
error rise while magnitude error stays flat, and match the signature.

That's the VSA's real gift. A single EVM number is a thermometer: it tells you the
patient has a fever. The VSA is the full workup — it tells you *which* system is
sick, which is the difference between "this capture is degraded" and "retune by
180 Hz and chase the phase noise; everything else is healthy."

## Where this goes next

You can now characterize a *known* signal to bench precision. But Mercury still
won't say what it is. [Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
turns to the unknown: blind signal identification, the offline signal-ID
reference database that *names* an undecodable carrier from its symbol rate and
modulation, and the wideband survey — where Ada finally gets a best guess for
Mercury and hands it off. The [SigLab docs]({{ '/siglab.html' | relative_url }})
carry the full VSA metric list.

## FAQ

**How is the VSA different from the dashboard's EVM?**
The dashboard gives one EVM number. The VSA decomposes it — magnitude vs phase
error, RMS vs peak — and adds carrier-frequency error, I/Q imbalance, quadrature
skew, origin offset, and per-symbol/per-frequency traces, so you learn *what kind*
of error you have, not just how much.

**Why split EVM into magnitude and phase?**
Because they point at different causes. Magnitude error suggests amplitude
problems (compression, AGC, fading); phase error suggests phase noise or residual
frequency error. The split tells you which mechanism to chase.

**Are gain imbalance and quadrature skew signal problems?**
No — they're front-end defects, introduced by the receiver's I/Q path. The VSA
isolates them so you don't mistake a receiver imperfection for a weak signal.

**What does the EVM-vs-symbol trace catch that RMS EVM misses?**
Transients. A capture can average a healthy EVM while being briefly terrible at
one instant (a collision or fade). The trace shows that spike; the average buries
it.

## Series navigation

**Part 7 of 10** · ←[Part 6]({{ '/blog/tutorials/signal-lab-06-synthesize-references/' | relative_url }}) · Next →
[Part 8: Naming the Unknown]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }})
