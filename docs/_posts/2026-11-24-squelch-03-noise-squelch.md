---
title: "Squelch, Part 3: Noise Squelch — When an AGC Makes dBFS Meaningless"
description: "A PlutoSky scan list opened a call 100 ms after start-up and recorded noise for 33 and 41 minutes, because a level threshold screens nothing when receiver noise sits above it. GopherTrunk's FM noise squelch measures discriminator noise against an empty channel's — 1 on noise at any gain, near 0 on a carrier — ANDed with squelch_dbfs."
category: deep-dives
keywords: fm noise squelch, noise operated squelch sdr, squelch under agc, fm quieting squelch, scanner records noise, plutosdr scanner squelch, discriminator noise squelch, squelch dbfs automatic gain, pmr446 sdr scanner, gophertrunk noise squelch
tags: [squelch, conventional, noise-squelch, analog-fm, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 3
---

*Part 3 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to tell it a call exists —
and every gate that decides when a channel opens.
[Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})
moved the level squelch from the SDR's whole span to the channel and ended
on a warning: a level in the right place is still a level. This part is the
report that proved it,
[#1329](https://github.com/MattCheramie/GopherTrunk/issues/1329), and the
gate that reads the same at any gain.*

> **TL;DR:** A PlutoSky scan list (PMR446, `squelch_dbfs: -60`) opened a
> call 100 ms after start-up and recorded noise for 33 and 41 minutes: the
> board's noise sat above the threshold, and the level was the FM squelch's
> only test. `fmNoiseMeter`, in
> `internal/scanner/conventional/fm_noise_squelch.go`, is the noise-operated
> squelch of an FM receiver. It FM-discriminates the channel's decimated,
> filtered IQ, high-passes the result above 5 kHz, and divides by what the
> same chain gives on white noise. That ratio reads 0.98–1.04 on noise at
> any level and rate, about 0.5 / 0.2 / 0.03 at 0 / 4 / 10 dB
> carrier-to-noise, and under 0.001 on real FM handhelds. The channel opens
> at 0.5 and holds to 0.6 over a 20 ms window, ANDed with the level. It
> costs about 25 ms of open latency. It has not been run on a PlutoSky.

**Key takeaways**

- **Under an AGC a dBFS threshold screens nothing.** An automatic gain
  control raises whatever it receives, noise included, to its own target, so
  an empty channel can sit tens of dB over the threshold.
- **An FM discriminator already knows whether a carrier is there.** Its
  output on noise alone is large and wideband whatever the input level; a
  carrier quiets it. Divided by a noise reference, that is a number that
  reads 1 on an empty channel.
- **Two tests, each covering the other's blind spot.** Noise above the
  level is stopped by the ratio; digital silence, which reads perfectly
  "quiet", is stopped by the level.
- **The price is latency.** A channel opens one measurement window, about
  25 ms, after its carrier comes up, where the level alone opened on the
  first chunk.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The statistic | high-passed discriminator power over the same chain's reading on white noise | `fmNoiseMeter.ratio` in `fm_noise_squelch.go` |
| Thresholds | open at or under 0.5, hold to 0.6, over a 20 ms window updated every 10 ms | `fmNoiseOpenRatio`, `fmNoiseHoldRatio`, `fmNoiseBlock`, `fmNoiseBlocks` |
| Calibration | 4 s of fixed-seed white noise through the channel filter, cached per rate | `fmNoiseReference`, `fmNoiseRefs` |
| The AND | opening needs level and quieting; a dwell stays active only with both | `scanWindow`, `beginDwell` in `scanner.go` |
| Regression | real receiver noise 23 dB over the threshold must open nothing | `TestScannerNoiseAboveSquelchOpensNoCall` |
| Live reading | quieting in dB: about 0 on noise, 3 where the squelch opens | `quieting_db` in `GET /api/v1/scanner` |
| A/B switch | leaves the level as that channel's only carrier test | `noise_squelch: false` (`Channel.NoiseSquelchOff`) |

## In this post

- **A call that opened 100 ms after start-up** — the report, and what is not known.
- **What a discriminator does with nothing** — why noise reads the same at every gain.
- **The meter** — discriminate, high-pass, divide by a calibrated reference.
- **What it reads** — the measured scale, noise to handheld.
- **Two tests that cover each other** — the AND with `squelch_dbfs`.
- **What it costs, and what it misses** — latency and the blind spots.

## A call that opened 100 ms after start-up

The reporter was scanning PMR446 on a PlutoSky — an AD936x board of the
[PlutoSDR]({{ '/reference/plutosdr/' | relative_url }}) family, whose native
driver is
[Cookbook Volume 2 Part 13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }}) —
with `squelch_dbfs: -60`. The scanner opened a call 100 ms after start-up on
an empty channel and recorded noise: one recording ran 33 minutes, another
41.

Nothing was broken in the sense Part 2 fixed. The power was being measured
in the channel. It was simply above the threshold. The 17 Nov log shows the
board's [AGC]({{ '/reference/automatic-gain-control/' | relative_url }})
holding the span at −12 dBFS across 4.8 MHz, and Part 2's arithmetic turns
that into about −37 dBFS in a ±8 kHz channel: 23 dB over −60, with no
transmitter anywhere. Once open, the call could not end either. Hangtime
counts time below the threshold, and noise does not go below it.

One thing is **not known**: whether the 18 Nov run still had `gain: auto`.
The gain line was not in the pasted log. The fix does not depend on the
answer, because the symptom has two causes with one shape — an AGC, or a
fixed gain with `squelch_dbfs` set under the receiver's noise floor
([The Analog Edge Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
is about never chasing such a threshold with the gain knob). Either way the
level answers "how is the gain set?" when the scanner needs "is a carrier
there?".
[The Analog Edge Part 13]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
is that rule at length: never gate a decision on absolute dBFS when a
scale-invariant statistic exists. AM channels already had one,
carrier-to-noise (Part 7), for exactly this reason. FM did not.

## What a discriminator does with nothing

FM has had the answer since long before software. An FM
[discriminator]({{ '/reference/quadrature-demodulation/' | relative_url }})
outputs the phase step between successive samples
([SDR in Pure Go Part 6]({{ '/blog/deep-dives/sdr-internals-06-demodulation/' | relative_url }})
has the demodulators):

```go
// internal/scanner/conventional/fm_noise_squelch.go (shape)
for _, s := range iq {
    // arg(z[n] · conj(z[n-1])): radians per sample
    ar := real(s)*real(m.last) + imag(s)*imag(m.last)
    ai := imag(s)*real(m.last) - real(s)*imag(m.last)
    m.disc = append(m.disc, float32(math.Atan2(float64(ai), float64(ar))))
    m.last = s
}
```

Two properties of that line do all the work. First, `atan2` sees only the
*angle* of its argument: scale the IQ by any constant and the output is
unchanged. No gain setting, manual or automatic, can move it. Second, the
angle of noise is random, so on an empty channel the output is large and
spread across the whole band; when a carrier is present its phase dominates
and the noise can only nudge it. That is FM quieting — the reason an FM
receiver hisses on an empty channel and falls silent on a carrier — and a
hardware receiver's noise squelch listens for that hiss above the voice band.

`fmNoiseMeter` is the same circuit. It listens above 5 kHz
(`fmNoiseHighpassHz`): "above speech, CTCSS/DCS and the 1200/1800 Hz data
bursts, so modulation does not count as noise." A talker, a sub-audible
tone or an MDC1200 burst puts energy below that line; only noise, and the
discriminator's own out-of-band products, live above it.

GopherTrunk has gated on a discriminator statistic before:
[DMR End to End Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
holds a simplex DMR receiver's trackers on the discriminator's variance,
because a phase increment is blind to amplitude.

## The meter

The meter does not run its own front end. `channelPowerMeter.process` — the
level meter from Part 2 — hands it the samples it has already decimated to
about 48 kHz and passed through the ±8 kHz channel filter. From there:

1. **Discriminate**, as above.
2. **High-pass** with a 63-tap filter built as "delta minus low-pass" from
   `filter.LowpassKaiser(fmNoiseHighpassTaps, fmNoiseHighpassHz/rate,
   fmNoiseHighpassBeta)` — "about 63 dB down below 3.5 kHz at 48 kHz".
3. **Sum squares** into 10 ms blocks (`fmNoiseBlock`), and take the ratio
   over the last two (`fmNoiseBlocks`): a 20 ms sliding window, updated
   every 10 ms.
4. **Divide by the reference**, `m.ref`.

```go
// internal/scanner/conventional/fm_noise_squelch.go (shape)
r := tot / (float64(fmNoiseBlocks*m.block) * m.ref)
m.bits.Store(math.Float64bits(r)) // ratio(): +Inf until the first full window
```

Step 4 is what makes the number mean something. How much power a
discriminator puts above 5 kHz on pure noise depends on the sample rate and
the channel filter ahead of it. So `fmNoiseReference` *measures* it: four
seconds of fixed-seed white noise through the channel filter the level meter
builds for that rate, then through an uncalibrated meter — "good to about
0.5 %" — cached per rate in `fmNoiseRefs`. The file comment credits the
method to the DMR carrier gate, which calibrates the same way. Below
`fmNoiseMinRateHz` (12 kHz) there is no room for the measured band and the
channel gets no meter.

The result is a ratio with a fixed point: **1 is an empty channel**, at any
gain and any rate, and a carrier pulls it toward 0. Until a full window has
been measured since the last retune, `ratio()` returns +Inf, "so a channel
never opens on a reading the meter has not made yet."

## What it reads

The numbers below are from the test suite and the project's engineering
notes. The receiver noise and the transmissions are real-air captures
committed in the repository; the
[carrier-to-noise]({{ '/reference/carrier-to-noise-ratio/' | relative_url }})
rows are a real handheld and synthetic FM speech with white noise added at a
set level. The clipped tone is the capture
[Rebuilt From the Air Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
is about.

| Input | Ratio |
|---|---|
| RTL-SDR noise at six empty offsets of a 2.4 MS/s capture; air-band receiver noise | medians 0.98–1.04 |
| The same RTL-SDR noise scaled up 40 dB and down 40 dB | median 0.98 both ways |
| White noise from −99 to −11 dBFS, at rates from 16 kHz to 4.8 MS/s | medians 0.98–1.03 |
| Lowest single window in two hours of white noise (720 000 windows) | 0.66 |
| FM at 0 / 4 / 10 dB carrier-to-noise in the channel | about 0.5 / 0.2 / 0.03 |
| P25 Phase 2 H-DQPSK carrier, the least FM-like signal measured | 0.12 |
| Analog test tone with 57 % of its samples at the ADC rail | 0.02 |
| Real NFM handhelds, with and without CTCSS | under 0.001 |

<figure class="lab-figure">
<svg viewBox="0 0 680 196" width="680" height="196" role="img" aria-label="A logarithmic scale of the noise squelch ratio. Real FM handhelds read under 0.001; a clipped tone 0.02; FM at 10, 4 and 0 decibels carrier-to-noise 0.03, 0.2 and about 0.5; an H-DQPSK carrier 0.12. The squelch opens at 0.5 and holds to 0.6. Receiver noise reads 0.98 to 1.04, never below 0.66.">
  <text x="340" y="13" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">noise squelch ratio (log scale): 1 is an empty channel, at any gain</text>
  <line x1="60" y1="100" x2="640" y2="100" stroke="currentColor"/>
  <line x1="60" y1="96" x2="60" y2="104" stroke="currentColor"/>
  <line x1="236" y1="96" x2="236" y2="104" stroke="currentColor"/>
  <line x1="412" y1="96" x2="412" y2="104" stroke="currentColor"/>
  <line x1="587" y1="96" x2="587" y2="104" stroke="currentColor"/>
  <text x="60" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0.001</text>
  <text x="236" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0.01</text>
  <text x="412" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0.1</text>
  <text x="587" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1</text>
  <line x1="534" y1="34" x2="534" y2="100" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="530" y="40" text-anchor="end" fill="var(--accent)" font-size="8">open at 0.5 or less</text>
  <line x1="548" y1="34" x2="548" y2="100" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="552" y="40" fill="var(--fg-muted)" font-size="8">hold up to 0.6</text>
  <line x1="60" y1="60" x2="534" y2="60" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="297" y="54" text-anchor="middle" fill="var(--accent)" font-size="9">carrier present: the squelch can open (if the level also clears)</text>
  <line x1="556" y1="76" x2="640" y2="76" stroke="var(--fg-muted)" stroke-width="1.5"/>
  <text x="598" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">noise: shut</text>
  <circle cx="60" cy="100" r="3" fill="none" stroke="currentColor"/>
  <circle cx="289" cy="100" r="3" fill="none" stroke="currentColor"/>
  <circle cx="320" cy="100" r="3" fill="none" stroke="currentColor"/>
  <circle cx="425" cy="100" r="3" fill="none" stroke="currentColor"/>
  <circle cx="464" cy="100" r="3" fill="none" stroke="currentColor"/>
  <circle cx="534" cy="100" r="3" fill="none" stroke="currentColor"/>
  <circle cx="556" cy="100" r="3" fill="none" stroke="var(--fg-muted)"/>
  <circle cx="587" cy="100" r="5" fill="none" stroke="var(--fg-muted)" stroke-width="1.5"/>
  <line x1="60" y1="104" x2="60" y2="126" stroke="var(--fg-muted)"/>
  <text x="62" y="136" fill="currentColor" font-size="8">real FM handhelds: under 0.001</text>
  <line x1="289" y1="104" x2="289" y2="146" stroke="var(--fg-muted)"/>
  <text x="289" y="156" text-anchor="middle" fill="currentColor" font-size="8">57 % clipped tone: 0.02</text>
  <line x1="320" y1="104" x2="320" y2="126" stroke="var(--fg-muted)"/>
  <text x="322" y="136" fill="currentColor" font-size="8">10 dB C/N: 0.03</text>
  <line x1="425" y1="104" x2="425" y2="146" stroke="var(--fg-muted)"/>
  <text x="425" y="156" text-anchor="middle" fill="currentColor" font-size="8">H-DQPSK: 0.12</text>
  <line x1="464" y1="104" x2="464" y2="126" stroke="var(--fg-muted)"/>
  <text x="466" y="136" fill="currentColor" font-size="8">4 dB C/N: 0.2</text>
  <line x1="534" y1="104" x2="534" y2="166" stroke="var(--fg-muted)"/>
  <text x="534" y="176" text-anchor="middle" fill="currentColor" font-size="8">0 dB C/N: about 0.5</text>
  <line x1="556" y1="104" x2="556" y2="146" stroke="var(--fg-muted)"/>
  <text x="560" y="156" fill="var(--fg-muted)" font-size="8">noise minimum, 2 h: 0.66</text>
  <line x1="587" y1="105" x2="587" y2="126" stroke="var(--fg-muted)"/>
  <text x="590" y="136" fill="var(--fg-muted)" font-size="8">noise: 0.98–1.04</text>
</svg>
<figcaption>Every real transmission measured lands well left of the open threshold, and receiver noise sits at 1 whatever its level; the gap between them is the squelch.</figcaption>
</figure>

Three readings set the thresholds. Noise never came closer than 0.66 in
two hours, so `fmNoiseHoldRatio` at 0.6 is not reached by chance. The open threshold, `fmNoiseOpenRatio` at 0.5, corresponds to about
0 dB carrier-to-noise, and `TestFMNoiseSquelchFollowsCarrierToNoise` pins
the approach: at 4 dB every window is open, at 2 dB the median window is.
Four decibels is under the FM threshold, where speech is not intelligible
anyway, so in the file comment's words the gate "takes nothing an operator
could have heard." And the worst real signal, a linear H-DQPSK carrier that
is barely FM at all, still reads 0.12.

## Two tests that cover each other

The noise squelch does not replace `squelch_dbfs`. It is ANDed with it:

```go
// internal/scanner/conventional/scanner.go (shape)
levelOK := lv >= open
powerOK := levelOK
if nsq != nil {
    r := nsq.ratio()
    powerOK = levelOK && r <= fmNoiseOpenRatio // scan window: open
}
// … and in the dwell:
active := lv >= keepAlive
if active && nsq != nil && nsq.ratio() > fmNoiseHoldRatio {
    active = false // noise again: start the hangtime countdown
}
```

Keeping the level is not conservatism. The ratio has a blind spot of its
own: an input with no noise at all. Digital silence — a stream of zeros, or
a tuner with its gain so low that the converter sees nothing — makes the
discriminator output exactly zero, which reads as the quietest carrier
imaginable. The level stops that case; the ratio stops noise above the
level. The blind spots do not overlap.

The dwell half matters as much as the open half: it ends a call when the
transmitter drops although the channel never falls under `squelch_dbfs`.
`TestScannerNoiseSquelchFollowsARealTransmission` plays real receiver noise,
a real NFM handheld, then noise again, all above the threshold: exactly one
call, opened on the transmission and ended by hangtime after it. With
`NoiseSquelchOff` the same stream opens on the first chunk of noise and is
still open when the test stops looking — the 41-minute recording in
miniature.

The headline regression, `TestScannerNoiseAboveSquelchOpensNoCall`, rebuilds
the report from real air: RTL-SDR noise raised to −37 dBFS in the channel,
an `nfm` channel at 446.00625 MHz with `SquelchDbFS: -60`. No call may open,
and the scanner must explain itself once per channel (the first five are
named, then one closing line):

```text
conv: channel is above squelch_dbfs with no carrier on it (noise, or a signal that is not FM); the noise squelch is keeping it shut. squelch_dbfs is below the noise floor at this gain, or the gain is automatic, so the level is not screening this channel
```

It carries `level_dbfs`, `squelch_dbfs` and `quieting_db`. Quieting is the
ratio in an FM receiver's units, −10·log10(ratio): about 0 dB on noise, 3 dB
at the open threshold, "15 and up on a usable signal". The same field is in
`GET /api/v1/scanner` and on an FM channel's `conv: squelch opened` line.

## What it costs, and what it misses

**Latency.** The level opened on the first chunk. The ratio needs its
filters filled and one full window: `fmNoiseLatency`, 25 ms, which
`TestFMNoiseSquelchLatency` holds to within one chunk at four chunk sizes.
Three pieces of the loop were adjusted so that 25 ms is all it costs:

- A scan window that ends before the meter's first reading gets one 60 ms
  extension (`fmNoiseSettle`) — but only if the level cleared.
- On a wideband conventional tap, where the voice chain can start at the
  transmission's first sample, the onset mark is back-dated by the latency
  (`streamPos.lead`), so the recording still starts where the carrier did.
- Data decoders and the tone detector keep running from the level alone,
  "as before the noise squelch: a burst's first bits and a tone's first
  block arrive before the noise squelch's window has filled."

One existing scanner test fed a single chunk and expected an open; a
carrier now takes a window to prove itself. The older fixture `loudChunk`, a
constant value, still passes because a DC carrier is perfectly quiet; noise
used as a stand-in "signal" would not.

**Blind spots.**

- **Wide FM.** A broadcast station's ±75 kHz deviation sweeps through the
  ±8 kHz channel and reads about 1, like noise; it never opens, steadily,
  and ±12 kHz still opens in every window
  (`TestFMNoiseSquelchOnFMWiderThanTheChannel`). The scan list has no
  wide-FM mode.
- **A `p25` channel without `nac`.** The noise squelch is built only for
  `fm` and `nfm`. Digital channels rely on their call gates, and a `p25`
  entry with no NAC has none: it still opens on the level alone. The remedy
  is `nac: "F7E"` — any NAC, but only P25 (Part 8).
- **Something that is not FM on an `fm` channel.** That is what
  `noise_squelch: false` is for; it is also the A/B switch when the gate
  itself is suspected.

**Status.** The statistic is measured on real air — an RTL-SDR's noise, an
air-band receiver's, real handhelds — and the scan-list regression replays
real captures. None of it is from the reporter's board. The fix has **not
been run on a PlutoSky**; until the reporter's run lands it stays
on-air-gated.

### How the noise squelch shaped the Go code

- **The reference is measured, not derived.** `fmNoiseReference` pushes
  fixed-seed noise through the real filter taps, so a change to the channel
  filter recalibrates the squelch.
- **The meter rides the level meter.** One decimator and one channel filter
  feed both tests, and `channelPowerMeter.reset` clears both.
- **The loop separates `levelOK` from `powerOK`.** Extensions and data
  decoders key off the first; opening keys off the second.

## Where this goes next

Both squelch tests assume the channel filter sees the channel and nothing
the tuner made up. A zero-IF tuner makes things up exactly there: a DC spur,
an image, and on an overloaded ADC a tone at four times the carrier offset.
[Part 4]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }})
moves the LO off the channel, and shows why the obvious offset is the one
that must never be used.

## FAQ

**Why does my SDR scanner record noise on an empty FM channel?**
Because receiver noise in the channel is above `squelch_dbfs`: the tuner is
on automatic gain, which raises noise to its own target level, or the
threshold is below the noise floor at the current gain. With the level as
the only test, the channel opens and never closes.

**What is a noise squelch, and how is it different from squelch_dbfs?**
`squelch_dbfs` compares the power in the channel with a fixed level. The
noise squelch measures how noisy the FM discriminator's output is above
5 kHz, relative to an empty channel: 1 on noise at any gain, near 0 on a
carrier. `fm` and `nfm` scan-list channels need both. It is on by default
and has nothing to set; `noise_squelch: false` turns it off for a channel.

**Does the noise squelch make weak signals harder to hear?**
It opens from about 0 dB carrier-to-noise in the channel, and at 4 dB every
measurement window is open — below the FM threshold, where speech is not
intelligible. What it adds is about 25 ms before a channel opens.

**Is the noise squelch verified on a PlutoSDR?**
No. It was measured on real-air captures from other receivers and pinned by
a regression that replays the reported condition from real noise, but it has
not been run on a PlutoSky.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Squelch That Measured the Whole Span]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }})
· Next →
[Part 4: Offset Tuning — The Tone a Clipped ADC Adds]({{ '/blog/deep-dives/squelch-04-offset-tuning/' | relative_url }})
