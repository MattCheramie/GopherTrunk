---
title: "The Analog Edge, Part 12: Front-End Classes — Shared LO vs Independent PLLs"
description: Why two receive channels on one chip keep a constant relative phase while two daughterboards with their own synthesizers lock at a random phase and walk afterwards, how that hardware class decides between mrc and mrc-static, and how the branch_phase_deg log field identifies your class without recording anything.
category: tutorials
keywords: shared lo sdr, independent pll phase drift, twinrx phase coherence, b210 dual channel phase, mrc static vs tracking, branch phase deg, rx subdev spec, frequency locked phase random, gophertrunk analog edge
tags: [analog-edge, diversity, hardware, pll, usrp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Analog Edge"
series_part: 12
---

*Part 12 of **The Analog Edge**, a 14-part field guide to the analog half of a
GopherTrunk system. [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})
put two antennas on the mast and left one question open: `mrc` or
`mrc-static`? The answer isn't a preference — it's a property of the silicon
between your two RX ports. This part sorts diversity-capable front ends into
two classes, shows why one of them makes a frozen calibration constant decay,
and hands you the field instrument that identifies your class from a log line
alone. Our marginal reader, now shopping for a dual-channel radio, gets to
make this choice on evidence.*

> **TL;DR:** Two front-end classes. **Shared LO** (one chip, one synthesizer —
> a B210's AD9361 driving both channels): branch-to-branch phase is fixed
> trace skew plus a power-up divider phase, genuinely constant while the
> synthesizer stays locked — a one-shot calibration (`mrc-static`) is correct.
> **Independent PLLs** (separate daughterboards — an X310 with two TwinRX
> cards, `rx_subdev_spec=B:0 A:0`): frequency-locked to a common reference so
> there's no beat, but the relative phase is **random at each lock and walks
> afterwards** — a constant measured once decays, and tracking (`mrc`) is
> required. The no-capture instrument is **`branch_phase_deg`** in the 30 s
> MRC health line: constant ⇒ shared LO; walking ⇒ independent PLLs. Either
> way, every retune re-randomizes the phase, which is why the calibrator
> resets on retune.

**Key takeaways**

- **"Frequency-locked" is not "phase-locked."** A common 10 MHz reference
  guarantees the two synthesizers don't drift apart in frequency; it says
  nothing about the phase each PLL settles at, or how that phase creeps with
  temperature afterwards.
- **The hardware class decides the software mode.** A constant needs measuring
  once; a walk needs following. Choosing `mrc-static` on TwinRX-class hardware
  gives you a combiner that slowly rotates out of alignment — worst case
  toward cancellation.
- **`branch_phase_deg` answers it for free.** Watch the health line across a
  few intervals: flat within a degree or two ⇒ shared LO; a steady walk ⇒
  independent PLLs. No capture, no test gear.
- **Tracking is deliberately slow and clamped.** The `mrc` loop smooths over
  ~200 ms and bounds each update, so it follows thermal drift without ever
  stepping the phase fast enough to disturb the decoders — the design proof
  lives in [Weak-Signal Engineering Part 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }}).

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The two modes | tracking vs one-shot frozen calibration | `sdr.soapy_remote[].diversity: "mrc"` / `"mrc-static"` |
| Static = alpha 0 | first accepted window snapped, then frozen | `internal/dsp/diversity/tracking.go` (`TrackingOptions.Alpha`) |
| Tracking loop | 2 ms windows, ~200 ms time constant, clamped steps | `internal/sdr/soapyremote/mrc.go` (`mrcCalWindowMs`, `mrcTrackTauMs`) |
| Field instrument | relative branch phase in the health line | `branch_phase_deg` (`diversityReporter`, `mrc.go`) |
| Retune behavior | drop the estimate — a new lock is a new phase | `TrackingCalibrator.Reset` (`tracking.go`) |
| Offline drift trace | windowed phase trace + °/s from your capture | `TestDiversityCombinerReplay` (`cmd/gophertrunk/diversity_replay_test.go`) |
| Differential safety | why tracking can't corrupt π/4-DQPSK decoding | [WSE Part 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }}), `TestTrackingCalibratorIsDifferentialSafe` |

## In this post

- **Class 1: shared LO** — one synthesizer, one constant.
- **Class 2: independent PLLs** — locked in frequency, free in phase.
- **The field instrument** — reading `branch_phase_deg` like a pro.
- **What each class needs from software** — static snap vs clamped tracking.
- **The decision table** — symptom → class → mode.

## Class 1: shared LO — one synthesizer, one constant

In a single-chip dual-RX front end — the AD9361 in a USRP B210 is the
canonical example — both receive channels are mixed down by *the same* local
oscillator. The phase relationship between channel 0 and channel 1 is set by
fixed board trace skew plus whatever phase the chip's internal dividers
powered up in: an arbitrary constant, but a **constant**, for as long as the
synthesizer stays locked. Measure it once and you're done — which is exactly
what `mrc-static` does. In GopherTrunk's calibrator, static mode is literally
tracking with `Alpha: 0` (`internal/dsp/diversity/tracking.go`): the first
window that passes the quality gates is **snapped** — no smoothing, no clamp —
making it bit-identical to a one-shot least-squares calibration on that
window. Both modes therefore *start* identically; they differ only in whether
the estimate is ever revisited.

One constant caveat even here: that divider phase is re-rolled every time the
synthesizer re-locks. Power cycle, retune, sample-rate change — new constant.
This is why `TrackingCalibrator.Reset` is called on every retune in both
modes: the old measurement isn't stale, it's *about different hardware state*.

## Class 2: independent PLLs — locked in frequency, free in phase

Now put the two channels on separate daughterboards, each with its own
synthesizer — an X310 carrying two TwinRX cards, selected with
`rx_subdev_spec=B:0 A:0` in the device `args`. Both PLLs discipline
themselves to the same 10 MHz reference, so their *frequencies* are identical
— no beat note, ever. But a PLL locks its output phase wherever its own loop
settles: **random at each lock**, and then **walking** afterwards as
temperature moves each synthesizer's components independently. The walk is
slow — this is drift, not modulation — but it never stops.

For a combiner, that difference is fatal to a frozen constant. `mrc-static`
on this hardware measures a perfect alignment at minute zero, and then the
true phase strolls away from it: the coherent sum degrades smoothly, and if
the walk carries the branches toward anti-phase, MRC's coherent addition
turns into partial cancellation — a diversity rig that's *worse* than one
antenna, with nothing in `branch_dbfs` to show for it. The fix isn't a better
initial measurement; it's a loop that keeps measuring. That's `mrc`.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Relative branch phase versus time for the two front-end classes. The shared-LO trace is a flat horizontal line at a constant phase offset, annotated as one chip, one synthesizer, where mrc-static is correct. The independent-PLL trace starts at a different random phase after each lock and walks steadily away over minutes, annotated as two synthesizers on a common reference, where tracking mrc is required. A dashed vertical line marks a retune, after which both traces jump to new random values.">
  <line x1="55" y1="20" x2="55" y2="165" stroke="var(--fg-muted)"/>
  <line x1="55" y1="165" x2="650" y2="165" stroke="var(--fg-muted)"/>
  <text x="10" y="30" fill="var(--fg-muted)" font-size="9">relative</text>
  <text x="10" y="42" fill="var(--fg-muted)" font-size="9">phase (°)</text>
  <text x="590" y="182" fill="var(--fg-muted)" font-size="9">time (min)</text>
  <line x1="380" y1="20" x2="380" y2="165" stroke="var(--fg-muted)" stroke-dasharray="3,4"/>
  <text x="380" y="14" text-anchor="middle" fill="var(--fg-muted)" font-size="9">retune (new LO lock)</text>
  <polyline points="55,70 380,70" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <polyline points="380,110 650,110" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="140" y="60" fill="var(--accent)" font-size="10">shared LO: constant (new constant per lock)</text>
  <polyline points="55,140 120,133 190,124 260,112 320,97 380,86" fill="none" stroke="currentColor"/>
  <polyline points="380,45 440,52 500,63 560,76 620,92 650,101" fill="none" stroke="currentColor"/>
  <text x="90" y="155" fill="currentColor" font-size="10">independent PLLs: random start, then walks</text>
  <text x="352" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">flat trace ⇒ mrc-static is enough · walking trace ⇒ a frozen constant decays, track it</text>
</svg>
<figcaption>The two classes in one plot: a shared LO holds a constant between locks; independent PLLs re-roll the phase at every lock and walk between them.</figcaption>
</figure>

## The field instrument: `branch_phase_deg`

You don't need a capture, a scope, or a datasheet to classify your radio. The
MRC health line from [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})
prints `branch_phase_deg` — the phase of the currently applied branch gain —
every 30 seconds. Watch it across a few intervals:

- **Constant** (jitters within a degree or two, no direction): shared-LO
  behavior. `mrc-static` is sufficient; `mrc` costs nothing extra (on this
  hardware every window re-measures the same constant and the smoothing is a
  variance-reducing no-op).
- **Walking** (a consistent drift, degrees per minute): independent PLLs.
  Tracking is doing real work; `mrc-static` would decay.

For a quantitative answer, the offline harness turns your own
`diversity_capture` into a drift measurement: `TestDiversityCombinerReplay`
prints a windowed phase trace and, when the phase walks, the drift rate in
degrees per second. Flat trace ⇒ the frozen constant was fine on this
hardware; walking trace ⇒ the number tells you how fast `mrc-static` would
have rotted. That trace is the primary deliverable of the harness — the
four decode arms are the verdict, the phase trace is the explanation.

## What each class needs from software

The operator-level view of what `mrc` does with a walking phase, in three
properties (the proofs and the expensive lessons behind them are
[WSE Part 11]({{ '/blog/deep-dives/weak-signal-engineering-11-tracking-mrc/' | relative_url }})'s
subject):

- **Slow on purpose.** Estimates accumulate over 2 ms windows and smooth with
  a ~200 ms time constant (`mrcCalWindowMs`, `mrcTrackTauMs`) — far slower
  than any burst, far faster than thermal drift. The loop follows the walk
  without ever moving fast enough to look like modulation.
- **Clamped on purpose.** One accepted update may move the applied phase at
  most ~2.9° (`trackingMaxStepRad`), a bound that holds regardless of sample
  rate or a freak calibration window. A rejected window *holds* the previous
  gains — it never drops back to passthrough mid-stream, because that step
  would itself be the glitch the design exists to avoid.
- **Anchored on purpose.** The reference branch's weight is pinned to exactly
  1+0j, so the combiner's output phase is anchored to the reference antenna's
  own signal — the property that makes this safe *ahead of* GopherTrunk's
  differential TETRA demodulator, pinned by
  `TestTrackingCalibratorIsDifferentialSafe`.

### How that principle shaped the Go code

- **Static is a special case of tracking, not a second implementation.**
  `mrc-static` is the same `TrackingCalibrator` with `Alpha: 0`
  (`TrackingOptions`, `tracking.go`) — one code path, so the two modes cannot
  drift apart, and any capture-verified fix to one is automatically a fix to
  the other.
- **The first window is snapped in both modes.** Before the first accepted
  estimate there is nothing to smooth *toward*, so the code snaps it raw —
  which is what makes the static/tracking A/B fair: both arms start from a
  bit-identical calibration and diverge only in what happens next.
- **Retunes reset, rejections hold.** `Reset()` on a retune (a new LO lock is
  new hardware state) but *hold* on a rejected window (falling back to
  passthrough mid-stream is itself a phase step) — two different failure
  classes, two deliberately different responses.

## The decision table

| What you observe | Hardware class | Mode | Notes |
|---|---|---|---|
| `branch_phase_deg` flat across intervals | shared LO (B210-class) | `mrc-static` or `mrc` | both fine; static is the minimal choice |
| `branch_phase_deg` walking steadily | independent PLLs (TwinRX-class) | `mrc` | a frozen constant decays toward cancellation |
| Phase jumps only at retunes/restarts | either (that's the re-lock) | keep current mode | expected in both classes; calibrator resets itself |
| Combined decode slowly degrades over an hour, recovers on restart | independent PLLs on `mrc-static` | switch to `mrc` | the restart re-measured the constant — classic decay signature |
| `coherence` stuck 0.3–0.5, phase noisy | not a class problem | — | wideband-scalar limit from [Part 11]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }}): antennas too far apart |
| Capture replay shows °/s drift figure | independent PLLs | `mrc` | `TestDiversityCombinerReplay`'s phase trace, quantified |

## Where this goes next

Twice now, a log field has answered a hardware question that used to need
test equipment — `coherence` in Part 11, `branch_phase_deg` here. That's not
an accident; it's a design rule with a history: the one time this project
gated a DSP decision on an *absolute level* instead, an operator raised their
front-end gain 17 dB to satisfy a software constant.
[Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
tells that story and unpacks the scale-invariant number that replaced the
gate — the most transferable idea in this series.

## FAQ

**My two channels share a reference clock. Isn't that enough for `mrc-static`?**
No — that's precisely the trap. A shared reference locks *frequency*, so the
phase difference doesn't spin; but each PLL still settles at a random phase
per lock and creeps with temperature. Shared *reference* is class 2; shared
*LO* — one synthesizer physically driving both mixers — is class 1.

**Which class is my radio?**
One RF chip driving both RX ports (B210, and other AD9361/AD9364-family
duals): shared LO. Separate daughterboards or separate full receive chains
per channel (X310 + TwinRX, `rx_subdev_spec` selecting two boards):
independent PLLs. When documentation is ambiguous, `branch_phase_deg` over
ten minutes settles it empirically.

**Is there any reason to prefer `mrc-static` when both work?**
It's the minimal, fully-deterministic choice: one measurement, then a fixed
linear combine — nothing adapts at runtime, which some operators prefer for
auditability, and it's the natural A/B baseline. On shared-LO hardware the
tracking mode converges to the same constant anyway, so the practical
difference is philosophical.

**How fast does the TwinRX-class phase actually walk?**
It's thermal, so it depends on your enclosure and airflow — the honest answer
is your own number: run `TestDiversityCombinerReplay` on a
`diversity_capture` and read the °/s off the phase trace. The tracking loop's
~200 ms time constant is orders of magnitude faster than any thermal walk, so
`mrc` has margin to spare in either case.

**Does any of this matter for a single-channel rig?**
The modes don't, but the lesson does: "calibrate once" is only valid when the
thing you measured is physically constant. The same reasoning shows up across
GopherTrunk — per-dongle crystal offsets are *tracked* as running averages
([autotune]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }}))
for exactly the same reason.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Two Antennas — Diversity & MRC From the Operator's Seat]({{ '/blog/tutorials/analog-edge-11-diversity-mrc/' | relative_url }})
· Next →
[Part 13: Coherence, Not dBFS — Scale-Invariant Health]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
