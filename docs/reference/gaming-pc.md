---
slug: gaming-pc
title: Gaming PC
entry_type: hardware
category: hw-personal-computers
description: A gaming PC is a personal computer built and tuned to run video games well, centered on a powerful graphics card, a fast multi-core CPU, ample fast memory, and cooling to sustain high frame rates.
keywords: gaming PC, gaming rig, GPU, graphics card, high refresh, overclocking, RGB build, frame rate
aka: [Gaming rig]
infobox:
  - { label: Type, value: Personal computer }
  - { label: Built for, value: Running games well }
  - { label: Centerpiece, value: Discrete GPU }
  - { label: Also wants, value: Fast CPU, RAM, SSD, cooling }
see_also: [personal-computer, desktop-computer, workstation, graphics-processing-unit, central-processing-unit, build-vs-buy]
related_lessons:
  - { title: "Choosing a dev machine", url: /learn/intro-hardware/choosing-a-dev-machine/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Gaming_computer
---

A **gaming PC** is a [personal computer](/reference/personal-computer/) built and tuned to run video games well — usually a [desktop computer](/reference/desktop-computer/) centered on a powerful graphics card and a fast processor.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A gaming PC built around its graphics card: a large GPU sits at the center, fed by a fast CPU, quick RAM, an SSD, and strong cooling, and it renders a high frame rate that streams out to a high-refresh-rate monitor." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.5">
    <rect x="120" y="58" width="120" height="64" rx="4" fill="currentColor" fill-opacity="0.16"/>
  </g>
  <text x="180" y="86" fill="currentColor" stroke="none" text-anchor="middle" font-size="10" font-weight="600">GPU</text>
  <text x="180" y="102" fill="currentColor" stroke="none" text-anchor="middle" font-size="7" fill-opacity="0.85">graphics card</text>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="30" y="24" width="60" height="20" rx="2" fill="currentColor" fill-opacity="0.08"/>
    <rect x="30" y="54" width="60" height="20" rx="2" fill="currentColor" fill-opacity="0.08"/>
    <rect x="30" y="84" width="60" height="20" rx="2" fill="currentColor" fill-opacity="0.08"/>
    <rect x="30" y="114" width="60" height="20" rx="2" fill="currentColor" fill-opacity="0.08"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" text-anchor="middle">
    <text x="60" y="38">CPU</text>
    <text x="60" y="68">RAM</text>
    <text x="60" y="98">SSD</text>
    <text x="60" y="128">cooling</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1" stroke-dasharray="3 3">
    <path d="M90 34 C110 48 110 62 120 68"/>
    <path d="M90 64 L120 78"/>
    <path d="M90 94 L120 96"/>
    <path d="M90 124 C110 112 110 104 120 100"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <path d="M240 90 H320" stroke-dasharray="0"/>
    <path d="M312 84 L322 90 L312 96"/>
    <rect x="326" y="46" width="110" height="76" rx="4"/>
    <rect x="334" y="54" width="94" height="60" fill="currentColor" fill-opacity="0.07"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="280" y="82" font-size="7.5">frames/s</text>
    <text x="381" y="88" font-size="8" fill-opacity="0.85">high-refresh</text>
    <text x="381" y="100" font-size="8" fill-opacity="0.85">monitor</text>
  </g>
</svg>
<figcaption>A gaming PC is organized around one part: the discrete GPU renders the 3D scene, while a fast CPU, quick RAM, an SSD, and strong cooling keep it fed so it can push a high frame rate to a high-refresh monitor.</figcaption>
</figure>

## Overview

The defining part is the discrete [GPU](/reference/graphics-processing-unit/), which renders the game's 3D scenes at high frame rates and resolutions. Around it a gaming PC pairs a fast multi-core [CPU](/reference/central-processing-unit/), plenty of quick [RAM](/reference/random-access-memory/), a [solid-state drive](/reference/solid-state-drive/) for fast loading, and enough [cooling](/reference/cooling-and-thermals/) to hold those parts at full speed without throttling.

Enthusiasts often overclock the CPU or GPU for extra frames and choose a [case](/reference/computer-case/) with strong airflow (and, frequently, RGB lighting). Because the priorities are clear, a gaming PC is a common first project for people who [build their own](/reference/build-vs-buy/) — the GPU dictates most of the budget and the rest is sized to keep it busy.

## Priorities

Where the budget goes reflects what actually limits frame rate:

| Part | Role in games | Priority |
|------|---------------|----------|
| GPU | Renders every frame | Highest |
| CPU | Game logic, feeds the GPU | High |
| RAM | Holds the working set | Medium |
| SSD | Level and asset loading | Medium |
| Cooling | Sustains clocks under load | Medium |

## Where it fits

A gaming PC overlaps heavily with the enthusiast desktop, and the same GPU-heavy hardware also handles GPU compute, machine learning, and video work, which blurs the line with a [workstation](/reference/workstation/) — the difference is mostly tuning, ECC memory, and certification rather than raw parts. For GopherTrunk a gaming PC's GPU and CPU headroom make it a capable bench for replaying captures and crunching many channels of DSP at once.

## Sources

[^wiki]: [Gaming computer](https://en.wikipedia.org/wiki/Gaming_computer) — Wikipedia, on PCs built and optimized for gaming.
