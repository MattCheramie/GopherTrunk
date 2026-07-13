---
slug: hilbert-transform
title: Hilbert transform
entry_type: algorithm
category: algorithms
description: The Hilbert transform applies a 90-degree phase shift to build the analytic signal, turning a real SDR input into complex I/Q for SSB, envelope, and frequency detection.
keywords: Hilbert transform, analytic signal, 90 degree phase shift, quadrature, real to complex, I/Q generation, single sideband, envelope detection, instantaneous frequency, David Hilbert
aka: [Hilbert transform, analytic signal, quadrature filter]
autolink: true
infobox:
  - { label: Type, value: 90-degree phase-shift filter }
  - { label: Produces, value: Analytic (complex I/Q) signal }
  - { label: Used for, value: SSB, envelope, instantaneous freq }
see_also: [iq-data, single-sideband, quadrature-demodulation, fir-filter, airspy]
cite_urls:
  - https://en.wikipedia.org/wiki/Hilbert_transform
  - https://en.wikipedia.org/wiki/Analytic_signal
---

The **Hilbert transform** shifts every frequency component of a real signal by −90° without
changing its amplitude, and pairing the original signal with its Hilbert transform on the
imaginary axis produces the **analytic signal** — a complex-valued
[I/Q](/reference/iq-data/) representation with no negative-frequency content.[^wiki][^as] That
real-to-complex conversion is the bridge between a receiver that samples a single real
voltage and the complex baseband on which nearly all modern
[demodulation](/reference/demodulation/) is done.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A real input splits into two paths: a direct path forming the in-phase component and a 90-degree phase-shift Hilbert filter forming the quadrature component, combining into a complex analytic signal." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="15" y1="75" x2="60" y2="75" stroke="currentColor" stroke-width="1.2" marker-end="url(#hbar)"/><text x="35" y="67">real x(t)</text>
    <circle cx="70" cy="75" r="4" fill="currentColor"/>
    <path d="M70 75 V 35 H 130" fill="none" stroke="currentColor" stroke-width="1.2" marker-end="url(#hbar)"/>
    <path d="M70 75 V 115 H 130" fill="none" stroke="currentColor" stroke-width="1.2" marker-end="url(#hbar)"/>
    <rect x="130" y="24" width="80" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="170" y="39">delay (I)</text>
    <rect x="130" y="103" width="110" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="185" y="118">Hilbert 90&#176; (Q)</text>
    <line x1="210" y1="36" x2="300" y2="36" stroke="currentColor" stroke-width="1.2" marker-end="url(#hbar)"/>
    <line x1="240" y1="115" x2="300" y2="115" stroke="currentColor" stroke-width="1.2" marker-end="url(#hbar)"/>
    <rect x="300" y="55" width="90" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="345" y="72">analytic</text><text x="345" y="84">I + jQ</text>
    <path d="M300 40 H 320 V 55" fill="none" stroke="currentColor" stroke-width="1.1"/>
    <path d="M300 115 H 320 V 95" fill="none" stroke="currentColor" stroke-width="1.1"/>
  </g>
  <defs><marker id="hbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The real input becomes the in-phase (I) arm; a 90-degree Hilbert filter forms the quadrature (Q) arm, and together they make the complex analytic signal.</figcaption>
</figure>

## How it works

In the frequency domain the ideal Hilbert transform is a filter with unit magnitude and a
±90° phase shift: it multiplies positive frequencies by −j and negative frequencies by +j.
Adding `j·H{x}` to the original `x` doubles the positive frequencies and exactly cancels the
negative ones, leaving a one-sided (analytic) spectrum. Written as a complex signal
`z(t) = x(t) + j·x̂(t)`, three quantities fall straight out:

- **Envelope** — the instantaneous amplitude is `|z(t)|`, giving distortion-free AM envelope
  detection.
- **Instantaneous phase and frequency** — the angle of `z(t)` is the instantaneous phase, and
  its time derivative is the instantaneous frequency, the basis of one kind of FM
  discriminator.
- **Sideband selection** — because `z(t)` has no negative-frequency image, it can be frequency
  shifted and its real part taken to synthesise or select one
  [sideband](/reference/single-sideband/) cleanly.

The ideal transform is non-causal and infinite, so in practice it is realised as a finite
[FIR filter](/reference/fir-filter/) — an odd-length, anti-symmetric kernel that approximates
the 90° shift over the band of interest — or, more commonly in DSP, by taking an FFT, zeroing
the negative-frequency bins and doubling the positive ones, and inverse-transforming.

## In practice

Two closely related structures show up constantly. The **Weaver** and **phasing** methods of
single-sideband generation and reception use a Hilbert (quadrature) network to reject the
unwanted sideband without a steep analog filter. And the general recipe "make the signal
complex, then process" almost always starts with an analytic-signal step, whether by Hilbert
filtering a real feed or by direct quadrature down-conversion.

## Relevance to SDR

Many SDR front ends already deliver complex [I/Q](/reference/iq-data/) by mixing against a
[local oscillator](/reference/local-oscillator/) in two quadrature phases, so no explicit
Hilbert stage is needed. But real-input receivers do need one: the **Airspy R2/Mini**, for
example, sample a real IF and apply a real-to-complex conversion — a Hilbert/analytic step —
to produce the I/Q the rest of the chain expects, and the same is true of many direct-sampling
HF radios and sound-card SDRs. Once the signal is analytic, the Hilbert-derived envelope and
instantaneous-frequency operations feed AM and FM demodulation, and the phasing method
underpins [SSB](/reference/single-sideband/) work.

GopherTrunk consumes complex I/Q from its supported devices and does its
[quadrature demodulation](/reference/quadrature-demodulation/) and symbol recovery on that
baseband, so where a device emits real samples the real-to-complex (Hilbert) conversion is
part of getting the data into GT's expected form rather than something GT reinvents in its
trunking decoders.

## Sources

[^wiki]: [Hilbert transform](https://en.wikipedia.org/wiki/Hilbert_transform) — Wikipedia, on the 90-degree phase-shift operator and its filter realisation.
[^as]: [Analytic signal](https://en.wikipedia.org/wiki/Analytic_signal) — Wikipedia, on building a one-sided complex signal and reading envelope and instantaneous frequency from it.
