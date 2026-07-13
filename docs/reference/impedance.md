---
slug: impedance
title: Impedance (Z)
entry_type: term
category: rf-fundamentals
description: Impedance is the complex opposition a circuit or transmission line presents to alternating current; RF systems standardise on 50 ohms to match sources, lines, and loads.
keywords: impedance, characteristic impedance, 50 ohm, complex impedance, reactance, resistance, Z0, impedance matching, transmission line
aka: [Z, characteristic impedance, Z0]
autolink: true
infobox:
  - { label: Symbol, value: "Z (Z0 for lines)" }
  - { label: Unit, value: Ohm (Ω) }
  - { label: Form, value: "Z = R + jX" }
see_also: [reflection-coefficient, standing-wave-ratio, return-loss, coaxial-cable, feedpoint-impedance, antenna-tuner]
cite_urls:
  - https://en.wikipedia.org/wiki/Electrical_impedance
  - https://en.wikipedia.org/wiki/Characteristic_impedance
---

**Impedance** is the total opposition a circuit, component, or transmission line
presents to an alternating current, written as a complex number *Z = R + jX* in
ohms (Ω).[^wiki] The real part *R* is resistance and the imaginary part *X* is
reactance, which stores and returns energy through capacitance or inductance. In
radio, most sources, cables, and antennas are built around a common
**characteristic impedance** of 50 Ω so that power flows between them without
reflection.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A source with 50-ohm impedance feeding a coaxial line of 50-ohm characteristic impedance into a load, with a phasor diagram showing impedance as resistance R on the real axis plus reactance X on the imaginary axis." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="impar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="60" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="50" y="79" font-size="9" text-anchor="middle" fill="currentColor">source</text>
  <text x="50" y="112" font-size="8" text-anchor="middle" fill="currentColor">50 Ω</text>
  <line x1="80" y1="68" x2="200" y2="68" stroke="currentColor" stroke-width="1.5"/>
  <line x1="80" y1="82" x2="200" y2="82" stroke="currentColor" stroke-width="1.5"/>
  <text x="140" y="60" font-size="8" text-anchor="middle" fill="currentColor">Z0 = 50 Ω line</text>
  <rect x="200" y="55" width="50" height="40" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="225" y="79" font-size="9" text-anchor="middle" fill="currentColor">load</text>
  <line x1="320" y1="130" x2="320" y2="30" stroke="currentColor" stroke-width="1" marker-end="url(#impar)"/>
  <line x1="300" y1="120" x2="440" y2="120" stroke="currentColor" stroke-width="1" marker-end="url(#impar)"/>
  <text x="330" y="28" font-size="9" fill="currentColor">jX (reactance)</text>
  <text x="360" y="134" font-size="9" fill="currentColor">R (resistance)</text>
  <line x1="320" y1="120" x2="400" y2="60" stroke="currentColor" stroke-width="2" marker-end="url(#impar)"/>
  <text x="404" y="60" font-size="9" fill="currentColor">Z</text>
</svg>
<figcaption>Impedance combines resistance and reactance into one complex quantity; RF systems fix a 50 Ω reference so source, line, and load match.</figcaption>
</figure>

## How it works

For a resistor, opposition is purely real and independent of frequency. Capacitors
and inductors add **reactance**: an inductor's reactance is *X = 2πfL* and rises
with frequency, while a capacitor's is *X = −1/(2πfC)* and falls with frequency.
Because these reactances shift the current's phase relative to the voltage, the
combined effect is captured by a complex number whose magnitude gives the amplitude
ratio and whose angle gives the phase shift.

A **transmission line** such as [coaxial cable](/reference/coaxial-cable/) has a
**characteristic impedance** *Z0* set by its geometry and dielectric — the ratio of
voltage to current for a wave travelling along it. For a lossless line
*Z0 = √(L/C)* from the per-metre inductance and capacitance. Crucially, *Z0* is not
a resistance you can measure with an ohmmeter; it is the impedance the line *looks
like* to a signal propagating down it.

The reason impedance dominates RF engineering is the **matching** condition. When a
source of impedance *Zs* drives a load *ZL* through a line of impedance *Z0*, maximum
power transfers and no energy reflects only when the impedances are equal (for the
line, when *ZL = Z0*). A mismatch sends part of the wave back toward the source,
creating standing waves — the subject of the
[reflection coefficient](/reference/reflection-coefficient/),
[return loss](/reference/return-loss/), and
[standing-wave ratio](/reference/standing-wave-ratio/).

## In practice

The 50 Ω convention is a historical compromise: coaxial lines carry peak power near
30 Ω and lowest loss near 77 Ω, and 50 Ω sits usefully between them while giving
convenient dimensions. Video and broadcast gear instead standardised on 75 Ω. An
[antenna's feedpoint impedance](/reference/feedpoint-impedance/) rarely lands
exactly on 50 Ω across a band, so an [antenna tuner](/reference/antenna-tuner/) or a
matching network transforms it back toward the system reference. Matching is done
with reactive elements — series or shunt inductors and capacitors, quarter-wave
line sections, or transformers — chosen to cancel the load's reactance and rotate
its resistance to the target value. The Smith chart is the classic graphical tool
for this, plotting normalised impedance so matching moves become geometric steps.

## Relevance to SDR

Every software-defined radio front end presents a nominal 50 Ω input at its antenna
connector, and the low-noise amplifier, filters, and mixer behind it are all designed
around that reference. If the antenna or feedline is badly mismatched, part of the
received signal reflects instead of reaching the ADC, degrading sensitivity; on a
transmit-capable SDR the reflected power can also stress the power amplifier. This is
why receivers benefit from a resonant, reasonably matched antenna rather than a random
length of wire, even though a receive-only mismatch mainly costs signal rather than
damaging hardware.

GopherTrunk is a pure-software decoder that operates on the digital IQ stream after
the SDR's analog front end, so it does not measure or correct impedance itself — that
work happens in the hardware and cabling. Understanding impedance still matters to
GopherTrunk users because a good antenna match improves the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) the decoder ultimately
sees, and poor matching is a common cause of weak control-channel reception that no
amount of DSP can recover.

## Sources

[^wiki]: [Electrical impedance](https://en.wikipedia.org/wiki/Electrical_impedance) — Wikipedia, the complex R + jX definition, reactance, and phase relationships.
