---
slug: laptop
title: Laptop
entry_type: hardware
category: hw-personal-computers
description: A laptop is a complete portable personal computer with screen, keyboard, and battery built in, trading some performance and expandability for mobility.
keywords: laptop, notebook computer, portable computer, ultrabook, thermal throttling, mobile development machine, clamshell
aka: [Laptop, Notebook, Notebook computer]
infobox:
  - { label: Type, value: Portable personal computer }
  - { label: Built in, value: Screen, keyboard, battery }
  - { label: Strength, value: Mobility }
  - { label: Limit, value: Thermal throttling }
  - { label: Best for, value: Working anywhere }
see_also: [personal-computer, desktop-computer, central-processing-unit, random-access-memory, computer-hardware, operating-system]
related_lessons:
  - { title: "Laptops", url: /learn/intro-hardware/laptops/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Laptop
---

**A laptop** is a complete, portable [personal computer](/reference/personal-computer/)
with screen, keyboard, and battery built into one folding shell.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A side cutaway of an open laptop clamshell: the upper half is the display panel, and the thin lower half packs the keyboard on top of the motherboard, a large flat battery, and a small heat pipe and fan — the cramped cooling that forces the CPU to throttle under sustained load." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <path d="M110 24 L250 24 L250 96 L110 108 Z"/>
    <path d="M122 34 L240 34 L240 92 L122 100 Z" fill="currentColor" fill-opacity="0.07"/>
    <path d="M110 108 L400 108 L392 140 L102 140 Z" fill="currentColor" fill-opacity="0.05"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1">
    <rect x="118" y="112" width="180" height="8" rx="1" fill="currentColor" fill-opacity="0.14"/>
    <rect x="120" y="126" width="150" height="9" rx="1" fill="currentColor" fill-opacity="0.1"/>
    <rect x="285" y="124" width="95" height="11" rx="1" fill="currentColor" fill-opacity="0.12"/>
    <rect x="300" y="112" width="26" height="9" rx="1" fill="currentColor" fill-opacity="0.16"/>
    <path d="M330 116 h34" stroke-dasharray="2 2"/>
    <circle cx="372" cy="116" r="6"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="180" y="66" text-anchor="middle" font-size="8" fill-opacity="0.8">display</text>
    <text x="160" y="118" text-anchor="middle" font-size="7">keyboard</text>
    <text x="185" y="133" text-anchor="middle" font-size="7">board</text>
    <text x="333" y="133" text-anchor="middle" font-size="7">battery</text>
    <text x="313" y="119" text-anchor="middle" font-size="6">CPU</text>
    <text x="372" y="106" text-anchor="middle" font-size="6">fan</text>
  </g>
  <text x="230" y="165" fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5" fill-opacity="0.85">everything folds into one thin battery-powered shell &#8212; little room to cool or expand</text>
</svg>
<figcaption>A laptop packs the whole computer into a folding shell: display in the lid, and beneath the keyboard a motherboard, a large flat battery, and a slim heat pipe and fan. The tight space limits cooling and expansion — the price of portability.</figcaption>
</figure>

## Overview

A laptop runs the same full [operating system](/reference/operating-system/) and
the same [CPU](/reference/central-processing-unit/),
[RAM](/reference/random-access-memory/), and
[storage](/reference/data-storage/) building blocks as a desktop, just packed into
a thin, battery-powered case. The packaging costs something: less room to expand,
and in thin models the cooling can't keep up, so the CPU slows itself down under
load — thermal throttling — capping sustained performance.

What it buys in return is self-containment. Screen, keyboard, trackpad, battery,
and wireless are all built in, so a laptop needs nothing but itself to be a working
computer — the reason it has become the default machine for most people.

## Trade-offs

Against the [desktop](/reference/desktop-computer/), the laptop offers one decisive thing — it goes where you go:

| Factor | Laptop | Desktop |
|--------|--------|---------|
| Portability | Goes anywhere | Stays put |
| Sustained performance | Throttles when thin | Full speed |
| Expandability | Minimal | Extensive |
| Power source | Battery or wall | Wall only |
| Everything built in | Yes | Separate parts |

For most developers that mobility outweighs the lost headroom, which is why the laptop is the default development machine.

## Where it fits

A laptop is the natural field machine for GopherTrunk: SDR capture work runs fine on one, and it can go to the antenna, the car, or the site while the heavy replay and wideband DSP wait for a desktop bench. Phones and tablets, by contrast, are screens to view the data on rather than machines to capture it.

## Sources

[^wiki]: [Laptop](https://en.wikipedia.org/wiki/Laptop) — Wikipedia, on the portable all-in-one personal computer and its trade-offs.
