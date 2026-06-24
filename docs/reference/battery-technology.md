---
slug: battery-technology
title: Battery technology
entry_type: concept
category: hw-mobile
description: Battery technology covers the rechargeable chemistries — chiefly lithium-ion and lithium-polymer — that store the energy powering phones, wearables, and portable devices, defined by capacity, energy density, and cycle life.
keywords: battery, lithium-ion, lithium-polymer, energy density, mAh, charge cycles, rechargeable, capacity, fast charging
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

## Overview

A battery is rated by capacity (milliamp-hours, mAh, or watt-hours, Wh) and judged on *energy density* — how much energy fits in a given size and weight — and *cycle life*, how many charge/discharge cycles it survives before fading. Lithium-ion dominates because it packs high energy density into a light, rechargeable cell. Lithium-polymer variants trade in a flexible pouch that can be shaped to fit thin phones and curved wearables. Charging is managed by control circuitry to balance speed, heat, and long-term wear.

## Where it fits

The battery is the hard constraint that shapes mobile design. A [mobile operating system](/reference/mobile-operating-system/) spends much of its effort stretching a charge; a [smartwatch](/reference/smartwatch/) or [wearable](/reference/wearable-computer/) lives or dies by how little its [SoC](/reference/system-on-a-chip/) and screen draw. The same power budget explains why a [smartphone](/reference/smartphone/) is a poor host for a continuous SDR decode load like GopherTrunk — sustained CPU and radio use drains a phone fast, where a mains-powered capture node runs indefinitely.

## Sources

[^wiki]: [Lithium-ion battery](https://en.wikipedia.org/wiki/Lithium-ion_battery) — Wikipedia, on the chemistry and metrics of modern rechargeable batteries.
