---
title: "Voice Coding, Part 3: MBE Synthesis — Voiced Sinusoids + Unvoiced Noise"
description: How GopherTrunk grows 160 samples of speech from an MBE parameter set — summed sinusoids for the voiced bands, FFT-shaped noise for the unvoiced bands, and the cross-frame phase and overlap-add smoothing that kills clicks.
category: deep-dives
keywords: mbe synthesis, voiced unvoiced synthesis, imbe speech synthesis, sinusoidal synthesis vocoder, overlap add unvoiced, cross frame phase continuity, spectral amplitude prediction, gophertrunk mbe synth
tags: [voice, mbe, dsp, synthesis, imbe, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 3
---

*Part 3 of **Voice Coding**. Part 2 gave us the MBE parameter set — a pitch, a
per-harmonic voicing decision, and spectral residuals. This post spends those
parameters: it grows 160 samples of 8 kHz speech from them, which is the moment a
list of numbers becomes a voice.*

> **TL;DR:** Synthesis runs `mbe.Params` through four stages on a per-call
> `SynthState`. First it recovers the real harmonic log-amplitudes from the
> transmitted residual `Tl` by **predicting across frames** (`PredictLog2Ml`).
> Then it splits the harmonics: **voiced** bands become summed sinusoids with
> phase carried coherently across frame boundaries (`SynthVoiced`), **unvoiced**
> bands become white noise FFT'd, shaped to the harmonic envelope, and IFFT'd
> back (`SynthUnvoicedOverlapAdd`). A power-complementary window overlap-adds the
> noise across frames so boundaries don't click. The two contributions sum into
> one buffer.

**Key takeaways**

- The transmitted `Tl` is a *residual*; `PredictLog2Ml` interpolates the previous
  frame's envelope to a pitch-scaled prediction, adds `Tl`, and removes the
  prediction's DC bias — that's how `Tl` becomes real `log2(Ml)`.
- Voiced harmonics are summed cosines with a **linear amplitude tilt** and a
  **quadratic phase term** across the 160 samples, so pitch glides and fade in/out
  are click-free.
- Unvoiced bands are synthesized in the frequency domain: noise → FFT → zero the
  voiced bins, scale the unvoiced bins by `Ml` → IFFT.
- Frame boundaries are healed two ways: **coherent phase memory** for voiced,
  **overlap-add with a power-complementary window** for unvoiced.

## Cheat sheet

| Stage | Function (`internal/voice/mbe/…`) | Role |
|---|---|---|
| Amplitude recovery | `PredictLog2Ml` (`synth.go`) | `Tl` + prev-frame prediction → `log2(Ml)` |
| State roll-forward | `UpdateLog2Ml` / `UpdateVoicedState` | store this frame's pitch/amp/phase for the next |
| Voiced excitation | `SynthVoiced` (`synth_voiced.go`) | summed sinusoids, phase-coherent across frames |
| Unvoiced excitation | `SynthUnvoicedOverlapAdd` (`synth_unvoiced.go`) | noise → FFT → shape → IFFT → overlap-add |
| Spectrum shaping | `ShapeUnvoicedSpectrum` | zero voiced/out-of-range bins, scale unvoiced by `Ml` |
| Inter-frame memory | `SynthState` (`synth.go`) | prev ω₀, L, log2Ml, Ml, phase, unvoiced tail |

## In this post

- **From residual to amplitude** — the cross-frame prediction, and the
  female-voice bug it fixed.
- **Voiced synthesis** — summed sinusoids with amplitude tilt and quadratic phase.
- **Unvoiced synthesis** — noise in the frequency domain, shaped to the envelope.
- **Healing the seams** — coherent phase and the overlap-add window.

## From residual to amplitude

Part 2 left `Tl[1..L]` as a *residual* — the difference between each harmonic's
log-amplitude and a prediction. The first synthesis stage reconstitutes the real
amplitudes. `PredictLog2Ml` implements TIA-102.BABA §6.1 equations 75–77:

```go
// internal/voice/mbe/synth.go (shape)
func PredictLog2Ml(s *SynthState, p Params, dst *[57]float64) {
    if p.Silent || p.L == 0 { return }
    L := p.L
    var pred [57]float64
    if s.PrevL > 0 && s.PrevW0 > 0 {
        gain := predictionGain(L)          // pitch-dependent ρ
        ratio := p.W0 / s.PrevW0
        for l := 1; l <= L; l++ {
            pos := float64(l) * ratio      // where this harmonic sat last frame
            // … clamp to [1, PrevL], linearly interpolate PrevLog2Ml at pos …
            pred[l] = gain * interp(s.PrevLog2Ml, pos)
        }
    }
    var ave float64
    for l := 1; l <= L; l++ { ave += pred[l] }
    ave /= float64(L)
    for l := 1; l <= L; l++ { dst[l] = pred[l] + p.Tl[l] - ave } // eq. 77
}
```

The idea: a harmonic at index `l` this frame sat near index `l · ω₀_curr/ω₀_prev`
last frame (pitch changed, so the harmonics moved). Interpolate the previous
frame's `log2(Ml)` at that position, scale it by a prediction gain ρ, average the
prediction across the frame, and combine `prediction + Tl − average` so the
residual carries the only intentional offset. On the very first frame there's no
history, so the prediction term is zero and `log2(Ml)[l] = Tl[l]`.

### The problem we hit: one prediction gain for every pitch

That `predictionGain(L)` call hides a real bug fix. The prediction gain ρ controls
how much the previous frame's envelope leaks into this one. GopherTrunk originally
**hardcoded ρ = 0.65 for every frame**. That's fine for a low-pitched voice with
many, closely-spaced harmonics — the envelope really is stationary frame to frame.
But a high-pitched voice has *few*, widely-spaced harmonics, and a strong
prediction over-smears the previous (differently-spaced) envelope onto the current
one. The audible result was poor female-voice intelligibility compared to OP25 and
Trunk Recorder.

The fix is to make ρ pitch-dependent, matching the reference decoders:

```go
// internal/voice/mbe/synth.go
func predictionGain(L int) float64 {
    switch {
    case L <= 15: return 0.4               // high pitch (e.g. female voices)
    case L <= 24: return 0.03*float64(L) - 0.05
    default:      return 0.7               // low pitch
    }
}
```

Low-L (high-pitch) frames now get a weak ρ = 0.4, so the current frame's own `Tl`
dominates and the mismatched previous envelope doesn't wash it out. It's a
three-line switch, but it's the difference between a codec that sounds broken on
half the population and one that doesn't.

## Voiced synthesis: summed sinusoids

Once `log2(Ml)` is linearized to `Ml` (a separate `AmplitudesFromLog2Ml` step),
the harmonics split on `Vl[l]`. Voiced harmonics go to `SynthVoiced`, which sums
one cosine per harmonic across the 160-sample frame — TIA-102.BABA §6.3:

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Three sinusoids at the fundamental frequency and its harmonics, each with its own amplitude, are summed to produce the voiced speech waveform for one frame">
  <text x="20" y="20" fill="var(--accent)" font-size="11">voiced harmonics l·ω₀, amplitude Ml</text>
  <path d="M40 55 q15 -20 30 0 q15 20 30 0 q15 -20 30 0 q15 20 30 0 q15 -20 30 0 q15 20 30 0 q15 -20 30 0 q15 20 30 0" fill="none" stroke="var(--accent)"/>
  <text x="360" y="52" fill="var(--fg-muted)" font-size="10">l = 1  (fundamental)</text>
  <path d="M40 100 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0 q7.5 -14 15 0 q7.5 14 15 0" fill="none" stroke="var(--accent)" opacity="0.7"/>
  <text x="360" y="97" fill="var(--fg-muted)" font-size="10">l = 2</text>
  <path d="M40 140 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0 q5 -9 10 0 q5 9 10 0" fill="none" stroke="var(--accent)" opacity="0.5"/>
  <text x="360" y="137" fill="var(--fg-muted)" font-size="10">l = 3</text>
  <line x1="300" y1="170" x2="340" y2="170" stroke="currentColor"/>
  <text x="320" y="188" text-anchor="middle" fill="currentColor" font-size="14">Σ</text>
  <path d="M430 170 q10 -22 18 4 q8 16 16 -8 q10 -14 18 10 q8 12 16 -6 q10 -18 18 6 q8 14 16 -4" fill="none" stroke="currentColor"/>
  <text x="560" y="167" fill="currentColor" font-size="10">= voiced s_v(n)</text>
</svg>
<figcaption>Each voiced harmonic is a cosine at l·ω₀ scaled by its amplitude Ml; summing them across the frame rebuilds the voiced part of the waveform. Their relative amplitudes are the spectral envelope from Part 2.</figcaption>
</figure>

The inner loop carries two subtleties that keep it click-free across frame
boundaries:

```go
// internal/voice/mbe/synth_voiced.go (shape) — per voiced harmonic l
a := lf * prevW0                      // linear phase coeff
b := lf * (p.W0 - prevW0) * invDoubleN // quadratic: integrates the ω₀ glide
dAmp := currAmp - prevAmp
for n := 0; n < N; n++ {              // N = 160
    amp   := prevAmp + dAmp*(float64(n)*invN)  // amplitude tilt across frame
    phase := thetaBase + a*float64(n) + b*float64(n)*float64(n)
    dst[n] += amp * math.Cos(phase)
}
```

- **Amplitude tilt.** The harmonic fades linearly from its *previous*-frame
  amplitude to its *current*-frame amplitude across the 160 samples. A harmonic
  that was unvoiced last frame starts at zero and fades in; one going unvoiced this
  frame fades out to zero. No amplitude step, no click.
- **Quadratic phase.** The phase is `θ₀ + a·n + b·n²`. The `n²` term integrates
  the linear frequency glide from `ω₀_prev` to `ω₀_curr`, so a rising pitch sweeps
  smoothly instead of jumping at the frame edge. `thetaBase` comes from
  `SynthState.PrevPhase[l]` — the phase where this harmonic *left off* last frame —
  which is what "phase-coherent" means. After the frame, `UpdateVoicedState`
  advances that phase memory by the closed-form average-frequency increment so the
  next frame picks up exactly where this one ended.

There's a subtlety even here: fully phase-coherent synthesis of *every* harmonic
sounds buzzy and robotic, because re-aligning all harmonics once per pitch period
radiates an impulse train. The IMBE path therefore uses `SynthVoicedDispersed`,
which perturbs the upper harmonics' *synthesis* phase by a bounded random offset
(scaled by the unvoiced fraction) — applied to this frame's excitation only, never
folded into the coherent memory. That's the difference between "de-buzzed" and "a
random walk into noise."

## Unvoiced synthesis: noise in the frequency domain

Unvoiced bands can't be sinusoids — they're hiss. §6.4 synthesizes them in the
frequency domain:

<figure class="lab-figure">
<svg viewBox="0 0 680 130" width="680" height="130" role="img" aria-label="Unvoiced synthesis pipeline: white noise is FFT'd, bins under voiced or out-of-range harmonics are zeroed while unvoiced bins are scaled by their amplitude, then inverse-FFT'd back to a time-domain noise band">
  <rect x="10" y="45" width="96" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="58" y="68" text-anchor="middle" fill="currentColor" font-size="11">white noise</text>
  <line x1="106" y1="65" x2="140" y2="65" stroke="currentColor"/><polygon points="140,61 150,65 140,69" fill="currentColor"/>
  <rect x="150" y="45" width="80" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="190" y="68" text-anchor="middle" fill="currentColor" font-size="11">FFT 256</text>
  <line x1="230" y1="65" x2="264" y2="65" stroke="currentColor"/><polygon points="264,61 274,65 264,69" fill="currentColor"/>
  <rect x="274" y="35" width="150" height="60" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="349" y="56" text-anchor="middle" fill="var(--accent)" font-size="11">ShapeUnvoiced</text>
  <text x="349" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">zero voiced bins</text>
  <text x="349" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="9">scale unvoiced × Ml</text>
  <line x1="424" y1="65" x2="458" y2="65" stroke="currentColor"/><polygon points="458,61 468,65 458,69" fill="currentColor"/>
  <rect x="468" y="45" width="80" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="508" y="68" text-anchor="middle" fill="currentColor" font-size="11">IFFT</text>
  <line x1="548" y1="65" x2="582" y2="65" stroke="currentColor"/><polygon points="582,61 592,65 582,69" fill="currentColor"/>
  <rect x="592" y="45" width="78" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="631" y="63" text-anchor="middle" fill="var(--accent)" font-size="10">noise band</text>
  <text x="631" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ overlap-add</text>
</svg>
<figcaption>Unvoiced bands are built in the frequency domain: FFT white noise, keep only the bins that fall under unvoiced harmonics (scaled by their amplitude), zero the rest, and IFFT back. The result carries the same spectral envelope as the voiced part but with no pitch.</figcaption>
</figure>

`ShapeUnvoicedSpectrum` does the classification, snapping each FFT bin to its
nearest harmonic centre `l = round(2π·k / (N·ω₀))` and either zeroing it (voiced
harmonic, or outside `[1..L]`) or scaling it by `Ml[l]` (unvoiced harmonic). It's
careful to scale the `(k, N−k)` conjugate pair by the same real value so the IFFT
stays real-valued. The noise itself is passed *in* rather than pulled from a source
inside the function — a deliberate choice so unit tests can feed deterministic
noise and pin the output exactly.

## Healing the seams: the overlap-add window

Consecutive frames carry **independent** noise blocks, so if you just butted each
frame's 160-sample IFFT against the next you'd get a click at every boundary. §6.4
fixes this with overlap-add: the FFT is 256 points (not 160), and each frame's
windowed IFFT extends 96 samples into the next frame's output, where it's summed.
`SynthState.PrevUnvoicedTail[96]` carries that overlap.

The window choice is subtle enough to have its own regression test. A naive
periodic Hann at the 160-sample hop makes the noise-power envelope ripple ~7 dB,
which leaks through as a **50 Hz buzzy tremolo** on fricatives — one of the
artifacts chased in the issue #644 "machine voice" report. The fix is a
**power-complementary** (tapered-cosine) window: a flat unity top over the
non-overlapping centre and a quarter-wave sine taper of length 96 on each edge.
With a sine taper, `w[n]² + w[95−n]² = 1`, so the summed noise power is exactly
flat across the frame (verified to `<1e-9`) while the edges still decay to zero to
suppress boundary clicks — *and* the flat top passes more aspiration energy than
the Hann did.

The two contributions — voiced sinusoids and unvoiced noise band — accumulate into
the **same** `dst` buffer (both functions add rather than overwrite), and that sum
is the frame's 160 samples of speech.

### How the state design shaped the Go

The through-line here is `SynthState`: prev ω₀, prev L, prev `log2Ml`, prev `Ml`,
prev phase, prev unvoiced tail. Every stateful stage reads it and a matching
`Update…`/roll-forward writes it, and the whole thing is **owned per call** by the
decoder — never shared. That's the same non-shared-state discipline the registry
enforced in Part 1, applied one level down: two parallel calls each have their own
`SynthState`, so their phase memories can't collide. `Reset()` (a plain
`*s = SynthState{}`) wipes it on a stream re-sync or a silence frame, and the zero
value *is* the valid "fresh stream, no previous frame" state — which is why the
first-frame prediction naturally collapses to `Tl`.

## Where this goes next

The synthesizer is codec-blind — it only ever sees `mbe.Params`. The next three
parts fill that struct in for IMBE.
[Part 4]({{ '/blog/deep-dives/voice-coding-04-imbe-decode/' | relative_url }})
unpacks the 4.4 kbps IMBE frame: the `b_0` pitch codeword, the PRBA gain blocks,
the DCT spectral coefficients, and the scrambler that whitens the whole thing.
Parts 5–6 then handle the FEC and de-interleave that get clean bits to unpack, and
the acquisition squelch that stops the synthesizer voicing garbage during symbol
lock.

## FAQ

**Why does MBE synthesize voiced and unvoiced bands with completely different
methods?**
Because they're physically different sound sources. Voiced bands are periodic
(pitched buzz), best rebuilt as summed sinusoids at the pitch harmonics; unvoiced
bands are aperiodic (hiss), best rebuilt as shaped noise. Using one method for
both would make vowels noisy or fricatives buzzy.

**What is cross-frame prediction and why is it needed?**
The codec transmits each harmonic's amplitude as a *residual* from a prediction
based on the previous frame, to save bits. `PredictLog2Ml` rebuilds the prediction
— interpolating the previous envelope to the new pitch — and adds the residual
back, recovering the real amplitude.

**Why did female voices sound worse before the fix?**
A single hardcoded prediction gain (0.65) over-smeared the previous frame's
widely-spaced envelope onto high-pitch frames. Making the gain pitch-dependent
(0.4 for few-harmonic, high-pitch frames) let the current frame's own data
dominate, matching OP25 and Trunk Recorder.

**Why 256-point FFT for a 160-sample frame?**
The extra 96 samples are the overlap-add region. Each frame's windowed noise
extends into the next frame's output and is summed there, so independent noise
blocks join without a click at the boundary.

**Why not a plain Hann window on the unvoiced band?**
A periodic Hann at the 160-sample hop isn't power-complementary — the summed noise
power ripples ~7 dB, audible as a 50 Hz tremolo on fricatives. A tapered-cosine
window with a flat top and sine edges keeps the summed power exactly flat while
still suppressing boundary clicks.

## Series navigation

**Part 3 of 12** · ←
[Part 2: The MBE Model]({{ '/blog/deep-dives/voice-coding-02-the-mbe-model/' | relative_url }})
· Next →
[Part 4: IMBE Decode]({{ '/blog/deep-dives/voice-coding-04-imbe-decode/' | relative_url }})
