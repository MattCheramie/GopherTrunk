---
slug: spectrum-analyzer
title: Spectrum analyzer
entry_type: hardware
category: test-equipment
description: "A spectrum analyzer is RF test gear that measures signal power versus frequency, using either a swept superheterodyne or an FFT/real-time architecture."
keywords: spectrum analyzer, swept spectrum analyzer, FFT analyzer, real-time spectrum analyzer, RTSA, RBW, resolution bandwidth, signal power vs frequency, RF test equipment
aka: [spectrum analyser, SA, RTSA]
autolink: true
infobox:
  - { label: Type, value: RF measurement instrument }
  - { label: Measures, value: "Power vs. frequency" }
  - { label: Architectures, value: "Swept / FFT / real-time" }
  - { label: Key spec, value: "Resolution bandwidth (RBW)" }
  - { label: TX, value: "No (receive-only)" }
  - { label: Typical price, value: "$100 – $50,000+" }
see_also: [tinysa, fast-fourier-transform, power-spectral-density, signal-generator, phase-noise, spectrogram]
cite_urls:
  - https://en.wikipedia.org/wiki/Spectrum_analyzer
  - https://www.keysight.com/us/en/assets/7018-06714/application-notes/5952-0292.pdf
---

**A spectrum analyzer** is an instrument that displays signal power as a function of
frequency[^wiki] — the frequency-domain counterpart to an oscilloscope's time-domain
trace. It answers "how much energy is present at each frequency?", making it the primary
tool for finding carriers, measuring occupied [bandwidth](/reference/bandwidth/), hunting
spurious emissions, and characterizing the [phase noise](/reference/phase-noise/) of an
oscillator.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A spectrum analyzer trace showing power on the vertical axis versus frequency on the horizontal axis, with a tall carrier peak, two smaller spur peaks, and a flat noise floor." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="20" x2="40" y2="140" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="40" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="12" y="82" font-size="9" fill="currentColor" transform="rotate(-90 12 82)" text-anchor="middle">Power (dBm)</text>
  <text x="240" y="160" font-size="9" fill="currentColor" text-anchor="middle">Frequency</text>
  <polyline points="40,128 120,126 150,124 175,40 200,124 260,127 300,90 320,126 360,125 390,100 410,126 440,128" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="40" y1="128" x2="440" y2="128" stroke="currentColor" stroke-opacity="0.25" stroke-dasharray="3 3"/>
  <text x="175" y="34" font-size="8" fill="currentColor" text-anchor="middle">carrier</text>
  <text x="300" y="82" font-size="8" fill="currentColor" text-anchor="middle">spur</text>
  <text x="120" y="122" font-size="8" fill="currentColor">noise floor</text>
</svg>
<figcaption>A spectrum analyzer plots power versus frequency: a strong carrier rises above a noise floor, with smaller spurs and harmonics revealed alongside it.</figcaption>
</figure>

## Overview

Two things define what a spectrum analyzer can show you: its **noise floor** (how weak a
signal it can resolve) and its **resolution bandwidth** (RBW, how finely it separates two
close-spaced tones). A narrower RBW lowers the displayed noise floor and sharpens
adjacent signals, but each halving of RBW roughly quadruples sweep time on a swept
instrument. Span, reference level, and detector mode round out the core controls.

## Swept vs. FFT / real-time

Classic bench analyzers are **swept-tuned superheterodyne**
[receivers](/reference/superheterodyne-receiver/): a
[local oscillator](/reference/local-oscillator/) ramps across the span, mixing each
frequency in turn down to a fixed [IF](/reference/intermediate-frequency/) where a narrow
filter and a detector measure power. Because the LO visits one frequency at a time, a
swept analyzer is **blind between visits** — a short burst that occurs while the sweep is
elsewhere is simply missed. Swept designs excel at wide spans with excellent dynamic
range and low noise, which is why high-end phase-noise and spurious measurements still
favor them.

**FFT analyzers** instead digitize a block of the input and compute a
[fast Fourier transform](/reference/fast-fourier-transform/), producing all frequency
bins in that block at once. This is exactly how a
[software-defined radio](/reference/software-defined-radio/) or the
[tinySA](/reference/tinysa/) in its lower bands measures spectrum. FFT processing is fast
over a limited span and yields the
[power spectral density](/reference/power-spectral-density/) directly, but the
instantaneous span is capped by the ADC [sample rate](/reference/sample-rate/).

A **real-time spectrum analyzer (RTSA)** overlaps successive FFTs so that no input gap
exists between transforms — it guarantees 100% probability of intercept for any event
longer than a stated minimum duration. RTSAs add persistence and density displays (a
[spectrogram](/reference/spectrogram/) or bitmap-style overlay) that reveal transient and
intermittent signals a swept trace would average away. Modern instruments frequently
combine a swept front end for wide spans with an FFT/real-time block for detailed,
gap-free analysis over a narrower window.

## In practice

Key settings to reason about on any analyzer:

- **RBW and VBW** — resolution bandwidth sets frequency selectivity and noise floor;
  video bandwidth smooths the trace.
- **Detector and trace mode** — peak, sample, RMS/average, and max-hold change what a bin
  reports; RMS is correct for noise-like signals, max-hold catches transients.
- **Reference level and attenuation** — set the top of the screen and protect the front
  end; too little attenuation invites internally generated
  [intermodulation](/reference/intermodulation/) that masquerades as real spurs.
- **Dynamic range** — the window between the noise floor and the onset of front-end
  compression or spurious products bounds what you can measure at once.

## Relevance to SDR

A spectrum analyzer is the natural companion to SDR scanning: use it to survey a band,
confirm which control-channel and voice frequencies are actually active, gauge signal
strength, and diagnose interference, images, and de-sense before committing an SDR to
decode. Any SDR receiver *is* a rudimentary FFT spectrum analyzer — GopherTrunk's own
waterfall and FFT views plot power versus frequency the same way — though a dedicated
instrument offers calibrated amplitude, far better dynamic range, and a much lower noise
floor. GopherTrunk does not drive external analyzers; it uses its internal FFT for
tuning and diagnostics, and a bench or handheld analyzer remains a useful external aid
for setting up antennas and filters.

## Sources

[^wiki]: [Spectrum analyzer](https://en.wikipedia.org/wiki/Spectrum_analyzer) — Wikipedia, on swept-tuned, FFT, and real-time spectrum analyzer architectures and their measurements.
