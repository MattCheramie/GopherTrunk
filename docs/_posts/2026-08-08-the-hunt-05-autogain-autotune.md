---
title: "The Hunt, Part 5: Autogain & Autotune — Settling the Front End"
description: How GopherTrunk sweeps front-end gain to minimize the decode error rate before committing to a control channel, and how a running-average autotune corrects each dongle's crystal offset so a decoder's carrier loop starts already near lock.
category: deep-dives
keywords: sdr gain sweep, decode error rate, front end gain staging, autotune crystal offset, ppm correction, afc residual, running average correction, error rate minimization, gophertrunk the hunt
tags: [the-hunt, sdr, gain, autotune, front-end, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 5
---

*Part 5 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 4]({{ '/blog/deep-dives/the-hunt-04-classifying-a-signal/' | relative_url }})
classified our 851 MHz carrier as digital C4FM — a strong P25 control-channel
candidate. But classification and the identify to come both assume the front end is
delivering the best signal the radio can give. This part is about earning that
assumption: two small, empirical loops that settle the analog front end before we
ask it to lock. One sweeps gain against a decode-quality number; the other corrects
the dongle's crystal offset.*

> **TL;DR:** Two front-end corrections, both **empirical**. **Autogain** sweeps a
> ladder of front-end gains at each candidate control channel, decodes a short
> dwell at each, and recommends the gain that **minimizes the decode error rate**
> (`CaptureReport.ErrorRate` from Part 1) — preferring a gain that locks, breaking
> ties toward less front-end strain. **Autotune** tracks each dongle's residual
> carrier offset — read from the decoder's AFC loop once it locks — as a
> running average, and reports a digital tuning correction so the *next* decode
> starts already near lock. Neither is applied silently: gain is advisory, autotune
> reports a value the operator bakes in.

**Key takeaways**

- **Gain is chosen by decode quality, not by signal strength.** The score is the
  protocol-neutral error rate straight from the decode, so the sweep optimizes the
  thing that actually matters — whether the control channel decodes cleanly.
- **The tie-break prefers a lock, then a lower gain.** Among gains that decode, one
  that locks wins; among equal error rates, the lower gain wins, sparing the front
  end intermodulation strain.
- **Autotune is a running average, glitch-guarded.** A single unconverged AFC
  reading can't yank tuning: implausible measurements are rejected, and a warm-up
  gate holds the correction at zero until enough samples accumulate.
- **Nothing is applied behind your back.** Autogain returns recommendations;
  autotune reports a suggested config value (and PPM) — the operator decides.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Gain sweep | score each gain, recommend one | `internal/hunt/autogain.go` (`AutoGainSweep`) |
| Per-gain score | set gain, tune, decode, read error rate | `internal/hunt/autogain.go` (`scoreGain`) |
| Best-gain pick | lock, then error rate, then lower gain | `internal/hunt/autogain.go` (`pickGain`) |
| Offset tracker | running-average carrier error | `internal/autotune/autotune.go` (`Manager`) |
| Measurement fold | reject glitches, average, warn on PPM | `internal/autotune/autotune.go` (`AddErrorMeasurement`) |
| Per-dongle registry | one Manager per serial | `internal/autotune/autotune.go` (`Registry`) |

## In this post

- **Two loops, one goal** — why gain and tuning are separate corrections.
- **Scoring a gain** — the set-tune-decode cycle and the number it produces.
- **Picking the best** — the lock-first, error-rate, lower-gain ordering.
- **Tracking the offset** — the running average and its glitch guards.
- **Warm-up and application** — why the correction stays zero at first.

## Two loops, one goal

The analog front end sits between the antenna and every bit GopherTrunk decodes, and
it has two independent ways of being wrong. It can be at the wrong **gain** — too
low and a weak carrier sinks into the noise; too high and strong neighbours drive
the amplifier into intermodulation, splattering spurs across the band. And it can be
at the wrong **frequency** — every RTL-SDR crystal is a little off, so the received
carrier lands off centre, and the decoder's carrier-recovery loop has to pull it in
from further away than it should.

These are separate problems with separate fixes. Gain is a discrete choice from the
tuner's ladder, so we *search* it: try several, measure decode quality, keep the
best. Frequency error is a slowly-drifting scalar, so we *track* it: measure the
residual each time the loop locks and average. This part is both loops. Crucially,
both are measured against real decoding — no proxy, no guesswork about what "good
signal" means.

## Scoring a gain

Autogain requires a gain-controllable, exclusively-held SDR — it mutates the device
gain as it runs, which the daemon's shared broker source can't allow. Given one, the
core is `scoreGain`: set the gain, tune (which drains the settle transient including
the gain change), capture a short dwell, decode it, and read out the outcome:

```go
// internal/hunt/autogain.go (shape) — scoreGain
sc := GainScore{GainTenthDB: gain}
if err := gc.SetGain(gain); err != nil { /* … */ }
if err := src.Tune(freq); err != nil { /* … */ }
iq, err := captureN(ctx, src, n)
// …
buf := siglab.EncodeCapture(iq, siglab.FormatF32)
rep := decodeAndAccumulate(&DiscoveredSystem{}, bytes.NewReader(buf),
    fmt.Sprintf("%.4f MHz @ gain %.1f dB", float64(freq)/1e6, float64(gain)/10),
    decodeParams{ /* Protocol, SampleRateHz, FrequencyHz, AutoTune, … */ })
sc.ErrorRate = rep.ErrorRate
sc.Locked = rep.Locked
```

The decode goes through `decodeAndAccumulate` — the *same* identify→decode body the
offline `Discover` and the live hunter use (Part 1). It throws the accumulated
system away (`&DiscoveredSystem{}` is a scratch sink); all it wants is the
`CaptureReport`. Two fields matter: `Locked` (did the control channel lock at this
gain?) and `ErrorRate` — the protocol-neutral decode-error events per 1000 symbols
that Part 1 promised the auto-gain sweep would use. That field is the whole reason
the sweep can be honest: it optimizes *decode quality*, not a power meter's idea of
strength. A higher gain that lifts the carrier but also lifts intermod can *raise*
the error rate, and the sweep will see that and reject it.

Gains are given in tenths of a dB; the blind default ladder spreads across a
plausible range, but RTL-SDR and Airspy ladders differ, so an operator should pass a
device-appropriate set:

```go
// internal/hunt/autogain.go (shape)
var defaultGainSet = []int{0, 87, 166, 240, 297, 370, 440, 496}
```

## Picking the best

Every gain's score goes into `pickGain`, which encodes the priorities in one
readable cascade:

```go
// internal/hunt/autogain.go (shape) — pickGain ordering
b := scores[best]
switch {
case s.Locked != b.Locked:
    if s.Locked { best = i }          // 1. a gain that LOCKS beats one that doesn't
case s.ErrorRate != b.ErrorRate:
    if s.ErrorRate < b.ErrorRate { best = i } // 2. lower decode error rate
case s.GainTenthDB < b.GainTenthDB:
    best = i                           // 3. lower gain (less front-end strain)
}
```

Read top to bottom, that's the whole philosophy. A lock is worth more than any error
rate — a control channel that locks at a mediocre error rate is more useful than one
that doesn't lock at a pristine one. Among equals, lower error rate. And the final
tie-break prefers the *lower* gain, because two gains that decode equally well are
not equally kind to the front end — the lower one leaves more headroom before a
strong neighbour drives the amplifier into intermod. Scores whose capture or decode
errored are skipped entirely, so a gain that produced nothing usable never wins by
accident. The recommendations are advisory: `AutoGainSweep` returns a
`GainRecommendation` per control channel and applies nothing — the operator restores
the working gain and decides whether to bake the suggestion in.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A U-shaped curve of decode error rate versus front-end gain: error rate is high at low gain where the carrier is weak, dips to a minimum in the middle where the channel locks cleanly, and rises again at high gain where intermodulation degrades the decode; the minimum is the recommended gain">
  <line x1="50" y1="30" x2="50" y2="160" stroke="var(--fg-muted)"/>
  <line x1="50" y1="160" x2="640" y2="160" stroke="var(--fg-muted)"/>
  <text x="20" y="40" fill="var(--fg-muted)" font-size="9">error</text>
  <text x="20" y="52" fill="var(--fg-muted)" font-size="9">rate</text>
  <text x="600" y="178" fill="var(--fg-muted)" font-size="9">gain (dB)</text>
  <polyline points="80,55 150,90 230,120 320,140 400,138 470,118 540,85 610,50" fill="none" stroke="currentColor"/>
  <circle cx="360" cy="141" r="5" fill="var(--accent)"/>
  <text x="360" y="128" text-anchor="middle" fill="var(--accent)" font-size="9">recommended</text>
  <line x1="360" y1="160" x2="360" y2="168" stroke="var(--accent)"/>
  <text x="150" y="80" fill="var(--fg-muted)" font-size="9">weak: carrier in noise</text>
  <text x="470" y="76" fill="var(--fg-muted)" font-size="9">hot: intermod spurs</text>
  <text x="345" y="196" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the sweep decodes at each gain and picks the error-rate minimum — preferring a gain that locks, then the lower gain on a tie</text>
</svg>
<figcaption>Decode error rate against gain is U-shaped: too low buries the carrier, too high adds intermod. The sweep decodes at each rung and recommends the minimum.</figcaption>
</figure>

### How that principle shaped the Go code

- **The gain sweep reuses the real decode path.** `scoreGain` calls
  `decodeAndAccumulate`, not a bespoke measurement — so the number it optimizes is
  exactly the number the live hunt produces. There's no separate "quality metric" to
  drift from reality.
- **Capability, not assumption.** Autogain is gated behind the `GainSettable`
  interface, which only the standalone live-SDR source implements. The daemon's
  shared broker source doesn't, so the feature is unavailable exactly when changing
  gain would disrupt other consumers — the type system encodes the constraint.
- **Advisory by construction.** Both loops return values; neither writes hardware
  state as a side effect. A hunt can't silently leave your dongle at a strange gain.

## Tracking the offset

Autotune is the second loop, and it's a *tracker*, not a search. Once a decoder
locks, its carrier-recovery loop knows how far off centre the carrier was — the
`AFCOffsetHz`. The `autotune.Manager` folds each such measurement into a running
average (ported in concept from trunk-recorder's `AutotuneManager`) and reports it as
a digital correction the caller applies by shifting the down-converter target:

```go
// internal/autotune/autotune.go (shape) — AddErrorMeasurement
total := observedHz + currentOffsetHz // observed residual + already-applied offset

// Reject implausible measurements before they poison the average.
if bound := m.rejectBoundHz(centerFreqHz); abs(total) > bound {
    m.log.Debug("autotune: measurement rejected (implausible, treated as glitch)", /* … */)
    return
}
m.history = append([]int{total}, m.history...) // push_front
if len(m.history) > MaxHistory {
    m.history = m.history[:MaxHistory]          // pop_back, cap at 20
}
m.samples++
// …m.avg = mean(history)
```

The glitch guard is the interesting part. A real crystal error is a few PPM —
single-digit Hz on a disciplined USRP, low hundreds on a loose RTL crystal at UHF. A
per-call AFC residual far beyond that isn't drift; it's a measurement glitch — an
unconverged loop, or a data-dependent bias snapshot at end of call. Folding it in
produces exactly the runaway kHz corrections this guard prevents: with a known
channel centre the bound is the PPM-threshold equivalent, and with an unknown centre
(the voice path passes 0) it falls back to `MaxAbsErrorHz` (~1.5 kHz). And because
`total = observed + already-applied`, the tracker measures the source's *true* error
even while a correction is already in effect — it converges rather than oscillating.

If the running average itself exceeds the PPM threshold (3.5 PPM), the Manager logs
a warning: the correction is doing more work than a well-configured dongle should
need, which usually means the configured PPM is wrong and worth fixing at the source.

## Warm-up and application

A single measurement isn't enough to trust, so the correction is gated on warm-up:

```go
// internal/autotune/autotune.go (shape) — Correction
func (m *Manager) Correction() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.samples < WarmupSamples { // 3
        return 0
    }
    return m.avg
}
```

Below `WarmupSamples` measurements, `Correction` returns zero — callers apply
nothing — so one noisy first reading can't yank tuning by hundreds of Hz before the
average has settled. After warm-up the average is the correction, applied as
`targetHz - Correction()`. Managers are per physical dongle, keyed by serial in a
`Registry` (a control SDR and a voice SDR have independent crystal errors), and when
the Registry is disabled `Get` returns nil so every call site collapses to the
pre-autotune behaviour at zero cost. `StatusString` renders both the live correction
and a rounded suggested config value — in Hz and PPM — ready to paste into the device
block. Like autogain, it never rewrites the dongle's hardware PPM; it reports a value
the operator bakes in by hand.

## Where this goes next

The front end is settled: gain chosen against real decode quality, tuning nudged
toward lock. Now the hunt stops probing single candidates and starts *supervising*.
[Part 6]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
introduces the control-channel hunt supervisor — the loop that dwells on candidate
control channels across many systems on one SDR, publishes `HuntProgress`, and backs
off the ones that don't lock so it can keep the ones that do.

## FAQ

**Why sweep gain instead of using the tuner's AGC?**
Hardware AGC optimizes for a full-scale signal, not for decode quality — and on a
busy band it can be driven by a strong neighbour rather than the channel you want.
Sweeping gain against the actual decode error rate optimizes the thing that matters:
whether *this* control channel decodes cleanly. AGC has no idea which carrier you
care about.

**What is `ErrorRate` and why is it the score?**
It's the protocol-neutral count of decode-error events per 1000 symbols
(`CaptureReport.ErrorRate`, from Signal Lab's `DecodeErrorRate`), introduced in
Part 1. Because it's protocol-independent and comes straight from the real decode, it
lets the gain sweep compare gains on a single honest number regardless of whether the
carrier is P25, DMR, or NXDN.

**Does autotune change my dongle's PPM setting?**
No. It computes a digital correction applied by shifting the channel's down-converter
target, and reports a suggested config value (in Hz and PPM) you can bake in by hand.
It never rewrites hardware state — that keeps the correction reversible and auditable.

**Why reject some AFC measurements instead of averaging them?**
A real crystal error is bounded by a few PPM of the channel centre. A residual far
beyond that is a measurement glitch — an unconverged loop or an end-of-call bias
snapshot — and averaging it in drags the applied correction into the kHz. Rejecting
implausible values keeps the running average honest.

**Why per-dongle Managers?**
Each physical dongle has its own crystal with its own error, and a system may use a
separate control SDR and voice SDR. Keying Managers by serial in the `Registry`
means each dongle's correction is tracked independently and never cross-contaminates
another's tuning.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Classifying a Signal — Analog, Digital, Encrypted]({{ '/blog/deep-dives/the-hunt-04-classifying-a-signal/' | relative_url }})
· Next →
[Part 6: Control-Channel Hunting — The Supervisor]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
