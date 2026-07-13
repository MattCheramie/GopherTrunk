---
slug: frequency
title: Frequency
entry_type: term
category: rf-fundamentals
description: Frequency is the number of cycles a wave completes per second, measured in hertz (Hz); for radio it sets the tuning point and, inversely, the wavelength.
keywords: frequency, hertz, Hz, cycles per second, kHz MHz GHz, period, tuning
infobox:
  - { label: Symbol, value: f }
  - { label: Unit, value: Hertz (Hz) }
  - { label: Relation, value: "wavelength = c / frequency" }
see_also: [wavelength, frequency-bands, electromagnetic-spectrum, radio-wave, local-oscillator, phase-noise]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency
  - https://www.bipm.org/en/si-base-units/second
---

**Frequency** is the number of cycles a periodic wave completes each second, measured in
**hertz (Hz)** — one hertz being one cycle per second.[^wiki] For a
[radio wave](/reference/radio-wave/) it is the quantity you tune to, it is inversely
related to [wavelength](/reference/wavelength/), and it is the coordinate along which the
whole [electromagnetic spectrum](/reference/electromagnetic-spectrum/) is laid out. When
you "tune to 162.550 MHz," you are selecting a frequency.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two sine waves: a low-frequency wave with few cycles on top and a high-frequency wave with many cycles below, showing that higher frequency packs more cycles into the same time." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="24" font-size="10" fill="currentColor">low frequency</text>
  <path d="M20 50 q30 -28 60 0 t60 0 t60 0 t60 0 t60 0 t60 0 t40 0" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="20" y="100" font-size="10" fill="currentColor">high frequency</text>
  <path d="M20 122 q12 -26 24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0 t24 0" fill="none" stroke="currentColor" stroke-width="2"/>
</svg>
<figcaption>Frequency is how many cycles pass each second; higher frequency packs more cycles into the same time (and means a shorter wavelength).</figcaption>
</figure>

## How it works

Frequency is the inverse of period: if one cycle takes a time *T* seconds, the frequency
is *f = 1 / T*. A 100 MHz FM signal repeats a hundred million times a second, so each
cycle lasts just 10 nanoseconds. Radio frequencies are large numbers, so they are scaled
into kilohertz (kHz, 10³ Hz), megahertz (MHz, 10⁶ Hz), and gigahertz (GHz, 10⁹ Hz). The
unit honours Heinrich Hertz, who first generated and detected radio waves in the 1880s.

Because all radio waves travel at the speed of light *c*, frequency and
[wavelength](/reference/wavelength/) are locked together by *λ = c / f*: doubling the
frequency halves the wavelength. Frequency is also the axis of the frequency domain — the
view a [spectrum analyzer](/reference/spectrum-analyzer/) or FFT shows — where a pure tone
appears as a single spike and a modulated signal spreads into a band of nonzero
[bandwidth](/reference/bandwidth/) around its centre frequency.

## In practice

- **Bands and allocation.** The radio spectrum is divided into
  [frequency bands](/reference/frequency-bands/) (VHF, UHF, and so on), and regulators
  assign slices of each to specific services. A frequency is not just a number but a
  legal and physical context.
- **Stability matters.** A transmitter and receiver must agree on frequency to within a
  small tolerance. Real oscillators drift with temperature and age, and their short-term
  jitter shows up as [phase noise](/reference/phase-noise/) that smears the carrier and
  degrades demodulation.
- **Doppler.** Motion between transmitter and receiver shifts the apparent frequency
  ([Doppler shift](/reference/doppler-shift/)), significant for satellites and fast
  vehicles.

## Relevance to SDR

Tuning an SDR sets the centre frequency its
[local oscillator](/reference/local-oscillator/) mixes down toward
[baseband](/reference/baseband/); the chosen frequency, within a
[band](/reference/frequency-bands/), determines what signal lands in the captured
passband. Because cheap tuners have imperfect reference oscillators, their actual
frequency is offset by a few parts per million, and GopherTrunk (like any decoder) must
apply a [PPM correction](/reference/ppm-frequency-correction/) and continuously track
residual frequency error with an automatic-frequency-control or frequency-locked loop so
symbols stay aligned. Getting the frequency right — and keeping it right — is the first
requirement for any decode.

## Sources

[^wiki]: [Frequency](https://en.wikipedia.org/wiki/Frequency) — Wikipedia, definition, the hertz unit, and the relationship to period and wavelength.
[^bipm]: [The second (SI base unit)](https://www.bipm.org/en/si-base-units/second) — BIPM, the SI definition of the second, on which the hertz (cycles per second) is based.
