---
slug: battery-technology
title: Battery technology
entry_type: concept
category: hw-mobile
description: Battery technology covers the rechargeable chemistries — chiefly lithium-ion and lithium-polymer — that store the energy powering phones, wearables, and portable devices, defined by capacity, energy density, and cycle life.
keywords: battery, lithium-ion, lithium-polymer, energy density, mAh, charge cycles, rechargeable, capacity, fast charging, discharge curve
infobox:
  - { label: Type, value: Energy storage }
  - { label: Common chemistry, value: Lithium-ion / Li-polymer }
  - { label: Rated in, value: mAh / Wh }
  - { label: Key metrics, value: Energy density, cycle life }
see_also: [smartphone, smartwatch, wearable-computer, mobile-operating-system, e-reader, system-on-a-chip]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Lithium-ion_battery
---

**Battery technology** is the set of rechargeable chemistries — most often lithium-ion and lithium-polymer — that store the electrical energy powering phones, wearables, and other portable devices.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A lithium-ion discharge curve. Voltage starts near 4.2 volts when full, holds a long, nearly flat plateau around 3.7 volts through most of the capacity, then drops steeply toward the 3.0-volt cutoff as the cell empties. This flat plateau is why a phone can report a stable voltage across most of its charge." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="48" y1="20" x2="48" y2="132"/>
    <line x1="48" y1="132" x2="430" y2="132"/>
  </g>
  <path d="M60 34 C90 40 100 52 130 56 C200 64 300 72 360 84 C390 92 405 108 418 126" stroke="currentColor" fill="none" stroke-width="1.8"/>
  <g stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3" fill="none">
    <line x1="48" y1="60" x2="418" y2="60"/>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5">
    <text x="20" y="37">4.2V</text>
    <text x="20" y="63">3.7V</text>
    <text x="20" y="135">3.0V</text>
    <text x="60" y="148">full</text>
    <text x="392" y="148">empty</text>
    <text x="210" y="148" text-anchor="middle">state of charge &#8594;</text>
    <text x="200" y="54" font-size="8">flat plateau &#8776; most of the usable charge</text>
    <text x="392" y="104" font-size="8" text-anchor="end">steep &#8595;</text>
  </g>
</svg>
<figcaption>A lithium-ion cell holds a long, nearly flat voltage plateau around 3.7 V through most of its charge, then falls sharply near empty — the shape that makes state-of-charge estimation from voltage alone tricky at the ends.</figcaption>
</figure>

## Overview

A battery is rated by capacity (milliamp-hours, mAh, or watt-hours, Wh) and judged on *energy density* — how much energy fits in a given size and weight — and *cycle life*, how many charge/discharge cycles it survives before fading. Lithium-ion dominates because it packs high energy density into a light, rechargeable cell with low self-discharge and no memory effect.

Lithium-polymer variants trade the rigid metal can for a flexible pouch that can be shaped to fit thin phones and curved wearables. Charging is managed by control circuitry that balances speed, heat, and long-term wear, since fast charging and high temperatures both accelerate aging. The flat discharge curve above means most of a cell's energy is delivered at a stable voltage, with sharp knees only when nearly full or nearly empty.

## Li-ion vs Li-polymer

The two lithium chemistries share the same electrochemistry but differ in packaging and how that shapes a device:

| Property | Lithium-ion | Lithium-polymer |
|----------|-------------|-----------------|
| Package | Rigid cylindrical/prismatic can | Flexible pouch |
| Energy density | Higher per weight | Slightly lower |
| Shape freedom | Fixed forms | Thin, curved, custom |
| Cost | Lower | Higher |
| Typical use | Laptops, power tools | Phones, wearables, drones |

The pouch's shape freedom is why nearly every slim phone and curved smartwatch uses Li-polymer despite the cost.

## Where it fits

The battery is the hard constraint that shapes mobile design. A [mobile operating system](/reference/mobile-operating-system/) spends much of its effort stretching a charge; a [smartwatch](/reference/smartwatch/) or [wearable](/reference/wearable-computer/) lives or dies by how little its [SoC](/reference/system-on-a-chip/) and screen draw. The same power budget explains why a [smartphone](/reference/smartphone/) is a poor host for a continuous SDR decode load like GopherTrunk — sustained CPU and radio use drains a phone fast, where a mains-powered capture node runs indefinitely.

## Sources

[^wiki]: [Lithium-ion battery](https://en.wikipedia.org/wiki/Lithium-ion_battery) — Wikipedia, on the chemistry and metrics of modern rechargeable batteries.
