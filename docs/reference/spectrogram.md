---
slug: spectrogram
title: Spectrogram
entry_type: term
category: sdr-dsp
description: "A spectrogram is a time-frequency image of a signal, built by stacking short-time FFTs, with color or brightness showing power in each time-frequency cell."
keywords: spectrogram, short-time Fourier transform, STFT, time-frequency, sonogram, spectral image, waterfall, FFT stacking, time-frequency analysis
aka: [sonogram, STFT display, time-frequency plot]
autolink: true
infobox:
  - { label: Type, value: Time-frequency representation }
  - { label: Built from, value: Short-time FFTs (STFT) }
  - { label: Axes, value: Time × frequency, color = power }
see_also: [waterfall-display, power-spectral-density, fast-fourier-transform, window-function, spectrum-analyzer]
cite_urls:
  - https://en.wikipedia.org/wiki/Spectrogram
  - https://en.wikipedia.org/wiki/Short-time_Fourier_transform
---

**A spectrogram** is a two-dimensional image of a signal in which one axis is time, the other is
frequency, and the color or brightness of each cell shows how much power the signal had at that
frequency and moment.[^wiki] It is produced by chopping the signal into short overlapping windows,
taking an [FFT](/reference/fast-fourier-transform/) of each — the short-time Fourier transform
(STFT) — and laying the resulting spectra side by side. Where a single spectrum shows *what*
frequencies are present, a spectrogram shows *when* each one appears, which is exactly what you
need to see a signal turn on, sweep, or hop.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A spectrogram grid with time on the horizontal axis and frequency on the vertical, showing a steady horizontal carrier trace and a diagonal frequency-sweeping trace." xmlns="http://www.w3.org/2000/svg">
  <rect x="45" y="20" width="395" height="120" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="242" y="160" font-size="8.5" fill="currentColor" text-anchor="middle">time →</text>
  <text x="30" y="82" font-size="8.5" fill="currentColor" text-anchor="middle" transform="rotate(-90 30 82)">frequency ↑</text>
  <line x1="60" y1="55" x2="425" y2="55" stroke="currentColor" stroke-width="3" opacity="0.85"/>
  <text x="150" y="50" font-size="7.5" fill="currentColor">steady carrier</text>
  <line x1="70" y1="125" x2="410" y2="35" stroke="currentColor" stroke-width="2.4" opacity="0.7"/>
  <text x="300" y="70" font-size="7.5" fill="currentColor">sweep / chirp</text>
  <rect x="120" y="95" width="40" height="14" fill="currentColor" opacity="0.55"/>
  <rect x="250" y="100" width="55" height="12" fill="currentColor" opacity="0.4"/>
  <text x="150" y="132" font-size="7" fill="currentColor" text-anchor="middle">bursts</text>
</svg>
<figcaption>Time runs left to right, frequency bottom to top, and intensity is power: a horizontal line is a steady carrier, a diagonal is a sweeping tone, and blocks are bursts.</figcaption>
</figure>

## How it works

The spectrogram is the squared magnitude of the STFT. The procedure is:

1. **Segment** the signal into frames of N samples, usually overlapping (50–75% overlap is
   common) so events near a frame edge are not missed.
2. **Window** each frame with a taper — Hann, Hamming, Blackman — to curb spectral leakage; the
   [window function](/reference/window-function/) choice trades main-lobe width against sidelobe
   suppression.
3. **Transform** each windowed frame with an FFT and take the magnitude squared to get its power
   spectrum (a [PSD](/reference/power-spectral-density/) estimate for that instant).
4. **Stack** the spectra along the time axis and map power to color.

The defining constraint is the **time–frequency resolution trade-off**. A long window gives fine
frequency resolution (narrow bins) but blurs time, smearing brief events across a wide interval; a
short window pins down timing but coarsens frequency. You cannot have both at once — the product
of time and frequency uncertainty is bounded — so window length is chosen for the task: long to
separate two nearby carriers, short to time a fast hop. FFT size and overlap set the pixel grid,
while zero-padding interpolates the display without adding real resolution.

## In practice

Spectrograms make otherwise invisible structure obvious: the staircase of a frequency-hopping
system, the chirp of a LoRa symbol, the on/off pattern of TDMA bursts, the Doppler curve of a
satellite pass, or the harmonics of interference. Because the eye integrates over the image, a
signal a few dB below the instantaneous noise can still be seen as a faint but persistent line —
the visual analog of coherent averaging.

## Relevance to SDR

The spectrogram is the workhorse display of software radio. In SDR applications its live,
scrolling form is the [waterfall display](/reference/waterfall-display/), the panel users watch to
spot activity, identify a modulation by its shape, and click to tune. For trunking work a
spectrogram reveals control-channel carriers, simulcast timing, and interference at a glance.
GopherTrunk is a decoder rather than a GUI SDR, so it does not render a live spectrogram itself,
but the same short-time FFT analysis underlies the channel-power measurements it uses to detect
activity, and captured [I/Q](/reference/iq-data/) files are routinely inspected in a spectrogram
tool when diagnosing why a signal did or did not decode.

## Sources

[^wiki]: [Spectrogram](https://en.wikipedia.org/wiki/Spectrogram) — Wikipedia, on the time-frequency representation built from the short-time Fourier transform.
