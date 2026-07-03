---
title: "RF Scope, Part 4: The I/O Graph — Per-Channel Activity Over Time"
description: rfscope's timeline analyzer is the RF analog of Wireshark's I/O Graphs — it groups bursts by frequency into channels and fills each with a 100-bin occupancy and power series plus duty cycle, occupancy percentage, burst rate, dominant class, and the sparkline the cockpit renders.
category: tutorials
keywords: rfscope, io graph, wireshark io graphs, channel activity, duty cycle, occupancy, burst rate, timeline, sparkline, rf analysis, gophertrunk, dominant class
tags: [rfscope, rf-analysis, getting-started, gophertrunk, dsp, sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 4
charts: true
---

*Part 4 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. The
hierarchy told you what kinds of signal are present; this view tells you when each
channel is busy.*

> **TL;DR:** The `timeline` analyzer is Wireshark's I/O Graphs for RF. It groups
> bursts by frequency into **Channels** and fills each with a **100-bin** occupancy
> and power series over the observation window, plus duty cycle, occupancy percentage,
> burst rate per minute, median power, median flatness, and a dominant modulation
> class. It also materializes `Scene.Channels`, which the timing analyzer (Part 5)
> then annotates. The cockpit renders each channel's timeline as a `█`/`·` sparkline.

**Key takeaways**

- **One Channel per frequency**, each with a 100-bin activity timeline across the
  whole window.
- **Duty cycle vs occupancy %** answer different questions — one is airtime fraction,
  the other is how many time buckets saw any activity.
- **Burst rate per minute** normalizes bursty channels so short and long captures are
  comparable.
- **`timeline` is foundational.** It creates `Scene.Channels`, so the timing analyzer
  depends on it.

## Cheat sheet

| Field | Meaning |
|---|---|
| `DutyCycle` | Airtime ÷ window (fraction of time on air) |
| `OccupancyPct` | Share of the 100 time bins that saw any activity |
| `BurstRatePerMin` | Bursts scaled to per-minute |
| `DominantClass` | Most common modulation class on the channel |
| `MedianPowerDbFS` | Median burst peak power |
| `MedianFlatness` | Median spectral flatness (0..1) |
| `Timeline []Bin` | 100 buckets of occupancy + power |

## In this post

- **What the RF I/O graph is** and how it maps to Wireshark's.
- **How channels are built** from bursts, and the 100-bin timeline.
- **Duty cycle vs occupancy** — why they differ and when each matters.
- **The per-channel scalars** — burst rate, dominant class, median power/flatness.
- **The cockpit sparkline** — `█`/`·` at a glance.

## The RF I/O graph

Wireshark's I/O Graphs plot packet activity over time, so you can *see* the shape of
a capture: a steady stream here, a burst of retransmissions there, a quiet stretch in
the middle. The `timeline` analyzer does the same for RF, one row per channel. It
answers the question the hierarchy cannot: not *what* is on the band, but *when* each
channel is busy, and how heavily.

It has **no dependencies** and reads bursts directly. Importantly, it is also the
analyzer that **materializes `Scene.Channels`** — the per-frequency rows every later
view builds on. That is why the timing analyzer in Part 5 lists `timeline` as a
dependency: without channels there is nothing to annotate.

## Building channels from bursts

The analyzer groups bursts by exact frequency, then folds each group into a `Channel`:

```go
// internal/rfscope/timeline.go
byFreq := map[uint32][]int{}
for i, b := range sc.Bursts {
    byFreq[b.FreqHz] = append(byFreq[b.FreqHz], i)
}
```

For each frequency it accumulates airtime, tallies modulation classes to find the
**dominant class**, takes the widest occupied bandwidth, and computes the **median**
peak power and median spectral flatness across the channel's bursts. Median, not mean,
so one anomalously loud or noisy burst does not skew the channel's summary.

Then it builds the timeline itself: **100 equal time buckets** (`timelineBins = 100`)
spanning the observation window. For each bucket it walks the channel's bursts and
marks the bucket occupied if any burst overlaps it, records the strongest overlapping
burst's power (falling back to the scene noise floor when idle), and counts how many
bursts *started* in that bucket:

```go
// internal/rfscope/timeline.go
for _, i := range idxs {
    b := sc.Bursts[i]
    if b.StartSec < t1 && b.EndSec > t0 { // overlap
        bin.Occupied = true
        if b.PeakDbFS > bin.PowerDbFS {
            bin.PowerDbFS = b.PeakDbFS
        }
    }
    if b.StartSec >= t0 && b.StartSec < t1 { // started in this bin
        bin.BurstCount++
    }
}
```

A hundred buckets is a deliberate resolution choice: fine enough to show structure —
a duty pattern, a periodic gap — coarse enough to stay readable as a sparkline and
cheap to autocorrelate for period detection in Part 5.

Two details in that loop repay attention. The overlap test (`b.StartSec < t1 &&
b.EndSec > t0`) marks a bucket occupied if a burst covers *any* part of it, so a burst
that straddles a bucket boundary correctly lights both buckets rather than being lost to
rounding. And the separate "started in this bin" count exists because *occupancy* and
*arrivals* are different questions: a single long burst occupies many buckets but arrives
in only one, while a burst of rapid short transmissions arrives many times. Keeping both
lets the later analyzers ask either question — occupancy drives the sparkline and duty,
arrivals drive the burst-rate and the inter-arrival histogram.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="timeline" width="560" height="260" role="img"
        aria-label="Per-channel occupancy timelines for four channels over the capture window"></canvas>
<script type="application/json" class="lab-chart-data">
{ "channels":[
{"label":"453.100 ctl","occ":[1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1]},
{"label":"453.325 voice","occ":[0,0,1,1,1,0,0,0,0,1,1,1,1,0,0,0,0,1,1,0]},
{"label":"453.550 hop","occ":[1,0,0,0,1,0,0,0,1,0,0,0,1,0,0,0,1,0,0,0]},
{"label":"453.775 data","occ":[0,0,0,1,0,0,0,0,0,0,1,0,0,0,0,0,0,1,0,0]}],
"xlabel":"time → (100 bins per channel, downsampled here)" }
</script>
<figcaption>Four channels with very different shapes: a near-continuous control channel, a duty-cycled voice channel, a periodic hopper, and a sparse data channel.</figcaption>
</figure>

## Duty cycle vs occupancy percentage

Two of the channel scalars look similar and are constantly confused, so it is worth
being precise:

- **Duty cycle** is `AirtimeSec ÷ window`, clamped to 0..1 — the true fraction of the
  window the channel spent transmitting. A channel on for 3 of every 10 seconds has a
  0.30 duty cycle.
- **Occupancy percentage** is the fraction of the **100 time bins** that saw *any*
  activity. A channel that fires a 10 ms burst in each of 30 bins has a low duty cycle
  (little total airtime) but 30% occupancy (lots of bins touched).

The gap between them is diagnostic. **High duty, high occupancy** is a continuously
busy channel — a control channel, a voice call in progress. **Low duty, high
occupancy** is a *bursty* channel: frequent short transmissions, like telemetry,
paging, or a data link. **Low duty, low occupancy** is genuinely quiet. That "low
duty, high occupancy" signature is exactly what the expert analyzer will later flag as
**intermittent** (Part 8).

## The per-channel scalars

Alongside the two occupancy measures, each channel carries:

- **Burst rate per minute** — `BurstCount ÷ (window / 60)`. Normalizing to per-minute
  lets you compare a 4-second cockpit refresh against a 5-minute capture without doing
  arithmetic in your head.
- **Dominant class** — the most common modulation class among the channel's bursts,
  ties broken deterministically by name. A channel whose bursts classify inconsistently
  (a hopper landing here between other channels) still gets a single representative
  class.
- **Median power (dBFS)** and **median flatness** — robust summaries that feed the
  emitter fingerprint (Part 6) and the `noise-like` anomaly rule (Part 8).

Together these turn a channel from a pile of bursts into a one-line character sketch:
*"453.325 MHz, c4fm, 18% duty, 22 bursts/min, −41 dBFS, flatness 0.34"* reads as a
moderately-busy narrowband digital voice channel at a glance.

## A worked reading of the channel table

The summary report lays the channels out as a table, and learning to read it top to
bottom is most of the skill:

```text
Channels:
  freq(MHz)     class    bursts   duty%    occ%  rate/min   period
  453.100000    c4fm          1    98.4    99.0       0.4   0.030s
  453.325000    c4fm         14    18.2    31.0       8.4   0.030s
  453.550000    fsk           9     6.1    22.0       5.4        -
  453.775000    fsk           2     1.1     2.0       1.2        -
```

Four channels, four characters. The first is a **control channel**: one long burst,
98% duty, and a 30 ms period — it is on almost continuously with a TDMA frame, exactly
what a P25 control channel looks like. The second shares the 30 ms frame but at 18%
duty across fourteen bursts — a **voice channel** carrying intermittent calls on the
same system. The third is the **hopper**: nine short FSK bursts, 6% duty, and — tellingly
— *no recovered period*, because its schedule only makes sense across channels, not down
one (Part 6 explains why). The fourth is barely there: two stray bursts, a candidate to
ignore or to watch, depending on what you are hunting.

Notice how the columns disagree in useful ways. Channel two's 18% duty against 31%
occupancy is the bursty-but-substantial signature; channel three's 6% duty against 22%
occupancy is more extreme still — lots of buckets touched, very little airtime. That
widening gap between duty and occupancy is a reliable "this channel is bursty" tell, and
it is precisely the input the expert analyzer thresholds into an `intermittent` flag.

## Median, not mean, everywhere

One quiet design choice runs through the whole analyzer: every per-channel summary uses
the **median**, not the mean. Median power, median flatness — both are computed by
sorting the channel's bursts and taking the middle value. The reason is robustness. A
single burst caught mid-fade, or a momentary co-channel collision that spikes the power,
would drag a mean off the channel's true character. The median shrugs it off: half the
bursts are above, half below, and one outlier cannot move the middle. It is the same
robust-statistics instinct segmentation used for the 25th-percentile noise floor in Part
2 — prefer an estimator that a handful of bad samples cannot capture. When you read a
channel's median power as −41 dBFS, that is genuinely where most of its bursts sit, not
an average pulled around by the loudest one.

## The cockpit sparkline

The live cockpit (Part 9) renders each channel's timeline as a fixed-width Unicode
**sparkline**: a filled block for an occupied bucket, a dot for idle.

```go
// cmd/gophertrunk/rfscope_cockpit.go
if occupied {
    out[i] = '█'
} else {
    out[i] = '·'
}
```

So a channel row in the cockpit looks like:

```text
   453.3250 c4fm   ··██·····██████···██·  18%
```

The sparkline compresses the 100-bin timeline to whatever width the panel has, marking
a block if *any* underlying bucket in that column was occupied. It is the fastest way
to read a band: your eye catches the near-solid control channel, the gappy hopper, and
the mostly-empty data channel without parsing a single number. Ada learned to scan the
sparkline column first and only read the duty-cycle numbers for the channels whose
shape looked interesting.

## Where this goes next

The timeline gives every channel a shape in time — and a shape in time is something
you can measure *periodicity* in. [Part
5]({{ '/blog/tutorials/rf-scope-05-timing-and-tdma-period/' | relative_url }})
introduces the `timing` analyzer, which depends on this one: it builds burst-length
and inter-arrival histograms and, by **autocorrelating** each channel's occupancy
series, recovers a TDMA or frame **period** hiding in the gaps.

## FAQ

**Why exactly 100 timeline bins?**
It is a balance: enough resolution to reveal duty patterns and periodic gaps, coarse
enough to render as a readable sparkline and cheap to autocorrelate for period
detection in Part 5.

**A channel shows high occupancy but a tiny duty cycle — is that a bug?**
No — that is a bursty channel. Occupancy counts *how many buckets* saw activity; duty
cycle measures *total airtime*. Frequent short bursts touch many buckets while adding
up to little airtime. The expert analyzer flags that pattern as `intermittent`.

**Does the timeline analyzer identify protocols?**
No. It only aggregates burst activity per frequency. Naming protocols is the
hierarchy analyzer's job (Part 3); the dominant *class* here is a blind modulation
label, not a protocol.

**Why does timing depend on timeline?**
Because timeline is what creates `Scene.Channels` and their 100-bin timelines. The
timing analyzer annotates those channels; it has nothing to work on until timeline
runs first.

## Series navigation

**Part 4 of 10** · ←
[Part 3: Protocol Hierarchy]({{ '/blog/tutorials/rf-scope-03-protocol-hierarchy/' | relative_url }})
· Next →
[Part 5: Timing & Periodicity]({{ '/blog/tutorials/rf-scope-05-timing-and-tdma-period/' | relative_url }})
