---
slug: resonance
title: Resonance
entry_type: term
category: rf-fundamentals
description: Resonance is the frequency where a system's reactances cancel and its response peaks; for an LC circuit f0 = 1/(2π√(LC)), the basis of tuning and filtering.
keywords: resonance, resonant frequency, LC circuit, tuned circuit, f0, series resonance, parallel resonance, tank circuit
aka: [resonant frequency, tuned circuit, tank circuit]
autolink: true
infobox:
  - { label: Symbol, value: "f0 (ω0)" }
  - { label: Unit, value: "hertz (Hz)" }
  - { label: Formula, value: "f0 = 1 / (2π√(LC))" }
see_also: [q-factor, impedance, rf-filter, crystal-filter, cavity-filter, antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Resonance
  - https://en.wikipedia.org/wiki/LC_circuit
---

**Resonance** is the condition at which a system stores and exchanges energy most
readily at a particular frequency, so a small periodic drive produces a large
response.[^wiki] In an electrical LC circuit it occurs where the inductive and
capacitive [reactances](/reference/impedance/) become equal and cancel, leaving a
purely resistive [impedance](/reference/impedance/), at the **resonant frequency
f₀ = 1/(2π√(LC))**. Resonance is the mechanism behind tuning a radio, the sharpness
of a filter, and the operation of an [antenna](/reference/antenna/), and its
selectivity is quantified by the [Q factor](/reference/q-factor/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An LC tank circuit of an inductor and capacitor in parallel, next to a response curve peaking sharply at the resonant frequency f0 where inductive and capacitive reactance cancel." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="40" y1="45" x2="40" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <line x1="120" y1="45" x2="120" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <line x1="40" y1="45" x2="120" y2="45" stroke="currentColor" stroke-width="1.5"/>
  <line x1="40" y1="120" x2="120" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <path d="M40 70 q6 -6 12 0 t12 0 t12 0" fill="none" stroke="currentColor" stroke-width="1.5" transform="translate(4,12)"/>
  <text x="30" y="88" font-size="9" fill="currentColor">L</text>
  <line x1="112" y1="75" x2="128" y2="75" stroke="currentColor" stroke-width="1.5"/>
  <line x1="112" y1="85" x2="128" y2="85" stroke="currentColor" stroke-width="1.5"/>
  <text x="132" y="84" font-size="9" fill="currentColor">C</text>
  <text x="45" y="140" font-size="9" fill="currentColor">LC tank</text>
  <line x1="220" y1="20" x2="220" y2="140" stroke="currentColor" stroke-width="1" marker-end="url(#rsar)"/>
  <line x1="220" y1="140" x2="440" y2="140" stroke="currentColor" stroke-width="1" marker-end="url(#rsar)"/>
  <path d="M240 138 Q330 -25 420 138" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="330" y1="35" x2="330" y2="140" stroke="currentColor" stroke-width="0.7" stroke-dasharray="3 3"/>
  <text x="330" y="153" font-size="9" text-anchor="middle" fill="currentColor">f0</text>
  <text x="360" y="45" font-size="8" fill="currentColor">peak response</text>
</svg>
<figcaption>An LC tank resonates where inductive and capacitive reactance cancel; its response peaks sharply at f0 = 1/(2π√(LC)), the basis of tuning and filtering.</figcaption>
</figure>

## How it works

An inductor's reactance rises with frequency (*XL = 2πfL*) while a capacitor's falls
(*XC = 1/(2πfC)*). At one frequency these are equal in magnitude and, because they
have opposite sign, they cancel. That crossover is resonance, and setting *XL = XC*
and solving gives *f₀ = 1/(2π√(LC))*. At that point the reactive parts of the
impedance vanish and the circuit looks purely resistive, so voltage and current fall
back into phase and the system exchanges energy most efficiently with its drive.

The two canonical topologies behave as mirror images:

- **Series resonance** — L and C in series present *minimum* impedance at f₀, so a
  series-resonant branch passes the resonant frequency and blocks others. Used to
  short unwanted frequencies to ground or to pass a wanted one.
- **Parallel resonance** (a "tank") — L and C in parallel present *maximum* impedance
  at f₀, so a tank develops a large voltage there and rejects that frequency from a
  series path. Used as the frequency-selecting element in oscillators and filters.

Mechanically the same idea appears as a struck bell or a plucked string: energy
sloshes between two stores (kinetic and potential, or magnetic and electric) at a
natural frequency, and driving it there builds a large amplitude.

## In practice

How *sharp* the resonance is depends on loss, captured by the
[Q factor](/reference/q-factor/): high Q gives a tall, narrow peak that selects one
frequency tightly, low Q a broad hump that responds over a wider span. Resonators are
realised in many forms with wildly different Q — lumped LC circuits, quartz
[crystals](/reference/crystal-filter/), coaxial [cavities](/reference/cavity-filter/),
dielectric pucks, and mechanical/SAW structures — chosen for the stability and
bandwidth an application needs. An [antenna](/reference/antenna/) is itself a
resonant structure: a half-wave dipole resonates where its length matches half the
signal's wavelength, which is why antennas are cut for a target band.

## Relevance to SDR

Resonance underlies the analog scaffolding around any SDR. Preselector and
[RF filter](/reference/rf-filter/) stages use resonant elements to pass the wanted
band and reject strong out-of-band energy before it can overload the front end, and
the [local oscillator](/reference/local-oscillator/) that the receiver mixes against
is stabilised by a high-Q resonant reference. Even the antenna and its matching
network are resonant devices tuned to the band of interest. The purity and
selectivity these resonators provide set the quality of the samples the SDR digitises.

In GopherTrunk the frequency selection that a physical tuned circuit performs is done
downstream in software: the digital [channelizer](/reference/channelizer/) and
channel filters isolate the wanted channel from the wideband IQ stream, playing the
role a resonant filter plays in analog radios. The analog resonances upstream (in the
preselector and reference oscillator) still bound what reaches the ADC, so
GopherTrunk depends on good physical resonance even though it does not implement it.

## Sources

[^wiki]: [Resonance](https://en.wikipedia.org/wiki/Resonance) — Wikipedia, resonance as the peak-response condition and the reactance-cancellation view of electrical resonance.
