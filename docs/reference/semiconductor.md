---
slug: semiconductor
title: Semiconductor
entry_type: concept
category: hw-foundations
description: A semiconductor is a material whose electrical conductivity sits between a conductor and an insulator and can be precisely controlled by doping, making it the basis of transistors and chips.
keywords: semiconductor, silicon, doping, conductivity, transistor, p-type, n-type, pn junction, fabrication, band gap
infobox:
  - { label: Type, value: Material class }
  - { label: Conductivity, value: Between conductor & insulator }
  - { label: Key material, value: Silicon }
  - { label: Tuned by, value: Doping (n-type / p-type) }
  - { label: Key structure, value: The p–n junction }
see_also: [transistor, integrated-circuit, logic-gate, moores-law, central-processing-unit, x86]
cite_urls:
  - https://en.wikipedia.org/wiki/Semiconductor
---

A **semiconductor** is a material whose electrical conductivity falls between that of a conductor and an insulator and — crucially — can be precisely controlled.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A p-n junction. On the left, n-type silicon doped to have extra free electrons shown as minus signs; on the right, p-type silicon doped to have holes shown as plus signs. Where they meet, a depletion region forms, and this junction conducts current in only one direction." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="60" y="40" width="150" height="64" fill="currentColor" fill-opacity="0.06"/>
    <rect x="250" y="40" width="150" height="64" fill="currentColor" fill-opacity="0.16"/>
    <line x1="210" y1="40" x2="210" y2="104"/>
    <line x1="250" y1="40" x2="250" y2="104"/>
    <rect x="210" y="40" width="40" height="64" fill="currentColor" fill-opacity="0.02" stroke-dasharray="3 3"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="10" text-anchor="middle">
    <text x="90" y="64">&#8722;</text><text x="120" y="64">&#8722;</text><text x="150" y="64">&#8722;</text><text x="180" y="64">&#8722;</text>
    <text x="90" y="90">&#8722;</text><text x="120" y="90">&#8722;</text><text x="150" y="90">&#8722;</text><text x="180" y="90">&#8722;</text>
    <text x="280" y="64">+</text><text x="310" y="64">+</text><text x="340" y="64">+</text><text x="370" y="64">+</text>
    <text x="280" y="90">+</text><text x="310" y="90">+</text><text x="340" y="90">+</text><text x="370" y="90">+</text>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="135" y="30" font-size="8.5" font-weight="600">n-type</text>
    <text x="135" y="122" font-size="7.5" fill-opacity="0.85">extra electrons</text>
    <text x="325" y="30" font-size="8.5" font-weight="600">p-type</text>
    <text x="325" y="122" font-size="7.5" fill-opacity="0.85">holes</text>
    <text x="230" y="122" font-size="7" fill-opacity="0.8">depletion</text>
    <text x="230" y="150" font-size="8" fill-opacity="0.9">the p–n junction conducts one way — the seed of the diode and transistor</text>
  </g>
</svg>
<figcaption>Doping one side of silicon to have spare electrons (n-type) and the other to have holes (p-type) creates a p–n junction that passes current in only one direction — the fundamental structure from which diodes and transistors are built.</figcaption>
</figure>

## Overview

Silicon is the workhorse semiconductor. In its pure form it barely conducts, but by *doping* it with tiny amounts of other elements, makers create regions that carry charge differently: *n-type* silicon has spare mobile electrons (negative carriers), while *p-type* has "holes" that behave as positive carriers. Neither alone is remarkable — the magic is at the boundary.

Where an n-type region meets a p-type region, a *p–n junction* forms that conducts current freely in one direction and blocks it in the other. That one-way behaviour is a diode; layering junctions so that a small signal controls a larger current gives a [transistor](/reference/transistor/). Many such structures are then fabricated together on one die to form an [integrated circuit](/reference/integrated-circuit/).

## Where materials sit

A material's conductivity is what places it in one of three broad classes, and the semiconductor's value is that it can be pushed toward either extreme on demand:

| Class | Conductivity | Example | Role |
|-------|--------------|---------|------|
| Conductor | High | Copper | Wires, contacts |
| Semiconductor | Controllable | Silicon | Switches, chips |
| Insulator | Very low | Glass, rubber | Isolation |

Because doping and applied voltage can swing a semiconductor between conducting and blocking, it can act as a controllable switch — exactly what digital logic needs.

## Where it fits

Semiconductors are the physical foundation under all of digital computing: without a material you can switch on and off reliably, there are no [logic gates](/reference/logic-gate/), no [CPUs](/reference/central-processing-unit/), no chips at all. The decades-long ability to shrink semiconductor features is what powered [Moore's law](/reference/moores-law/). The same physics serves radio too — the diodes, amplifiers, and mixers in an SDR front end are semiconductor devices handling the RF before GopherTrunk ever sees a sample.

## Sources

[^wiki]: [Semiconductor](https://en.wikipedia.org/wiki/Semiconductor) — Wikipedia, on semiconductor materials, doping, n-type and p-type, and the p–n junction.
