---
slug: desktop-computer
title: Desktop computer
entry_type: hardware
category: hw-personal-computers
description: A desktop computer is a stationary personal computer that delivers the most performance per dollar and is the easiest to upgrade and expand.
keywords: desktop computer, desktop PC, tower, workstation, upgradeable computer, stationary computer, thermal headroom
aka: [Desktop computer, Desktop PC, Desktop]
infobox:
  - { label: Type, value: Stationary personal computer }
  - { label: Strength, value: Most power per dollar }
  - { label: Upgradeable, value: Yes, easily }
  - { label: Portability, value: None }
  - { label: Best for, value: Sustained heavy work }
see_also: [personal-computer, laptop, central-processing-unit, random-access-memory, computer-case, computer-hardware, data-storage]
related_lessons:
  - { title: "Desktop computers", url: /learn/intro-hardware/desktop-computers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Desktop_computer
---

**A desktop computer** is a stationary [personal computer](/reference/personal-computer/)
that trades portability for the most performance per dollar.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A desktop computer as a set of separate boxes: a roomy tower on the left with swappable CPU, RAM, GPU, and drive modules, wired to a monitor on the right and a keyboard and mouse in front — the modular layout that makes desktops easy to upgrade." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <rect x="40" y="30" width="90" height="120" rx="4"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1">
    <rect x="52" y="42" width="66" height="16" rx="1" fill="currentColor" fill-opacity="0.14"/>
    <rect x="52" y="66" width="66" height="12" rx="1" fill="currentColor" fill-opacity="0.1"/>
    <rect x="52" y="86" width="66" height="22" rx="1" fill="currentColor" fill-opacity="0.1"/>
    <rect x="52" y="116" width="66" height="24" rx="1" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7" text-anchor="middle">
    <text x="85" y="53">CPU</text>
    <text x="85" y="75">RAM</text>
    <text x="85" y="100">GPU</text>
    <text x="85" y="131">drive</text>
    <text x="85" y="166" font-size="7.5">tower (swappable)</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <rect x="270" y="34" width="150" height="94" rx="4"/>
    <rect x="280" y="44" width="130" height="74" fill="currentColor" fill-opacity="0.07"/>
    <path d="M330 128 V144 M310 144 H370"/>
    <rect x="270" y="152" width="80" height="14" rx="2"/>
    <ellipse cx="382" cy="159" rx="9" ry="7"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2" stroke-dasharray="4 3">
    <path d="M130 74 H270"/>
    <path d="M130 120 C180 120 200 158 270 158"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" text-anchor="middle">
    <text x="345" y="88" fill-opacity="0.8">monitor</text>
    <text x="310" y="164" font-size="7">keyboard</text>
    <text x="382" y="146" font-size="7">mouse</text>
  </g>
</svg>
<figcaption>A desktop separates the computer into a roomy tower plus its display and input devices; because the tower runs on wall power and has space to breathe, its CPU, RAM, GPU, and drives are full-size and swap out one at a time.</figcaption>
</figure>

## Overview

Because the [case](/reference/computer-case/) is roomy and runs on wall power, a desktop fits larger, faster
[CPUs](/reference/central-processing-unit/), more
[RAM](/reference/random-access-memory/), and bigger
[storage](/reference/data-storage/) than a comparable laptop — and it stays
cool enough to run them flat out. The same roominess makes it the easiest form to
upgrade and expand: parts swap out individually rather than being soldered down.

That separation into tower, monitor, and input devices is exactly what buys the
headroom. Nothing has to be miniaturized or battery-friendly, so a desktop reaches
performance a laptop of the same price can't sustain — and outlives it, because you
can replace the graphics card or double the memory instead of the whole machine.

## Trade-offs

The catch is in the name: it stays on the desk. The choice between a desktop and a [laptop](/reference/laptop/) comes down to whether the work travels:

| Factor | Desktop | Laptop |
|--------|---------|--------|
| Performance per dollar | Higher | Lower |
| Sustained load | Excellent | Throttles when thin |
| Upgradeable | Easily | Minimal |
| Portability | None | Goes anywhere |

For sustained heavy work — compiling large projects, crunching data, running long SDR captures — a desktop's thermal headroom and upgrade path are hard to beat. When the work needs to travel, that argument flips and the laptop wins.

## Where it fits

A desktop is the natural GopherTrunk bench machine: it can run wideband captures and many channels of DSP for hours without throttling, and its expansion slots leave room for extra storage or a capture card. Pair it with a wide [monitor](/reference/computer-monitor/) for the waterfall and it stays put while a [laptop](/reference/laptop/) handles fieldwork.

## Sources

[^wiki]: [Desktop computer](https://en.wikipedia.org/wiki/Desktop_computer) — Wikipedia, on the stationary personal computer and its expandability.
