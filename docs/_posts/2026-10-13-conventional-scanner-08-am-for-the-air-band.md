---
title: "The Conventional Scanner, Part 8: AM for the Air Band — The Carrier as the AGC Reference"
description: "How mode: am on a scan-list channel becomes an am-conv grant and the composer's AM chain — a ±4.5 kHz channel filter, demod.AM's envelope-over-tracked-carrier detector, a 3 kHz low-pass and amAudioScale 0.4 with no de-emphasis or equalizer — and why the AM squelch is a carrier-to-noise meter from a Welch spectrum, never dBFS, so a gain change cannot move it."
category: tutorials
keywords: sdr air band scanner am, rtl-sdr airband am demodulation, am envelope detector carrier reference, am squelch carrier to noise, squelch_cn_db, 8.33 khz channel carrier frequency, conventional scanner mode am, welch psd squelch, gophertrunk conventional scanner
tags: [conventional-scanner, am, air-band, squelch, composer, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 8
---

*Part 8 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's conventional (non-trunked) scanner, `scanner.conventional`, from
the dwell loop to a verified kitchen-sink config.
[Part 7]({{ '/blog/tutorials/conventional-scanner-07-fm-voice-chain/' | relative_url }})
walked the composer's FM chain from the discriminator to int16 PCM. An FM
discriminator cannot hear amplitude modulation at all — a lone AM carrier
through it is silence — so until
[#1219](https://github.com/MattCheramie/GopherTrunk/issues/1219) the VHF air
band could be tuned and squelched but recorded nothing. This part adds the
mode, the chain and the squelch that fixed that, and explains why the squelch
had to be a ratio rather than a level.*

> **TL;DR:** `mode: am` on a `scanner.conventional` entry is validated by
> `ValidMode` (`am_squelch.go`), produces a synthetic grant with
> `Protocol: "am-conv"` (`conventionalProtocol`), and `classifyVoiceKind`
> routes it to `voiceKindAM` → `runFMChain(…, am=true)`. The AM branch
> swaps in `newAMChannelFilter` (181 Kaiser taps, ±`amChannelCutoffHz` =
> 4500 Hz: keeps a 3 kHz DSB voice signal with a carrier a kHz or two off,
> rejects an 8.33 kHz neighbour whose nearest sideband starts ~5.3 kHz
> away), `demod.AM` (envelope ÷ a 100 ms-tracked carrier − 1 = modulation
> depth, clamped ±1.5), a 127-tap 3 kHz audio low-pass and `amAudioScale`
> = 0.4 so an AM channel records at the level of a ±3 kHz NFM one; the FM-only
> stages — de-emphasis and the CMA equalizer — are skipped, the tone
> high-pass, AGC, resampler and squelch-tail mute are shared. The squelch is
> `amCNMeter`: decimate to ~48 kHz, a Welch average of 8 × 256-point Hann
> spectra (~188 Hz bins), the strongest bin within ±4.5 kHz over the 20th
> percentile of the bins within ±12 kHz. Noise reads ~3–7 dB at any level;
> `squelch_cn_db` defaults to `DefaultAMSquelchCNDb` = 12 and `squelch_dbfs`
> is ignored. On-air verified 29 Sep: "very good and clear."

**Key takeaways**

- **An AM carrier is its own gain reference.** `demod.AM` emits
  `|z|/C − 1`: the output depends on how deeply the transmitter modulates,
  never on how strongly it arrives — the AGC an FM chain has to bolt on
  afterwards is built into the detector.
- **The squelch is a ratio because a level would be a gain trap.** Both the
  carrier bin and the noise floor scale with gain, so C/N reads the same at
  any front-end setting; `TestAMSquelchIgnoresAbsoluteLevel` opens at
  −85 dBFS and stays shut on noise at −17 dBFS.
- **`frequency_hz` is the carrier, not the channel name.** An 8.33 kHz
  "channel" like 118.005 is the 118.000 MHz carrier; the filter is centred
  on what you type.
- **Two carriers in one discriminator beat.** That is why the FM-chain
  counterfactual in the tests uses a lone carrier — and why an AM channel
  through the old chain was not silence but noise.

## Cheat sheet

| Line / key | Meaning | Healthy / worry when |
|---|---|---|
| `mode: am` | envelope detection + C/N squelch; protocol `am-conv` | `ValidMode`, `conventionalProtocol` (`am_squelch.go`) |
| `squelch_cn_db` | carrier-to-noise open threshold in one ~188 Hz bin; `0` → 12 | noise reads ~3–7 dB; set 12 for weak-but-intelligible, higher to ignore distant carriers |
| `squelch_dbfs` | ignored on an AM channel | if an AM channel chatters with gain, it is NOT this key |
| `amCNMeter` | Welch 8×256 Hann at ~48 kHz; peak ±4.5 kHz over 20th-percentile floor ±12 kHz | `am_squelch.go`, `TestAMCNMeterNoiseStaysClosed`, `TestAMCNMeterIsLevelIndependent` |
| `amMinDwell` | 100 ms floor on `MinDwellPerChannel` (first full Welch average ≈ 43 ms) | `TestAMModeValidationAndDefaults` |
| `demod.AM` | `|z|/C − 1`, `AMCarrierTau` 0.1 s, `AMClamp` ±1.5 | `internal/dsp/demod/am.go` |
| AM chain constants | `amChannelCutoffHz` 4500, `amAudioCutoffHz` 3000, `amAudioScale` 0.4 | `composer.go`, `TestComposerAMChainRecoversAirBandAudio` |
| Real-air pin | KRIC 126.400 slices: carrier C/N 47–61 dB, noise 2.7–6.1 dB | `am_realair_test.go` (`TestAMSquelchOnRealAirCarrier`) |

## In this post

- **Why the FM chain records nothing on the air band** — what a discriminator does to AM.
- **From `mode: am` to `am-conv`** — validation, the grant, the classifier.
- **The AM chain** — filter, detector, low-pass, scale; what is skipped and what is shared.
- **Squelch as a carrier-to-noise ratio** — `amCNMeter` and the rule behind it.
- **What is verified, and what to watch** — the 29 Sep run, the KRIC slices, the known limits.

## Why the FM chain records nothing on the air band

Aircraft and towers use double-sideband AM, `C·(1 + m(t))`: the information
is in the envelope. An FM discriminator computes the phase increment between
samples, and for a lone AM carrier that is a constant — the carrier offset —
with the envelope invisible. `TestComposerAMChainRecoversAirBandAudio` keeps
this as a fixture check: the same IQ through `fm-conv` must recover the
1 kHz test tone at less than a tenth of the AM chain's level, or the test
"would not distinguish the chains".

One subtlety the same test records: with a *neighbouring* carrier present,
two carriers in one discriminator **beat**, and the beat carries their
envelopes — so the pre-#1219 air band through the FM chain was not clean
silence but a distorted ghost of the audio. The FM counterfactual therefore
uses a lone carrier.

## From `mode: am` to `am-conv`

The config side is one key. `config_validate.go` accepts `mode` as
`fm|nfm|am` and `squelch_cn_db ≥ 0`; the scanner's `New` fills
`SquelchCNDb` with `DefaultAMSquelchCNDb` when it is zero and raises
`MinDwellPerChannel` to at least `amMinDwell` (100 ms) so the meter's first
full average is in before the scan window can expire. The example entry from
`config.example.yaml`:

```yaml
scanner:
  conventional:
    # AM channel (VHF air band, #1219). mode: am switches to envelope
    # detection and a carrier-to-noise squelch measured from the channel's
    # own spectrum — squelch_dbfs is ignored, so a gain change doesn't move
    # it. frequency_hz is the ACTUAL carrier: an 8.33 kHz "channel name"
    # like 118.005 is the 118.000 MHz carrier.
    - label: "Tower"
      frequency_hz: 118700000
      mode: am
      squelch_cn_db: 12    # default 12 (noise alone reads ~3–7 dB)
```

The 8.33 kHz note matters: channel *names* step by 5 kHz in the published
tables (118.000, 118.005, 118.010) but the carriers sit on the 8.33 kHz grid,
so 118.005 is the 118.000 MHz carrier. GopherTrunk centres its ±4.5 kHz
filter on `frequency_hz` — type the carrier.

From there the path is three one-liners: `conventionalProtocol(ch)` returns
`"am-conv"` for `ModeAM`, `beginDwell` stamps it on the synthetic grant, and
`classifyVoiceKind` has `case "am-conv": return voiceKindAM`, after which
`handleStart` runs `runFMChain` with `am=true` (`TestClassifyAMConv`,
`TestAMChannelGrantsAMConvAndEndsOnCarrierDrop`).

## The AM chain

`runFMChain`'s AM flag changes four stages and skips two. The front-end
decimator to 48 kHz is shared; after it:

```go
// internal/voice/composer/composer.go (shape) — runFMChain, am == true
chanFilter = newAMChannelFilter(intermediateHzf)   // 181 taps, ±amChannelCutoffHz
amDet      = demod.NewAM(fe.OutRateHz())
amAudioLPF = filter.NewRealFIR(filter.LowpassKaiser(127, amAudioCutoffHz/intermediateHzf, 8.6))
// FM-only stages are not built when am: no equalizer.CMA, no filter.DeEmphasis
// …per chunk:
audio = amDet.Process(nil, decimated)
audio = amAudioLPF.Process(audio, audio)
for i := range audio { audio[i] *= amAudioScale }
for _, hp := range audioHP { hp.ProcessFloat32(audio) }  // shared tone/DC high-pass
```

<figure class="lab-figure">
<svg viewBox="0 0 700 210" width="700" height="210" role="img" aria-label="Two parallel pipelines after a shared 48 kHz decimator. The FM branch runs the optional channel filter, optional CMA equalizer, FM discriminator, high-pass, de-emphasis and audio low-pass. The AM branch runs the fixed plus or minus 4.5 kHz channel filter, the demod.AM envelope detector that divides by a tracked carrier and subtracts one, a 3 kHz low-pass and a 0.4 scale, then the same high-pass; de-emphasis and the equalizer are marked as skipped. Both branches rejoin at the AGC, resampler and int16 conversion.">
  <text x="350" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">runFMChain: the FM and AM branches share the ends and differ in the middle</text>
  <rect x="8" y="80" width="80" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="48" y="96" text-anchor="middle" fill="currentColor" font-size="8">81-tap decim.</text>
  <text x="48" y="108" text-anchor="middle" fill="currentColor" font-size="8">→ 48 kHz</text>
  <g font-size="8">
    <text x="100" y="40" fill="var(--fg-muted)">FM (fm-conv)</text>
    <rect x="100" y="46" width="88" height="30" fill="none" stroke="var(--fg-muted)"/>
    <text x="144" y="65" text-anchor="middle" fill="var(--fg-muted)">fm_channel_bw (opt.)</text>
    <rect x="196" y="46" width="60" height="30" fill="none" stroke="var(--fg-muted)"/>
    <text x="226" y="65" text-anchor="middle" fill="var(--fg-muted)">CMA (opt.)</text>
    <rect x="264" y="46" width="84" height="30" fill="none" stroke="var(--fg-muted)"/>
    <text x="306" y="65" text-anchor="middle" fill="var(--fg-muted)">discriminator</text>
    <rect x="356" y="46" width="60" height="30" fill="none" stroke="var(--fg-muted)"/>
    <text x="386" y="65" text-anchor="middle" fill="var(--fg-muted)">HPF ×2</text>
    <rect x="424" y="46" width="72" height="30" fill="none" stroke="var(--fg-muted)"/>
    <text x="460" y="65" text-anchor="middle" fill="var(--fg-muted)">de-emphasis</text>
    <rect x="504" y="46" width="68" height="30" fill="none" stroke="var(--fg-muted)"/>
    <text x="538" y="65" text-anchor="middle" fill="var(--fg-muted)">LPF 3400</text>
  </g>
  <g font-size="8">
    <text x="100" y="140" fill="var(--accent)">AM (am-conv)</text>
    <rect x="100" y="146" width="88" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="144" y="159" text-anchor="middle" fill="var(--accent)">±4.5 kHz filter</text>
    <text x="144" y="170" text-anchor="middle" fill="var(--accent)">181 taps</text>
    <rect x="196" y="146" width="100" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="246" y="159" text-anchor="middle" fill="var(--accent)">demod.AM</text>
    <text x="246" y="170" text-anchor="middle" fill="var(--accent)">|z| / C − 1</text>
    <rect x="304" y="146" width="64" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="336" y="159" text-anchor="middle" fill="var(--accent)">LPF 3 kHz</text>
    <text x="336" y="170" text-anchor="middle" fill="var(--accent)">127 taps</text>
    <rect x="376" y="146" width="48" height="30" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="400" y="164" text-anchor="middle" fill="var(--accent)">× 0.4</text>
    <rect x="432" y="146" width="60" height="30" fill="none" stroke="currentColor"/>
    <text x="462" y="164" text-anchor="middle" fill="currentColor">HPF ×2</text>
    <text x="538" y="164" text-anchor="middle" fill="var(--fg-muted)">no de-emph · no CMA</text>
  </g>
  <rect x="592" y="80" width="100" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="642" y="96" text-anchor="middle" fill="currentColor" font-size="8">AGC · resample</text>
  <text x="642" y="108" text-anchor="middle" fill="currentColor" font-size="8">× 10 000 → int16</text>
  <path d="M88 100 L94 100 L94 61 L100 61" fill="none" stroke="var(--fg-muted)"/>
  <path d="M88 100 L94 100 L94 161 L100 161" fill="none" stroke="var(--accent)"/>
  <path d="M572 61 L584 61 L584 100 L592 100" fill="none" stroke="var(--fg-muted)"/>
  <path d="M492 161 L584 161 L584 100" fill="none" stroke="var(--accent)"/>
  <text x="350" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the carrier C is tracked with a 100 ms one-pole average of the envelope, so the output is modulation depth — the detector is its own AGC</text>
</svg>
<figcaption>The AM branch replaces the discriminator with a carrier-referenced envelope detector and drops the two FM-only stages; the tone high-pass, AGC, resampler and squelch-tail mute are shared.</figcaption>
</figure>

Each constant has a reason in the code:

- **`amChannelCutoffHz` = 4500.** Air-band voice is DSB with ~3 kHz audio,
  and the carrier may sit a kHz or two off the tuned frequency; ±4.5 kHz
  keeps that while rejecting an 8.33 kHz neighbour, whose nearest sideband
  starts ~5.3 kHz away. 181 Kaiser taps give a ~1.5 kHz skirt at 48 kHz.
- **`demod.AM`.** For `C·(1 + m(t))` the envelope `|z|` is proportional to
  `1 + m(t)`. The detector tracks `C` with a one-pole average of the envelope
  (`AMCarrierTau` = 100 ms — ~30 periods of a 300 Hz tone, still fast enough
  for propeller flutter) and emits `|z|/C − 1`: the modulation depth, DC
  removed, gain set by the carrier. Overmodulation reads −1; `AMClamp` = 1.5
  stops a noise spike on a faded carrier producing a full-scale click.
- **`amAudioCutoffHz` = 3000**, a 127-tap FIR. Its skirt is ~2 kHz wide
  (−2.4 dB at 2.5 kHz) — noted in the repo as a known softness, not changed.
- **`amAudioScale` = 0.4.** The detector's ±1 at 100 % depth is mapped onto
  the discriminator's scale so an AM channel records at the level of a
  ±3 kHz NFM one (±0.39 rad/sample at 48 kHz) before any AGC.
- **Skipped:** de-emphasis (AM is not pre-emphasised) and the CMA equalizer
  (its constant-modulus target is wrong for a signal whose envelope *is* the
  audio). **Shared:** the `fm_audio_highpass_hz` biquads — the 300 Hz default
  applies to AM too, which
  [Part 9]({{ '/blog/tutorials/conventional-scanner-09-am-carrier-tracker/' | relative_url }})
  shows matters on a hangar full of AC hum — the AGC, the resampler and the
  #1090 squelch-tail mute.

`TestComposerAMChainRecoversAirBandAudio` pins the level claim: a 1 kHz tone
at 60 % depth records at `0.6 × amAudioScale × 10 000` ±20 % whether the
carrier amplitude is 0.002 or 0.2 — **40 dB apart, same level** — with an
8.33 kHz neighbour's 2.2 kHz tone more than 40 dB under it.

## Squelch as a carrier-to-noise ratio

The FM channels squelch on in-channel power against `squelch_dbfs`
([Part 4]({{ '/blog/tutorials/conventional-scanner-04-squelch-on-the-channel/' | relative_url }})).
AM cannot borrow a noise-quieting statistic — AM has no capture effect — and
the rule the repo has learned repeatedly (the
[coherence-not-dBFS]({{ '/blog/tutorials/analog-edge-13-coherence-not-dbfs/' | relative_url }})
lesson) is that **no new gate may depend on absolute dBFS**: a gain change
would silently move it. So an AM channel's squelch is the carrier's
carrier-to-noise ratio, measured from the channel's own spectrum. An AM
transmitter always radiates its carrier, a spectral line holding at least
half the signal power; receiver noise is flat.

```go
// internal/scanner/conventional/am_squelch.go (shape) — amCNMeter.block
var peak float64
for _, k := range a.carrier {            // bins within ±amCNCarrierHalfHz (4500)
    peak = math.Max(peak, a.sum[k])      // Welch sum over amCNBlocks = 8 blocks
}
for i, k := range a.floor {              // bins within ±amCNFloorHalfHz (12 000)
    a.sorted[i] = a.sum[k]
}
sort.Float64s(a.sorted)
floor := a.sorted[int(amCNFloorPercentile*float64(len(a.sorted)))]  // 20th percentile
a.level = 10 * math.Log10(peak/floor)
```

The meter decimates the scanner's full-rate IQ to ~48 kHz (`amCNRefRateHz`),
takes a Welch average of `amCNBlocks` = 8 Hann-windowed 256-point spectra —
~188 Hz bins, a full average every ~43 ms — and reports the strongest bin
within ±4.5 kHz over the noise floor. The floor is a **percentile**, not a
mean: the 20th percentile of the bins within ±12 kHz stays honest when the
voice sidebands or an 8.33/25 kHz neighbour occupy part of the view. Both
terms scale together with gain, so the ratio is level-independent; on noise
alone it sits a few dB above zero.

`squelchMeasure` hands the dwell loop `am.process` and `ch.SquelchCNDb` in
place of the power meter and `SquelchDbFS`; the hysteresis of Part 4 applies
unchanged (`keepAlive = open − squelch_hysteresis_db`). Three tests pin the
design: `TestAMCNMeterNoiseStaysClosed` (noise never reaches the close level
over 10 s at several noise levels), `TestAMCNMeterIsLevelIndependent` (the
same C/N at noise floors 60 dB apart reads within 3 dB), and
`TestAMSquelchIgnoresAbsoluteLevel` — a 25 dB C/N carrier at **−85 dBFS**
opens, receiver noise at **−17 dBFS** stays shut, which the dBFS squelch gets
exactly backwards.

If no sample rate is available the meter cannot be built, and
`buildAMMeter` says so loudly:

```text
WRN conv: AM channel but the scanner sample rate is zero; using the squelch_dbfs power squelch instead of carrier-to-noise freq_hz=118700000 label=Tower
```

## What is verified, and what to watch

**On air.** AM is on-air verified: on 29 September, on a build with Part 9's
carrier tracker (PR #1227), the reporter heard "very good and clear" audio,
on frequency, with no extra noise or artifacts. Their 126.400 MHz KRIC
Potomac Departure capture — carrier up ~1.5 s, receiver noise after unkey,
same receiver and gain — is committed as two 48 kHz slices
(`am_kric_126400k_carrier_48k.cs16`, `am_kric_126400k_noise_48k.cs16` under
`internal/scanner/conventional/testdata`); `TestAMSquelchOnRealAirCarrier`
interpolates them back to 2.4 MS/s and demands the meter never dip below
12 dB during the transmission (**47–61 dB**) nor reach the close level on
the noise (**2.7–6.1 dB**).

**Known limits.** An ACARS burst on an AM channel also opens an ordinary AM
"call" — the carrier is up — so short data-burst recordings appear in the
call history beside the decoded messages
([Part 10]({{ '/blog/tutorials/conventional-scanner-10-acars/' | relative_url }})).
The 3 kHz low-pass's wide skirt is noted above. Per-channel `gain` (Part 4)
applies to AM channels like any other, which matters on a mixed list where
the air band is often the weakest.

## Where this goes next

The ±4.5 kHz filter is centred on the frequency you typed — and the first
on-air captures carried the carrier 3315 Hz away from it, which cut a whole
sideband without any log line saying so.
[Part 9]({{ '/blog/tutorials/conventional-scanner-09-am-carrier-tracker/' | relative_url }})
adds `amCarrierAFC`: find the carrier line, mix it to DC ahead of the filter,
hold on noise, and report at call end when it sat more than 1 kHz off — plus
the measurement that actually shows the loss, which audio correlation does
not.

## FAQ

**How do I scan the VHF air band with GopherTrunk?**
Add a `scanner.conventional` entry with `mode: am` and the carrier frequency
in `frequency_hz` (131550000 for 131.550 MHz). The scanner squelches on
carrier-to-noise (`squelch_cn_db`, default 12) and the composer runs the AM
chain (`am-conv`): a ±4.5 kHz channel filter, an envelope detector
referenced to the tracked carrier, and a 3 kHz low-pass. It is on-air
verified (29 Sep).

**Why is squelch_dbfs ignored on an AM channel?**
Because a dBFS threshold moves with front-end gain. The AM squelch is
`amCNMeter`'s carrier-to-noise ratio — the strongest bin within ±4.5 kHz
over the 20th-percentile noise floor within ±12 kHz, from a Welch-averaged
spectrum — which reads the same at any level (noise ~3–7 dB, a weak
intelligible carrier ~12 dB). Set `squelch_cn_db` instead.

**What frequency do I enter for an 8.33 kHz air-band channel?**
The actual carrier. 8.33 kHz channel names step by 5 kHz (118.005, 118.010)
but the carriers sit on the 8.33 kHz grid; 118.005 is the 118.000 MHz
carrier. GopherTrunk centres its ±4.5 kHz channel filter on `frequency_hz`,
so a channel name typed literally lands the filter off the carrier.

**Why does the AM chain skip de-emphasis and the equalizer?**
AM is not pre-emphasised, so there is nothing to de-emphasise; and the CMA
blind equalizer minimises deviation from a constant modulus, which is wrong
for a signal whose envelope is the audio. The tone high-pass, AGC, resampler
and squelch-tail mute are shared with the FM chain.

## Series navigation

**Part 8 of 14** · ←
[Part 7: The FM Voice Chain — De-emphasis, Band Limit, the 300 Hz High-Pass, Bandwidth]({{ '/blog/tutorials/conventional-scanner-07-fm-voice-chain/' | relative_url }})
· Next →
[Part 9: The AM Carrier Tracker — Why a 3 kHz Tuning Error Cost a Sideband]({{ '/blog/tutorials/conventional-scanner-09-am-carrier-tracker/' | relative_url }})
