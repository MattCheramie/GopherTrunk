---
slug: gustav-kirchhoff
title: Gustav Kirchhoff
entry_type: person
category: people
description: "Gustav Kirchhoff (1824–1887) was a German physicist whose circuit laws for current and voltage underpin all electrical and RF network analysis."
keywords: Gustav Kirchhoff, Kirchhoff's laws, Kirchhoff current law, Kirchhoff voltage law, KCL, KVL, circuit analysis, spectroscopy, blackbody radiation
aka: [Gustav Kirchhoff, Gustav Robert Kirchhoff, Kirchhoff]
autolink: true
infobox:
  - { label: Lived, value: "1824–1887" }
  - { label: Field, value: Physics }
  - { label: Known for, value: "Kirchhoff's circuit laws" }
see_also: [impedance, georg-ohm, james-clerk-maxwell, oliver-heaviside, resonance]
cite_urls:
  - https://en.wikipedia.org/wiki/Gustav_Kirchhoff
---

**Gustav Kirchhoff** (1824–1887) was a German physicist best known for the two **circuit
laws** that bear his name — the foundation of how engineers analyze any electrical
network, from a battery-and-resistor loop to an RF matching stage.[^wiki] His current law
and voltage law, together with the resistance relation of
[Georg Ohm](/reference/georg-ohm/), let a circuit's currents and voltages be solved
exactly.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A circuit node where currents entering equal currents leaving, alongside a loop where the voltages around it sum to zero, illustrating Kirchhoff's current and voltage laws." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="kfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="30" y1="55" x2="95" y2="55" marker-end="url(#kfar)"/>
    <line x1="30" y1="85" x2="95" y2="85" marker-end="url(#kfar)"/>
    <circle cx="100" cy="70" r="4" fill="currentColor"/>
    <line x1="104" y1="70" x2="165" y2="70" marker-end="url(#kfar)"/>
  </g>
  <text x="115" y="40" font-size="9" fill="currentColor" text-anchor="middle">ΣI_in = ΣI_out</text>
  <text x="60" y="118" font-size="8.5" fill="currentColor" text-anchor="middle">current law (KCL)</text>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="285" y="45" width="120" height="55" rx="3"/>
  </g>
  <text x="345" y="35" font-size="9" fill="currentColor" text-anchor="middle">ΣV = 0</text>
  <text x="345" y="118" font-size="8.5" fill="currentColor" text-anchor="middle">voltage law (KVL)</text>
</svg>
<figcaption>Kirchhoff's current law: charge entering a node equals charge leaving it. Voltage law: the voltages around any closed loop sum to zero.</figcaption>
</figure>

## Life and work

Kirchhoff was born in Königsberg, Prussia, and formulated his circuit laws in 1845 while
still a student, generalizing Ohm's work to networks of arbitrary topology. He went on to
professorships at Breslau, Heidelberg, and Berlin. Beyond circuits, he made landmark
contributions to **spectroscopy** — with the chemist Robert Bunsen he showed that each
element emits and absorbs light at characteristic wavelengths, founding the field and
leading to the discovery of new elements. His law of **thermal (blackbody) radiation**
later became a cornerstone on the road to quantum theory.[^wiki]

## Contribution

Kirchhoff's two laws are statements of conservation:

- **Current law (KCL):** at any node, the sum of currents flowing in equals the sum
  flowing out — charge is conserved.
- **Voltage law (KVL):** around any closed loop, the algebraic sum of the voltages is
  zero — energy is conserved.

Applied together, they turn a circuit into a solvable system of linear equations. For
alternating-current and radio-frequency work the same laws hold when resistance is
generalized to complex [impedance](/reference/impedance/), so they describe how signals
divide across capacitors, inductors, and transmission-line stubs — the everyday math of
filters, matching networks, and [resonance](/reference/resonance/).

## Legacy

Every SPICE simulation, every filter design, and every antenna-matching calculation rests
on Kirchhoff's laws. They sit alongside the field equations of
[James Clerk Maxwell](/reference/james-clerk-maxwell/) and the operational methods of
[Oliver Heaviside](/reference/oliver-heaviside/) as the working toolkit of electrical
engineering, taught in the first weeks of any circuits course and used silently inside
every SDR front end.

## Sources

[^wiki]: [Gustav Kirchhoff](https://en.wikipedia.org/wiki/Gustav_Kirchhoff) — Wikipedia, for his biography, the circuit laws, and his work in spectroscopy and radiation.
