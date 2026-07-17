---
slug: cooling-and-thermals
title: Cooling & thermals
entry_type: concept
category: hw-foundations
description: Cooling is how a computer moves heat away from its chips to keep them within safe temperatures; thermal limits cap sustained performance through throttling when heat builds up.
keywords: cooling, thermals, heat sink, thermal throttling, TDP, fan, liquid cooling, heat dissipation, thermal paste, passive cooling
infobox:
  - { label: Goal, value: Keep chips in safe range }
  - { label: Methods, value: Heat sink, fan, liquid }
  - { label: Limit, value: TDP / thermal throttling }
  - { label: Heat source, value: Switching transistors }
see_also: [central-processing-unit, graphics-processing-unit, power-supply-unit, clock-speed, transistor, raspberry-pi]
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_cooling
---

**Cooling** is how a computer carries heat away from its chips, and **thermals** are the temperature limits that ultimately cap how hard those chips can run.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="The heat path from a chip to the air. Heat leaves the silicon die, crosses a thin layer of thermal paste into the metal base of a heat sink, spreads up into its fins, and a fan blows cooler air through the fins to carry the heat away." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="150" y="128" width="120" height="18" rx="2"/>
    <rect x="150" y="120" width="120" height="8" fill="currentColor" fill-opacity="0.10" stroke-dasharray="2 2"/>
    <rect x="150" y="96" width="120" height="24" rx="2" fill="currentColor" fill-opacity="0.12"/>
    <line x1="162" y1="96" x2="162" y2="52"/>
    <line x1="180" y1="96" x2="180" y2="52"/>
    <line x1="198" y1="96" x2="198" y2="52"/>
    <line x1="216" y1="96" x2="216" y2="52"/>
    <line x1="234" y1="96" x2="234" y2="52"/>
    <line x1="252" y1="96" x2="252" y2="52"/>
    <circle cx="360" cy="74" r="26"/>
    <path d="M360 74 l18 -8 M360 74 l-4 19 M360 74 l-16 -12"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none" stroke-opacity="0.8">
    <path d="M334 66 h-40 m8 -4 l-8 4 l8 4"/>
    <path d="M334 82 h-40 m8 -4 l-8 4 l8 4"/>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none" stroke-opacity="0.6">
    <path d="M170 44 v-14 m-3 6 l3 -6 l3 6"/>
    <path d="M210 44 v-14 m-3 6 l3 -6 l3 6"/>
    <path d="M250 44 v-14 m-3 6 l3 -6 l3 6"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="210" y="141" font-size="8">chip die (heat source)</text>
    <text x="300" y="112" font-size="7.5" fill-opacity="0.85">thermal paste</text>
    <text x="210" y="70" font-size="8">heat-sink fins</text>
    <text x="360" y="112" font-size="8">fan</text>
    <text x="210" y="24" font-size="7.5" fill-opacity="0.85">heat rises out</text>
    <text x="300" y="60" font-size="7.5" fill-opacity="0.85">cool air in</text>
  </g>
</svg>
<figcaption>Heat leaves the silicon die, crosses a thin film of thermal paste into the heat-sink base, spreads up through the fins, and a fan pushes cooler air across them to carry it away — the standard air-cooling path from chip to room.</figcaption>
</figure>

## Overview

Every switching [transistor](/reference/transistor/) dissipates a little energy as heat, and a modern processor has billions of them switching billions of times a second. That heat has to leave the die or the chip cooks itself. It is moved off by a *heat sink* — a finned metal block bonded to the chip through a thin layer of thermal paste — often with a fan forcing air across the fins, or by liquid loops that carry heat to a larger radiator in high-end systems.

A chip's *thermal design power* (TDP), quoted in watts, states how much sustained heat the cooling solution must be able to remove. When a part exceeds its safe temperature anyway, it protects itself by *throttling*: lowering [clock speed](/reference/clock-speed/) (and voltage) to cut heat production until it cools back down.

## Ways to move heat

Cooling methods trade cost, noise, and moving parts against how much heat they can shed:

| Method | How it works | Typical use | Moving parts |
|--------|--------------|-------------|--------------|
| Passive heat sink | Fins radiate/convect to air | Low-power SoCs, fanless PCs | None |
| Active air | Fan forces air over fins | Most desktops and laptops | Fan |
| Liquid loop | Coolant carries heat to a radiator | High-end CPUs and GPUs | Pump + fans |

Passive cooling is silent and reliable but limited; adding a fan or liquid dramatically raises the heat that can be removed at the cost of noise and parts that can fail.

## Where it fits

Cooling sets the ceiling on sustained performance: a [CPU](/reference/central-processing-unit/) or [GPU](/reference/graphics-processing-unit/) can boost briefly but must back off once heat accumulates. Most of the power a [PSU](/reference/power-supply-unit/) delivers ultimately ends up as heat the cooling system has to shed. For an unattended GopherTrunk capture node baking in the sun by an antenna, a passive heat sink and good airflow are what keep a fanless [Raspberry Pi](/reference/raspberry-pi/) from throttling mid-decode — thermals are a real-world constraint, not an afterthought.

## Sources

[^wiki]: [Computer cooling](https://en.wikipedia.org/wiki/Computer_cooling) — Wikipedia, on heat sinks, fans, liquid cooling, TDP, and thermal throttling.
