---
title: "Signal Lab, Part 4: Constellations & Eye Diagrams — Seeing Modulation Quality"
description: Signal Lab's constellation, eye diagram, and symbol scope — how to read clean versus noisy C4FM (four rails) and CQPSK modulation, what opening and closing eyes mean, and how these plots make EVM and SNR visible.
category: tutorials
keywords: constellation diagram, eye diagram, symbol scope, c4fm four rails, cqpsk constellation, modulation quality, opening eye closing eye, evm visualization, p25 modulation, signal lab visuals
tags: [siglab, dsp, modulation, demodulation, p25, getting-started]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 4
charts: true
---

*Part 4 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. The dashboard gave you EVM and SNR as numbers; now we
give them a picture.*

> **TL;DR:** The constellation, eye diagram, and symbol scope are three views of
> the same thing — how cleanly the demod is separating symbols. A clean C4FM
> signal shows **four tight rails**; a clean CQPSK signal shows four tight
> clusters in the I/Q plane. As SNR drops, rails blur and clusters spread — the
> **eye closes**. An open eye means comfortable symbol decisions; a closed eye
> means the demod is guessing.

**Key takeaways**

- **Constellation = where symbols land** in the I/Q plane; tight points mean low
  EVM.
- **Eye diagram = symbol transitions overlaid**; a wide-open eye means a big
  decision margin.
- **C4FM shows four rails; CQPSK shows four clusters** — different modulations,
  same "tight is healthy" rule.
- **These plots explain the numbers.** A high EVM *looks* like a fuzzy
  constellation and a closing eye.
- **The symbol scope** ties it to time — which symbols, in what order, drifted.

## Cheat sheet

| Surface / plot | What it shows |
|---|---|
| Constellation | Symbol positions in the I/Q plane (rails or clusters) |
| Eye diagram | Overlaid symbol transitions; eye opening = decision margin |
| Symbol scope | Recovered symbols over time |
| `gophertrunk siglab serve` | Renders all three in the browser console |
| [/constellation.html]({{ '/constellation.html' | relative_url }}) | Docs: constellation reference |
| [/eye-diagram.html]({{ '/eye-diagram.html' | relative_url }}) | Docs: eye-diagram reference |

## In this post

- **What a constellation is** and how to read it.
- **Clean vs noisy C4FM** — the four rails.
- **CQPSK constellations** — clusters, not rails.
- **The eye diagram** — opening and closing, and why it matters.
- **The symbol scope** — the time-domain companion.

## What a constellation is

A **constellation diagram** plots each recovered symbol as a point in the I/Q
plane — in-phase on one axis, quadrature on the other. Its shape is a fingerprint
of the modulation, and its *tightness* is a direct readout of quality. Ideal
symbols land exactly on their target points; real symbols scatter around them,
and the spread of that scatter is precisely what EVM measures. So the
constellation is the picture behind the EVM number from Part 2: low EVM is a tidy
constellation, high EVM is a smear.

The plot is theme-aware and lives in the browser console's visualization panel,
alongside the docs' own
[constellation reference]({{ '/constellation.html' | relative_url }}). You don't
read absolute coordinates off it — you read *shape* and *spread*.

## Clean vs noisy C4FM: the four rails

C4FM — the 4-level continuous-phase FM that carries P25 Phase 1, and the family
GopherTrunk's channel rate is built around — encodes two bits per symbol as one
of four frequency deviations. On a constellation it shows up as **four vertical
rails**: four distinct symbol levels. A healthy C4FM capture draws four tight,
well-separated rails. A weak one blurs them until adjacent rails touch and the
demod can no longer tell a level from its neighbor.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="constellation" width="480" height="360" role="img"
        aria-label="Clean C4FM constellation with four tight vertical rails"></canvas>
<script type="application/json" class="lab-chart-data">
{ "rails":[-3,-1,1,3], "dot":2.0,
"points":[[-3,0.5],[-3,-0.4],[-3,1.1],[-3,-1.0],[-3,0.1],[-1,0.6],[-1,-0.5],[-1,0.2],[-1,1.0],[-1,-0.9],[1,0.4],[1,-0.6],[1,0.0],[1,0.9],[1,-1.1],[3,0.3],[3,-0.4],[3,0.8],[3,-0.7],[3,0.2]] }
</script>
<figcaption>Clean C4FM: four tight vertical rails, each a symbol level, with generous space between them. This is what low EVM looks like.</figcaption>
</figure>

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="constellation" width="480" height="360" role="img"
        aria-label="Noisy C4FM constellation with smeared, overlapping rails"></canvas>
<script type="application/json" class="lab-chart-data">
{ "rails":[-3,-1,1,3], "dot":2.0,
"points":[[-2.6,0.9],[-3.4,-1.3],[-2.2,1.6],[-3.1,-1.9],[-2.0,0.4],[-1.5,1.4],[-0.6,-1.6],[-1.2,0.9],[-0.4,1.8],[-1.7,-1.4],[0.6,1.3],[1.5,-1.5],[0.4,0.7],[1.6,1.7],[0.5,-1.9],[2.5,1.2],[3.5,-1.4],[2.2,1.8],[3.6,-1.7],[2.3,0.6]] }
</script>
<figcaption>Noisy C4FM: the same four rails, but spread and drifting toward each other. As they merge the eye closes and symbol errors climb.</figcaption>
</figure>

Reese's shortcut: **count the gaps.** Four clear gaps between rails means a
comfortable decoder. When the gaps vanish, so does the lock margin — and you'll
see it in the histogram (Part 2) and the decode-error rate at the same time.

## CQPSK: clusters, not rails

Not every P25 signal is C4FM. The CQPSK / LSM path (linear-simulcast-friendly
modulation, used by some P25 Phase 1 systems and the natural companion to
simulcast infrastructure) is a phase modulation: symbols land as **four clusters**
arranged in the I/Q plane rather than four vertical rails. The health rule is
identical — tight, well-separated clusters are good; clusters that bloom into
each other are bad — but the *shape* tells you which demod path is in play. If you
expected rails and see clusters (or vice versa), you're looking at the wrong
modulation for the system, which is itself a useful diagnosis. Part 7's VSA
metrics quantify exactly how far each cluster sits from ideal.

## The eye diagram: opening and closing

The **eye diagram** overlays many symbol transitions on top of each other,
aligned to the symbol clock. The result looks like one or more "eyes." The
**height** of an open eye is the vertical decision margin — how much room the
slicer has to place a symbol correctly — and the **width** is timing margin, how
much clock jitter it can tolerate before sampling at the wrong instant.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="eye" width="560" height="320" role="img"
        aria-label="Eye diagram with clearly open eyes for a healthy signal"></canvas>
<script type="application/json" class="lab-chart-data">
{ "xlabel":"symbol period", "ylabel":"amplitude",
"traces":[[-0.9,-0.3,0.4,0.9,0.4,-0.3,-0.9],[0.9,0.4,-0.2,-0.9,-0.3,0.3,0.9],[-0.9,-0.2,0.5,0.95,0.5,-0.2,-0.9],[0.9,0.3,-0.3,-0.95,-0.4,0.2,0.9],[-0.85,-0.35,0.35,0.85,0.35,-0.35,-0.85],[0.85,0.35,-0.25,-0.85,-0.35,0.25,0.85]] }
</script>
<figcaption>An open eye: transitions cross cleanly and leave a wide gap in the middle. The taller and wider that opening, the more margin the demod has for correct symbol decisions.</figcaption>
</figure>

As SNR falls, noise thickens every trace and the opening shrinks — the eye
**closes**. A fully closed eye means transitions cover the decision point and the
demod can no longer reliably tell one symbol from the next; that's the visual
partner of a rising decode-error rate and a lost lock. Reading the eye is a fast
gut check: *can I still see daylight in the middle?* If yes, the capture has
margin; if no, it's marginal no matter what the lock line optimistically says
for an instant.

## The symbol scope: quality over time

The constellation and eye pool *all* symbols together; the **symbol scope** puts
them back in order, plotting recovered symbols over time. That time axis matters
when quality isn't constant — a capture that's clean for four seconds and then
craters when a neighboring transmitter keys up will show a tidy constellation on
average but an obvious degradation in the scope right at that moment. Pair the
scope with the console's live event stream (Part 3) and you can line up a burst
of symbol errors with the exact stage or grant where it happened.

Together these three plots — constellation, eye, symbol scope — turn the
abstract EVM/SNR pair into something you can *see* go wrong. Ada's habit, once
Reese drilled it in: read the numbers first for the verdict, then open the
constellation to confirm the story and the eye to judge how much margin is left.

## The same plots across protocols

C4FM and CQPSK are the P25 Phase 1 cases, but the constellation is a fingerprint
for *every* protocol the engine decodes, and learning to read the shape is a fast
way to sanity-check what you're actually looking at:

- **DMR** and **NXDN** are 4-level FSK families like C4FM — expect four rails.
  DMR's TDMA structure means the constellation is built from alternating time
  slots, so a healthy DMR capture still shows four clean rails even though the
  bursts interleave two logical channels.
- **P25 Phase 2** uses H-CPM (a continuous-phase modulation), so its picture is
  denser than the four crisp rails of Phase 1 — a reminder that "which protocol"
  changes what "clean" looks like.
- **TETRA** is π/4-DQPSK, a phase modulation whose ideal constellation is eight
  points on two rotated QPSK rings; a healthy TETRA capture shows those points
  tight and the ring well-formed.

The practical lesson is that the *number and arrangement* of clusters or rails
should match the modulation you expect. If you told the engine "P25 Phase 1
C4FM" and the constellation shows phase-plane clusters instead of vertical rails,
either the system is actually running the CQPSK/LSM variant or you've mislabeled
the capture — and the plot caught it before the metrics did. When a signal won't
decode and you're not sure what it is at all, that's the cue to hand it to blind
identification (Part 8), which reads exactly this kind of structure to name a
candidate.

## A reading workflow

Reese's routine for any new capture is a fixed order, because each plot answers a
different question and answering them out of order wastes time:

1. **Histogram and dashboard first** — is the rate right (the 2% rule) and did it
   lock? No point plotting a capture that's decoding at the wrong sample rate.
2. **Constellation** — does the shape match the expected modulation, and how tight
   is the spread? This is your EVM sanity check.
3. **Eye** — is there daylight in the middle? This is your margin check: a tight
   constellation with a barely-open eye is closer to the edge than it looks.
4. **Symbol scope** — if quality is uneven, *when* did it degrade? Line the dip up
   against the live event stream to find the cause.

Four plots, four questions, in the order that fails fast. Ada now runs it by
reflex; it's the single habit that turned her "why won't this lock" flailing into
a diagnosis in under a minute.

## Where this goes next

You can now *see* modulation quality, not just read it.
[Part 5]({{ '/blog/tutorials/signal-lab-05-psd-spectrogram-occupancy/' | relative_url }})
zooms out from individual symbols to the *spectrum* — the PSD, the
spectrogram/waterfall, and the spectral-occupancy metrics (occupied bandwidth,
channel power, ACPR, flatness) that describe a signal's shape in frequency rather
than its symbols in the I/Q plane. The docs' [constellation]({{ '/constellation.html' | relative_url }})
and [eye-diagram]({{ '/eye-diagram.html' | relative_url }}) pages go deeper on
each plot.

## FAQ

**Why four rails for C4FM but four clusters for CQPSK?**
They're different modulations. C4FM is a 4-level frequency modulation, so its
symbols separate along one axis (four rails). CQPSK is a phase modulation, so its
four symbols separate as points around the I/Q plane (four clusters). Same
symbol count, different geometry.

**What exactly does a "closing eye" mean?**
The vertical opening in the eye is the demod's decision margin. As noise grows,
traces thicken and the opening shrinks; a closed eye means noise now covers the
decision point and the demod can't reliably separate symbols — expect errors and
lost lock.

**How does this relate to EVM?**
Directly. EVM measures how far symbols scatter from their ideal points, which is
exactly the spread you see in the constellation. Low EVM is a tight constellation
and an open eye; high EVM is a smear and a closing eye.

**Where do I view these plots?**
In the browser console (`gophertrunk siglab serve`, Part 3), which renders the
constellation, eye diagram, and symbol scope from the analyzed capture.

## Series navigation

**Part 4 of 10** · ←[Part 3]({{ '/blog/tutorials/signal-lab-03-browser-console-live-decode/' | relative_url }}) · Next →
[Part 5: PSD, Spectrogram & Occupancy]({{ '/blog/tutorials/signal-lab-05-psd-spectrogram-occupancy/' | relative_url }})
