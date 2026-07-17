---
slug: power-supply-unit
title: Power supply unit (PSU)
entry_type: hardware
category: hw-foundations
description: A power supply unit converts AC mains electricity into the regulated low-voltage DC rails a computer's components need, and is rated by wattage and conversion efficiency.
keywords: power supply unit, PSU, SMPS, switched-mode power supply, voltage rails, 12V 5V 3.3V, wattage, efficiency, 80 PLUS
aka: [PSU, Power supply]
infobox:
  - { label: Type, value: AC-to-DC converter }
  - { label: Outputs, value: "+12 V, +5 V, +3.3 V rails" }
  - { label: Rated by, value: Wattage & efficiency }
  - { label: Tech, value: Switched-mode (SMPS) }
  - { label: Efficiency mark, value: 80 PLUS }
see_also: [motherboard, cooling-and-thermals, central-processing-unit, graphics-processing-unit, computer-hardware]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_supply_unit_(computer)
---

A **power supply unit** (**PSU**) converts alternating-current mains electricity into the regulated low-voltage direct current that a computer's components run on.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A power supply takes AC mains on the left, rectifies and switches it, and outputs three regulated DC rails on the right: plus twelve volts feeding the CPU and GPU, plus five volts feeding drives and USB, and plus three point three volts feeding the chipset and memory." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="120" y="52" width="90" height="56" rx="4" fill="currentColor" fill-opacity="0.12"/>
    <path d="M18 80 q10 -14 20 0 t20 0 t20 0 t20 0"/>
    <line x1="210" y1="66" x2="300" y2="66"/>
    <line x1="210" y1="80" x2="300" y2="80"/>
    <line x1="210" y1="94" x2="300" y2="94"/>
  </g>
  <g fill="currentColor" stroke="none">
    <text x="18" y="100" font-size="8">AC mains</text>
    <text x="165" y="76" font-size="8.5" text-anchor="middle" font-weight="600">SMPS</text>
    <text x="165" y="88" font-size="7" text-anchor="middle" fill-opacity="0.85">rectify +</text>
    <text x="165" y="97" font-size="7" text-anchor="middle" fill-opacity="0.85">switch</text>
    <text x="306" y="69" font-size="8.5">+12 V</text>
    <text x="360" y="69" font-size="7.5" fill-opacity="0.85">CPU, GPU</text>
    <text x="306" y="83" font-size="8.5">+5 V</text>
    <text x="360" y="83" font-size="7.5" fill-opacity="0.85">drives, USB</text>
    <text x="306" y="97" font-size="8.5">+3.3 V</text>
    <text x="360" y="97" font-size="7.5" fill-opacity="0.85">chipset, RAM</text>
    <text x="230" y="135" font-size="8" text-anchor="middle" fill-opacity="0.9">rated by total wattage · efficiency wasted as heat (80 PLUS marks the good ones)</text>
  </g>
</svg>
<figcaption>A switched-mode supply rectifies and chops the AC mains, then delivers three steady DC rails — +12 V, +5 V, and +3.3 V — each feeding the parts of the machine that expect that voltage.</figcaption>
</figure>

## Overview

Most PCs use a *switched-mode* power supply (SMPS), which rapidly switches the incoming power at high frequency to regulate its output; this makes it far smaller and more efficient than the older linear designs it replaced. It first rectifies the AC mains to a high DC voltage, then switches and steps it down to the low voltages the machine needs.

The output is delivered as several DC *rails* — typically +12 V, +5 V, and +3.3 V — routed to the [motherboard](/reference/motherboard/), drives, and add-in cards through standardized connectors. A unit is rated by total **wattage** (how much power it can deliver at once) and by conversion **efficiency** (how little it wastes as heat), with certification schemes such as 80 PLUS grading the latter.

## The rails

Different components expect different voltages, so the PSU provides each on its own rail:

| Rail | Typical loads | Notes |
|------|---------------|-------|
| +12 V | CPU, GPU, fans, motors | Carries most of the power in a modern PC |
| +5 V | Drives, USB, some logic | The traditional workhorse rail |
| +3.3 V | Chipset, memory, low-voltage logic | Lowest of the three |

Modern designs put nearly all the power on the +12 V rail and derive the lower voltages from it on the board, because high-current parts like the CPU and GPU dominate the budget.

## Where it fits

The PSU sizes the whole machine: a power-hungry [GPU](/reference/graphics-processing-unit/) and [CPU](/reference/central-processing-unit/) demand more wattage headroom, and whatever power is wasted becomes heat the [cooling](/reference/cooling-and-thermals/) system must remove. Small computers skip a full PSU — a [Raspberry Pi](/reference/raspberry-pi/) running a GopherTrunk capture node takes regulated 5 V straight from a USB adapter — but the job is identical: turn wall power into the clean, steady rails the chips expect, and a noisy or undersized supply can quietly corrupt an SDR capture.

## Sources

[^wiki]: [Power supply unit (computer)](https://en.wikipedia.org/wiki/Power_supply_unit_(computer)) — Wikipedia, on PC power supplies, switched-mode design, voltage rails, and efficiency ratings.
