---
title: "RF Scope, Part 2: From IQ to Bursts — The Segmentation Pre-Pass"
description: rfscope's segmentation pre-pass turns a wideband IQ span into bursts — discovering carriers with a wideband FFT, estimating a robust noise floor, downconverting each carrier to a narrow baseband, and slicing onset→offset bursts with per-burst spectral metrics and a blind modulation class.
category: tutorials
keywords: rfscope, segmentation, carrier discovery, wideband fft, noise floor, peak detection, digital downconversion, burst detection, occupied bandwidth, spectral flatness, dbfs, sdr
tags: [rfscope, sdr, dsp, rf-analysis, getting-started, gophertrunk]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 2
charts: true
---

*Part 2 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer. Before
any analyzer runs, the raw IQ has to become **bursts** — this post is the pre-pass
that makes them.*

> **TL;DR:** Segmentation is the producer step every analyzer consumes. It runs a
> wideband FFT (`-fft`, default 4096) to find carriers, sets a robust noise floor at
> the 25th percentile of the averaged bins, picks peaks that clear `-peak-threshold-db`
> (default 10 dB) spaced at least `-min-spacing` apart (default 12500 Hz), caps at 64
> channels, downconverts each carrier to ~50 kHz (`-channel-rate`), and slices
> onset→offset **bursts** with a hysteresis + debounce state machine. Each burst gets
> spectral metrics and a blind modulation class.

**Key takeaways**

- **Bursts are the atom.** No hierarchy, timeline, emitter, or anomaly exists until
  segmentation produces bursts.
- **Carrier discovery is a wideband FFT plus robust peak detection** — a 25th-percentile
  noise floor keeps a few loud carriers from hiding the quiet ones.
- **Each carrier is downconverted and decimated to ~50 kHz** before burst detection,
  so the work is cheap and rate-invariant.
- **Onset/offset uses hysteresis + debounce** anchored on a quiet floor, so a
  transmission already keyed up at the start of the capture is still found.

## Cheat sheet

| Flag | Default | What it tunes |
|---|---|---|
| `-fft` | 4096 | Wideband FFT size for carrier discovery (power of two) |
| `-peak-threshold-db` | 10 | Minimum dB above the noise floor for a carrier to count |
| `-min-spacing` | 12500 | Minimum Hz between carriers / channel raster |
| `-channel-rate` | 50000 | Per-channel decimated baseband rate (Hz) |
| `-sample-rate` | 2000000 (analyze) | Wideband IQ sample rate |
| `-window <sec>` | whole capture | Analyze only the first N seconds |

## In this post

- **The three stages** of segmentation: discover, downconvert, slice.
- **Carrier discovery** — the wideband FFT and the robust noise floor.
- **Peak detection** — threshold, spacing, and the 64-channel cap.
- **Per-carrier downconversion** to a narrow baseband, and why it makes RF Scope
  rate-invariant.
- **Burst onset/offset** — the hysteresis + debounce state machine.
- **Per-burst metrics** — occupied bandwidth, channel power, ACPR, spectral flatness,
  and the blind class.

## Three stages

Segmentation runs in `internal/rfscope/segment.go` and produces the Scene's header
plus its `Bursts` slice. It has three stages:

1. **Discover carriers** across the whole wideband span with an FFT.
2. **Downconvert** each discovered carrier to a narrow baseband and decimate it.
3. **Slice** each baseband into onset→offset bursts, measuring and classifying each.

Everything downstream — the hierarchy, the I/O graph, timing, topology, entropy,
expert info — reads bursts. Get segmentation wrong and every later view is wrong, so
it is worth understanding the knobs.

## Carrier discovery: FFT and a robust floor

The span is read into memory, then averaged into a power spectrum:

```go
// internal/rfscope/segment.go
avg := spectrum.AverageDB(raw, center, rate, cfg.FFTSize)
noiseFloor := spectrum.Percentile(avg.Bins, 0.25)
```

Two things are happening. First, `AverageDB` splits the capture into `-fft`-sized
windows (default **4096**, and the code forces it to a power of two — anything else
falls back to 4096) and averages their magnitude spectra. Averaging trades frequency
resolution against variance: more windows means a smoother spectrum, which makes weak
carriers stand out from the noise instead of drowning in a single noisy FFT frame.

Second, the **noise floor is the 25th percentile of the averaged bins**, not the mean
or the minimum. This is the robust-statistics trick. If a band has three loud
carriers and a lot of quiet spectrum, the mean floor is dragged up by the carriers
and you miss the weak ones; the minimum is dragged down by one dead bin and you see
phantom carriers everywhere. The lower quartile sits in the genuinely-quiet part of
the spectrum regardless of how many strong signals share the band, so the floor
tracks the *actual* noise.

> **dBFS, briefly.** Powers here are in **dBFS** — decibels relative to full scale,
> where 0 dBFS is the loudest an ADC sample can represent and everything real is
> negative. It is a receiver-relative scale, not an absolute one, which is exactly
> right for "how far above the local noise is this carrier?"

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="heatmap" width="560" height="300" role="img"
        aria-label="Wideband carrier-discovery spectrum with three carriers above the noise floor"></canvas>
<script type="application/json" class="lab-chart-data">
{ "matrix":[
[0.10,0.12,0.11,0.13,0.85,0.90,0.84,0.14,0.12,0.11,0.10,0.13,0.55,0.62,0.58,0.12,0.11,0.10,0.12,0.11],
[0.11,0.10,0.12,0.14,0.88,0.92,0.86,0.13,0.11,0.12,0.11,0.12,0.60,0.66,0.61,0.13,0.10,0.11,0.13,0.10],
[0.10,0.13,0.11,0.12,0.83,0.89,0.82,0.14,0.12,0.10,0.12,0.11,0.12,0.14,0.13,0.75,0.80,0.74,0.12,0.11],
[0.12,0.11,0.10,0.13,0.86,0.91,0.85,0.12,0.11,0.13,0.11,0.10,0.13,0.12,0.11,0.78,0.83,0.77,0.11,0.12]],
"xlabel":"frequency bin","ylabel":"averaged frames →" }
</script>
<figcaption>Three carriers rise above a quiet floor; averaging frames flattens the noise so peak detection can find the weak one on the right.</figcaption>
</figure>

## Peak detection: threshold, spacing, and the cap

With a floor in hand, `carriers.DetectPeaks` walks the averaged spectrum:

```go
// internal/rfscope/segment.go
peaks := carriers.DetectPeaks(avg, carriers.PeakOptions{
    ThresholdDb:  cfg.PeakThresholdDb, // default 10 dB above the floor
    MinSpacingHz: cfg.MinSpacingHz,    // default 12500 Hz
})
```

A bin is a carrier only if it clears the floor by `-peak-threshold-db` — **10 dB** by
default. Ten decibels above a robust floor is a deliberately conservative bar: it
keeps noise humps and spectral leakage from registering as channels. Turn it down for
weak-signal work, up to reject marginal carriers.

`-min-spacing` (**12500 Hz**) is the minimum distance between two accepted peaks —
the channel raster. It stops one strong carrier's shoulders from being counted as
several adjacent channels, and it encodes an assumption that neighboring channels are
at least a narrowband raster apart. For a band on a 25 kHz plan you might raise it; for
tightly-packed 6.25 kHz channels, lower it.

Two more details matter. `DetectPeaks` drops the DC bin (a front-end artifact on live
SDRs), but a real carrier can sit *exactly* at the tuned center — common for a
single-channel capture — so segmentation always adds the band center as a candidate
and lets the burst detector decide whether anything is actually there. And the
candidate list is capped at **MaxChannels = 64** (strongest first), so a pathological
band cannot explode into thousands of channels.

## Downconvert and decimate to ~50 kHz

For every confirmed carrier, segmentation shifts it to baseband and decimates:

```go
// internal/rfscope/segment.go
offset := float64(pk.FreqHz) - float64(center)
ddc := ccdecoder.NewDownconverterWithOffset(rate, cfg.ChannelTargetRateHz, offset)
base := ddc.Process(nil, raw)
```

This is **digital downconversion (DDC)**: multiply the wideband IQ by a complex tone
at `-offset` Hz to slide the carrier of interest down to 0 Hz, low-pass filter, then
**decimate** — throw away samples — down to the target `-channel-rate` of **50 kHz**.
Fifty kilohertz comfortably holds any land-mobile, IoT, or paging narrowband channel
while keeping the per-burst buffers small and the later FFTs cheap.

This step is why RF Scope is **rate-invariant**. Whether you captured at 2.4 MS/s or
10 MS/s, each carrier arrives at burst detection as the same ~50 kHz baseband, so a
signal's measured shape does not depend on your capture rate. (RF Scope uses the
single-tap `ccdecoder` downconverter here — the same one `replay -tune-hz` uses — not
the wideband multi-tap channelizer in `internal/dsp/tuner`. We return to that
distinction in Part 10.)

## Burst onset and offset

A decimated carrier is not yet a burst — it is a channel that is sometimes active and
sometimes quiet. `detectBursts` finds the active runs. It computes a per-frame power
envelope, then runs a **hysteresis + debounce** state machine lifted from the DMR
onset detector:

- **Hysteresis** means two thresholds, not one. The channel turns *on* when the
  envelope rises `OnThreshDb` (10 dB) above a quiet floor, and turns *off* only when
  it falls back below `OffThreshDb` (6 dB). The gap between on and off keeps a signal
  hovering near the threshold from chattering on and off every frame.
- **Debounce** requires the condition to hold for several consecutive frames
  (`Debounce = 3`) before the state flips, rejecting single-frame spikes.

The floor is a low percentile (10th) of the channel's *own* envelope over the whole
capture, not an EWMA seeded from the first frame. That subtlety matters: a
transmission that is already keyed up when the capture starts has no "off" period to
seed a running estimate from, and an EWMA-seeded detector would miss it entirely.
Anchoring on a fixed quiet floor finds it. Bursts shorter than `MinBurstSec` (5 ms)
are dropped as debounce noise.

When a channel has too little dynamic range to segment — a continuous carrier that is
on for the whole window, or dead air — the wideband carrier SNR decides between one
full-span burst and none.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="timeline" width="560" height="220" role="img"
        aria-label="Two channels sliced into onset-offset bursts over time"></canvas>
<script type="application/json" class="lab-chart-data">
{ "channels":[
{"label":"453.100","occ":[0,0,1,1,1,0,0,0,1,1,1,1,0,0,0,0,1,1,0,0]},
{"label":"453.325","occ":[0,1,1,0,0,0,0,1,1,1,0,0,0,1,1,1,1,0,0,0]}],
"xlabel":"time → (onset/offset bursts per channel)" }
</script>
<figcaption>The onset state machine turns each channel's power envelope into discrete bursts — the atoms the analyzers consume.</figcaption>
</figure>

## Per-burst metrics and the blind class

For each burst span, segmentation measures and classifies the slice:

```go
// internal/rfscope/segment.go
cls := survey.Classify(slice, outRate)
occ := spectrum.ComputeOccupancy(
    spectrum.AverageDB(slice, 0, outRate, occFFTFor(len(slice))).Bins,
    outRate, ...)
```

Two calls do the work. `spectrum.ComputeOccupancy` (over a 1024-point occupancy FFT,
or smaller for short bursts) yields the spectral metrics:

- **Occupied bandwidth** — how wide the burst actually is, in Hz. (RF Scope prefers
  `survey`'s above-noise-floor outward walk here, because the 99%-power span
  under-measures FM-family carriers whose strong DC component concentrates the power.)
- **Channel power (dBFS)** — the burst's in-channel power.
- **ACPR** — adjacent-channel power ratio, upper and lower: how much the burst spills
  into its neighbors.
- **Spectral flatness** — the **Wiener entropy**, a 0..1 number. It is the ratio of
  the geometric mean of the power spectrum to its arithmetic mean. Near **1** the
  spectrum is flat and noise-like or spread; well below 1 it is peaky — a tone or a
  narrow carrier. Flatness is one of RF Scope's highest-value protocol-agnostic
  discriminators, and it drives both the emitter fingerprint and a `noise-like`
  anomaly later.

`survey.Classify` supplies the blind **modulation class** — NBFM, C4FM, FSK, PSK,
continuous data, paging, and so on — plus features like the estimated baud rate. No
protocol is named here; that is the hierarchy analyzer's job in Part 3. Ada's first
run over her 453 MHz capture produces a handful of bursts, each a row with a
frequency, a duration, a class, an occupied bandwidth, and a flatness — the raw
material for everything that follows.

## Where this goes next

With bursts in hand, the analyzers can run. [Part
3]({{ '/blog/tutorials/rf-scope-03-protocol-hierarchy/' | relative_url }}) builds the
first view on top of them — the **protocol hierarchy**, which groups bursts by
modulation class, then by occupied-bandwidth bucket, then names the protocol of the
longest digital burst in each bucket. That is where Mercury first shows its face, as
an unknown-protocol node the tree cannot name.

## FAQ

**Why 4096 for the FFT, and why must it be a power of two?**
Power-of-two sizes let the FFT use the fast radix-2 path; 4096 balances frequency
resolution against averaging variance for typical 2–10 MS/s captures. Non-power-of-two
values are rejected and fall back to 4096.

**Why decimate every carrier to 50 kHz?**
Narrowband LMR/IoT/paging channels fit comfortably in 50 kHz, and decimating there
keeps per-burst buffers and FFTs small and makes the analysis rate-invariant — the
same signal looks the same whether you captured at 2.4 or 10 MS/s.

**My weak carrier isn't detected — what do I change?**
Lower `-peak-threshold-db` (try 6–8) so carriers closer to the floor register, and
check `-min-spacing` isn't merging it into a louder neighbor. Part 10 covers
segmentation tuning for unusual bands in depth.

**What counts as one burst?**
One onset→offset run on one channel, at least `MinBurstSec` (5 ms) long, detected by a
hysteresis + debounce state machine anchored on the channel's quiet floor.

## Series navigation

**Part 2 of 10** · ←
[Part 1: Wireshark for the RF physical layer]({{ '/blog/tutorials/rf-scope-01-wireshark-for-the-airwaves/' | relative_url }})
· Next →
[Part 3: Protocol Hierarchy]({{ '/blog/tutorials/rf-scope-03-protocol-hierarchy/' | relative_url }})
