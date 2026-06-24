---
slug: power-supply-unit
title: Power supply unit (PSU)
entry_type: hardware
category: hw-foundations
description: A power supply unit converts AC mains electricity into the regulated low-voltage DC rails a computer's components need, and is rated by wattage and conversion efficiency.
keywords: power supply unit, PSU, SMPS, switched-mode power supply, voltage rails, wattage, efficiency, 80 PLUS
aka: [PSU, Power supply]
infobox:
  - { label: Type, value: AC-to-DC converter }
  - { label: Outputs, value: "+12 V, +5 V, +3.3 V rails" }
  - { label: Rated by, value: Wattage & efficiency }
  - { label: Tech, value: Switched-mode (SMPS) }
see_also: [motherboard, cooling-and-thermals, central-processing-unit, graphics-processing-unit, computer-hardware]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_supply_unit_(computer)
---

A **power supply unit** (**PSU**) converts alternating-current mains electricity into the regulated low-voltage direct current that a computer's components run on.[^wiki]

## Overview

Most PCs use a *switched-mode* power supply (SMPS), which is small and efficient compared with older linear designs. It delivers several DC *rails* — typically +12 V, +5 V, and +3.3 V — to the [motherboard](/reference/motherboard/), drives, and add-in cards through standardized connectors. A unit is rated by total **wattage** (how much it can deliver) and by conversion **efficiency** (how little it wastes as heat), with schemes like 80 PLUS certifying the latter.

## Where it fits

The PSU sizes the whole machine: a power-hungry [GPU](/reference/graphics-processing-unit/) and [CPU](/reference/central-processing-unit/) demand more headroom, and wasted power becomes heat the [cooling](/reference/cooling-and-thermals/) system must remove. Small computers skip a full PSU — a [Raspberry Pi](/reference/raspberry-pi/) running a GopherTrunk capture node takes regulated 5 V straight from a USB adapter — but the job is identical: turn wall power into the clean, steady rails the chips expect.

## Sources

[^wiki]: [Power supply unit (computer)](https://en.wikipedia.org/wiki/Power_supply_unit_(computer)) — Wikipedia, on PC power supplies, rails, and efficiency.
