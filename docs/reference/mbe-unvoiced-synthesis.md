---
slug: mbe-unvoiced-synthesis
title: MBE unvoiced synthesis
entry_type: term
category: voice-coding
description: "MBE unvoiced synthesis (TIA-102.BABA §6.4) excites the noise-like bands: a 256-point FFT of white noise is zeroed under the voiced harmonics, scaled under unvoiced harmonics, inverse-transformed, and windowed into a click-free overlap-add."
keywords: MBE unvoiced synthesis, IMBE 6.4, noise excitation, 256-point FFT, spectrum shaping, overlap-add, synthesis window, Tukey window, TIA-102.BABA, multi-band excitation, FFT
aka: ["unvoiced synthesis", "noise-band synthesis", "6.4 synthesis"]
autolink: true
infobox:
  - { label: Role, value: Unvoiced (noise) band synthesis }
  - { label: Transform, value: 256-point FFT of white noise }
  - { label: Overlap, value: "96-sample power-complementary window" }
  - { label: Spec, value: TIA-102.BABA §6.4 }
see_also: [multi-band-excitation, fast-fourier-transform, imbe, mbe-voiced-synthesis, mbe-spectral-enhancement, discrete-fourier-transform]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Overlap%E2%80%93add_method
  - https://en.wikipedia.org/wiki/Fast_Fourier_transform
---

**MBE unvoiced synthesis** is the [IMBE](/reference/imbe/) decoder stage — TIA-102.BABA §6.4 —
that fills the **noise-like** portion of the speech spectrum, the bands the
[multi-band-excitation](/reference/multi-band-excitation/) model marked *unvoiced*.[^mbe] Where the
[voiced synthesis](/reference/mbe-voiced-synthesis/) generates deterministic sinusoids, §6.4 shapes
filtered white noise: it takes a [FFT](/reference/fast-fourier-transform/) of a noise block, keeps
only the frequency bins that fall under unvoiced harmonics (scaled by those harmonics'
amplitudes), zeros the rest, transforms back, and stitches successive frames together with a
windowed overlap-add so nothing clicks at the boundaries.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 140" role="img" aria-label="White noise is forward-transformed to a 256-point spectrum, bins under voiced harmonics are zeroed while bins under unvoiced harmonics are scaled by the harmonic amplitude, the shaped spectrum is inverse-transformed, and the result is windowed and overlap-added with the previous frame." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="uvar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" fill="none" font-size="7">
    <rect x="12" y="52" width="60" height="30"/><rect x="96" y="52" width="52" height="30"/><rect x="172" y="52" width="86" height="30"/><rect x="282" y="52" width="52" height="30"/><rect x="358" y="52" width="96" height="30"/>
  </g>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <text x="42" y="70">white noise</text>
    <text x="122" y="67">256-pt</text><text x="122" y="77">FFT</text>
    <text x="215" y="64">zero voiced bins,</text><text x="215" y="74">scale unvoiced ×Ml</text>
    <text x="308" y="67">IFFT</text>
    <text x="406" y="64">window +</text><text x="406" y="74">overlap-add</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="72" y1="67" x2="96" y2="67" marker-end="url(#uvar)"/>
    <line x1="148" y1="67" x2="172" y2="67" marker-end="url(#uvar)"/>
    <line x1="258" y1="67" x2="282" y2="67" marker-end="url(#uvar)"/>
    <line x1="334" y1="67" x2="358" y2="67" marker-end="url(#uvar)"/>
  </g>
</svg>
<figcaption>The unvoiced band is filtered white noise: shape a 256-point noise spectrum to the unvoiced harmonics, invert it, and overlap-add successive windowed frames.</figcaption>
</figure>

## Shaping the noise spectrum

`SynthUnvoicedFromNoise` in `internal/voice/mbe/synth_unvoiced.go` runs the core pipeline:
interpret a length-256 noise buffer as a real time signal, forward-FFT it, shape the spectrum, and
inverse-FFT back. The shaping step, `ShapeUnvoicedSpectrum`, classifies every FFT bin by the
harmonic it falls under using IMBE's nearest-centre rule, `l = round(2π·k_eff / (N·ω₀))`, and then:

- **zeros** the bin if the harmonic is voiced, or if it is outside the modelled range `[1..L]`;
- **scales** the bin by that harmonic's linear amplitude `Ml[l]` if the harmonic is unvoiced.

The mapping uses each bin's *effective* frequency — bins above `N/2` are mirrored back through the
conjugate symmetry — so both halves of a `(k, N−k)` pair get the same real scale factor. That keeps
the spectrum Hermitian-symmetric and guarantees the inverse transform is real-valued. The
[FFT](/reference/fast-fourier-transform/) length is fixed at **256** (`UnvoicedFFTSize`): long
enough that each harmonic band spans several bins, short enough to stay cheap.

## Windowed overlap-add

A 256-point frame is produced every 160 samples, so consecutive frames overlap by
`UnvoicedTailSamples = 96` samples. The production path, `SynthUnvoicedOverlapAdd`, emits the
previous frame's stored 96-sample tail into the start of the current frame, computes the new
windowed frame, adds its first 160 samples to the output, and stashes its last 96 for next time —
threading the overlap state through `SynthState.PrevUnvoicedTail`.

The synthesis window is chosen with care. Consecutive frames carry *independent* noise blocks, so
in the overlap region their variances **add** rather than their amplitudes. A plain periodic Hann
window at this 160-sample hop leaves the noise-power envelope rippling ~7 dB, which leaks through as
a 50 Hz frame-rate amplitude modulation — an audible buzzy tremolo on fricatives. §6.4's window is
instead **power-complementary** at this hop: a flat unity top over the non-overlapping centre
`[96, 160)` with a quarter-wave sine taper of length 96 on each edge. Because a sine taper satisfies
`w[n]² + w[95−n]² = 1`, the summed noise power `P[n]` is flat to within 1e-9 across the whole frame,
so the tremolo vanishes while the taper still decays to zero at the edges to suppress boundary
clicks — and its flat top passes more aspiration energy than the Hann did.

## Where it sits

Unvoiced synthesis is summed into the same output buffer as the voiced synthesis (both stages *add*
rather than overwrite), so a frame's noise bands and harmonic bands combine into one waveform. Its
input amplitudes are the same enhanced, smoothed `Ml` the voiced path uses. Silent and zero-*L*
frames still emit the previous tail so a voiced→silent transition fades cleanly, then clear the
tail so the next frame starts fresh.

## Relevance to SDR

The unvoiced band is what makes decoded speech sound like speech and not a buzzer: the breathy
noise of fricatives (s, f, sh) and aspiration lives entirely here. Getting the overlap-add window
right is a concrete example of a synthesis detail that is inaudible in a spectrum plot but obvious
to the ear — it was one of the artifacts examined while chasing GopherTrunk's "machine voice"
report. Together with the [voiced synthesis](/reference/mbe-voiced-synthesis/) it completes the
pure-Go [P25 Phase 1](/reference/imbe/) voice decoder.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the voiced/unvoiced band model whose unvoiced part §6.4 synthesizes.
[^ola]: [Overlap–add method](https://en.wikipedia.org/wiki/Overlap%E2%80%93add_method) — Wikipedia, on stitching windowed frames into a continuous signal.
