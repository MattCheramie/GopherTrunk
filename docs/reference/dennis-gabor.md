---
slug: dennis-gabor
title: Dennis Gabor
entry_type: person
category: people
description: "Dennis Gabor (1900–1979) was a Hungarian-British physicist who invented holography and the Gabor transform, and formalised the analytic signal in signal analysis."
keywords: Dennis Gabor, Gabor transform, holography, analytic signal, Gabor atom, time-frequency, Nobel Prize, short-time Fourier transform
aka: [Dennis Gabor, Dénes Gábor]
autolink: true
infobox:
  - { label: Lived, value: "1900–1979" }
  - { label: Field, value: "Physics, engineering" }
  - { label: Known for, value: "Holography; the Gabor transform" }
see_also: [hilbert-transform, spectrogram, fourier-transform, window-function, discrete-fourier-transform]
cite_urls:
  - https://en.wikipedia.org/wiki/Dennis_Gabor
  - https://en.wikipedia.org/wiki/Gabor_transform
---

**Dennis Gabor** (1900–1979) was a Hungarian-British physicist and electrical engineer
who invented **holography**, for which he won the 1971 Nobel Prize in Physics, and who
introduced the **Gabor transform** and the notion of the **analytic signal** that underpin
modern time-frequency analysis.[^wiki][^gt] His 1946 paper *"Theory of Communication"*
reshaped how engineers think about a signal's joint description in time and frequency.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A signal analysed by sliding a Gaussian window along it and taking a local Fourier transform, producing a time-frequency map." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M30 50 q15 -25 30 0 q15 25 30 0 q15 -25 30 0 q15 25 30 0 q15 -25 30 0 q15 25 30 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <path d="M90 50 q10 -22 20 0 q10 22 20 0" fill="none" stroke="currentColor" stroke-width="2.4"/>
  <text x="110" y="30" text-anchor="middle" font-size="9" fill="currentColor">Gaussian window</text>
  <line x1="240" y1="45" x2="300" y2="45" stroke="currentColor" marker-end="url(#dgar)"/>
  <g fill="none" stroke="currentColor">
    <rect x="320" y="25" width="110" height="80"/>
  </g>
  <g fill="currentColor" fill-opacity="0.4"><rect x="330" y="70" width="20" height="16"/><rect x="360" y="50" width="20" height="16"/><rect x="390" y="60" width="20" height="16"/></g>
  <text x="375" y="120" text-anchor="middle" font-size="9" fill="currentColor">time → frequency map</text>
</svg>
<figcaption>The Gabor transform slides a Gaussian window along a signal and takes a local Fourier transform, yielding a joint time-frequency picture — the basis of the spectrogram.</figcaption>
</figure>

## Life and work

Gabor was born in Budapest, studied engineering in Berlin, and fled Nazi Germany for
Britain in 1933, settling at the firm British Thomson-Houston and later at Imperial
College London.[^wiki] He conceived holography in 1947–48 while trying to improve the
resolution of electron microscopes: he reasoned that recording the full wavefront —
both amplitude and phase — as an interference pattern, then reconstructing it with light,
could sidestep the microscope's lens aberrations. The idea was decades ahead of the
technology needed to realise it; only after the laser arrived in the 1960s did holography
flourish, earning Gabor the Nobel Prize in 1971.

## Contribution

Two of Gabor's ideas are central to signal processing.

The first is the **analytic signal**. Gabor showed that a real signal can be paired with
a companion built from its [Hilbert transform](/reference/hilbert-transform/) to form a
complex signal with no negative-frequency content. This complex representation cleanly
separates a signal's instantaneous **amplitude** (its envelope) from its instantaneous
**phase and frequency**, which is exactly what a receiver needs to demodulate — and it is
the conceptual basis for the I/Q representation used throughout software radio.

The second is the **Gabor transform**, a short-time Fourier transform that uses a
**Gaussian window**. Gabor argued that neither a pure time description nor a pure
frequency description captures a real signal well; what matters is *when* each frequency
occurs. By multiplying the signal with a Gaussian window and Fourier-transforming it,
localised in both domains, he produced a joint time-frequency map. He also showed that the
Gaussian window achieves the best possible simultaneous localisation — the tightest
time-frequency "cell," an uncertainty-principle limit — giving rise to the elementary
"Gabor atoms" from which such analyses are built.[^gt]

## Legacy

The Gabor transform is the direct ancestor of the [spectrogram](/reference/spectrogram/)
and the modern short-time [Fourier transform](/reference/fourier-transform/), and it
seeded the later field of wavelet analysis. His analytic-signal formulation made the
amplitude/phase decomposition of signals rigorous, and holography opened an entire branch
of optics. Few researchers have left foundational marks on optics and signal processing
alike.

## Relevance to SDR

Gabor's ideas are visible on any SDR screen. The waterfall and spectrogram displays that
operators watch are Gabor/short-time Fourier transforms, trading time resolution against
frequency resolution through the choice of [window function](/reference/window-function/).
The analytic-signal concept is the theoretical backbone of I/Q sampling, on which every
software radio — GopherTrunk included — is built: GopherTrunk consumes complex baseband
I/Q, and its FFT-based channelisation and spectral displays are Gabor analysis in
practice.

## Sources

[^wiki]: [Dennis Gabor](https://en.wikipedia.org/wiki/Dennis_Gabor) — Wikipedia, for biography, holography, and the Nobel Prize.
[^gt]: [Gabor transform](https://en.wikipedia.org/wiki/Gabor_transform) — Wikipedia, for the Gaussian-windowed short-time Fourier transform and time-frequency localisation.
