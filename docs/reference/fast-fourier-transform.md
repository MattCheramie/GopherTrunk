---
slug: fast-fourier-transform
title: Fast Fourier transform (FFT)
entry_type: algorithm
category: algorithms
description: The fast Fourier transform is an efficient O(N log N) algorithm for computing the discrete Fourier transform, making real-time spectrum, waterfall, and OFDM processing practical.
keywords: FFT, fast Fourier transform, DFT, Cooley-Tukey, radix-2, butterfly, spectrum, waterfall, bins, windowing, OFDM
aka: [fast Fourier transform, FFT]
autolink: true
infobox:
  - { label: Type, value: Algorithm }
  - { label: Computes, value: Discrete Fourier transform efficiently }
  - { label: Complexity, value: O(N log N) vs O(N²) }
see_also: [fourier-transform, discrete-fourier-transform, window-function, welch-method, overlap-add-overlap-save, ofdm]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/rf-sdr/fft-and-waterfall/ }
related_reading:
  - { title: "SDR Internals, Part 8: Equalization, diversity & the FFT", url: /blog/deep-dives/sdr-internals-08-equalization-diversity-fft/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Fast_Fourier_transform
  - https://en.wikipedia.org/wiki/Cooley%E2%80%93Tukey_FFT_algorithm
---

The **fast Fourier transform** (**FFT**) is an efficient algorithm for computing the
discrete [Fourier transform](/reference/discrete-fourier-transform/) (DFT), reducing the
arithmetic from O(N²) to O(N log N) so the same result can be produced fast enough to run
many times a second in real time.[^wiki] It computes exactly the same spectrum as a direct
DFT — it is a shortcut, not an approximation — and that speedup is what makes live spectrum
displays, waterfalls, and OFDM radios practical at all.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A block of time samples feeding an FFT block that outputs a row of frequency bins." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="30" cy="40" r="2.5"/><circle cx="30" cy="55" r="2.5"/><circle cx="30" cy="70" r="2.5"/><circle cx="30" cy="85" r="2.5"/></g>
  <text x="30" y="105" text-anchor="middle" font-size="8" fill="currentColor">samples</text>
  <rect x="70" y="35" width="80" height="60" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="110" y="69" text-anchor="middle" font-size="11" fill="currentColor">FFT</text>
  <line x1="150" y1="65" x2="190" y2="65" stroke="currentColor" marker-end="url(#fftar)"/>
  <line x1="200" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="3"><line x1="220" y1="100" x2="220" y2="80"/><line x1="250" y1="100" x2="250" y2="55"/><line x1="280" y1="100" x2="280" y2="40"/><line x1="310" y1="100" x2="310" y2="70"/><line x1="340" y1="100" x2="340" y2="85"/><line x1="370" y1="100" x2="370" y2="60"/><line x1="400" y1="100" x2="400" y2="90"/></g>
  <text x="320" y="118" text-anchor="middle" font-size="8" fill="currentColor">frequency bins</text>
  <defs><marker id="fftar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The FFT computes the spectrum efficiently, splitting the band into equal frequency bins — the basis of the waterfall.</figcaption>
</figure>

## How it works

The FFT exploits the redundancy in the DFT sum. The most common variant, the
**Cooley–Tukey radix-2** algorithm, recursively splits an N-point transform into two
N/2-point transforms — one over the even-indexed samples, one over the odd — and then
recombines them with a **butterfly** that multiplies one half by a *twiddle factor* (a
complex root of unity) and adds and subtracts. Applying this split log₂N times leaves only
2-point butterflies, so the total cost is N/2 · log₂N butterflies rather than N² complex
multiplies. Radix-2 wants a power-of-two length; mixed-radix and Bluestein variants handle
other sizes.

The band is divided into a number of **bins** equal to the FFT size; the frequency
resolution is roughly [sample rate](/reference/sample-rate/) ÷ FFT size. More bins give
finer resolution but cover a longer time window, so the update rate drops and CPU cost
rises — the familiar time-versus-frequency resolution trade-off.

## In practice

Feeding raw samples straight into an FFT causes **spectral leakage**: because the block is
a finite chunk cut from a longer signal, its edges act like a discontinuity and smear
energy across neighbouring bins. The fix is to multiply the block by a tapered
[window function](/reference/window-function/) (Hann, Hamming, Blackman-Harris…) before the
transform, trading a slightly wider main lobe for far lower sidelobes. To turn short, noisy
transforms into a stable power-spectral-density estimate, [Welch's method](/reference/welch-method/)
averages the windowed FFTs of many overlapping segments.

The FFT also accelerates *filtering*: convolving a signal with a long filter is far cheaper
as a multiply in the frequency domain, and streaming that idea across successive blocks is
exactly the [overlap-add and overlap-save](/reference/overlap-add-overlap-save/) method.

## Relevance to SDR

The FFT drives the spectrum and waterfall displays used to find signals and spot a steady
[control channel](/reference/control-channel/), and GopherTrunk relies on it for band
surveys and inside polyphase channelizers. It is also the heart of
[OFDM](/reference/ofdm/): systems such as Wi-Fi, LTE, 5G NR, DVB-T, and DAB transmit data on
thousands of orthogonal subcarriers, and the receiver recovers them all with a single FFT
per symbol — the algorithm that makes wideband digital broadcasting economical.

## Sources

[^wiki]: [Fast Fourier transform](https://en.wikipedia.org/wiki/Fast_Fourier_transform) — Wikipedia, for the algorithm and its efficiency over the direct DFT. See also [Cooley–Tukey FFT algorithm](https://en.wikipedia.org/wiki/Cooley%E2%80%93Tukey_FFT_algorithm) for the radix-2 decomposition.
