---
slug: semiconductor
title: Semiconductor
entry_type: concept
category: hw-foundations
description: A semiconductor is a material whose electrical conductivity sits between a conductor and an insulator and can be precisely controlled, making it the basis of transistors and chips.
keywords: semiconductor, silicon, doping, conductivity, transistor, p-type, n-type, fabrication
infobox:
  - { label: Type, value: Material class }
  - { label: Conductivity, value: Between conductor & insulator }
  - { label: Key material, value: Silicon }
  - { label: Tuned by, value: Doping }
see_also: [transistor, integrated-circuit, logic-gate, moores-law, central-processing-unit, x86]
cite_urls:
  - https://en.wikipedia.org/wiki/Semiconductor
---

A **semiconductor** is a material whose electrical conductivity falls between that of a conductor and an insulator and — crucially — can be precisely controlled.[^wiki]

## Overview

Silicon is the workhorse semiconductor. By *doping* it with tiny amounts of other elements, makers create regions that carry charge differently (*n-type* and *p-type*), and the junctions between them behave in controllable ways. That controllability is what makes a [transistor](/reference/transistor/) possible: a small signal can switch or modulate a larger current. Many such structures are then fabricated together on one die to form an [integrated circuit](/reference/integrated-circuit/).

## Where it fits

Semiconductors are the physical foundation under all of digital computing: without a material you can switch on and off reliably, there are no [logic gates](/reference/logic-gate/), no [CPUs](/reference/central-processing-unit/), no chips at all. The decades-long ability to shrink semiconductor features is what powered [Moore's law](/reference/moores-law/). The same physics serves radio too — the diodes, amplifiers, and mixers in an SDR front end are semiconductor devices handling the RF before GopherTrunk decodes it.

## Sources

[^wiki]: [Semiconductor](https://en.wikipedia.org/wiki/Semiconductor) — Wikipedia, on semiconductor materials, doping, and conductivity.
