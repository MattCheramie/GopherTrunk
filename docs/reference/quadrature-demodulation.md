---
slug: quadrature-demodulation
title: Quadrature demodulation
entry_type: algorithm
category: sdr-dsp
description: Quadrature demodulation recovers FM or PM by taking the phase difference between successive complex IQ samples with an arctangent discriminator; it is the standard SDR FM demodulator feeding C4FM symbol slicing.
keywords: quadrature demodulation, FM demodulation, IQ demodulation, phase discriminator, arctangent demodulator, differentiated phase, polar discriminator, instantaneous frequency, C4FM, FSK, SDR
aka: [quadrature FM demodulator, arctan discriminator, polar discriminator]
autolink: true
infobox:
  - { label: Type, value: FM/PM demodulator }
  - { label: Recovers, value: Instantaneous frequency from IQ }
  - { label: Feeds, value: C4FM / FSK symbol slicing }
see_also: [demodulation, frequency-modulation, iq-data, c4fm, cordic]
cite_urls:
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
  - https://en.wikipedia.org/wiki/Frequency_modulation
---

**Quadrature demodulation** recovers a frequency- or phase-modulated signal by measuring the
**phase change between successive complex [IQ](/reference/iq-data/) samples** — the rate of
phase rotation *is* the instantaneous frequency, so a differentiated-phase (arctangent)
discriminator directly yields the [FM](/reference/frequency-modulation/) baseband.[^wiki] It
is the workhorse FM/FSK demodulator in software radio and the front end that feeds
[C4FM](/reference/c4fm/) symbol slicing in P25 and DMR.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Two consecutive IQ samples shown as vectors on the complex plane; the small angle swept between them, divided by the sample interval, is the instantaneous frequency that forms the demodulated output." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="30" y1="115" x2="30" y2="20"/><line x1="30" y1="115" x2="200" y2="115"/>
  </g>
  <g font-size="8" fill="currentColor"><text x="20" y="18">Q</text><text x="196" y="128">I</text></g>
  <g stroke="currentColor" stroke-width="1.6" fill="none">
    <line x1="30" y1="115" x2="150" y2="55"/><line x1="30" y1="115" x2="120" y2="30"/>
  </g>
  <path d="M138 61 A70 70 0 0 0 108 40" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <g font-size="8" fill="currentColor"><text x="152" y="52">z[n]</text><text x="122" y="26">z[n+1]</text><text x="120" y="58">Δφ</text></g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="250" y="60" font-size="8">Δφ = arg( z[n+1] · z*[n] )</text>
    <line x1="250" y1="70" x2="250" y2="88" stroke="currentColor" marker-end="url(#qdar)"/>
    <rect x="300" y="70" width="130" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="365" y="89" font-size="8">Δφ / Δt = demod out</text>
    <line x1="290" y1="85" x2="300" y2="85" stroke="currentColor" marker-end="url(#qdar)"/>
  </g>
  <defs><marker id="qdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The angle swept between consecutive IQ vectors is the instantaneous frequency; computing it as the argument of z[n+1]·conj(z[n]) is quadrature FM demodulation.</figcaption>
</figure>

## How it works

A complex baseband sample is a vector *z[n] = I[n] + jQ[n]* whose angle is the signal's
instantaneous phase. Frequency is the *time derivative* of phase, so the demodulator's job is
to estimate how far the vector rotated from one sample to the next:

- **Phase difference.** Multiply the current sample by the **conjugate** of the previous one,
  *z[n]·z\*[n−1]*. This product's angle is exactly the phase increment Δφ between the two
  samples, and its magnitude is the product of amplitudes (which FM ignores).
- **Take the argument.** Δφ = `atan2(Q_prod, I_prod)`. Dividing by the sample interval scales
  it to Hz, but for demodulation the arctangent output is already the FM message signal. This
  is the **differentiated-phase** or **polar** discriminator.
- **Amplitude-blind.** Because only the *angle* matters, the demodulator is inherently immune
  to amplitude variations and needs no envelope tracking — ideal for constant-envelope modes
  like FM and FSK. An AGC or limiter still helps keep the arctangent well-conditioned.
- **Cheap approximations.** `atan2` can be replaced by a [CORDIC](/reference/cordic/) rotation
  or, for small Δφ, by the algebraic form (I·ΔQ − Q·ΔI)/(I²+Q²), which avoids a transcendental
  call in a hot loop.

Because the whole operation lives in complex baseband after the tuner has mixed the signal to
zero IF, it is a natural fit for SDR — no analog discriminator, PLL, or slope detector is
needed.

## In practice

The raw discriminator output is noisy, so it is normally followed by matched or low-pass
filtering. For digital voice the demodulated frequency deviation maps onto discrete symbol
levels: four-level **[C4FM](/reference/c4fm/)** (±1800, ±600 Hz deviations) for P25 Phase 1
and DMR, or two-level FSK for POCSAG and many telemetry links. The demodulator's output is
then symbol-timing-recovered and sliced into dibits or bits. A discriminator-based receiver is
simple and robust, though for the CQPSK/linear-modulation view of the same C4FM signal a
coherent phase-tracking demodulator can perform better on weak signals.

## Relevance to SDR

Quadrature demodulation is the standard FM/FSK demod in essentially every SDR stack, and
GopherTrunk is no exception: after the digital down-converter tunes and decimates a channel to
complex baseband, an arctangent/differentiated-phase discriminator produces the frequency
baseband that C4FM and FSK slicers turn into symbols. It is the hinge between raw
[IQ](/reference/iq-data/) and bits for P25, DMR, NXDN, and paging modes — the concrete meaning
of "[demodulation](/reference/demodulation/)" in the decode chain.

## Sources

[^wiki]: [Frequency modulation](https://en.wikipedia.org/wiki/Frequency_modulation) — Wikipedia, for instantaneous frequency as the derivative of phase, the basis of the discriminator.
