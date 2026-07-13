---
slug: digital-filter
title: Digital filter
entry_type: term
category: sdr-dsp
description: A digital filter passes some frequencies and attenuates others by arithmetic on a sample stream; low-pass, band-pass, and channel filters isolate signals in an SDR.
keywords: digital filter, FIR, IIR, low-pass, band-pass, channel filter, taps, convolution, DSP
aka: [digital filter]
autolink: true
infobox:
  - { label: Type, value: DSP operation }
  - { label: Kinds, value: FIR, IIR; low-pass, band-pass }
  - { label: Use, value: Isolate a channel, shape pulses }
see_also: [decimation, fir-filter, iir-filter, cic-filter, root-raised-cosine-filter, matched-filter, digital-down-converter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_filter
  - https://en.wikipedia.org/wiki/Finite_impulse_response
---

A **digital filter** passes some frequencies and attenuates others by performing arithmetic
on a stream of samples — no physical components, just multiply-and-add.[^wiki] The two main
families are FIR (finite impulse response) and IIR (infinite impulse response), and both are
defined entirely by a small table of coefficients.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A frequency-response curve that passes a band of frequencies and attenuates those outside it." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="110" x2="440" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="110" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M40 95 L150 95 C 180 95, 180 35, 210 35 L 270 35 C 300 35, 300 95, 330 95 L 440 95" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="28" text-anchor="middle" font-size="10" fill="currentColor">passband</text>
  <text x="90" y="86" font-size="9" fill="currentColor">rejected</text><text x="390" y="86" font-size="9" fill="currentColor">rejected</text>
  <text x="240" y="130" text-anchor="middle" font-size="9" fill="currentColor">frequency →</text>
</svg>
<figcaption>A digital filter passes a chosen band of frequencies and attenuates the rest — isolating one channel.</figcaption>
</figure>

## How it works

A filter computes each output sample as a weighted sum of input samples — a **convolution**
of the signal with the filter's impulse response. An [FIR filter](/reference/fir-filter/)
weights a finite window of *past inputs* (its "taps"); more taps give a sharper transition
between passband and stopband but cost more arithmetic per sample. An
[IIR filter](/reference/iir-filter/) also feeds back *past outputs*, so a short recursion can
mimic a very long impulse response — cheap and steep, at the price of possible instability
and non-linear phase. The coefficient table *is* the filter: change the numbers and the same
code becomes a low-pass, a band-pass, or a pulse shaper.

The frequency response and the impulse response are a Fourier-transform pair, so filter
design is the art of choosing an impulse response whose transform has the passband and
stopband you want. A [window function](/reference/window-function/) is often applied to trade
transition sharpness against stopband ripple.

## Variants

- **FIR** — always stable, can be made exactly linear-phase (no group-delay distortion, vital
  for digital modulation); the default in SDR channelisers. Includes the
  [root-raised-cosine](/reference/root-raised-cosine-filter/) and
  [matched](/reference/matched-filter/) filters used for pulse shaping.
- **IIR** — Butterworth, Chebyshev, elliptic responses; steep for few coefficients, used for
  audio and control loops where phase linearity matters less.
- **[CIC](/reference/cic-filter/)** — a multiplierless integrator-comb structure for large
  [decimation](/reference/decimation/) factors in hardware.
- **[Adaptive filters](/reference/adaptive-filter/)** — coefficients that update themselves,
  as in equalisers that cancel multipath.

## In practice

By response shape, a low-pass keeps frequencies below a cutoff, a band-pass keeps a chosen
range, and a notch removes a narrow interferer. In an SDR the workhorse is the narrow
**channel filter** that isolates one signal from a wide capture, almost always paired with
decimation so the two run as a single efficient stage (a polyphase decimating FIR). Filters
also appear as the pulse-shaping and matched filters that maximise signal-to-noise at the
symbol decision, and as the loop filters inside timing and carrier recovery.

## Relevance to SDR

Filtering is fundamental to channelising the [IQ](/reference/iq-data/) stream and to pulse
shaping. GopherTrunk uses low-pass FIR filters inside its
[down-converters](/reference/digital-down-converter/) to isolate each control and voice
channel, and matched/root-raised-cosine filtering in the demodulator to recover symbols
cleanly.

## Sources

[^wiki]: [Digital filter](https://en.wikipedia.org/wiki/Digital_filter) — Wikipedia, on FIR/IIR families and frequency response.
[^fir]: [Finite impulse response](https://en.wikipedia.org/wiki/Finite_impulse_response) — Wikipedia, on the tapped-delay-line filter and its linear-phase property.
