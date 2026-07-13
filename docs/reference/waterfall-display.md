---
slug: waterfall-display
title: Waterfall display
entry_type: term
category: sdr-dsp
description: "A waterfall display is a live, scrolling spectrogram in SDR software that shows signal power across frequency over time, with color mapping power intensity."
keywords: waterfall display, waterfall plot, scrolling spectrogram, SDR waterfall, live spectrum, FFT display, panadapter, color intensity spectrum, real-time spectrogram
aka: [waterfall, waterfall plot, scrolling spectrum]
autolink: true
infobox:
  - { label: Type, value: Real-time SDR display }
  - { label: Shows, value: Power vs frequency over time }
  - { label: Made of, value: Stacked, scrolling FFTs }
see_also: [spectrogram, power-spectral-density, fast-fourier-transform, gqrx, sdr-sharp, spectrum-analyzer]
cite_urls:
  - https://en.wikipedia.org/wiki/Spectrogram
  - https://en.wikipedia.org/wiki/Waterfall_plot
---

**A waterfall display** is the live, scrolling [spectrogram](/reference/spectrogram/) at the
heart of nearly every SDR application: each new row is a fresh power spectrum across the tuned
frequency span, colored by intensity, and the rows scroll so time flows down (or up) the
screen.[^wiki] It is the panel operators actually watch — a signal that flicks on shows up
instantly as a bright vertical streak, and its color, width, and shape hint at what it is before a
single bit is decoded. Practically, the waterfall turns an invisible slice of spectrum into
something you can read at a glance.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A waterfall display with a spectrum trace along the top and a scrolling color panel below, where bright vertical lines are active carriers and time flows downward." xmlns="http://www.w3.org/2000/svg">
  <path d="M45 48 L110 46 L150 45 L175 22 L200 45 L245 47 L270 30 L295 47 L360 46 L440 47" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="70" y="18" font-size="8" fill="currentColor">spectrum (newest row)</text>
  <rect x="45" y="60" width="395" height="100" fill="none" stroke="currentColor" stroke-width="1"/>
  <rect x="169" y="60" width="12" height="100" fill="currentColor" opacity="0.85"/>
  <rect x="262" y="60" width="8" height="70" fill="currentColor" opacity="0.6"/>
  <rect x="300" y="105" width="30" height="30" fill="currentColor" opacity="0.5"/>
  <text x="175" y="173" font-size="7.5" fill="currentColor" text-anchor="middle">steady carrier</text>
  <text x="315" y="173" font-size="7.5" fill="currentColor" text-anchor="middle">burst</text>
  <line x1="452" y1="65" x2="452" y2="155" stroke="currentColor" stroke-width="1" marker-end="url(#wfar)"/>
  <text x="452" y="60" font-size="7.5" fill="currentColor" text-anchor="middle">time</text>
  <defs><marker id="wfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A live spectrum forms the newest row; older rows scroll downward, so a persistent carrier draws a vertical line and a brief transmission draws a short bright dash.</figcaption>
</figure>

## How it works

A waterfall is a spectrogram rendered incrementally in real time. The receiver takes blocks of
[I/Q](/reference/iq-data/) samples, applies a [window function](/reference/window-function/), runs
an [FFT](/reference/fast-fourier-transform/), converts each bin to power in dB (a
[PSD](/reference/power-spectral-density/) estimate), and draws that vector as one horizontal line
of colored pixels. The line is pushed onto the display and the rest scroll to make room, so the
image is a moving history a few seconds to a few minutes deep. Several controls shape what you
see:

- **Color map and range** — power is mapped through a palette between a floor and a ceiling in dB.
  Setting the floor just above the [noise floor](/reference/noise-floor/) and the ceiling near the
  strongest signal gives the most contrast; a poorly set range either washes out weak signals or
  saturates strong ones.
- **FFT size** — more bins give finer frequency resolution (you can split two close carriers) but
  cost more computation and blur fast events, the same time–frequency trade-off as any
  spectrogram.
- **Averaging and scroll rate** — averaging several FFTs per row lowers the grassy variance so
  weak steady carriers emerge, at the price of temporal smearing; the scroll rate sets how much
  history fits on screen.

Because a persistent-but-weak signal accumulates as a continuous line, the human eye picks it out
of noise far better than it could from a single spectrum — the waterfall's time integration is a
free processing gain for detection.

## In practice

The waterfall is paired with a spectrum ("panadapter") trace and is the primary way SDR users
find, identify, and tune signals: click a streak to set the receive frequency, judge modulation by
the streak's width and texture, and watch a trunked system's control channel sit as a steady line
while voice channels blink on and off across the band. It is also the fastest way to spot
interference, birdies, and images.

## Relevance to SDR

Waterfall displays are ubiquitous in SDR software — [GQRX](/reference/gqrx/),
[SDR#](/reference/sdr-sharp/), SDRangel, CubicSDR, and web-based receivers all center their UI on
one. For monitoring trunked systems it is invaluable for locating control-channel carriers and
seeing simulcast or interference conditions. GopherTrunk itself is a headless decoder rather than a
graphical SDR, so it does not draw a waterfall; users typically identify candidate frequencies in a
waterfall-based tool first, then hand the frequency to GopherTrunk to decode, and inspect captured
[I/Q](/reference/iq-data/) in a waterfall when troubleshooting a stubborn signal.

## Sources

[^wiki]: [Spectrogram](https://en.wikipedia.org/wiki/Spectrogram) — Wikipedia, on the time-frequency display that the scrolling SDR waterfall renders in real time.
