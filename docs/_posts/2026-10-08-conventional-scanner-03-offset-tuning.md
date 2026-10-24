---
title: "The Conventional Scanner, Part 3: Offset Tuning — Why the LO Sits Below the Channel"
description: "Why GopherTrunk's conventional scanner no longer tunes the SDR onto the channel: a zero-IF tune of an overloaded ADC puts a tone at four times the carrier offset into the audio, so convScannerFrontEnd parks the LO below the channel, mixes it back, and searches for an offset whose DC spur, image and clipping products all miss the channel."
category: tutorials
keywords: sdr offset tuning analog fm, zero-if dc spur image clipping, tone at four times carrier offset, lo_offset_hz scanner, dc_avoid fs/4 clipping product, convScannerFrontEnd nco mix, rtl-sdr overload whistle fm audio, adc rail pinned warning, issue 1184 kenwood whistle, gophertrunk conventional scanner
tags: [conventional-scanner, offset-tuning, zero-if, clipping, analog-fm, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 3
---

*Part 3 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's `scanner.conventional` scan list.
[Part 2]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }})
verified every key of a scan-list entry and ended on the one the scanner
never tunes to literally: `frequency_hz`. This part is why. Since
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) the LO
sits hundreds of kilohertz **below** the channel and an NCO mixes the
channel back to baseband, because a zero-IF tune of an overloaded ADC puts
a tone at exactly four times the carrier offset into the audio. The
postmortem of that diagnosis is
[A Tone at Four Times the Offset]({{ '/blog/solution-postmortem/issue-tracker-s2-02-tone-at-four-delta/' | relative_url }});
this part is the operator's view of the fix in `cmd/gophertrunk/conv_offset.go`.*

> **TL;DR:** Tuned on-channel, three front-end artefacts land inside an
> analog FM channel at frequencies set by the residual carrier offset δ: the
> DC spur beats at δ, the I/Q image at 2δ, and per-axis ADC clipping — which
> squares the constellation — emits a tone at **4δ** and its harmonics. The
> reporter's Kenwood captures were rail-pinned (|IQ| ∈ [1.0, 1.414]) and
> carried that tone 18–25 dB above the voice floor (NX-5000 δ≈−420 Hz →
> 1676 Hz; NX-300 δ≈−850 Hz → 3381 Hz). `convScannerFrontEnd` now tunes the
> LO `offsetHz` below the channel and `mixToChannel` shifts it back with a
> `dsp.NCO`; `pickClipSafeLOOffsetHz` searches 150 kHz..0.35·fs in 1 kHz
> steps for the offset whose DC spur, image and `4k·offset` products
> (k ≤ 6, each spread by (4k+1)×5 kHz) stay furthest outside a ±12.5 kHz
> channel — 518 kHz at 2.4 MS/s. `fs/4`, the `dc_avoid` default, is the
> worst choice: 4·(fs/4) ≡ 0 mod fs puts the 3rd-order product on the
> channel. `scanner.lo_offset_hz` is 0 auto / > 0 pin / < 0 off, and the
> raw stream is watched for rail pinning (`conv: front end overloaded`).

**Key takeaways**

- **The whistle is arithmetic, not RF.** Per-axis clipping makes the phase
  error periodic in π/2, so the discriminator emits 4·|δ|.
- **Moving the LO moves every product.** DC spur, image and clipping lines
  all sit at multiples of the offset; put the offset far enough away and
  they all leave the channel filter.
- **The offset is searched, not assumed.** `fs/4` aliases the 3rd-order
  product exactly onto the channel — the simulated audio is as wrecked as
  on-channel.
- **The mix hides the artefacts; it does not fix the overload.** A clipped
  front end still loses weak signals, so the raw stream is watched and the
  WARN says lower the gain.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| LO placement | `SetCenterFreq(hz)` tunes the inner device to `hz − offsetHz` | `convScannerFrontEnd.SetCenterFreq` (`conv_offset.go`) |
| Mix-back | `dsp.NewNCO(offsetHz, fs)` shifts +offset → DC on every chunk | `mixToChannel`, also `convScanVoiceSource.StreamIQ` for the FM chain |
| Offset choice | max over 150 kHz..0.35·fs of the minimum product clearance | `pickClipSafeLOOffsetHz`, `loOffsetClearanceHz`, `TestPickClipSafeLOOffset` |
| Config | `scanner.lo_offset_hz`: 0 auto, > 0 pin (≤ 35 % fs), < 0 disable | `convScannerLOOffsetHz`, `TestConvScannerLOOffsetConfig` |
| Overload watch | rail-pinned fraction > 0.002 over 1 s → WARN, ≤ once per 5 min | `observeRaw`, `siglab.CountClipped`, `TestConvScannerFrontEndWarnsOnOverload` |
| Failing-first pin | on-channel spur > −30 dB, SINAD < 10 dB; offset: spur < −45, SINAD > 40; fs/4 SINAD < 10 | `TestConvScannerOffsetTuningRemovesClippingWhistle` |

## In this post

- **Three products of a zero-IF tune** — δ, 2δ and 4δ, and the captures that showed them.
- **The front end** — `convScannerFrontEnd`, the NCO, and the FM chain's matching mix.
- **Choosing the offset** — the clearance score and what it picks at common rates.
- **Why fs/4 is the worst offset** — and its relation to `dc_avoid`.
- **The overload watch** — what the WARN measures and what it asks you to do.
- **What is verified** — simulation, the live AM runs, and the gap.

## Three products of a zero-IF tune

An SDR tuned exactly onto an analog FM channel is a zero-IF receiver, and
a real transmitter is never exactly where the LO thinks the channel is —
the residual offset δ is a few hundred hertz, *in the audio band*. Three front-end artefacts then land in the channel at frequencies
set by δ. The DC spur (LO leakage, ADC offset) beats with the carrier at δ.
The I/Q-imbalance image of the carrier sits at −δ and beats at 2δ. And the
one that dominated the reporter's captures: an overloaded 8-bit ADC clips I
and Q *independently*, squaring the unit-circle constellation. The phase
error that introduces is periodic in the carrier phase with period π/2, so
the FM discriminator emits a tone at 4·|δ| plus harmonics.

The `conv_offset.go` header records the measurements. Every Kenwood file
from the #1184 reporter was pinned to the rail — |IQ| ∈ [1.0, 1.414], a
square — and the demodulated audio carried a tone 18–25 dB above the voice
floor at exactly 4× each file's carrier offset: the NX-5000 at δ≈−420 Hz
produced 1676 Hz, the NX-300 at δ≈−850 Hz produced 3381 Hz. The unclipped
Radtel file at −12 dBFS showed no such line. The operator's bench test said
the same from the other side: an unmodulated −60 dBm carrier exactly
on-channel was noisy, ±3.25 kHz off it was clean. SDR# and SDR++, which
tune the LO off the VFO, never show it. The clipping itself is
[The Analog Edge Part 4]({{ '/blog/tutorials/analog-edge-04-clipping-overload-intermod/' | relative_url }})'s
subject; what is specific here is that zero-IF tuning puts its products in
the one place an FM discriminator cannot ignore.

## The front end

The fix mirrors the bench test. `convScannerFrontEnd` wraps whatever the
scanner drives — the device's iqtap broker, or a bare device — and changes
two calls:

```go
// cmd/gophertrunk/conv_offset.go (shape)
func (f *convScannerFrontEnd) SetCenterFreq(hz uint32) error {
    f.centerHz = hz                              // the channel, not the LO
    return f.inner.SetCenterFreq(hz - uint32(f.offsetHz))
}

func (f *convScannerFrontEnd) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
    in, err := f.inner.StreamIQ(ctx)
    return mixToChannel(ctx, in, f.offsetHz, f.rateHz, f.observeRaw), nil
}
```

`mixToChannel` builds `dsp.NewNCO(offsetHz, fs)` once per stream and, for
each chunk, hands the observer the raw samples first and then mixes into a
fresh buffer (the broker's fan-out may share the chunk with other
subscribers). With `offsetHz == 0` and no observer it returns the input
untouched. After the mix the channel sits at DC and the artefacts
sit at −offset (DC spur), −2·offset (image) and ±4k·offset (clipping
products), all wrapped modulo the sample rate — far outside the ±8 kHz
channel filter of Part 4's power meter and Part 5's tone detector, provided
the offset was chosen so none of them alias back.

The FM voice chain has to see the same stream. Part 1's
`convScanVoiceSource` subscribes to the broker, which delivers the channel
at +offset, so its `StreamIQ` runs the same `mixToChannel` with the offset
the daemon recorded in `scannerLOOffsets[serial]`. The daemon logs the
decision once at start-up:

```text
INF conv: scanner LO offset tuning serial=00000001 lo_offset_hz=518000 mode=auto sample_rate_hz=2400000
```

`mode` is `auto`, `configured`, `disabled (scanner.lo_offset_hz < 0)`,
`scanner.lo_offset_hz exceeds 35% of the sample rate` or
`no room for an LO offset at this sample rate` — the last two fall back to
on-channel tuning. `TestConvScannerFrontEndTunesBelowChannel` pins the
direction: a 200 kHz offset on 447.100 MHz puts the LO at 446.900 MHz.

## Choosing the offset

`pickClipSafeLOOffsetHz(fs)` scans offsets from `convOffsetMinHz`
(150 kHz — the DC spur must sit well inside the stopband of the FM chain's
decimating FIR, whose transition band at ~2.4 MS/s is ~100 kHz wide) to
`convOffsetMaxFrac · fs` (0.35, keeping the channel clear of the tuner's
band-edge roll-off) in `convOffsetStepHz` = 1 kHz steps, and keeps the one
with the best `loOffsetClearanceHz` score. The score is the minimum, over
every product, of how far the product's *modulated extent* stays outside a
±`convOffsetChanHalfHz` (12.5 kHz) channel after wrapping into
(−fs/2, fs/2]:

```go
// cmd/gophertrunk/conv_offset.go (shape)
clear := func(f, spread float64) float64 {
    return math.Abs(wrapHz(f, fs)) - convOffsetChanHalfHz - spread
}
score := clear(off, 0)                          // DC spur, unmodulated
score = math.Min(score, clear(2*off, convOffsetDevHz)) // image, 1× deviation
for k := 1; k <= convOffsetMaxOrder; k++ {       // clipping products, k ≤ 6
    spread := float64(4*k+1) * convOffsetDevHz  // (4k+1)× 5 kHz
    score = math.Min(score, clear(float64(4*k)*off, spread))
}
```

The (4k±1)th-order product carries (4k±1)× the deviation, and
`convOffsetDevHz` = 5 kHz covers wideband 25 kHz FM, so the k = 6 product
is allowed a 125 kHz spread. Negative means a product overlaps the channel;
ties go to the smaller offset.
Reproducing the search outside the test gives 518 kHz at 2.4 MS/s (minimum
clearance 290.5 kHz, set by the k = 1 product wrapping to −328 kHz), 443 kHz
at 2.048 MS/s, 647 kHz at 3 MS/s, and 0 — no room — at 250 kS/s, where
0.35·fs is below the 150 kHz floor.
`TestPickClipSafeLOOffset` pins that every common RTL/Airspy rate from
960 kS/s to 10 MS/s gets a positive-clearance offset that is never fs/4.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Two frequency axes after mix-back at 2.4 megasamples per second. Top, the automatic 518 kilohertz offset: DC spur, image and six clipping products all wrap clear of the channel at zero, minimum clearance 290 kilohertz. Bottom, an fs over 4 offset of 600 kilohertz: the four-times clipping product wraps to exactly zero and lands on the channel.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">after mix-back, fs = 2.4 MS/s: where the products land</text>
  <text x="20" y="40" fill="var(--accent)" font-size="9">auto 518 kHz</text>
  <line x1="40" y1="70" x2="640" y2="70" stroke="var(--fg-muted)"/>
  <text x="40" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">−1.2 MHz</text>
  <text x="340" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">0</text>
  <text x="640" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">+1.2 MHz</text>
  <rect x="337" y="56" width="6" height="14" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="340" y="52" text-anchor="middle" fill="var(--accent)" font-size="8">channel ±12.5 k</text>
  <line x1="210.5" y1="60" x2="210.5" y2="70" stroke="currentColor" stroke-width="2"/>
  <text x="210" y="98" text-anchor="middle" fill="currentColor" font-size="8">DC −518 k</text>
  <line x1="81" y1="60" x2="81" y2="70" stroke="currentColor"/>
  <text x="81" y="98" text-anchor="middle" fill="currentColor" font-size="8">image −1036 k</text>
  <rect x="251" y="62" width="12" height="8" fill="var(--fg-muted)"/>
  <text x="258" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">k=1 −328 k</text>
  <rect x="166" y="62" width="22" height="8" fill="var(--fg-muted)"/>
  <text x="177" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">k=2 −656 k</text>
  <rect x="78" y="62" width="32" height="8" fill="var(--fg-muted)"/>
  <rect x="595" y="62" width="40" height="8" fill="var(--fg-muted)"/>
  <text x="612" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">k=4 +1088 k</text>
  <rect x="505" y="62" width="50" height="8" fill="var(--fg-muted)"/>
  <text x="530" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="8">k=5 +760 k</text>
  <rect x="417" y="62" width="62" height="8" fill="var(--fg-muted)"/>
  <text x="448" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">k=6 +432 k</text>
  <path d="M263 120 L337 120" stroke="var(--accent)" stroke-dasharray="3 2"/>
  <text x="300" y="132" text-anchor="middle" fill="var(--accent)" font-size="8">min clearance 290.5 kHz</text>
  <text x="20" y="160" fill="currentColor" font-size="9">fs/4 = 600 kHz</text>
  <line x1="40" y1="190" x2="640" y2="190" stroke="var(--fg-muted)"/>
  <rect x="337" y="176" width="6" height="14" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="190" y1="180" x2="190" y2="190" stroke="currentColor" stroke-width="2"/>
  <text x="190" y="204" text-anchor="middle" fill="currentColor" font-size="8">DC −600 k</text>
  <line x1="640" y1="180" x2="640" y2="190" stroke="currentColor"/>
  <text x="620" y="204" text-anchor="middle" fill="currentColor" font-size="8">image +1200 k</text>
  <rect x="328" y="182" width="24" height="8" fill="currentColor"/>
  <text x="340" y="220" text-anchor="middle" fill="currentColor" font-size="8" font-weight="bold">4·600 k ≡ 0 mod 2400 k: the 3rd-order clipping product lands ON the channel</text>
</svg>
<figcaption>At 518 kHz every product wraps somewhere else; at fs/4 the first clipping product wraps to exactly zero.</figcaption>
</figure>

## Why fs/4 is the worst offset

The obvious offset is a quarter of the sample rate: far from DC, far from
the band edge, and the default `dc_avoid_offset_hz` auto-selects for the
`sdr.devices[].dc_avoid` mode that control and voice SDRs use to dodge the
DC spike ([#402](https://github.com/MattCheramie/GopherTrunk/issues/402)).
For an *unclipped* signal it is fine. For a clipped one it is the single
worst choice: the k = 1 clipping product sits at 4·offset, and
4·(fs/4) = fs ≡ 0 mod fs — the 3rd-order product aliases exactly onto the
channel. The clearance score for 600 kHz at 2.4 MS/s is −137.5 kHz, and
`TestPickClipSafeLOOffset` asserts it is negative.

`TestConvScannerOffsetTuningRemovesClippingWhistle` is the failing-first
regression. Its `clippingFrontEnd` models the reporter's rig: a 1 kHz tone
at 2.5 kHz deviation on a carrier 420 Hz off the channel, overdriven 3× into
an 8-bit quantiser that clips I and Q independently, plus a DC spur. Through
a `convScannerFrontEnd` and a minimal FM chain it measures the tone at
4·|δ| relative to the wanted tone and the audio SINAD. On-channel
(`lo_offset_hz: -1`) the spur must be above −30 dB and SINAD below 10 dB or
the fixture no longer reproduces the damage; at the auto offset the spur
must be below −45 dB and SINAD above 40 dB; at a pinned 600 kHz (fs/4)
SINAD must be below 10 dB. The project note from the fix records the
simulated numbers: 0.5 dB SINAD on-channel *and* at fs/4, 56 dB at the
searched 518 kHz. The `dc_avoid` path on control and voice SDRs keeps its
fs/4 default and the same exposure — deliberately not changed.

## The overload watch

Mixing the artefacts out of the channel does not make the front end
healthy: a rail-pinned ADC still loses weak signals, and a quiet recording
can hide that. `convScannerFrontEnd.observeRaw` therefore sees every chunk
*before* the NCO, counts rail-pinned samples with `siglab.CountClipped`,
and over each `convClipWindow` (1 s) computes the clipped fraction. Above
`convClipWarnFrac` = 0.002 — the same threshold the ccdecoder and widebandt2
engines use ([#402](https://github.com/MattCheramie/GopherTrunk/issues/402),
[#749](https://github.com/MattCheramie/GopherTrunk/issues/749)) — it logs,
at most once per `convClipWarnGap` (5 min):

```text
WRN conv: front end overloaded — IQ pinned to the ADC rail; a strong nearby transmitter is clipping the SDR, which distorts analog FM audio (a whistle/tone at 4x the carrier offset when tuned on-channel). Reduce gain or add attenuation (do NOT raise gain). issue #1184 serial=… channel_hz=… clipped_fraction=… lo_offset_hz=…
```

The instruction is the whole point: the Kenwood files were rail-pinned, no
software stage recovers that, and raising gain to "get more signal" is
exactly backwards. `clipped_fraction` is the number to drive the gain down
against — Part 4's per-channel `gain` key exists partly so a strong local
channel can run at a lower gain than the rest of the list.
`TestConvScannerFrontEndWarnsOnOverload` feeds three clean windows (no
WARN) then six railed ones and demands exactly one line. With
`lo_offset_hz` disabled the watch still runs: `offsetHz == 0` keeps the
legacy on-channel tuning with only the overload observer attached.

## What is verified

The arithmetic and the search are pinned by the tests above, and the
failing-first fixture reproduces the reporter's 4δ tone before asserting
its removal. The mechanism has run live: the 28–29 Sep AM runs of Part 9
went through this front end — the reporter's voice-IQ debug capture is the
chain's input *after* `convScanVoiceSource` mixed the offset out — with
clear audio on frequency. What the repository does not record is the FM
case closing the loop: a Kenwood run on a fixed build with the whistle gone.
Per the project's
[#764/#771 rule]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
a green simulation is not that confirmation, so the status is
simulation-pinned and live-exercised, with the clipped-FM confirmation open.

### How offset tuning shaped the Go code

- **The wrapper owns the channel, the inner device owns the LO.**
  `centerHz` records what the operator asked for; `inner.SetCenterFreq`
  gets the subtraction, so every log line can report the channel.
- **One mix function, two callers.** `mixToChannel` serves the scanner's
  stream and the FM chain's subscription, so the two can never disagree
  about the offset.

## Where this goes next

With the channel sitting at DC and the artefacts parked half a megahertz
away, the next question is what the squelch actually measures. Until
[#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239) it measured
the whole 2.4 MHz span — so any other carrier in it, or the noise floor an
auto-gain tuner holds near full scale, held every channel open.
[Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})
reads `channel_power.go`: decimate, ±8 kHz filter, RMS, and the per-channel
`gain` that landed with it.

## FAQ

**Why is there a whistle in my analog FM audio from an SDR?**
If the tone sits at four times the carrier offset, your front end is
clipping and the SDR is tuned on-channel: per-axis ADC clipping makes the
discriminator emit 4·|δ|. GopherTrunk's conventional scanner now tunes the
LO `scanner.lo_offset_hz` below the channel (auto 518 kHz at 2.4 MS/s) and
mixes back, and warns `conv: front end overloaded` — reduce gain.

**What does scanner.lo_offset_hz do?**
0 (default) lets `pickClipSafeLOOffsetHz` choose an offset whose DC spur,
image and clipping products all clear a ±12.5 kHz channel; a positive value
pins the offset in Hz (at most 35 % of `sdr.sample_rate`, else it falls back
to on-channel); a negative value disables offset tuning. The start-up line
`conv: scanner LO offset tuning` reports the result and its `mode`.

**Why not use sample_rate/4 like dc_avoid?**
Because 4·(fs/4) ≡ 0 mod fs: the first clipping product aliases exactly onto
the channel, and the simulated audio is as damaged as on-channel (SINAD
0.5 dB vs 56 dB at the searched offset). `TestPickClipSafeLOOffset` asserts
the picker never returns fs/4. The `dc_avoid` default on control and voice
SDRs is unchanged and keeps that exposure.

**Does offset tuning fix front-end overload?**
No. It moves the overload's audible products out of the channel; a
rail-pinned ADC still loses weak signals. The front end watches the raw
stream and WARNs when `clipped_fraction` exceeds 0.002 over a 1 s window,
once per 5 min. The fix is lower gain or attenuation.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Scan List as Config — Channels, Modes, Hangtime, Priority]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }})
· Next →
[Part 4: Squelch on the Channel, Not the Span — The In-Channel Power Meter]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})
