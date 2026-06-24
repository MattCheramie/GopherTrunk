---
slug: cooling-and-thermals
title: Cooling & thermals
entry_type: concept
category: hw-foundations
description: Cooling is how a computer moves heat away from its chips to keep them within safe temperatures; thermal limits cap sustained performance through throttling when heat builds up.
keywords: cooling, thermals, heat sink, thermal throttling, TDP, fan, liquid cooling, heat dissipation
infobox:
  - { label: Goal, value: Keep chips in safe range }
  - { label: Methods, value: Heat sink, fan, liquid }
  - { label: Limit, value: TDP / thermal throttling }
  - { label: Heat source, value: Switching transistors }
see_also: [central-processing-unit, graphics-processing-unit, power-supply-unit, clock-speed, transistor, moores-law]
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_cooling
---

**Cooling** is how a computer carries heat away from its chips, and **thermals** are the temperature limits that ultimately cap how hard those chips can run.[^wiki]

## Overview

Every switching [transistor](/reference/transistor/) dissipates a little energy as heat, and a modern processor has billions of them. That heat is moved off the die by a *heat sink* (a finned metal block), often with a fan forcing air across it, or by liquid loops in high-end systems. A chip's *thermal design power* (TDP) states how much heat the cooling must remove. When a part gets too hot, it *throttles* — lowering [clock speed](/reference/clock-speed/) to cut heat and stay safe.

## Where it fits

Cooling sets the ceiling on sustained performance: a [CPU](/reference/central-processing-unit/) or [GPU](/reference/graphics-processing-unit/) can boost briefly but must back off once heat accumulates. Most of the power a [PSU](/reference/power-supply-unit/) delivers ends up as heat the cooling system has to shed. For an unattended GopherTrunk capture node baking in the sun by an antenna, passive heat sinks and good airflow keep a fanless [Raspberry Pi](/reference/raspberry-pi/) from throttling mid-decode — thermals are a real-world constraint, not an afterthought.

## Sources

[^wiki]: [Computer cooling](https://en.wikipedia.org/wiki/Computer_cooling) — Wikipedia, on heat sinks, fans, liquid cooling, and thermal limits.
