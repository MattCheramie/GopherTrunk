---
slug: fast-fourier-transform
title: Fast Fourier transform (FFT)
entry_type: algorithm
category: algorithms
description: The fast Fourier transform is an efficient algorithm for computing the discrete Fourier transform, making real-time spectrum and waterfall displays practical.
keywords: FFT, fast Fourier transform, DFT, spectrum, waterfall, bins, Cooley-Tukey
aka: [fast Fourier transform, FFT]
autolink: true
infobox:
  - { label: Type, value: Algorithm }
  - { label: Computes, value: Discrete Fourier transform efficiently }
  - { label: Output, value: Spectrum (bins) }
see_also: [fourier-transform, bandwidth, sample-rate, iq-data]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/fft-and-waterfall/ }
external:
  - { title: "Fast Fourier transform (Wikipedia)", url: https://en.wikipedia.org/wiki/Fast_Fourier_transform }
---

The **fast Fourier transform** (**FFT**) is an efficient algorithm for computing the
discrete [Fourier transform](/reference/fourier-transform/), reducing the work enough to
run many times a second in real time.

## How it works

It splits the captured [bandwidth](/reference/bandwidth/) into a number of **bins** (the
FFT size); resolution ≈ [sample rate](/reference/sample-rate/) ÷ FFT size. More bins give
finer resolution but slower updates and more CPU.

## Relevance to SDR

The FFT drives the spectrum and waterfall displays used to find signals and spot a steady
[control channel](/reference/control-channel/).
