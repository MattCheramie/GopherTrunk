---
title: "From the Issue Tracker, Season 2, Part 13: The Whole-Span Squelch — A Scanner Gated by Every Carrier in 2.4 MHz"
description: "Why a conventional scanner's CTCSS channels opened on noise at squelch settings of −10, −5 and −1 dBFS: squelch_dbfs was compared against the RMS power of the SDR's entire 2.4 MHz span, so any carrier anywhere kept the power half of the gate open. The channelPowerMeter fix, the per-channel gain that shipped with it, and the reporter's on-air confirmation."
category: solution-postmortem
keywords: scanner squelch opens on noise, squelch_dbfs whole span, ctcss csq and tone, in-channel power squelch, conventional scanner rtl-sdr 2.4 mhz, per-channel gain scan list, PowerDbFS channel filter, auto gain noise floor squelch, gophertrunk conventional scanner, gophertrunk from the issue tracker s2
tags: [from-the-issue-tracker-s2, conventional, squelch, ctcss, scanner, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 13
---

*Part 13 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems on GopherTrunk bugs that fought back.
[Part 12]({{ '/blog/solution-postmortem/issue-tracker-s2-12-cgo-disabled-is-not-static/' | relative_url }})
found a dynamic section in a binary every install page called static.
This part returns to the conventional scanner and a report
([#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239)) whose
third observation — "CTCSS/DCS gating ignores squelch, behaves as Tone
only" — was exactly right about the symptom and pointed at a gate that had
been measuring the wrong population since the day it was written.*

> **TL;DR:** The conventional scanner's FM squelch compared `squelch_dbfs`
> against `PowerDbFS` of the scanner's **whole IQ chunk** — the full
> 2.4 MHz span of an RTL-SDR — not the channel. Any other carrier in the
> span, or the noise floor a `gain: auto` tuner holds near full scale, read
> as "carrier present" on every channel scanned: untoned channels opened
> calls on empty air, toned channels fell through to the CTCSS/DCS detector
> alone, and no threshold short of −1 dBFS changed it. The fix
> (`internal/scanner/conventional/channel_power.go`) is `channelPowerMeter`:
> decimate to ~48 kHz, a ±8 kHz Kaiser channel filter — the tone front
> end's own geometry — then RMS. An on-channel signal reads the same dBFS
> as before (`TestChannelPowerMeterCalibration`), so thresholds keep their
> meaning; everything outside the channel stops counting. Failing-first:
> `TestConvScannerSquelchIgnoresCarrierElsewhereInSpan` (empty channel, a
> −20 dBFS carrier 500 kHz away, −50 dBFS squelch → zero calls). The same
> PR added `scanner.conventional[].gain` (`applyChannelGain`, written before
> each tune). The reporter confirmed both on v1.2.3 across five FM and five
> air-band channels. Still open: the gate is absolute dBFS, so a big gain
> change still moves it.

**Key takeaways**

- **A squelch measures a channel, so it must look at a channel.** RMS over
  the whole span is a statement about the band and the tuner's AGC, not
  about the frequency being scanned.
- **"Tone only" was the correct diagnosis of a different fault.** The gate
  always required power AND tone; the power half was simply never false.
- **Keep the scale when you change the measurement.** The meter reads the
  same dBFS as the old one for an on-channel signal, so every existing
  `squelch_dbfs` survived the change.
- **Absolute dBFS is still a gain-staging trap.** The fix narrows the
  measurement; it does not make the threshold gain-independent. That is
  the noise-quieting squelch, not yet built.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| In-channel power | decimate to ~48 kHz, ±8 kHz Kaiser LPF, `PowerDbFS` | `channel_power.go` (`channelPowerMeter`, `newChannelPowerMeter`) |
| Level selection | AM → C/N meter; FM → channel meter; whole span only with no sample rate | `am_squelch.go` (`squelchMeasure`), `warnSpanPowerSquelch` |
| Front-end geometry shared with tones | `toneRefRateHz` = 48 000, `toneChannelCutoffHz` = 8 000 | `tone_frontend.go` |
| Scan-window gate | power ≥ `squelch_dbfs` first, then the tone detector | `scanner.go` (`scanWindow`) |
| Dwell hysteresis | keep-alive at `squelch_dbfs − squelch_hysteresis_db` | `scanner.go` (`beginDwell`) |
| Per-channel gain | written before the tune; device default for channels without one | `gain.go` (`applyChannelGain`, `GainSetter`), `cmd/gophertrunk/conv_gain.go` |
| Failing-first pins | off-channel carrier ignored; in-channel −40 dBFS opens | `channel_power_test.go` |
| Reference | what a squelch is and the kinds that exist | [squelch]({{ '/reference/squelch/' | relative_url }}) |

## In this post

- **What the reporter saw** — four observations, one of them a bug.
- **A gate that was never false** — `PowerDbFS` over 2.4 MHz.
- **The channel power meter** — the tone front end, reused.
- **Per-channel gain** — the other half of a mixed scan list.
- **On air, and what is still absolute** — confirmed on v1.2.3; dBFS still absolute.

## What the reporter saw

The issue arrived on 3 Oct as a structured field report from an operator
running the scanner "in detail and in real-world scenarios" with one SDR
pinned to it. Four observations: `auto` gain served a mixed list badly,
especially AM; the CTCSS band-pass felt wide; CTCSS/DCS gating "should
require CSQ AND Tone, but in practice behaves as Tone only" — most
noticeable on VHF FM, where "even with very tight squelch settings (−10,
−5, −1, etc.), random noise opens the gate when no solid carrier is
present"; and digital conventional channels could not be scanned.

The fourth — digital conventional channels such as the P25
interoperability frequencies — and the first half of the first, letting
the scanner own more than one SDR, are feature requests and stay open on
the issue. The reporter offered to test anything in real-world scenarios,
which is the offer that closes this part. This post is about the bug.

The third item is the postmortem, and those thresholds are the clue. A
`squelch_dbfs` of −1 is one decibel below a full-scale tone; nothing a
real NFM channel delivers at sane gain sits there. If a gate still opened
above −1 dBFS, the quantity being compared could not have been the
channel's power. The scanner's design, from
[Season 1 Part 16]({{ '/blog/solution-postmortem/from-the-issue-tracker-16-conventional-fm-broker/' | relative_url }}),
chose an IQ-domain squelch deliberately — measurable before any demod
chain spins up — and `PowerDbFS` is its primitive: mean `I² + Q²` over a
chunk, in dB re full scale. It is correct. It was being fed the wrong buffer.

## A gate that was never false

The scanner's IQ stream is the device's **whole span**. An RTL-SDR at
2.4 MS/s delivers 2.4 MHz of spectrum per chunk, centred on the tuned
channel, and `scanWindow` did this with it:

```go
// internal/scanner/conventional/scanner.go (shape, before #1239)
case iq, ok := <-stream:
    powerOK := PowerDbFS(iq) >= ch.SquelchDbFS   // iq = the full 2.4 MHz span
    if !powerOK { continue }
    if det == nil { return true }
    if det.Process(iq) { return true }           // CTCSS / DCS
```

`PowerDbFS(iq)` is the RMS of every carrier in 2.4 MHz plus the noise
floor. On a busy VHF band some repeater is always keyed somewhere in the
span, and with `gain: auto` the tuner's AGC holds the floor itself near
full scale. So `powerOK` was true on essentially every chunk of every
channel, which has two faces. On an **untoned** channel the scanner
opened a call on empty air and recorded hiss until hangtime. On a
**toned** channel the power half of the AND was a constant, which leaves
the CTCSS or DCS detector as the only gate — the reporter's "Tone only".
And because a tone detector on open-squelch noise sees whatever its
Goertzel bins happen to integrate, it opened in short bursts, as
observation two described from the other side.

Lowering `squelch_dbfs` could not help, because the number under test did
not depend on the channel. It is the family of absolute-dBFS gates this
project keeps meeting — the MRC calibration gate an operator once pushed
past by raising gain — with the twist that here the measurement was not
even of the thing being gated.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="A 2.4 MHz spectrum: the scanned channel empty at −90 dBFS, a −20 dBFS carrier 500 kHz away. Whole-span RMS reads about −20 dBFS, above the −50 dBFS squelch line, so the old gate opens; the ±8 kHz meter reads about −90 dBFS and stays shut.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">one 2.4 MS/s chunk, channel at 0 kHz — what PowerDbFS saw vs what the channel carries</text>
  <line x1="40" y1="120" x2="640" y2="120" stroke="var(--fg-muted)"/>
  <g fill="var(--fg-muted)" font-size="8" text-anchor="middle">
    <text x="40" y="134">−1.2 MHz</text><text x="340" y="134">0 (channel)</text><text x="465" y="134">+500 kHz</text><text x="640" y="134">+1.2 MHz</text>
  </g>
  <path d="M40 112 L100 110 L180 113 L260 110 L330 112 L350 112 L420 111 L455 111 L462 40 L468 40 L475 111 L560 113 L640 110" fill="none" stroke="currentColor"/>
  <text x="465" y="34" text-anchor="middle" fill="currentColor" font-size="8">−20 dBFS carrier, 500 kHz away</text>
  <line x1="40" y1="76" x2="640" y2="76" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="44" y="72" fill="var(--accent)" font-size="8">squelch_dbfs = −50</text>
  <path d="M40 150 L640 150" stroke="var(--fg-muted)"/>
  <line x1="40" y1="146" x2="40" y2="154" stroke="var(--fg-muted)"/>
  <line x1="640" y1="146" x2="640" y2="154" stroke="var(--fg-muted)"/>
  <text x="340" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="8">old: PowerDbFS over the whole span ≈ −20 dBFS → open on an EMPTY channel</text>
  <rect x="336" y="104" width="8" height="12" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <path d="M336 182 L344 182" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="198" text-anchor="middle" fill="var(--accent)" font-size="8">new: channelPowerMeter ±8 kHz ≈ −90 dBFS → shut · decimate ÷50 → 48 kHz · LowpassKaiser(63, 8000/48000, 8.6)</text>
</svg>
<figcaption>The old gate integrated 2.4 MHz, so the strong neighbour half a megahertz away was the number compared to squelch_dbfs. The new meter integrates ±8 kHz around the channel and reads the floor.</figcaption>
</figure>

## The channel power meter

The fix reuses geometry the scanner already owned. The tone detectors had
learned in
[Part 3]({{ '/blog/solution-postmortem/issue-tracker-s2-03-radians-per-sample/' | relative_url }})
to decimate to a reference rate and channel-filter before measuring;
`channelPowerMeter` is that front end with a power reading on the end:

```go
// internal/scanner/conventional/channel_power.go (shape)
func newChannelPowerMeter(sampleHz float64) *channelPowerMeter {
    m := int(sampleHz / toneRefRateHz)      // 2.4 MS/s → 50
    if m < 2 { m = 1 } else { pre = dsp.NewResampler(1, m, m*8+1, 8.6) }
    rate := sampleHz / float64(m)
    if fc := toneChannelCutoffHz / rate; fc < 0.45 {  // 8000 / 48000
        chanFilter = filter.NewFIR(filter.LowpassKaiser(63, fc, 8.6))
    }
    …
}
func (m *channelPowerMeter) process(iq []complex64) float64 {
    iq = m.pre.Process(…); iq = m.chanFilter.Process(…)
    if len(iq) == 0 { return m.level }  // tiny chunk: hold, don't read as silence
    m.level = PowerDbFS(iq)
    return m.level
}
```

Three properties were deliberate. **The scale is unchanged**: a carrier
on the channel dominates either measurement, so `TestChannelPowerMeterCalibration`
pins an on-channel −20 dBFS tone to read −21..−19 through the meter —
existing thresholds keep their meaning, and only energy outside ±8 kHz
stops counting. **A chunk too short to yield a decimated sample holds the
last reading** rather than returning −∞, because a tiny chunk reading as
silence would start a hangtime countdown
(`TestChannelPowerMeterHoldsOnTinyChunk`). And the meter is **reset on
every retune**, with the tone detectors, so one channel's filter history
cannot leak into the next channel's first reading.

`squelchMeasure` picks the level function per channel: the AM channels
keep their carrier-to-noise meter from #1219, FM channels get the power
meter, and the whole-span `PowerDbFS` survives only when the scanner has
no sample rate to build a filter from — in which case `warnSpanPowerSquelch`
says so once at construction. `scanWindow` and `beginDwell` both consume
the same function, so the open threshold and the dwell's keep-alive
(`squelch_dbfs − squelch_hysteresis_db`) are measured on the same
quantity.

The regression is the reporter's scene in miniature.
`TestConvScannerSquelchIgnoresCarrierElsewhereInSpan` builds 200 chunks
of 4096 samples at 2.4 MS/s with the scanned channel empty (noise 90 dB
down) and a −20 dBFS carrier 500 kHz away, scans it with `squelch_dbfs: −50`,
and asserts the engine saw **zero** call starts; on the old code the
whole-span RMS reads ~−20 dBFS and a call opens. The no-harm half,
`TestConvScannerSquelchOpensOnInChannelCarrier`, adds a −40 dBFS carrier
2 kHz off centre beside the same strong neighbour and asserts the squelch
opens.

## Per-channel gain

The reporter's first observation shipped in the same PR, because a mixed
scan list needs it once the squelch is honest. One scanner SDR serves
VHF FM beside air-band AM beside UHF, and no single tuner gain suits all
three — `auto` serves AM worst. `scanner.conventional[].gain` takes the same form as
`sdr.devices[].gain` — `"auto"`, or tenths of a dB, so `"280"` is 28 dB —
and is written to the device **before** the channel is tuned:

```yaml
scanner:
  conventional:
    - label: "Tower"
      frequency_hz: 126400000
      mode: am
      gain: "400"        # 40 dB for air band
    - label: "County Fire"
      frequency_hz: 154265000
      squelch_dbfs: -50  # no gain: back to the device's own
```

`applyChannelGain` has three rules. The scanner **only touches the gain
when at least one channel sets one** — a list without per-channel gains
behaves exactly as before. Once any channel does, the scanner owns the
device's gain: a channel without one returns it to
`DefaultGainTenthDB`, the gain the pool applied at open (`convDeviceGain`,
−1 for automatic), so leaving an AM channel at 40 dB puts the FM channels
back where the operator configured them. And an unchanged gain is not
rewritten — `appliedGain` is compared first — because a tuner write can
stall (the #248/#753 class), and a failed write is logged once per value
while the scan carries on at whatever gain the device holds. `convChannelGain` parses with the device-gain parser and
warns when a value looks like whole dB ("28" is 2.8 dB; write "280" or
"28.0"). Pinned by `TestConvScannerAppliesPerChannelGain` (gain lands
before each tune, in order), `TestConvScannerLeavesGainAloneWithoutChannelGains`
and `TestConvScannerGainFailureKeepsScanning`.

## On air, and what is still absolute

Both changes shipped in v1.2.3. The thread's test prescription: a fixed
gain instead of `auto`, `squelch_dbfs` around −50 — expect a *lower*
value than before, since the number is now the channel's — then check
that noise no longer opens a CTCSS/DCS channel. The reporter's reply on 5 Oct is the on-air verdict:
v1.2.3 "resolved the CTCSS/gain/noise situation without degrading the
scanning performance" on five FM and five air-band channels; air band now
gets a higher gain, FM repeaters and intermod-prone frequencies a lower
one, "without the continued open/close noise". The gate and the gain
request are closed by that report; multi-SDR scanning and digital
conventional channels remain open feature requests, and the CTCSS
band-pass question is not touched by this fix.

What the fix does **not** do is make the threshold gain-independent. The
meter reads in-channel power in dBFS, so a 10 dB gain change still moves
every channel's reading by 10 dB. The gain-independent answer is a
noise-quieting squelch — the FM discriminator's variance, the statistic
the DMR receiver's `carrierGate` uses to tell bursts from gaps at any
gain — and it is not built on the conventional path. Until it is, the
[cookbook recipe]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})'s
advice stands: fix the gain first, then set `squelch_dbfs` against what
the channel reads. The companion tutorial series,
[The Conventional Scanner]({{ '/blog/series/conventional-scanner/' | relative_url }}),
walks the whole subsystem.

### How the squelch shaped the Go code

- **One level function per channel, chosen once.** `squelchMeasure`
  returns a closure and its threshold, and both `scanWindow` and
  `beginDwell` call it, so the two gates cannot drift apart.
- **The meter holds rather than lies.** An empty decimated chunk returns
  the previous level; −∞ from `PowerDbFS(nil)` is reserved for the
  constructor's initial state.
- **Reset is part of retune.** `powerMeterFor(idx).reset()` sits beside
  the detector resets in the scan loop.
- **Gain is a device concern, not a broker one.** `Options.Gain` is the
  pool entry's device, because the IQ broker in front of the scanner has
  no gain control.

## Where this goes next

The season closes on P25 and a report whose own clue solved it: every
`call.start` on a UHF site landed on exactly 450.000 MHz — the band plan's
base — and every call timed out at 7 s. Opcode 0x03, the explicit group
voice channel update, was being parsed with opcode 0x00's layout.
[Part 14]({{ '/blog/solution-postmortem/issue-tracker-s2-14-the-0x03-that-read-as-0x00/' | relative_url }})
reads the two layouts byte by byte, pins the real one with literal bit
positions, and sums up what fourteen bugs had in common.

## FAQ

**Why did my CTCSS channel open on noise even with squelch_dbfs at −1?**
Before v1.2.3 the FM squelch compared `squelch_dbfs` against the RMS
power of the SDR's entire span (2.4 MHz on an RTL-SDR), not the channel.
Any carrier in that span, or an auto-gain noise floor near full scale,
kept the power gate open and left the tone detector as the only gate.
`channelPowerMeter` now measures the channel through a ±8 kHz filter.

**Do I need to change squelch_dbfs after the fix?**
Probably lower it. An on-channel signal reads the same dBFS as before, but
an empty channel now reads its own floor rather than the whole band's
power, so a threshold set just above band power may sit far above the
channel. Start around −50 with a fixed gain and adjust from what the
channel actually reads.

**How does per-channel gain on the conventional scanner work?**
`scanner.conventional[].gain` takes `"auto"` or tenths of a dB, like
`sdr.devices[].gain`, and `applyChannelGain` writes it to the scanner SDR
before tuning that channel. Channels without one return to the device's
configured gain. If no channel sets a gain the scanner never touches it,
and an unchanged value is not rewritten.

**Is the conventional squelch now independent of tuner gain?**
No. It measures in-channel power in dBFS, so a gain change still moves
every reading. A noise-quieting squelch based on discriminator variance —
the approach the DMR receiver's `carrierGate` uses — would be
gain-independent and is the natural next step; it is not built on the
conventional path yet.

**Was the #1239 squelch fix verified on air?**
Yes. The reporter ran v1.2.3 on a mixed list of five FM and five air-band
channels with per-channel gains and reported the CTCSS/gain/noise
situation resolved with no loss of scanning performance. The regression
`TestConvScannerSquelchIgnoresCarrierElsewhereInSpan` fails on the old
code and passes on the new.

## Series navigation

**Part 13 of 14** · ←
[Part 12: CGO_ENABLED=0 Is Not Static — purego, libasound and the Termux Build]({{ '/blog/solution-postmortem/issue-tracker-s2-12-cgo-disabled-is-not-static/' | relative_url }})
· Next →
[Part 14: The 0x03 That Read as 0x00 — An Explicit Channel Update Parsed With the Grant Layout]({{ '/blog/solution-postmortem/issue-tracker-s2-14-the-0x03-that-read-as-0x00/' | relative_url }})
