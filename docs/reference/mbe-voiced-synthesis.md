---
slug: mbe-voiced-synthesis
title: MBE voiced synthesis
entry_type: term
category: voice-coding
description: "MBE voiced synthesis (TIA-102.BABA §6.3) sums one quadratic-phase sinusoid per voiced harmonic, interpolating amplitude and phase across each 160-sample frame so harmonics stay continuous through frame boundaries."
keywords: MBE voiced synthesis, IMBE 6.3, sinusoidal synthesis, quadratic phase, harmonic synthesis, amplitude interpolation, phase continuity, TIA-102.BABA, multi-band excitation
aka: ["voiced synthesis", "sinusoidal synthesis", "6.3 synthesis"]
autolink: true
infobox:
  - { label: Role, value: Voiced (periodic) waveform synthesis }
  - { label: Model, value: One sinusoid per voiced harmonic }
  - { label: Continuity, value: Amplitude + quadratic-phase interp }
  - { label: Spec, value: TIA-102.BABA §6.3 }
see_also: [multi-band-excitation, imbe, mbe-unvoiced-synthesis, mbe-spectral-enhancement, mbe-adaptive-smoothing]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Sinusoidal_model
  - https://en.wikipedia.org/wiki/Project_25
---

**MBE voiced synthesis** is the [IMBE](/reference/imbe/) decoder stage — TIA-102.BABA §6.3 — that
turns the voiced harmonic amplitudes into audio by summing **one sinusoid per voiced harmonic**,
each with amplitude and phase interpolated smoothly across the frame.[^mbe] The
[multi-band-excitation](/reference/multi-band-excitation/) model represents the periodic part of
speech as harmonics of a fundamental ω₀; §6.3 regenerates the corresponding oscillators and mixes
them, taking care that every harmonic stays continuous through the boundary between frames so the
output has no clicks.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="Across one 160-sample frame, each voiced harmonic is a cosine whose amplitude ramps linearly from its previous-frame value to its current-frame value and whose phase follows a quadratic curve, and the sum of all such harmonics forms the voiced output waveform." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.35"><line x1="30" y1="45" x2="440" y2="45"/><line x1="30" y1="95" x2="440" y2="95"/></g>
  <path d="M30 45 Q 90 25 150 45 T 270 45 T 390 45 T 450 45" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M30 95 Q 60 70 90 95 T 150 95 T 210 95 T 270 95 T 330 95 T 390 95 T 450 95" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="455" y="47" font-size="7.5" fill="currentColor" text-anchor="end">harmonic l</text>
  <text x="455" y="97" font-size="7.5" fill="currentColor" text-anchor="end">harmonic 2l</text>
  <text x="40" y="122" font-size="7.5" fill="currentColor">amplitude ramps + quadratic phase → Σ over voiced harmonics = s_v(n)</text>
</svg>
<figcaption>Each voiced harmonic is a cosine with a per-sample amplitude ramp and a quadratic phase track; the voiced output is their sum over the frame.</figcaption>
</figure>

## The §6.3 model

Over one output frame of `N = 160` samples (20 ms at 8 kHz), harmonic `l` is a cosine whose
amplitude and phase both move continuously from where they were last frame to where they are this
frame:

    a(n) = (1 − n/N)·M_prev[l] + (n/N)·M_curr[l]                       (eq. 88)
    θ(n) = θ_prev[l] + n·l·ω₀_prev + l·(ω₀_curr − ω₀_prev)·n²/(2N)
    s_v(n) = Σ  a(n)·cos(θ(n))                                          (eq. 89)

The amplitude term is a straight linear ramp between the previous and current linear amplitudes.
The phase term is **quadratic in n**: a linear part `l·ω₀_prev` (the frequency at the start of the
frame) plus a quadratic correction that sweeps the frequency toward `l·ω₀_curr` by the end, so the
oscillator's instantaneous frequency glides between the two pitch estimates rather than jumping.
GopherTrunk's `synthVoiced` in `internal/voice/mbe/synth_voiced.go` implements exactly this,
computing the linear coefficient `a = l·ω_prev` and quadratic coefficient
`b = l·(ω_curr − ω_prev)/(2N)` once per harmonic and accumulating `a(n)·cos(θ₀ + a·n + b·n²)`.

## Cross-frame continuity

The sum runs over every harmonic that is voiced this frame **or** was voiced last frame. That dual
condition is what makes voicing transitions clean: on an unvoiced→voiced onset `M_prev` is zero, so
the amplitude ramp fades the harmonic *in* over the frame; on a voiced→unvoiced offset `M_curr` is
zero, so it fades *out*. Neither produces the click a hard on/off switch would. On the very first
frame of a stream (`PrevW0 == 0`) the previous ω₀ is set equal to the current one, collapsing the
quadratic term to zero so the harmonic simply starts at its target frequency with no synthetic
sweep.

Continuity across frames depends on carrying phase memory forward correctly.
`UpdateVoicedState`, called after synthesis, advances each harmonic's stored phase by the
closed-form average-frequency increment `Δθ_l = N·l·(ω_prev + ω_curr)/2` (the integral of the
linear-frequency tilt over the frame), wraps it into `[0, 2π)`, and stores the current amplitudes
as next frame's `M_prev`. Keeping this memory phase-locked is essential — it is what lets harmonic
`l` pick up next frame exactly where `θ(n=N)` left off.

## Phase dispersion

Fully phase-coherent synthesis re-aligns every harmonic once per pitch period, which radiates a
buzzy impulse train — the "robotic" artifact of a naive MBE decoder. §6.3 addresses it by
regenerating the phase of the *upper* harmonics: for `l > L/4`, GopherTrunk's
`SynthVoicedDispersed` starts the frame from the coherent phase plus a bounded random offset
(`PHIl = PSIl + offset`), scaled by the unvoiced-harmonic fraction so mostly-voiced frames get
near-zero dispersion. The offset perturbs only that frame's synthesis phase and is deliberately
**not** folded into the phase memory, so it never accumulates into a random walk — the memory stays
coherent while the audible buzz is broken up.

## Relevance to SDR

Voiced synthesis is the half of MBE decoding that carries the pitched, sonorant character of
speech — vowels and voiced consonants. Together with the
[unvoiced synthesis](/reference/mbe-unvoiced-synthesis/) that supplies the noise-like bands, it
reconstructs the full [P25 Phase 1](/reference/imbe/) voice waveform in pure Go, no vendor DSP
chip. The amplitudes it renders are those already sharpened by
[spectral enhancement](/reference/mbe-spectral-enhancement/) and, on weak channels, cleaned by
[adaptive smoothing](/reference/mbe-adaptive-smoothing/).

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the harmonic (voiced) model synthesized by §6.3.
[^sin]: [Sinusoidal model](https://en.wikipedia.org/wiki/Sinusoidal_model) — Wikipedia, on additive sinusoidal speech synthesis with amplitude and phase interpolation.
