---
slug: mbe-spectral-enhancement
title: MBE spectral-amplitude enhancement
entry_type: term
category: voice-coding
description: "MBE spectral-amplitude enhancement (TIA-102.BABA §6.2) weights each harmonic amplitude by a factor derived from the spectral moments R_M0 and R_M1 to sharpen formants before synthesis, then rescales to preserve frame energy."
keywords: MBE spectral enhancement, spectral amplitude enhancement, IMBE 6.2, formant sharpening, spectral moments, R_M0 R_M1, per-harmonic weight, TIA-102.BABA, multi-band excitation
aka: ["spectral-amplitude enhancement", "amplitude enhancement", "6.2 enhancement"]
autolink: true
infobox:
  - { label: Role, value: Sharpen formants pre-synthesis }
  - { label: Driver, value: "Spectral moments R_M0, R_M1" }
  - { label: Weight clamp, value: "W ∈ [0.5, 1.2]" }
  - { label: Spec, value: TIA-102.BABA §6.2 }
see_also: [multi-band-excitation, imbe, mbe-voiced-synthesis, mbe-unvoiced-synthesis, mbe-adaptive-smoothing, imbe-parameter-quantization]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Formant
---

**MBE spectral-amplitude enhancement** is the [IMBE](/reference/imbe/) decoder stage — TIA-102.BABA
§6.2 — that reshapes the recovered per-harmonic amplitudes just before synthesis, boosting the
harmonics the model under-represents so the spectral envelope tilts more naturally on
playback.[^mbe] After the quantizer round-trip, mid-band harmonics tend to have some of their
energy averaged into neighbouring bands; §6.2 restores the peaks by multiplying each harmonic
amplitude by a weight derived from the frame's **spectral moments**, then renormalizing so total
frame energy is unchanged.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="A harmonic amplitude envelope with softened formant peaks is sharpened by a per-harmonic weight, so the enhanced envelope has taller peaks and deeper valleys while its total area, the frame energy, is held constant." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-opacity="0.4"><line x1="30" y1="100" x2="220" y2="100"/><line x1="250" y1="100" x2="440" y2="100"/></g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="55" y1="100" x2="55" y2="78"/><line x1="80" y1="100" x2="80" y2="62"/><line x1="105" y1="100" x2="105" y2="70"/><line x1="130" y1="100" x2="130" y2="58"/><line x1="155" y1="100" x2="155" y2="72"/><line x1="180" y1="100" x2="180" y2="80"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="275" y1="100" x2="275" y2="82"/><line x1="300" y1="100" x2="300" y2="50"/><line x1="325" y1="100" x2="325" y2="74"/><line x1="350" y1="100" x2="350" y2="44"/><line x1="375" y1="100" x2="375" y2="76"/><line x1="400" y1="100" x2="400" y2="84"/>
  </g>
  <text x="125" y="118" text-anchor="middle" font-size="8" fill="currentColor">recovered amplitudes</text>
  <text x="345" y="118" text-anchor="middle" font-size="8" fill="currentColor">after §6.2 (energy preserved)</text>
  <path d="M225 70 L250 70" stroke="currentColor" stroke-width="1.2" marker-end="url(#enar)"/>
  <defs><marker id="enar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Enhancement raises formant peaks and lowers the valleys between them; a final rescale keeps total harmonic energy equal to the pre-enhancement frame energy.</figcaption>
</figure>

## Spectral moments

The whole stage is driven by two scalar summaries of the frame's amplitude spectrum, the
**spectral moments**:

- `R_M0 = Σ Ml²` — the integrated power across all *L* harmonics (the frame energy).
- `R_M1 = Σ Ml² · cos(ω₀·l)` — the same power weighted by the cosine of each harmonic's angular
  frequency, so it captures the spectral *tilt*.

GopherTrunk computes these once per frame (`FrameEnergy` and `SpectralCosineSum`) and reuses them
for every harmonic's weight, which is what keeps the stage cheap.

## The per-harmonic weight

`EnhanceAmplitudes` in `internal/voice/mbe/enhance.go` walks harmonics `l = 1..L` and multiplies
each amplitude `M[l]` by a weight `W_l`:

- **Low band (`8·l ≤ L`)** and the top of the band are left at `W_l = 1` — the model already
  represents them well, so they are untouched.
- **Mid band**, with `c = cos(ω₀·l)`:

      num = R_M0² + R_M1² − 2·R_M0·R_M1·c
      den = R_M0 · (R_M0² − R_M1²)
      ξ   = 0.96 · num / den
      W_l = ξ^0.25,  clamped to [0.5, 1.2]

The exponent of ¼ makes the weight a gentle contour rather than a hard boost, and the clamp
`EnhanceWMin = 0.5`, `EnhanceWMax = 1.2` bounds it so a near-pure-tone frame — where
`R_M0² − R_M1²` approaches zero and the ratio would blow up — cannot produce a runaway multiplier.
A degenerate frame (`den ≤ 0`, which by Cauchy-Schwarz means a single dominant harmonic) simply
gets `W_l = 1` and is skipped.

## Energy preservation

Reshaping the envelope redistributes energy between harmonics, which would otherwise change the
frame's loudness and make enhanced frames jump in level against un-enhanced ones. §6.2 closes with
a renormalization: after the per-harmonic multiply, the stage recomputes the enhanced energy and
scales every amplitude by `sqrt(R_M0_orig / R_M0_enhanced)`, so the integrated power `R_M0` is
exactly restored. The formants are sharper but the frame is neither louder nor quieter — the
enhancement changes the *shape* of the spectrum, not its total energy. Silent frames, zero-*L*
frames, and all-zero-amplitude frames are no-ops, so the synthesis path can call
`EnhanceAmplitudes` unconditionally.

## Where it sits

Enhancement runs after §6.1 cross-frame log-amplitude recovery and the log-to-linear amplitude
conversion, and before synthesis: the enhanced amplitudes feed both the
[voiced sinusoidal synthesis](/reference/mbe-voiced-synthesis/) and the
[unvoiced noise synthesis](/reference/mbe-unvoiced-synthesis/). It is a perceptual polish on the
decoded [MBE](/reference/multi-band-excitation/) model parameters, distinct from the
error-driven [adaptive smoothing](/reference/mbe-adaptive-smoothing/) that cleans up *corrupted*
parameters on a weak channel.

## Relevance to SDR

For a scanner the payoff is speech that sounds like the reference [P25 Phase 1](/reference/imbe/)
decoder rather than a flat, muffled approximation — the formant contrast §6.2 restores is a large
part of what makes decoded voice intelligible and natural. Because it is a specified part of the
IMBE standard, GopherTrunk implements the exact closed form rather than an ad-hoc equalizer, with
the constants cross-checked against public reference decoders.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE/IMBE vocoder whose decoder includes the §6.2 enhancement.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard that specifies IMBE (TIA-102.BABA).
