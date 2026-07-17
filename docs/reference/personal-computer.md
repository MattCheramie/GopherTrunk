---
slug: personal-computer
title: Personal computer
entry_type: hardware
category: hw-personal-computers
description: A personal computer is a general-purpose computer sized and priced for a single user, built from a CPU, RAM, storage, and I/O and running a full operating system.
keywords: personal computer, PC, development machine, desktop, laptop, general-purpose computer, building blocks
aka: [Personal computer, PC]
infobox:
  - { label: Type, value: General-purpose computer }
  - { label: Users, value: One at a time }
  - { label: Main forms, value: Desktop, laptop }
  - { label: Building blocks, value: CPU, RAM, storage, I/O }
  - { label: Typical role, value: Development machine }
see_also: [desktop-computer, laptop, central-processing-unit, random-access-memory, operating-system, computer-hardware]
related_lessons:
  - { title: "Choosing a development machine", url: /learn/intro-hardware/choosing-a-dev-machine/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Personal_computer
---

**A personal computer** is a general-purpose computer sized and priced for one
user, in two main forms — the [desktop](/reference/desktop-computer/) and the
[laptop](/reference/laptop/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A personal computer built from four building blocks — CPU, RAM, storage, and input/output — all linked by a system bus and running one operating system, then packaged into either of two forms: a desktop tower or a folding laptop." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="70" x2="420" y2="70" stroke="currentColor" stroke-width="1.5"/>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="46" y="30" width="76" height="26" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="140" y="30" width="76" height="26" rx="3" fill="currentColor" fill-opacity="0.1"/>
    <rect x="234" y="30" width="76" height="26" rx="3" fill="currentColor" fill-opacity="0.1"/>
    <rect x="328" y="30" width="76" height="26" rx="3" fill="currentColor" fill-opacity="0.1"/>
    <path d="M84 56 V70 M178 56 V70 M272 56 V70 M366 56 V70"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8">
    <text x="84" y="47">CPU</text>
    <text x="178" y="47">RAM</text>
    <text x="272" y="47">storage</text>
    <text x="366" y="47">I/O</text>
    <text x="230" y="86" font-size="7" fill-opacity="0.85">system bus &#183; one operating system</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <path d="M160 70 L120 108 M300 70 L340 108" stroke-dasharray="4 3"/>
    <rect x="90" y="112" width="30" height="50" rx="3"/>
    <path d="M300 118 L370 118 L370 150 L292 150 Z" fill="currentColor" fill-opacity="0.06"/>
    <path d="M300 118 L316 106 L386 106 L370 118 Z" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5">
    <text x="105" y="176">desktop</text>
    <text x="338" y="168">laptop</text>
    <text x="230" y="182" font-size="7" fill-opacity="0.85">same blocks, two packages</text>
  </g>
</svg>
<figcaption>Whatever its shape, a personal computer is the same four building blocks — CPU, RAM, storage, and I/O — wired to a system bus under one operating system, then packaged as either a stationary desktop or a portable laptop.</figcaption>
</figure>

## Overview

Whatever the form factor, a personal computer is built from the same four
[building blocks](/reference/computer-hardware/): a
[CPU](/reference/central-processing-unit/), [RAM](/reference/random-access-memory/),
[storage](/reference/data-storage/), and [input/output](/reference/input-output/).
What sets it apart from smaller devices is headroom: a full
[operating system](/reference/operating-system/) and gigabytes of memory, enough
to run essentially the whole programming-language landscape without compromise.

The two main packages simply arrange those blocks differently — a desktop spreads
them across a roomy case for power and upgrades, a laptop folds them into one
portable shell — but the computing model underneath is identical.

## Main forms

The category splits mainly on whether the machine moves:

| Form | Portability | Performance | Upgradeable | Typical use |
|------|-------------|-------------|-------------|-------------|
| Desktop | Stationary | Highest per dollar | Easily | Bench, heavy work |
| Laptop | Goes anywhere | Good, may throttle | Minimal | Everyday, mobile |
| All-in-one | Stationary | Moderate | Limited | Tidy home/office |
| Mini PC | Movable-ish | Moderate | Limited | Always-on node |

## Where it fits

The personal computer is the machine most developers write and test code on — the
"development machine." Because it has the OS, memory, and storage to host
compilers, editors, and full toolchains, SDR software and decoders like
GopherTrunk run comfortably on one. This entry is the umbrella for the
[desktop](/reference/desktop-computer/)-vs-[laptop](/reference/laptop/) category; the two sub-entries cover the trade-offs.

## Sources

[^wiki]: [Personal computer](https://en.wikipedia.org/wiki/Personal_computer) — Wikipedia, on the general-purpose single-user computer and its forms.
