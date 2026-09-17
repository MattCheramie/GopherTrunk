---
title: "The Field Notebook, Part 4: The Carrier-Offset WARN — Alias Buckets & Persistence"
description: An operator's reading of GopherTrunk's issue #815 carrier-offset WARN — why offset_hz=5004 decomposes as 504 plus one 4500 Hz alias bucket, why the WARN now requires ten continuous seconds of excursion before it fires, when it really means a wrong-site lock, and the mixer-panel jitter behind the "AFC spikes to −5 kHz" report.
category: tutorials
keywords: carrier offset warn, issue 815 wrong site, offset_hz 5004, afc alias bucket 4500 hz, adjacent channel lock, carrier_offset_warn_hz, afc spikes to -5 khz, mixer panel jitter, read debug log, gophertrunk field notebook
tags: [field-notebook, logs, afc, carrier-offset, tetra, troubleshooting, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 4
---

*Part 4 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time.
[Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
ended on the field that gives the locked-but-deaf state away: `carrier_off_hz`
sitting one alias bucket off centre. This part reads the WARN that surfaces
it to an operator — `ccdecoder: control carrier offset far from configured
frequency` (issue [#815](https://github.com/MattCheramie/GopherTrunk/issues/815))
— and the arithmetic that decides whether it means "you locked the wrong
site" or "your AFC blipped for 200 ms". The difference is a persistence gate,
and getting it wrong sends you chasing a condition that never existed.*

> **TL;DR:** `ccdecoder: control carrier offset far from configured
> frequency … (issue #815)` fires with `offset_hz= freq_hz= system=` when the
> **total** offset (autotune correction + the receiver's residual
> `AFCOffsetHz`) exceeds `carrierOffsetWarnHz` (`defaultCarrierOffsetWarnHz`
> = 4000, override `carrier_offset_warn_hz`) while locked. A field report
> filmed the mixer panel spiking to ~−5 kHz on a carrier really ~500 Hz off,
> logged `offset_hz=5004` — and 5004 = 504 + 4500, where 4500 Hz is exactly
> one f_sym/4 AFC alias bucket. The old per-chunk check turned every
> sub-second alias blip into a scary wrong-site WARN. Now `checkCarrierOffsetLocked`
> requires the offset to sit over threshold for `carrierOffsetWarnPersist`
> (10 s continuous), throttled to `carrierOffsetWarnInterval` (30 s), reset on
> any dip or teardown — pinned by `TestDecoderCarrierOffsetWarnRequiresPersistence`.
> A genuine adjacent-site lock outlasts 10 s and still warns; the transient
> can't. The AFC fix that ended the false spikes lives in
> `internal/radio/tetra/receiver/afc.go` (`omegaReprimeStreakEstablished`).

**Key takeaways**

- **A number that is an exact multiple of 4500 Hz off is an alias, not a
  move.** `offset_hz=5004` = 504 + 4500. TETRA's f_sym/4 = 4500 Hz is the
  alias spacing; a residual near it is a wrong AFC bucket, not a mistuned
  crystal.
- **The WARN is a diagnosis, so it must outlast a transient.** Ten seconds
  of continuous excursion (`carrierOffsetWarnPersist`) separate a persistent
  wrong-site lock from a sub-second AFC spike; the old per-chunk check did not.
- **When it really fires, verify the site identity.** A sustained offset near
  the channel spacing means GopherTrunk may be decoding a neighbour's stronger
  carrier under your configured frequency — check `nac`/`mcc` against the site.
- **The WARN never retunes.** It is advisory only. It tells you the locked
  carrier sits far from where you pointed it; the fix is `ppm`, `autotune`, or
  a correct frequency, never the WARN itself.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The WARN | advisory wrong-site / mistune hint | `internal/scanner/ccdecoder/decoder.go` (`checkCarrierOffsetLocked`) |
| Threshold | total offset above which it can fire | `defaultCarrierOffsetWarnHz` = 4000, `carrier_offset_warn_hz` |
| Persistence gate | 10 s continuous before the first line | `carrierOffsetWarnPersist`, `TestDecoderCarrierOffsetWarnRequiresPersistence` |
| Repeat throttle | one line per 30 s while sustained | `carrierOffsetWarnInterval` |
| The offset itself | autotune applied + receiver residual | `AFCOffsetHz` (`afcReporter`), `autotuneApplied` |
| Alias spacing | ±f_sym/4 = 4500 Hz for TETRA | `internal/radio/tetra/receiver/afc.go`, [AFC alias traps]({{ '/reference/afc-alias-traps/' | relative_url }}) |
| The AFC fix | established track holds an alias transient | `afc.go` (`omegaReprimeStreakEstablished`, `aliasClassOffset`) |
| The panel view | raw vs tuned carrier marker | [mixer panel]({{ '/mixer.html' | relative_url }}) |

## In this post

- **What this WARN is telling you** — total offset, and the two things it means.
- **What healthy looks like** — the WARN silent, `carrier_off_hz` small.
- **Reading `offset_hz=5004`** — the alias-bucket arithmetic.
- **The persistence gate** — why one blip no longer warns.
- **When it doesn't look like that** — symptom → cause → where to read.

## What this WARN is telling you

The #815 WARN answers one question: *is the carrier I locked actually the one
I configured?* `checkCarrierOffsetLocked` runs while the CC is locked and
computes the **total** offset — the autotune correction already applied
(`autotuneApplied`) plus the receiver's live residual (`AFCOffsetHz`, the
`afcReporter` capability the decoder type-asserts). With autotune off (the
default) the applied term is 0 and the total is the raw residual; with
autotune on, a large genuine offset is folded into the applied term, so
summing the two still catches an adjacent-channel lock that autotune would
otherwise hide by chasing the wrong carrier. If that total exceeds
`carrierOffsetWarnHz` (4000 Hz by default), the WARN can fire:

```
WRN ccdecoder: control carrier offset far from configured frequency — GT may be locked onto an adjacent site's stronger carrier (wrong site identity) or the receiver oscillator is badly mistuned; verify the reported site against the configured frequency (issue #815) offset_hz=12500 freq_hz=857262500 system=Metro-P25
```

The 4000 Hz default sits in a deliberate gap: a good front end (Airspy/TCXO)
after autotune leaves a sub-kHz residual and never trips it; the P25
6.25/12.5 kHz adjacent-channel steps all clear it, so an adjacent-site lock
always trips; and a reasonably-corrected dongle's 1–3 kHz crystal error stays
under it. The WARN names two causes because it cannot tell them apart from the
offset alone: a stronger neighbour bleeding through the passband (the wrong
site — [adjacent-channel lock]({{ '/reference/carrier-offset-adjacent-lock/' | relative_url }}))
or a badly mistuned oscillator. It **never retunes** — it is advisory, and the
fix is `ppm`, `sdr.autotune`, or a correct control frequency.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="A frequency axis centred on zero, the configured control channel. Two markers sit off centre: one near plus 500 Hz labelled the true residual, and one near plus 5000 Hz labelled offset_hz=5004, which decomposes as 504 plus one 4500 Hz alias bucket. The 4500 Hz f_sym over 4 alias spacing is drawn as a bracket. Below, two timelines: a sub-second spike that dips back under threshold before the 10 second persistence window elapses and does not warn, and a sustained excursion that stays over threshold past 10 seconds and warns once, then again after the 30 second throttle.">
  <line x1="40" y1="56" x2="640" y2="56" stroke="var(--fg-muted)"/>
  <line x1="120" y1="50" x2="120" y2="62" stroke="currentColor"/>
  <text x="120" y="44" text-anchor="middle" fill="currentColor" font-size="9">0 Hz (configured)</text>
  <circle cx="150" cy="56" r="4" fill="currentColor"/>
  <text x="150" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+504 true</text>
  <circle cx="420" cy="56" r="4" fill="var(--accent)"/>
  <text x="452" y="44" text-anchor="middle" fill="var(--accent)" font-size="9">offset_hz=5004</text>
  <path d="M150 88 L420 88" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="285" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+4500 Hz = one f_sym/4 alias bucket</text>
  <line x1="40" y1="150" x2="640" y2="150" stroke="var(--fg-muted)"/>
  <text x="14" y="140" fill="var(--fg-muted)" font-size="9">blip</text>
  <path d="M60 150 L120 150 L140 120 L160 150 L640 150" fill="none" stroke="currentColor"/>
  <line x1="40" y1="132" x2="640" y2="132" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="360" y="146" text-anchor="middle" fill="currentColor" font-size="9">dips under threshold before 10 s → NO warn (clock reset)</text>
  <line x1="40" y1="220" x2="640" y2="220" stroke="var(--fg-muted)"/>
  <text x="14" y="210" fill="var(--accent)" font-size="9">real</text>
  <path d="M60 220 L150 220 L170 196 L640 196" fill="none" stroke="var(--accent)"/>
  <line x1="40" y1="202" x2="640" y2="202" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <circle cx="470" cy="196" r="3" fill="var(--accent)"/>
  <text x="470" y="188" text-anchor="middle" fill="var(--accent)" font-size="8">warn @ +10 s</text>
  <circle cx="640" cy="196" r="3" fill="var(--accent)"/>
  <text x="600" y="214" text-anchor="middle" fill="var(--fg-muted)" font-size="8">again @ +30 s</text>
</svg>
<figcaption>An offset that is an exact multiple of 4500 Hz off centre is an AFC alias, not a carrier move. The persistence gate lets a real, sustained excursion warn after 10 s while a sub-second blip resets the clock and stays silent.</figcaption>
</figure>

## What healthy looks like

A healthy locked control channel simply never prints the #815 line. The
evidence is on the [decode-status line]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
from Part 3, whose `carrier_off_hz` field is the same residual the WARN sums:

```
DBG tetra: decode status system=250_013 locked=true carrier_off_hz=-412.5 … sch_pdus=47 …
```

`carrier_off_hz=-412.5` is well under the 4000 Hz threshold, `sch_pdus` is
climbing, and the WARN stays quiet — the correct state. On the web side, the
[mixer panel]({{ '/mixer.html' | relative_url }}) shows the same thing
visually: a **raw peak** off-centre with the **tuned peak centred** is a
healthy lock, the amber carrier-offset marker flagging where the loop found
the carrier. A large *steady* raw offset there is your tuner's ppm error (a
420 MHz / 50 ppm RTL-SDR can sit ~20 kHz off), which the tuned view recentres
once the loop locks.

| Field / view | Measures | Healthy | Worry when |
|---|---|---|---|
| `offset_hz` (WARN) | total offset when the line fires | line absent | present, and near the channel spacing |
| `carrier_off_hz` (status) | the AFC residual | within a few hundred Hz | near ±4500 or a multiple — alias |
| `freq_hz` (WARN) | the configured control frequency | matches your config | fine — it's the reference, not the fault |
| mixer tuned peak | the re-mixed carrier | centred on 0 Hz | still off-centre — loop not acquired |
| WARN cadence | how often it repeats | never | every 30 s — a sustained excursion |

## Reading `offset_hz=5004`

The arithmetic is the whole diagnosis. A TETRA field report — filmed, of the
mixer panel's carrier readout jittering ±400 Hz then spiking to ~−5 kHz on a
carrier really ~500 Hz off — logged `offset_hz=5004`. Decompose it:
**5004 = 504 + 4500**, and 4500 Hz is exactly f_sym/4 for TETRA's 18000 sym/s.
The receiver's carrier estimate is a coarse spectral-centroid stage that
picks the alias bucket, plus a fine 4×Δφ estimator that is unambiguous only
within ±f_sym/8 = ±2250 Hz. Any transient coarse bias past that wrap — a
neighbour's skirts through the channel-filter edge while the wanted carrier
fades — lands the raw estimate exactly ±4500 Hz off. So a residual near
504 Hz becomes a reported 5004 Hz for a couple of hundred milliseconds.

That is the same alias structure behind Part 3's locked-but-deaf state, seen
from the other side: there the wrong bucket persists and kills `sch_pdus`;
here it is a sub-second transient on an otherwise-healthy lock. The AFC fix
that stopped the false spikes is in `afc.go`: `track` counts accepted blocks
since the last prime (`accSincePrime`, capped at `omegaEstablishedBlocks`
= 16), and on an *established* track an alias-class reject cluster (cluster
mean congruent to the EMA modulo π/2 — `aliasClassOffset`) must persist
`omegaReprimeStreakEstablished` = 18 blocks (~1 s) before it re-primes, while
a fresh track keeps the fast 3-block escape (`omegaReprimeStreak`). Duration
disambiguates: a coarse-bias transient passes in well under a second; a
carrier that genuinely stepped a bucket does not. `TestAFCEstablishedTrackHoldsThroughAliasTransient`
pins that a long-corroborated EMA holds through the ~340 ms transient the old
code jumped on.

## The persistence gate

The AFC fix stopped the *estimate* from latching, but the WARN itself had the
matching bug: it diagnosed a *persistent* condition — a wrong-site lock —
from **one per-chunk instantaneous sample**, so every alias blip became a
scary wrong-site line. The fix is a persistence gate. `checkCarrierOffsetLocked`
now starts an excursion clock when the total first crosses threshold and
emits nothing until it has stayed over for `carrierOffsetWarnPersist`
(10 s continuous); any dip back under resets the clock, and pipeline teardown
clears it so a stale excursion from a torn-down lock can't warn instantly on
the next acquisition:

```go
// internal/scanner/ccdecoder/decoder.go (shape) — checkCarrierOffsetLocked
// 10 s cannot be crossed by an estimator transient (offset_hz=5004 lasts
// well under a second), while a genuine adjacent-site lock still warns
// shortly after acquisition.
if total < d.carrierOffsetWarnHz {
    d.offsetOverWarnSince = time.Time{} // dipped under — reset the clock
    return
}
if d.offsetOverWarnSince.IsZero() {
    d.offsetOverWarnSince = now
}
if now.Sub(d.offsetOverWarnSince) < d.carrierOffsetWarnPersist {
    return // over threshold, but not long enough yet
}
```

Once it does fire, `carrierOffsetWarnInterval` (30 s) throttles the repeat so
a genuinely stuck adjacent-channel lock logs steadily rather than flooding.
The design rule, from [From Spec to Shipping Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }}):
**match the gate to the physics of the condition, not the cadence of the
sampler** — the condition (wrong site, or a badly mistuned oscillator) is
persistent, so the gate is too. `TestDecoderCarrierOffsetWarnRequiresPersistence`
pins it: a `fakeAFCPipeline` at `offsetHz=5004` warns zero times across a
burst of checks, and a sustained `offsetHz=12500` warns exactly once after
the window elapses. The companion `checkPayloadResync` from Part 3 bounds the
worst case — if a real wrong-bucket latch does hold, the payload drought
forces a re-hunt within seconds regardless of the WARN.

### How this line shapes operator practice

- **Decompose the number before you act.** An `offset_hz` that is a small
  residual plus a multiple of 4500 Hz is an AFC alias, not a wrong site — do
  not retune to chase it.
- **A single #815 line no longer exists.** The WARN needs 10 s of continuous
  excursion, so if you see it, the excursion is real — verify the site
  identity (`nac`/`mcc`) against your configured frequency.
- **Raise `carrier_offset_warn_hz` only for a genuinely high-drift dongle.**
  The 4000 Hz default clears a corrected crystal's 1–3 kHz error and every
  P25 adjacent step; raising it hides real adjacent-site locks.
- **The mixer panel is the visual twin.** A raw peak off-centre with the
  tuned peak centred is a healthy lock; a tuned peak still off-centre is a
  loop that hasn't acquired.

## Where this goes next

TETRA's decode chain and its AFC are behind us; the next protocol family
speaks a different status line. [Part 5]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }})
reads the DMR activity line from the wideband engine — `sync_hits`, `dibits`,
`rekeys`, `late_entries`, `csbk_crc_fail`, `beacons`, `deaf_heals`, and the
`mm_mu`/`mm_sps`/`agc_level`/`coarse_offset_hz` a deaf-heal prints — where a
raw detection is a noise meter and a qualified one is traffic, the same
lesson the DMO line will hammer home in Part 6.

## FAQ

**What does "control carrier offset far from configured frequency" mean?**
GopherTrunk's total carrier offset (autotune correction plus the receiver's
residual) exceeded `carrier_offset_warn_hz` (4000 Hz default) while locked.
It means you may be decoding an adjacent site's stronger carrier under your
configured frequency, or the oscillator is badly mistuned. It is advisory —
it never retunes; verify the reported site identity.

**Why did offset_hz read 5004 when my carrier is only ~500 Hz off?**
Because 5004 = 504 + 4500, and 4500 Hz is one f_sym/4 AFC alias bucket for
TETRA. A transient coarse-frequency bias wrapped the estimate one bucket off
for a fraction of a second. It is an aliasing artefact of the 4th-power
estimator, not a real carrier move — see [AFC alias traps]({{ '/reference/afc-alias-traps/' | relative_url }}).

**Why doesn't the #815 WARN fire on a brief AFC spike anymore?**
Because it now requires the offset to sit over threshold for 10 continuous
seconds (`carrierOffsetWarnPersist`) before the first line. A sub-second alias
blip dips back under and resets the excursion clock, so it stays silent, while
a genuine adjacent-site lock outlasts 10 s and still warns. The gate matches
the physics: the condition is persistent, the estimate is not.

**Should I raise carrier_offset_warn_hz to stop the warning?**
Only if your dongle has a genuinely large, legitimate crystal error that the
default 4000 Hz would trip. Raising it also hides real adjacent-site locks and
gross mistunes. The better fix for a persistent offset is a correct `ppm`, or
`sdr.autotune: true` to track it — the WARN is telling you the carrier is off,
not that the threshold is wrong.

**Is the WARN the same thing as the mixer panel's amber marker?**
They read the same loop state. The amber marker on the [mixer panel]({{ '/mixer.html' | relative_url }})
flags the carrier-offset estimate visually; the #815 WARN fires when that
estimate's magnitude stays large for 10 s. A raw peak off-centre with the
tuned peak centred is the healthy case the marker shows and the WARN stays
silent on.

## Series navigation

**Part 4 of 14** · ←
[Part 3: The TETRA Decode-Status Line]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
· Next →
[Part 5: The DMR Activity Line]({{ '/blog/tutorials/field-notebook-05-dmr-activity-line/' | relative_url }})
