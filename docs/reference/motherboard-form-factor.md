---
slug: motherboard-form-factor
title: Motherboard form factor
entry_type: concept
category: hw-personal-computers
description: A motherboard form factor is a standardized specification for a motherboard's size, shape, mounting holes, and connector placement, so boards, cases, and power supplies from different makers fit together.
keywords: form factor, ATX, Micro-ATX, Mini-ITX, E-ATX, motherboard size, mounting holes, expansion slots, rear I/O
aka: [Form factor]
infobox:
  - { label: Type, value: Hardware standard }
  - { label: Defines, value: Size, mounts, port layout }
  - { label: Common, value: ATX, Micro-ATX, Mini-ITX }
  - { label: Ensures, value: Case / PSU / board fit }
see_also: [motherboard, computer-case, desktop-computer, power-supply-unit, pci-express]
related_lessons:
  - { title: "Desktop computers", url: /learn/intro-hardware/desktop-computers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_form_factor
---

A **motherboard form factor** is a standardized specification for a [motherboard](/reference/motherboard/)'s physical size, shape, mounting-hole positions, and connector placement, so that boards, cases, and power supplies from different makers fit and work together.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="Three common motherboard form factors drawn to scale from a shared corner: full-size ATX at 305 by 244 millimeters, smaller Micro-ATX at 244 by 244, and tiny Mini-ITX at 170 by 170, each with corner mounting holes, showing how smaller boards trade expansion slots for a smaller footprint." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none">
    <rect x="50" y="34" width="128" height="102" stroke-width="1.5" fill="currentColor" fill-opacity="0.04"/>
    <rect x="50" y="34" width="102" height="102" stroke-width="1.4" fill="currentColor" fill-opacity="0.06"/>
    <rect x="50" y="34" width="71" height="71" stroke-width="1.4" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1">
    <circle cx="56" cy="40" r="2.4"/>
    <circle cx="172" cy="40" r="2.4"/>
    <circle cx="56" cy="130" r="2.4"/>
    <circle cx="172" cy="130" r="2.4"/>
    <circle cx="115" cy="99" r="2.4"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="126" y="126" font-weight="600">ATX</text>
    <text x="104" y="98" font-weight="600" font-size="7.5">mATX</text>
    <text x="76" y="66" font-weight="600" font-size="7">ITX</text>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="250" y="46" font-weight="600">ATX</text>
    <text x="250" y="58" font-size="7">305 &#215; 244 mm · most slots</text>
    <text x="250" y="88" font-weight="600">Micro-ATX</text>
    <text x="250" y="100" font-size="7">244 &#215; 244 mm · compact</text>
    <text x="250" y="130" font-weight="600">Mini-ITX</text>
    <text x="250" y="142" font-size="7">170 &#215; 170 mm · one slot, SFF</text>
    <text x="250" y="176" font-size="7" fill-opacity="0.85">bigger board &#8594; more PCIe &amp; RAM slots</text>
    <text x="250" y="188" font-size="7" fill-opacity="0.85">smaller board &#8594; smaller case</text>
  </g>
</svg>
<figcaption>The common desktop form factors nest inside one another: full ATX has the most expansion, Micro-ATX trims it for a compact build, and Mini-ITX shrinks to a single slot for small-form-factor machines — all sharing the same mounting-hole and port conventions so cases and power supplies fit.</figcaption>
</figure>

## Overview

By fixing dimensions and the location of mounting holes, the rear I/O panel, expansion slots, and the power connector, a form factor turns PC parts into interchangeable building blocks. Because the standards share conventions, a case advertised for ATX also accepts the smaller Micro-ATX and Mini-ITX boards that mount on the same holes.

The common desktop standards descend from Intel's ATX. A given [computer case](/reference/computer-case/) lists which form factors it accepts, and a [power supply unit](/reference/power-supply-unit/) follows matching standards so the connectors line up regardless of who made each part.

## Common sizes

The size you pick caps how much you can add later:

| Form factor | Size (mm) | PCIe slots (typ.) | RAM slots | Best for |
|-------------|-----------|-------------------|-----------|----------|
| E-ATX | 305 × 330 | 4–7 | 8 | High-end workstations |
| ATX | 305 × 244 | 4–7 | 4 | Standard desktop |
| Micro-ATX | 244 × 244 | 1–4 | 2–4 | Compact builds |
| Mini-ITX | 170 × 170 | 1 | 2 | Small-form-factor |

## Where it fits

The form factor is the first decision in a desktop build because it caps everything downstream: how many [PCI Express](/reference/pci-express/) and [RAM](/reference/random-access-memory/) slots you get, how much room there is for cooling, and how small the case can be. Bigger boards mean more expansion and easier airflow; smaller ones save space at the cost of slots. Pre-built [all-in-one computers](/reference/all-in-one-computer/) and [mini PCs](/reference/mini-pc/) often use proprietary boards, trading this standardization for compactness.

## Sources

[^wiki]: [Computer form factor](https://en.wikipedia.org/wiki/Computer_form_factor) — Wikipedia, on motherboard form-factor standards.
