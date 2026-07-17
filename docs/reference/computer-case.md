---
slug: computer-case
title: Computer case
entry_type: hardware
category: hw-personal-computers
description: A computer case is the enclosure that houses and protects a desktop PC's components, providing mounting points, airflow, and cable routing for the motherboard, power supply, drives, and cards.
keywords: computer case, PC case, chassis, tower, ATX case, airflow, form factor, small form factor, cable management
aka: [PC case, Chassis, Tower]
infobox:
  - { label: Type, value: PC enclosure }
  - { label: Houses, value: Motherboard, PSU, drives, cards }
  - { label: Sizes, value: Full / mid / mini tower, SFF }
  - { label: Job, value: Mounting + airflow + cabling }
  - { label: Matches, value: Motherboard form factor }
see_also: [desktop-computer, motherboard, motherboard-form-factor, power-supply-unit, cooling-and-thermals]
related_lessons:
  - { title: "Desktop computers", url: /learn/intro-hardware/desktop-computers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_case
---

A **computer case** is the enclosure that houses and protects a [desktop computer](/reference/desktop-computer/)'s components, giving each part a place to mount and a path for air and cables.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A cutaway of a tower PC case showing where the parts mount: the motherboard on the right wall, the power supply in a bottom bay, drives in a front bay, and a graphics card in a horizontal slot, with a front intake fan and a rear exhaust fan setting the airflow front to back." xmlns="http://www.w3.org/2000/svg">
  <rect x="70" y="20" width="320" height="170" rx="4" stroke="currentColor" fill="none" stroke-width="1.5"/>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="210" y="34" width="130" height="96" fill="currentColor" fill-opacity="0.07"/>
    <rect x="230" y="52" width="34" height="34" fill="currentColor" fill-opacity="0.14"/>
    <rect x="210" y="140" width="130" height="34" fill="currentColor" fill-opacity="0.1"/>
    <rect x="86" y="120" width="60" height="54" fill="currentColor" fill-opacity="0.1"/>
    <rect x="150" y="150" width="150" height="16" fill="currentColor" fill-opacity="0.14"/>
    <circle cx="98" cy="60" r="18"/>
    <circle cx="362" cy="60" r="14"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1" stroke-dasharray="4 3">
    <path d="M118 60 H230"/>
    <path d="M300 60 H346"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" text-anchor="middle">
    <text x="247" y="72">CPU</text>
    <text x="290" y="44" font-size="7">motherboard</text>
    <text x="275" y="160">PSU</text>
    <text x="116" y="150">drive</text>
    <text x="116" y="162">bay</text>
    <text x="225" y="161" font-size="7">graphics card</text>
    <text x="98" y="62" font-size="6.5">intake</text>
    <text x="362" y="62" font-size="6.5">exhaust</text>
    <text x="174" y="52" font-size="7" fill-opacity="0.85">airflow &#8594;</text>
  </g>
</svg>
<figcaption>The case gives every part a home and a path for air: the motherboard and CPU on one wall, the power supply in a bottom bay, drives up front, and a graphics card in a slot — with a front intake and rear exhaust fan pulling cool air across the hot parts.</figcaption>
</figure>

## Overview

The case holds the [motherboard](/reference/motherboard/), the [power supply unit](/reference/power-supply-unit/), drives, and expansion cards, with standoffs and brackets sized to standard layouts. Its job beyond mounting is *thermal*: fans and vents move air across the hot parts so [cooling](/reference/cooling-and-thermals/) keeps the CPU and GPU from throttling. Good cases also route cables out of the airflow, which keeps the machine both tidy and cool.

Cases come in a range of sizes, and each is built to accept a particular [motherboard form factor](/reference/motherboard-form-factor/) such as ATX or Mini-ITX. Bigger cases take bigger boards, more drives, and larger coolers; smaller ones save desk space but tighten every clearance.

## Sizes

Case size is chosen to match the board and the amount of expansion you need:

| Case size | Board it fits | Drive / card room | Best for |
|-----------|---------------|-------------------|----------|
| Full tower | E-ATX, ATX | Most | Workstations, big cooling |
| Mid tower | ATX, Micro-ATX | Plenty | Typical desktop / gaming |
| Mini tower | Micro-ATX | Some | Compact desktop |
| Small form factor | Mini-ITX | Minimal | Living room, small desk |

## Where it fits

The case is the one part that does no computing yet shapes the whole build: it decides which boards fit, how many drives and cards you can add, how quiet the machine runs, and how it looks on the desk. A roomy tower eases upgrades and airflow; a small case saves space at the cost of expansion and cooling headroom. A pre-built [all-in-one computer](/reference/all-in-one-computer/) or [mini PC](/reference/mini-pc/) hides this choice inside a fixed enclosure — fine for a GopherTrunk host that never needs opening, less so for a bench you plan to grow.

## Sources

[^wiki]: [Computer case](https://en.wikipedia.org/wiki/Computer_case) — Wikipedia, on PC cases and their role.
