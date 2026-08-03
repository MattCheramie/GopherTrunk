---
title: "The Hunt, Part 3: Peaks & Occupancy — Finding Carriers in the Noise"
description: How GopherTrunk estimates a robust noise floor, picks true carrier peaks without triple-counting one emitter, snaps them onto the land-mobile channel raster, and separates narrow carriers from wide OFDM plateaus in one spectrum frame.
category: deep-dives
keywords: noise floor estimation, peak detection, occupancy grid, low quartile percentile, channel raster snap, local maximum, minimum spacing, wideband occupancy, gophertrunk the hunt
tags: [the-hunt, dsp, carriers, spectrum, detection, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 3
---

*Part 3 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 2]({{ '/blog/deep-dives/the-hunt-02-wideband-sweep-engine/' | relative_url }})
tiled a wide band into overlapping receiver windows and handed each one an averaged
power spectrum. This part is what happens **inside** one of those frames: turning a
row of dB-per-bin numbers into a short list of "there is a carrier here." Our
851–869 MHz stray is, at this stage, one bump in a noisy line — and the job is to
be sure it's a carrier, get its frequency right, and not mistake it for three.*

> **TL;DR:** Carrier detection is three sub-problems. First a **robust noise
> floor** — the low quartile of the bin powers, which a strong carrier can't drag
> up. Then **peak picking** — strict local maxima that stand a threshold above the
> floor, with a minimum spacing so one emitter isn't reported three times. Then a
> **grid snap** — infer the channel raster the carriers sit on and correct the FFT
> quantization so the reported frequency actually locks. A separate **occupancy**
> scan finds plateaus too wide to be a channel. It all lives in the neutral
> `internal/carriers` package, shared by the live sweep and the offline survey.

**Key takeaways**

- **The floor is a percentile, not a mean.** The low quartile of bin powers is
  robust to strong carriers occupying the upper tail — a mean would be pulled up by
  the very signals you're trying to find.
- **A peak must be a strict local maximum above threshold.** Ten dB over the floor
  *and* higher than its immediate neighbours, with DC and the band-edge guard bins
  excluded, keeps noise ripples and the DC spike out.
- **Minimum spacing collapses an emitter's shoulders.** Strongest-first, suppress
  anything within `MinSpacingHz` — one carrier yields one peak, not a cluster.
- **The grid snap makes carriers lockable.** Inferring the raster (6.25/12.5/25 kHz)
  and snapping within one bin turns an 851.0123 MHz FFT artifact back into the real
  851.012500 MHz channel centre.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Peak type | a found carrier: freq, power, SNR | `internal/carriers/peaks.go` (`Peak`) |
| Peak detector | floor → local maxima → spacing | `internal/carriers/peaks.go` (`DetectPeaks`) |
| Occupancy type | one wide contiguous span | `internal/carriers/occupancy.go` (`Occupancy`) |
| Occupancy scan | coalesce occupied bins into runs | `internal/carriers/occupancy.go` (`DetectOccupancy`) |
| Grid inference | which channel raster fits | `internal/carriers/grid.go` (`InferGrid`) |
| Grid snap | correct sub-bin frequency error | `internal/carriers/grid.go` (`SnapHz`) |
| Re-exports | keep the hunt surface | `internal/hunt/peaks.go` |

## In this post

- **The noise floor** — why a robust percentile beats a mean, and how it's computed.
- **Picking peaks** — the local-maximum test, the guards, and the spacing rule.
- **Snapping to the grid** — inferring the raster and correcting the FFT offset.
- **Occupancy** — the wideband counterpart and the edge-clip flag that enables stitching.
- **One package, two callers** — why this lives below both hunt and siglab.

## The noise floor: a robust percentile

Every threshold in carrier detection is measured *relative to the noise floor*, so
the floor estimate has to be right — and it has to be right in the presence of the
very carriers you're hunting. If you estimate the floor as the mean of the bin
powers, a band with a few strong carriers reports a floor that's been dragged
upward by those carriers, and weak signals sink below your threshold. The fix is to
use a low percentile:

```go
// internal/carriers/peaks.go (shape)
// noiseFloorPercentile is the percentile of bin powers used as the noise-floor
// estimate. The low quartile is robust to strong carriers occupying the upper
// tail, unlike a mean.
const noiseFloorPercentile = 0.25

floor := spectrum.Percentile(frame.Bins, noiseFloorPercentile)
```

The **low quartile** — the 25th percentile of the sorted bin powers — sits down in
the noise regardless of how many carriers occupy the top of the distribution.
Carriers live in the upper tail; the lower quartile is, by construction, mostly
empty spectrum. This one choice is why the detector works on a busy band and a
quiet one alike, and it's why the sweeper's per-step floor (Part 2) and this
detector agree — they compute the same quantity.

## Picking peaks

With a floor, a peak is a bin that (a) stands `ThresholdDb` above it, and (b) is a
strict local maximum against its immediate neighbours — plus we skip DC and the
band-edge guard bins where the SDR rolloff lives:

```go
// internal/carriers/peaks.go (shape) — DetectPeaks inner loop
dcBin := n / 2 // FFT-shifted: DC sits in the middle
for i := guard; i < n-guard; i++ {
    if i == dcBin {
        continue
    }
    v := frame.Bins[i]
    if v-floor < threshold {
        continue
    }
    // Strict local maximum against immediate neighbors.
    if v <= frame.Bins[i-1] || v < frame.Bins[i+1] {
        continue
    }
    peaks = append(peaks, Peak{
        FreqHz:    binToHz(frame, i, binHz),
        PowerDbFS: v,
        SNRDb:     v - floor,
    })
}
```

The defaults are chosen for land-mobile radio: `ThresholdDb` 10 dB, `GuardBins`
about 1.5% of the band at each edge (`n/64`), and DC excluded because an SDR's DC
offset produces a persistent centre spike that is emphatically not a carrier. The
asymmetric comparison (`<=` on the left, `<` on the right) is a deliberate
tie-break so a perfectly flat two-bin plateau reports its right bin exactly once
rather than zero or twice.

That first pass can still report a strong carrier's shoulders as separate peaks, so
a second pass enforces minimum spacing — strongest first, suppress the rest:

```go
// internal/carriers/peaks.go (shape) — spacing suppression
sort.Slice(peaks, func(a, b int) bool { return peaks[a].SNRDb > peaks[b].SNRDb })
var kept []Peak
for _, p := range peaks {
    ok := true
    for _, k := range kept {
        if absDiff(p.FreqHz, k.FreqHz) < minSpacing {
            ok = false
            break
        }
    }
    if ok {
        kept = append(kept, p)
    }
}
```

`MinSpacingHz` defaults to 6.25 kHz — the tightest trunking channel step — so two
genuinely adjacent channels survive but one carrier's spectral shoulders don't
spawn phantom neighbours. Because we walk strongest-first, the *carrier* always
wins its neighbourhood and the shoulders are the ones suppressed, never the reverse.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="A spectrum frame with a robust low-quartile noise floor line, a detection threshold line ten dB above it, one tall carrier peak that clears the threshold and is kept, and small noise ripples below the threshold that are rejected">
  <line x1="30" y1="30" x2="30" y2="160" stroke="var(--fg-muted)"/>
  <line x1="30" y1="160" x2="650" y2="160" stroke="var(--fg-muted)"/>
  <text x="8" y="34" fill="var(--fg-muted)" font-size="9">dB</text>
  <line x1="30" y1="135" x2="650" y2="135" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="590" y="131" fill="var(--fg-muted)" font-size="9">floor (Q25)</text>
  <line x1="30" y1="95" x2="650" y2="95" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="560" y="91" fill="var(--accent)" font-size="9">floor + threshold</text>
  <polyline points="30,138 90,134 140,140 190,132 240,136 300,138 340,135 360,55 380,135 430,140 480,133 540,138 600,134 650,137" fill="none" stroke="currentColor"/>
  <circle cx="360" cy="55" r="4" fill="var(--accent)"/>
  <text x="360" y="46" text-anchor="middle" fill="var(--accent)" font-size="9">kept peak</text>
  <line x1="360" y1="160" x2="360" y2="168" stroke="var(--accent)"/>
  <text x="360" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="9">snapped to nearest 6.25/12.5/25 kHz grid point</text>
  <text x="180" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">noise ripples: below threshold → rejected</text>
</svg>
<figcaption>Detection in one frame: a robust floor, a threshold above it, strict local maxima that clear the threshold kept, everything below rejected, the survivor snapped to the channel grid.</figcaption>
</figure>

## Snapping to the grid

An FFT quantizes frequency to `SampleRate/N`-wide bins. At 2.4 MS/s with a 4096-bin
FFT that's ~586 Hz per bin — so a carrier truly on 851.012500 MHz can report a few
hundred Hz off, and a decoder tuned to the reported value hands its carrier-recovery
loop a worse starting point than it needs. The `grid.go` logic corrects this by
first *inferring* the raster the carriers collectively sit on:

```go
// internal/carriers/grid.go (shape) — InferGrid
// For each standard step it treats the residuals (f mod step) as points on a
// circle and measures their concentration (mean resultant length); the raster
// fits when the residuals pile up at one phase.
func InferGrid(freqs []uint32) (stepHz, phaseHz uint32) {
    if len(freqs) < minGridCandidates {
        return standardChannelSteps[0], 0 // finest step, phase 0
    }
    bestStep, bestPhase := uint32(0), uint32(0)
    for _, step := range standardChannelSteps { // 6.25, 12.5, 25 kHz
        phase, r := gridFit(freqs, step)
        if r >= gridFitThreshold { // 0.8: tightly clustered
            bestStep, bestPhase = step, phase // coarsest qualifying wins
        }
    }
    // …fall back to the finest step at phase 0 when nothing clusters
}
```

The trick is treating each candidate's residual modulo the step as an **angle on a
circle** and measuring the mean resultant length — a circular-statistics
concentration score in `[0,1]`. If the carriers really sit on a 12.5 kHz raster,
their residuals pile up at one phase and the resultant is near 1; if they're
scattered, it's near 0. The coarsest step that clears 0.8 wins, because a system on
25 kHz channels also happens to fit 12.5 and 6.25 kHz, and we want the *real* plan.

Then `SnapHz` moves each carrier to the nearest grid point — but only if that point
is within a bounded correction (one FFT bin), so a genuine off-raster signal is left
exactly where it is:

```go
// internal/carriers/grid.go (shape) — SnapHz
k := math.Round((float64(f) - float64(phase)) / float64(step))
nearest := int64(phase) + int64(k)*int64(step)
d := int64(f) - nearest
if d < 0 { d = -d }
if d <= int64(maxCorrectionHz) {
    return uint32(nearest) // correct the bin quantization
}
return f // genuinely off-grid — leave it
```

With too few candidates to infer anything, it falls back to the finest step at
phase 0 — so even a lone carrier (the 440.125 MHz repro case in the comments) still
snaps to the nearest 6.25 kHz point, correcting the bin offset without needing a
whole band to vote.

### How that principle shaped the Go code

- **The bound is the safety valve.** Snapping is *always* limited to
  `maxCorrectionHz`. An analog channel on a different plan, or a carrier that
  genuinely sits off-raster, is never yanked onto a grid it doesn't belong to.
- **Coarsest-fit-wins avoids over-fitting.** Because finer rasters trivially fit
  coarser channels, walking finest-to-coarsest and keeping the last qualifying step
  recovers the true channel plan instead of the tightest one that happens to match.
- **The floor computation is shared.** `DetectOccupancy` uses the same
  `noiseFloorPercentile` low-quartile estimate as `DetectPeaks`, so the two
  detectors never disagree about where the noise is.

## Occupancy: the wideband counterpart

A trunking channel is narrow; a cellular or WiFi block is a wide, flat plateau that
peak detection ignores by design. `DetectOccupancy` is the mirror image: a lower
threshold (6 dB — a wide plateau is flatter per-bin than a narrow spike), then
coalesce *contiguous* above-floor bins into runs and keep those at least `MinBwHz`
wide:

```go
// internal/carriers/occupancy.go (shape) — run coalescing
for i := guard; i <= lastScan; i++ {
    occupied := frame.Bins[i]-floor >= threshold
    switch {
    case occupied && runStart < 0:
        runStart = i
    case !occupied && runStart >= 0:
        flush(runStart, i-1) // emit the run if ≥ MinBwHz
        runStart = -1
    }
}
```

`MinBwHz` defaults to **200 kHz** — about 16× the widest C4FM channel — so a
narrowband carrier can never be mistaken for a wideband span; only genuinely wide
emitters register. Each emitted `Occupancy` carries `EdgeClippedLow`/`High` flags
set when its run reaches the first or last scanned bin. Those flags are the hook
Part 2's `stitchWideband` uses: a span clipped at a tile edge is a signal wider
than the tune, and its true width is recovered by joining it to the clipped span in
the adjacent overlapping tile. The `FloorDbFS` override exists for exactly the case
where a signal fills a whole step and the per-frame floor sits *inside* it — the
sweeper passes a sweep-wide floor so a fully-occupied tile is still recognised.

## One package, two callers

All of this lives in `internal/carriers`, not `internal/hunt`, and that placement
is deliberate. The live sweeper (`hunt`) and the offline wideband survey (`siglab`)
both need identical peak and channel-grid logic, and the surest way for two copies
to drift apart is to have two copies. So the import direction is
`carriers ← {hunt, siglab}`, never the reverse, and `internal/hunt/peaks.go` is a
thin set of type aliases and wrappers that keep the sweeper's existing surface
(`hunt.Peak`, `hunt.DetectPeaks`) while the implementation lives once in the neutral
package. What the offline survey finds in a recording is, bin for bin, what the live
sweep finds on the air.

## Where this goes next

[Part 4]({{ '/blog/deep-dives/the-hunt-04-classifying-a-signal/' | relative_url }})
takes the ranked, grid-snapped candidates this stage produces and asks the next
question: *what is each one?* Before we spend an expensive protocol identify on our
851 MHz carrier, a cheap blind classifier decides whether it's analog, digital, or
encrypted — narrowing forty carriers down to the handful worth decoding.

## FAQ

**Why the low quartile specifically, and not the median?**
The median (50th percentile) can itself sit inside signal energy on a heavily
occupied band, biasing the floor upward. The low quartile stays down in genuinely
empty spectrum as long as fewer than ~75% of bins are occupied, which holds for any
realistic land-mobile band. It's a conservative, robust choice.

**What if two real carriers sit closer than the minimum spacing?**
With the 6.25 kHz default they won't — that's the tightest channel step any
land-mobile system uses, so two distinct channels are always at least that far
apart. If you're surveying an unusual plan you can lower `MinSpacingHz`, at the risk
of a strong carrier's shoulders re-appearing as phantom peaks.

**Why infer the grid instead of assuming 12.5 kHz?**
Different systems and regions use 6.25, 12.5, or 25 kHz rasters, and analog channels
may sit off any of them. Inferring the grid from the carriers themselves means the
snap corrects real bin-quantization error without imposing a plan that doesn't
apply — and the bounded correction guarantees an off-grid carrier is never moved.

**How does occupancy detection avoid flagging a normal carrier as wideband?**
Two guards: a `MinBwHz` floor of 200 kHz (far wider than any narrowband channel),
and a contiguity requirement (the run of above-threshold bins must be unbroken).
A 12.5 kHz P25 channel is neither wide enough nor, at the occupancy threshold, a
broad plateau, so it never registers as an `Occupancy`.

**Is any of this protocol-specific?**
No — that's the point of the `carriers` package. Peak detection, floor estimation,
and grid snapping are pure spectrum operations with no knowledge of P25 or DMR.
Protocol identity enters only later, at classification (Part 4) and identify
(Part 7). Keeping detection protocol-neutral is what lets the same code serve every
band.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Wideband Sweep Engine]({{ '/blog/deep-dives/the-hunt-02-wideband-sweep-engine/' | relative_url }})
· Next →
[Part 4: Classifying a Signal — Analog, Digital, Encrypted]({{ '/blog/deep-dives/the-hunt-04-classifying-a-signal/' | relative_url }})
