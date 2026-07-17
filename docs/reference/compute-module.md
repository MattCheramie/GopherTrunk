---
slug: compute-module
title: Compute Module
entry_type: hardware
category: hw-sbc
description: A Compute Module is a single-board computer stripped to a small module without ports, designed to be soldered or socketed into a custom carrier board for embedded products.
keywords: Raspberry Pi Compute Module, CM4, CM5, SO-DIMM module, carrier board, system on module, embedded SBC, custom carrier
aka: [Raspberry Pi Compute Module, CM4]
infobox:
  - { label: Type, value: Embeddable SBC module }
  - { label: Form, value: SO-DIMM or board-to-board }
  - { label: Needs, value: A custom carrier board }
  - { label: Runs, value: Linux }
  - { label: Best-known, value: Raspberry Pi CM4 / CM5 }
see_also: [raspberry-pi, single-board-computer, system-on-a-chip, hat-add-on-board, gpio, embedded-system]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi#Compute_Module
---

**A Compute Module** is a [single-board computer](/reference/single-board-computer/) stripped down to a small module — the processor, memory, and storage — without the usual ports, meant to plug into a custom carrier board.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A Compute Module plugging into a carrier board. The small module carries only the system-on-chip, RAM, and storage, and its edge connector mates into a slot on a larger custom carrier board. The carrier is where the product designer places the actual ports — USB, Ethernet, GPIO, power — laid out exactly as the product needs." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="96" width="400" height="70" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="150" y="120" width="160" height="10" rx="2" fill-opacity="0.2" fill="currentColor"/>
    <rect x="150" y="22" width="160" height="66" rx="4" fill-opacity="0.1" fill="currentColor"/>
    <rect x="164" y="34" width="30" height="24" rx="2" fill-opacity="0.2" fill="currentColor"/>
    <rect x="204" y="34" width="20" height="24" rx="2" fill-opacity="0.16" fill="currentColor"/>
    <rect x="234" y="34" width="20" height="24" rx="2" fill-opacity="0.16" fill="currentColor"/>
    <path d="M175 88 V120 M285 88 V120" stroke-width="1.1"/>
    <rect x="46" y="120" width="14" height="30" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="66" y="120" width="14" height="30" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="336" y="120" width="24" height="18" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="370" y="120" width="24" height="18" rx="2" fill-opacity="0.14" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="230" y="19" text-anchor="middle" font-size="9" font-weight="600">Compute Module</text>
    <text x="179" y="50" text-anchor="middle" font-size="7.5">SoC</text>
    <text x="214" y="50" text-anchor="middle" font-size="7">RAM</text>
    <text x="244" y="50" text-anchor="middle" font-size="7">eMMC</text>
    <text x="230" y="128" text-anchor="middle" font-size="7.5" fill-opacity="0.9">edge connector</text>
    <text x="230" y="160" text-anchor="middle" font-weight="600">custom carrier board</text>
    <text x="63" y="143" text-anchor="middle" font-size="7">ports</text>
    <text x="365" y="132" text-anchor="middle" font-size="7">ports</text>
  </g>
</svg>
<figcaption>A Compute Module holds only the brains — SoC, RAM, and storage — and mates through an edge connector into a carrier board that the product designer lays out with exactly the ports and shape the product requires.</figcaption>
</figure>

## Overview

The best-known example is the Raspberry Pi Compute Module (CM4, CM5), but the idea is general: take the [system on a chip](/reference/system-on-a-chip/) and memory of a board like the [Raspberry Pi](/reference/raspberry-pi/) and put it on a compact module that breaks all of its signals out to an edge connector. A product designer then lays out a *carrier board* that routes the [GPIO](/reference/gpio/), USB, Ethernet, and power exactly as the product needs, rather than working around a fixed consumer layout.

This separation lets one well-supported compute module serve many different products. The same module drops into a digital-signage player, an industrial gateway, or a camera, and each carrier exposes only the connectors that product uses. It also decouples the two design cadences: the module vendor handles the tricky high-speed SoC-to-memory routing, while your team designs the comparatively simple carrier.

## Module vs stock board

| | Stock SBC | Compute Module |
|---|-----------|----------------|
| Ports | Fixed, on the board | You choose, on the carrier |
| Shape / size | Fixed | Whatever the product needs |
| Effort to deploy | Plug in and go | Design + build a carrier |
| Best for | Prototypes, hobby, general use | Volume embedded products |
| Extending | [HAT](/reference/hat-add-on-board/) on the header | Integrated into the carrier |

## Where it fits

A Compute Module is the choice when an [embedded system](/reference/embedded-system/) needs the brains of an SBC but its own enclosure, connectors, and shape — a step beyond bolting a [HAT](/reference/hat-add-on-board/) onto a stock board. In a GopherTrunk-style product, a Compute Module on a custom carrier could host the SDR front end, timing, and storage in a sealed, antenna-mounted capture node with only the connectors it actually uses. The trade-off is engineering effort: you design, lay out, and manufacture the carrier yourself, which only pays off past a certain volume.

## Sources

[^wiki]: [Raspberry Pi Compute Module](https://en.wikipedia.org/wiki/Raspberry_Pi#Compute_Module) — Wikipedia, on the modular, carrier-board form of the Raspberry Pi.
