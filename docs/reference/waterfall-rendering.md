---
slug: waterfall-rendering
title: Waterfall rendering
entry_type: concept
category: sdr-app-building
description: "Waterfall rendering turns a stream of FFTs into a scrolling spectrogram image by mapping each bin's power in dB through a colormap, one row per frame, often on the GPU."
keywords: waterfall rendering, waterfall display, spectrogram, FFT to image, colormap, dB scaling, scrolling spectrogram, WebGL waterfall, GPU rendering, spectrum visualization
aka: ["waterfall display rendering", "scrolling spectrogram rendering"]
autolink: true
infobox:
  - { label: Type, value: "Spectrum-visualization pipeline" }
  - { label: Pipeline, value: "FFT → magnitude → dB → colormap → scroll" }
  - { label: Renders on, value: "CPU raster or GPU (WebGL/shader)" }
see_also: [spectrogram, power-spectral-density, fast-fourier-transform, gpu-dsp, waterfall-display]
cite_urls:
  - https://en.wikipedia.org/wiki/Spectrogram
  - https://en.wikipedia.org/wiki/Fast_Fourier_transform
---

**Waterfall rendering** is the process that turns a continuous stream of frequency
spectra into the scrolling, colored [spectrogram](/reference/spectrogram/) — the
"waterfall" — that dominates most SDR interfaces. Each fresh
[FFT](/reference/fast-fourier-transform/) becomes one horizontal line of the image:
frequency runs left to right, each pixel's color encodes signal power at that
frequency, and successive lines push older ones down (or up), so time flows down
the screen and a transmission traces a bright vertical streak.[^spec]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="An FFT magnitude spectrum has its bins converted to decibels, mapped through a colormap to produce one row of pixels, which is appended to a scrolling waterfall image while older rows shift down." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="wfrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <polyline points="16,60 26,58 36,52 46,30 56,52 66,57 76,40 86,58 96,60" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="56" y="74" font-size="7">FFT |X|² → dB</text>
    <line x1="100" y1="50" x2="132" y2="50" stroke="currentColor" stroke-width="1.1" marker-end="url(#wfrar)"/>
    <rect x="134" y="40" width="64" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="166" y="53" font-size="7">colormap</text>
    <line x1="198" y1="50" x2="230" y2="50" stroke="currentColor" stroke-width="1.1" marker-end="url(#wfrar)"/>
    <g stroke="currentColor" stroke-width="0.6">
      <rect x="232" y="42" width="8" height="14" fill="currentColor" fill-opacity="0.15"/><rect x="240" y="42" width="8" height="14" fill="currentColor" fill-opacity="0.3"/><rect x="248" y="42" width="8" height="14" fill="currentColor" fill-opacity="0.85"/><rect x="256" y="42" width="8" height="14" fill="currentColor" fill-opacity="0.3"/><rect x="264" y="42" width="8" height="14" fill="currentColor" fill-opacity="0.55"/><rect x="272" y="42" width="8" height="14" fill="currentColor" fill-opacity="0.15"/>
    </g>
    <text x="256" y="34" font-size="7">new row</text>
    <line x1="256" y1="60" x2="256" y2="78" stroke="currentColor" stroke-width="1.1" marker-end="url(#wfrar)"/>
    <rect x="228" y="80" width="56" height="70" fill="none" stroke="currentColor" stroke-width="1.1"/>
    <g stroke="currentColor" stroke-width="0.4">
      <rect x="228" y="80" width="56" height="10" fill="currentColor" fill-opacity="0.5"/>
      <rect x="228" y="90" width="56" height="10" fill="currentColor" fill-opacity="0.2"/>
      <rect x="228" y="100" width="56" height="10" fill="currentColor" fill-opacity="0.35"/>
      <rect x="228" y="110" width="56" height="10" fill="currentColor" fill-opacity="0.15"/>
    </g>
    <text x="330" y="112" font-size="7" text-anchor="start">older rows scroll down</text>
  </g>
</svg>
<figcaption>Each FFT frame is scaled to dB, colored, and appended as one row; the accumulated rows form a scrolling time-frequency image.</figcaption>
</figure>

## How it works

The pipeline is a fixed sequence of stages, repeated once per frame:

1. **FFT.** A block of IQ samples is windowed (to control spectral leakage) and
   transformed to the frequency domain, yielding one complex value per bin.
2. **Magnitude and dB.** Each bin's magnitude squared gives its
   [power](/reference/power-spectral-density/); taking `10·log10` compresses the
   enormous dynamic range of RF into decibels, so a weak signal 60 dB below a
   strong one is still visible.
3. **Normalize to a range.** The dB values are clamped between a floor and a ceiling
   (often user-adjustable "contrast/brightness"), mapping the interesting band into
   the 0–1 range the colormap expects.
4. **Colormap.** That 0–1 value indexes a color gradient — a lookup table — turning
   each bin into a pixel. Perceptually uniform maps (viridis, inferno) are
   preferred over rainbow maps because equal power steps look like equal color
   steps.
5. **Scroll and blit.** The new row is written to the display and previous rows are
   shifted, commonly by treating the image as a ring buffer of rows so no pixels are
   actually copied — only the starting offset moves.

## In practice

The rate mismatch between data and eyes drives most design choices. FFTs may arrive
hundreds of times a second, far faster than a useful scroll; renderers therefore
*average or decimate* frames per output row, or accept every FFT but scroll slowly.
Averaging several FFTs per row (Welch-style) also smooths the noise so faint
carriers stand out. The visible frequency resolution is set by FFT size, and the
time resolution by how many samples each frame covers — the classic time-frequency
trade.

Rendering is a natural fit for the [GPU](/reference/gpu-dsp/): the FFT output is
uploaded as a texture and a fragment shader applies the colormap, so the CPU never
touches individual pixels. Browser SDR clients do exactly this with WebGL — the
colormap becomes a 1-D texture lookup in the shader and the scroll is a texture
coordinate offset, letting a full-width waterfall run at display refresh rate on
modest hardware. On the CPU, the same result is achieved with a precomputed color
lookup table and a ring-buffer image.

## Relevance to SDR

The waterfall is the signature visualization of software radio: it makes bursts,
frequency-hopping systems, trunking control channels, and interference immediately
legible in a way a single instantaneous [spectrum](/reference/spectrogram/) trace
cannot, because it preserves history. Essentially every SDR GUI — SDR#, GQRX,
SDRangel, CubicSDR, and [web-based](/reference/web-sdr/) clients — is built around
one, and the rendering quality (colormap choice, dB scaling, frame averaging)
strongly affects how weak a signal a human can spot.

**GopherTrunk** is primarily a headless decoding engine rather than a spectrum GUI,
so heavy interactive waterfall rendering is not its focus — it spends its compute
locking and decoding trunking traffic, not painting pixels. The underlying
[FFT](/reference/fast-fourier-transform/) and dB-[power](/reference/power-spectral-density/)
math it performs for signal detection and diagnostics is exactly the front half of
the waterfall pipeline; the colormap-and-scroll back half belongs to a UI layer,
and GT relates it to the broader ecosystem of SDR front-ends rather than shipping a
GPU renderer of its own.

## Sources

[^spec]: [Spectrogram](https://en.wikipedia.org/wiki/Spectrogram) — Wikipedia, on the time-frequency-intensity image that a waterfall renders row by row.
[^fft]: [Fast Fourier transform](https://en.wikipedia.org/wiki/Fast_Fourier_transform) — Wikipedia, on the transform that produces each spectrum row of the display.
