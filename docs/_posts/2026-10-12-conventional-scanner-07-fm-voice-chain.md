---
title: "The Conventional Scanner, Part 7: The FM Voice Chain — De-emphasis, Band Limit, the 300 Hz High-Pass, Bandwidth"
description: "What happens to a conventional channel's IQ once the dwell opens — the composer's runFMChain stage by stage, the four recordings.fm_* keys that shape analog audio (fm_channel_bandwidth_hz, fm_audio_highpass_hz, fm_deemphasis, fm_audio_lowpass_hz), why the high-pass must run before de-emphasis, why the channel filter is a second filter at 48 kHz, and when to change any of them."
category: tutorials
keywords: analog fm voice chain sdr, fm de-emphasis 75us 50us, fm_audio_highpass_hz, fm_channel_bandwidth_hz nfm, ctcss hum in recording, sdr scanner audio hiss, rtl-sdr nfm bandwidth 12.5 khz, squelch tail mute, conventional scanner recording, gophertrunk conventional scanner
tags: [conventional-scanner, analog-fm, composer, audio, de-emphasis, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Conventional Scanner"
series_part: 7
---

*Part 7 of **The Conventional Scanner**, a 14-part operator's tutorial on
GopherTrunk's conventional (non-trunked) scanner, `scanner.conventional`, from
the dwell loop to a verified kitchen-sink config.
[Part 6]({{ '/blog/tutorials/conventional-scanner-06-dcs/' | relative_url }})
closed the sub-audible gates that decide whether a dwell starts. This part
follows the dwell into the composer — the one analog voice chain every FM
scan-list channel shares — and reads the four `recordings.fm_*` keys that
landed for [#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)
in three commits over three days: de-emphasis and band limit (19 Sep), the
high-pass (20 Sep), the selectable channel bandwidth (21 Sep). Each exists
because the reporter's Kenwood recordings were harsh, hissy and hum-ridden
without it.*

> **TL;DR:** A conventional FM dwell publishes a synthetic grant with
> `Protocol: "fm-conv"`; `classifyVoiceKind` maps it to `voiceKindFM` and
> `handleStart` launches `runFMChain(…, am=false)` in
> `internal/voice/composer/composer.go`. The chain: an 81-tap decimating
> anti-alias FIR from the SDR rate to 48 kHz (`VoiceBandwidthHz`, default
> 12 500) → the optional `fm_channel_bandwidth_hz` complex low-pass at 48 kHz
> (±half the width, 81 Kaiser taps; 0 = legacy 25 kHz) → optional CMA
> equalizer → `demod.NewFM` discriminator → **two cascaded Butterworth
> high-pass biquads at `fm_audio_highpass_hz` (default 300)** → de-emphasis
> (`fm_deemphasis`: `us`/75 µs default, `eu`/50 µs, `off`) → the
> `fm_audio_lowpass_hz` FIR (default 3400, 81 taps, also the anti-alias for
> the drop to PCM) → optional AGC → decimate or polyphase-resample to the PCM
> rate → ×10 000 to int16. The scanner's debounced squelch decision feeds
> `SquelchState`, so the hangtime tail is faded to silence rather than
> recorded as noise. `mode: nfm` reaches the chain as the same `fm-conv`
> grant; the channel width is the global `fm_channel_bandwidth_hz`.

**Key takeaways**

- **Order is the design.** The high-pass runs *before* de-emphasis because
  de-emphasis boosts the low end: without the high-pass the DC bias and the
  CTCSS tone come out louder, not softer.
- **A selectable bandwidth has to live at 48 kHz.** An 81-tap FIR at
  2.4 MS/s has a ~100 kHz transition band; the same filter at 48 kHz is
  2–3 kHz sharp, which is the only place a 12.5 kHz NFM channel can be
  separated from its neighbour.
- **Every analog key defaults on.** Leave `fm_deemphasis`,
  `fm_audio_highpass_hz` and `fm_audio_lowpass_hz` alone unless you are
  diagnosing; `fm_channel_bandwidth_hz` is the one that defaults *off*
  (legacy 25 kHz) so existing recordings stay byte-identical.
- **The recorder does not know it is a scanner.** The grant has a system, a
  group ID (`0x80000000|index` or `talkgroup_id`), a label and a frequency;
  everything downstream treats the dwell as a call.

## Cheat sheet

| Line / key | Meaning | Healthy / worry when |
|---|---|---|
| `fm_channel_bandwidth_hz` | TOTAL IF width of the 48 kHz channel filter; `0` = legacy 25 kHz front end; 2500..50000 | `12500` for NFM radios with adjacent-channel hiss; worry if voice sounds clipped (too narrow for the deviation) |
| `fm_audio_highpass_hz` | post-discriminator 4th-order high-pass corner; `0` → 300; `<0` off | hum or a steady tone under voice ⇒ it is off or too low |
| `fm_deemphasis` | `us`/`75us` (default), `eu`/`50us`, `off`/`flat` | harsh, bright, hissy audio ⇒ `off` or an unknown token |
| `fm_audio_lowpass_hz` | audio band limit + anti-alias before PCM; `0` → 3400; `<0` off | broadband noise at full quieting ⇒ it is off (aliased) |
| `voiceKindFM` / `fm-conv` | the chain a conventional FM dwell runs | `classifyVoiceKind` (`composer.go`) |
| `SquelchState.SquelchOpen` | scanner's debounced decision, mutes the hangtime tail | `squelchState` (`scanner.go`), #1090 |
| `resolveFMDeEmphasis` etc. | daemon maps the keys to composer options | `cmd/gophertrunk/daemon.go` |

## In this post

- **From a dwell to a chain** — the synthetic grant and `classifyVoiceKind`.
- **The chain, stage by stage** — decimator, channel filter, discriminator, the audio stages, PCM.
- **Why the high-pass runs first** — DC, sub-audible tones and the de-emphasis integrator.
- **A second filter at 48 kHz** — `fm_channel_bandwidth_hz` and the transition-band arithmetic.
- **The keys, and when to change them** — the YAML, the resolvers, and the defaults.
- **The squelch tail and the recorder** — what the recorder sees.

## From a dwell to a chain

[Part 1]({{ '/blog/tutorials/conventional-scanner-01-what-conventional-means/' | relative_url }})
ended on `beginDwell` synthesizing a `trunking.Grant`. The field that picks
the voice chain is `Protocol`, set by `conventionalProtocol(ch)` in
`am_squelch.go`: `"am-conv"` for `mode: am`, `"fm-conv"` for everything else.
`mode: fm` and `mode: nfm` therefore arrive at the composer as the same
grant. The `Channel.Mode` comment and the config-builder help describe `nfm`
as "narrow", but nothing in the decode path branches on it; the channel
width an FM dwell gets is the global `recordings.fm_channel_bandwidth_hz`,
described below. If you expected `nfm` to narrow the filter per channel, it
does not — set the key.

The composer classifies the grant in `classifyVoiceKind`:

```go
// internal/voice/composer/composer.go (shape)
case "am-conv":
    return voiceKindAM
}
proto := cs.Grant.Protocol
isAnalogTrunk := proto == "motorola" || proto == "ltr" || proto == "mpt1327" ||
    (proto == "edacs" && !cs.Grant.ProVoice)
if proto == "" || proto == "fm" || proto == "fm-conv" || proto == "analog" || isAnalogTrunk {
    return voiceKindFM
}
```

`voiceKindFM` also covers the analog trunked voice channels — Motorola
Type II, LTR, MPT 1327, EDACS without ProVoice — which is why the
`recordings.fm_*` keys are documented as affecting both. `handleStart` then
runs `runFMChain(ctx, serial, iqCh, rate, false, done)`; the `true` variant
is the AM chain of
[Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }}).

## The chain, stage by stage

`runFMChain` is deliberately linear. On the conventional path the LO offset
of [Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})
has already been mixed out by `convScanVoiceSource`, so the channel sits at
DC.

<figure class="lab-figure">
<svg viewBox="0 0 700 200" width="700" height="200" role="img" aria-label="A left-to-right pipeline of the composer's analog FM chain. The IQ at the SDR rate enters an 81-tap decimating FIR that outputs 48 kHz, then the optional fm_channel_bandwidth_hz complex low-pass, then the optional CMA equalizer, then the FM discriminator. The real audio then passes two cascaded high-pass biquads at fm_audio_highpass_hz, the de-emphasis single-pole low-pass, the fm_audio_lowpass_hz FIR, an optional AGC, and finally decimation or a polyphase resampler to the PCM rate and conversion to int16. A bracket marks the audio stages as running at 48 kHz, and a note marks that the squelch-closed tail is faded to silence after conversion.">
  <text x="350" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">runFMChain(am=false) · IQ at the SDR rate → PCM at recordings.sample_rate</text>
  <g font-size="8" fill="currentColor">
    <rect x="8" y="34" width="92" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="54" y="50" text-anchor="middle">81-tap decimating</text>
    <text x="54" y="62" text-anchor="middle">FIR → 48 kHz</text>
    <rect x="112" y="34" width="104" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="164" y="50" text-anchor="middle" fill="var(--accent)">fm_channel_bandwidth_hz</text>
    <text x="164" y="62" text-anchor="middle" fill="var(--accent)">±half, 81 taps (opt.)</text>
    <rect x="228" y="34" width="72" height="40" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
    <text x="264" y="50" text-anchor="middle" fill="var(--fg-muted)">CMA eq.</text>
    <text x="264" y="62" text-anchor="middle" fill="var(--fg-muted)">(opt.)</text>
    <rect x="312" y="34" width="84" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="354" y="50" text-anchor="middle">FM discriminator</text>
    <text x="354" y="62" text-anchor="middle">demod.NewFM</text>
    <rect x="408" y="34" width="92" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="454" y="50" text-anchor="middle" fill="var(--accent)">2× HP biquad</text>
    <text x="454" y="62" text-anchor="middle" fill="var(--accent)">fm_audio_highpass_hz</text>
    <rect x="512" y="34" width="84" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="554" y="50" text-anchor="middle" fill="var(--accent)">de-emphasis</text>
    <text x="554" y="62" text-anchor="middle" fill="var(--accent)">fm_deemphasis</text>
    <rect x="608" y="34" width="84" height="40" fill="none" stroke="var(--accent)" stroke-width="1.5"/>
    <text x="650" y="50" text-anchor="middle" fill="var(--accent)">audio LPF FIR</text>
    <text x="650" y="62" text-anchor="middle" fill="var(--accent)">fm_audio_lowpass_hz</text>
  </g>
  <g font-size="8" fill="currentColor">
    <rect x="312" y="110" width="84" height="40" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
    <text x="354" y="126" text-anchor="middle" fill="var(--fg-muted)">audio AGC</text>
    <text x="354" y="138" text-anchor="middle" fill="var(--fg-muted)">(opt.)</text>
    <rect x="408" y="110" width="108" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="462" y="126" text-anchor="middle">decimate / polyphase</text>
    <text x="462" y="138" text-anchor="middle">→ PCM rate</text>
    <rect x="528" y="110" width="84" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <text x="570" y="126" text-anchor="middle">×10 000 → int16</text>
    <text x="570" y="138" text-anchor="middle">squelch-tail fade</text>
  </g>
  <path d="M650 74 L650 92 L354 92 L354 110" fill="none" stroke="var(--fg-muted)"/>
  <line x1="100" y1="54" x2="112" y2="54" stroke="currentColor"/>
  <line x1="216" y1="54" x2="228" y2="54" stroke="currentColor"/>
  <line x1="300" y1="54" x2="312" y2="54" stroke="currentColor"/>
  <line x1="396" y1="54" x2="408" y2="54" stroke="currentColor"/>
  <line x1="500" y1="54" x2="512" y2="54" stroke="currentColor"/>
  <line x1="596" y1="54" x2="608" y2="54" stroke="currentColor"/>
  <line x1="396" y1="130" x2="408" y2="130" stroke="currentColor"/>
  <line x1="516" y1="130" x2="528" y2="130" stroke="currentColor"/>
  <text x="8" y="172" fill="var(--fg-muted)" font-size="8">complex IQ · 48 kHz</text>
  <text x="408" y="172" fill="var(--fg-muted)" font-size="8">real audio · 48 kHz · every stage before the resampler runs here</text>
  <text x="350" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="8">accent = the four recordings.fm_* keys · dashed = opt-in stages (recordings.equalizer, audio AGC)</text>
</svg>
<figcaption>The analog chain. Every operator key sits between the decimator and the resampler, at the 48 kHz intermediate rate; the discriminator divides the complex and the real halves.</figcaption>
</figure>

1. **Front-end decimator.** `newDecimatingFIR(iqHz, 48_000, c.bw, true)`
   — an 81-tap anti-alias FIR convolved only at the output positions. `c.bw`
   is `VoiceBandwidthHz`, default 12 500: the cutoff of this first filter is
   where the "legacy 25 kHz channel" comes from.
2. **Channel-select filter** — `newFMChannelFilter`, only when
   `fm_channel_bandwidth_hz` is set. Next section.
3. **CMA equalizer** — `recordings.equalizer`, off by default; see the
   [composer post]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }}).
4. **Discriminator** — `demod.NewFM`, radians per sample.
5. **High-pass** — two cascaded `filter.NewHighPass` biquads at
   `fm_audio_highpass_hz`.
6. **De-emphasis** — `filter.NewDeEmphasis(tau, 48 kHz)`: `y[n] = (1−α)·x[n]
   + α·y[n−1]`, `α = exp(−1/(τ·fs))`, unity DC gain, −3 dB at `1/(2πτ)`.
7. **Audio low-pass** — a Kaiser FIR (default 81 taps) at
   `fm_audio_lowpass_hz`, band-limiting voice and anti-aliasing the drop to
   PCM.
8. **AGC, resample, convert** — optional `dsp.AudioAGC`; naive decimation
   by `48000/pcmHz` or the opt-in polyphase `dsp.RealResampler`;
   `decimateAndConvert` scales by 10 000 and clamps to int16.

## Why the high-pass runs first

The 20 September commit's comment in `runFMChain` is the clearest statement
of the problem. An FM discriminator turns a residual carrier-frequency offset
into a constant DC bias, and the sub-audible CTCSS/DCS tones of Parts 5 and 6
(67–250 Hz) ride under the voice as a low hum. De-emphasis is a low-pass with
unity DC gain, so relative to the treble it is *boosting* the low end —
"without this the tone/DC is louder, not softer". So the high-pass is the
first audio stage, before the de-emphasis integrator and before the AGC ever
sees the DC:

```go
// internal/voice/composer/composer.go (shape) — runFMChain
var audioHP []*filter.Biquad
if c.hpfCfg.Enabled && c.hpfCfg.CutoffHz > 0 {
    audioHP = []*filter.Biquad{
        filter.NewHighPass(intermediateHzf, float64(c.hpfCfg.CutoffHz)),
        filter.NewHighPass(intermediateHzf, float64(c.hpfCfg.CutoffHz)),
    }
}
// …per chunk:
audio = fm.Process(nil, decimated)
for _, hp := range audioHP { hp.ProcessFloat32(audio) }
if deemph != nil { audio = deemph.Process(audio, audio) }
```

Two sections, not one: 241.8 Hz is a standard CTCSS tone just under a 300 Hz
corner, and a single 2nd-order section trims it only ~6 dB there; cascaded,
the 4th-order ~24 dB/octave slope gives real rejection. It is the stage
rtl_fm, SDR# and OpenWebRX all apply, and the analog chain simply had not —
which is why the reporter's tone-gated channels recorded their own tone.

## A second filter at 48 kHz

The front-end decimator already band-limits the IQ, so why a second complex
filter? `newFMChannelFilter`'s comment does the arithmetic: at the SDR rate,
~2.4 MS/s, an 81-tap filter has a **~100 kHz transition band**, far too wide
to separate a 12.5 kHz NFM channel from its neighbour. At 48 kHz the same 81
taps are sharp (transition ≈ 2–3 kHz), so this is where a selectable
"bandwidth" — the control SDR# and SDR++ expose for NFM — can actually
reject adjacent-channel energy and out-of-channel noise. The reporter's
Kenwood NFM hiss of #1184 is the case it was built for.

```go
// internal/voice/composer/composer.go (shape)
func (c *Composer) newFMChannelFilter(intermediateHzf float64) *filter.FIR {
    if c.fmChannelBWHz == 0 {
        return nil            // legacy front end: byte-for-byte unchanged
    }
    fc := (float64(c.fmChannelBWHz) / 2) / intermediateHzf
    if fc > 0.45 { fc = 0.45 }
    return filter.NewFIR(filter.LowpassKaiser(81, fc, 8.6))
}
```

The key is the **total** width; the cutoff is ±half of it, centred on DC
because the taps are real and symmetric. `12500` selects a 12.5 kHz NFM
channel (±6.25 kHz), `25000` a 25 kHz one; the range is 2500..50000. It
affects only `runFMChain` — the digital chains size their own filter from
`VoiceBandwidthHz`. Too narrow clips a normally-deviated signal (a ±5 kHz
radio with 3 kHz audio occupies ~±8 kHz by Carson's rule), so `12500` is for
radios that really are tightly deviated.

## The keys, and when to change them

The `recordings` block, as `config.example.yaml` ships it:

```yaml
recordings:
  # Analog-FM voice audio (conventional scanner + analog SmartNet/Type II voice).
  # All default ON — leaving them off records harsh, hissy, hum-ridden audio (issue #1184).
  fm_deemphasis: us          # us/75us (default, North America) | eu/50us | off (flat).
  fm_audio_lowpass_hz: 3400  # post-demod audio low-pass corner: band-limits voice AND
                             # anti-aliases the decimation to sample_rate. 0 = default
                             # 3400; <0 disables (flat, aliased — diagnostic only).
  fm_audio_highpass_hz: 300  # post-demod audio high-pass corner: strips the DC bias a
                             # carrier-tuning offset leaves and the sub-audible CTCSS/DCS
                             # squelch tones (the low hum de-emphasis amplifies). 0 =
                             # default 300; <0 disables.
  fm_channel_bandwidth_hz: 0 # analog-FM IF channel filter TOTAL width (the "bandwidth"
                             # SDR#/SDR++ expose); the front-end low-pass is ±half this.
                             # 12500 = a 12.5 kHz NFM channel, 25000 = a 25 kHz wideband
                             # channel. 0 = the legacy 25 kHz front end; range 2500..50000.
```

Four small resolvers in `cmd/gophertrunk/daemon.go` map them to composer
options. `resolveFMDeEmphasis` calls `config.ParseFMDeEmphasis` —
`""`/`us`/`75us`/`75`/`na` (75 µs), `eu`/`50us`/`50`, `off`/`none`/`flat`/`false`
— and an unrecognised token falls back to 75 µs enabled rather than silently
flat. `resolveFMAudioHPF` maps `<0` to disabled, `0` to 300 Hz, positive
through; `resolveFMAudioLPF` likewise with 3400; `resolveFMChannelBandwidth`
returns 0 for anything non-positive.

When to touch each: **`fm_channel_bandwidth_hz: 12500`** when a
tightly-deviated NFM radio records a hiss the tone high-pass does not remove
— adjacent-channel energy, which only this filter rejects.
**`fm_deemphasis: eu`** outside North America; `off` only to capture
sub-audible signalling flat. **`fm_audio_highpass_hz`** and
**`fm_audio_lowpass_hz`** rarely; a negative low-pass is a diagnostic setting
that records an aliased, broadband-noisy file.

Honest status: the chain was built against the #1184 reporter's Kenwood
captures and each stage's reason is in the code, but the repo records no
separate on-air sign-off for these keys as a set. What it does record from
the same issue is the 23 September root cause of the "noise/tone on an
on-frequency signal" report — zero-IF tuning of an overloaded ADC —
[Part 3]({{ '/blog/tutorials/conventional-scanner-03-offset-tuning/' | relative_url }})'s
story, itself still on-air-gated.

## The squelch tail and the recorder

The last stage ties the chain back to the scanner. `beginDwell` publishes
its debounced decision into `squelchState` — open exactly while the
hangtime countdown is *not* running — and the composer reads it through
`SquelchState.SquelchOpen(serial)`. While closed, the chain keeps running
(filters warm, PCM timeline continuous for the recorder and the live
stream), but the AGC is frozen so it cannot ride up on demodulated noise,
and the PCM is replaced by a ~10 ms fade into silence
(`muteFade`/`muteStep`, carried across chunks). That is the #1090 fix for the
loud squelch crash every hangtime used to record; `emitTail` does the same
fade at chain end so the WAV does not click.

The recorder sees nothing unusual. The grant carries `System`, `GroupID`
(the `0x80000000|index` default or the pinned `talkgroup_id` of
[Part 2]({{ '/blog/tutorials/conventional-scanner-02-scan-list-as-config/' | relative_url }})),
`GroupLabel: ch.Label` and `FrequencyHz`. The composer writes with plain
`WritePCM(serial, pcm)` — no call-ID fence, because an analog chain keys on
a stable physical serial, one chain per serial, torn down on `CallEnd`
before the next `CallStart`. From there the
[recording session]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
and the call log treat the dwell exactly like a trunked call.

## Where this goes next

A discriminator cannot hear an AM carrier — through this chain the VHF air
band is silence. [Part 8]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }})
takes `mode: am` from the config to the `am-conv` grant, swaps the
discriminator for `demod.AM`'s carrier-referenced envelope detector, drops
de-emphasis and the equalizer, and replaces the dBFS squelch with a
carrier-to-noise meter that reads the same number at any gain.

## FAQ

**Why does my conventional FM recording have a hum or a steady low tone under the voice?**
A residual carrier offset leaves DC out of the discriminator, and the
channel's CTCSS or DCS signalling (67–250 Hz) rides under the voice. The
chain strips both with two cascaded high-pass sections at
`recordings.fm_audio_highpass_hz` (default 300), placed *before*
de-emphasis so the low-end boost cannot amplify them. Check the key is not
negative.

**What does recordings.fm_channel_bandwidth_hz do and what value should I set?**
It inserts a complex low-pass at the 48 kHz intermediate rate, ±half the
configured total width, ahead of the discriminator. `12500` selects a
12.5 kHz NFM channel and rejects adjacent-channel hiss; `0` (default) keeps
the legacy 25 kHz front end so existing recordings are unchanged. Range
2500..50000.

**Should I use 75 µs or 50 µs de-emphasis?**
`fm_deemphasis: us` (75 µs) in North America, `eu` (50 µs) in Europe and
most other regions — it must match the transmitter's pre-emphasis. `off`
records a flat, harsh response for capturing sub-audible signalling. An
unrecognised token falls back to 75 µs, never flat.

**Does mode: nfm narrow the filter on a conventional channel?**
No. `fm` and `nfm` both produce an `fm-conv` grant and run the same
`runFMChain`; nothing in the decode path branches on `nfm`. The width is the
global `recordings.fm_channel_bandwidth_hz`, whatever the `Channel.Mode`
comment and the config-builder help say.

## Series navigation

**Part 7 of 14** · ←
[Part 6: DCS — The On-Air Bit Order, Polarity, and the Aliases You Cannot Fix]({{ '/blog/tutorials/conventional-scanner-06-dcs/' | relative_url }})
· Next →
[Part 8: AM for the Air Band — The Carrier as the AGC Reference]({{ '/blog/tutorials/conventional-scanner-08-am-for-the-air-band/' | relative_url }})
