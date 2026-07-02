---
title: "Signal Lab, Part 5: PSD, Spectrogram & Spectral Occupancy"
description: "Signal Lab's frequency-domain views — the power spectral density, the spectrogram/waterfall, and the spectral-occupancy metrics computed off the PSD: 99% occupied bandwidth, channel power, adjacent-channel power ratio (ACPR), and spectral flatness."
category: tutorials
keywords: power spectral density, psd, spectrogram, waterfall, occupied bandwidth, channel power, acpr, adjacent channel power ratio, spectral flatness, spectral occupancy, signal lab
tags: [siglab, dsp, spectrum, occupancy, rf, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 5
charts: true
---

*Part 5 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. We've looked at symbols; now we look at spectrum.*

> **TL;DR:** Signal Lab's frequency-domain views describe a capture's *shape*
> rather than its symbols. The **PSD** shows power versus frequency; the
> **spectrogram/waterfall** shows how that PSD evolves over time; and a set of
> **spectral-occupancy metrics** computed off the PSD — **99% occupied
> bandwidth**, **channel power**, **adjacent-channel power ratio (ACPR)**, and
> **spectral flatness** — turn the picture into numbers you can compare and
> regress against.

**Key takeaways**

- **PSD = power vs frequency.** It's where you confirm a signal is where and how
  wide you think it is.
- **Spectrogram = PSD over time.** Intermittent and hopping emitters reveal
  themselves here that a single PSD averages away.
- **99% occupied bandwidth** answers "how wide is this signal, really?"
- **ACPR** measures spillover into adjacent channels; **channel power** is the
  in-band total; **spectral flatness** distinguishes tone-like from noise-like.
- **These are computed off the PSD**, so they're deterministic and repeatable for
  a given capture.

## Cheat sheet

| Metric / view | What it tells you |
|---|---|
| PSD (line) | Power spectral density — power vs frequency |
| Spectrogram / waterfall (heatmap) | PSD evolving over time |
| 99% occupied bandwidth | Bandwidth containing 99% of the power |
| Channel power | Total power in the channel of interest |
| ACPR | Adjacent-channel power ratio — spillover into neighbors |
| Spectral flatness | Tone-like (low) vs noise-like (high) |

## In this post

- **The PSD** — reading power versus frequency.
- **The spectrogram** — the time dimension the PSD hides.
- **Occupied bandwidth** — the honest width of a signal.
- **Channel power and ACPR** — in-band energy and spillover.
- **Spectral flatness** — the tone-vs-noise number.

## The PSD: power versus frequency

The **power spectral density** is the workbench's most basic frequency-domain
view: estimated power on the vertical axis, frequency on the horizontal. It's
where you answer the first questions about any capture — *is the signal actually
centered where I think? how wide is it? is there an interferer sitting next to
it?* For a well-behaved narrowband system like a 12.5 kHz P25 channel, the PSD
shows a compact bump over the channel with the noise floor on either side; a
mistuned capture shows that bump offset from center (which is exactly what
`-auto-tune` corrects before demod).

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="line" width="560" height="300" role="img"
        aria-label="Power spectral density of a narrowband channel over its noise floor"></canvas>
<script type="application/json" class="lab-chart-data">
{ "series":[{"label":"PSD","points":[[-12,-96],[-9,-95],[-7,-92],[-6,-78],[-5,-58],[-4,-46],[-3,-42],[-2,-40],[-1,-39],[0,-38],[1,-39],[2,-40],[3,-42],[4,-46],[5,-58],[6,-78],[7,-92],[9,-95],[12,-96]]}],
"xlabel":"frequency offset (kHz)","ylabel":"power (dBFS)","xmin":-12,"xmax":12,"ymin":-100,"ymax":-30 }
</script>
<figcaption>A narrowband channel in the PSD: a compact power bump centered in the channel, sitting well above the noise floor on either side. Occupancy metrics are read straight off this curve.</figcaption>
</figure>

Everything else in this post is derived from this curve, which is why the PSD is
worth reading carefully first. A clean, centered, well-separated bump is the
precondition for a trustworthy occupancy measurement.

## The spectrogram: the time dimension

A single PSD averages a capture's whole duration into one curve — which throws
away *time*. The **spectrogram** (or **waterfall**) keeps it: frequency on one
axis, time on the other, power as color. A steady carrier is a solid vertical
stripe; a bursty or intermittent emitter is a stack of short dashes; a
frequency-hopper is a scatter of energy skipping across bins.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="heatmap" width="560" height="320" role="img"
        aria-label="Spectrogram waterfall showing an intermittent bursty carrier over time"></canvas>
<script type="application/json" class="lab-chart-data">
{ "xlabel":"frequency bin","ylabel":"time →",
"matrix":[
[0.05,0.06,0.1,0.7,0.9,0.7,0.1,0.06,0.05],
[0.05,0.05,0.08,0.6,0.85,0.6,0.09,0.05,0.05],
[0.04,0.05,0.06,0.08,0.09,0.08,0.06,0.05,0.04],
[0.05,0.06,0.1,0.72,0.92,0.72,0.1,0.06,0.05],
[0.05,0.05,0.09,0.65,0.88,0.65,0.09,0.05,0.05],
[0.04,0.05,0.06,0.07,0.08,0.07,0.06,0.05,0.04],
[0.04,0.05,0.06,0.07,0.08,0.07,0.06,0.05,0.04],
[0.05,0.06,0.11,0.7,0.9,0.7,0.11,0.06,0.05]] }
</script>
<figcaption>A waterfall of an intermittent channel: energy appears in the center bins, drops out, and returns. This time structure is exactly what a single averaged PSD hides — and it's how Mercury's on/off bursts first stood out to Ada.</figcaption>
</figure>

This is the view where **Mercury** first looked strange. Its bursts near 453 MHz
come and go, so on a single PSD it barely registers, but on the waterfall it's an
unmistakable stutter of short dashes — present, gone, present again. The
spectrogram is where "intermittent" stops being a hunch and becomes visible; we
pick that thread back up in Part 8.

## Occupied bandwidth: the honest width

The first occupancy metric is **99% occupied bandwidth**: the width of the
frequency band that contains 99% of the signal's power. It's the honest answer to
"how wide is this signal?" — more honest than eyeballing the PSD, because it
integrates power rather than trusting where the bump *looks* like it ends.
Occupied bandwidth is how you confirm a capture matches its channel plan: a P25
system on a 12.5 kHz raster should show an occupied bandwidth comfortably inside
that channel. A measurement that's too wide means splatter, an interferer folded
in, or the wrong assumption about what you're looking at.

## Channel power and ACPR

Two related numbers describe energy in and around the channel:

- **Channel power** is the total power integrated across the channel of interest —
  the in-band energy. It's a level reference for everything else and a quick way
  to compare two captures of the same system recorded at different gains.
- **Adjacent-channel power ratio (ACPR)** measures how much of a transmitter's
  power spills into the *neighboring* channels relative to its own. High ACPR
  means a signal is bleeding into its neighbors — a transmitter problem, a
  nonlinearity, or (from the receive side) a strong nearby emitter you're picking
  up. For anyone sharing a band, ACPR is the politeness metric: a well-behaved
  emitter keeps its energy in its own channel.

Read together, channel power and ACPR tell you both *how much* signal you have and
*how cleanly* it's confined. A strong channel with poor ACPR is loud and messy; a
modest channel with excellent ACPR is quiet and well-mannered.

## Spectral flatness: tone-like vs noise-like

The last metric, **spectral flatness**, is a single number that captures the
*character* of a spectrum: it's near zero for a peaky, tone-like signal (energy
concentrated at a few frequencies) and near one for a flat, noise-like spectrum
(energy spread evenly). It's a compact way to distinguish "there's a real
modulated carrier here" from "this is basically noise." A capture whose channel
region reads highly flat is telling you there may be nothing to decode — which,
before you spend time chasing a lock, is worth knowing.

Flatness also helps with the unknown-signal problem. A carrier with structure —
a clear modulated shape — reads lower flatness than the surrounding noise floor,
which is one of the cues blind signal-ID leans on when it tries to *name* an
undecodable carrier in Part 8. Reese's rule: **flatness tells you if there's a
signal; occupied bandwidth tells you how wide it is; ACPR tells you if it's
staying in its lane.** Three numbers, three different questions, all read off the
same PSD.

## Resolution, windowing, and what you can trust

A PSD is an *estimate*, and like any estimate it has a resolution-versus-noise
trade-off worth understanding before you over-read it. Splitting a capture into
more, longer FFT segments buys finer frequency resolution — narrower bins, so two
close carriers separate instead of blurring into one bump — at the cost of a
coarser time axis on the spectrogram. Splitting it into more, shorter segments
does the opposite: it smooths the PSD and sharpens the waterfall's time
resolution but widens each frequency bin. There's no free lunch; the estimator
picks a sensible middle, but the consequence is real. Two carriers 3 kHz apart
may or may not resolve depending on the bin width, and an occupied-bandwidth
measurement inherits the same uncertainty at its edges.

The practical upshot for occupancy metrics: they're *relative* instruments, best
used to compare captures or to track a change, and they're most trustworthy when
the signal is well above the noise floor. A channel-power number on a capture
that's barely above the floor is dominated by the noise you're integrating along
with the signal; the same measurement on a strong, clean capture is tight. When
Reese compares two recordings of the same system, he compares them at the *same*
settings against the *same* reference — because it's the delta between them, not
the absolute dBFS, that carries the meaning.

## Occupancy as a fingerprint

Read together, the occupancy set doubles as a coarse *fingerprint* for an unknown
carrier, which is where this part quietly feeds Part 8. A 12.5 kHz occupied
bandwidth with low flatness and modest ACPR is consistent with a narrowband
land-mobile channel; a much wider occupied bandwidth points at a different class
of system entirely. You can't name a protocol from occupancy alone — that takes
the blind symbol-rate and modulation estimate — but you *can* rule large families
in or out before you ever try to decode. For Mercury, the occupancy said
"narrowband, ~12.5 kHz, real modulated structure, not splattering into its
neighbors" — enough to know it was a deliberate signal worth chasing, long before
anything named it.

## Where this goes next

You can now describe a capture in both domains — symbols (Part 4) and spectrum
(this part). [Part 6]({{ '/blog/tutorials/signal-lab-06-synthesize-references/' | relative_url }})
flips the workbench around: instead of measuring a captured signal, you'll
*generate* one — an idealized reference — and then deliberately impair it (SNR,
carrier offset, multipath, DC, I/Q imbalance) to build clean baselines and
known-bad fixtures whose PSD and occupancy you already know the answer to. The
[SigLab docs]({{ '/siglab.html' | relative_url }}) list the full metric set.

## FAQ

**What's the difference between a PSD and a spectrogram?**
The PSD is one curve — power versus frequency, averaged over the capture. The
spectrogram adds time as a second axis and shows how that PSD evolves, which is
essential for intermittent or hopping signals a single PSD would smear away.

**Why 99% occupied bandwidth and not just "the width"?**
Because "the width" depends on where you decide the signal stops. Occupied
bandwidth defines it rigorously as the band containing 99% of the power, so two
people (or two runs) measuring the same capture get the same answer.

**What does high ACPR indicate?**
Power spilling into adjacent channels relative to the channel of interest —
either a transmitter that isn't confining its energy well or a strong neighboring
emitter bleeding in. Low ACPR is a clean, well-confined signal.

**Are these metrics repeatable?**
Yes. They're computed deterministically off the PSD of a fixed capture, so the
same file yields the same numbers every run — which is what makes them useful as
regression baselines.

## Series navigation

**Part 5 of 10** · ←[Part 4]({{ '/blog/tutorials/signal-lab-04-constellations-and-eye-diagrams/' | relative_url }}) · Next →
[Part 6: Synthesize References]({{ '/blog/tutorials/signal-lab-06-synthesize-references/' | relative_url }})
