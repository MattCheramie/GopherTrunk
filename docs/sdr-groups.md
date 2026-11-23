---
layout: page
title: SDR groups
description: Gang several role wideband SDRs into one logical receiver that covers more spectrum than any one of them, with a planned channel split, voice following across the whole span and a stitched spectrum view
nav_group: Reference
---

# SDR groups

One RTL-SDR at 2.4 MS/s covers about 2.16 MHz of usable spectrum. Many
trunked systems spread their control and voice channels wider than that. An
**SDR group** gangs several `role: wideband` sticks into one logical receiver:
you give the group a span and a channel plan, and GopherTrunk decides where
each stick is tuned and which stick decodes which channel. Three 2.4 MS/s
sticks cover about 6.3 MHz.

The members' IQ is **never combined**. Each channel, and each voice grant, is
decoded by the one member whose window covers it. That is why ordinary
dongles with independent clocks work: nothing ever has to line two streams up
sample for sample. Combining the streams into one coherent wideband capture
needs shared-clock hardware and is a separate design
([coherent stitching](design/coherent-sdr-stitching.md)).

## Configuration

```yaml
sdr:
  sample_rate: 2400000
  devices:
    - { serial: "00000011", role: wideband, group: south, gain: "auto" }
    - { serial: "00000012", role: wideband, group: south, gain: "auto" }
    - { serial: "00000013", role: wideband, group: south, gain: "auto" }
  groups:
    - name: south
      low_hz: 851_000_000
      high_hz: 857_000_000
      voice_taps: 2
      channels:
        - { frequency_hz: 851_012_500, system: "Metro P25" }
        - { frequency_hz: 856_987_500, system: "Metro P25" }
```

**Members** are ordinary `sdr.devices` entries with `role: wideband` and
`group: <name>`. A member keeps its own `gain`, `ppm` and `bias_tee`. It may
set `center_freq_hz` to pin itself; otherwise the planner chooses its centre.
A member must not set `channels`, `conventional`, `voice_taps`,
`signalling_taps`, `tuner_strategy` or its own `sample_rate`: those belong to
the group, and every member runs at `sdr.sample_rate`.

**The group** (`sdr.groups[]`):

| Key | Meaning |
|---|---|
| `name` | Members name it with `group:`. No spaces, `:` or `/`. |
| `low_hz`, `high_hz` | Optional span to cover. Leave both at 0 to cover just the channel list. For a trunked system, set them to the system's whole voice band so grants anywhere in it are followed. |
| `overlap_hz` | How far adjacent windows overlap. 0 = auto: the widest channel tap (48 kHz, 144 kHz for TETRA) plus 25 kHz. |
| `voice_taps`, `signalling_taps` | Allocated on **every** member (see the `sdr.devices` keys of the same name). |
| `tuner_strategy` | Used on every member. |
| `channels`, `conventional` | The same entries a standalone wideband device takes. |

## What the planner does

1. Pinned members claim the part of the span they cover.
2. The free members tile what is left so that adjacent windows overlap by at
   least `overlap_hz`. Spare members are spread over the most crowded gaps.
3. An auto-placed centre that would land within a channel's tap (plus 10 kHz)
   is nudged off it, so no channel sits on a stick's DC spur.
4. Every channel goes to the covering member where it sits closest to centre.

If the members cannot cover the span, the config is rejected with the
uncovered range named: add a member, raise `sdr.sample_rate`, or narrow the
span. The plan is logged at start-up, one `sdr group member` line per stick
with its centre, coverage and channels.

## Behaviour

- **Control channels** decode on the member that owns them, exactly as on a
  standalone wideband dongle.
- **Voice**: every member gets `voice_taps` virtual voice tuners. The voice
  pool tries each tuner in turn and uses one whose window holds the grant, so
  a grant anywhere in the span lands on a member that covers it. A member with
  no channel still streams, so its voice taps work.
- **Conventional channels** run continuously on the member that owns them.
- **Spectrum**: the web Spectrum panel lists the group as `group:<name>` and
  shows one panorama. Each member's DC spur is interpolated away, member
  levels are matched across the overlaps (so different gains leave no step),
  the seams are marked, and the hover readout names the member. Click-to-tune
  is off: members hold their planned centres, and tuning one member from the
  Spectrum panel is refused too. The symbol, eye and mixer views need one
  stream, so open a member there.
- **Recording**: a `baseband.record` entry whose `serial` names the group
  records every member to its own file (the serial is in the file name) and
  writes `<group>_plan_<UTC>.json` beside them: each member's centre,
  coverage, owned slice and channels. `tap: ddc` is not available on a group.
- **Signal Lab capture**: picking the group records from the one member whose
  window holds every requested slice. A full-band capture, or slices that
  straddle a seam, are refused with the members named.
- **Hunts** never pick a group member automatically, and an explicitly chosen
  member is treated as a borrow (its decode is suspended for the run, with
  confirmation).
- **A member that loses its stream** logs which slice of the span went dark
  and which channels it carried. The other members keep decoding while the
  daemon reacquires it.

## Limits

- Members' windows overlap, but each channel still needs one member to hold
  its whole tap. A signal wider than the overlap that sits across a seam is
  seen in pieces in the spectrum view and cannot be decoded.
- A per-member `ppm` error moves that member's channels by the error, as on
  any wideband dongle. Calibrate each stick.
- Each member costs what a standalone wideband dongle costs: one USB stream at
  `sdr.sample_rate` plus its channel and voice taps.
- Synthetic and construction tests cover the planner, the daemon wiring, the
  stitcher and the API. The feature has not yet been run on a real
  multi-stick rig.
