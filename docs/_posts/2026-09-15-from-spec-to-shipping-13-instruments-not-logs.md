---
title: "From Spec to Shipping, Part 13: Instruments, Not Logs"
description: How GopherTrunk designs decoder diagnostics as instruments — counters on every branch including the failing ones, verdict lines written for operators, WARNs gated on persistence instead of one sampled blip, and health checks that never judge a channel by absolute dBFS.
category: deep-dives
keywords: designing diagnostics for dsp, decoder status line design, counters vs log messages, warn persistence gate, false warning transients, absolute dbfs trap, sdr health monitoring, structured logging go slog, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, diagnostics, logging, observability, methodology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 13
---

*Part 13 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
proved a fix with a test that fails first — once, on a fixture. This part is
about what the shipped decoder says about itself afterwards, because the next
investigation starts from its output. The rule: build **instruments**, not
logs — counters on every branch including the failing ones, verdict lines an
operator can act on, and WARNs that can tell a persistent condition from a
sampled transient.*

> **TL;DR:** A success-only log line carries no information — count every
> branch. GopherTrunk's DMO decode-status line pairs `dnb_total` (a **noise
> meter**: the burst correlator fires ~18×/s on an idle channel) with
> `dnb_qualified` (slot-grid-confirmed traffic) so a huge gap between them
> reads as *normal*, not alarming. Verdict lines flip their text on operator
> knowledge (`GT_TETRA_DMO_CLEAR=1` turns "encrypted or decode defect" into
> "decode defect — keep chasing"). WARNs carry persistence gates:
> `carrierOffsetWarnPersist` (10 s, `internal/scanner/ccdecoder/decoder.go`)
> ended the fake wrong-site warnings a sub-second AFC alias blip produced,
> and `lowPowerDecodeGrace` (`internal/scanner/widebandt2/engine.go`) stops
> a channel that is *decoding* from being called unhealthy at −56 dBFS. And
> the MRC health line now WARNs when `updates` freezes while `holds` climbs —
> the failure a plain INFO line reported as healthy for eleven minutes.

**Key takeaways**

- **Count every branch, especially the failing ones.** `dsb_total` next to
  `dsb_schs_crc`, `dnb_total` next to `dnb_qualified`, `tch_crc` next to both
  — ratios diagnose; lone success counters flatter.
- **Name what a counter measures, not what you hope it means.** `dnb_total`
  is a noise meter and the docs say so; and when `colour_known` could be true
  without a verified recovery, it earned a separate `colourRecovered` flag —
  a field must not claim more than the code checked.
- **A WARN is a diagnosis, so it must outlast a transient.** The condition
  the #815 carrier WARN names is persistent by nature; the estimate that
  feeds it is not. Ten seconds of continuous excursion separates them.
- **Never gate health on absolute power when decode evidence exists.**
  dBFS is a gain-staging number; CRC-valid decodes are a health verdict.
  Every absolute-power gate is a trap that re-fires on the next front end.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Branch-complete status line | `dsb_total/dsb_schs_crc`, `dnb_total/dnb_qualified`, `tch_crc`, `colour_known` | `internal/scanner/ccdecoder/pipelines_dmo.go` (`maybeLogStatus`) |
| Noise-meter math | ~18 spurious DNB detections/s on an idle channel | `internal/radio/tetra/dmo_grid.go` (doc comment) |
| Operator verdict line | `GT_TETRA_DMO_CLEAR=1` flips encrypted-vs-defect text | `cmd/gophertrunk/tetra_dmo_replay_test.go` |
| WARN persistence gate | offset must sit over threshold 10 s continuously | `decoder.go` (`carrierOffsetWarnPersist`) |
| Decode-evidence grace | recent decodes suppress the low-power WARN | `internal/scanner/widebandt2/engine.go` (`lowPowerDecodeGrace`) |
| Staleness watchdog | WARN when `updates` freezes while `holds` climbs | `internal/sdr/soapyremote/mrc.go` (`mrcStaleUpdateIntervals`) |

## In this post

- **A success-only line carries no information** — the census principle,
  built forward.
- **Anatomy of one status line** — the DMO decode status, field by field.
- **Verdict lines are written for the operator** — text that changes with
  what the operator knows.
- **A WARN must outlast a transient** — the persistence gate.
- **Decode evidence beats absolute dBFS** — the grace that ended a false
  health alarm.
- **The counter that noticed its own silence** — staleness as a first-class
  signal.

## A success-only line carries no information

[From the Issue Tracker Part 21]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
taught this as a postmortem: a pipeline that only announces what it decoded
cannot tell you *why* it isn't decoding. Taught forward, the principle is
about ratios. "Decoded 46 SCH/S" is a number; "46 CRC-valid of 54 detected"
is a diagnosis — the detector works, the channel is mostly clean, the FEC is
earning its keep. Every stage boundary in a decode chain is a place where
work either survives or dies, and an instrument counts **both** outcomes at
every boundary, because the failing count is usually the one that localises
the fault.

The DMO investigation ran on exactly this. The operator's log showed
`dnb_total` climbing 1076→4541 over 185 seconds — 18.7 per second — while
`dsb_schs_crc` sat frozen at 46. One counter racing while its CRC-gated
neighbour stands still is not two facts, it's one conclusion: the raw
detections are noise. That arithmetic, done in
[Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }}),
became the fix; here it becomes the instrument.

## Anatomy of one status line

The DMO pipeline's periodic decode-status DEBUG line is the shape to copy:

```go
// internal/scanner/ccdecoder/pipelines_dmo.go (shape) — maybeLogStatus
p.log.Debug("tetra dmo: decode status",
    "locked", p.locked,
    "carrier_off_hz", math.Round(p.rx.CarrierOffsetHz()*10)/10,
    "dsb_total", p.dsbTotal,       // sync bursts detected
    "dsb_schs_crc", p.dsbCRC,      // …of which CRC-valid — the lock evidence
    "dnb_total", p.dnbTotal,       // raw traffic-burst detections: a NOISE METER
    "dnb_qualified", p.dnbQualified, // …slot-grid-confirmed: the number that means traffic
    "tch_crc", p.tchCRC,           // CRC-valid speech blocks
    "distinct_fn", len(p.fnSeen),  // frame counter advancing = a genuine lock
    "colour", p.colour, "colour_known", p.colourKnown,
    "grant_active", p.grantActive,
)
```

Every field is one side of a ratio or one liveness proof. `dsb_total` against
`dsb_schs_crc` grades the channel; `distinct_fn` proves the lock is a
transmission and not a chance CRC (a frame counter that advances cannot be
faked by noise); and the pair at the centre carries the lesson:

| Field pair | One is… | The other is… | A big gap means… |
|---|---|---|---|
| `dsb_total` / `dsb_schs_crc` | detections | CRC-verified syncs | marginal RF or wrong channel |
| `dnb_total` / `dnb_qualified` | a noise meter (~18/s idle) | slot-grid-confirmed traffic | **nothing — this is normal** |
| `dnb_qualified` / `tch_crc` | confirmed bursts | decoded speech | wrong colour code, or encryption |

That middle row is the design point. The correlator that finds DMO traffic
bursts is deliberately loose — 11 dibits at tolerance 2 under eight matched
filters — so it fires ~18 times a second on an idle channel by construction
(`internal/radio/tetra/dmo_grid.go` derives the number). Logging `dnb_total`
alone would manufacture a permanent false alarm; logging only `dnb_qualified`
would hide whether the correlator is alive at all. Logging **both, with the
gap documented as normal**, turns a would-be scary number into the reference
arm of a measurement — `dmo_grid.go`'s own comment says it in one line: *"a
raw DNB count is not evidence of traffic — it is a noise meter."*
`dnb_qualified` is the number that means traffic, and a large gap between
them is not a fault.

One more field earned its keep the hard way: `colour_known`. An early version
set it `true` even when the recovery's confidence gate had *not* cleared, so
a call that decoded nothing logged `colour=0 colour_known=true` — which reads
as "recovered colour 0" and sends the investigation after the wrong thing.
The flag was split from a separate `colourRecovered`. The rule generalises:
**a diagnostic field must not claim more than the code actually verified**,
because a wrong instrument is worse than no instrument.

<figure class="lab-figure">
<svg viewBox="0 0 680 235" width="680" height="235" role="img" aria-label="One DMO decode-status log line annotated field by field. The line shows locked, dsb totals and CRC counts, dnb total and qualified, tch crc, distinct fn, and colour flags. Callouts group the fields: liveness proof, channel grade ratio, noise meter versus traffic, decode yield, and honest claim flags.">
  <rect x="14" y="92" width="652" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="24" y="104" fill="var(--fg-muted)" font-size="8" font-family="monospace">DEBUG tetra dmo: decode status</text>
  <text x="24" y="116" fill="currentColor" font-size="9" font-family="monospace">locked=true dsb_total=54 dsb_schs_crc=46 dnb_total=4541 dnb_qualified=212 tch_crc=196 distinct_fn=9 colour=3</text>
  <line x1="90" y1="92" x2="70" y2="58" stroke="var(--fg-muted)"/>
  <text x="24" y="48" fill="var(--fg-muted)" font-size="9">liveness: the lock claim…</text>
  <line x1="200" y1="92" x2="190" y2="58" stroke="var(--fg-muted)"/>
  <text x="140" y="34" fill="currentColor" font-size="9">channel grade: 46/54 CRC-valid</text>
  <text x="140" y="48" fill="var(--fg-muted)" font-size="9">detector AND verifier, side by side</text>
  <line x1="360" y1="92" x2="370" y2="58" stroke="var(--accent)"/>
  <text x="310" y="34" fill="var(--accent)" font-size="9">noise meter vs traffic: 4541 raw, 212 qualified</text>
  <text x="310" y="48" fill="var(--accent)" font-size="9">the gap is DOCUMENTED as normal (~18 false/s)</text>
  <line x1="480" y1="122" x2="470" y2="158" stroke="var(--fg-muted)"/>
  <text x="380" y="176" fill="currentColor" font-size="9">yield: qualified bursts → CRC-valid speech</text>
  <line x1="560" y1="122" x2="570" y2="158" stroke="var(--fg-muted)"/>
  <text x="480" y="194" fill="var(--fg-muted)" font-size="9">…proved by an advancing frame counter (distinct_fn)</text>
  <line x1="640" y1="122" x2="620" y2="205" stroke="var(--fg-muted)"/>
  <text x="330" y="216" fill="currentColor" font-size="9">honest claims: colour_known split from colourRecovered — a flag may not claim more than the code verified</text>
</svg>
<figcaption>One line, five instruments: liveness proof, a CRC-graded ratio, a documented noise meter beside its qualified twin, decode yield, and flags that say only what was verified.</figcaption>
</figure>

## Verdict lines are written for the operator

The replay harnesses from
[Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
end in **verdict lines** — prose conclusions computed from the counters,
because the person running the test on their own capture is an operator, not
the author of the decode chain. The DMO harness's verdict is the template
(`cmd/gophertrunk/tetra_dmo_replay_test.go`): when signalling decodes well
but speech sits at the ~1/256 chance floor, the honest conclusion is
genuinely ambiguous — *either* air-interface encryption *or* a remaining
clear-voice decode defect — and the line says exactly that, both branches.

But the operator often knows something the code cannot: whether the radios
are clear. `GT_TETRA_DMO_CLEAR=1` asserts it, and **flips the verdict text**
— the same counters now print "this is a clear-voice DECODE defect to keep
chasing … NOT encryption", with the next suspects named (DNB geometry, the
DM colour code). That flip matters because the un-flipped reading already
burned this project once: the chance floor was misread as encryption for
weeks until the reporter confirmed TEA0-clear radios
([the on-air gate]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})).
A verdict line that encodes *what evidence would change the conclusion* is
the difference between a diagnostic and a verdict you have to argue with.

## A WARN must outlast a transient

A WARN is a diagnosis addressed to a human, and a wrong one has a cost: the
operator goes hunting for a condition that never existed. GopherTrunk's #815
carrier-offset WARN — "your receiver may be locked onto a different site" —
produced exactly that failure. A reporter filmed the mixer panel's carrier
readout spiking to ~−5 kHz on a carrier really ~500 Hz off; the log said
`offset_hz=5004`. The number itself carries the explanation: 5004 = 504 +
4500, and 4500 Hz is exactly one AFC alias bucket (f_sym/4) — a sub-second
estimator transient, not a carrier move. The old code diagnosed a
*persistent* condition from **one per-chunk instantaneous sample**, so every
blip became a scary wrong-site warning.

```go
// internal/scanner/ccdecoder/decoder.go (shape)
// carrierOffsetWarnPersist is how long the total offset must sit
// continuously over the WARN threshold before the first #815 line is
// emitted. The condition the WARN diagnoses — locked onto an adjacent
// site's carrier, or a grossly mistuned oscillator — is persistent by
// nature, but the sampled AFC estimate is not […] 10 s cannot be crossed
// by an estimator transient, while a genuine adjacent-site lock still
// warns shortly after acquisition.
const carrierOffsetWarnPersist = 10 * time.Second
```

The design rule is in that comment: **match the gate to the physics of the
condition, not the cadence of the sampler.** The excursion clock resets when
the offset dips back under threshold and on pipeline teardown, and the
duration is a test-overridable field so
`TestDecoderCarrierOffsetWarnRequiresPersistence` can pin the behaviour
(failing-first, per Part 12). The AFC transient itself got a separate fix in
the estimator — the WARN gate is not a bandage over it but an acknowledgment
that *any* estimator blips, and a diagnosis should never be cheaper to emit
than the condition it names. The full alias-bucket story is in
[TETRA End to End Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }}).

## Decode evidence beats absolute dBFS

The sibling trap: judging health by absolute power. The wideband Tier II
engine used to WARN "channel iq power very low" every 5 seconds — on a
Tier III control channel that was **decoding every C_ALOHA** at −56 dBFS.
Absolute dBFS is a gain-staging number, a function of front end, attenuation
and configuration; it grades the installation, not the decode. Meanwhile the
strongest possible health evidence — protocol decodes with valid CRCs — was
sitting in the same process, ignored.

`lowPowerDecodeGrace` (15 s, `internal/scanner/widebandt2/engine.go`) encodes
the priority: any channel whose decode counters advanced recently is healthy
*whatever its power gauge reads*, with the grace covering inter-beacon gaps
so the WARN doesn't flap. It is the same lesson the MRC calibrator learned
when an operator raised front-end gain 65→82 dB purely to push a number past
a −40 dBFS software constant — chronicled in
[Coherence, Not dBFS]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }}) —
generalised into a design rule: **never gate a health verdict on absolute
power when scale-invariant evidence (decodes, CRC yield, coherence) exists.**
Every absolute-power gate is a trap that re-fires on the next front end.

## The counter that noticed its own silence

The subtlest instrument failure is the line that keeps printing while its
meaning quietly dies. The MRC diversity health line logged `updates`
(accepted calibration windows) and `holds` (rejected ones) every interval —
census-complete, both branches counted. An operator's field log then showed
`updates` **frozen at 6682 for eleven minutes** while `holds` climbed ~6000
per interval… at INFO, reported as healthy, because `calibrated` was still
`true`. Every individual field was accurate. The line as a whole was a lie of
omission: the combiner was applying a gain measured minutes ago to branches
that no longer agreed, and nothing said so.

The fix makes staleness itself a tracked quantity: the reporter
(`internal/sdr/soapyremote/mrc.go`) remembers `lastUpdates` across intervals,
and three consecutive intervals with no accepted window
(`mrcStaleUpdateIntervals`, 90 s) escalates to a WARN naming the real state —
locked once, coasting since. The generalisation: counters catch what single
events miss, but only **derivatives of counters** catch a system that stopped
making progress while still running. An instrument isn't finished when it
reports its state; it's finished when it can report that its state has
stopped changing — designed, like everything here, for the *next*
investigation rather than the last one.

## Where this goes next

Instruments tell you when a fix holds in the field — which is the last
ingredient of the claim this series has been building toward: *the problem is
gone*. [Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }}),
the finale, defines that claim precisely — what "verified" means, the
issue-closing policy that enforces it, the guard hook that asks a human
before any close, and the full shipping checklist from Parts 1–13.

## FAQ

**Why log failure counters when nobody reads them until something breaks?**
Because "until something breaks" is exactly when they're read, and they can't
be added retroactively to a log you already collected. The DMO diagnosis ran
on `dnb_total` racing while `dsb_schs_crc` froze — two counters that were
"noise" right up until they were the whole answer.

**Doesn't logging noise counters like `dnb_total` cause false alarms?**
Only if logged alone. Paired with its qualified twin and documented — the
status line's design says a large gap is normal — it becomes the reference
arm of a measurement. Removing it would be worse: you could no longer tell a
dead correlator from a silent channel.

**How long should a WARN persistence gate be?**
Longer than the longest transient the underlying estimator can produce,
shorter than the time an operator would care about the real condition.
`carrierOffsetWarnPersist` is 10 s because the AFC alias blip lasts well
under a second while an adjacent-site lock lasts until someone fixes it —
any value between those regimes works.

**Should decoder diagnostics live at DEBUG or INFO?**
GopherTrunk's split: periodic status lines at DEBUG (they're for
investigations), state *transitions* at INFO, and conditions needing operator
action at WARN — with the low-power WARN firing regardless of level because
operator feedback is its whole point. The failure mode to avoid is a steady
state logging identically forever; the Tier II engine moved parked channels
to a 30 s summary cadence after a field report of "endless spam".

**What's the fastest way to audit an existing log line?**
Ask, for each field: what ratio is this one side of, and could it be true
without the thing it implies being true? The second question is what caught
`colour_known` — true without a verified recovery — and it generalises to
any flag whose name promises more than its assignment checks.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Failing First — The Regression Rule]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
· Next →
[Part 14: The Definition of Verified]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }})
