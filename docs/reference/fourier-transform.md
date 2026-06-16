---
slug: fourier-transform
title: Fourier transform
entry_type: algorithm
category: algorithms
description: The Fourier transform decomposes a signal into its constituent frequencies, converting between the time and frequency domains; it is the mathematical basis of spectrum analysis.
keywords: Fourier transform, frequency domain, time domain, spectrum, Joseph Fourier
aka: [Fourier transform]
autolink: true
infobox:
  - { label: Type, value: Mathematical transform }
  - { label: Converts, value: Time domain ↔ frequency domain }
  - { label: Named for, value: Joseph Fourier }
see_also: [fast-fourier-transform, joseph-fourier, bandwidth, software-defined-radio]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/fft-and-waterfall/ }
external:
  - { title: "Fourier transform (Wikipedia)", url: https://en.wikipedia.org/wiki/Fourier_transform }
---

The **Fourier transform** decomposes a signal into the frequencies that compose it,
converting between the time and frequency domains. It is the mathematical foundation of
spectrum analysis, named for [Joseph Fourier](/reference/joseph-fourier/).

## How it works

Any signal can be expressed as a sum of sinusoids; the transform reports how much energy
exists at each frequency. Its discrete, efficient form is the
[FFT](/reference/fast-fourier-transform/).

## Relevance to SDR

Turning [IQ](/reference/iq-data/) samples into a spectrum — and thus a waterfall — is a
Fourier transform, the reason you can *see* signals on an SDR.
