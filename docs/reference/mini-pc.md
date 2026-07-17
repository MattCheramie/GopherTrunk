---
slug: mini-pc
title: Mini PC
entry_type: hardware
category: hw-personal-computers
description: A mini PC is a small-form-factor desktop computer, often the size of a paperback, that uses low-power laptop-class parts to deliver a full PC in a fraction of the space and power of a tower.
keywords: mini PC, small form factor, NUC, SFF PC, low-power desktop, fanless PC, always-on node
aka: [Mini PC, SFF PC, NUC]
infobox:
  - { label: Type, value: Small-form-factor PC }
  - { label: Size, value: Paperback to lunchbox }
  - { label: Internals, value: Laptop-class CPU/RAM }
  - { label: Power, value: Low (often 10–65 W) }
see_also: [personal-computer, desktop-computer, all-in-one-computer, thin-client, single-board-computer]
related_lessons:
  - { title: "Desktop computers", url: /learn/intro-hardware/desktop-computers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Nettop
---

A **mini PC** is a small-form-factor [desktop computer](/reference/desktop-computer/) — often about the size of a paperback book — that packs a full PC into a fraction of the volume and power of a conventional tower.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A mini PC drawn to scale beside a full tower: a small paperback-sized box holding a laptop-class CPU, compact RAM, and an M.2 SSD, drawing only tens of watts, next to a much larger tower for comparison, with a row of USB, HDMI, and Ethernet ports on the mini PC's back." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="40" y="28" width="70" height="118" rx="3" fill="currentColor" fill-opacity="0.05"/>
  </g>
  <text x="75" y="160" fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5" fill-opacity="0.8">full tower</text>
  <g stroke="currentColor" fill="none" stroke-width="1.5">
    <rect x="180" y="86" width="96" height="60" rx="4" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="0.9">
    <rect x="190" y="98" width="34" height="12" rx="1" fill="currentColor" fill-opacity="0.16"/>
    <rect x="232" y="98" width="34" height="12" rx="1" fill="currentColor" fill-opacity="0.12"/>
    <rect x="190" y="118" width="76" height="14" rx="1" fill="currentColor" fill-opacity="0.12"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="6.5">
    <text x="207" y="107">CPU</text>
    <text x="249" y="107">RAM</text>
    <text x="228" y="128">M.2 SSD</text>
    <text x="228" y="160" font-size="7.5" fill-opacity="0.8">mini PC (~paperback)</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="326" y="42" width="18" height="12" rx="1"/>
    <rect x="326" y="60" width="18" height="12" rx="1"/>
    <rect x="326" y="78" width="18" height="12" rx="1"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="350" y="52">USB</text>
    <text x="350" y="70">HDMI</text>
    <text x="350" y="88">Ethernet</text>
    <text x="380" y="112" text-anchor="middle" font-size="7.5" fill-opacity="0.85">10&#8211;65 W</text>
  </g>
</svg>
<figcaption>A mini PC shrinks a full desktop to roughly the size of a paperback by using laptop-class parts — an efficient CPU, compact RAM, and an M.2 SSD — while keeping standard USB, HDMI, and Ethernet ports and sipping only tens of watts.</figcaption>
</figure>

## Overview

To shrink the box, a mini PC uses low-power laptop-class parts: an efficient [CPU](/reference/central-processing-unit/) with integrated graphics, soldered or compact [RAM](/reference/random-access-memory/), and an M.2 [SSD](/reference/solid-state-drive/). It still runs a full desktop [operating system](/reference/operating-system/) and connects to an external [monitor](/reference/computer-monitor/), [keyboard](/reference/keyboard/), and [mouse](/reference/mouse/).

Intel's NUC line popularized the format; many vendors now ship similar units, some fanless and silent. Because they draw so little power, mini PCs make natural always-on machines — left running a service in a corner where a tower would be too big or too loud.

## How it compares

A mini PC lands between a hobbyist board and a full tower:

| Machine | Size | Internals | Runs desktop OS | Expansion |
|---------|------|-----------|-----------------|-----------|
| Single-board computer | Credit-card | ARM SoC | Sometimes | Headers/HATs |
| Mini PC | Paperback | Laptop-class x86 | Yes | Limited |
| Desktop tower | Large | Full desktop | Yes | Extensive |

## Where it fits

A mini PC suits anywhere a tower is too big or loud: media boxes, digital signage, small offices, and always-on home services. It sits between a richer [single-board computer](/reference/single-board-computer/) and a full desktop — more powerful and more standard than a Pi, smaller and quieter than a tower. A small fanless mini PC makes a tidy GopherTrunk node: it can sit by the antenna running a capture daemon around the clock and sip power doing it.

## Sources

[^wiki]: [Nettop](https://en.wikipedia.org/wiki/Nettop) — Wikipedia, on small-form-factor desktop PCs.
