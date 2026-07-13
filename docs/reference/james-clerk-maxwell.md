---
slug: james-clerk-maxwell
title: James Clerk Maxwell
entry_type: person
category: people
description: James Clerk Maxwell (1831–1879) was a Scottish physicist whose equations unified electricity and magnetism and predicted electromagnetic waves travelling at the speed of light.
keywords: James Clerk Maxwell, Maxwell's equations, electromagnetism, electromagnetic waves, physics, displacement current, speed of light
aka: [James Clerk Maxwell]
autolink: true
infobox:
  - { label: Lived, value: "1831–1879" }
  - { label: Field, value: Physics }
  - { label: Known for, value: "Maxwell's equations" }
  - { label: Predicted, value: Electromagnetic waves }
see_also: [heinrich-hertz, oliver-heaviside, hendrik-lorentz, nikola-tesla, electromagnetic-spectrum, radio-wave, wavelength]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/James_Clerk_Maxwell
  - https://www.britannica.com/biography/James-Clerk-Maxwell
---

**James Clerk Maxwell** (1831–1879) was a Scottish physicist whose equations unified
electricity, magnetism, and light, **predicting electromagnetic waves** that travel at the
speed of light.[^wiki] That prediction — made with pen and paper roughly two decades before
anyone detected such a wave — is the theoretical foundation on which all of radio,
including [software-defined radio](/reference/software-defined-radio/), ultimately rests.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Oscillating electric and magnetic fields at right angles forming a travelling electromagnetic wave." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="440" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 60 q35 -34 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <path d="M20 60 q35 22 70 0 t70 0 t70 0 t70 0 t70 0 t50 0" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="120" y="28" font-size="9" fill="currentColor">electric field</text><text x="120" y="100" font-size="9" fill="currentColor">magnetic field</text>
</svg>
<figcaption>Maxwell's equations unified electricity and magnetism and predicted electromagnetic waves — the foundation of radio.</figcaption>
</figure>

## Life and work

Maxwell was born in Edinburgh in 1831 into a comfortable Scottish family and showed a
prodigious mathematical talent early, publishing his first academic paper on oval curves at
fourteen. He studied at Edinburgh and then Cambridge, and held chairs at Marischal College
in Aberdeen and King's College London before becoming, in 1871, the first Cavendish
Professor of Physics at Cambridge, where he designed and directed the celebrated Cavendish
Laboratory. His interests ranged far beyond electromagnetism: he produced the first durable
colour photograph, developed the kinetic theory of gases and the Maxwell–Boltzmann
distribution, and analysed the stability of Saturn's rings, showing they had to be made of
countless small particles.

His work on electromagnetism unfolded through the 1850s and 1860s, building on Michael
Faraday's experimental picture of fields as lines of force filling space. Faraday had the
physical intuition but not the mathematics; Maxwell supplied the mathematics, translating
Faraday's field lines into a rigorous system of differential equations. The decisive step
came in his 1865 paper *A Dynamical Theory of the Electromagnetic Field*.

## Contribution

Maxwell's central insight was a missing term. The known laws of electricity and magnetism,
assembled from Coulomb, Ampère, Gauss, and Faraday, were mutually inconsistent for
changing currents. Maxwell added what he called the **displacement current** — the idea
that a changing electric field acts like a current and produces a magnetic field, even in
empty space. With that term the equations became symmetric and self-consistent: a changing
electric field sustains a magnetic field, which sustains an electric field, and the pair
propagate together as a wave. When he computed the speed of that wave from purely
electrical and magnetic constants measured in the laboratory, the answer matched the
measured speed of light so closely that he concluded light itself is an electromagnetic
disturbance — one entry in a vast [electromagnetic spectrum](/reference/electromagnetic-spectrum/)
that must also contain waves of every other [wavelength](/reference/wavelength/).[^brit]
This prediction was confirmed experimentally by [Heinrich Hertz](/reference/heinrich-hertz/)
in the late 1880s, a decade after Maxwell's death.

## Legacy

Maxwell died of abdominal cancer in 1879 at only 48, never seeing his waves detected. His
original formulation used some twenty coupled equations in awkward notation;
[Oliver Heaviside](/reference/oliver-heaviside/) later recast them into the four compact
vector equations now universally taught and called "Maxwell's equations," and
[Hendrik Lorentz](/reference/hendrik-lorentz/) supplied the force law linking those fields
back to charged particles. Einstein kept a photograph of Maxwell on his study wall and
credited the field concept as the deepest change in physics since Newton. For radio the
equations are not history but working tools: they govern how an [antenna](/reference/antenna/)
radiates, how a [radio wave](/reference/radio-wave/) propagates and attenuates, how energy
couples into a receiver, and how every filter and transmission line behaves. Whatever
protocol GopherTrunk is decoding — P25, DMR, TETRA — the signal reaching the
[software-defined radio](/reference/software-defined-radio/) got there by obeying Maxwell's
equations at the speed of light.

## Sources

[^wiki]: [James Clerk Maxwell](https://en.wikipedia.org/wiki/James_Clerk_Maxwell) — Wikipedia, for biography and his equations predicting electromagnetic waves.
[^brit]: [James Clerk Maxwell](https://www.britannica.com/biography/James-Clerk-Maxwell) — Encyclopædia Britannica, for the displacement current and his identification of light as an electromagnetic wave.
