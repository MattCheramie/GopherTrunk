---
title: "The Hunt, Part 2: The Wideband Sweep Engine"
description: How GopherTrunk steps a wide SDR receiver across megahertz of spectrum, estimates a power spectrum from short IQ grabs, and stitches overlapping tiles into one occupancy picture without dropping a carrier at a step boundary.
category: deep-dives
keywords: wideband sweep, sdr spectrum sweep, power spectral density, fft frame averaging, spectrum stitching, step overlap guard band, carrier discovery, tune step, gophertrunk the hunt
tags: [the-hunt, sdr, dsp, spectrum, sweep, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 2
---

*Part 2 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 1]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
mapped the whole journey — sweep, identify, map — and planted the thread we keep
chasing: a stray digital carrier somewhere in 851–869 MHz. This part builds the
first stage. Before we can classify or decode that carrier, we have to **find its
frequency**, and finding it means dragging a receiver whose window is a few
megahertz wide across a band eighteen megahertz wide, and not losing anything in
the seams.*

> **TL;DR:** A wideband sweep is a **tiling problem**. The SDR sees one window at
> a time (the sample rate, a few MHz); the band is much wider. GopherTrunk's
> `Sweeper` **tunes** across the band in **overlapping steps**, estimates a
> **power spectrum** at each step by windowing + FFT-ing short IQ grabs and
> **averaging** them, then **de-duplicates** the peaks a carrier produces when it
> shows up in two overlapping tiles. The overlap is the whole trick: advance by
> *less* than the window so no carrier ever lives only in the rolloff.

**Key takeaways**

- **The step is smaller than the window on purpose.** Advancing by the usable
  bandwidth (window minus a guard fraction) makes adjacent tiles overlap, so a
  carrier near one step's edge sits comfortably inside its neighbour.
- **A power spectrum is averaged, not snapshotted.** Each step accumulates
  several windowed FFT frames and averages their magnitude-squared, lifting a weak
  carrier out of the noise a single frame would bury.
- **The same carrier in two tiles becomes one candidate.** Peaks are bucketed to
  a quantized frequency; the strongest sighting wins, so overlap costs nothing
  downstream.
- **Wide signals get a second pass.** Alongside narrowband peak detection, an
  optional occupancy scan finds plateaus too wide to fit one tile and *stitches*
  the clipped spans across steps into one true bandwidth.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Sweep options | band list, FFT size, dwell, guard | `internal/hunt/sweeper.go` (`SweepOptions`) |
| The walk | tune → capture → detect, per step | `internal/hunt/sweeper.go` (`Sweeper.Sweep`) |
| PSD estimate | window + FFT + power-average | `internal/hunt/sweeper.go` (`captureFrame`) |
| Candidate | a found carrier (freq + SNR) | `internal/hunt/sweeper.go` (`Candidate`) |
| Wide-signal stitch | join clipped spans across tiles | `internal/hunt/wideband_sweep.go` (`stitchWideband`) |
| Grid snap | correct FFT-bin quantization | `internal/hunt/sweeper.go` (`snapCandidatesToGrid`) |

## In this post

- **The tiling problem** — why one tune can't see a whole band, and what overlap buys.
- **Estimating the spectrum** — how a short IQ grab becomes averaged power per bin.
- **The walk** — the tune → capture → detect loop, step by step.
- **De-duplicating carriers** — collapsing two sightings of one emitter into one.
- **Stitching wide signals** — the second pass that recovers a cellular block.

## The tiling problem

An SDR delivers a slice of spectrum as wide as its sample rate. Run an RTL-SDR at
2.4 MS/s and you see 2.4 MHz at a time. The 800 MHz trunking band we care about is
851–869 MHz — **18 MHz**. So a sweep is fundamentally a loop: tune the center,
grab a window, detect what's in it, advance, repeat.

The naive version of that loop advances the center by exactly one window. It has a
fatal flaw: **the edges of an SDR window are garbage.** Anti-alias filtering rolls
off toward Nyquist, so a carrier sitting near the edge of a tile is attenuated and
distorted — and if you advance by a full window, some carriers land *only* near an
edge and are never seen cleanly. The fix is to reserve a guard fraction at each
edge and advance by less:

```go
// internal/hunt/sweeper.go (shape) — Sweeper.Sweep step size
rate := s.opts.Source.SampleRateHz()
// Advance the center by the usable bandwidth (minus the guard) so adjacent
// steps overlap slightly and a carrier near a step edge is still seen
// somewhere away from the rolloff.
step := uint32(float64(rate) * (1 - 2*s.guardFrac))
```

With the default `GuardFrac` of `0.1`, the step is `rate * 0.8` — we throw away
the outer 10% at each edge and let the next tile cover it. A carrier that was in
the rolloff of step *N* is 20% of a window deep into the clean interior of step
*N+1*. Nothing hides in a seam because every seam is covered twice.

The first step is centered half a usable-bandwidth above the band's low edge, so
the low edge itself is inside the first tile rather than at its rolloff, and the
loop stops as soon as a step's coverage reaches the high edge:

```go
// internal/hunt/sweeper.go (shape) — Sweeper.Sweep band walk
halfUsable := step / 2
for center := band.LowHz + halfUsable; ; center += step {
    if err := s.opts.Source.Tune(center); err != nil { /* … */ }
    frame, err := s.captureFrame(ctx, center, rate, framesPerStep)
    // …detect peaks in-band, collect candidates…
    if center+halfUsable >= band.HighHz {
        break
    }
}
```

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="A wide band covered by four overlapping receiver tiles; each tile discards its guard-band edges and the clean interiors overlap so a carrier near one tile edge lands inside the next tile's interior">
  <line x1="20" y1="40" x2="660" y2="40" stroke="var(--fg-muted)"/>
  <text x="20" y="28" fill="var(--fg-muted)" font-size="9">851 MHz</text>
  <text x="618" y="28" fill="var(--fg-muted)" font-size="9">869 MHz</text>
  <rect x="24" y="52" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="124" y="71" text-anchor="middle" fill="currentColor" font-size="10">tile 1</text>
  <rect x="180" y="90" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="280" y="109" text-anchor="middle" fill="currentColor" font-size="10">tile 2</text>
  <rect x="336" y="52" width="200" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="436" y="71" text-anchor="middle" fill="var(--accent)" font-size="10">tile 3</text>
  <rect x="452" y="90" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="552" y="109" text-anchor="middle" fill="currentColor" font-size="10">tile 4</text>
  <line x1="360" y1="48" x2="360" y2="124" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="360" y="140" text-anchor="middle" fill="var(--accent)" font-size="9">our carrier — in tile 2's edge, tile 3's interior</text>
  <text x="340" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">advance by the usable bandwidth (window − 2×guard) so interiors overlap and no carrier lives only in a rolloff</text>
</svg>
<figcaption>The band is tiled by overlapping receiver windows. Because the step is smaller than the window, every frequency is in some tile's clean interior.</figcaption>
</figure>

## Estimating the spectrum

Inside one tile we need a power spectral density: how much energy sits at each
frequency. That is a windowed FFT, magnitude-squared. But one FFT of one grab is
noisy — a weak carrier can be indistinguishable from a lucky noise bin. So
`captureFrame` captures several frames per step and **power-averages** them, which
is what pulls a real carrier up out of the variance while the noise averages down:

```go
// internal/hunt/sweeper.go (shape) — captureFrame
acc := make([]float64, n) // accumulated linear power per (shifted) bin
for f := 0; f < frames; f++ {
    iq, err := s.opts.Source.Capture(ctx, captureFrameSamples(n))
    // …
    s.accumulatePower(acc, iq[:n]) // window, FFT, add |X|²
    got++
}
bins := make([]float32, n)
norm := 1.0 / (float64(n) * s.winSum)
for i := 0; i < n; i++ {
    power := acc[i] / float64(got) * norm
    bins[i] = float32(10 * math.Log10(power))
}
```

Two details earn their keep. The window is a **Hann** window (`window.Hann`), and
its squared sum `winSum` is folded into the normalisation so the reported dB
values are calibrated rather than window-dependent. And `accumulatePower`
**FFT-shifts** the output — DC lands in the middle bin — so a `spectrum.Frame`
reads left-to-right as low-to-high frequency, the convention every downstream
consumer (the peak detector, the occupancy scan) expects.

How many frames? The `SweepDwell` option controls it: roughly one frame per 10 ms
of dwell, capped between 1 and 64. A longer dwell buys sensitivity — more
averaging, a cleaner floor — at the cost of sweep time. A blind first pass runs
short; a follow-up on a promising region can dwell longer.

### How that principle shaped the Go code

- **FFT scratch is reused across steps.** The `Sweeper` owns its `fft.Plan`,
  window, and `bufIQ`/`bufOut` buffers, allocated once in `NewSweeper`. A sweep of
  a whole device is hundreds of steps; per-step allocation would thrash the GC.
- **Frames are consumed, not retained.** A step's frames are averaged into one
  `spectrum.Frame` and the raw frames dropped. Only the small summary survives —
  which matters enormously for the wideband stitch pass below, which needs
  every step but not every sample.
- **The estimate is rate-agnostic.** `captureFrame` takes the rate as an argument
  and computes bin spacing as `SampleRate/N`; nothing hard-codes 2.4 MS/s. Run the
  same sweeper on an Airspy at 10 MS/s and the tiles are just wider.

## The walk, and de-duplicating carriers

Each tile's averaged frame goes to `DetectPeaks` (Part 3's subject), which returns
carriers as `(frequency, SNR)`. But overlap means the same physical carrier
appears in two tiles — once in each interior. We do *not* want two candidates for
one emitter, so the sweep buckets every peak to a quantized frequency and keeps
the strongest sighting:

```go
// internal/hunt/sweeper.go (shape) — Sweeper.Sweep candidate dedup
bucket := s.opts.PeakOpts.MinSpacingHz // 0 ⇒ 6.25 kHz, the tightest channel step
best := map[uint32]Candidate{}
// …per in-band peak p:
key := p.FreqHz / bucket
if cur, ok := best[key]; !ok || p.SNRDb > cur.SNRDb {
    best[key] = Candidate{FreqHz: p.FreqHz, SNRDb: p.SNRDb}
}
```

The bucket width is the minimum channel spacing (6.25 kHz by default — the
tightest land-mobile step), so two sightings of one carrier collapse while two
genuinely adjacent channels stay distinct. Only peaks that fall inside the
requested band survive — a carrier from a neighbouring tile that happens to sit
outside `[band.LowHz, band.HighHz]` is discarded, because the band that *owns* it
will see it cleanly in its own interior. The result is a `Candidate` set sorted by
descending SNR: the strongest, most-likely-to-lock carriers first. Our 851–869 MHz
stray now has a row in that list — a frequency and an SNR, nothing more yet.

One last correction runs before the list is returned. FFT bins quantize frequency
to `SampleRate/N` steps, so a P25 trunk truly on 851.012500 MHz might report as
851.0123 MHz — a few hundred Hz off, enough that a decoder tuned there never
locks. `snapCandidatesToGrid` infers the channel raster the carriers actually sit
on and snaps each candidate to the nearest grid point *within one bin*, so genuine
carriers are corrected while an off-raster signal is left alone. That is Part 3's
grid logic (`carriers.InferGrid`) doing quiet, load-bearing work.

## Stitching wide signals across tiles

Narrowband peak detection is blind to signals *wider than a channel* — a cellular
or WiFi OFDM block is a flat plateau, not a spike, and it can be wider than a
whole tile. When `DetectWideband` is set, each step also runs an occupancy scan
and keeps a small summary, and after the sweep those summaries are stitched:

```go
// internal/hunt/wideband_sweep.go (shape) — per-step summary kept for stitching
type wbStep struct {
    floorDb float32     // this step's per-frame low-quartile noise floor
    lowHz   uint32      // low edge of the step's scanned (non-guard) coverage
    highHz  uint32      // high edge of the step's scanned coverage
    spans   []Occupancy // occupancy spans found with this step's per-frame floor
}
```

The subtlety is that a signal wider than the tune **fills a whole step**, so that
step's own noise-floor estimate sits *inside* the signal and the per-frame
occupancy scan finds nothing. `stitchWideband` solves this by first learning the
**sweep-wide** floor — the quietest step's floor — then treating any step whose
own floor sits `fullStepClipDb` (default 6 dB) above it as fully occupied, and
merging every span (real occupancy plus these synthesized full-step spans) that
overlaps or falls within a bin-width seam tolerance:

```go
// internal/hunt/wideband_sweep.go (shape) — stitchWideband merge
if s.lowHz <= cur.highHz+mergeTolHz {   // overlapping or within a seam ⇒ same emitter
    if s.highHz > cur.highHz { cur.highHz = s.highHz }
    if s.powerDb > cur.powerDb { cur.powerDb = s.powerDb }
    continue
}
```

Each merged interval becomes a `Candidate` with `IsWideband = true` and `BwHz` set
to its stitched width. Those never go to the decoder — a cellular block is *named
by allocation and shape*, not demodulated — but they belong in the inventory so an
operator sees the whole band, not just the trunked slivers. It is the same
degrade-gracefully instinct from Part 1: surface everything with power, decode only
what warrants it.

## Where this goes next

[Part 3]({{ '/blog/deep-dives/the-hunt-03-peak-occupancy-detection/' | relative_url }})
opens the box the sweep leaned on twice — `DetectPeaks` and `DetectOccupancy` in
`internal/carriers`. How do you estimate a noise floor that a strong carrier can't
poison? How do you pick peaks without reporting one carrier three times? And how
does the occupancy grid tell a narrow trunking channel from a wide OFDM plateau?
That's where our stray carrier stops being "energy at a frequency" and becomes a
ranked, grid-snapped candidate worth classifying.

## FAQ

**Why not just use a wider SDR and skip the tiling?**
Even a 10 MS/s Airspy sees only 10 MHz; the 700/800 MHz trunking ranges are wider,
and VHF/UHF surveys span far more. Tiling is unavoidable above some bandwidth, so
GopherTrunk does it well — overlapping steps, averaged frames, de-duplicated
peaks — rather than pretending a single tune suffices. The engine is the same
whether the band needs two tiles or two hundred.

**Why average multiple FFT frames instead of taking a longer FFT?**
A longer FFT narrows bins (better frequency resolution) but doesn't reduce the
variance of each bin's power estimate — one realization is still noisy. Averaging
several independent frames reduces that variance, which is what lifts a weak
carrier above the noise floor for detection. The two knobs are independent:
`FFTSize` sets resolution, `SweepDwell` sets averaging.

**What stops one carrier being reported as several candidates?**
Two things. Within a tile, `DetectPeaks` enforces a minimum spacing so a single
carrier's shoulders aren't counted as extra peaks. Across tiles, the sweep buckets
peaks to `MinSpacingHz` and keeps the strongest, so a carrier seen in two
overlapping interiors collapses to one `Candidate`.

**What's the difference between a narrowband candidate and a wideband span?**
A narrowband `Candidate` is a single carrier — a frequency and an SNR, headed for
classification and possibly decode. A wideband span (`IsWideband`, with `BwHz`) is
a plateau far wider than a channel — cellular, WiFi, wide data — stitched across
tiles and named by its allocation rather than decoded. Both live in the same
result list; the `IsWideband` flag distinguishes them.

**Does the guard fraction waste spectrum?**
It never *hides* spectrum — the discarded edges of one tile are covered by the
clean interior of the next, because the step is smaller than the window. It costs
a little sweep time (more steps for the same band) in exchange for never missing a
carrier that happened to sit in a rolloff. The default 10% is a conservative,
reliable trade.

## Series navigation

**Part 2 of 14** · ←
[Part 1: What Discovery Means — From a Blank Band to a Known System]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
· Next →
[Part 3: Peaks & Occupancy — Finding Carriers in the Noise]({{ '/blog/deep-dives/the-hunt-03-peak-occupancy-detection/' | relative_url }})
