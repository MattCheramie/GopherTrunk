---
title: "Voice Coding, Part 10: Enhancement & Loudness — Making It Sound Right"
description: The post-synthesis audio chain in GopherTrunk's MBE decoders — spectral enhancement, DC blocking, AGC, tone tilt, an opt-in voice enhancer, and per-call EBU R128 loudness normalization.
category: deep-dives
keywords: mbe spectral enhancement, vocoder agc, dc block filter, tone tilt shelf, voice enhancer, ebu r128 normalization, bs.1770 loudness, soft limiter, gophertrunk voice coding
tags: [mbe, agc, audio, dsp, loudness, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 10
---

*Part 10 of **Voice Coding**. Parts 3–8 got us from bits to synthesized PCM.
That PCM is correct, but "correct" and "sounds right" are different claims.
This post is the conditioning stack that closes the gap — and the line
GopherTrunk draws between staying faithful to the transmission and making it
pleasant to listen to.*

> **TL;DR:** MBE audio conditioning happens in two domains. In the **parameter
> domain**, §6.2 spectral enhancement boosts under-represented mid-band
> harmonics before synthesis (energy-preserving). In the **PCM domain**, a
> fixed chain runs per frame: DC block → optional tone tilt → optional voice
> enhancer → AGC with a `tanh` soft-limiter. Finally, per *call*, optional EBU
> R128 loudness normalization rewrites the finished WAV with a single linear
> gain. Everything past the DC block and AGC is **opt-in** — the default path
> is byte-faithful to the transmission.

**Key takeaways**

- **Spectral enhancement is not post-processing** — it runs on harmonic
  amplitudes *before* synthesis (`EnhanceAmplitudes`, TIA-102.BABA §6.2), with
  a per-harmonic weight clamped to `[0.5, 1.2]` and total energy preserved.
- The per-frame PCM chain is **DC block → tone tilt → enhancer → AGC**, run
  inside every decoder's `applyOutput`. The optional stages are nil
  pass-throughs by default, so faithful output is byte-identical.
- The **AGC** is a fast-attack / slow-release peak tracker to `TargetPeak`,
  with a per-frame output ceiling that was the fix for the "robotic /
  over-driven" field reports.
- **Loudness normalization** is a separate per-call pass: EBU R128 / BS.1770
  measure-then-apply-one-gain, pure gain, no compression — so within-call
  dynamics survive.

## Cheat sheet

| Stage | File | Domain | Default |
|---|---|---|---|
| Spectral enhancement | `mbe/enhance.go` | parameter (pre-synth) | on (spec) |
| DC block | `mbe/dcblock.go` | PCM | on |
| Tone tilt (warm shelf) | `mbe/tonetilt.go` | PCM | off (opt-in) |
| Voice enhancer (HPF/shelf/LPF/comp) | `mbe/enhancer.go` | PCM | off (opt-in) |
| AGC + soft-limiter | `mbe/agc.go` | PCM | on |
| Loudness normalize | `voice/normalize.go` | whole WAV | off (opt-in) |

## In this post

- **Two domains** — enhancement before synthesis, conditioning after.
- **The PCM chain** — DC block, tone tilt, enhancer, and their order.
- **The AGC** — envelope tracking, the soft-limiter, and one nasty bug.
- **Loudness** — why normalization is a separate per-call gain, not a stage.

## Two domains of "enhancement"

The word "enhancement" is overloaded here, so let's split it. The first kind is
part of the vocoder *spec*: after cross-frame amplitude recovery and the
log2→linear conversion, TIA-102.BABA §6.2 boosts harmonics the model
under-represents — typically mid-band harmonics whose energy the PRBA + HOC
quantizer averages into neighbours. This is `EnhanceAmplitudes`, and it runs on
the harmonic amplitudes `M[l]` **before** any samples exist:

```go
// internal/voice/mbe/enhance.go (shape)
// Per-harmonic weight, then a total-energy-preserving rescale.
//   if 8·l ≤ L:  W_l = 1.0                      (low band untouched)
//   else:        W_l = (0.96·num/den)^0.25       (clamped to [0.5, 1.2])
func EnhanceAmplitudes(p Params, M *[57]float64) { /* ... */ }
```

Two properties keep it safe to run unconditionally: low-frequency harmonics
(`8·l ≤ L`) are left alone, and after the per-harmonic multiply the whole frame
is rescaled so `Σ M_l²` (total power) is unchanged — enhancement *redistributes*
spectral energy, it doesn't add loudness. The `[0.5, 1.2]` clamp stops a
near-pure-tone frame (where the closed form's denominator approaches zero) from
producing a runaway weight.

The *second* kind is everything that happens to the finished PCM, which is
where the rest of this post lives.

<figure class="lab-figure">
<svg viewBox="0 0 680 172" width="680" height="172" role="img" aria-label="Spectral enhancement operates on harmonic amplitudes before synthesis, while DC block, tone tilt, enhancer and AGC operate on the synthesized PCM, followed by a separate per-call loudness normalization of the WAV">
  <rect x="10" y="20" width="300" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="160" y="40" text-anchor="middle" fill="var(--accent)" font-size="12">parameter domain (before synthesis)</text>
  <text x="160" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="10">EnhanceAmplitudes · energy-preserving</text>
  <line x1="310" y1="46" x2="352" y2="46" stroke="currentColor"/>
  <polygon points="352,42 362,46 352,50" fill="currentColor"/>
  <rect x="362" y="20" width="120" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="422" y="42" text-anchor="middle" fill="currentColor" font-size="12">synthesis</text>
  <text x="422" y="58" text-anchor="middle" fill="var(--fg-muted)" font-size="10">→ 160 PCM</text>
  <line x1="422" y1="72" x2="422" y2="96" stroke="currentColor"/>
  <polygon points="418,96 422,106 426,96" fill="currentColor"/>
  <rect x="70" y="106" width="504" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="322" y="124" text-anchor="middle" fill="var(--accent)" font-size="12">PCM domain: DC block → [tone tilt] → [enhancer] → AGC</text>
  <text x="322" y="139" text-anchor="middle" fill="var(--fg-muted)" font-size="10">optional stages are nil pass-throughs by default</text>
  <line x1="574" y1="126" x2="612" y2="126" stroke="currentColor"/>
  <polygon points="612,122 622,126 612,130" fill="currentColor"/>
  <text x="636" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="10">WAV →</text>
  <text x="636" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="10">normalize</text>
</svg>
<figcaption>Spectral enhancement redistributes harmonic energy before samples exist; the PCM chain conditions the samples; loudness normalization is a final per-call gain on the whole file.</figcaption>
</figure>

## The PCM chain

Every MBE decoder runs the same `applyOutput` after synthesis, and the order is
load-bearing:

```go
// internal/voice/ambe2/decoder.go (shape) — identical shape in imbe
func (d *Decoder) applyOutput(pcm []float64, out []int16, freezeEnvelope bool) {
    d.dc.Process(pcm)        // DC-removal high-pass (always on)
    d.tone.Process(pcm)      // optional warm shelf (nil = pass-through)
    d.enhancer.Process(pcm)  // optional voice enhancer (nil = pass-through)
    d.agc.Apply(pcm, out, freezeEnvelope) // level + soft-limit → int16
}
```

**DC block** comes first because the synthesizer can leave a small DC / very-low
offset (the voiced cosine sum and the §6.4 overlap-add noise aren't guaranteed
zero-mean per frame). Left in, that offset wastes AGC headroom and adds
inaudible rumble the limiter then has to absorb. It's a one-pole high-pass with
a pole at `z ≈ 0.99`, continuous across frames:

```go
// internal/voice/mbe/dcblock.go (shape)
y[n] = x[n] − x[n−1] + 0.99·y[n−1]
```

**Tone tilt** is an optional first-order high-shelf. GopherTrunk's software
AMBE+2 decode measured ~1.5–2 dB brighter than the mbelib reference across
1.5–4 kHz (issue #644), which reads as a thin, harsh "digital" timbre. The
`ambe2-dmr-warm` decoder trades a little of that brightness back:
`out = lp + g·(x − lp)`, a low-pass split with the high band scaled by
`g = 10^(−shelfDb/20)`. It's a *tone preference*, not a codec fix — the residual
synthetic character of low-bitrate resynthesis is intrinsic to software
vocoding.

**The voice enhancer** is the fuller opt-in chain, enabled per-recording. It
replicates what the louder rival decoders do: a ~250 Hz rumble high-pass, a
presence high-shelf (the same brightness trim), a ~3.4 kHz telephone-band
low-pass (OP25's trick to kill out-of-band quantization buzz), and an optional
soft-knee compressor. Crucially, it does **not** add a gain stage — output
loudness is handled by raising the AGC's `TargetPeak`, so the enhancer never
fights the limiter:

```go
// internal/voice/mbe/enhancer.go (shape)
func DefaultEnhancerConfig() EnhancerConfig {
    return EnhancerConfig{
        Enabled: true, HPFHz: 250, LPFHz: 3400, ShelfHz: 1500, ShelfDB: 2.0,
        AGCTarget: 22000, // louder than faithful 18000, still under the 26213 knee
        Compress: CompressConfig{ThresholdDB: -18, Ratio: 2, AttackMs: 5, ReleaseMs: 80},
    }
}
```

## The AGC and its soft-limiter

The synthesizer's per-frame output magnitude is stable *within* a frame but
swings wildly *between* frames depending on `Tl`, voicing, and the noise draw.
Without leveling, every frame would either clip or nearly vanish. The AGC is a
fast-attack / slow-release peak-envelope tracker that scales each frame toward
`TargetPeak` (default 18000, ~5.2 dB below full scale):

```go
// internal/voice/mbe/agc.go (shape)
if peak < a.env { coef = cfg.Release } else { coef = cfg.Attack } // 0.02 vs 0.5
a.env += (peak - a.env) * coef
gain := cfg.TargetPeak / max(a.env, cfg.NoiseFloor)
// per-frame ceiling: never push THIS frame's peak past the soft-limiter knee
if ceil := softLimitKnee / peak; gain > ceil { gain = ceil }
```

Samples above the knee (`0.80·32767 = 26213`) are `tanh`-compressed toward — but
never onto — the int16 rail, replacing a hard clip whose sharp corners injected
the harmonic distortion that read as "robotic."

<figure class="lab-figure">
<svg viewBox="0 0 640 190" width="640" height="190" role="img" aria-label="A transfer curve showing input samples passing through linearly below the soft-limiter knee and tanh-compressing toward the int16 rail above it, contrasted with a hard clip">
  <line x1="60" y1="150" x2="600" y2="150" stroke="var(--fg-muted)"/>
  <line x1="60" y1="150" x2="60" y2="20" stroke="var(--fg-muted)"/>
  <text x="330" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="10">input magnitude →</text>
  <text x="30" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="10" transform="rotate(-90 30 90)">output</text>
  <line x1="430" y1="150" x2="430" y2="40" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="430" y="164" text-anchor="middle" fill="var(--accent)" font-size="10">knee 26213</text>
  <line x1="60" y1="150" x2="600" y2="90" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="560" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="9">hard clip</text>
  <path d="M60,150 L430,58 Q520,36 600,32" fill="none" stroke="var(--accent)"/>
  <text x="360" y="72" text-anchor="middle" fill="var(--accent)" font-size="11">linear below knee</text>
  <text x="540" y="30" text-anchor="middle" fill="var(--accent)" font-size="10">tanh above → rail</text>
  <line x1="60" y1="40" x2="600" y2="40" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="96" y="36" text-anchor="middle" fill="var(--fg-muted)" font-size="9">±32767</text>
</svg>
<figcaption>Below the knee the AGC passes samples through unchanged; above it, tanh compression approaches the rail smoothly instead of the sharp corner a hard clip would leave.</figcaption>
</figure>

### The problem we hit here

The AGC has a `MinGain` of 0.05 for a reason field captures made painfully
clear. An early build set a gain *floor* too high (a floor of 10 forced ≥10×
amplification of already-loud frames straight into the limiter), and a
near-silent onset frame could seed a tiny envelope that then over-amplified the
*next* louder frame. Captures showed `mean_gain ≈ 1e5`, `peak_preclip ≈ 8e7`,
`clip_pct ≈ 65%` — the measurable signature of the "over-driven" reports. The
per-frame output ceiling (`softLimitKnee / peak`) is the one-sided clamp that
fixed it: it can only *reduce* gain, so ordinary frames and the fast-attack
inter-frame dynamics are untouched, but no frame can be shoved past the knee by
a stale envelope. Natural speech lands at a crest factor around 9.

## Loudness: a separate per-call gain

Everything above runs per frame. **Loudness normalization** runs once per
*call*, after the WAV is finished, and it is deliberately not a chain stage. It
mirrors ffmpeg's two-pass linear `loudnorm`: measure the whole call's
integrated loudness (EBU R128 / BS.1770), compute one linear gain toward
`TargetLUFS`, back it off if the true peak would exceed `TruePeakDBTP`, and
rewrite the file atomically:

```go
// internal/voice/normalize.go (shape)
gain, ok := loudness.NormalizeGain(fs, sampleRate, cfg.params())
if !ok { return nil } // silent / too short — leave the recording untouched
for i, v := range fs { samples[i] = clampInt16(v * gain * 32768.0) }
return rewriteWAV(path, samples, sampleRate)
```

Pure gain, no compression — so the within-call dynamics the AGC produced are
preserved, and a call is made *consistently* loud rather than *dynamically*
squashed. `MaxBoostDB` caps the gain so a near-silent call doesn't amplify hiss
up to target. It's off by default because it rewrites files; operators who
prioritise uniform playback loudness across a scanner feed turn it on.

## Where this goes next

The audio is now decoded, conditioned, and (optionally) leveled to a target.
[Part 11]({{ '/blog/deep-dives/voice-coding-11-recording-encoding/' | relative_url }})
takes it out of the decoder: WAV writing, pure-Go MP3 encoding with a Xing/LAME
header, Goertzel-based tone-out detection, and the optional DVSI hardware
vocoder fallback.

## FAQ

**Is spectral enhancement the same as the voice enhancer?**
No. Spectral enhancement (`EnhanceAmplitudes`, §6.2) is part of the vocoder spec
and runs on harmonic amplitudes before synthesis, preserving total energy. The
voice enhancer is an *opt-in* PCM chain (high-pass, shelf, low-pass, compressor)
that trades faithfulness for a cleaner, louder subjective sound.

**Why is the default output not "enhanced"?**
Because the default path aims to be faithful to the transmission — byte-for-byte
what the spec synthesis produces. The tone tilt, voice enhancer, and loudness
normalization are all opt-in nil pass-throughs, so enabling none of them leaves
output identical to the faithful decode.

**What causes "robotic" or over-driven audio?**
Historically, an AGC that over-amplified quiet frames and slammed the next loud
frame into a hard clip — measurable as a crest factor near 3–4 with high clip
percentage. The `tanh` soft-limiter and the per-frame output ceiling fixed it;
natural speech now sits around a crest factor of 9.

**What does loudness normalization do to dynamics?**
Nothing within a call — it applies a single linear gain toward a LUFS target
(with a true-peak backoff), so the call's internal loud/quiet dynamics survive.
It only makes different calls *consistently* loud relative to each other.

## Series navigation

**Part 10 of 12** · ←
[Part 9: The Composer]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
· Next →
[Part 11: Recording & Encoding]({{ '/blog/deep-dives/voice-coding-11-recording-encoding/' | relative_url }})
