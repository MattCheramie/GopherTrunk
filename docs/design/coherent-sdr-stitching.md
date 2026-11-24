---
layout: page
title: "Design: Coherent SDR-group stitching"
description: Combining several SDRs' IQ into one coherent wideband stream on shared-clock hardware, what it needs, how it would be built, and the instrument that decides whether a rig can support it
nav_group: Reference
---

# Design: Coherent SDR-group stitching

Status: **design only.** Nothing here ships as a decode path yet. What ships
is the [virtual SDR group](../sdr-groups.md) and the measurement instrument
described at the end, which decides whether a given rig could support this.

## The problem

An [SDR group](../sdr-groups.md) gives several SDRs one channel plan, but each
member still streams on its own. Every channel is decoded by the member that
covers it, and the spectrum view is a composite of the members' FFTs. That is
the right design for ordinary dongles, and it does everything decoding needs.

Some things need **one coherent IQ stream** wider than any member:

- a signal wider than a member's overlap that sits across a seam;
- a wideband IQ recording that other tools can re-channelize as one file;
- a single FFT across the whole span, rather than stitched pieces.

Coherent stitching builds that stream from the members' IQ.

## Why ordinary RTL-SDRs cannot do it

Each RTL-SDR derives its LO and its sample clock from its own crystal. Two
sticks typically differ by 1 to 50 ppm:

- **Sample clocks**: 20 ppm at 2.4 MS/s is 48 samples per second of drift.
  No fixed alignment exists, and a seam smears within a fraction of a second.
- **LOs**: 20 ppm at 850 MHz is 17 kHz. The same carrier appears at different
  offsets in each member.

Both errors also wander with temperature. Tracking them continuously is
possible in principle, but every seam would then need a running resampler
whose ratio follows the measured drift. The error left after that tracking
lands in the combined stream as distortion at the seams, with nothing to
warn about it. The rule for this feature is: **no shared reference clock, no
coherent stitching.**

## Hardware prerequisites

- **One reference clock for every member.** Either a board that already
  shares one (KrakenSDR, a multi-channel USRP), or RTL-SDRs modified to take
  an external 28.8 MHz reference from one distribution amplifier.
- **The same sample rate on every member**, from that reference.
- **Overlapping windows**, which the group planner already gives, so that
  each seam has a shared band to align on.

A shared reference makes the sample clocks and LOs frequency-locked, but not
phase-aligned. Each member still has a start-time offset (USB enumeration and
stream start are not simultaneous) and a fixed LO phase. Both are constant on
a locked rig, so they can be measured once per stream and removed.

## Alignment

For each seam (adjacent members `a` < `b`):

1. Cut the overlap band out of both members with identical down-converters
   (same rate, same filter, so the same group delay).
2. **Coarse delay**: cross-correlate the two bands' power envelopes. A
   frequency offset leaves the envelopes unchanged, so this works even before
   the LO difference is known.
3. **Residual frequency**: the spectrum of `b[n+L]·conj(a[n])` has a line at
   the LO difference. On a locked rig it is zero; anything else is a fault.
4. **Fine delay and phase**: parabolic refinement of the correlation peak
   gives the fractional delay, and its angle gives the phase. Gate on
   `diversity.CrossStats` coherence: noise alone has no coherence, so a quiet
   overlap must report that rather than calibrate on nothing. This is the MRC
   rule from `internal/dsp/diversity`.
5. Chain the seams from the lowest member up, so every member is expressed
   relative to member 0.

`soapyremote.branchAligner` already implements an interpolating delay line
with a latch gate for two-branch MRC. It is the starting point for applying
the measured delays.

## Synthesis

Inside a composite device (a `sdr.Device` the pool hands out like any other,
built the way `Pool.WrapDevice` wraps the input decimator):

1. Delay each member by its measured offset and rotate it by its phase.
2. Up-sample each member to the combined rate and shift it to its place in
   the span.
3. In each overlap, cross-fade the two members with power-complementary
   filters (`|H_a|² + |H_b|² = 1`), so the noise floor and a carrier crossing
   the seam keep their level.
4. Sum.

Downstream, a group would then be **one wide device**: one wideband engine,
one spectrum producer and one baseband recording, with nothing group-aware
past the composite device.

## Refusal rule

The composite device re-measures every seam continuously (the same analysis,
on a slow cadence) and stops combining when a seam's coherence drops below
the gate or its delay or phase drifts. It falls back to the virtual group and
logs which seam failed and why. A silent bad combine is the failure mode to
design out, as the MRC wideband-skew lesson showed: combining skewed branches
cost 22 % of the decode yield with nothing to show for it.

## The instrument: does this rig qualify?

`TestSDRGroupSeamReplay` runs the alignment analysis above on a real group
recording and prints a verdict per seam. Record a group with
`baseband.record` naming the group (each member writes its own file and the
plan is written beside them), with a steady carrier in each overlap, then:

```sh
GT_SDR_GROUP_PLAN=<dir>/<group>_plan_<UTC>.json \
  go test ./cmd/gophertrunk -run TestSDRGroupSeamReplay -v
```

| Verdict | Meaning |
|---|---|
| `independent sample clocks (N ppm apart)` | No shared reference. Coherent stitching is impossible; the virtual group is the right mode. Every pair of unmodified RTL-SDRs lands here. |
| `sample clocks locked but the phase walks` | Shared sample clock, but the LOs differ (for example, separate PLLs on one reference that are not phase-locked). Stitching would need per-seam phase tracking. |
| `coherent stitching feasible on this seam` | Fixed delay and phase. A composite device could align this pair. |
| `no common signal in the overlap` | Nothing to measure. Put a carrier or a noise source in the overlap and record again. |

The analysis is pinned by `TestAnalyzeSeam` on synthetic scenes for each case
(including a realistic 20 ppm RTL pair, whose LOs sit 17 kHz apart). No real
multi-SDR recording has been analysed yet.

## What would be built, in order

1. A shared-reference rig recorded, with the instrument reporting
   "feasible" on every seam. Without this, stop.
2. The composite device with fixed per-seam calibration measured at stream
   start, behind an explicit `sdr.groups[].coherent: true`.
3. Continuous re-measurement and the refusal rule.
4. A/B on decode yield of a seam-straddling signal against the virtual group.
   Yield is the verdict, never EVM.
