---
slug: transistor
title: Transistor
entry_type: hardware
category: hw-foundations
description: A transistor is a semiconductor device that switches or amplifies electrical signals; as a tiny on/off switch it is the fundamental building block of all modern digital electronics.
keywords: transistor, semiconductor, MOSFET, switch, amplifier, BJT, gate, solid-state
infobox:
  - { label: Type, value: Semiconductor device }
  - { label: Does, value: Switches or amplifies }
  - { label: Common type, value: MOSFET }
  - { label: Invented, value: "Bell Labs, 1947" }
see_also: [semiconductor, integrated-circuit, logic-gate, moores-law, central-processing-unit, clock-speed]
cite_urls:
  - https://en.wikipedia.org/wiki/Transistor
---

A **transistor** is a [semiconductor](/reference/semiconductor/) device that switches or amplifies electrical signals — and as a tiny, electrically controlled on/off switch it is the fundamental building block of all modern digital electronics.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 196" role="img" aria-label="A transistor shown as a voltage-controlled switch between source and drain. With no voltage on the gate the switch is open and no current flows; with a voltage on the gate the switch closes and current flows from source to drain." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="52" y="46">Source</text>
    <text x="388" y="46">Drain</text>
    <text x="52" y="146">Source</text>
    <text x="388" y="146">Drain</text>
  </g>
  <circle cx="52" cy="60" r="4" fill="currentColor"/><circle cx="388" cy="60" r="4" fill="currentColor"/>
  <line x1="52" y1="60" x2="176" y2="60" stroke="currentColor" stroke-width="1.4"/>
  <line x1="264" y1="60" x2="388" y2="60" stroke="currentColor" stroke-width="1.4"/>
  <line x1="176" y1="60" x2="238" y2="38" stroke="currentColor" stroke-width="1.6"/>
  <circle cx="176" cy="60" r="3" fill="currentColor"/><circle cx="264" cy="60" r="3" fill="currentColor"/>
  <line x1="220" y1="92" x2="220" y2="66" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2" marker-end="url(#tr_ar)"/>
  <text x="220" y="104" text-anchor="middle" font-size="8.5" fill="currentColor" font-weight="600">Gate 0 V</text>
  <text x="330" y="78" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">OFF — no current</text>
  <circle cx="52" cy="160" r="4" fill="currentColor"/><circle cx="388" cy="160" r="4" fill="currentColor"/>
  <line x1="52" y1="160" x2="388" y2="160" stroke="currentColor" stroke-width="1.4"/>
  <circle cx="176" cy="160" r="3" fill="currentColor"/><circle cx="264" cy="160" r="3" fill="currentColor"/>
  <line x1="120" y1="160" x2="150" y2="160" stroke="currentColor" stroke-width="2.4" marker-end="url(#tr_ar)"/>
  <line x1="290" y1="160" x2="320" y2="160" stroke="currentColor" stroke-width="2.4" marker-end="url(#tr_ar)"/>
  <line x1="220" y1="192" x2="220" y2="166" stroke="currentColor" stroke-width="1.2" marker-end="url(#tr_ar)"/>
  <text x="220" y="184" text-anchor="middle" font-size="8.5" fill="currentColor" font-weight="600">Gate +V</text>
  <text x="330" y="150" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">ON — current flows</text>
  <defs><marker id="tr_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A small voltage on the gate controls a much larger current between source and drain: no gate voltage leaves the switch open (off), a gate voltage closes it (on). Billions of these MOSFET switches, etched together on one chip and wired into logic gates, are what carry out every computation.</figcaption>
</figure>

## Overview

A transistor uses a small voltage or current at one terminal to control a much larger current between two others, so it can act as an *amplifier* or as a *switch*. Invented at Bell Labs in 1947, it replaced the bulky, hot vacuum tube. The dominant type in digital chips is the **MOSFET**, billions of which are etched together on a single [integrated circuit](/reference/integrated-circuit/). Wired together, transistors form the [logic gates](/reference/logic-gate/) that implement all computation.

## Where it fits

Every digit a computer manipulates ultimately comes down to transistors switching on and off. Their relentless shrinking is what [Moore's law](/reference/moores-law/) describes and what lets a modern [CPU](/reference/central-processing-unit/) pack billions of switches into a fingernail-sized die. The same devices also do analog work: in an SDR front end, transistors amplify faint RF in a low-noise amplifier before the signal is digitized for GopherTrunk to decode.

## Sources

[^wiki]: [Transistor](https://en.wikipedia.org/wiki/Transistor) — Wikipedia, on the transistor as switch and amplifier and the basic building block of electronics.
