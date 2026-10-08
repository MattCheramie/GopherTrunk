---
title: "The Conventional Scanner, Part 4: Squelch on the Channel, Not the Span — The In-Channel Power Meter"
description: "How issue 1239 changed what squelch_dbfs measures: from the RMS power of the whole SDR span, which any other carrier or an auto-gain noise floor held above threshold, to the power inside a ±8 kHz channel filter at 48 kHz; the hysteresis and debounce that act on it, why it still moves with gain, and per-channel gain."
category: tutorials
keywords: sdr scanner squelch dbfs, in-channel power squelch, rtl-sdr squelch opens on empty channel, carrier squelch vs tone squelch, channel filter 8 khz decimate, squelch hysteresis debounce scanner, per-channel gain scan list, noise quieting squelch not dbfs, issue 1239 conventional scanner, gophertrunk conventional scanner
tags: [conventional-scanner, squelch, dbfs, gain, analog-fm, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 4
---

*Part 4 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's `scanner.conventional` scan list.
[Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})
put the LO below the channel and mixed the channel back to DC. This part is
about the number compared against `squelch_dbfs` on every chunk of that
stream — a number that, until
[#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239), was the
power of the entire SDR span. The reporter's phrase for the result was
"Tone only, not CSQ AND Tone": the carrier squelch never closed, so the
CTCSS detector was the only gate left. `channel_power.go` is the fix, and
the per-channel `gain` key in `gain.go` landed in the same pull request.*

> **TL;DR:** The FM squelch used to be `PowerDbFS` of the scanner's whole
> IQ stream (2.4 MHz on an RTL-SDR): any carrier anywhere in the span, or the
> noise floor an auto-gain tuner holds near full scale, read as "carrier
> present" on every channel, so untoned channels opened calls on empty air
> and toned ones were gated by the tone detector alone. `channelPowerMeter`
> now measures the channel: integer-decimate to ~48 kHz
> (`dsp.NewResampler(1, m, m·8+1, 8.6)`, m = 50 at 2.4 MS/s), a 63-tap
> Kaiser low-pass at ±`toneChannelCutoffHz` = 8 kHz — the same front end
> the tone detectors use — then RMS in dBFS. An on-channel carrier reads
> what it read before (−20 dBFS → −21..−19), a 12.5 kHz neighbour below
> −50, a carrier 500 kHz away below −70, so existing `squelch_dbfs` values
> keep their meaning. The squelch opens at `squelch_dbfs`, counts down only
> below `squelch_dbfs − 3 dB`, and ignores blips under 50 ms. It is still
> absolute dBFS, so a gain change moves it; `gain` per channel is the lever
> for a mixed-band list, and a noise-quieting squelch is the gain-independent
> answer not yet built.

**Key takeaways**

- **Measure the channel, not the device.** A squelch that reads the whole
  span is a detector for "is anything on this dongle", which is always yes.
- **Calibration is preserved by construction.** For a signal on the
  channel the filtered reading equals the old whole-span reading, so
  thresholds tuned before #1239 still work.
- **Three numbers govern one dwell.** `squelch_dbfs` opens,
  `squelch_hysteresis_db` sets the close level, `activity_debounce_ms` sets
  how long activity must last to matter.
- **dBFS is still a gain-staging trap.** The meter is scale-dependent;
  per-channel `gain` manages that, and only a noise-quieting squelch would
  remove it.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The meter | ↓m to ~48 kHz, ±8 kHz Kaiser LPF (63 taps), `PowerDbFS` | `channelPowerMeter` (`internal/scanner/conventional/channel_power.go`) |
| Shared front-end constants | `toneRefRateHz` 48 000, `toneChannelCutoffHz` 8 000 | `tone_frontend.go` |
| Selection | FM → power meter + `squelch_dbfs`; AM → C/N meter + `squelch_cn_db`; no rate → whole-span `PowerDbFS` | `squelchMeasure`, `buildChannelPowerMeter`, `warnSpanPowerSquelch` |
| Dwell thresholds | open at `SquelchDbFS`, keep-alive at `open − SquelchHysteresisDb` (3), debounce 50 ms | `beginDwell` (Part 1) |
| Regression pins | empty channel + −20 dBFS carrier 500 kHz away → no call; −40 dBFS 2 kHz off → call | `TestConvScannerSquelchIgnoresCarrierElsewhereInSpan`, `TestConvScannerSquelchOpensOnInChannelCarrier`, `TestChannelPowerMeterCalibration` |
| Per-channel gain | `GainSetter.SetGain(tenthDB)` before each tune, device default otherwise | `gain.go`, `cmd/gophertrunk/conv_gain.go`, `TestConvScannerAppliesPerChannelGain` |

## In this post

- **A squelch that measured the dongle** — the #1239 symptom and its two faces.
- **The in-channel meter** — decimate, filter, RMS, and the calibration that makes it drop-in.
- **Open, keep-alive, debounce** — how the reading drives a dwell.
- **Why it still moves with gain** — dBFS, the auto-gain floor, and the squelch not built.
- **Gain per channel** — the key, the parser, and what the scanner does with the tuner.

## A squelch that measured the dongle

`PowerDbFS` in `squelch.go` is the primitive the scanner has always used:
mean of |s|² over a chunk, in dB relative to a unit-amplitude tone. The
package comment explains the choice — IQ-domain power is measurable before
any demod chain spins up, which makes blind channel visits cheap, and an FM
carrier is constant-envelope so the same metric serves hangtime detection.
The problem was the input. The scanner's IQ stream is the SDR's whole span,
and the FM squelch computed `PowerDbFS` over all of it.

That has two faces, and the reporter hit both. Any other carrier in the
span — a paging transmitter 500 kHz away, a strong repeater on the next
channel — puts the span's power above any sensible `squelch_dbfs`, so an
untoned channel opened a call on empty air every time the scanner visited
it. On a tone-gated channel the power test was permanently satisfied, so
the CTCSS/DCS detector became the only gate — "Tone only, not CSQ AND
Tone". And a tuner on `gain: auto` holds its *noise floor* near full scale,
which does the same thing with no other carrier at all.
`TestConvScannerSquelchIgnoresCarrierElsewhereInSpan` is the fixture: an
empty channel (noise 90 dB down) with a −20 dBFS carrier 500 kHz away,
`squelch_dbfs: -50`. Whole-span power reads about −20 dBFS and the old
scanner opened a call; the test demands zero.

## The in-channel meter

`channelPowerMeter` measures what the channel carries, through the same two
stages the CTCSS and DCS detectors already ran ahead of their discriminator
(`toneFrontEnd` in `tone_frontend.go`):

```go
// internal/scanner/conventional/channel_power.go (shape)
func newChannelPowerMeter(sampleHz float64) *channelPowerMeter {
    m := int(sampleHz / toneRefRateHz)          // 50 at 2.4 MS/s
    if m < 2 { m = 1 } else { pre = dsp.NewResampler(1, m, m*8+1, 8.6) }
    rate := sampleHz / float64(m)
    if fc := toneChannelCutoffHz / rate; fc < 0.45 {
        chanFilter = filter.NewFIR(filter.LowpassKaiser(63, fc, 8.6))
    }
    return &channelPowerMeter{pre: pre, chanFilter: chanFilter, level: PowerDbFS(nil)}
}

func (m *channelPowerMeter) process(iq []complex64) float64 {
    if m.pre != nil { iq = m.pre.Process(m.scratch[:0], iq) }
    if m.chanFilter != nil { iq = m.chanFilter.Process(m.chanBuf[:0], iq) }
    if len(iq) == 0 { return m.level }           // hold, never read as silence
    m.level = PowerDbFS(iq)
    return m.level
}
```

The decimator is a polyphase anti-alias FIR of `m·8+1` taps (401 at
2.4 MS/s, Kaiser β = 8.6) dropping the rate by the largest integer that
lands near `toneRefRateHz` = 48 000 — 48 kHz from 2.4 MS/s (m = 50),
48.76 kHz from 2.048 MS/s (m = 42). The channel filter is a 63-tap Kaiser low-pass at
`toneChannelCutoffHz` = 8 kHz one-sided: an FM channel's ±2.5–5 kHz
deviation plus Carson margin passes, a 12.5 or 25 kHz neighbour does not,
and the rest of the span — whose FM noise would otherwise swamp the reading
— is gone. The RMS of what is left is the in-channel power.

Two details carry the drop-in property. **Scale**: a unit-amplitude tone on
the channel still reads 0 dBFS, so a carrier that dominated the whole-span
reading reads the same through the filter —
`TestChannelPowerMeterCalibration` pins an on-channel −20 dBFS carrier at
−21..−19 dBFS, a 12.5 kHz neighbour below −50 and a carrier 500 kHz away
below −70. **Hold**: a chunk too short to yield one decimated sample
repeats the last reading rather than returning −∞, because a stream of tiny
chunks would otherwise read as silence and break a dwell's hangtime
accounting (`TestChannelPowerMeterHoldsOnTinyChunk`). `reset()` clears the
filter histories on every retune so the previous channel's samples cannot
leak into the first reading of the next. The no-harm half of the regression
is `TestConvScannerSquelchOpensOnInChannelCarrier`: the same −20 dBFS
carrier 500 kHz away plus a −40 dBFS carrier 2 kHz off centre, and a
−50 dBFS squelch must open.

The meter exists only when the scanner knows its sample rate.
`buildChannelPowerMeter` returns nil for an AM channel (which squelches on
carrier-to-noise, Part 8) and when `SampleRateHz` is zero; in the latter
case `squelchMeasure` falls back to whole-span `PowerDbFS` and
`warnSpanPowerSquelch` says so once:
`conv: scanner sample rate is zero; squelch_dbfs measures the whole SDR span instead of each channel`.
The daemon always passes `cfg.SDR.SampleRate`, so that line is a sign of a
hand-built scanner, not a config mistake.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="A pipeline from 2.4 megasample per second IQ through a decimate-by-50 filter to 48 kilohertz, a plus or minus 8 kilohertz low-pass, and an RMS stage compared against squelch_dbfs. Below, a spectrum sketch: a minus 20 dBFS carrier 500 kilohertz away and the auto-gain noise floor are rejected, a minus 40 dBFS carrier 2 kilohertz off centre passes and sets the reading.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">channelPowerMeter: what reaches the squelch comparison</text>
  <rect x="20" y="30" width="110" height="36" fill="none" stroke="var(--fg-muted)"/>
  <text x="75" y="46" text-anchor="middle" fill="currentColor" font-size="9">IQ 2.4 MS/s</text>
  <text x="75" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="8">whole span</text>
  <line x1="130" y1="48" x2="160" y2="48" stroke="var(--fg-muted)"/>
  <rect x="160" y="30" width="130" height="36" fill="none" stroke="currentColor"/>
  <text x="225" y="46" text-anchor="middle" fill="currentColor" font-size="9">↓50 polyphase FIR</text>
  <text x="225" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="8">401 taps → 48 kHz</text>
  <line x1="290" y1="48" x2="320" y2="48" stroke="var(--fg-muted)"/>
  <rect x="320" y="30" width="130" height="36" fill="none" stroke="currentColor"/>
  <text x="385" y="46" text-anchor="middle" fill="currentColor" font-size="9">±8 kHz Kaiser LPF</text>
  <text x="385" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="8">63 taps</text>
  <line x1="450" y1="48" x2="480" y2="48" stroke="var(--fg-muted)"/>
  <rect x="480" y="30" width="180" height="36" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="570" y="46" text-anchor="middle" fill="var(--accent)" font-size="9">PowerDbFS → ≥ squelch_dbfs?</text>
  <text x="570" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="8">keep-alive: open − 3 dB</text>
  <line x1="40" y1="150" x2="640" y2="150" stroke="var(--fg-muted)"/>
  <text x="340" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0 (channel)</text>
  <text x="40" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−1.2 MHz</text>
  <text x="640" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+1.2 MHz</text>
  <rect x="336" y="136" width="8" height="14" fill="none" stroke="var(--accent)" stroke-dasharray="2 2"/>
  <text x="340" y="130" text-anchor="middle" fill="var(--accent)" font-size="8">±8 kHz passband</text>
  <line x1="465" y1="100" x2="465" y2="150" stroke="currentColor" stroke-width="2"/>
  <text x="465" y="96" text-anchor="middle" fill="currentColor" font-size="8">−20 dBFS @ +500 kHz: rejected (&lt; −70)</text>
  <line x1="340.5" y1="122" x2="340.5" y2="150" stroke="var(--accent)" stroke-width="2"/>
  <text x="250" y="116" text-anchor="middle" fill="var(--accent)" font-size="8">−40 dBFS @ +2 kHz: passes, reads −40</text>
  <path d="M40 142 Q340 136 640 142" fill="none" stroke="var(--fg-muted)" stroke-dasharray="2 3"/>
  <text x="560" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="8">auto-gain noise floor: outside the passband it no longer counts</text>
  <text x="120" y="186" text-anchor="middle" fill="currentColor" font-size="8">old reading −20 dBFS (span) → new reading −40 dBFS (channel)</text>
</svg>
<figcaption>The meter runs the tone detectors' front end and reads RMS after it; an on-channel signal reads the same dBFS it always did, everything outside ±8 kHz stops counting.</figcaption>
</figure>

## Open, keep-alive, debounce

The reading drives Part 1's dwell through `squelchMeasure`, which returns
the meter's `process` and the channel's open threshold. In `scanWindow` a
chunk opens the squelch when `level(iq) >= open`. In `beginDwell` the
comparison is against `keepAlive := open − ch.SquelchHysteresisDb`: a chunk
counts as active while the channel's power is within 3 dB of the opening
level, so a signal that flickers across `squelch_dbfs` cannot start and stop
the countdown. The debounce then acts on time rather than level — activity
must be continuous for `ActivityDebounce` (50 ms) to reset a running
countdown, so an impulse that clears the keep-alive level for one chunk
leaves the countdown alone.

In numbers, with `squelch_dbfs: -48`: the call opens at −48, stays alive
down to −51, starts counting at the first chunk below −51, ignores a 2 ms
return above −51, resets on 50 ms above it, and ends after 1500 ms below it.
`squelch_hysteresis_db` is the key to raise when a weak repeater's tail
chatters the countdown; `activity_debounce_ms` is the one to lower when a
real but choppy transmission is being cut short.

## Why it still moves with gain

The meter is better, not different in kind: it is still absolute dBFS, so
a tuner gain change moves every channel's reading by the same amount and a
`squelch_dbfs` tuned at one gain is wrong at another. [The Analog Edge Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
is the long form of what the number means, and
[Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
of the same series is the rule this project keeps relearning: never chase
a software threshold with the gain knob. The project note that closed
#1239 says what the gain-independent answer is — a noise-quieting squelch
on the FM discriminator, the `dmrrx.carrierGate` variance idea that
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
describes, which separates carrier from noise by a phase statistic at any
level — and that it is **not built**. Until it is, the in-channel meter is a
dBFS gate on a clean input, and the scale-invariance argument of
[Coherence, Not dBFS]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
applies to it as much as to the MRC calibration that lesson came from.

## Gain per channel

What the same pull request did build is the lever an operator has in the
meantime. One scanner SDR serves a list that can mix VHF FM, air-band AM
and UHF, and no single tuner gain suits all of them — `auto` serves AM
badly in particular. A channel may therefore carry `gain`, in the same form
as `sdr.devices[].gain`: `"auto"`, or tenths of a dB. `convChannelGain`
(`cmd/gophertrunk/conv_gain.go`) parses it with the device parser
(`parseGain`), WARNs through `gainLooksLikeDBMistake` when a value looks
like whole decibels (`"28"` is 2.8 dB; write `"280"`), and
`convDeviceGain` reads the gain the pool applied at open — automatic (−1)
when unconfigured — as `DefaultGainTenthDB`. The daemon passes the device
itself as the `GainSetter`, because the iqtap broker has no gain control.

The scanner's rule, from the `gain.go` header: **it only touches the gain
when at least one channel sets one.** `usesChannelGain` decides that at
construction; a list without per-channel gains never calls `SetGain`
(`TestConvScannerLeavesGainAloneWithoutChannelGains`). Once any channel
does, the scanner owns the device's gain: `applyChannelGain` runs before
every tune, writes the channel's gain or the device default for a channel
without one, and skips the write when the wanted gain is the one it last
applied. `TestConvScannerAppliesPerChannelGain` pins the sequence — a 400
channel followed by two default-280 channels writes 400, 280, 400, 280 with
no rewrite between the two defaults, and the first gain lands before the
first tune. A gain that changes from elsewhere (the API) is overwritten at
the next channel whose gain differs from the last one the scanner applied.
A failed write — the test's `i2c stall` stands in for an RTL-SDR's
control-pipe faults — is logged once per value:

```text
WRN conv: per-channel gain write failed; scanning at the device's current gain freq_hz=155000000 label=A gain_tenth_db=300 err="i2c stall"
```

and the scan carries on at whatever gain the device holds
(`TestConvScannerGainFailureKeepsScanning`). Verification for both halves of this part is the same: pinned by synthetic
span fixtures (`spanChunks`, carriers plus white noise at a chosen span
power) and unit tests; the repository records no on-air run that
specifically confirms the in-channel squelch on the reporter's rig.

### How the meter shaped the Go code

- **One front end, three consumers.** `toneRefRateHz` and
  `toneChannelCutoffHz` are shared with the CTCSS and DCS detectors, so the
  power the squelch measures is the power the tone detector sees.
- **Per-channel meters, reset per visit.** `powerMeters[i]` parallels
  `channels`, and `Run` calls `reset()` on every retune.
- **Hold, never −∞.** A short chunk repeats the last reading; the scanner's
  hangtime accounting assumes every chunk yields a level.
- **Gain is a scanner-owned resource.** Once active, `appliedGain` is the
  single source of truth and `gainWarned` rate-limits failures per value.

## Where this goes next

With the carrier squelch finally measuring the channel, the tone gate is a
second condition again rather than the only one.
[Part 5]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }})
reads `ctcss.go`: why a Goertzel bin rounded to 5 Hz made 162.2 Hz open on
159.8 Hz, why a 200 ms block could not separate tones 2.3 Hz apart, why the
old magnitude threshold needed ~540 Hz of deviation that no narrowband radio
sends, and the reporter's real-air slices that pin the fix at the SDR's own
sample rate.

## FAQ

**Why did my conventional scanner open calls on empty channels?**
Before issue #1239 `squelch_dbfs` was compared against the RMS power of the
whole SDR span, so any carrier elsewhere in it — or an auto-gain noise floor
near full scale — read as carrier present on every channel.
`channelPowerMeter` now measures only the ±8 kHz channel after decimation
to 48 kHz; a carrier 500 kHz away reads below −70 dBFS.

**Do I need to retune squelch_dbfs after the in-channel change?**
Usually not. For a signal on the channel the filtered reading equals the old
reading (`TestChannelPowerMeterCalibration`: a −20 dBFS carrier reads
−21..−19 dBFS), so a threshold that worked for real signals still does. If
your old value was raised to fight off-channel energy, it can come down.

**Is squelch_dbfs gain-independent now?**
No. It is still absolute dBFS, so a tuner gain change moves every reading.
Set `gain` per channel for a mixed-band list, and avoid `gain: auto` on a
scan list. A noise-quieting squelch on the discriminator — the
`dmrrx.carrierGate` idea — would be gain-independent and is not built.

**What does the per-channel gain key do?**
`gain: "280"` (tenths of a dB) or `"auto"` is written to the scanner SDR
before that channel is tuned; a channel without one returns the device to
its configured gain. If no channel sets a gain the scanner never touches it.
A failed write logs `conv: per-channel gain write failed` once per value and
scanning continues.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Offset Tuning — Why the LO Sits Below the Channel]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})
· Next →
[Part 5: CTCSS Done Right — Exact Bins, Reverse Bins, Deviation Thresholds]({{ '/blog/tutorials/conventional-scanner-05-ctcss-done-right/' | relative_url }})
