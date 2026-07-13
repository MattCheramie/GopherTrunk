---
slug: iq-modulation
title: IQ modulation
entry_type: term
category: modulation
description: "IQ modulation impresses two independent baseband signals onto one carrier using sine and cosine (quadrature) components, the universal basis of modern digital radio."
keywords: IQ modulation, I/Q, in-phase, quadrature, quadrature modulator, upconversion, downconversion, complex baseband, image reject
aka: [IQ modulation, I/Q modulation, quadrature modulation]
autolink: true
infobox:
  - { label: Type, value: Complex-baseband modulation }
  - { label: Uses, value: Sine + cosine carriers (90° apart) }
  - { label: Carries, value: Two real channels (I and Q) }
see_also: [iq-data, quadrature-amplitude-modulation, phase-shift-keying, quadrature-demodulation, mixer-rf]
cite_urls:
  - https://en.wikipedia.org/wiki/In-phase_and_quadrature_components
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
---

**IQ modulation** impresses two independent baseband signals — the **in-phase (I)** and
**quadrature (Q)** components — onto a single [carrier](/reference/carrier-wave/) by
multiplying one with a cosine and the other with a sine of the same frequency and summing
them.[^wiki] Because sine and cosine are orthogonal (90° apart), the two channels share the
same band without interfering, and any amplitude-and-phase state of the carrier can be
reached by choosing the pair (I, Q). This is the machinery behind essentially every modern
digital modulation, and the reason SDRs work in [I/Q data](/reference/iq-data/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An IQ modulator block diagram: I times cosine and Q times sine are summed to form the RF output, shown alongside a constellation point defined by its I and Q coordinates." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="iqmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="20" y="48" font-size="10" fill="currentColor">I</text>
  <line x1="30" y1="45" x2="70" y2="45" stroke="currentColor" marker-end="url(#iqmar)"/>
  <circle cx="85" cy="45" r="15" fill="none" stroke="currentColor"/><text x="85" y="49" text-anchor="middle" font-size="11" fill="currentColor">×</text>
  <text x="85" y="80" text-anchor="middle" font-size="8" fill="currentColor">cos ωt</text>
  <text x="20" y="128" font-size="10" fill="currentColor">Q</text>
  <line x1="30" y1="125" x2="70" y2="125" stroke="currentColor" marker-end="url(#iqmar)"/>
  <circle cx="85" cy="125" r="15" fill="none" stroke="currentColor"/><text x="85" y="129" text-anchor="middle" font-size="11" fill="currentColor">×</text>
  <text x="85" y="150" text-anchor="middle" font-size="8" fill="currentColor">sin ωt</text>
  <line x1="100" y1="45" x2="160" y2="80" stroke="currentColor" marker-end="url(#iqmar)"/>
  <line x1="100" y1="125" x2="160" y2="85" stroke="currentColor" marker-end="url(#iqmar)"/>
  <circle cx="175" cy="82" r="15" fill="none" stroke="currentColor"/><text x="175" y="86" text-anchor="middle" font-size="12" fill="currentColor">+</text>
  <line x1="190" y1="82" x2="235" y2="82" stroke="currentColor" marker-end="url(#iqmar)"/>
  <text x="212" y="74" text-anchor="middle" font-size="8" fill="currentColor">RF out</text>
  <line x1="300" y1="150" x2="300" y2="25" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="270" y1="90" x2="430" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="428" y="102" font-size="8" fill="currentColor">I</text><text x="306" y="35" font-size="8" fill="currentColor">Q</text>
  <line x1="300" y1="90" x2="370" y2="55" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="370" cy="55" r="3.5" fill="currentColor"/>
  <line x1="370" y1="90" x2="370" y2="55" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 2"/>
  <line x1="300" y1="55" x2="370" y2="55" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 2"/>
</svg>
<figcaption>An IQ modulator sums I·cos and Q·sin; the pair (I, Q) is the Cartesian coordinate of a point in the constellation.</figcaption>
</figure>

## How it works

A carrier of arbitrary amplitude *A* and phase *φ* can be written A·cos(ωt + φ) =
I·cos(ωt) − Q·sin(ωt), where **I = A cos φ** and **Q = A sin φ**. So instead of directly
varying amplitude and phase — awkward to do at RF — a transmitter varies two ordinary
baseband voltages, multiplies them by a cosine and a sine from the same local oscillator (a
90° hybrid supplies the quadrature pair), and adds the products. The composite carrier then
sits at whatever amplitude and phase the (I, Q) pair encodes. Plotting I horizontally and Q
vertically gives the [constellation](/reference/constellation-diagram/): a
[PSK](/reference/phase-shift-keying/) scheme places points on a circle (constant amplitude,
varying phase); [QAM](/reference/quadrature-amplitude-modulation/) fills a grid (varying
both). The receiver reverses the process — a
[quadrature demodulator](/reference/quadrature-demodulation/) mixes the incoming RF against
the same cosine and sine and low-pass filters to recover I and Q.

Mathematically the (I, Q) pair is a **complex number** I + jQ, and the modulated RF is the
real part of (I + jQ)·e^{jωt}. Treating baseband as complex is what makes the whole framework
so powerful: a positive baseband frequency ends up above the carrier and a negative one below
it, so I/Q can distinguish the two sides of the carrier that a single real signal cannot. That
distinction is the entire reason the constellation is two-dimensional — the horizontal I axis
and vertical Q axis are genuinely independent degrees of freedom, doubling the information a
carrier can hold at a given instant compared with amplitude-only or phase-only schemes.

## Relevance to SDR

IQ modulation is the reason software radio exists in the form it does. An SDR front end is
essentially an IQ demodulator that hands the CPU a stream of complex samples, I + jQ, and
almost every operation downstream — filtering, tuning, demodulation — is arithmetic on that
complex baseband. GopherTrunk consumes I/Q from RTL-SDR, Airspy, and similar receivers and
does all of its channelization and symbol recovery in the complex domain, so IQ modulation
(and its inverse) is foundational to the whole decode chain. Practical IQ hardware is
imperfect: gain or phase mismatch between the I and Q paths creates
[IQ imbalance](/reference/iq-imbalance/), which raises an unwanted image of the signal that
software must estimate and correct.

## In practice

The great practical payoff is the **image-reject** property. Because I and Q are orthogonal,
an IQ mixer can separate signal energy above the local oscillator from energy below it —
something a single real mixer cannot do. That lets [zero-IF](/reference/zero-if/) and low-IF
receivers place the LO in or near the band of interest without the mirror-image problem that
forces classic superheterodyne designs to use bulky image filters.

The same orthogonality is what lets one carrier carry two data streams. Because the cosine and
sine branches do not interfere, a modulator can send one bit stream on I and an independent one
on Q — the definition of QPSK and the general principle behind every QAM constellation. It also
means the receiver can, in software, rotate the whole constellation (multiply the complex
samples by e^{jθ}) to correct a carrier phase offset, or spin it steadily to correct a frequency
offset, using nothing but complex multiplication. These are the exact operations GopherTrunk's
carrier-recovery loops perform to lock onto a signal before slicing symbols.

## Sources

[^wiki]: [In-phase and quadrature components](https://en.wikipedia.org/wiki/In-phase_and_quadrature_components) — Wikipedia, for the I/Q decomposition and quadrature-modulator description.
