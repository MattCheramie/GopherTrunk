---
title: "The Field Notebook, Part 8: The MRC Health Line"
description: "How to read GopherTrunk's soapyremote MRC diversity branches health line — branch_dbfs, coherence, branch_gain_db, branch_phase_deg, updates and holds, lock_gate — the anchor and inter-branch delay lines around it, the four WARNs and how their text history changed, and which Weak-Signal or Analog Edge post to open when a two-antenna rig stops earning its keep."
category: tutorials
keywords: mrc diversity health line, sdr coherence log, branch_phase_deg, updates holds mrc, lock_gate coherence, inter-branch delay samples, diversity branches not coherent warn, stale calibration warn, gophertrunk mrc health line
tags: [field-notebook, logs, diversity, mrc, soapyremote, diagnostics, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 8
---

*Part 8 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's debug.log one line family at a time — what each field measures,
what a healthy rig prints, which number is a noise meter and which one means
traffic, and which deep dive to open when a line goes wrong.
[Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
read the overrun family, whose absence is the good news. This part reads a
line that prints every 30 s and has to be *read* to know whether a second
antenna is helping at all: the **MRC diversity health line**, the anchor
and delay lines before it, and four WARNs whose wording carries the history
of every wrong gate this feature has had.*

> **TL;DR:** `soapyremote: MRC diversity branches` (`internal/sdr/soapyremote/mrc.go`,
> `diversityReporter.observe`, INFO every 30 s) prints `branch_dbfs`,
> `reference_branch`, `calibrated`, `coherence`, `branch_gain_db`,
> `branch_phase_deg`, `mode`, `updates`, `holds`. Read three things:
> `coherence` (gain-independent — never raise RF gain to move it), the
> **derivative** of `branch_phase_deg` (constant ⇒ shared LO, walking ⇒
> independent PLLs, tracking `mrc`), and the **derivative** of `updates`
> (frozen while `holds` climbs ⇒ coasting on a stale gain, WARN after
> `mrcStaleUpdateIntervals` = 3). The `not coherent` WARN carries the
> effective `lock_gate` — ≈0.11 for a 4096-sample window, a phase-error
> bound, not a fixed 0.5 — and the one-shot `MRC inter-branch delay
> measured … delay_samples=` line removed a 2.60-sample skew that cost 22%
> of decodes. Tracking-as-default and a real gain *over* the best branch
> remain unverified on air — every capture so far decodes at its ceiling.

**Key takeaways**

- **`coherence` is the verdict; `branch_dbfs` is gain staging.** |rho| does
  not move when you turn a gain knob. An 18 Aug rig decoded 1425 CRC-clean
  BSCH at wideband |rho| 0.16 — low coherence with fine decode is bandwidth
  dilution.
- **Read `branch_phase_deg` and `updates` as derivatives.** One line tells
  you nothing; two tell you the hardware class and whether the calibrator is
  alive.
- **Anchor and delay lines are once-per-stream events.** At every stream
  start and retune they are normal; an anchor *moving* mid-stream is a
  discontinuity worth noting.
- **The WARN text has been wrong before.** "Raising RF gain will NOT help"
  was exactly wrong for a floor-limited branch; today's text sends you to
  *that branch's* gain staging.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Health line | 30 s INFO with both branches' state | `mrc.go` (`diversityReporter.observe`, `mrcHealthInterval`) |
| Staleness WARN | `updates` frozen for 3 intervals (90 s) | `mrc.go` (`mrcStaleUpdateIntervals`) |
| Coherence gates | bound the estimate's phase error, not |rho| | `internal/dsp/diversity/tracking.go` (`LockPhaseSigmaRad` 0.10, `TrackPhaseSigmaRad` 0.16) |
| Anchor | latched on the first live branch, frozen once calibrated | `mrc.go` (`selectReference`) |
| Inter-branch aligner | ±16-lag scan over 2^17 samples, latch at |rho| ≥ 0.1 | `internal/sdr/soapyremote/align.go` (`branchAligner`) |
| Dead-branch WARN | > 20 dB below the reference | `mrc.go` (`mrcBranchDeadMarginDb`) |
| The background | why each gate is shaped this way | [Weak-Signal 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }}), [11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }}) |

## In this post

- **What this line is telling you** — the fields and their derivatives.
- **What healthy looks like** — the 29 Aug X310 session.
- **Field by field** — measures, healthy, worry-when.
- **When it doesn't look like that** — frozen updates, diluted coherence,
  the skew, the anchor flip; then symptom → cause → read.

## What this line is telling you

```go
// internal/sdr/soapyremote/mrc.go (shape) — diversityReporter.observe
attrs := []any{
    "addr", r.addr,
    "branch_dbfs", formatBranchPowers(h.powDbFS), // "ch0=-51.2 ch1=-50.6"
    "reference_branch", h.refIdx,                 // the phase anchor
    "calibrated", h.calibrated,
    "coherence", round2(h.coherence),             // |rho| of the last window
    "branch_gain_db", round2(h.gainDb),           // applied branch-1 gain vs reference
    "branch_phase_deg", round2(h.phaseDeg),       // constant: shared LO; walking: PLLs
    "mode", h.mode,                                // mrc | mrc-static
    "updates", h.updates,                          // accepted calibration windows
    "holds", h.holds,                              // rejected windows (gain held)
}
```

The combiner estimates one complex gain per calibration window
(`mrcCalWindowMs` = 2 ms, clamped to ≥ 4096 samples — so 4096 at 200 kS/s,
a 16.4 ms window) from DC-removed cross-statistics
([coherence]({{ '/reference/coherence/' | relative_url }}) is their
normalised cross-correlation). A window is *accepted* when the projected
phase error of its estimate, `sqrt((1−ρ²)/(2Nρ²))`, is bounded — 0.10 rad
to lock, 0.16 rad to keep tracking — which puts the minimum |rho| at
`1/sqrt(1+2Nσ²)`: ≈0.11 for N = 4096. An accepted window increments
`updates` and smooths the gain (τ ≈ 200 ms in `mrc`; snapped once in
`mrc-static`); a rejected window increments `holds` and **keeps** the
previous gain, never dropping to passthrough — falling back mid-stream is
itself a phase step ahead of a differential decoder.

So the line is read as derivatives. `branch_phase_deg` constant across lines
means a shared-LO front end; walking (a TwinRX pair measured −0.22°/s on
17 Aug, −0.11°/s on 29 Aug) means independent PLLs. `updates` climbing with
`holds` near zero means the calibrator is alive; frozen, it is coasting.

## What healthy looks like

The 29 Aug X310 session (`rx_subdev_spec=B:0 A:0`, TETRA CC 467.9125 MHz,
200 kS/s), after the operator fixed the weak feedline the 19 Aug antenna
swap had exposed:

```
INF soapyremote: MRC phase anchor latched reference_branch=1 branch_dbfs="ch0=-51.4 ch1=-50.9"
INF soapyremote: MRC inter-branch delay measured — delaying the early branch to align the combine delay_samples=0.41 peak_rho=0.94
INF soapyremote: MRC diversity branches addr=192.168.1.60:55132 branch_dbfs="ch0=-51.2 ch1=-50.6" reference_branch=1 calibrated=true coherence=0.95 branch_gain_db=-0.8 branch_phase_deg=131.6 mode=mrc updates=1830 holds=0
```

Branches balanced within ±1.4 dB, `coherence` 0.95–0.96 every interval,
`holds=0`, `updates` climbing, the phase walking a few degrees per 30 s.
The delay line reads **0.41 samples** where the same rig measured 2.60 on
19 Aug — a per-stream start skew, hence re-measured at every stream and
retune. The offline A/B on this session's 60 s pre-combine capture scored
every combined arm within one CRC-clean BSCH of the best branch (2626) — no
harm — and narrowband coherence (0.958) ≈ wideband (0.945), so the wideband
scalar is not the bottleneck here.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two panels of the MRC health line over time. Left: the healthy 29 August session, with updates climbing every interval, holds flat at zero, coherence near 0.95 and branch phase walking slowly. Right: the 17 August failure, with updates frozen at 6682 while holds climb by about six thousand per interval and calibrated still reads true; a marker shows where the staleness WARN now fires after three intervals.">
  <text x="170" y="20" text-anchor="middle" fill="currentColor" font-size="10">29 Aug — healthy</text>
  <line x1="30" y1="180" x2="310" y2="180" stroke="var(--fg-muted)"/>
  <polyline points="40,170 100,140 160,110 220,80 280,50" fill="none" stroke="currentColor"/>
  <text x="200" y="60" fill="currentColor" font-size="8">updates climbing every interval</text>
  <polyline points="40,176 280,176" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="44" y="194" fill="var(--fg-muted)" font-size="8">holds = 0 · coherence 0.95–0.96 · phase −0.11°/s</text>
  <text x="510" y="20" text-anchor="middle" fill="currentColor" font-size="10">17 Aug — frozen</text>
  <line x1="370" y1="180" x2="650" y2="180" stroke="var(--fg-muted)"/>
  <polyline points="380,120 420,100 440,100 640,100" fill="none" stroke="currentColor"/>
  <text x="470" y="94" fill="currentColor" font-size="8">updates frozen at 6682 for eleven minutes</text>
  <polyline points="440,176 500,150 560,124 620,98" fill="none" stroke="var(--accent)"/>
  <text x="520" y="150" fill="var(--accent)" font-size="8">holds +~6000 per interval</text>
  <line x1="530" y1="60" x2="530" y2="180" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="534" y="72" fill="var(--accent)" font-size="8">3 intervals ⇒ stale WARN</text>
  <text x="510" y="194" fill="var(--fg-muted)" font-size="8">calibrated=true the whole time — the INFO line called it healthy</text>
  <text x="340" y="228" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a counter that stops moving is a signal; the reporter now tracks the derivative of updates</text>
</svg>
<figcaption>Same line, two rigs: on the healthy session updates climb and holds stay zero; on the 17 Aug log every field was accurate and the line as a whole was a lie of omission until staleness became a tracked quantity.</figcaption>
</figure>

## Field by field

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `branch_dbfs` | per-branch mean power | both alive, within a few dB | one ~10 dB down — weak antenna/feedline; MRC on a floor-limited branch is ~no gain |
| `calibrated` | a gain is applied | `true` within seconds | `false` with `updates+holds>0` — no window ever passed the gate |
| `coherence` | \|rho\| of the last window, gain-independent | 0.9+ at moderate bandwidth; lower on wide captures | at the `lock_gate` with decode fine — dilution; stuck 0.3–0.5 with `holds` climbing — separated antennas |
| `branch_gain_db` | applied branch-1 gain vs reference (the MRC weight, not a power ratio) | within a few dB | drifting to −30 dB with `calibrated=true` — the pre-clamp random walk |
| `branch_phase_deg` | applied phase vs reference | constant, or walking slowly | jumps of tens of degrees — an anchor change or re-lock |
| `updates` / `holds` | accepted / rejected windows | `updates` climbing, `holds` ≈ 0 | `updates` frozen while `holds` climbs — coasting |
| `lock_gate` (WARN only) | effective minimum \|rho\| for a first lock | ≈0.11 at 4096-sample windows | the WARN fires *at all* with decode fine — check per-branch gain staging |

**A live and an offline reading of the same capture disagree by design**:
the 17 Aug log said coherence 0.28–0.55 and `branch_gain_db` −11..−17 where
the harness measured 0.66 and a −6.4 dB power ratio — the per-window
least-squares estimate is biased low by noise, and the logged gain is the
MRC weight (∝ ρ·√(P₀/P₁)), not a power ratio. And the tracking coefficient
once derived its loop constant from the *nominal* 2 ms window while the
window was clamped to 4096 samples, so at 250 kHz the 1/e time was silently
~1.6 s instead of 200 ms; `mrcTrackAlpha` now follows the actual window.

## When it doesn't look like that

Four field logs, four different lines.

**17 Aug — the counter that stopped.** Every field accurate, the line a lie
of omission:

```
INF soapyremote: MRC diversity branches addr=192.168.1.60:55132 branch_dbfs="ch0=-52.1 ch1=-50.3" reference_branch=1 calibrated=true coherence=0.31 branch_gain_db=-14.2 branch_phase_deg=146.7 mode=mrc updates=6682 holds=47311
```

`updates` frozen at 6682 for eleven minutes while `holds` climbed ~6000 per
interval, at INFO, because `calibrated` was still `true` — a gain measured
minutes ago applied to branches that no longer agreed. Today the reporter
remembers `lastUpdates`, and after three intervals with no accepted window
it prints the WARN instead:

```
WRN soapyremote: MRC diversity has not accepted a calibration window in 90s — the combine is running on a stale gain because every window since has failed the coherence gate. Two receivers whose coherence stays low across a wide span is the wideband-combine limitation: one complex gain cannot align every carrier at once when the antennas are metres apart. Co-locate them, or try diversity: mrc-static to freeze the estimate deliberately. … stale_intervals=3
```

**18 Aug — the gate that gained.** Three 30 s pre-combine captures at
200/250 kS/s: the CC decoded **1425** CRC-clean BSCH off branch 1 while
wideband |rho| sat at ~0.16 and MRC never calibrated — `updates=0` — until
+5 dB of RF gain pushed the number past a fixed 0.50 constant. Wideband
|rho| is diluted by every hertz of noise-only bandwidth around the carrier
(`rho_wb ≈ rho_ch·sqrt(f0·f1)`), so the constant made calibration depend on
capture bandwidth and each branch's noise floor. Worse, that build's WARN
said raising RF gain would *not* help — exactly wrong for a branch 9 dB
down with a gain-independent floor, whose in-channel fraction +5 dB lifted
from 0.14 to 0.37. The gates now bound the estimate's phase error, and the
text sends you to *that branch's* gain staging:

```
WRN soapyremote: MRC diversity branches are not coherent — the two receivers are not seeing the same signal through a constant complex gain, so there is no diversity gain and the reference branch is being passed through. … If one branch_dbfs sits far below the other, that branch may be buried under its own front-end noise floor — fix ITS gain staging (a per-branch gain or attenuator), not the overall level. … lock_gate=0.11
```

Post-fix, all three captures calibrate and `wb-tracking` matches the best
branch within one BSCH (1424/1425, 401/402, 1591/1591).

**19 Aug — the skew and the anchor.** Branch 0 lagged branch 1 by a
constant **2.60 samples** (13 µs at 200 kS/s): per-frequency coherence 0.99
against broadband ρ diluted to 0.78 is the signature of a pure delay, which
a scalar gain cannot represent — the skewed combine decoded 22% *fewer* BSCH
than the best branch alone (886 vs 1142); aligned, exactly 1142. That is
the `inter-branch delay measured` line's reason to exist. The same log had
the anchor flip on a cc-hunt retune and the applied gain random-walk to
−34 dB with `calibrated=true`; now the anchor survives rearm and
same-frequency retunes, every anchor change is logged (`MRC phase anchor
moved off a dead reference — expect one decode discontinuity`), and
`clampMagnitude` floors the applied gain.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| `updates` frozen, `holds` climbing, stale WARN | separated antennas — one wideband gain can't align every carrier | co-locate, or `mrc-static`; [MRC gotchas]({{ '/reference/mrc-diversity-gotchas/' | relative_url }}), [Weak-Signal 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }}) |
| `not coherent` WARN, decode fine, one branch far down | that branch is floor-limited — dilution | raise *its* gain; [Coherence, Not dBFS]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }}) |
| `branch is dead` WARN | > 20 dB down: antenna, feedline, per-channel gain | `antenna:` selects the port; [Analog Edge 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }}) |
| `got 1 of 2 channels` WARN | the server never delivered RX1 | `rx_subdev_spec`; [Cookbook 12]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }}) |
| combine decodes worse than one branch | pre-aligner skew (fixed) or a branch dragging the estimate | run the pre-combine A/B; [Weak-Signal 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }}) |
| tempted to raise gain to "engage" diversity | the absolute-power reflex | don't — `coherence` is gain-independent; [dBFS]({{ '/reference/dbfs/' | relative_url }}) |

### How this line shapes operator practice

- **Copy two lines, never one.** Hardware class and calibrator liveness are
  both derivatives; one health line shows neither.
- **`branch_dbfs` is the only field a gain knob should move.** If turning
  gain changes `coherence`, a branch was floor-limited — fix that branch, not
  the overall level.
- **Expect the anchor and delay lines at every stream start.** They are
  once-per-stream measurements, not faults; an anchor line *mid*-stream is.
- **The A/B is the verdict, scored by decode yield.** `diversity_capture`
  plus `TestDiversityCombinerReplay` — never EVM, never dBFS. A real gain
  over the best branch still needs a weak-signal capture.

## Where this goes next

Every MRC verdict above rested on a pre-combine capture lined up against a
log. [Part 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})
reads the capture lines themselves — `siglab: capture started/ended/aborted`,
`requested_center_hz`, the FLAC sample-rate refusal — and how the 12 Sep
IPSC log was aligned to its 20-minute capture.

## FAQ

**What does `coherence` mean in the GopherTrunk MRC health line?**
The normalised cross-correlation |rho| between the two branches over the
last calibration window, DC-removed. It is gain-independent by construction:
raising RF gain does not move it. Healthy co-located branches at moderate
bandwidth read 0.9+; wide captures read lower because noise-only bandwidth
dilutes it.

**Why are `updates` frozen while `holds` climbs?**
Every window since the last accepted one failed the coherence gate, so the
combine is coasting on a gain measured minutes ago; after three health
intervals GopherTrunk WARNs. The usual cause is antennas metres apart — one
wideband gain cannot align every carrier — or a branch buried in its own
noise floor.

**What is `lock_gate` in the not-coherent WARN?**
The effective minimum |rho| a first lock needs at the stream's window
length, derived from bounding the estimate's phase error to 0.10 rad:
`1/sqrt(1+2Nσ²)`, about 0.11 for a 4096-sample window. It replaced a fixed
0.5 that made calibration depend on capture bandwidth and per-branch noise
floor.

**Should `branch_phase_deg` be constant?**
Only on a shared-LO front end. Independent daughterboards (X310 + TwinRX)
lock at a random relative phase and walk afterwards — about −0.1 to −0.2°/s
on field rigs — so a frozen constant decays over minutes and tracking `mrc`
is right. Constant phase means `mrc-static` would serve.

**Is MRC diversity verified to help on air?**
It is verified not to hurt: post-aligner, every combined arm matches the
best branch within one decoded frame. A real gain over the best branch is
undemonstrated because every capture so far decodes at its ceiling;
tracking-as-default awaits a weak-signal pre-combine capture.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Overruns & host_drops — Reading a Downstream Signal]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
· Next →
[Part 9: Capture Lines — Aligning a Capture to the Log]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})
