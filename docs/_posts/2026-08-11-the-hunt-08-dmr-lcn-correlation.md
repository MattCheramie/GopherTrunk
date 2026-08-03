---
title: "The Hunt, Part 8: DMR LCN Correlation — Rebuilding a Channel Map"
description: How GopherTrunk learns a DMR Tier III band plan that is never broadcast — correlating granted logical channel numbers to the RF carriers that key up in response, confirming each pairing by decoding a DMR sync, and fitting a base-plus-spacing grid it hot-swaps into the live control channel.
category: deep-dives
keywords: dmr tier iii band plan, logical channel number, lcn correlation, onset detection, decode confirmation, base spacing fit, weighted median, hot swap resolver, gophertrunk the hunt
tags: [the-hunt, dmr, trunking, band-plan, correlation, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 8
---

*Part 8 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 7]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
confirmed a P25 system whose band plan is *broadcast* — the `IDEN_UP` messages hand
you base and spacing. DMR Tier III is crueller: a grant tells you a talkgroup is on
Logical Channel Number 7, and nothing on the air tells you what frequency LCN 7
actually is. This part rebuilds that missing map by watching which carrier keys up
when — the same instinct our 851 MHz hunt used to find carriers, now turned on a
system that hides its own channel plan.*

> **TL;DR:** DMR Tier III grants carry an **LCN**, not a frequency, and the
> LCN→frequency map is **not broadcast**. GopherTrunk's `dmrlcn.Learner` reconstructs
> it empirically: an **onset detector** watches a wideband tune for carriers keying
> up on the DMR grid; a **correlator** pairs each onset with a recently-granted LCN
> (only when the pairing is unambiguous); a **confirmer** briefly DDC-taps the
> carrier and decodes a DMR sync to prove it's really that call; and a **fitter**
> solves a regular `base + (LCN)·spacing` grid — or falls back to an explicit table —
> then **hot-swaps** the learned plan into the running control channel.

**Key takeaways**

- **The map is learned, not read.** No message carries the DMR LCN→frequency
  mapping, so it's inferred from grant/onset coincidences plus decode confirmation.
- **Ambiguous pairings are dropped, not guessed.** If two different LCNs were granted
  near one onset, the correlator refuses to pair — repeated observations eventually
  yield an unambiguous match for each LCN.
- **Confirmation weighs the evidence.** A DDC tap that decodes a *voice* sync weighs
  more than a data burst, which weighs more than bare temporal coincidence — and the
  weight biases the fitter's estimates and confidence.
- **A regular grid beats a table.** The fitter snaps spacing to a known DMR grid and
  rejects outliers; only an irregular layout falls back to a per-LCN table.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Learner | own the goroutines, apply the plan | `internal/scanner/dmrlcn/dmrlcn.go` (`Learner`) |
| Onset detector | grid-power idle→active edges | `internal/scanner/dmrlcn/onset.go` (`onsetDetector`) |
| Correlator | pair onset with a granted LCN | `internal/scanner/dmrlcn/correlate.go` (`correlator`) |
| Confirmer | DDC-tap and decode a DMR sync | `internal/scanner/dmrlcn/confirm.go` (`ddcConfirmer`) |
| Fitter | solve base+spacing or a table | `internal/scanner/dmrlcn/fit.go` (`FitBandPlan`) |
| Result | render as a resolver / writeback | `internal/scanner/dmrlcn/fit.go` (`FitResult`) |

## In this post

- **The missing map** — why a DMR grant can't tell you a frequency.
- **Detecting an onset** — grid-power edges with hysteresis.
- **Correlating** — pairing an onset with a grant, unambiguously.
- **Confirming** — proving the carrier is really that call, with a weight.
- **Fitting and hot-swapping** — solving the grid and applying it live.

## The missing map

On a P25 trunk, a neighbour arrives as `(channel id, number)` and the broadcast band
plan resolves it (Part 7). DMR Tier III grants instead carry a bare **Logical Channel
Number** — LCN 1, LCN 2, and so on — and the mapping from LCN to downlink frequency
lives only in the *system's own configuration*, never on the air. A scanner handed a
DMR Tier III system it's never seen knows a talkgroup just moved to LCN 7 and has no
idea where to point. So GopherTrunk learns the map the only way available: by
watching. When a grant says "LCN 7," some carrier on the DMR channel grid keys up a
fraction of a second later. Do that enough times, confirm each pairing by actually
decoding DMR on the carrier, and the map falls out. The `Learner` owns that whole
pipeline:

```go
// internal/scanner/dmrlcn/dmrlcn.go (shape) — Learner
// Learner discovers a DMR Tier III system's LCN→frequency band plan from observed
// voice grants and the RF carriers that key up in response, then hot-swaps the
// learned plan into the running control channel. One Learner serves one wideband
// dongle / system.
type Learner struct {
    // …bus, broker, system, confirmer, fitOpts, onsetCfg
    corr   *correlator
    pairs  []Pair
    locked bool
}
```

`Run` wires four inputs into one loop: the wideband IQ broker (feeding the onset
detector), the event bus (feeding grant observations to the correlator), a bounded
pool of confirmation probe workers, and an expiry ticker. It never returns an error —
failures are logged and the learner degrades to a no-op — and once the fitter locks a
plan, it stops.

## Detecting an onset

The onset detector runs windowed FFTs over the wideband stream, integrates power onto
the **DMR channel grid** (12.5 kHz), and emits a `CarrierOnset` on each channel's
idle→active edge. The per-channel state machine uses hysteresis and an adaptive noise
floor so a keyup is a real edge, not a flicker:

```go
// internal/scanner/dmrlcn/onset.go (shape) — updateChannel
onLevel := gc.floorDb + d.cfg.OnThreshDb   // default +10 dB
offLevel := gc.floorDb + d.cfg.OffThreshDb // default +6 dB (hysteresis)
switch {
case db >= onLevel:
    gc.aboveCount++; gc.belowCount = 0
    if !gc.active && gc.aboveCount >= d.cfg.Debounce { // default 3 frames
        gc.active = true
        if !gc.excluded {
            d.emit(CarrierOnset{FreqHz: gc.freqHz, At: now, PeakDb: float32(db)})
        }
    }
case db <= offLevel:
    gc.belowCount++; gc.aboveCount = 0
    if gc.active && gc.belowCount >= d.cfg.Debounce { gc.active = false }
}
// Track the noise floor only while idle so an active carrier doesn't mask itself.
if !gc.active {
    gc.floorDb = (1-d.cfg.NoiseAlpha)*gc.floorDb + d.cfg.NoiseAlpha*db
}
```

Three details make it robust. **Debounce** requires several consecutive frames past
the threshold before flipping state, so a single noisy frame can't fake a keyup.
**Hysteresis** — a higher on-level than off-level — stops a carrier hovering at
threshold from chattering. And the noise floor tracks *only while the channel is
idle*, so an active carrier doesn't drag its own floor up and mask itself. The
`Exclude` list drops the always-on control-channel carriers, which would otherwise
generate phantom onset edges. `buildGrid` precomputes, per grid channel, the exact
FFT bins that fall inside its occupied bandwidth, so each frame is just a sum over
those bins.

## Correlating — unambiguously

An onset alone means "something keyed up here." To turn it into an LCN pairing, the
correlator matches it against recently-granted LCNs within a delay window — a carrier
keys up shortly *after* the grant CSBK decodes:

```go
// internal/scanner/dmrlcn/correlate.go (shape) — timing window
const (
    defaultMinDelay = -100 * time.Millisecond // absorbs clock skew
    defaultMaxDelay = 700 * time.Millisecond
    defaultGrantTTL = 2 * time.Second
)
```

The critical rule is what happens when the window is *ambiguous*. If grants for more
than one distinct LCN fall inside the window, the correlator does **not** guess — it
drops the onset entirely:

```go
// internal/scanner/dmrlcn/correlate.go (shape) — matchOnset ambiguity guard
switch {
case !haveLCN:
    lcn, grant, haveLCN = g.obs.LCN, g.obs, true
    idxs = append(idxs, i)
case g.obs.LCN == lcn:
    idxs = append(idxs, i)
default:
    multiLCN = true // a different LCN also fits the window
}
if !haveLCN || multiLCN {
    return Candidate{}, false // ambiguous ⇒ refuse to pair
}
```

This is the same conservatism as the P25 identity note: don't claim what you can't be
sure of. A busy system grants several LCNs a second, and near-simultaneous grants make
a single onset genuinely ambiguous. Rather than pair it wrongly and poison the fit,
the correlator waits — over many observations each LCN eventually gets an onset it's
alone in the window for. Matched grants are consumed so they can't re-match a later
onset, and an expiry sweep drops grants past their TTL so the pending slice stays
bounded.

## Confirming — with a weight

A temporal coincidence is suggestive, not proof — RF is messy, and an adjacent
carrier could key up in the window by chance. So each candidate is *confirmed* by
briefly DDC-tapping the shared wideband stream at the carrier's offset and running the
DMR receiver until it decodes a sync word:

```go
// internal/scanner/dmrlcn/confirm.go (shape) — ddcConfirmer.Confirm
offset := float64(freqHz) - float64(c.centerHz)
bank := tuner.NewDDCBank(float64(c.sampleRateHz), confirmNarrowbandHz, c.guardFrac)
// …rx decodes; the DibitSink counts DMR sync matches, noting Voice vs Data…
for {
    select {
    case <-deadline.C:
        return false, 0 // ~700 ms probe timed out — not a live DMR burst
    case chunk, ok := <-sub.C:
        bank.Process(chunk)
        if syncCount >= c.minSyncs {
            return true, syncWeight(sawVoice, sawData)
        }
    }
}
```

Confirmation does two jobs. It rejects false coincidences — a probe that decodes no
DMR sync in ~700 ms returns `false`, and the candidate is discarded. And it *weights*
the evidence: a **voice** sync is worth 1.0, a **data** burst 0.8, anything else 0.7.
That weight rides on the resulting `Pair` and biases the fitter's weighted-median
estimates and confidence downstream. The probes run on a bounded worker pool
(`maxConcurrentProbes`) off the main loop, so a ~700 ms tap never stalls IQ pumping
into the onset detector. The DMR receiver constants mirror the wideband Tier III path
exactly, so a probe tap decodes identically to the dongle's primary control tap.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="The DMR LCN learning pipeline: grants carrying logical channel numbers and carrier onsets from the wideband stream feed a correlator that pairs them when unambiguous, a confirmer that DDC-taps and decodes a DMR sync to weight the pair, and a fitter that solves a base-plus-spacing grid and hot-swaps it into the running control channel">
  <rect x="16" y="30" width="120" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="76" y="46" text-anchor="middle" fill="currentColor" font-size="10">grant · LCN</text>
  <text x="76" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="9">from the bus</text>
  <rect x="16" y="128" width="120" height="38" rx="6" fill="none" stroke="currentColor"/>
  <text x="76" y="144" text-anchor="middle" fill="currentColor" font-size="10">carrier onset</text>
  <text x="76" y="158" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grid keyup edge</text>
  <line x1="136" y1="49" x2="196" y2="80" stroke="currentColor"/><polygon points="192,76 202,84 190,84" fill="currentColor"/>
  <line x1="136" y1="147" x2="196" y2="112" stroke="currentColor"/><polygon points="190,108 202,108 194,117" fill="currentColor"/>
  <rect x="202" y="78" width="120" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="262" y="94" text-anchor="middle" fill="var(--accent)" font-size="10">correlator</text>
  <text x="262" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">unambiguous only</text>
  <line x1="322" y1="97" x2="360" y2="97" stroke="currentColor"/><polygon points="360,93 370,97 360,101" fill="currentColor"/>
  <rect x="370" y="78" width="120" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="430" y="94" text-anchor="middle" fill="var(--accent)" font-size="10">confirmer</text>
  <text x="430" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DDC-tap → sync + weight</text>
  <line x1="490" y1="97" x2="528" y2="97" stroke="currentColor"/><polygon points="528,93 538,97 528,101" fill="currentColor"/>
  <rect x="538" y="78" width="128" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="602" y="94" text-anchor="middle" fill="var(--accent)" font-size="10">fitter → hot-swap</text>
  <text x="602" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">base+spacing / table</text>
  <text x="340" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">grant says LCN 7, a carrier keys up 700 ms later, a DMR sync confirms it — repeat until the grid solves</text>
</svg>
<figcaption>Learning the DMR band plan: pair a granted LCN with the carrier that keyed up, confirm it by decoding DMR, weight the evidence, and fit the grid — then apply it live.</figcaption>
</figure>

### How that principle shaped the Go code

- **Confirmation is an interface.** `Confirmer` is one method, so tests inject a fake
  and the learner can run confirm-less at a reduced weight (0.4, temporal-only
  evidence) when no confirmer is wired.
- **Probes are bounded and off-loop.** `maxConcurrentProbes` worker goroutines drain
  a candidate channel, so the slow DDC tap never blocks the fast onset detection —
  the queue simply drops when full, and that LCN is observed again.
- **The fitter is pure.** `FitBandPlan` does no I/O and owns no state; every branch —
  linear fit, outlier rejection, table fallback — is exhaustively unit-testable in
  isolation from the goroutine machinery.

## Fitting and hot-swapping

Confirmed `(LCN, frequency, weight)` pairs feed `FitBandPlan`, which tries to solve
the regular grid most DMR Tier III sites use before falling back to a table. It
estimates spacing from the median per-LCN slope, **snaps it to a known DMR grid**
(6.25 / 12.5 / 25 kHz) within tolerance, solves the base by weighted median, then
iteratively rejects the worst-residual outlier until everything fits:

```go
// internal/scanner/dmrlcn/fit.go (shape) — fitLinear outlier rejection
worst, maxRes := worstResidual(work, base, spacing, fitOffset)
if maxRes <= opts.ResidualTolHz { // 1.5 kHz ceiling
    if len(work) < opts.MinLCNs { return FitResult{}, false }
    return FitResult{ Linear: &trunking.DMRLinearBandPlan{ BaseHz: base, SpacingHz: spacing, Offset: fitOffset },
        Confidence: linearConfidence(work, maxRes, opts), NumPairs: len(work), ResidualHz: maxRes }, true
}
if len(work)-1 < opts.MinLCNs { return FitResult{}, false }
work = append(work[:worst], work[worst+1:]...) // drop the worst and refit
```

A regular grid is preferred because it *generalizes*: solve base and spacing from a
handful of confirmed LCNs and you can resolve every LCN the system will ever grant,
not just the ones you happened to observe. Only an irregular layout falls back to an
explicit per-LCN table. When the fit locks (default: at least 4 distinct LCNs), the
learner `apply`s it — hot-swapping a `tier3.Resolver` built from the plan into the
*running* control channel, publishing `KindDMRBandPlanLearned`, and optionally
persisting it back to config. The system that hid its channel map is now fully
scannable, learned live, with no restart.

## Where this goes next

We've now confirmed a P25 system whose plan is broadcast (Part 7) and learned a DMR
system whose plan is hidden. [Part 9]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }})
scales the P25 case up: watching *many* channels of a multi-site P25 system at once
across a single wideband tune, so a whole site's control and voice activity is
observed together instead of one channel at a time.

## FAQ

**Why can't GopherTrunk just read the DMR band plan like it reads P25's?**
Because DMR Tier III doesn't broadcast one. P25 sends `IDEN_UP` messages with base and
spacing; DMR grants carry only a Logical Channel Number, and the LCN→frequency mapping
lives in the system's private configuration. The only way to recover it without that
config is to observe which carrier keys up for which granted LCN.

**What stops a wrong LCN→frequency pairing from poisoning the map?**
Three guards. The correlator refuses to pair an ambiguous onset (more than one LCN in
the window). The confirmer requires an actual decoded DMR sync on the carrier, not
just a temporal coincidence. And the fitter rejects outlier pairs whose residual
exceeds tolerance before committing a linear plan.

**Why weight voice syncs higher than data?**
A voice grant keying up a voice call is the cleanest possible evidence that the
carrier belongs to that LCN. A data burst is still confirmation but slightly weaker,
and bare temporal coincidence (no confirmer) is weakest at 0.4. The weight biases the
fitter's weighted-median base estimate and its confidence, so stronger evidence counts
for more.

**When does it fall back to a table instead of a grid?**
When the confirmed pairs don't fit a regular `base + LCN·spacing` grid within the snap
and residual tolerances after outlier rejection. Most sites are regular, so the linear
fit usually wins — but an irregular allocation gets an explicit per-LCN table so it's
still usable.

**Does learning the plan interrupt scanning?**
No. The learner runs alongside the live control channel on the shared wideband dongle,
and when the fit locks it hot-swaps a new resolver into the *running* control channel
via `SetResolver`. There's no retune and no restart — the system simply becomes
resolvable mid-operation.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Locking a P25 System — Candidate to Confirmed]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
· Next →
[Part 9: Wideband Multi-Site P25 — Watching Many Channels at Once]({{ '/blog/deep-dives/the-hunt-09-wideband-multisite-p25/' | relative_url }})
