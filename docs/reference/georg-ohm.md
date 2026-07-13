---
slug: georg-ohm
title: Georg Ohm
entry_type: person
category: people
description: "Georg Ohm (1789–1854) was a German physicist who established Ohm's law, the proportional relation between voltage, current, and resistance."
keywords: Georg Ohm, Georg Simon Ohm, Ohm's law, resistance, ohm unit, voltage current resistance, conductor, electrical resistance
aka: [Georg Ohm, Georg Simon Ohm, Ohm]
autolink: true
infobox:
  - { label: Lived, value: "1789–1854" }
  - { label: Field, value: Physics }
  - { label: Known for, value: "Ohm's law" }
see_also: [impedance, gustav-kirchhoff, resonance, james-clerk-maxwell, decibel]
cite_urls:
  - https://en.wikipedia.org/wiki/Georg_Ohm
---

**Georg Ohm** (1789–1854) was a German physicist who discovered the simple proportional
relationship between the voltage across a conductor and the current through it — the law
that carries his name, **V = I·R**.[^wiki] The relation is the most basic tool in all of
electronics, and its alternating-current generalization to complex
[impedance](/reference/impedance/) governs every RF matching network and filter.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A voltage source drives a current through a resistor, with the relation V equals I times R, illustrating Ohm's law." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="goar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <circle cx="70" cy="60" r="20"/>
    <text x="70" y="64" font-size="11" fill="currentColor" stroke="none" text-anchor="middle">V</text>
    <line x1="90" y1="45" x2="180" y2="45"/>
    <path d="M180 45 l8 -6 l8 12 l8 -12 l8 12 l8 -12 l8 12 l8 -6" />
    <line x1="244" y1="45" x2="330" y2="45"/>
    <line x1="330" y1="45" x2="330" y2="75"/>
    <line x1="330" y1="75" x2="90" y2="75"/>
    <line x1="90" y1="75" x2="90" y2="45"/>
  </g>
  <line x1="120" y1="35" x2="160" y2="35" stroke="currentColor" stroke-width="1.1" marker-end="url(#goar)"/>
  <text x="140" y="28" font-size="8.5" fill="currentColor" text-anchor="middle">I</text>
  <text x="212" y="35" font-size="8.5" fill="currentColor" text-anchor="middle">R</text>
  <text x="400" y="60" font-size="11" fill="currentColor" text-anchor="middle">V = I·R</text>
</svg>
<figcaption>Ohm's law: the current I through a resistor equals the applied voltage V divided by the resistance R.</figcaption>
</figure>

## Life and work

Georg Simon Ohm was born in Erlangen, Bavaria, and worked for years as a schoolteacher
while pursuing physics research, often with apparatus he built himself. In 1827 he
published *Die galvanische Kette, mathematisch bearbeitet* ("The Galvanic Circuit
Investigated Mathematically"), stating the proportional law between potential difference
and current. The work was poorly received at first — some German academics found its
mathematical style unwelcome — and recognition came only slowly.[^wiki]

Ohm was eventually vindicated: the Royal Society awarded him the Copley Medal in 1841,
and late in life he obtained a professorship at the University of Munich. The **ohm**, the
SI unit of electrical resistance, is named in his honour.

## Contribution

Ohm's law states that, for a metallic conductor at constant temperature, the current is
directly proportional to the applied voltage, with resistance as the constant of
proportionality:

- **V = I·R** — voltage equals current times resistance.

Though elementary, it is the starting point for circuit analysis. Combined with the
conservation laws of [Gustav Kirchhoff](/reference/gustav-kirchhoff/), it lets any
resistive network be solved. Extended to alternating current, resistance becomes complex
[impedance](/reference/impedance/), which describes how capacitors and inductors respond
to frequency and thus how tuned circuits reach [resonance](/reference/resonance/) — the
everyday arithmetic of filters, attenuators, and the [decibel](/reference/decibel/)
power relations used throughout RF work.

## Legacy

Ohm's law is the first equation of electronics, quietly present in every resistor value,
every bias calculation, and every impedance match. It provided the empirical footing that
the later field theory of [James Clerk Maxwell](/reference/james-clerk-maxwell/) built
upon, and it remains indispensable in the design of the analog front end of any
[software-defined radio](/reference/software-defined-radio/).

## Sources

[^wiki]: [Georg Ohm](https://en.wikipedia.org/wiki/Georg_Ohm) — Wikipedia, for his biography, the 1827 publication of Ohm's law, and the naming of the ohm unit.
