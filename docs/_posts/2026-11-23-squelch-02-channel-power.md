---
title: "Squelch, Part 2: The Squelch That Measured the Whole Span"
description: "For most of its life GopherTrunk's conventional scanner compared squelch_dbfs with the power of its entire 2.4 MHz span, so any carrier in view held every channel open and tone-gated channels ran on the tone alone. Issue 1239's fix measures the channel instead — decimate, a ±8 kHz filter, then RMS — and adds per-channel gain."
category: deep-dives
keywords: scanner squelch dbfs, in-channel power squelch, carrier squelch sdr, squelch opens on empty channel, ctcss tone only squelch, noise bandwidth decibels, per-channel gain scanner, rtl-sdr scan list squelch, channel filter power meter, gophertrunk squelch
tags: [squelch, conventional, channel-power, dbfs, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 2
---

*Part 2 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list that has to work out for itself whether a channel
carries a call — and every gate that decides when a channel opens.
[Part 1]({{ '/blog/deep-dives/squelch-01-no-control-channel/' | relative_url }})
walked the dwell loop and left one edge unexplained: "open". This part is
the oldest test on that edge, the level squelch, and the discovery in
[#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239) that it had
never measured the channel at all.*

> **TL;DR:** Until #1239 the scanner's FM squelch was `PowerDbFS` of every IQ
> chunk it read, and what it reads is the SDR's whole span: 2.4 MHz on an
> RTL-SDR. Any carrier anywhere in that span, or the noise floor an auto-gain
> tuner holds near full scale, read as "carrier present" on every channel.
> Untoned channels opened calls on empty air; toned channels were gated by
> the tone detector alone — the reporter's "Tone only, not CSQ AND Tone".
> `channelPowerMeter` (`internal/scanner/conventional/channel_power.go`)
> measures the channel instead: decimate to about 48 kHz, a ±8 kHz Kaiser
> filter, then the same RMS. An on-channel carrier reads what it always did,
> so thresholds keep their meaning; a carrier 500 kHz away stops counting;
> span-wide noise reads about 22 dB lower. Per-channel `gain` landed in the
> same change. It is still an absolute level, which is Part 3's problem.

**Key takeaways**

- **A power measurement has a bandwidth, and the old one's was the tuner's.**
  `PowerDbFS` sums everything in the stream it is handed; handed the SDR's
  whole span, it reported the loudest thing in view, not the channel.
- **Bandwidth is decibels.** The same receiver noise reads
  10·log10(2.4 MHz ÷ 16 kHz) = 21.8 dB higher across a 2.4 MHz span than in a
  ±8 kHz channel, before a single other carrier is counted.
- **The fix kept the number's meaning.** A carrier on the channel dominates
  either measurement, so it reads the same dBFS before and after and no
  threshold set against a real signal had to move.
- **A level is still a level.** Channel power says how much energy is in the
  channel, not what it is, and it moves with the gain. Narrowing the
  bandwidth fixed the wrong-place half of the problem only.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Old statistic | RMS power of the whole stream, in dB relative to full scale | `PowerDbFS` in `squelch.go` |
| New statistic | decimate to about 48 kHz, ±8 kHz channel filter, then the same RMS | `channelPowerMeter.process` in `channel_power.go` |
| Filter recipe | the decimator and 63-tap Kaiser filter the tone detectors also use | `channelRateFor`, `channelFilterFor`, `toneChannelCutoffHz` |
| Dispatch | chooses the statistic and its threshold for the channel's mode | `squelchMeasure` in `am_squelch.go` |
| Failing-first pin | a carrier 500 kHz away must not open an empty channel | `TestConvScannerSquelchIgnoresCarrierElsewhereInSpan` |
| Calibration pin | on-channel reads the same; a neighbour and a far carrier do not register | `TestChannelPowerMeterCalibration` |
| Live reading | the smoothed level the squelch is comparing, per channel | `level_dbfs` in `GET /api/v1/scanner` (`levelGauge`) |
| Per-channel gain | tuner gain written before the tune; device default restored afterwards | `applyChannelGain` in `gain.go` |

## In this post

- **What squelch_dbfs actually measured** — one correct function, pointed at the wrong stream.
- **Bandwidth is decibels** — why the span sits tens of dB above the channel on noise alone.
- **Measuring the channel** — `channelPowerMeter`, its filter, and its edge cases.
- **The pins** — the failing-first test, and the calibration that kept thresholds valid.
- **Per-channel gain** — why it landed in the same change.
- **What it does not fix** — the reason there is a Part 3.

## What squelch_dbfs actually measured

The whole of the original squelch is one function, and there is nothing
wrong with it:

```go
// internal/scanner/conventional/squelch.go
func PowerDbFS(iq []complex64) float64 {
    if len(iq) == 0 {
        return math.Inf(-1)
    }
    var acc float64
    for _, s := range iq {
        i := float64(real(s))
        q := float64(imag(s))
        acc += i*i + q*q
    }
    mean := acc / float64(len(iq))
    if mean <= 0 {
        return math.Inf(-1)
    }
    return 10 * math.Log10(mean)
}
```

Mean of |s|², in decibels; a unit-amplitude tone reads 0
[dBFS]({{ '/reference/dbfs/' | relative_url }}). The scanner called it on
each chunk of its IQ stream and compared the result with the channel's
`squelch_dbfs`. The defect was in the noun. The scanner's stream is not a
channel. It is everything the tuner digitised: 2.4 MHz of spectrum at an
RTL-SDR's usual rate. The mean of |s|² over that stream is the total power
of all of it — the channel, every other transmitter in view, and all the
noise in between.

Two symptoms followed, and the reporter on #1239 saw both.

**Untoned channels opened on empty air.** A carrier anywhere in the span
that cleared the threshold made *every* channel scanned inside that span
read "carrier present". Scan lists tend to cluster by band, so one busy
repeater could hold its neighbours open as well as itself.

**Toned channels ran on the tone alone.** A channel with a
[CTCSS]({{ '/reference/ctcss/' | relative_url }}) or DCS gate is supposed to
open on carrier *and* tone. With the carrier half stuck true, the conjunction
collapsed to its other term. The reporter's words, preserved in the
regression test's comment: "Tone only, not CSQ AND Tone." That matters
beyond tidiness, because the dwell loop from Part 1 ends a call when either
half goes false, and half of it could no longer go false.

There was a third case the same sentence in `channel_power.go` names: "the
noise floor an AGC tuner holds near full scale." An automatic gain control
raises whatever it receives until the converter is comfortably driven. No
carrier is needed at all; the span's own noise clears the threshold.
[The Analog Edge Part 2]({{ '/blog/tutorials/analog-edge-02-dbfs/' | relative_url }})
called dBFS a headroom meter rather than a signal meter, and this is that
point with a squelch attached.

## Bandwidth is decibels

It is worth doing the arithmetic, because it says how wrong the old number
was even on a perfectly quiet band. Receiver noise is flat across the span,
so its power is proportional to the
[bandwidth]({{ '/reference/bandwidth/' | relative_url }}) it is measured
over. An FM channel is narrow; the meter this post ends with passes ±8 kHz,
16 kHz in all. The span is whatever the sample rate is.

| Sample rate | Span ÷ 16 kHz | Span noise reads higher than channel noise by |
|---|---|---|
| 2.048 MS/s | 128 | 21.1 dB |
| 2.4 MS/s | 150 | 21.8 dB |
| 4.8 MS/s | 300 | 24.8 dB |

Put a real receiver in that table. In the 2.4 MS/s RTL-SDR capture the
scanner's squelch tests run on, empty channels read about −53 dBFS. Flat
noise at that density totals about −31 dBFS across the span: 19 dB over the
default −50 dBFS threshold with nothing transmitting anywhere. Raising
`squelch_dbfs` far enough to clear that
[noise floor]({{ '/reference/noise-floor/' | relative_url }}) raises it
against real signals by the same amount.

A carrier does not dilute the same way. Its power sits in its own few
kilohertz, so a carrier on the channel contributes the same power to a
16 kHz measurement as to a 2.4 MHz one. That asymmetry is what makes the fix
clean: narrowing the measurement removes the noise and the strangers, and
leaves the wanted signal's reading where it was.

## Measuring the channel

`channelPowerMeter` puts a channel filter in front of the same function:

```go
// internal/scanner/conventional/channel_power.go (shape)
func (m *channelPowerMeter) process(iq []complex64) float64 {
    if m.pre != nil { // decimate to ~48 kHz behind an anti-alias filter
        m.scratch = m.pre.Process(m.scratch[:0], iq)
        iq = m.scratch
    }
    if m.chanFilter != nil { // ±8 kHz channel filter
        m.chanBuf = m.chanFilter.Process(m.chanBuf[:0], iq)
        iq = m.chanBuf
    }
    if len(iq) == 0 {
        return m.level // too short to yield a sample: repeat the last reading
    }
    m.level = PowerDbFS(iq)
    return m.level
}
```

The two stages are deliberately the ones the tone detectors already had (the
file comment: "the same decimate-to-~48 kHz + ±toneChannelCutoffHz channel
filter the tone detectors use"):

- **Decimation.** `channelRateFor` takes `m = int(sampleHz / 48000)` — 50 at
  2.4 MS/s, landing on exactly 48 kHz; 42 at 2.048 MS/s, a little above — and
  `dsp.NewResampler(1, m, m*8+1, 8.6)` is a Kaiser low-pass of 8m+1 taps that
  keeps one sample in m
  ([decimation]({{ '/reference/decimation/' | relative_url }})).
- **Channel filter.** `channelFilterFor` builds
  `filter.LowpassKaiser(63, fc, 8.6)` with the cutoff at
  `toneChannelCutoffHz`, 8 kHz. The constant's comment gives the reasoning:
  "an FM channel (±2.5–5 kHz deviation plus Carson margin) passes, a
  12.5/25 kHz neighbour — and the rest of the SDR's band … — does not."

Three details matter in practice. The meter has filter history, so the loop
resets it on every visit; otherwise the previous channel's samples would
leak into this channel's first reading. A chunk too short to produce a
decimated sample repeats the last level instead of reading −∞, because a
stream of tiny chunks would otherwise look like silence and break the
dwell's hangtime accounting (`TestChannelPowerMeterHoldsOnTinyChunk`). And
the meter needs to know the sample rate: `buildChannelPowerMeter` returns
nil without one, `squelchMeasure` falls back to whole-span `PowerDbFS`, and
`New` says so once — `conv: scanner sample rate is zero; squelch_dbfs
measures the whole SDR span instead of each channel`. The daemon passes the
scanner SDR's rate, so in practice that line is for tests and library users,
but the fallback is not silent.

Every mode except AM gets this meter. `squelchMeasure` hands an AM channel
its carrier-to-noise meter and `squelch_cn_db` instead (Part 7); `fm`,
`nfm`, `p25`, `dmr`, `nxdn`, `nxdn48`, `dstar` and `ysf` channels all
compare in-channel power with `squelch_dbfs`. The number is also published:
`level_dbfs` in `GET /api/v1/scanner` is the same reading smoothed over
200 ms, updated only while the scanner is on the channel, which makes it the
honest instrument for choosing a threshold.

<figure class="lab-figure">
<svg viewBox="0 0 680 232" width="680" height="232" role="img" aria-label="Left: a 2.4 megahertz span with an empty channel at the centre, a flat noise floor, and one carrier at minus 20 dBFS sitting 500 kilohertz above the channel. The old squelch measured the power of the whole span; the new one measures only a narrow slot of plus and minus 8 kilohertz around the channel. Right: two meters against a squelch threshold of minus 50 dBFS. The whole-span meter reads minus 20 dBFS and opens; the channel meter reads only noise, about minus 112 dBFS, and stays shut.">
  <text x="250" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">the #1239 regression fixture: one 2.4 MHz span</text>
  <rect x="40" y="26" width="420" height="140" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="48" y="40" fill="var(--fg-muted)" font-size="8">before: PowerDbFS summed all of this</text>
  <line x1="40" y1="150" x2="460" y2="150" stroke="currentColor"/>
  <path d="M40 142 L460 142" fill="none" stroke="var(--fg-muted)"/>
  <text x="48" y="137" fill="var(--fg-muted)" font-size="8">noise, −90 dBFS across the span</text>
  <line x1="337" y1="142" x2="337" y2="58" stroke="currentColor" stroke-width="2"/>
  <text x="343" y="64" fill="currentColor" font-size="8">carrier, −20 dBFS</text>
  <text x="343" y="75" fill="currentColor" font-size="8">500 kHz away</text>
  <rect x="246" y="48" width="8" height="102" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="240" y="60" text-anchor="end" fill="var(--accent)" font-size="8">after: ±8 kHz channel</text>
  <text x="240" y="71" text-anchor="end" fill="var(--accent)" font-size="8">(drawn wider than scale)</text>
  <text x="40" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−1.2 MHz</text>
  <text x="250" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">channel</text>
  <text x="460" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+1.2 MHz</text>
  <line x1="505" y1="46" x2="505" y2="166" stroke="var(--fg-muted)"/>
  <text x="500" y="49" text-anchor="end" fill="var(--fg-muted)" font-size="8">0</text>
  <text x="500" y="169" text-anchor="end" fill="var(--fg-muted)" font-size="8">−120</text>
  <text x="500" y="99" text-anchor="end" fill="var(--accent)" font-size="8">−50</text>
  <line x1="505" y1="96" x2="660" y2="96" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="660" y="90" text-anchor="end" fill="var(--accent)" font-size="8">squelch_dbfs</text>
  <rect x="525" y="66" width="40" height="100" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="545" y="60" text-anchor="middle" fill="currentColor" font-size="8">−20: opens</text>
  <rect x="600" y="158" width="40" height="8" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="620" y="152" text-anchor="middle" fill="var(--accent)" font-size="8">≈ −112: shut</text>
  <text x="545" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">whole span</text>
  <text x="620" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="8">channel</text>
  <text x="340" y="206" text-anchor="middle" fill="var(--fg-muted)" font-size="9">same samples, same function; the only change is which 16 kHz of the 2.4 MHz it is given</text>
  <text x="340" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="8">channel reading = span noise less the 22 dB bandwidth ratio; the far carrier is in the filter's stopband</text>
</svg>
<figcaption>The old squelch read the loudest thing in the tuner's span; the new one reads the channel, where a carrier 500 kHz away contributes nothing.</figcaption>
</figure>

## The pins

The regression test builds the situation the figure draws, and it builds a
*span*, not a channel. `spanChunks` synthesises 2.4 MS/s chunks holding any
number of carriers at given offsets plus white noise at a stated total
power. That matters: the scanner's older tests fed it `loudChunk`, a
constant `0.7+0.7i` — a carrier at exactly zero offset and nothing else —
and a fixture with one signal in it cannot tell a channel meter from a span
meter. It is the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
in its mildest form: the test world was smaller than the real one.

`TestConvScannerSquelchIgnoresCarrierElsewhereInSpan` scans one channel with
`SquelchDbFS: -50`. The channel is empty; span-wide noise is at −90 dBFS; a
carrier of amplitude 0.1, −20 dBFS, sits 500 kHz away. Whole-span power is
about −20 dBFS, 30 dB over the threshold, so the old scanner opened a call.
The test demands zero calls. Its no-harm twin,
`TestConvScannerSquelchOpensOnInChannelCarrier`, keeps the strong far
carrier and adds a −40 dBFS carrier 2 kHz off the channel centre; that one
must open, and its log line shows the meter reading the weak wanted signal
rather than the strong stranger:

```text
conv: squelch opened freq_hz=155000000 label=A mode=fm level_dbfs=-40 squelch_dbfs=-50 quieting_db=60
```

`TestChannelPowerMeterCalibration` pins the scale, which is the part that
protects existing configurations:

| Carrier at −20 dBFS, placed | The meter must read |
|---|---|
| 1.5 kHz from the channel centre | between −21 and −19 dBFS |
| 12.5 kHz away, the adjacent channel | below −50 dBFS |
| 500 kHz away | below −70 dBFS |

The status is **synthetic-verified**: a failing-first test on a synthetic
span, reproducing a symptom reported from a real rig. The project's
engineering notes record no reporter confirmation for this change on its
own, so it does not get a stronger word.

## Per-channel gain

The same change added `gain` to scan-list entries, and the pairing is not a
coincidence. A level threshold means something only at one gain, and one
scanner SDR may serve a list that mixes bands and modes. The header of
`gain.go`: "VHF FM next to air-band AM next to UHF — and no single tuner
gain suits all of them; 'auto' in particular serves AM badly."

```yaml
scanner:
  conventional:
    - label: "Sheriff Repeater"
      frequency_hz: 155895000
      mode: fm
      squelch_dbfs: -48
      gain: "280"          # tenths of a dB, like sdr.devices[].gain
    - label: "Tower"
      frequency_hz: 118700000
      mode: am
      squelch_cn_db: 12
```

`applyChannelGain` runs in the loop just before `SetCenterFreq`
(`TestConvScannerAppliesPerChannelGain` checks the order). Its rules are
conservative:

- **Opt-in.** If no channel sets a gain the scanner never writes one —
  `TestConvScannerLeavesGainAloneWithoutChannelGains` — so existing lists
  behave exactly as before.
- **Return to default.** A channel without its own gain runs at
  `DefaultGainTenthDB`, the device's configured gain, so the list comes back
  to it after leaving a channel that set one.
- **No redundant writes.** A gain equal to the one last applied is skipped.
- **Failure is survivable.** A failed write is logged once per value —
  `conv: per-channel gain write failed; scanning at the device's current
  gain` — and the channel is still tuned.

Two things to know before using it. Once any channel sets a gain, the
scanner owns the device's gain: a value changed through the API is
overwritten at the next channel whose gain differs. And the value is in
tenths of a dB, so `"28"` is 2.8 dB; `convChannelGain` warns when a figure
looks like whole decibels
([The Analog Edge Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
is the longer argument about staging gain by evidence).

## What it does not fix

Channel power answered "where". It did not answer "what", and it is still
measured against full scale. The project's engineering notes on #1239 say
so in one line: "It is still absolute dBFS, so a big gain change still moves
it."

The case that broke it is the subject of Part 3. A board running an AGC
held its span at −12 dBFS across 4.8 MHz. By the table above that is about
−37 dBFS in a ±8 kHz channel: the channel meter did exactly its job, reading
25 dB below the span — and −37 is still 23 dB over a `squelch_dbfs` of −60.
With the level as the only test, noise opens that channel, measured in
exactly the right place.
[The Analog Edge Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
had already stated the rule this violates: a decision gated on an absolute
level is a decision about the gain knob.

### How the whole-span squelch shaped the Go code

- **One definition of "the channel".** `channelRateFor` and
  `channelFilterFor` are shared helpers; the level meter, the tone
  detectors' front end and the noise squelch's calibration all build the
  same 63-tap filter, so they cannot disagree about what is in the channel.
- **The loop does not know its statistic.** `squelchMeasure` returns a
  function and the threshold to compare it with; adding the AM meter later
  changed no line of `scanWindow`.
- **Stateful meters reset per visit.** Anything with filter history is
  cleared when the scanner retunes, and holds its last reading rather than
  inventing silence.
- **Fixtures model the stream, not the signal.** New squelch tests are built
  on `spanChunks`, with something else in the span.

## Where this goes next

A level in the right place still opens on noise when the gain is automatic
or the threshold is under the floor.
[Part 3]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }})
is the gate that does not care about level at all: the FM noise squelch,
which reads 1 on an empty channel at any gain and collapses toward 0 when a
carrier arrives.

## FAQ

**Why did my scanner open on channels with nothing on them?**
Before the #1239 fix, `squelch_dbfs` was compared with the power of the
SDR's whole span, not the channel. Any carrier within the tuner's bandwidth
that cleared the threshold made every channel in that span read "carrier
present", and so did an auto-gain tuner's raised noise floor.

**What does squelch_dbfs measure in GopherTrunk now?**
The RMS power inside the channel: the scanner's IQ is decimated to about
48 kHz, passed through a ±8 kHz Kaiser filter, and measured in dBFS. It
applies to `fm`, `nfm` and the digital scan-list modes; AM channels use
`squelch_cn_db` instead. The live value is `level_dbfs` in
`GET /api/v1/scanner`.

**Why did my CTCSS channel behave like tone-only squelch?**
A tone gate is meant to open on carrier and tone together. While the carrier
test measured the whole span it was true almost all the time, so the tone
detector became the only working gate. With in-channel power the carrier
half can go false again.

**Do I need to change my squelch_dbfs values?**
Not for thresholds set against real signals: a carrier on the channel reads
the same dBFS as before, which `TestChannelPowerMeterCalibration` pins. What
changed is the idle reading, which is roughly 22 dB lower at 2.4 MS/s, so a
threshold that was raised to fight false opens can usually come back down.

**How do I give one scan-list channel its own gain?**
Add `gain` to the channel entry, as `"auto"` or tenths of a dB (`"280"` is
28.0 dB). It is written to the scanner's SDR before that channel is tuned,
and channels without one run at the device's configured gain.

## Series navigation

**Part 2 of 14** · ←
[Part 1: No Control Channel — How a Scan List Decides a Call Exists]({{ '/blog/deep-dives/squelch-01-no-control-channel/' | relative_url }})
· Next →
[Part 3: Noise Squelch — When an AGC Makes dBFS Meaningless]({{ '/blog/deep-dives/squelch-03-noise-squelch/' | relative_url }})
