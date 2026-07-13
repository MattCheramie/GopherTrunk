---
slug: window-function
title: Window function
entry_type: algorithm
category: algorithms
description: A window function tapers a signal block before the FFT to suppress spectral leakage, trading frequency resolution for lower side-lobes in SDR spectra.
keywords: window function, spectral leakage, Hann window, Hamming window, Blackman-Harris, Kaiser window, Nuttall window, main lobe, side lobe, FFT windowing, apodization
aka: [window function, windowing, apodization, taper function]
autolink: true
infobox:
  - { label: Type, value: Pre-FFT taper }
  - { label: Reduces, value: Spectral leakage / side-lobes }
  - { label: Cost, value: Wider main lobe (coarser resolution) }
see_also: [fast-fourier-transform, discrete-fourier-transform, welch-method, hardware-spectrum, matched-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Window_function
  - https://web.mit.edu/xiphmont/Public/windows.pdf
---

A **window function** is a smooth taper multiplied onto a block of samples before an
[FFT](/reference/fast-fourier-transform/), forcing the block's edges gently to zero so the
transform does not treat the abrupt cut-off as a real signal feature.[^wiki] Without it, the
[DFT](/reference/discrete-fourier-transform/) assumes the finite block repeats forever; the
step discontinuity where the record wraps radiates energy across the whole spectrum —
**spectral leakage** — burying weak signals near strong ones. A window suppresses that
leakage at the cost of slightly blurring the frequencies it keeps.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Left: a rectangular window gives a narrow main lobe but tall side-lobes. Right: a tapered window gives a wider main lobe but much lower side-lobes." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="115" y="16">rectangular: narrow lobe, tall side-lobes</text>
    <line x1="20" y1="120" x2="210" y2="120" stroke="currentColor" stroke-width="1"/>
    <path d="M20 118 Q90 118 105 55 Q115 118 130 108 Q140 118 150 112 Q160 118 170 114 Q185 118 210 118" fill="none" stroke="currentColor" stroke-width="1.4"/>
    <text x="345" y="16">Hann/Blackman: wide lobe, low side-lobes</text>
    <line x1="250" y1="120" x2="440" y2="120" stroke="currentColor" stroke-width="1"/>
    <path d="M250 119 Q320 119 335 58 Q350 119 360 117 Q375 119 390 118 Q420 119 440 119" fill="none" stroke="currentColor" stroke-width="1.4"/>
    <text x="105" y="50">main</text><text x="335" y="52">main</text>
  </g>
</svg>
<figcaption>Windowing trades a wider main lobe (coarser resolution) for far lower side-lobes, so a nearby weak signal is no longer masked by a strong one's leakage.</figcaption>
</figure>

## How it works

The transform of a windowed block is the transform of the signal *convolved* with the
transform of the window itself. Every window's spectrum has two features that define the
trade-off:

- **Main-lobe width** sets frequency resolution — how far apart two tones must be to appear
  as two peaks. A wider main lobe means coarser resolution.
- **Side-lobe level** sets dynamic range — how far a leaked skirt sits below the true peak.
  Lower side-lobes let a faint signal survive next to a loud neighbour.

These pull against each other: you cannot narrow the main lobe and lower the side-lobes at
once for a fixed block length. Choosing a window is choosing where on that curve to sit.
The rectangular ("no window") case has the narrowest possible main lobe but side-lobes only
about 13 dB down — usually unacceptable.

## Variants

- **Hann** (raised cosine): side-lobes ~31 dB down with fast roll-off; a sensible general
  default for waterfalls.
- **Hamming**: a tuned raised cosine with the nearest side-lobe pushed to ~43 dB, but a
  slower far-out roll-off than Hann.
- **Blackman–Harris** (3- and 4-term): side-lobes down 71–92 dB for high-dynamic-range work,
  at the price of a noticeably wider main lobe.
- **Nuttall**: a Blackman-family window optimised for very low, fast-decaying side-lobes.
- **Kaiser**: a single parameter β continuously trades main-lobe width against side-lobe
  level, so it can be dialled to a spec rather than picked from a fixed list.
- **Flat-top**: a wide main lobe deliberately flattened so a tone's *amplitude* is measured
  accurately even when it falls between bins — used for calibration, not resolution.

The same tapers also shape [FIR filters](/reference/fir-filter/) via the window design
method, where the identical resolution-versus-leakage trade governs the filter's transition
width and stop-band rejection.

## Relevance to SDR

Windowing is applied to almost every FFT block a software radio takes of its
[I/Q data](/reference/iq-data/) before it becomes a line in a
[spectrum display](/reference/hardware-spectrum/) or waterfall. The right choice depends on
the task: Hann or Hamming for general band-scanning where you want to *see* many signals;
Blackman–Harris or Nuttall when a strong local carrier would otherwise smear over the weak
signal you actually care about; flat-top when you need an accurate power reading.
[Welch's method](/reference/welch-method/) for power-spectral-density estimation windows
every overlapping segment before averaging, and the choice of window sets both its bias and
the effective bin bandwidth.

GopherTrunk applies windowing in its FFT-based spectral visualisation and signal-search
tooling, where it improves the operator's ability to pick control channels and weak carriers
out of a busy band. The choice is a display-quality decision rather than part of the digital
symbol-decode path, which relies on time-domain matched filtering instead.

## Sources

[^wiki]: [Window function](https://en.wikipedia.org/wiki/Window_function) — Wikipedia, on spectral leakage, the main-lobe/side-lobe trade-off, and the common window families.
