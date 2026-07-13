---
slug: minimum-shift-keying
title: Minimum-shift keying (MSK)
entry_type: technology
category: modulation
description: Minimum-shift keying (MSK) is continuous-phase FSK with modulation index 0.5, the smallest that keeps tones orthogonal; it is the constant-envelope precursor to GMSK.
keywords: MSK, minimum shift keying, CPFSK, modulation index 0.5, continuous phase, orthogonal tones, GMSK, constant envelope, OQPSK
aka: [minimum-shift keying, MSK]
autolink: true
infobox:
  - { label: Type, value: Continuous-phase FSK }
  - { label: Modulation index, value: h = 0.5 (minimum orthogonal) }
  - { label: Leads to, value: GMSK }
see_also: [continuous-phase-modulation, gmsk, frequency-shift-keying, four-fsk, phase-shift-keying, modulation-index]
cite_urls:
  - https://en.wikipedia.org/wiki/Minimum-shift_keying
  - https://en.wikipedia.org/wiki/Continuous_phase_modulation
---

**Minimum-shift keying** (**MSK**) is a form of binary continuous-phase
[FSK](/reference/frequency-shift-keying/) in which the frequency spacing is the smallest
that still keeps the two tones orthogonal — a
[modulation index](/reference/modulation-index/) of exactly **h = 0.5**.[^wiki] Keeping
the phase continuous across symbol boundaries gives MSK a **constant envelope** and a
compact spectrum, and it is the direct precursor to
[GMSK](/reference/gmsk/), which simply adds a Gaussian pre-filter.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A phase trellis rising by plus or minus ninety degrees per symbol interval, showing the continuous phase of minimum-shift keying." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="mskar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="110" x2="440" y2="110" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#mskar)"/>
  <line x1="30" y1="20" x2="30" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M30 110 L110 70 L190 30 L270 70 L350 30 L430 70" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="34" y="16" font-size="9" fill="currentColor">phase</text>
  <text x="360" y="126" font-size="9" fill="currentColor">time (symbols)</text>
  <text x="120" y="55" font-size="9" fill="currentColor">±90° per symbol, always continuous</text>
</svg>
<figcaption>MSK advances the carrier phase by exactly ±90° each symbol with no jumps — continuous-phase FSK at the minimum orthogonal spacing.</figcaption>
</figure>

## How it works

In MSK, a data one nudges the instantaneous frequency up and a zero nudges it down, but
by just enough that the carrier phase rotates by exactly **±90° over one symbol
period**. Because the deviation equals one-quarter of the bit rate, the two tones differ
by half the bit rate — the minimum separation at which they remain orthogonal and can be
distinguished by a matched detector. The phase never jumps; it ramps smoothly, so the
signal has a constant amplitude and no abrupt spectral splatter.

There is a second, equivalent way to see MSK: it is identical to **offset QPSK
(OQPSK)** shaped with half-sine symbol pulses. The even and odd bits drive the I and Q
channels, staggered by half a symbol, and the half-sinusoid weighting turns the abrupt
[QPSK](/reference/qpsk/) phase steps into the smooth ±90° ramps above. This duality lets
MSK be built and demodulated with either an FSK-style frequency discriminator or a
coherent I/Q [PSK](/reference/phase-shift-keying/) receiver.

## In practice

MSK's constant envelope means it can be amplified by cheap, efficient saturated power
amplifiers without spectral regrowth — a decisive advantage over
[ASK](/reference/amplitude-shift-keying/) or unfiltered PSK. Its main limitation is that
the raw spectrum, while compact, still has sidelobes that fall off only moderately fast.
Passing the data through a Gaussian filter before the modulator squeezes those sidelobes
down further, producing [GMSK](/reference/gmsk/) — the modulation of GSM, AIS, and
D-STAR. MSK is thus best understood as the clean, unfiltered baseline that GMSK refines.

## Relevance to SDR

MSK and its GMSK descendant are everywhere in software radio: GSM cellular bursts, AIS
ship reports, some satellite telemetry, and a range of low-power data links all rest on
this continuous-phase family. Recognising a signal as MSK on a waterfall — a narrow,
constant-power carrier with no amplitude blinking — tells you to reach for either a
frequency-discriminator or a coherent OQPSK-style demodulator.

GopherTrunk decodes the closely related C4FM/4FSK land-mobile modes directly and uses a
GMSK demodulator in its AIS path; pure MSK is documented here as the theoretical hinge
between the FSK and PSK worlds and as the parent of the GMSK that GopherTrunk actually
implements.

## Sources

[^wiki]: [Minimum-shift keying](https://en.wikipedia.org/wiki/Minimum-shift_keying) — Wikipedia, for the h = 0.5 continuous-phase definition, the OQPSK/half-sine equivalence, and the relationship to GMSK.
