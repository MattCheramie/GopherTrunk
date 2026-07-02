---
title: "RF Scope, Part 5: Timing & Periodicity — Recovering the TDMA Frame"
description: rfscope's timing analyzer builds burst-length and inter-arrival histograms per channel and recovers a TDMA or frame period by autocorrelating each channel's occupancy series, keeping only peaks above a minimum confidence — the time-domain counterpart to cryptolab's byte-period detection.
category: tutorials
keywords: rfscope, tdma period, timing analysis, autocorrelation, inter-arrival histogram, burst length, periodicity, frame period, rf analysis, gophertrunk, occupancy correlation
tags: [rfscope, rf-analysis, dsp, advanced, gophertrunk, sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 5
charts: true
---

*Part 5 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. The
timeline gave each channel a shape in time; this post measures the rhythm inside it.*

> **TL;DR:** The `timing` analyzer (which depends on `timeline`) annotates each channel
> with two 10-bin histograms — burst length and inter-arrival gap — and recovers a
> **frame/TDMA period** by autocorrelating the channel's binary occupancy series. A
> detected period is kept only if its normalized autocorrelation peak clears a minimum
> confidence of **0.2**; otherwise the channel is reported aperiodic. It is the
> time-domain twin of the byte-period detection cryptolab runs on payloads.

**Key takeaways**

- **Two histograms per channel:** burst length and inter-arrival gap, 10 bins each.
- **Period detection is autocorrelation** of the channel's occupancy — a strong peak
  at a non-zero lag is a repeating frame.
- **A confidence floor of 0.2** discards weak peaks, so RF Scope reports "aperiodic"
  rather than inventing a period from noise.
- **`timing` depends on `timeline`** because it annotates the channels timeline built.

## Cheat sheet

| Field / constant | Meaning |
|---|---|
| `BurstLenHist` | 10-bin histogram of burst durations |
| `InterArrivalHist` | 10-bin histogram of gaps between burst starts |
| `PeriodSec` | Recovered frame/TDMA period (0 if none) |
| `PeriodConfidence` | Normalized autocorrelation height (0..1) |
| `minPeriodConfidence` | 0.2 — peaks below this are discarded |
| `-analyzers timing` | Auto-pulls `timeline` as a dependency |

## In this post

- **Why timing matters** for an unknown signal.
- **The two histograms** — burst length and inter-arrival.
- **Autocorrelation** — how a repeating frame falls out of the occupancy series.
- **The confidence floor** — why RF Scope refuses to guess.
- **Reading a recovered period** against a known TDMA plan.

## Why timing

Two signals can share a modulation class, a bandwidth, and a duty cycle and still be
completely different protocols — because they *schedule* their air time differently.
A P25 Phase 2 TDMA channel packs two logical slots into a repeating 30 ms frame; a
paging system fires a burst on a fixed cadence; a random-access data link has no
cadence at all. Timing is the dimension that separates them, and it is measurable
without decoding a single bit.

The `timing` analyzer captures that dimension in three numbers per channel: a
distribution of how long bursts are, a distribution of how far apart they start, and —
the headline — a recovered **period**. Because it annotates the channels the timeline
analyzer built, it lists `timeline` in its `DependsOn`; asking for `-analyzers timing`
pulls `timeline` in automatically (the registry design we cover in Part 10).

## The two histograms

Both histograms are built with `dsp/stats.Histogram`, which returns paired counts and
bin edges — the same `(counts, edges)` shape the rest of GopherTrunk's statistics use, so
the data drops straight into any plotting code without reshaping. The bin count adapts to
the sample: up to ten bins, but never more than there are data points, so a channel with
four bursts gets a four-bin histogram rather than ten mostly-empty ones. That keeps the
shape honest on sparse channels, where over-binning would manufacture spurious structure.

For each channel, the analyzer collects burst durations and the gaps between
consecutive burst *starts*, then bins each:

```go
// internal/rfscope/timing.go
ch.BurstLenHist = histOf(lengths)
gaps := make([]float64, 0, len(starts))
for i := 1; i < len(starts); i++ {
    gaps = append(gaps, starts[i]-starts[i-1])
}
ch.InterArrivalHist = histOf(gaps)
```

Each histogram has up to **10 bins** (`timingHistBins`, capped at the sample count for
tiny channels). What you read from them:

- **Burst-length histogram.** A tight cluster at one length says fixed-size frames — a
  slot, a packet, a page. A broad spread says variable-length transmissions — voice
  calls of different durations, say. Two distinct clusters can mean two signal types
  sharing a channel.
- **Inter-arrival histogram.** A sharp spike at one gap is a strong hint of a fixed
  cadence — the same thing the autocorrelation confirms. A smooth exponential-looking
  falloff is the signature of random (Poisson) arrivals, i.e. no schedule.

The histograms are the human-readable evidence; the period detector is the automated
verdict.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="560" height="260" role="img"
        aria-label="Inter-arrival histogram with a dominant bin at the frame period"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["8","16","24","30","38","46","54","60","68","76"],
"values":[1,2,3,41,4,2,1,3,1,1],
"ylabel":"burst count","xlabel":"inter-arrival gap (ms)" }
</script>
<figcaption>An inter-arrival histogram spiking at ~30 ms — the classic fingerprint of a fixed TDMA frame, before autocorrelation confirms it.</figcaption>
</figure>

## Autocorrelation: pulling the frame out of the gaps

Histograms hint at a cadence; **autocorrelation proves it**. The detector takes the
channel's 100-bin binary occupancy series, subtracts its mean, and correlates it with
itself at every lag:

```go
// internal/rfscope/timing.go
mean := stats.Mean(occ)
for i, v := range occ {
    sig[i] = v - mean
}
corr, _ := stats.Correlate(sig, sig)
pos := corr[n-1:] // lag 0 at index 0, value ~1
peaks := stats.FindPeaks(pos[1:], stats.PeakOpts{Height: minPeriodConfidence, MinDistance: 1})
```

The intuition: if a channel is active every 30 ms, then shifting its occupancy series
by 30 ms lines the "on" buckets back up with each other, and the correlation at that
lag spikes. Shift by any non-multiple and the on-buckets fall on off-buckets and the
correlation drops. So the **first strong peak at a non-zero lag is the period**, and
its normalized height (0..1, where lag 0 is 1 by construction) is the confidence. The
recovered `PeriodSec` is that lag times the bin duration.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="line" width="560" height="300" role="img"
        aria-label="Occupancy autocorrelation with a confident peak at the frame lag"></canvas>
<script type="application/json" class="lab-chart-data">
{ "series":[{"label":"autocorrelation","dots":true,"points":[[0,1.0],[1,0.18],[2,0.12],[3,0.64],[4,0.16],[5,0.10],[6,0.58],[7,0.14],[8,0.09],[9,0.52],[10,0.12]]},
{"label":"confidence floor 0.2","dashed":true,"points":[[0,0.2],[10,0.2]]}],
"xlabel":"lag (occupancy bins)","ylabel":"normalized correlation","xmin":0,"xmax":10,"ymin":0,"ymax":1.05 }
</script>
<figcaption>The first non-zero-lag peak (lag 3 here, ~0.64) clears the 0.2 floor and becomes the recovered period; the decaying peaks at lags 6 and 9 are its harmonics.</figcaption>
</figure>

## The confidence floor: refusing to guess

Autocorrelation *always* produces a strongest non-zero peak — even for pure noise. The
question is whether that peak means anything. RF Scope answers it with a single
threshold:

```go
// internal/rfscope/timing.go
const minPeriodConfidence = 0.2
```

If no lag clears **0.2**, `detectPeriod` returns `(0, 0)` and the channel is reported
with no period. This is the same discipline the segmentation noise floor uses and the
same one Crypto Lab applies to its randomness verdicts: *when the evidence is weak, say
so, do not invent structure.* A reported `PeriodSec` of 0 is a real answer — "this
channel is aperiodic" — not a missing measurement. Reese's version: *"a tool that
always finds a period is a tool that has never found one."*

The floor is intentionally low. At 0.2 the detector will surface a genuine but noisy
frame — a hopper that only visits this channel intermittently, a TDMA channel seen for
just a few frames — while still rejecting the fractional correlation you get from
random arrivals. If you are chasing a marginal periodicity you can read the raw
`PeriodConfidence`; anything from 0.2 to ~0.4 is "plausible, verify," and above ~0.6
is a confident frame.

## Reading a recovered period

A recovered period is a strong lead, but it is not a decode. Match it against what you
know: a period near **30 ms** on a narrowband digital voice channel points at P25
Phase 2 TDMA; a period of a few seconds on a paging channel points at a paging frame
structure; a sub-millisecond period is more likely a symbol-rate artifact than a frame.
Watch for **harmonics** too — a true 30 ms period also produces (weaker) peaks at 60
and 90 ms, as in the chart above. The detector returns the *first* qualifying peak, so
it lands on the fundamental, but if you read the raw autocorrelation you will see the
harmonic ladder that confirms it.

For Mercury, the timing view is quietly informative: its bursts are short and its
inter-arrival histogram is *broad*, with no dominant bin, and its occupancy
autocorrelation never clears 0.2 on any single channel. That is consistent with what
Part 6 will confirm — Mercury is not sitting on one channel with a clean frame; it is
**hopping**, so its schedule only makes sense once you look across channels, not down
one.

## Why occupancy, not raw timestamps

There is a subtle but important choice in *what* gets autocorrelated. The detector does
not correlate the raw burst start times; it correlates the channel's **binary occupancy
series** — the same 100-bin on/off vector the timeline analyzer built. That has two
consequences worth understanding.

First, it makes the method **duration-aware**. A frame is not just "a burst starts every
30 ms"; it is "the channel is *busy* in a repeating pattern." A TDMA slot that is on for
part of each frame and off for the rest produces a cleaner autocorrelation from the
occupancy series than from start times alone, because the *shape* of the busy interval
repeats, not just its onset.

Second, it ties the period's resolution to the timeline's bin width. With 100 bins over
the window, the finest period the detector can resolve is one bin, and the reported
`PeriodSec` is always an integer number of bins times the bin duration. On a short
window the bins are fine and the period is precise; on a long capture the bins are
coarser and the period is quantized more heavily. If you need a sharper period estimate
on a long capture, analyze a shorter `-window` so the same 100 bins cover less time —
the classic time-resolution trade.

This is also why timing genuinely *depends* on timeline rather than recomputing its own
activity signal: the occupancy vector is exactly the artifact timeline already produces,
and reusing it keeps the two views consistent. A period the timing analyzer reports is a
period you can see in the sparkline the channel panel draws.

## Where this goes next

So far every view has been per-channel. [Part
6]({{ '/blog/tutorials/rf-scope-06-topology-emitters-conversations/' | relative_url }})
crosses channels: the `topology` analyzer clusters bursts into **emitters** by RF
fingerprint — collapsing a frequency hopper's scattered bursts into one logical source
— and then links emitters into **conversations**. That is where Mercury's hopping,
invisible to a single-channel timing view, finally becomes legible.

## FAQ

**How is the period actually recovered?**
By autocorrelating the channel's binary occupancy series and taking the first non-zero
lag whose normalized correlation clears 0.2. That lag times the timeline bin duration
is the period; the correlation height is the confidence.

**Why 10 histogram bins?**
It is enough resolution to distinguish a tight, fixed-size distribution from a broad
or bimodal one without over-fragmenting small samples; the bin count is capped at the
number of bursts for sparse channels.

**A channel I know is TDMA shows no period — why?**
The autocorrelation peak did not clear the 0.2 confidence floor — usually too few
frames were captured, or the channel is a hopper seen only intermittently. Capture a
longer window, or look for the pattern across channels via the topology analyzer.

**Is this the same period detection Crypto Lab uses?**
Conceptually yes — it is the time-domain counterpart. Crypto Lab autocorrelates a
*byte* stream to find a repeating key or scrambler period; RF Scope autocorrelates a
*channel occupancy* series to find a repeating frame.

## Series navigation

**Part 5 of 10** · ←
[Part 4: The I/O Graph]({{ '/blog/tutorials/rf-scope-04-io-graph-channel-activity/' | relative_url }})
· Next →
[Part 6: Topology — Emitters & Conversations]({{ '/blog/tutorials/rf-scope-06-topology-emitters-conversations/' | relative_url }})
