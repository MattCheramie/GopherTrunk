---
slug: fourier-transform
title: Fourier transform
entry_type: algorithm
category: algorithms
description: The Fourier transform decomposes a signal into its constituent frequencies, converting between the time and frequency domains; it is the mathematical basis of spectrum analysis.
keywords: Fourier transform, frequency domain, time domain, spectrum, spectral analysis, sinusoids, Joseph Fourier, DFT, FFT, IQ
aka: [Fourier transform, FT]
autolink: true
infobox:
  - { label: Type, value: Mathematical transform }
  - { label: Converts, value: Time domain ↔ frequency domain }
  - { label: Named for, value: Joseph Fourier }
see_also: [discrete-fourier-transform, fast-fourier-transform, hilbert-transform, joseph-fourier, bandwidth, software-defined-radio]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/rf-sdr/fft-and-waterfall/ }
related_reading:
  - { title: "SDR Internals, Part 8: Equalization, diversity & the FFT", url: /blog/deep-dives/sdr-internals-08-equalization-diversity-fft/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Fourier_transform
  - https://en.wikipedia.org/wiki/Frequency_domain
---

The **Fourier transform** decomposes a signal into the frequencies that compose it,
converting a description in terms of *time* into an equivalent description in terms of
*frequency*.[^wiki] It is the mathematical foundation of spectrum analysis — the reason a
waveform that looks like noise in the time domain can reveal sharp carriers, tones, and
sidebands once viewed as a spectrum. It is named for [Joseph Fourier](/reference/joseph-fourier/),
who showed that arbitrary functions can be written as sums of sinusoids.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A complex time-domain waveform on the left transforms into a set of frequency peaks on the right." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q10 -22 20 0 q10 18 20 0 q10 -28 20 0 q10 24 20 0 q10 -16 20 0 q10 20 20 0 q10 -22 20 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="100" y="105" text-anchor="middle" font-size="9" fill="currentColor">time domain</text>
  <line x1="225" y1="60" x2="265" y2="60" stroke="currentColor" marker-end="url(#ftar)"/><text x="245" y="50" text-anchor="middle" font-size="8" fill="currentColor">FT</text>
  <line x1="290" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="2"><line x1="320" y1="90" x2="320" y2="55"/><line x1="360" y1="90" x2="360" y2="35"/><line x1="400" y1="90" x2="400" y2="68"/></g>
  <text x="365" y="105" text-anchor="middle" font-size="9" fill="currentColor">frequency domain</text>
  <defs><marker id="ftar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The Fourier transform expresses a signal as a sum of frequencies — converting a time waveform into a spectrum.</figcaption>
</figure>

## How it works

The transform works by correlating the signal against sinusoids of every candidate
frequency. Where the signal contains energy at a given frequency, its correlation with a
sinusoid at that frequency is large; where it does not, the correlation averages to zero.
The result is a **spectrum**: a function that reports, for each frequency, both the
*amplitude* and the *phase* of the sinusoid present there. Because it captures phase as
well as magnitude, the transform is invertible — the inverse Fourier transform reassembles
the original waveform exactly, so no information is lost in the round trip.

For a real-valued signal the spectrum is symmetric, but SDR works with complex
[IQ data](/reference/iq-data/), whose spectrum is one-sided and can distinguish frequencies
above and below the tuned centre. This is why quadrature sampling matters: it lets the
transform place a signal on the correct side of the local oscillator.

## Variants

The Fourier family is really several related transforms that differ in whether time and
frequency are continuous or discrete:

- **Continuous Fourier transform** — the pure mathematical form, defined by an integral
  over all time, mapping a continuous signal to a continuous spectrum.
- **Fourier series** — for periodic signals, giving a spectrum of discrete harmonics.
- **Discrete-time / discrete Fourier transform** — the
  [discrete Fourier transform](/reference/discrete-fourier-transform/) (DFT) operates on a
  finite block of samples and produces a finite set of frequency *bins*. This is the only
  form a computer can evaluate directly, and it is what every SDR spectrum display actually
  computes.
- **Fast Fourier transform** — the [FFT](/reference/fast-fourier-transform/) is not a
  different transform but an efficient *algorithm* for the DFT, cutting the cost from
  O(N²) to O(N log N) and making real-time waterfalls possible.

A closely related operation is the [Hilbert transform](/reference/hilbert-transform/),
which shifts every frequency component by 90° to build the analytic signal used in
single-sideband and envelope detection.

## Relevance to SDR

Turning IQ samples into a spectrum — and thus a waterfall — is a Fourier transform, the
reason you can *see* signals on an SDR. GopherTrunk uses FFTs to survey a band, to locate a
steady [control channel](/reference/control-channel/), and inside multi-channel
[channelizers](/reference/channelizer/) that split a wide capture into per-channel streams.
Beyond scanning, the transform underpins OFDM demodulation, matched filtering, and fast
convolution throughout modern radio, so it is fair to call it the single most-used idea in
digital signal processing.

## Sources

[^wiki]: [Fourier transform](https://en.wikipedia.org/wiki/Fourier_transform) — Wikipedia, for the mathematical definition and time/frequency-domain duality. See also [Frequency domain](https://en.wikipedia.org/wiki/Frequency_domain) for the spectrum concept.
