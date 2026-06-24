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

## Overview

A transistor uses a small voltage or current at one terminal to control a much larger current between two others, so it can act as an *amplifier* or as a *switch*. Invented at Bell Labs in 1947, it replaced the bulky, hot vacuum tube. The dominant type in digital chips is the **MOSFET**, billions of which are etched together on a single [integrated circuit](/reference/integrated-circuit/). Wired together, transistors form the [logic gates](/reference/logic-gate/) that implement all computation.

## Where it fits

Every digit a computer manipulates ultimately comes down to transistors switching on and off. Their relentless shrinking is what [Moore's law](/reference/moores-law/) describes and what lets a modern [CPU](/reference/central-processing-unit/) pack billions of switches into a fingernail-sized die. The same devices also do analog work: in an SDR front end, transistors amplify faint RF in a low-noise amplifier before the signal is digitized for GopherTrunk to decode.

## Sources

[^wiki]: [Transistor](https://en.wikipedia.org/wiki/Transistor) — Wikipedia, on the transistor as switch and amplifier and the basic building block of electronics.
